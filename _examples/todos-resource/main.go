//
// Todos Resource
// ==============
// This example demonstrates a project structure that defines a subrouter and its
// handlers on a struct, and mounting them as subrouters to a parent router.
// See also _examples/rest for an in-depth example of a REST service, and apply
// those same patterns to this structure.
//
package main

import (
	"context"

	"github.com/swaggest/fchi"
	"github.com/swaggest/fchi/middleware"
	"github.com/valyala/fasthttp"
)

func main() {
	r := fchi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	r.Get("/", fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte("."))
	}))

	r.Mount("/users", usersResource{}.Routes())
	r.Mount("/todos", todosResource{}.Routes())

	fasthttp.ListenAndServe(":3333", fchi.RequestHandler(r))
}
