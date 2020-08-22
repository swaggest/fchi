package fasthttp

import (
	"context"
	"github.com/swaggest/fchi"
	"github.com/swaggest/rest/fasthttp/fchirouter"
	"github.com/swaggest/usecase"
)

// UseCaseMiddlewares applies use case middlewares to rest.Handler.
func UseCaseMiddlewares(mw ...usecase.Middleware) func(fchi.Handler) fchi.Handler {
	return func(handler fchi.Handler) fchi.Handler {
		if len(mw) == 0 {
			return handler
		}

		var uh *Handler
		if !fchirouter.HandlerAs(handler, &uh) {
			return handler
		}

		u := uh.UseCase()
		fu := usecase.Interact(func(ctx context.Context, input, output interface{}) error {
			return output.(error)
		})

		uh.SetUseCase(usecase.Wrap(u, mw...))
		uh.failingUseCase = usecase.Wrap(fu, mw...)

		return handler
	}
}
