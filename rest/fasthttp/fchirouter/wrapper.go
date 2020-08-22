package fchirouter

import (
	"net/http"
	"reflect"
	"runtime"
	"strings"

	"github.com/swaggest/fchi"
)

// NewWrapper creates router wrapper with handler pre-processor callback.
func NewWrapper(r fchi.Router) *Wrapper {
	return &Wrapper{
		Router: r,
	}
}

// Wrapper wraps Router to pre-process Handler and add support for ActionHandler.
type Wrapper struct {
	fchi.Router
	basePattern string

	middlewares []func(fchi.Handler) fchi.Handler
}

var _ fchi.Router = &Wrapper{}

func (r *Wrapper) copy(router fchi.Router, pattern string) *Wrapper {
	return &Wrapper{
		Router:      router,
		basePattern: r.basePattern + pattern,
		middlewares: r.middlewares,
	}
}

// Use appends one of more middlewares onto the Router stack.
func (r *Wrapper) Use(middlewares ...func(fchi.Handler) fchi.Handler) {
	r.middlewares = append(r.middlewares, middlewares...)
}

// With adds inline middlewares for an endpoint handler.
func (r Wrapper) With(middlewares ...func(fchi.Handler) fchi.Handler) fchi.Router {
	c := r.copy(r.Router, "")
	c.Use(middlewares...)
	return c
}

// Group adds a new inline-router along the current routing path, with a fresh middleware stack for the inline-router.
func (r *Wrapper) Group(fn func(r fchi.Router)) fchi.Router {
	im := r.With()

	if fn != nil {
		fn(im)
	}

	return im
}

// Route mounts a sub-router along a `basePattern` string.
func (r *Wrapper) Route(pattern string, fn func(r fchi.Router)) fchi.Router {
	subRouter := r.copy(fchi.NewRouter(), pattern)

	if fn != nil {
		fn(subRouter)
	}

	r.Mount(pattern, subRouter)

	return subRouter
}

// Mount attaches another Handler along "./basePattern/*".
func (r *Wrapper) Mount(pattern string, h fchi.Handler) {
	p := r.prepareHandler("", pattern, h)
	r.Router.Mount(pattern, p)
}

// Handle adds routes for `basePattern` that matches all HTTP methods.
func (r *Wrapper) Handle(pattern string, h fchi.Handler) {
	r.Router.Handle(pattern, r.prepareHandler("", pattern, h))
}

// Method adds routes for `basePattern` that matches the `method` HTTP method.
func (r *Wrapper) Method(method, pattern string, h fchi.Handler) {
	r.Router.Method(method, pattern, r.prepareHandler(method, pattern, h))
}

// Connect adds the route `pattern` that matches a CONNECT http method to execute the `handlerFn` HandlerFunc.
func (r *Wrapper) Connect(pattern string, handlerFn fchi.Handler) {
	r.Method(http.MethodConnect, pattern, handlerFn)
}

// Cancel adds the route `pattern` that matches a DELETE http method to execute the `handlerFn` HandlerFunc.
func (r *Wrapper) Delete(pattern string, handlerFn fchi.Handler) {
	r.Method(http.MethodDelete, pattern, handlerFn)
}

// Get adds the route `pattern` that matches a GET http method to execute the `handlerFn` HandlerFunc.
func (r *Wrapper) Get(pattern string, handlerFn fchi.Handler) {
	r.Method(http.MethodGet, pattern, handlerFn)
}

// Head adds the route `pattern` that matches a HEAD http method to execute the `handlerFn` HandlerFunc.
func (r *Wrapper) Head(pattern string, handlerFn fchi.Handler) {
	r.Method(http.MethodHead, pattern, handlerFn)
}

// HandlerTrait adds the route `pattern` that matches a OPTIONS http method to execute the `handlerFn` HandlerFunc.
func (r *Wrapper) Options(pattern string, handlerFn fchi.Handler) {
	r.Method(http.MethodOptions, pattern, handlerFn)
}

// Patch adds the route `pattern` that matches a PATCH http method to execute the `handlerFn` HandlerFunc.
func (r *Wrapper) Patch(pattern string, handlerFn fchi.Handler) {
	r.Method(http.MethodPatch, pattern, handlerFn)
}

// Post adds the route `pattern` that matches a POST http method to execute the `handlerFn` HandlerFunc.
func (r *Wrapper) Post(pattern string, handlerFn fchi.Handler) {
	r.Method(http.MethodPost, pattern, handlerFn)
}

// Put adds the route `pattern` that matches a PUT http method to execute the `handlerFn` HandlerFunc.
func (r *Wrapper) Put(pattern string, handlerFn fchi.Handler) {
	r.Method(http.MethodPut, pattern, handlerFn)
}

// Trace adds the route `pattern` that matches a TRACE http method to execute the `handlerFn` HandlerFunc.
func (r *Wrapper) Trace(pattern string, handlerFn fchi.Handler) {
	r.Method(http.MethodTrace, pattern, handlerFn)
}

func (r *Wrapper) resolvePattern(pattern string) string {
	return r.basePattern + strings.Replace(pattern, "/*/", "/", -1)
}

func (r *Wrapper) prepareHandler(method, pattern string, h fchi.Handler) fchi.Handler {
	mw := append(r.middlewares, HandlerWithRouteMiddleware(method, r.resolvePattern(pattern)))
	h = WrapHandler(h, mw...)

	return h
}

type handlerWithRoute struct {
	fchi.Handler
	method      string
	pathPattern string
}

func (h handlerWithRoute) RouteMethod() string {
	return h.method
}

func (h handlerWithRoute) RoutePattern() string {
	return h.pathPattern
}

// HandlerWithRouteMiddleware wraps handler with routing information.
func HandlerWithRouteMiddleware(method, pathPattern string) func(fchi.Handler) fchi.Handler {
	return func(handler fchi.Handler) fchi.Handler {
		return handlerWithRoute{
			Handler:     handler,
			pathPattern: pathPattern,
			method:      method,
		}
	}
}

func WrapHandler(h fchi.Handler, mw ...func(fchi.Handler) fchi.Handler) fchi.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		w := mw[i](h)
		if w == nil {
			panic("nil handler returned from middleware: " + runtime.FuncForPC(reflect.ValueOf(mw[i]).Pointer()).Name())
		}
		h = &wrappedHandler{
			Handler: w,
			wrapped: h,
		}
	}

	return h
}

// HandlerAs finds the first http.Handler in http.Handler's chain that matches target, and if so, sets
// target to that http.Handler value and returns true.
//
// An http.Handler matches target if the http.Handler's concrete value is assignable to the value
// pointed to by target.
//
// HandlerAs will panic if target is not a non-nil pointer to either a type that implements
// http.Handler, or to any interface type.
func HandlerAs(handler fchi.Handler, target interface{}) bool {
	if target == nil {
		panic("target cannot be nil")
	}
	val := reflect.ValueOf(target)
	typ := val.Type()
	if typ.Kind() != reflect.Ptr || val.IsNil() {
		panic("target must be a non-nil pointer")
	}
	if e := typ.Elem(); e.Kind() != reflect.Interface && !e.Implements(handlerType) {
		panic("*target must be interface or implement fchi.Handler")
	}
	targetType := typ.Elem()

	for {
		wrap, isWrap := handler.(*wrappedHandler)

		if isWrap {
			handler = wrap.Handler
		}

		if reflect.TypeOf(handler).AssignableTo(targetType) {
			val.Elem().Set(reflect.ValueOf(handler))
			return true
		}

		if !isWrap {
			break
		}

		handler = wrap.wrapped
	}

	return false
}

var handlerType = reflect.TypeOf((*fchi.Handler)(nil)).Elem()

type wrappedHandler struct {
	fchi.Handler
	wrapped fchi.Handler
}
