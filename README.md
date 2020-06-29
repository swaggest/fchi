# <img src="https://avatars2.githubusercontent.com/u/42277636?s=100&v=4" /> <img alt="chi" src="https://cdn.rawgit.com/go-chi/chi/master/_examples/chi.svg" width="220" />

[![Build Status](https://github.com/swaggest/fchi/workflows/test/badge.svg)](https://github.com/swaggest/fchi/actions?query=branch%3Amaster+workflow%3Atest)
[![Coverage Status](https://codecov.io/gh/swaggest/fchi/branch/master/graph/badge.svg)](https://codecov.io/gh/swaggest/fchi)
[![GoDevDoc](https://img.shields.io/badge/dev-doc-00ADD8?logo=go)](https://pkg.go.dev/github.com/swaggest/fchi)
![Code lines](https://sloc.xyz/github/swaggest/fchi/?category=code)
![Comments](https://sloc.xyz/github/swaggest/fchi/?category=comments)

`chi` is a lightweight, idiomatic and composable router for building Go HTTP services. It's
especially good at helping you write large REST API services that are kept maintainable as your
project grows and changes. `chi` is built on the new `context` package introduced in Go 1.7 to
handle signaling, cancelation and request-scoped values across a handler chain.

The focus of the project has been to seek out an elegant and comfortable design for writing
REST API servers, written during the development of the Pressly API service that powers our
public API service, which in turn powers all of our client-side applications.

The key considerations of chi's design are: project structure, maintainability, standard http
handlers (stdlib-only), developer productivity, and deconstructing a large system into many small
parts. The core router `github.com/go-chi/chi` is quite small (less than 1000 LOC), but we've also
included some useful/optional subpackages: [middleware](/middleware), [render](https://github.com/go-chi/render) and [docgen](https://github.com/go-chi/docgen). We hope you enjoy it too!

## This Fork

This fork changes `chi` implementation to work with [`github.com/valyala/fasthttp`](https://github.com/valyala/fasthttp).

## Install

`go get -u github.com/swaggest/fchi`


## Features

* **Lightweight** - cloc'd in ~1000 LOC for the chi router
* **Fast** - yes, see [benchmarks](#benchmarks)
* **Designed for modular/composable APIs** - middlewares, inline middlewares, route groups and subrouter mounting
* **Context control** - built on new `context` package, providing value chaining, cancellations and timeouts


## Examples

See [_examples/](https://github.com/go-chi/chi/blob/master/_examples/) for a variety of examples.


**As easy as:**

```go
package main

import (
	"context"

	"github.com/swaggest/fchi"
	"github.com/swaggest/fchi/middleware"
    "github.com/valyala/fasthttp"
)

func main() {
	r := fchi.NewRouter()
	r.Use(middleware.NoCache)
	r.Get("/", fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte("welcome"))
	}))
	fasthttp.ListenAndServe(":3000", fchi.RequestHandler(r))
}
```

**REST Preview:**

Here is a little preview of how routing looks like with chi. Also take a look at the generated routing docs
in JSON ([routes.json](https://github.com/go-chi/chi/blob/master/_examples/rest/routes.json)) and in
Markdown ([routes.md](https://github.com/go-chi/chi/blob/master/_examples/rest/routes.md)).

I highly recommend reading the source of the [examples](https://github.com/go-chi/chi/blob/master/_examples/) listed
above, they will show you all the features of chi and serve as a good form of documentation.

```go
import (
  //...
  "context"
  "github.com/swaggest/fchi"
  "github.com/swaggest/fchi/middleware"
)

func main() {
  r := fchi.NewRouter()

  // A good base middleware stack
  r.Use(middleware.RequestID)
  r.Use(middleware.Recoverer)

  // Set a timeout value on the request context (ctx), that will signal
  // through ctx.Done() that the request has timed out and further
  // processing should be stopped.
  r.Use(middleware.Timeout(60 * time.Second))

  r.Get("/", fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
    rc.Write([]byte("hi"))
  }))

  // RESTy routes for "articles" resource
  r.Route("/articles", func(r fchi.Router) {
    r.With(paginate).Get("/", fchi.HandlerFunc(listArticles))                           // GET /articles
    r.With(paginate).Get("/{month}-{day}-{year}", fchi.HandlerFunc(listArticlesByDate)) // GET /articles/01-16-2017

    r.Post("/", fchi.HandlerFunc(createArticle))                                        // POST /articles
    r.Get("/search", fchi.HandlerFunc(searchArticles))                                  // GET /articles/search

    // Regexp url parameters:
    r.Get("/{articleSlug:[a-z-]+}", fchi.HandlerFunc(getArticleBySlug))                // GET /articles/home-is-toronto

    // Subrouters:
    r.Route("/{articleID}", func(r fchi.Router) {
      r.Use(ArticleCtx)
      r.Get("/", fchi.HandlerFunc(getArticle))                                          // GET /articles/123
      r.Put("/", fchi.HandlerFunc(updateArticle))                                       // PUT /articles/123
      r.Delete("/", fchi.HandlerFunc(deleteArticle))                                    // DELETE /articles/123
    })
  })

  // Mount the admin sub-router
  r.Mount("/admin", adminRouter())

  fasthttp.ListenAndServe(":3333", fchi.RequestHandler(r))
}

func ArticleCtx(next fchi.Handler) fchi.Handler {
  return fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
    articleID := fchi.URLParam(rc, "articleID")
    article, err := dbGetArticle(articleID)
    if err != nil {
      rc.Error(http.StatusText(404), 404)
      return
    }
    ctx = context.WithValue(ctx, "article", article)
    next.ServeHTTP(ctx, rc)
  })
}

func getArticle(ctx context.Context, rc *fasthttp.RequestCtx) {
  article, ok := ctx.Value("article").(*Article)
  if !ok {
    rc.Error(http.StatusText(422), 422)
    return
  }
  rc.Write([]byte(fmt.Sprintf("title:%s", article.Title)))
}

// A completely separate router for administrator routes
func adminRouter() fchi.Handler {
  r := chi.NewRouter()
  r.Use(AdminOnly)
  r.Get("/", fchi.HandlerFunc(adminIndex))
  r.Get("/accounts", fchi.HandlerFunc(adminListAccounts))
  return r
}

func AdminOnly(next fchi.Handler) fchi.Handler {
  return fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
    perm, ok := ctx.Value("acl.permission").(YourPermissionType)
    if !ok || !perm.IsAdmin() {
      rc.Error(http.StatusText(403), 403)
      return
    }
    next.ServeHTTP(ctx, rc)
  })
}
```


## Router design

chi's router is based on a kind of [Patricia Radix trie](https://en.wikipedia.org/wiki/Radix_tree).
The router is fully compatible with `fasthttp`.

Built on top of the tree is the `Router` interface:

```go
// Router consisting of the core routing methods used by chi's Mux.
type Router interface {
	fchi.Handler
	Routes

	// Use appends one or more middlewares onto the Router stack.
	Use(middlewares ...func(fchi.Handler) fchi.Handler)

	// With adds inline middlewares for an endpoint handler.
	With(middlewares ...func(fchi.Handler) fchi.Handler) Router

	// Group adds a new inline-Router along the current routing
	// path, with a fresh middleware stack for the inline-Router.
	Group(fn func(r Router)) Router

	// Route mounts a sub-Router along a `pattern`` string.
	Route(pattern string, fn func(r Router)) Router

	// Mount attaches another fchi.Handler along ./pattern/*
	Mount(pattern string, h fchi.Handler)

	// Handle and HandleFunc adds routes for `pattern` that matches
	// all HTTP methods.
	Handle(pattern string, h fchi.Handler)

	// Method and MethodFunc adds routes for `pattern` that matches
	// the `method` HTTP method.
	Method(method, pattern string, h fchi.Handler)

	// HTTP-method routing along `pattern`
	Connect(pattern string, h fchi.HandlerFunc)
	Delete(pattern string, h fchi.HandlerFunc)
	Get(pattern string, h fchi.HandlerFunc)
	Head(pattern string, h fchi.HandlerFunc)
	Options(pattern string, h fchi.HandlerFunc)
	Patch(pattern string, h fchi.HandlerFunc)
	Post(pattern string, h fchi.HandlerFunc)
	Put(pattern string, h fchi.HandlerFunc)
	Trace(pattern string, h fchi.HandlerFunc)

	// NotFound defines a handler to respond whenever a route could
	// not be found.
	NotFound(h fchi.HandlerFunc)

	// MethodNotAllowed defines a handler to respond whenever a method is
	// not allowed.
	MethodNotAllowed(h fchi.HandlerFunc)
}

// Routes interface adds two methods for router traversal, which is also
// used by the github.com/go-chi/docgen package to generate documentation for Routers.
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
```

Each routing method accepts a URL `pattern` and chain of `handlers`. The URL pattern
supports named params (ie. `/users/{userID}`) and wildcards (ie. `/admin/*`). URL parameters
can be fetched at runtime by calling `chi.URLParam(r, "userID")` for named parameters
and `chi.URLParam(r, "*")` for a wildcard parameter.


### Middleware handlers

Here is an example of a fasthttp middleware handler using the new request context
available in Go. This middleware sets a hypothetical user identifier on the request
context and calls the next handler in the chain.

```go
// HTTP middleware setting a value on the request context
func MyMiddleware(next fchi.Handler) fchi.Handler {
  return fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
    ctx = context.WithValue(r.Context(), "user", "123")
    next.ServeHTTP(ctx, rc)
  })
}
```


### Request handlers

chi uses fasthttp request handlers. This little snippet is an example of a fchi.Handler
func that reads a user identifier from the request context - hypothetically, identifying
the user sending an authenticated request, validated+set by a previous middleware handler.

```go
// HTTP handler accessing data from the request context.
func MyRequestHandler(ctx context.Context, rc *fasthttp.RequestCtx) {
  user := ctx.Value("user").(string)
  rc.Write([]byte(fmt.Sprintf("hi %s", user)))
}
```


### URL parameters

chi's router parses and stores URL parameters right onto the request context. Here is
an example of how to access URL params in your fasthttp handlers. And of course, middlewares
are able to access the same information.

```go
// HTTP handler accessing the url routing parameters.
func MyRequestHandler(ctx context.Context, rc *fasthttp.RequestCtx) {
  userID := fchi.URLParam(rc, "userID") // from a route like /users/{userID}

  key := ctx.Value("key").(string)

  rc.Write([]byte(fmt.Sprintf("hi %v, %v", userID, key)))
}
```


## Middlewares

chi comes equipped with an optional `middleware` package, providing a suite of
`fasthttp` middlewares. 

### Core middlewares

-----------------------------------------------------------------------------------------------------------
| chi/middleware Handler | description                                                                    |
|:----------------------|:---------------------------------------------------------------------------------
| AllowContentType      | Explicit whitelist of accepted request Content-Types                            |
| BasicAuth             | Basic HTTP authentication                                                       |
| Compress              | Gzip compression for clients that accept compressed responses                   |
| GetHead               | Automatically route undefined HEAD requests to GET handlers                     |
| Heartbeat             | Monitoring endpoint to check the servers pulse                                  |
| Logger                | Logs the start and end of each request with the elapsed processing time         |
| NoCache               | Sets response headers to prevent clients from caching                           |
| Profiler              | Easily attach net/http/pprof to your routers                                    |
| RealIP                | Sets a http.Request's RemoteAddr to either X-Forwarded-For or X-Real-IP         |
| Recoverer             | Gracefully absorb panics and prints the stack trace                             |
| RequestID             | Injects a request ID into the context of each request                           |
| RedirectSlashes       | Redirect slashes on routing paths                                               |
| SetHeader             | Short-hand middleware to set a response header key/value                        |
| StripSlashes          | Strip slashes on routing paths                                                  |
| Throttle              | Puts a ceiling on the number of concurrent requests                             |
| Timeout               | Signals to the request context when the timeout deadline is reached             |
| URLFormat             | Parse extension from url and put it on request context                          |
| WithValue             | Short-hand middleware to set a key/value on the request context                 |
-----------------------------------------------------------------------------------------------------------

### Extra middlewares & packages

Please see https://github.com/go-chi for additional packages.

--------------------------------------------------------------------------------------------------------------------
| package                                            | description                                                 |
|:---------------------------------------------------|:-------------------------------------------------------------
| [cors](https://github.com/go-chi/cors)             | Cross-origin resource sharing (CORS)                        |
| [docgen](https://github.com/go-chi/docgen)         | Print chi.Router routes at runtime                          |
| [jwtauth](https://github.com/go-chi/jwtauth)       | JWT authentication                                          |
| [hostrouter](https://github.com/go-chi/hostrouter) | Domain/host based request routing                           |
| [httplog](https://github.com/go-chi/httplog)       | Small but powerful structured HTTP request logging          |
| [httprate](https://github.com/go-chi/httprate)     | HTTP request rate limiter                                   |
| [httptracer](https://github.com/go-chi/httptracer) | HTTP request performance tracing library                    |
| [httpvcr](https://github.com/go-chi/httpvcr)       | Write deterministic tests for external sources              |
| [stampede](https://github.com/go-chi/stampede)     | HTTP request coalescer                                      |
--------------------------------------------------------------------------------------------------------------------

please [submit a PR](./CONTRIBUTING.md) if you'd like to include a link to a chi-compatible middleware


## context?

`context` is a tiny pkg that provides simple interface to signal context across call stacks
and goroutines. It was originally written by [Sameer Ajmani](https://github.com/Sajmani)
and is available in stdlib since go1.7.

Learn more at https://blog.golang.org/context

and..
* Docs: https://golang.org/pkg/context
* Source: https://github.com/golang/go/tree/master/src/context


## Benchmarks

The benchmark suite: https://github.com/pkieltyka/go-http-routing-benchmark

Results as of Jan 9, 2019 with Go 1.11.4 on Linux X1 Carbon laptop

```shell
BenchmarkMux/route:/-16                  6323047               167 ns/op              83 B/op          0 allocs/op
BenchmarkMux/route:/hi-16                5262666               193 ns/op              83 B/op          1 allocs/op
BenchmarkMux/route:/sup/123/and/this-16                  2106927               499 ns/op             435 B/op          1 allocs/op
BenchmarkMux/route:/sup/123/foo/this-16                  1526112               798 ns/op             677 B/op          1 allocs/op
BenchmarkMux/route:/sharing/z/aBc-16                     6755617               159 ns/op              97 B/op          0 allocs/op
BenchmarkMux/route:/sharing/z/aBc/twitter-16             9653289               133 ns/op             107 B/op          0 allocs/op
BenchmarkMux/route:/sharing/z/aBc/direct-16              6590539               153 ns/op              80 B/op          0 allocs/op
BenchmarkMux/route:/sharing/z/aBc/direct/download-16             9125960               115 ns/op              90 B/op          0 allocs/op
```

Comparison with other routers: https://gist.github.com/pkieltyka/123032f12052520aaccab752bd3e78cc

NOTE: the allocs in the benchmark above are from the calls to http.Request's
`WithContext(context.Context)` method that clones the http.Request, sets the `Context()`
on the duplicated (alloc'd) request and returns it the new request object. This is just
how setting context on a request in Go works.


## Credits

* Carl Jackson for https://github.com/zenazn/goji
  * Parts of chi's thinking comes from goji, and chi's middleware package
    sources from goji.
* Armon Dadgar for https://github.com/armon/go-radix
* Contributions: [@VojtechVitek](https://github.com/VojtechVitek)

We'll be more than happy to see [your contributions](./CONTRIBUTING.md)!


## Beyond REST

chi is just a http router that lets you decompose request handling into many smaller layers.
Many companies use chi to write REST services for their public APIs. But, REST is just a convention
for managing state via HTTP, and there's a lot of other pieces required to write a complete client-server
system or network of microservices.

Looking beyond REST, I also recommend some newer works in the field:
* [webrpc](https://github.com/webrpc/webrpc) - Web-focused RPC client+server framework with code-gen
* [gRPC](https://github.com/grpc/grpc-go) - Google's RPC framework via protobufs
* [graphql](https://github.com/99designs/gqlgen) - Declarative query language
* [NATS](https://nats.io) - lightweight pub-sub


## License

Copyright (c) 2015-present [Peter Kieltyka](https://github.com/pkieltyka)

Licensed under [MIT License](./LICENSE)
