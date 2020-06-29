package main

import (
	"context"
	"errors"

	"github.com/swaggest/fchi"
	"github.com/valyala/fasthttp"
)

type Handler func(ctx context.Context, rc *fasthttp.RequestCtx) error

func (h Handler) ServeHTTP(ctx context.Context, rc *fasthttp.RequestCtx) {
	if err := h(ctx, rc); err != nil {
		// handle returned error here.
		rc.Response.Header.SetStatusCode(503)
		rc.Write([]byte("bad"))
	}
}

func main() {
	r := fchi.NewRouter()
	r.Method("GET", "/", Handler(customHandler))
	fasthttp.ListenAndServe(":3333", fchi.RequestHandler(r))
}

func customHandler(ctx context.Context, rc *fasthttp.RequestCtx) error {
	q := rc.Request.URI().QueryArgs().Peek("err")

	if len(q) > 0 {
		return errors.New(string(q))
	}

	rc.Write([]byte("foo"))
	return nil
}
