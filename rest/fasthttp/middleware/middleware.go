package middleware

import (
	"github.com/swaggest/fchi"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/fasthttp"
	"github.com/swaggest/rest/fasthttp/fchirouter"
	"github.com/swaggest/usecase"
)

type withRequestDecoder interface {
	SetRequestDecoder(decoder fasthttp.RequestDecoder)
}

// RequestDecoderMiddleware sets up request decoder in suitable handlers.
func RequestDecoderMiddleware(factory fasthttp.RequestDecoderFactory) func(fchi.Handler) fchi.Handler {
	return func(handler fchi.Handler) fchi.Handler {
		var (
			withRoute          rest.HandlerWithRoute
			withUseCase        rest.HandlerWithUseCase
			withRequestDecoder withRequestDecoder
			useCaseWithInput   usecase.HasInputPort
		)

		if !fchirouter.HandlerAs(handler, &withRequestDecoder) ||
			!fchirouter.HandlerAs(handler, &withRoute) ||
			!fchirouter.HandlerAs(handler, &withUseCase) ||
			!usecase.As(withUseCase.UseCase(), &useCaseWithInput) {
			return handler
		}

		input := useCaseWithInput.InputPort()
		if input != nil {
			withRequestDecoder.SetRequestDecoder(factory.MakeDecoder(withRoute.RouteMethod(), useCaseWithInput.InputPort()))
		}

		return handler
	}
}

// RequestValidatorMiddleware sets up request validator in suitable handlers.
func RequestValidatorMiddleware(factory rest.ValidatorFactory) func(fchi.Handler) fchi.Handler {
	return func(handler fchi.Handler) fchi.Handler {
		var (
			withRoute           rest.HandlerWithRoute
			withUseCase         rest.HandlerWithUseCase
			setRequestValidator interface {
				SetRequestValidator(validator rest.Validator)
			}
			useCaseWithInput usecase.HasInputPort
		)

		if !fchirouter.HandlerAs(handler, &setRequestValidator) ||
			!fchirouter.HandlerAs(handler, &withRoute) ||
			!fchirouter.HandlerAs(handler, &withUseCase) ||
			!usecase.As(withUseCase.UseCase(), &useCaseWithInput) {
			return handler
		}

		setRequestValidator.SetRequestValidator(factory.MakeValidator(withRoute.RouteMethod(), useCaseWithInput.InputPort()))

		return handler
	}
}
