package fchi

import (
	"context"
	"github.com/valyala/fasthttp"
	"strings"
)

var (
	// routeUserValueKey is the user value key to store the request context.
	routeUserValueKey = "fchiRouteCtx"

	ctxUserValueKey = "fchiCtx"
)

func Ctx(rc *fasthttp.RequestCtx) context.Context {
	if ctx, ok := rc.UserValue(ctxUserValueKey).(context.Context); ok {
		return ctx
	}

	return context.Background()
}

func SetCtx(ctx context.Context, rc *fasthttp.RequestCtx) {
	rc.SetUserValue(ctxUserValueKey, ctx)
}

// Context is the default routing context set on the root node of a
// request context to track route patterns, URL parameters and
// an optional routing path.
type Context struct {
	Routes Routes

	// Routing path/method override used during the route search.
	// See Mux#routeHTTP method.
	RoutePath   string
	RouteMethod string

	// Routing pattern stack throughout the lifecycle of the request,
	// across all connected routers. It is a record of all matching
	// patterns across a stack of sub-routers.
	RoutePatterns []string

	// URLParams are the stack of routeParams captured during the
	// routing lifecycle across a stack of sub-routers.
	URLParams RouteParams

	// The endpoint routing pattern that matched the request URI path
	// or `RoutePath` of the current sub-router. This value will update
	// during the lifecycle of a request passing through a stack of
	// sub-routers.
	routePattern string

	// Route parameters matched for the current sub-router. It is
	// intentionally unexported so it cant be tampered.
	routeParams RouteParams

	// methodNotAllowed hint
	methodNotAllowed bool
}

// NewRouteContext returns a new routing Context object.
func NewRouteContext() *Context {
	return &Context{}
}

// Reset a routing context to its initial state.
func (x *Context) Reset() {
	x.Routes = nil
	x.RoutePath = ""
	x.RouteMethod = ""
	x.RoutePatterns = x.RoutePatterns[:0]
	x.URLParams.Keys = x.URLParams.Keys[:0]
	x.URLParams.Values = x.URLParams.Values[:0]

	x.routePattern = ""
	x.routeParams.Keys = x.routeParams.Keys[:0]
	x.routeParams.Values = x.routeParams.Values[:0]
	x.methodNotAllowed = false
}

// URLParam returns the corresponding URL parameter value from the request
// routing context.
func (x *Context) URLParam(key string) string {
	for k := len(x.URLParams.Keys) - 1; k >= 0; k-- {
		if x.URLParams.Keys[k] == key {
			return x.URLParams.Values[k]
		}
	}
	return ""
}

// RoutePattern builds the routing pattern string for the particular
// request, at the particular point during routing. This means, the value
// will change throughout the execution of a request in a router. That is
// why its advised to only use this value after calling the next handler.
//
// For example,
//
//   func Instrument(next Handler) Handler {
//     return HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//       next.ServeHTTP(w, r)
//       routePattern := chi.RouteContext(r.Context()).RoutePattern()
//       measure(w, r, routePattern)
//   	 })
//   }
func (x *Context) RoutePattern() string {
	routePattern := strings.Join(x.RoutePatterns, "")
	return strings.Replace(routePattern, "/*/", "/", -1)
}

// RouteContext returns chi's routing Context object from a
// http.Request Context.
func RouteContext(rc *fasthttp.RequestCtx) *Context {
	val, _ := rc.UserValue(routeUserValueKey).(*Context)
	return val
}

// URLParam returns the url parameter from a http.Request object.
func URLParam(rc *fasthttp.RequestCtx, key string) string {
	if rctx := RouteContext(rc); rctx != nil {
		return rctx.URLParam(key)
	}
	return ""
}

// URLParamFromCtx returns the url parameter from a http.Request Context.
func URLParamFromCtx(rc *fasthttp.RequestCtx, key string) string {
	if rctx := RouteContext(rc); rctx != nil {
		return rctx.URLParam(key)
	}
	return ""
}

// RouteParams is a structure to track URL routing parameters efficiently.
type RouteParams struct {
	Keys, Values []string
}

// Add will append a URL parameter to the end of the route param
func (s *RouteParams) Add(key, value string) {
	s.Keys = append(s.Keys, key)
	s.Values = append(s.Values, value)
}
