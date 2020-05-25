package middleware

import (
	"context"
	"github.com/swaggest/fchi"
	"github.com/valyala/fasthttp"
	"time"
)

// Timeout is a middleware that cancels ctx after a given timeout and return
// a 504 Gateway Timeout error to the client.
//
// It's required that you select the ctx.Done() channel to check for the signal
// if the context has reached its deadline and return, otherwise the timeout
// signal will be just ignored.
//
// ie. a route/handler may look like:
//
//  r.Get("/long", func(w http.ResponseWriter, r *http.Request) {
// 	 ctx := r.Context()
// 	 processTime := time.Duration(rand.Intn(4)+1) * time.Second
//
// 	 select {
// 	 case <-ctx.Done():
// 	 	return
//
// 	 case <-time.After(processTime):
// 	 	 // The above channel simulates some hard work.
// 	 }
//
// 	 w.Write([]byte("done"))
//  })
//
func Timeout(timeout time.Duration) func(next fchi.Handler) fchi.Handler {
	return func(next fchi.Handler) fchi.Handler {
		fn := func(rc *fasthttp.RequestCtx) {
			ctx, cancel := context.WithTimeout(fchi.Ctx(rc), timeout)
			defer func() {
				cancel()
				if ctx.Err() == context.DeadlineExceeded {
					rc.SetStatusCode(fasthttp.StatusGatewayTimeout)
				}
			}()

			fchi.SetCtx(ctx, rc)
			next.ServeHTTP(rc)
		}
		return fchi.HandlerFunc(fn)
	}
}
