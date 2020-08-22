package decoder

import (
	"github.com/swaggest/rest"
	fasthttp2 "github.com/swaggest/rest/fasthttp"
	"github.com/valyala/fasthttp"
	"net/url"
	"reflect"
	"strings"

	"github.com/swaggest/form"
)

var _ fasthttp2.RequestDecoderFactory = &Factory{}

// Factory decodes http requests.
//
// Please use NewFactory to create instance.
type Factory struct {
	formDecoders     map[rest.ParamIn]*form.Decoder
	decoderFunctions map[rest.ParamIn]decoderFunc
}

// NewFactory creates request decoder factory.
func NewFactory(pathToURLValues func(rc *fasthttp.RequestCtx) (url.Values, error)) *Factory {
	f := &Factory{
		decoderFunctions: map[rest.ParamIn]decoderFunc{
			rest.ParamInCookie:   cookiesToURLValues,
			rest.ParamInFormData: formDataToURLValues,
			rest.ParamInHeader:   headerToURLValues,
			rest.ParamInQuery:    queryToURLValues,
			rest.ParamInPath:     pathToURLValues,
		},
	}
	f.formDecoders = make(map[rest.ParamIn]*form.Decoder, len(f.decoderFunctions))

	for in := range f.decoderFunctions {
		dec := form.NewDecoder()
		dec.SetTagName(string(in))
		dec.SetMode(form.ModeExplicit)
		f.formDecoders[in] = dec
	}

	return f
}

// MakeDecoder creates request.RequestDecoder for a http method and request structure.
//
// Only for methods with body semantics (POST, PUT, PATCH) request structure is checked for `json`, `file` tags.
func (rd *Factory) MakeDecoder(method string, input interface{}) fasthttp2.RequestDecoder {
	inputType := reflect.TypeOf(input)

	m := decoder{
		decoders: make([]valueDecoderFunc, 0),
		in:       make([]rest.ParamIn, 0),
	}

	for in, formDecoder := range rd.formDecoders {
		if hasFieldTags(inputType, string(in)) {
			m.decoders = append(m.decoders, makeDecoder(in, formDecoder, rd.decoderFunctions[in]))
			m.in = append(m.in, in)
		}
	}

	method = strings.ToUpper(method)

	if method != fasthttp.MethodPost && method != fasthttp.MethodPut && method != fasthttp.MethodPatch {
		return &m
	}

	// Checking for body tags.
	if hasFieldTags(inputType, `json`) || isSliceOrMap(inputType) {
		m.decoders = append(m.decoders, decodeJSONBody)
		m.in = append(m.in, rest.ParamInBody)
	}

	if hasFileFields(inputType, "file") || hasFileFields(inputType, "formData") {
		m.decoders = append(m.decoders, decodeFiles)
		m.in = append(m.in, rest.ParamInFormData)
	}

	return &m
}

// RegisterFunc adds custom type handling.
func (rd *Factory) RegisterFunc(fn form.DecodeFunc, types ...interface{}) {
	for _, fd := range rd.formDecoders {
		fd.RegisterFunc(fn, types...)
	}
}
