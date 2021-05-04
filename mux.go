package fchi

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/valyala/fasthttp"
)

var _ Router = &Mux{}

// Mux is a simple HTTP route multiplexer that parses a request path,
// records any URL params, and executes an end handler. It implements
// the Handler interface and is friendly with the standard library.
//
// Mux is designed to be fast, minimal and offer a powerful API for building
// modular and composable HTTP services with a large set of handlers. It's
// particularly useful for writing large REST API services that break a handler
// into many smaller parts composed of middlewares and end handlers.
type Mux struct {
	// The computed mux handler made of the chained middleware stack and
	// the tree router
	handler Handler

	// The radix trie router
	tree *node

	// Custom method not allowed handler
	methodNotAllowedHandler Handler

	// Controls the behaviour of middleware chain generation when a mux
	// is registered as an inline group inside another mux.
	parent *Mux

	// Routing context pool
	pool *sync.Pool

	// Custom route not found handler
	notFoundHandler Handler

	// The middleware stack
	middlewares []func(Handler) Handler

	inline bool
}

// NewMux returns a newly initialized Mux object that implements the Router
// interface.
func NewMux() *Mux {
	mux := &Mux{tree: &node{}, pool: &sync.Pool{}}
	mux.pool.New = func() interface{} {
		return NewRouteContext()
	}
	return mux
}

// ServeHTTP is the single method of the Handler interface that makes
// Mux interoperable with the standard library. It uses a sync.Pool to get and
// reuse routing contexts for each request.
func (mx *Mux) ServeHTTP(ctx context.Context, rc *fasthttp.RequestCtx) {
	// Ensure the mux has some routes defined on the mux
	if mx.handler == nil {
		mx.NotFoundHandler().ServeHTTP(ctx, rc)
		return
	}

	// Check if a routing context already exists from a parent router.
	rctx, _ := rc.UserValue(routeUserValueKey).(*Context)
	if rctx != nil {
		mx.handler.ServeHTTP(ctx, rc)
		return
	}

	// Fetch a RouteContext object from the sync pool, and call the computed
	// mx.handler that is comprised of mx.middlewares + mx.routeHTTP.
	// Once the request is finished, reset the routing context and put it back
	// into the pool for reuse from another request.
	rctx = mx.pool.Get().(*Context)
	rctx.Reset()
	rctx.Routes = mx
	rctx.parentCtx = r.Context()

	rc.SetUserValue(routeUserValueKey, rctx)

	// Serve the request and once its done, put the request context back in the sync pool
	mx.handler.ServeHTTP(ctx, rc)
	mx.pool.Put(rctx)
}

// Use appends a middleware handler to the Mux middleware stack.
//
// The middleware stack for any Mux will execute before searching for a matching
// route to a specific handler, which provides opportunity to respond early,
// change the course of the request execution, or set request-scoped values for
// the next Handler.
func (mx *Mux) Use(middlewares ...func(Handler) Handler) {
	if mx.handler != nil {
		panic("chi: all middlewares must be defined before routes on a mux")
	}
	mx.middlewares = append(mx.middlewares, middlewares...)
}

// Handle adds the route `pattern` that matches any http method to
// execute the `handler` Handler.
func (mx *Mux) Handle(pattern string, handler Handler) {
	mx.handle(mALL, pattern, handler)
}

// Method adds the route `pattern` that matches `method` http method to
// execute the `handler` Handler.
func (mx *Mux) Method(method, pattern string, handler Handler) {
	m, ok := methodMap[strings.ToUpper(method)]
	if !ok {
		panic(fmt.Sprintf("chi: '%s' http method is not supported.", method))
	}
	mx.handle(m, pattern, handler)
}

// Connect adds the route `pattern` that matches a CONNECT http method to
// execute the `handlerFn` HandlerFunc.
func (mx *Mux) Connect(pattern string, handler Handler) {
	mx.handle(mCONNECT, pattern, handler)
}

// Delete adds the route `pattern` that matches a DELETE http method to
// execute the `handlerFn` HandlerFunc.
func (mx *Mux) Delete(pattern string, handler Handler) {
	mx.handle(mDELETE, pattern, handler)
}

// Get adds the route `pattern` that matches a GET http method to
// execute the `handlerFn` HandlerFunc.
func (mx *Mux) Get(pattern string, handler Handler) {
	mx.handle(mGET, pattern, handler)
}

// Head adds the route `pattern` that matches a HEAD http method to
// execute the `handlerFn` HandlerFunc.
func (mx *Mux) Head(pattern string, handler Handler) {
	mx.handle(mHEAD, pattern, handler)
}

// Options adds the route `pattern` that matches a OPTIONS http method to
// execute the `handlerFn` HandlerFunc.
func (mx *Mux) Options(pattern string, handler Handler) {
	mx.handle(mOPTIONS, pattern, handler)
}

// Patch adds the route `pattern` that matches a PATCH http method to
// execute the `handlerFn` HandlerFunc.
func (mx *Mux) Patch(pattern string, handler Handler) {
	mx.handle(mPATCH, pattern, handler)
}

// Post adds the route `pattern` that matches a POST http method to
// execute the `handlerFn` HandlerFunc.
func (mx *Mux) Post(pattern string, handler Handler) {
	mx.handle(mPOST, pattern, handler)
}

// Put adds the route `pattern` that matches a PUT http method to
// execute the `handlerFn` HandlerFunc.
func (mx *Mux) Put(pattern string, handler Handler) {
	mx.handle(mPUT, pattern, handler)
}

// Trace adds the route `pattern` that matches a TRACE http method to
// execute the `handlerFn` HandlerFunc.
func (mx *Mux) Trace(pattern string, handler Handler) {
	mx.handle(mTRACE, pattern, handler)
}

// NotFound sets a custom HandlerFunc for routing paths that could
// not be found. The default 404 handler is `http.NotFound`.
func (mx *Mux) NotFound(handler Handler) {
	// Build NotFound handler chain
	m := mx
	if mx.inline && mx.parent != nil {
		m = mx.parent
		handler = Chain(mx.middlewares...).Handler(handler)
	}

	// Update the notFoundHandler from this point forward
	m.notFoundHandler = handler
	m.updateSubRoutes(func(subMux *Mux) {
		if subMux.notFoundHandler == nil {
			subMux.NotFound(handler)
		}
	})
}

// MethodNotAllowed sets a custom HandlerFunc for routing paths where the
// method is unresolved. The default handler returns a 405 with an empty body.
func (mx *Mux) MethodNotAllowed(handler Handler) {
	// Build MethodNotAllowed handler chain
	m := mx
	if mx.inline && mx.parent != nil {
		m = mx.parent
		handler = Chain(mx.middlewares...).Handler(handler)
	}

	// Update the methodNotAllowedHandler from this point forward
	m.methodNotAllowedHandler = handler
	m.updateSubRoutes(func(subMux *Mux) {
		if subMux.methodNotAllowedHandler == nil {
			subMux.MethodNotAllowed(handler)
		}
	})
}

// With adds inline middlewares for an endpoint handler.
func (mx *Mux) With(middlewares ...func(Handler) Handler) Router {
	// Similarly as in handle(), we must build the mux handler once additional
	// middleware registration isn't allowed for this stack, like now.
	if !mx.inline && mx.handler == nil {
		mx.updateRouteHandler()
	}

	// Copy middlewares from parent inline muxs
	var mws Middlewares
	if mx.inline {
		mws = make(Middlewares, len(mx.middlewares))
		copy(mws, mx.middlewares)
	}
	mws = append(mws, middlewares...)

	im := &Mux{
		pool: mx.pool, inline: true, parent: mx, tree: mx.tree, middlewares: mws,
		notFoundHandler: mx.notFoundHandler, methodNotAllowedHandler: mx.methodNotAllowedHandler,
	}

	return im
}

// Group creates a new inline-Mux with a fresh middleware stack. It's useful
// for a group of handlers along the same routing path that use an additional
// set of middlewares. See _examples/.
func (mx *Mux) Group(fn func(r Router)) Router {
	im := mx.With().(*Mux)
	if fn != nil {
		fn(im)
	}
	return im
}

// Route creates a new Mux with a fresh middleware stack and mounts it
// along the `pattern` as a subrouter. Effectively, this is a short-hand
// call to Mount. See _examples/.
func (mx *Mux) Route(pattern string, fn func(r Router)) Router {
	if fn == nil {
		panic(fmt.Sprintf("chi: attempting to Route() a nil subrouter on '%s'", pattern))
	}
	subRouter := NewRouter()
	fn(subRouter)
	mx.Mount(pattern, subRouter)
	return subRouter
}

// Mount attaches another Handler or chi Router as a subrouter along a routing
// path. It's very useful to split up a large API as many independent routers and
// compose them as a single service using Mount. See _examples/.
//
// Note that Mount() simply sets a wildcard along the `pattern` that will continue
// routing at the `handler`, which in most cases is another chi.Router. As a result,
// if you define two Mount() routes on the exact same pattern the mount will panic.
func (mx *Mux) Mount(pattern string, handler Handler) {
	if handler == nil {
		panic(fmt.Sprintf("chi: attempting to Mount() a nil handler on '%s'", pattern))
	}

	// Provide runtime safety for ensuring a pattern isn't mounted on an existing
	// routing pattern.
	if mx.tree.findPattern(pattern+"*") || mx.tree.findPattern(pattern+"/*") {
		panic(fmt.Sprintf("chi: attempting to Mount() a handler on an existing path, '%s'", pattern))
	}

	// Assign sub-Router's with the parent not found & method not allowed handler if not specified.
	subr, ok := handler.(*Mux)
	if ok && subr.notFoundHandler == nil && mx.notFoundHandler != nil {
		subr.NotFound(mx.notFoundHandler)
	}
	if ok && subr.methodNotAllowedHandler == nil && mx.methodNotAllowedHandler != nil {
		subr.MethodNotAllowed(mx.methodNotAllowedHandler)
	}

	mountHandler := HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rctx := RouteContext(rc)

		// shift the url path past the previous subrouter
		rctx.RoutePath = mx.nextRoutePath(rctx)

		// reset the wildcard URLParam which connects the subrouter
		n := len(rctx.URLParams.Keys) - 1
		if n >= 0 && rctx.URLParams.Keys[n] == "*" && len(rctx.URLParams.Values) > n {
			rctx.URLParams.Values[n] = ""
		}

		handler.ServeHTTP(ctx, rc)
	})

	if pattern == "" || pattern[len(pattern)-1] != '/' {
		mx.handle(mALL|mSTUB, pattern, mountHandler)
		mx.handle(mALL|mSTUB, pattern+"/", mountHandler)
		pattern += "/"
	}

	method := mALL
	subroutes, _ := handler.(Routes)
	if subroutes != nil {
		method |= mSTUB
	}
	n := mx.handle(method, pattern+"*", mountHandler)

	if subroutes != nil {
		n.subroutes = subroutes
	}
}

// Routes returns a slice of routing information from the tree,
// useful for traversing available routes of a router.
func (mx *Mux) Routes() []Route {
	return mx.tree.routes()
}

// Middlewares returns a slice of middleware handler functions.
func (mx *Mux) Middlewares() Middlewares {
	return mx.middlewares
}

// Match searches the routing tree for a handler that matches the method/path.
// It's similar to routing a http request, but without executing the handler
// thereafter.
//
// Note: the *Context state is updated during execution, so manage
// the state carefully or make a NewRouteContext().
func (mx *Mux) Match(rctx *Context, method, path string) bool {
	m, ok := methodMap[method]
	if !ok {
		return false
	}

	node, _, h := mx.tree.FindRoute(rctx, m, path)

	if node != nil && node.subroutes != nil {
		rctx.RoutePath = mx.nextRoutePath(rctx)
		return node.subroutes.Match(rctx, method, rctx.RoutePath)
	}

	return h != nil
}

// NotFoundHandler returns the default Mux 404 responder whenever a route
// cannot be found.
func (mx *Mux) NotFoundHandler() Handler {
	if mx.notFoundHandler != nil {
		return mx.notFoundHandler
	}
	return HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.SetStatusCode(fasthttp.StatusNotFound)
		rc.SetContentType("text/plain; charset=utf-8")
		_, _ = rc.Write([]byte("404 page not found"))
	})
}

// MethodNotAllowedHandler returns the default Mux 405 responder whenever
// a method cannot be resolved for a route.
func (mx *Mux) MethodNotAllowedHandler() Handler {
	if mx.methodNotAllowedHandler != nil {
		return mx.methodNotAllowedHandler
	}

	return HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.SetStatusCode(fasthttp.StatusMethodNotAllowed)
	})
}

// handle registers a Handler in the routing tree for a particular http method
// handle registers a http.Handler in the routing tree for a particular http method
// and routing pattern.
func (mx *Mux) handle(method methodTyp, pattern string, handler Handler) *node {
	if len(pattern) == 0 || pattern[0] != '/' {
		panic(fmt.Sprintf("chi: routing pattern must begin with '/' in '%s'", pattern))
	}

	// Build the computed routing handler for this routing pattern.
	if !mx.inline && mx.handler == nil {
		mx.updateRouteHandler()
	}

	// Build endpoint handler with inline middlewares for the route
	var h Handler
	if mx.inline {
		mx.handler = HandlerFunc(mx.routeHTTP)
		h = Chain(mx.middlewares...).Handler(handler)
	} else {
		h = handler
	}

	// Add the endpoint to the tree and return the node
	return mx.tree.InsertRoute(method, pattern, h)
}

// routeHTTP routes a http.Request through the Mux routing tree to serve
// the matching handler for a particular http method.
func (mx *Mux) routeHTTP(ctx context.Context, rc *fasthttp.RequestCtx) {
	// Grab the route context object
	rctx := RouteContext(rc)

	// The request routing path
	routePath := rctx.RoutePath
	if routePath == "" {
		routePath = string(rc.URI().PathOriginal())
		if routePath == "" {
			routePath = "/"
		}
	}

	// Check if method is supported by chi
	if rctx.RouteMethod == "" {
		rctx.RouteMethod = string(rc.Method())
	}
	method, ok := methodMap[rctx.RouteMethod]
	if !ok {
		mx.MethodNotAllowedHandler().ServeHTTP(ctx, rc)
		return
	}

	// Find the route
	if _, _, h := mx.tree.FindRoute(rctx, method, routePath); h != nil {
		h.ServeHTTP(ctx, rc)
		return
	}
	if rctx.methodNotAllowed {
		mx.MethodNotAllowedHandler().ServeHTTP(ctx, rc)
	} else {
		mx.NotFoundHandler().ServeHTTP(ctx, rc)
	}
}

func (mx *Mux) nextRoutePath(rctx *Context) string {
	routePath := "/"
	nx := len(rctx.routeParams.Keys) - 1 // index of last param in list
	if nx >= 0 && rctx.routeParams.Keys[nx] == "*" && len(rctx.routeParams.Values) > nx {
		routePath = "/" + rctx.routeParams.Values[nx]
	}
	return routePath
}

// Recursively update data on child routers.
func (mx *Mux) updateSubRoutes(fn func(subMux *Mux)) {
	for _, r := range mx.tree.routes() {
		subMux, ok := r.SubRoutes.(*Mux)
		if !ok {
			continue
		}
		fn(subMux)
	}
}

// updateRouteHandler builds the single mux handler that is a chain of the middleware
// stack, as defined by calls to Use(), and the tree router (Mux) itself. After this
// point, no other middlewares can be registered on this Mux's stack. But you can still
// compose additional middlewares via Group()'s or using a chained middleware handler.
func (mx *Mux) updateRouteHandler() {
	mx.handler = chain(mx.middlewares, HandlerFunc(mx.routeHTTP))
}