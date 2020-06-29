package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/swaggest/fchi"
	"github.com/valyala/fasthttp"
)

func main() {
	r := fchi.NewRouter()
	r.Get("/", fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte("root."))
	}))

	r.Route("/road", func(r fchi.Router) {
		r.Get("/left", fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			rc.Write([]byte("left road"))
		}))
		r.Post("/right", fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			rc.Write([]byte("right road"))
		}))
	})

	r.Put("/ping", fchi.HandlerFunc(Ping))

	walkFunc := func(method string, route string, handler fchi.Handler, middlewares ...func(fchi.Handler) fchi.Handler) error {
		route = strings.Replace(route, "/*/", "/", -1)
		fmt.Printf("%s %s\n", method, route)
		return nil
	}

	if err := fchi.Walk(r, walkFunc); err != nil {
		fmt.Printf("Logging err: %s\n", err.Error())
	}
}

// Ping returns pong
func Ping(ctx context.Context, rc *fasthttp.RequestCtx) {
	rc.Write([]byte("pong"))
}
