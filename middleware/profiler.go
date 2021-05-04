package middleware

import (
	"context"
	"expvar"
	"fmt"
	"net/http"
	"net/http/pprof"

	"github.com/swaggest/fchi"
	"github.com/valyala/fasthttp"
)

// Profiler is a convenient subrouter used for mounting net/http/pprof. ie.
//
//  func MyService() http.Handler {
//    r := chi.NewRouter()
//    // ..middlewares
//    r.Mount("/debug", middleware.Profiler())
//    // ..routes
//    return r
//  }
func Profiler() fchi.Handler {
	r := fchi.NewRouter()
	r.Use(NoCache)

	r.Get("/", fchi.HandlerFunc(func(_ context.Context, rc *fasthttp.RequestCtx) {
		rc.Redirect(string(rc.Request.URI().RequestURI())+"/pprof/", fasthttp.StatusMovedPermanently)
	}))
	r.Handle("/pprof", fchi.HandlerFunc(func(_ context.Context, rc *fasthttp.RequestCtx) {
		rc.Redirect(string(rc.Request.URI().RequestURI())+"/pprof/", fasthttp.StatusMovedPermanently)
	}))

	r.Handle("/pprof/*", fchi.Adapt(http.HandlerFunc(pprof.Index)))
	r.Handle("/pprof/cmdline", fchi.Adapt(http.HandlerFunc(pprof.Cmdline)))
	r.Handle("/pprof/profile", fchi.Adapt(http.HandlerFunc(pprof.Profile)))
	r.Handle("/pprof/symbol", fchi.Adapt(http.HandlerFunc(pprof.Symbol)))
	r.Handle("/pprof/trace", fchi.Adapt(http.HandlerFunc(pprof.Trace)))
	r.Handle("/vars", fchi.HandlerFunc(expVars))

	r.Handle("/pprof/goroutine", fchi.Adapt(pprof.Handler("goroutine")))
	r.Handle("/pprof/threadcreate", fchi.Adapt(pprof.Handler("threadcreate")))
	r.Handle("/pprof/mutex", fchi.Adapt(pprof.Handler("mutex")))
	r.Handle("/pprof/heap", fchi.Adapt(pprof.Handler("heap")))
	r.Handle("/pprof/block", fchi.Adapt(pprof.Handler("block")))
	r.Handle("/pprof/allocs", fchi.Adapt(pprof.Handler("allocs")))

	return r
}

// Replicated from expvar.go as not public.
func expVars(_ context.Context, rc *fasthttp.RequestCtx) {
	first := true
	rc.Response.Header.SetContentType("application/json")
	fmt.Fprintf(rc.Response.BodyWriter(), "{\n")
	expvar.Do(func(kv expvar.KeyValue) {
		if !first {
			fmt.Fprintf(rc.Response.BodyWriter(), ",\n")
		}
		first = false
		fmt.Fprintf(rc.Response.BodyWriter(), "%q: %s", kv.Key, kv.Value)
	})
	fmt.Fprintf(rc.Response.BodyWriter(), "\n}\n")
}
