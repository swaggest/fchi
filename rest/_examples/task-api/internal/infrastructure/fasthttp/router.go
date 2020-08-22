package fasthttp

import (
	"context"
	"github.com/swaggest/fchi"
	middleware3 "github.com/swaggest/fchi/middleware"
	"github.com/swaggest/rest/_examples/task-api/internal/infrastructure/schema"
	"github.com/swaggest/rest/_examples/task-api/internal/infrastructure/service"
	"github.com/swaggest/rest/_examples/task-api/internal/usecase"
	"github.com/swaggest/rest/fasthttp"
	"github.com/swaggest/rest/fasthttp/decoder"
	"github.com/swaggest/rest/fasthttp/fchirouter"
	middleware2 "github.com/swaggest/rest/fasthttp/middleware"
	"github.com/swaggest/rest/fasthttp/openapi"
	"github.com/swaggest/rest/jsonschema"
	"github.com/swaggest/swgui/v3cdn"
	usecase2 "github.com/swaggest/usecase"
	fasthttp2 "github.com/valyala/fasthttp"
	"net/http"
)

func NewRouter(locator *service.Locator) fchi.Handler {
	apiSchema := schema.NewOpenAPICollector()
	decoderFactory := decoder.NewFactory(fchirouter.PathToURLValues)
	validatorFactory := jsonschema.Factory{}

	r := fchirouter.NewWrapper(fchi.NewRouter())

	r.Use(
		fasthttp.UseCaseMiddlewares(usecase2.MiddlewareFunc(func(next usecase2.Interactor) usecase2.Interactor {
			return usecase2.Interact(func(ctx context.Context, input, output interface{}) error {
				return next.Interact(ctx, input, output)
			})
		})),
		openapi.Middleware(apiSchema),
		middleware2.RequestDecoderMiddleware(decoderFactory),
		middleware2.RequestValidatorMiddleware(validatorFactory),
	)

	adminAuth := middleware3.BasicAuth("Admin Access", map[string]string{"admin": "admin"})
	userAuth := middleware3.BasicAuth("User Access", map[string]string{"user": "user"})

	r.Use(
		func(handler fchi.Handler) fchi.Handler {
			return fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp2.RequestCtx) {
				handler.ServeHTTP(ctx, rc)
			})
		},
		middleware3.NoCache,
		//middleware3.Timeout(time.Second),
		middleware3.Recoverer,
	)

	// API version 1.
	r.Route("/v0", func(r fchi.Router) {
		r.Group(func(r fchi.Router) {
			r.Method(http.MethodPost, "/tasks", fasthttp.NewHandler(usecase.CreateTask(locator)))
			r.Method(http.MethodPut, "/tasks/{id}", fasthttp.NewHandler(usecase.UpdateTask(locator)))
			r.Method(http.MethodGet, "/tasks/{id}", fasthttp.NewHandler(usecase.FindTask(locator)))
			r.Method(http.MethodGet, "/tasks", fasthttp.NewHandler(usecase.FindTasks(locator)))
			r.Method(http.MethodDelete, "/tasks/{id}", fasthttp.NewHandler(usecase.CloseTask(locator)))
		})
	})

	// API version 1.
	r.Route("/v1", func(r fchi.Router) {
		r.Group(func(r fchi.Router) {
			r.Use(adminAuth, openapi.HTTPBasicSecurityMiddleware(apiSchema, "Admin", "Admin access"))

			r.Method(http.MethodPost, "/tasks", fasthttp.NewHandler(usecase.CreateTask(locator)))
		})

	})

	// API version 2.
	r.Route("/v2", func(r fchi.Router) {
		r.Group(func(r fchi.Router) {
			r.Use(userAuth, openapi.HTTPBasicSecurityMiddleware(apiSchema, "User", "User access"))

			r.Method(http.MethodPost, "/tasks", fasthttp.NewHandler(usecase.CreateTask(locator)))
		})
	})

	r.Method(http.MethodGet, "/docs/openapi.json", fchi.Adapt(apiSchema))
	r.Mount("/docs", fchi.Adapt(v3cdn.NewHandler(apiSchema.Reflector().Spec.Info.Title,
		"/docs/openapi.json", "/docs")))

	r.Mount("/debug", middleware3.Profiler())

	return r
}
