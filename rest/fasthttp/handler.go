package fasthttp

import (
	"context"
	"encoding/json"
	"github.com/swaggest/fchi"
	"github.com/swaggest/rest"
	"net/http"
	"reflect"

	"github.com/swaggest/usecase"
	"github.com/swaggest/usecase/status"
	"github.com/valyala/fasthttp"
)

var (
	_ fchi.Handler = &Handler{}
)

// NewHandler creates use case http handler.
func NewHandler(useCase usecase.Interactor, options ...func(h *Handler)) *Handler {
	h := &Handler{
		options: options,
	}
	h.SetUseCase(useCase)

	return h
}

// UseCase returns use case interactor.
func (h *Handler) UseCase() usecase.Interactor {
	return h.useCase
}

// SetUseCase prepares handler for a use case.
func (h *Handler) SetUseCase(useCase usecase.Interactor) {
	h.useCase = useCase

	for _, option := range h.options {
		option(h)
	}

	h.setupInputBuffer()
	h.setupOutputBuffer()
}

// Handler is a use case http handler with documentation and inputPort validation.
//
// Please use NewHandler to create instance.
type Handler struct {
	rest.HandlerTrait

	// WriteResponse overrides default JSON writer,
	// can be used to alter content type and/or marshaler.
	WriteResponse func(rc *fasthttp.RequestCtx, v interface{})

	// requestDecoder maps data from http.Request into structured Go input value.
	requestDecoder RequestDecoder

	options []func(h *Handler)

	// failingUseCase allows to pass input decoding error through use case middlewares.
	failingUseCase usecase.Interactor

	useCase usecase.Interactor

	outputBufferType reflect.Type
	inputBufferType  reflect.Type
	skipRendering    bool
}

func (h *Handler) SetRequestDecoder(requestDecoder RequestDecoder) {
	h.requestDecoder = requestDecoder
}

type noContent interface {
	NoContent() bool
}

// ServeHTTP serves http inputPort with use case interactor.
func (h *Handler) ServeHTTP(ctx context.Context, rc *fasthttp.RequestCtx) {
	var (
		input, output interface{}
		err           error
	)

	if h.inputBufferType != nil {
		if h.requestDecoder == nil {
			panic("request decoder is not initialized, please use SetRequestDecoder")
		}

		input = reflect.New(h.inputBufferType).Interface()

		err = h.requestDecoder.Decode(rc, input, h.RequestValidator())
		if err != nil {
			err = status.Wrap(err, status.InvalidArgument)

			if h.failingUseCase != nil {
				err = h.failingUseCase.Interact(ctx, "decoding failed", err)
			}

			if h.MakeErrResp != nil {
				er, sc := h.MakeErrResp(ctx, err)
				h.writeResponse(rc, sc, er)
			} else {
				er, sc := rest.Err(err)
				h.writeResponse(rc, sc, er)
			}

			return
		}
	}

	if h.outputBufferType != nil {
		output = reflect.New(h.outputBufferType).Interface()
	}
	skipRendering := h.skipRendering

	if withWriter, ok := output.(usecase.OutputWithWriter); ok {
		skipRendering = true

		withWriter.SetWriter(rc.Response.BodyWriter())
	}

	err = h.useCase.Interact(ctx, input, output)

	if err != nil {
		if h.MakeErrResp != nil {
			er, sc := h.MakeErrResp(ctx, err)
			h.writeResponse(rc, sc, er)
		} else {
			er, sc := rest.Err(err)
			h.writeResponse(rc, sc, er)
		}

		return
	}

	successfulResponseCode := h.SuccessfulResponseCode
	if !skipRendering {
		if nc, ok := output.(noContent); ok {
			skipRendering = nc.NoContent()
			if skipRendering {
				successfulResponseCode = http.StatusNoContent
			}
		}
	}

	if skipRendering {
		if successfulResponseCode != http.StatusOK {
			rc.SetStatusCode(successfulResponseCode)
		}
		return
	}

	statusCode := http.StatusOK
	if successfulResponseCode != statusCode {
		statusCode = successfulResponseCode
	}

	h.writeResponse(rc, statusCode, output)
}

func (h *Handler) writeResponse(rc *fasthttp.RequestCtx, statusCode int, v interface{}) {
	if h.WriteResponse != nil {
		h.WriteResponse(rc, v)
		return
	}

	rc.SetContentType("application/json; charset=utf-8")

	b, err := json.Marshal(v)
	if err != nil {
		rc.SetStatusCode(fasthttp.StatusInternalServerError)
		rc.SetContentType("text/plain; charset=utf-8")
		_, _ = rc.Write([]byte(err.Error()))

		return
	}

	rc.SetStatusCode(statusCode)
	_, _ = rc.Write(b)
}

func (h *Handler) setupInputBuffer() {
	h.inputBufferType = nil

	var withInput usecase.HasInputPort
	if !usecase.As(h.useCase, &withInput) {
		return
	}

	h.inputBufferType = reflect.TypeOf(withInput.InputPort())
	if h.inputBufferType != nil {
		if h.inputBufferType.Kind() == reflect.Ptr {
			h.inputBufferType = h.inputBufferType.Elem()
		}
	}
}

func (h *Handler) setupOutputBuffer() {
	h.outputBufferType = nil

	var withOutput usecase.HasOutputPort
	if !usecase.As(h.useCase, &withOutput) {
		return
	}

	h.outputBufferType = reflect.TypeOf(withOutput.OutputPort())
	if h.outputBufferType != nil {
		if h.outputBufferType.Kind() == reflect.Ptr {
			h.outputBufferType = h.outputBufferType.Elem()
		}
	} else {
		h.skipRendering = true
	}

	if h.SuccessfulResponseCode != 0 {
		return
	}

	if h.outputBufferType == nil {
		h.SuccessfulResponseCode = http.StatusNoContent
	} else {
		h.SuccessfulResponseCode = http.StatusOK
	}
}
