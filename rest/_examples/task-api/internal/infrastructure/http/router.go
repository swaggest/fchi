package http

import (
	"context"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/swaggest/rest/_examples/task-api/internal/infrastructure/schema"
	"github.com/swaggest/rest/_examples/task-api/internal/infrastructure/service"
	"github.com/swaggest/rest/_examples/task-api/internal/usecase"
	"github.com/swaggest/rest/jsonschema"
	"github.com/swaggest/rest/nethttp"
	"github.com/swaggest/rest/nethttp/chirouter"
	"github.com/swaggest/rest/nethttp/decoder"
	"github.com/swaggest/rest/nethttp/openapi"
	"github.com/swaggest/swgui/v3cdn"
	usecase2 "github.com/swaggest/usecase"
	"net/http"
	"time"
)

func NewRouter(locator *service.Locator) http.Handler {
	apiSchema := schema.NewOpenAPICollector()
	decoderFactory := decoder.NewFactory(chirouter.PathToURLValues)
	validatorFactory := jsonschema.Factory{}

	r := chirouter.NewWrapper(chi.NewRouter())

	r.Use(
		middleware.Recoverer,
		nethttp.UseCaseMiddlewares(usecase2.MiddlewareFunc(func(next usecase2.Interactor) usecase2.Interactor {
			return usecase2.Interact(func(ctx context.Context, input, output interface{}) error {
				return next.Interact(ctx, input, output)
			})
		})),
		openapi.Middleware(apiSchema),
		nethttp.RequestDecoderMiddleware(decoderFactory),
		nethttp.RequestValidatorMiddleware(validatorFactory),
	)

	//auth := func(next http.Handler) http.Handler {
	//	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
	//		next.ServeHTTP(rw, r)
	//	})
	//}

	adminAuth := middleware.BasicAuth("Admin Access", map[string]string{"admin": "admin"})
	userAuth := middleware.BasicAuth("User Access", map[string]string{"user": "user"})

	r.Use(
		middleware.NoCache,
		middleware.Timeout(time.Second),
	)

	// API version 1.
	r.Route("/v0", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Method(http.MethodPost, "/tasks", nethttp.NewHandler(usecase.CreateTask(locator)))
			r.Method(http.MethodPut, "/tasks/{id}", nethttp.NewHandler(usecase.UpdateTask(locator)))
			r.Method(http.MethodGet, "/tasks/{id}", nethttp.NewHandler(usecase.FindTask(locator)))
			r.Method(http.MethodGet, "/tasks", nethttp.NewHandler(usecase.FindTasks(locator)))
			r.Method(http.MethodDelete, "/tasks/{id}", nethttp.NewHandler(usecase.CloseTask(locator)))
		})

	})

	// API version 1.
	r.Route("/v1", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(adminAuth, openapi.HTTPBasicSecurityMiddleware(apiSchema, "Admin", "Admin access"))

			r.Method(http.MethodPost, "/tasks", nethttp.NewHandler(usecase.CreateTask(locator)))
		})

	})

	// API version 2.
	r.Route("/v2", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(userAuth, openapi.HTTPBasicSecurityMiddleware(apiSchema, "User", "User access"))

			r.Method(http.MethodPost, "/tasks", nethttp.NewHandler(usecase.CreateTask(locator)))
		})
	})

	r.Method(http.MethodGet, "/docs/openapi.json", apiSchema)
	r.Mount("/docs", v3cdn.NewHandler(apiSchema.Reflector().Spec.Info.Title,
		"/docs/openapi.json", "/docs"))

	r.Mount("/debug", middleware.Profiler())

	return r
}
