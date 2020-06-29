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
		rc.Write([]byte("hello world"))
	}))

	fasthttp.ListenAndServe(":3333", fchi.RequestHandler(r))
}
