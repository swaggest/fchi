package openapi

import (
	"github.com/swaggest/fchi"
	"github.com/swaggest/openapi-go/openapi3"
	"github.com/swaggest/rest"
	"github.com/swaggest/rest/fasthttp"
	"github.com/swaggest/rest/fasthttp/fchirouter"
	"github.com/swaggest/rest/openapi"
	"net/http"
)

// Middleware reads info and adds validation to handler.
func Middleware(s *openapi.Collector) func(handler fchi.Handler) fchi.Handler {
	return func(handler fchi.Handler) fchi.Handler {
		var (
			withRoute   rest.HandlerWithRoute
			restHandler *fasthttp.Handler
		)

		if !fchirouter.HandlerAs(handler, &withRoute) || !fchirouter.HandlerAs(handler, &restHandler) {
			return handler
		}

		u := restHandler.UseCase()

		err := s.Collect(withRoute.RouteMethod(), withRoute.RoutePattern(), u, restHandler.HandlerTrait)
		if err != nil {
			panic(err)
		}

		return handler
	}
}

func HTTPBasicSecurityMiddleware(s *openapi.Collector, name, description string) func(fchi.Handler) fchi.Handler {
	hss := openapi3.HTTPSecurityScheme{}

	hss.WithScheme("basic")

	if description != "" {
		hss.WithDescription(description)
	}

	s.Reflector().SpecEns().ComponentsEns().SecuritySchemesEns().WithMapOfSecuritySchemeOrRefValuesItem(
		name,
		openapi3.SecuritySchemeOrRef{
			SecurityScheme: &openapi3.SecurityScheme{
				HTTPSecurityScheme: &hss,
			},
		},
	)

	return securityMiddleware(s, name)
}

func securityMiddleware(s *openapi.Collector, name string) func(fchi.Handler) fchi.Handler {
	return func(next fchi.Handler) fchi.Handler {
		var withRoute rest.HandlerWithRoute

		if fchirouter.HandlerAs(next, &withRoute) {
			err := s.Reflector().SpecEns().SetupOperation(withRoute.RouteMethod(), withRoute.RoutePattern(), func(op *openapi3.Operation) error {
				op.Security = append(op.Security, map[string][]string{name: {}})
				return s.Reflector().SetJSONResponse(op, rest.ErrResponse{}, http.StatusUnauthorized)
			})
			if err != nil {
				panic(err)
			}
		}

		return next
	}
}
