//
// Package fchi is a small, idiomatic and composable router for building HTTP services.
//
// fchi requires Go 1.11 or newer.
//
// Example:
//  package main
//
//  import (
//		"context"
//
//  	"github.com/swaggest/fchi"
//  	"github.com/swaggest/fchi/middleware"
//		"github.com/valyala/fasthttp"
//  )
//
//  func main() {
//  	r := chi.NewRouter()
//  	r.Use(middleware.Recoverer)
//
//  	r.Get("/", func(ctx context.Context, rc *fasthttp.RequestCtx) {
//  		rc.Write([]byte("root."))
//  	})
//
//  	fasthttp.ListenAndServe(":3333", fchi.RequestHandler(r))
//  }
//
// See github.com/swaggest/fchi/_examples/ for more in-depth examples.
//
// URL patterns allow for easy matching of path components in HTTP
// requests. The matching components can then be accessed using
// chi.URLParam(). All patterns must begin with a slash.
//
// A simple named placeholder {name} matches any sequence of characters
// up to the next / or the end of the URL. Trailing slashes on paths must
// be handled explicitly.
//
// A placeholder with a name followed by a colon allows a regular
// expression match, for example {number:\\d+}. The regular expression
// syntax is Go's normal regexp RE2 syntax, except that regular expressions
// including { or } are not supported, and / will never be
// matched. An anonymous regexp pattern is allowed, using an empty string
// before the colon in the placeholder, such as {:\\d+}
//
// The special placeholder of asterisk matches the rest of the requested
// URL. Any trailing characters in the pattern are ignored. This is the only
// placeholder which will match / characters.
//
// Examples:
//  "/user/{name}" matches "/user/jsmith" but not "/user/jsmith/info" or "/user/jsmith/"
//  "/user/{name}/info" matches "/user/jsmith/info"
//  "/page/*" matches "/page/intro/latest"
//  "/page/*/index" also matches "/page/intro/latest"
//  "/date/{yyyy:\\d\\d\\d\\d}/{mm:\\d\\d}/{dd:\\d\\d}" matches "/date/2017/04/01"
//
package fchi

import (
	"context"
	"net/http"

	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttpadaptor"
)

// NewRouter returns a new Mux object that implements the Router interface.
func NewRouter() *Mux {
	return NewMux()
}

// HandlerFunc implements Handler.
type HandlerFunc func(ctx context.Context, rc *fasthttp.RequestCtx)

// ServeHTTP serves http request.
func (f HandlerFunc) ServeHTTP(ctx context.Context, rc *fasthttp.RequestCtx) {
	f(ctx, rc)
}

// Handler is a FastHTTP handler.
type Handler interface {
	ServeHTTP(ctx context.Context, rc *fasthttp.RequestCtx)
}

// RequestHandler makes fasthttp.RequestHandler function from Handler.
func RequestHandler(handler Handler) fasthttp.RequestHandler {
	return func(rc *fasthttp.RequestCtx) {
		handler.ServeHTTP(rc, rc)
	}
}

// Adapt makes Handler from http.Handler.
func Adapt(h http.Handler) Handler {
	f := fasthttpadaptor.NewFastHTTPHandler(h)

	return HandlerFunc(func(_ context.Context, rc *fasthttp.RequestCtx) {
		f(rc)
	})
}

// Router consisting of the core routing methods used by chi's Mux,
// using only the standard net/http.
type Router interface {
	Handler
	Routes

	// Use appends one or more middlewares onto the Router stack.
	Use(middlewares ...func(Handler) Handler)

	// With adds inline middlewares for an endpoint handler.
	With(middlewares ...func(Handler) Handler) Router

	// Group adds a new inline-Router along the current routing
	// path, with a fresh middleware stack for the inline-Router.
	Group(fn func(r Router)) Router

	// Route mounts a sub-Router along a `pattern`` string.
	Route(pattern string, fn func(r Router)) Router

	// Mount attaches another Handler along ./pattern/*
	Mount(pattern string, h Handler)

	// Handle and HandleFunc adds routes for `pattern` that matches
	// all HTTP methods.
	Handle(pattern string, h Handler)

	// Method and MethodFunc adds routes for `pattern` that matches
	// the `method` HTTP method.
	Method(method, pattern string, h Handler)

	// HTTP-method routing along `pattern`
	Connect(pattern string, h Handler)
	Delete(pattern string, h Handler)
	Get(pattern string, h Handler)
	Head(pattern string, h Handler)
	Options(pattern string, h Handler)
	Patch(pattern string, h Handler)
	Post(pattern string, h Handler)
	Put(pattern string, h Handler)
	Trace(pattern string, h Handler)

	// NotFound defines a handler to respond whenever a route could
	// not be found.
	NotFound(h Handler)

	// MethodNotAllowed defines a handler to respond whenever a method is
	// not allowed.
	MethodNotAllowed(h Handler)
}

// Routes interface adds two methods for router traversal, which is also
// used by the `docgen` subpackage to generation documentation for Routers.
type Routes interface {
	// Routes returns the routing tree in an easily traversable structure.
	Routes() []Route

	// Middlewares returns the list of middlewares in use by the router.
	Middlewares() Middlewares

	// Match searches the routing tree for a handler that matches
	// the method/path - similar to routing a http request, but without
	// executing the handler thereafter.
	Match(rctx *Context, method, path string) bool
}

// Middlewares type is a slice of standard middleware handlers with methods
// to compose middleware chains and Handler's.
type Middlewares []func(Handler) Handler
