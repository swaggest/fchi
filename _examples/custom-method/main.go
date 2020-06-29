package main

import (
	"context"

	"github.com/swaggest/fchi"
	"github.com/swaggest/fchi/middleware"
	"github.com/valyala/fasthttp"
)

func init() {
	fchi.RegisterMethod("LINK")
	fchi.RegisterMethod("UNLINK")
	fchi.RegisterMethod("WOOHOO")
}

func main() {
	r := fchi.NewRouter()
	r.Use(middleware.RequestID)
	r.Get("/", fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte("hello world"))
	}))
	r.Method("LINK", "/link", fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte("custom link method"))
	}))
	r.Method("WOOHOO", "/woo", fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte("custom woohoo method"))
	}))
	r.Handle("/everything", fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte("capturing all standard http methods, as well as LINK, UNLINK and WOOHOO"))
	}))
	fasthttp.ListenAndServe(":3333", fchi.RequestHandler(r))
}
