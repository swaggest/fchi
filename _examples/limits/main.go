//
// Limits
// ======
// This example demonstrates the use of Timeout, and Throttle middlewares.
//
// Timeout:
//   cancel a request if processing takes longer than 2.5 seconds,
//   server will respond with a http.StatusGatewayTimeout.
//
// Throttle:
//   limit the number of in-flight requests along a particular
//   routing path and backlog the others.
//
package main

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/swaggest/fchi"
	"github.com/swaggest/fchi/middleware"
	"github.com/valyala/fasthttp"
)

func main() {
	r := fchi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)

	r.Get("/", fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte("root."))
	}))

	r.Get("/ping", fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte("pong"))
	}))

	r.Get("/panic", fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		panic("test")
	}))

	// Slow handlers/operations.
	r.Group(func(r fchi.Router) {
		// Stop processing after 2.5 seconds.
		r.Use(middleware.Timeout(2500 * time.Millisecond))

		r.Get("/slow", fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			rand.Seed(time.Now().Unix())

			// Processing will take 1-5 seconds.
			processTime := time.Duration(rand.Intn(4)+1) * time.Second

			select {
			case <-ctx.Done():
				return

			case <-time.After(processTime):
				// The above channel simulates some hard work.
			}

			rc.Write([]byte(fmt.Sprintf("Processed in %v seconds\n", processTime)))
		}))
	})

	// Throttle very expensive handlers/operations.
	r.Group(func(r fchi.Router) {
		// Stop processing after 30 seconds.
		r.Use(middleware.Timeout(30 * time.Second))

		// Only one request will be processed at a time.
		r.Use(middleware.Throttle(1))

		r.Get("/throttled", fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			select {
			case <-ctx.Done():
				switch ctx.Err() {
				case context.DeadlineExceeded:
					rc.Response.Header.SetStatusCode(504)
					rc.Write([]byte("Processing too slow\n"))
				default:
					rc.Write([]byte("Canceled\n"))
				}
				return

			case <-time.After(5 * time.Second):
				// The above channel simulates some hard work.
			}

			rc.Write([]byte("Processed\n"))
		}))
	})

	fasthttp.ListenAndServe(":3333", fchi.RequestHandler(r))
}
