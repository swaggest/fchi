package fchi

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/valyala/fasthttp"
)

func TestMuxBasic(t *testing.T) {
	var count uint64
	countermw := func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			count++
			next.ServeHTTP(ctx, rc)
		})
	}

	usermw := func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			ctx = context.WithValue(ctx, ctxKey{"user"}, "peter")
			next.ServeHTTP(ctx, rc)
		})
	}

	exmw := func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			ctx = context.WithValue(ctx, ctxKey{"ex"}, "a")
			next.ServeHTTP(ctx, rc)
		})
	}

	logbuf := bytes.NewBufferString("")
	logmsg := "logmw test"
	logmw := func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			logbuf.WriteString(logmsg)
			next.ServeHTTP(ctx, rc)
		})
	}

	cxindex := HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		user := ctx.Value(ctxKey{"user"}).(string)
		rc.SetStatusCode(200)
		rc.Write([]byte(fmt.Sprintf("hi %s", user)))
	})

	ping := func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.SetStatusCode(200)
		rc.Write([]byte("."))
	}

	headPing := func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Response.Header.Set("X-Ping", "1")
		rc.Response.SetStatusCode(200)
	}

	createPing := func(ctx context.Context, rc *fasthttp.RequestCtx) {
		// create ....
		rc.Response.SetStatusCode(201)
	}

	pingAll := func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Response.SetStatusCode(200)
		rc.Write([]byte("ping all"))
	}

	pingAll2 := func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Response.SetStatusCode(200)
		rc.Write([]byte("ping all2"))
	}

	pingOne := func(ctx context.Context, rc *fasthttp.RequestCtx) {
		idParam := URLParam(rc, "id")
		rc.Response.SetStatusCode(200)
		rc.Write([]byte(fmt.Sprintf("ping one id: %s", idParam)))
	}

	pingWoop := func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Response.SetStatusCode(200)
		rc.Write([]byte("woop." + URLParam(rc, "iidd")))
	}

	catchAll := func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Response.SetStatusCode(200)
		rc.Write([]byte("catchall"))
	}

	m := NewRouter()
	m.Use(countermw)
	m.Use(usermw)
	m.Use(exmw)
	m.Use(logmw)
	m.Get("/", cxindex)
	m.Method("GET", "/ping", HandlerFunc(ping))
	m.Method("GET", "/pingall", HandlerFunc(pingAll))
	m.Method("get", "/ping/all", HandlerFunc(pingAll))
	m.Get("/ping/all2", HandlerFunc(pingAll2))

	m.Head("/ping", HandlerFunc(headPing))
	m.Post("/ping", HandlerFunc(createPing))
	m.Get("/ping/{id}", HandlerFunc(pingWoop))
	m.Get("/ping/{id}", HandlerFunc(pingOne)) // expected to overwrite to pingOne handler
	m.Get("/ping/{iidd}/woop", HandlerFunc(pingWoop))
	m.Handle("/admin/*", HandlerFunc(catchAll))
	// m.Post("/admin/*", catchAll)

	ts := NewTestServer(m)
	defer ts.Close()

	// GET /
	if _, body := testRequest(t, ts, "GET", "/", nil); body != "hi peter" {
		t.Fatalf(body)
	}
	tlogmsg, _ := logbuf.ReadString(0)
	if tlogmsg != logmsg {
		t.Error("expecting log message from middleware:", logmsg)
	}

	// GET /ping
	if _, body := testRequest(t, ts, "GET", "/ping", nil); body != "." {
		t.Fatalf(body)
	}

	// GET /pingall
	if _, body := testRequest(t, ts, "GET", "/pingall", nil); body != "ping all" {
		t.Fatalf(body)
	}

	// GET /ping/all
	if _, body := testRequest(t, ts, "GET", "/ping/all", nil); body != "ping all" {
		t.Fatalf(body)
	}

	// GET /ping/all2
	if _, body := testRequest(t, ts, "GET", "/ping/all2", nil); body != "ping all2" {
		t.Fatalf(body)
	}

	// GET /ping/123
	if _, body := testRequest(t, ts, "GET", "/ping/123", nil); body != "ping one id: 123" {
		t.Fatalf(body)
	}

	// GET /ping/allan
	if _, body := testRequest(t, ts, "GET", "/ping/allan", nil); body != "ping one id: allan" {
		t.Fatalf(body)
	}

	// GET /ping/1/woop
	if _, body := testRequest(t, ts, "GET", "/ping/1/woop", nil); body != "woop.1" {
		t.Fatalf(body)
	}

	// HEAD /ping
	resp, err := http.Head(ts.URL + "/ping")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Error("head failed, should be 200")
	}
	if resp.Header.Get("X-Ping") == "" {
		t.Error("expecting X-Ping header")
	}

	// GET /admin/catch-this
	if _, body := testRequest(t, ts, "GET", "/admin/catch-thazzzzz", nil); body != "catchall" {
		t.Fatalf(body)
	}

	// POST /admin/catch-this
	resp, err = http.Post(ts.URL+"/admin/casdfsadfs", "text/plain", bytes.NewReader([]byte{}))
	if err != nil {
		t.Fatal(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Error("POST failed, should be 200")
	}

	if string(body) != "catchall" {
		t.Error("expecting response body: 'catchall'")
	}

	// Custom http method DIE /ping/1/woop
	if resp, body := testRequest(t, ts, "DIE", "/ping/1/woop", nil); body != "" || resp.StatusCode != 405 {
		t.Fatalf(fmt.Sprintf("expecting 405 status and empty body, got %d '%s'", resp.StatusCode, body))
	}
}

func TestMuxMounts(t *testing.T) {
	r := NewRouter()

	r.Get("/{hash}", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		v := URLParam(rc, "hash")
		rc.Write([]byte(fmt.Sprintf("/%s", v)))
	}))

	r.Route("/{hash}/share", func(r Router) {
		r.Get("/", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			v := URLParam(rc, "hash")
			rc.Write([]byte(fmt.Sprintf("/%s/share", v)))
		}))
		r.Get("/{network}", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			v := URLParam(rc, "hash")
			n := URLParam(rc, "network")
			rc.Write([]byte(fmt.Sprintf("/%s/share/%s", v, n)))
		}))
	})

	m := NewRouter()
	m.Mount("/sharing", r)

	ts := NewTestServer(m)
	defer ts.Close()

	if _, body := testRequest(t, ts, "GET", "/sharing/aBc", nil); body != "/aBc" {
		t.Fatalf(body)
	}
	if _, body := testRequest(t, ts, "GET", "/sharing/aBc/share", nil); body != "/aBc/share" {
		t.Fatalf(body)
	}
	if _, body := testRequest(t, ts, "GET", "/sharing/aBc/share/twitter", nil); body != "/aBc/share/twitter" {
		t.Fatalf(body)
	}
}

func TestMuxPlain(t *testing.T) {
	r := NewRouter()
	r.Get("/hi", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte("bye"))
	}))
	r.NotFound(HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Response.SetStatusCode(404)
		rc.Write([]byte("nothing here"))
	}))

	ts := NewTestServer(r)
	defer ts.Close()

	if _, body := testRequest(t, ts, "GET", "/hi", nil); body != "bye" {
		t.Fatalf(body)
	}
	if _, body := testRequest(t, ts, "GET", "/nothing-here", nil); body != "nothing here" {
		t.Fatalf(body)
	}
}

func TestMuxEmptyRoutes(t *testing.T) {
	mux := NewRouter()

	apiRouter := NewRouter()
	// oops, we forgot to declare any route handlers

	mux.Handle("/api*", apiRouter)

	if body := testHandler(mux, "GET", "/"); body != "404 page not found" {
		t.Fatalf(body)
	}

	if body := testHandler(apiRouter, "GET", "/"); body != "404 page not found" {
		t.Fatalf(body)
	}
}

// Test a mux that routes a trailing slash, see also middleware/strip_test.go
// for an example of using a middleware to handle trailing slashes.
func TestMuxTrailingSlash(t *testing.T) {
	r := NewRouter()
	r.NotFound(HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Response.SetStatusCode(404)
		rc.Write([]byte("nothing here"))
	}))

	subRoutes := NewRouter()
	indexHandler := func(ctx context.Context, rc *fasthttp.RequestCtx) {
		accountID := URLParam(rc, "accountID")
		rc.Write([]byte(accountID))
	}
	subRoutes.Get("/", HandlerFunc(indexHandler))

	r.Mount("/accounts/{accountID}", subRoutes)
	r.Get("/accounts/{accountID}/", HandlerFunc(indexHandler))

	ts := NewTestServer(r)
	defer ts.Close()

	if _, body := testRequest(t, ts, "GET", "/accounts/admin", nil); body != "admin" {
		t.Fatalf(body)
	}
	if _, body := testRequest(t, ts, "GET", "/accounts/admin/", nil); body != "admin" {
		t.Fatalf(body)
	}
	if _, body := testRequest(t, ts, "GET", "/nothing-here", nil); body != "nothing here" {
		t.Fatalf(body)
	}
}

func TestMuxNestedNotFound(t *testing.T) {
	r := NewRouter()

	r.Use(func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			ctx = context.WithValue(ctx, ctxKey{"mw"}, "mw")
			next.ServeHTTP(ctx, rc)
		})
	})

	r.Get("/hi", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte("bye"))
	}))

	r.With(func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			ctx = context.WithValue(ctx, ctxKey{"with"}, "with")
			next.ServeHTTP(ctx, rc)
		})
	}).NotFound(HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		chkMw := ctx.Value(ctxKey{"mw"}).(string)
		chkWith := ctx.Value(ctxKey{"with"}).(string)
		rc.Response.SetStatusCode(404)
		rc.Write([]byte(fmt.Sprintf("root 404 %s %s", chkMw, chkWith)))
	}))

	sr1 := NewRouter()

	sr1.Get("/sub", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte("sub"))
	}))
	sr1.Group(func(sr1 Router) {
		sr1.Use(func(next Handler) Handler {
			return HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
				ctx = context.WithValue(ctx, ctxKey{"mw2"}, "mw2")
				next.ServeHTTP(ctx, rc)
			})
		})
		sr1.NotFound(HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			chkMw2 := ctx.Value(ctxKey{"mw2"}).(string)
			rc.Response.SetStatusCode(404)
			rc.Write([]byte(fmt.Sprintf("sub 404 %s", chkMw2)))
		}))
	})

	sr2 := NewRouter()
	sr2.Get("/sub", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte("sub2"))
	}))

	r.Mount("/admin1", sr1)
	r.Mount("/admin2", sr2)

	ts := NewTestServer(r)
	defer ts.Close()

	if _, body := testRequest(t, ts, "GET", "/hi", nil); body != "bye" {
		t.Fatalf(body)
	}
	if _, body := testRequest(t, ts, "GET", "/nothing-here", nil); body != "root 404 mw with" {
		t.Fatalf(body)
	}
	if _, body := testRequest(t, ts, "GET", "/admin1/sub", nil); body != "sub" {
		t.Fatalf(body)
	}
	if _, body := testRequest(t, ts, "GET", "/admin1/nope", nil); body != "sub 404 mw2" {
		t.Fatalf(body)
	}
	if _, body := testRequest(t, ts, "GET", "/admin2/sub", nil); body != "sub2" {
		t.Fatalf(body)
	}

	// Not found pages should bubble up to the root.
	if _, body := testRequest(t, ts, "GET", "/admin2/nope", nil); body != "root 404 mw with" {
		t.Fatalf(body)
	}
}

func TestMuxNestedMethodNotAllowed(t *testing.T) {
	r := NewRouter()
	r.Get("/root", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte("root"))
	}))
	r.MethodNotAllowed(HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Response.SetStatusCode(405)
		rc.Write([]byte("root 405"))
	}))

	sr1 := NewRouter()
	sr1.Get("/sub1", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte("sub1"))
	}))
	sr1.MethodNotAllowed(HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Response.SetStatusCode(405)
		rc.Write([]byte("sub1 405"))
	}))

	sr2 := NewRouter()
	sr2.Get("/sub2", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte("sub2"))
	}))

	pathVar := NewRouter()
	pathVar.Get("/{var}", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte("pv"))
	}))
	pathVar.MethodNotAllowed(HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Response.SetStatusCode(405)
		rc.Write([]byte("pv 405"))
	}))

	r.Mount("/prefix1", sr1)
	r.Mount("/prefix2", sr2)
	r.Mount("/pathVar", pathVar)

	ts := NewTestServer(r)
	defer ts.Close()

	if _, body := testRequest(t, ts, "GET", "/root", nil); body != "root" {
		t.Fatalf(body)
	}
	if _, body := testRequest(t, ts, "PUT", "/root", nil); body != "root 405" {
		t.Fatalf(body)
	}
	if _, body := testRequest(t, ts, "GET", "/prefix1/sub1", nil); body != "sub1" {
		t.Fatalf(body)
	}
	if _, body := testRequest(t, ts, "PUT", "/prefix1/sub1", nil); body != "sub1 405" {
		t.Fatalf(body)
	}
	if _, body := testRequest(t, ts, "GET", "/prefix2/sub2", nil); body != "sub2" {
		t.Fatalf(body)
	}
	if _, body := testRequest(t, ts, "PUT", "/prefix2/sub2", nil); body != "root 405" {
		t.Fatalf(body)
	}
	if _, body := testRequest(t, ts, "GET", "/pathVar/myvar", nil); body != "pv" {
		t.Fatalf(body)
	}
	if _, body := testRequest(t, ts, "DELETE", "/pathVar/myvar", nil); body != "pv 405" {
		t.Fatalf(body)
	}
}

func TestMuxComplicatedNotFound(t *testing.T) {
	decorateRouter := func(r *Mux) {
		// Root router with groups
		r.Get("/auth", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			rc.Write([]byte("auth get"))
		}))
		r.Route("/public", func(r Router) {
			r.Get("/", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
				rc.Write([]byte("public get"))
			}))
		})

		// sub router with groups
		sub0 := NewRouter()
		sub0.Route("/resource", func(r Router) {
			r.Get("/", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
				rc.Write([]byte("private get"))
			}))
		})
		r.Mount("/private", sub0)

		// sub router with groups
		sub1 := NewRouter()
		sub1.Route("/resource", func(r Router) {
			r.Get("/", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
				rc.Write([]byte("private get"))
			}))
		})
		r.With(func(next Handler) Handler { return next }).Mount("/private_mw", sub1)
	}

	testNotFound := func(t *testing.T, r *Mux) {
		ts := NewTestServer(r)
		defer ts.Close()

		// check that we didn't break correct routes
		if _, body := testRequest(t, ts, "GET", "/auth", nil); body != "auth get" {
			t.Fatalf(body)
		}
		if _, body := testRequest(t, ts, "GET", "/public", nil); body != "public get" {
			t.Fatalf(body)
		}
		if _, body := testRequest(t, ts, "GET", "/public/", nil); body != "public get" {
			t.Fatalf(body)
		}
		if _, body := testRequest(t, ts, "GET", "/private/resource", nil); body != "private get" {
			t.Fatalf(body)
		}
		// check custom not-found on all levels
		if _, body := testRequest(t, ts, "GET", "/nope", nil); body != "custom not-found" {
			t.Fatalf(body)
		}
		if _, body := testRequest(t, ts, "GET", "/public/nope", nil); body != "custom not-found" {
			t.Fatalf(body)
		}
		if _, body := testRequest(t, ts, "GET", "/private/nope", nil); body != "custom not-found" {
			t.Fatalf(body)
		}
		if _, body := testRequest(t, ts, "GET", "/private/resource/nope", nil); body != "custom not-found" {
			t.Fatalf(body)
		}
		if _, body := testRequest(t, ts, "GET", "/private_mw/nope", nil); body != "custom not-found" {
			t.Fatalf(body)
		}
		if _, body := testRequest(t, ts, "GET", "/private_mw/resource/nope", nil); body != "custom not-found" {
			t.Fatalf(body)
		}
		// check custom not-found on trailing slash routes
		if _, body := testRequest(t, ts, "GET", "/auth/", nil); body != "custom not-found" {
			t.Fatalf(body)
		}
	}

	t.Run("pre", func(t *testing.T) {
		r := NewRouter()
		r.NotFound(HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			rc.Write([]byte("custom not-found"))
		}))
		decorateRouter(r)
		testNotFound(t, r)
	})

	t.Run("post", func(t *testing.T) {
		r := NewRouter()
		decorateRouter(r)
		r.NotFound(HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			rc.Write([]byte("custom not-found"))
		}))
		testNotFound(t, r)
	})
}

func TestMuxWith(t *testing.T) {
	var cmwInit1, cmwHandler1 uint64
	var cmwInit2, cmwHandler2 uint64
	mw1 := func(next Handler) Handler {
		cmwInit1++
		return HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			cmwHandler1++
			ctx = context.WithValue(ctx, ctxKey{"inline1"}, "yes")
			next.ServeHTTP(ctx, rc)
		})
	}
	mw2 := func(next Handler) Handler {
		cmwInit2++
		return HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			cmwHandler2++
			ctx = context.WithValue(ctx, ctxKey{"inline2"}, "yes")
			next.ServeHTTP(ctx, rc)
		})
	}

	r := NewRouter()
	r.Get("/hi", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte("bye"))
	}))
	r.With(mw1).With(mw2).Get("/inline", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		v1 := ctx.Value(ctxKey{"inline1"}).(string)
		v2 := ctx.Value(ctxKey{"inline2"}).(string)
		rc.Write([]byte(fmt.Sprintf("inline %s %s", v1, v2)))
	}))

	ts := NewTestServer(r)
	defer ts.Close()

	if _, body := testRequest(t, ts, "GET", "/hi", nil); body != "bye" {
		t.Fatalf(body)
	}
	if _, body := testRequest(t, ts, "GET", "/inline", nil); body != "inline yes yes" {
		t.Fatalf(body)
	}
	if cmwInit1 != 1 {
		t.Fatalf("expecting cmwInit1 to be 1, got %d", cmwInit1)
	}
	if cmwHandler1 != 1 {
		t.Fatalf("expecting cmwHandler1 to be 1, got %d", cmwHandler1)
	}
	if cmwInit2 != 1 {
		t.Fatalf("expecting cmwInit2 to be 1, got %d", cmwInit2)
	}
	if cmwHandler2 != 1 {
		t.Fatalf("expecting cmwHandler2 to be 1, got %d", cmwHandler2)
	}
}

func TestRouterFromMuxWith(t *testing.T) {
	t.Parallel()

	r := NewRouter()

	with := r.With(func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			next.ServeHTTP(ctx, rc)
		})
	})

	with.Get("/with_middleware", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {}))

	ts := NewTestServer(with)
	defer ts.Close()

	// Without the fix this test was committed with, this causes a panic.
	testRequest(t, ts, http.MethodGet, "/with_middleware", nil)
}

func TestMuxMiddlewareStack(t *testing.T) {
	var stdmwInit, stdmwHandler uint64
	stdmw := func(next Handler) Handler {
		stdmwInit++
		return HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			stdmwHandler++
			next.ServeHTTP(ctx, rc)
		})
	}
	_ = stdmw

	var ctxmwInit, ctxmwHandler uint64
	ctxmw := func(next Handler) Handler {
		ctxmwInit++
		return HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			ctxmwHandler++
			ctx = context.WithValue(ctx, ctxKey{"count.ctxmwHandler"}, ctxmwHandler)
			next.ServeHTTP(ctx, rc)
		})
	}

	var inCtxmwInit, inCtxmwHandler uint64
	inCtxmw := func(next Handler) Handler {
		inCtxmwInit++
		return HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			inCtxmwHandler++
			next.ServeHTTP(ctx, rc)
		})
	}

	r := NewRouter()
	r.Use(stdmw)
	r.Use(ctxmw)
	r.Use(func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			if string(rc.Request.URI().Path()) == "/ping" {
				rc.Write([]byte("pong"))
				return
			}
			next.ServeHTTP(ctx, rc)
		})
	})

	var handlerCount uint64

	r.With(inCtxmw).Get("/", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		handlerCount++

		ctxmwHandlerCount := ctx.Value(ctxKey{"count.ctxmwHandler"}).(uint64)
		rc.Write([]byte(fmt.Sprintf("inits:%d reqs:%d ctxValue:%d", ctxmwInit, handlerCount, ctxmwHandlerCount)))
	}))

	r.Get("/hi", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte("wooot"))
	}))

	ts := NewTestServer(r)
	defer ts.Close()

	testRequest(t, ts, "GET", "/", nil)
	testRequest(t, ts, "GET", "/", nil)
	var body string
	_, body = testRequest(t, ts, "GET", "/", nil)
	if body != "inits:1 reqs:3 ctxValue:3" {
		t.Fatalf("got: '%s'", body)
	}

	_, body = testRequest(t, ts, "GET", "/ping", nil)
	if body != "pong" {
		t.Fatalf("got: '%s'", body)
	}
}

func TestMuxRouteGroups(t *testing.T) {
	var stdmwInit, stdmwHandler uint64

	stdmw := func(next Handler) Handler {
		stdmwInit++
		return HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			stdmwHandler++
			next.ServeHTTP(ctx, rc)
		})
	}

	var stdmwInit2, stdmwHandler2 uint64
	stdmw2 := func(next Handler) Handler {
		stdmwInit2++
		return HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			stdmwHandler2++
			next.ServeHTTP(ctx, rc)
		})
	}

	r := NewRouter()
	r.Group(func(r Router) {
		r.Use(stdmw)
		r.Get("/group", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			rc.Write([]byte("root group"))
		}))
	})
	r.Group(func(r Router) {
		r.Use(stdmw2)
		r.Get("/group2", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			rc.Write([]byte("root group2"))
		}))
	})

	ts := NewTestServer(r)
	defer ts.Close()

	// GET /group
	_, body := testRequest(t, ts, "GET", "/group", nil)
	if body != "root group" {
		t.Fatalf("got: '%s'", body)
	}
	if stdmwInit != 1 || stdmwHandler != 1 {
		t.Logf("stdmw counters failed, should be 1:1, got %d:%d", stdmwInit, stdmwHandler)
	}

	// GET /group2
	_, body = testRequest(t, ts, "GET", "/group2", nil)
	if body != "root group2" {
		t.Fatalf("got: '%s'", body)
	}
	if stdmwInit2 != 1 || stdmwHandler2 != 1 {
		t.Fatalf("stdmw2 counters failed, should be 1:1, got %d:%d", stdmwInit2, stdmwHandler2)
	}
}

func TestMuxBig(t *testing.T) {
	r := bigMux()

	ts := NewTestServer(r)
	defer ts.Close()

	var body, expected string

	_, body = testRequest(t, ts, "GET", "/favicon.ico", nil)
	if body != "fav" {
		t.Fatalf("got '%s'", body)
	}
	_, body = testRequest(t, ts, "GET", "/hubs/4/view", nil)
	if body != "/hubs/4/view reqid:1 session:anonymous" {
		t.Fatalf("got '%v'", body)
	}
	_, body = testRequest(t, ts, "GET", "/hubs/4/view/index.html", nil)
	if body != "/hubs/4/view/index.html reqid:1 session:anonymous" {
		t.Fatalf("got '%s'", body)
	}
	_, body = testRequest(t, ts, "POST", "/hubs/ethereumhub/view/index.html", nil)
	if body != "/hubs/ethereumhub/view/index.html reqid:1 session:anonymous" {
		t.Fatalf("got '%s'", body)
	}
	_, body = testRequest(t, ts, "GET", "/", nil)
	if body != "/ reqid:1 session:elvis" {
		t.Fatalf("got '%s'", body)
	}
	_, body = testRequest(t, ts, "GET", "/suggestions", nil)
	if body != "/suggestions reqid:1 session:elvis" {
		t.Fatalf("got '%s'", body)
	}
	_, body = testRequest(t, ts, "GET", "/woot/444/hiiii", nil)
	if body != "/woot/444/hiiii" {
		t.Fatalf("got '%s'", body)
	}
	_, body = testRequest(t, ts, "GET", "/hubs/123", nil)
	expected = "/hubs/123 reqid:1 session:elvis"
	if body != expected {
		t.Fatalf("expected:%s got:%s", expected, body)
	}
	_, body = testRequest(t, ts, "GET", "/hubs/123/touch", nil)
	if body != "/hubs/123/touch reqid:1 session:elvis" {
		t.Fatalf("got '%s'", body)
	}
	_, body = testRequest(t, ts, "GET", "/hubs/123/webhooks", nil)
	if body != "/hubs/123/webhooks reqid:1 session:elvis" {
		t.Fatalf("got '%s'", body)
	}
	_, body = testRequest(t, ts, "GET", "/hubs/123/posts", nil)
	if body != "/hubs/123/posts reqid:1 session:elvis" {
		t.Fatalf("got '%s'", body)
	}
	_, body = testRequest(t, ts, "GET", "/folders", nil)
	if body != "404 page not found" {
		t.Fatalf("got '%s'", body)
	}
	_, body = testRequest(t, ts, "GET", "/folders/", nil)
	if body != "/folders/ reqid:1 session:elvis" {
		t.Fatalf("got '%s'", body)
	}
	_, body = testRequest(t, ts, "GET", "/folders/public", nil)
	if body != "/folders/public reqid:1 session:elvis" {
		t.Fatalf("got '%s'", body)
	}
	_, body = testRequest(t, ts, "GET", "/folders/nothing", nil)
	if body != "404 page not found" {
		t.Fatalf("got '%s'", body)
	}
}

func bigMux() Router {
	var r *Mux
	var sr3 *Mux
	// var sr1, sr2, sr3, sr4, sr5, sr6 *Mux
	r = NewRouter()
	r.Use(func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			ctx = context.WithValue(ctx, ctxKey{"requestID"}, "1")
			next.ServeHTTP(ctx, rc)
		})
	})
	r.Use(func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			next.ServeHTTP(ctx, rc)
		})
	})
	r.Group(func(r Router) {
		r.Use(func(next Handler) Handler {
			return HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
				ctx = context.WithValue(ctx, ctxKey{"session.user"}, "anonymous")
				next.ServeHTTP(ctx, rc)
			})
		})
		r.Get("/favicon.ico", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			rc.Write([]byte("fav"))
		}))
		r.Get("/hubs/{hubID}/view", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {

			s := fmt.Sprintf("/hubs/%s/view reqid:%s session:%s", URLParam(rc, "hubID"),
				ctx.Value(ctxKey{"requestID"}), ctx.Value(ctxKey{"session.user"}))
			rc.Write([]byte(s))
		}))
		r.Get("/hubs/{hubID}/view/*", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {

			s := fmt.Sprintf("/hubs/%s/view/%s reqid:%s session:%s", URLParamFromCtx(rc, "hubID"),
				URLParam(rc, "*"), ctx.Value(ctxKey{"requestID"}), ctx.Value(ctxKey{"session.user"}))
			rc.Write([]byte(s))
		}))
		r.Post("/hubs/{hubSlug}/view/*", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {

			s := fmt.Sprintf("/hubs/%s/view/%s reqid:%s session:%s", URLParamFromCtx(rc, "hubSlug"),
				URLParam(rc, "*"), ctx.Value(ctxKey{"requestID"}), ctx.Value(ctxKey{"session.user"}))
			rc.Write([]byte(s))
		}))
	})
	r.Group(func(r Router) {
		r.Use(func(next Handler) Handler {
			return HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
				ctx = context.WithValue(ctx, ctxKey{"session.user"}, "elvis")
				next.ServeHTTP(ctx, rc)
			})
		})
		r.Get("/", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {

			s := fmt.Sprintf("/ reqid:%s session:%s", ctx.Value(ctxKey{"requestID"}), ctx.Value(ctxKey{"session.user"}))
			rc.Write([]byte(s))
		}))
		r.Get("/suggestions", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {

			s := fmt.Sprintf("/suggestions reqid:%s session:%s", ctx.Value(ctxKey{"requestID"}), ctx.Value(ctxKey{"session.user"}))
			rc.Write([]byte(s))
		}))

		r.Get("/woot/{wootID}/*", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			s := fmt.Sprintf("/woot/%s/%s", URLParam(rc, "wootID"), URLParam(rc, "*"))
			rc.Write([]byte(s))
		}))

		r.Route("/hubs", func(r Router) {
			_ = r.(*Mux) // sr1
			r.Route("/{hubID}", func(r Router) {
				_ = r.(*Mux) // sr2
				r.Get("/", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {

					s := fmt.Sprintf("/hubs/%s reqid:%s session:%s",
						URLParam(rc, "hubID"), ctx.Value(ctxKey{"requestID"}), ctx.Value(ctxKey{"session.user"}))
					rc.Write([]byte(s))
				}))
				r.Get("/touch", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {

					s := fmt.Sprintf("/hubs/%s/touch reqid:%s session:%s", URLParam(rc, "hubID"),
						ctx.Value(ctxKey{"requestID"}), ctx.Value(ctxKey{"session.user"}))
					rc.Write([]byte(s))
				}))

				sr3 = NewRouter()
				sr3.Get("/", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {

					s := fmt.Sprintf("/hubs/%s/webhooks reqid:%s session:%s", URLParam(rc, "hubID"),
						ctx.Value(ctxKey{"requestID"}), ctx.Value(ctxKey{"session.user"}))
					rc.Write([]byte(s))
				}))
				sr3.Route("/{webhookID}", func(r Router) {
					_ = r.(*Mux) // sr4
					r.Get("/", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {

						s := fmt.Sprintf("/hubs/%s/webhooks/%s reqid:%s session:%s", URLParam(rc, "hubID"),
							URLParam(rc, "webhookID"), ctx.Value(ctxKey{"requestID"}), ctx.Value(ctxKey{"session.user"}))
						rc.Write([]byte(s))
					}))
				})

				r.Mount("/webhooks", Chain(func(next Handler) Handler {
					return HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
						next.ServeHTTP(context.WithValue(ctx, ctxKey{"hook"}, true), rc)
					})
				}).Handler(sr3))

				r.Route("/posts", func(r Router) {
					_ = r.(*Mux) // sr5
					r.Get("/", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {

						s := fmt.Sprintf("/hubs/%s/posts reqid:%s session:%s", URLParam(rc, "hubID"),
							ctx.Value(ctxKey{"requestID"}), ctx.Value(ctxKey{"session.user"}))
						rc.Write([]byte(s))
					}))
				})
			})
		})

		r.Route("/folders/", func(r Router) {
			_ = r.(*Mux) // sr6
			r.Get("/", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {

				s := fmt.Sprintf("/folders/ reqid:%s session:%s",
					ctx.Value(ctxKey{"requestID"}), ctx.Value(ctxKey{"session.user"}))
				rc.Write([]byte(s))
			}))
			r.Get("/public", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {

				s := fmt.Sprintf("/folders/public reqid:%s session:%s",
					ctx.Value(ctxKey{"requestID"}), ctx.Value(ctxKey{"session.user"}))
				rc.Write([]byte(s))
			}))
		})
	})

	return r
}

func TestMuxSubroutesBasic(t *testing.T) {
	hIndex := HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte("index"))
	})
	hArticlesList := HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte("articles-list"))
	})
	hSearchArticles := HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte("search-articles"))
	})
	hGetArticle := HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte(fmt.Sprintf("get-article:%s", URLParam(rc, "id"))))
	})
	hSyncArticle := HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte(fmt.Sprintf("sync-article:%s", URLParam(rc, "id"))))
	})

	r := NewRouter()
	// var rr1, rr2 *Mux
	r.Get("/", hIndex)
	r.Route("/articles", func(r Router) {
		// rr1 = r.(*Mux)
		r.Get("/", hArticlesList)
		r.Get("/search", hSearchArticles)
		r.Route("/{id}", func(r Router) {
			// rr2 = r.(*Mux)
			r.Get("/", hGetArticle)
			r.Get("/sync", hSyncArticle)
		})
	})

	// log.Println("~~~~~~~~~")
	// log.Println("~~~~~~~~~")
	// debugPrintTree(0, 0, r.tree, 0)
	// log.Println("~~~~~~~~~")
	// log.Println("~~~~~~~~~")

	// log.Println("~~~~~~~~~")
	// log.Println("~~~~~~~~~")
	// debugPrintTree(0, 0, rr1.tree, 0)
	// log.Println("~~~~~~~~~")
	// log.Println("~~~~~~~~~")

	// log.Println("~~~~~~~~~")
	// log.Println("~~~~~~~~~")
	// debugPrintTree(0, 0, rr2.tree, 0)
	// log.Println("~~~~~~~~~")
	// log.Println("~~~~~~~~~")

	ts := NewTestServer(r)
	defer ts.Close()

	var body, expected string

	_, body = testRequest(t, ts, "GET", "/", nil)
	expected = "index"
	if body != expected {
		t.Fatalf("expected:%s got:%s", expected, body)
	}
	_, body = testRequest(t, ts, "GET", "/articles", nil)
	expected = "articles-list"
	if body != expected {
		t.Fatalf("expected:%s got:%s", expected, body)
	}
	_, body = testRequest(t, ts, "GET", "/articles/search", nil)
	expected = "search-articles"
	if body != expected {
		t.Fatalf("expected:%s got:%s", expected, body)
	}
	_, body = testRequest(t, ts, "GET", "/articles/123", nil)
	expected = "get-article:123"
	if body != expected {
		t.Fatalf("expected:%s got:%s", expected, body)
	}
	_, body = testRequest(t, ts, "GET", "/articles/123/sync", nil)
	expected = "sync-article:123"
	if body != expected {
		t.Fatalf("expected:%s got:%s", expected, body)
	}
}

func TestMuxSubroutes(t *testing.T) {
	hHubView1 := HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte("hub1"))
	})
	hHubView2 := HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte("hub2"))
	})
	hHubView3 := HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte("hub3"))
	})
	hAccountView1 := HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte("account1"))
	})
	hAccountView2 := HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte("account2"))
	})

	r := NewRouter()
	r.Get("/hubs/{hubID}/view", hHubView1)
	r.Get("/hubs/{hubID}/view/*", hHubView2)

	sr := NewRouter()
	sr.Get("/", hHubView3)
	r.Mount("/hubs/{hubID}/users", sr)
	r.Get("/hubs/{hubID}/users/", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte("hub3 override"))
	}))

	sr3 := NewRouter()
	sr3.Get("/", hAccountView1)
	sr3.Get("/hi", hAccountView2)

	// var sr2 *Mux
	r.Route("/accounts/{accountID}", func(r Router) {
		_ = r.(*Mux) // sr2
		// r.Get("/", hAccountView1)
		r.Mount("/", sr3)
	})

	// This is the same as the r.Route() call mounted on sr2
	// sr2 := NewRouter()
	// sr2.Mount("/", sr3)
	// r.Mount("/accounts/{accountID}", sr2)

	ts := NewTestServer(r)
	defer ts.Close()

	var body, expected string

	_, body = testRequest(t, ts, "GET", "/hubs/123/view", nil)
	expected = "hub1"
	if body != expected {
		t.Fatalf("expected:%s got:%s", expected, body)
	}
	_, body = testRequest(t, ts, "GET", "/hubs/123/view/index.html", nil)
	expected = "hub2"
	if body != expected {
		t.Fatalf("expected:%s got:%s", expected, body)
	}
	_, body = testRequest(t, ts, "GET", "/hubs/123/users", nil)
	expected = "hub3"
	if body != expected {
		t.Fatalf("expected:%s got:%s", expected, body)
	}
	_, body = testRequest(t, ts, "GET", "/hubs/123/users/", nil)
	expected = "hub3 override"
	if body != expected {
		t.Fatalf("expected:%s got:%s", expected, body)
	}
	_, body = testRequest(t, ts, "GET", "/accounts/44", nil)
	expected = "account1"
	if body != expected {
		t.Fatalf("request:%s expected:%s got:%s", "GET /accounts/44", expected, body)
	}
	_, body = testRequest(t, ts, "GET", "/accounts/44/hi", nil)
	expected = "account2"
	if body != expected {
		t.Fatalf("expected:%s got:%s", expected, body)
	}

	// Test that we're building the routingPatterns properly
	router := r
	rc := fasthttp.RequestCtx{}
	rc.Request.Header.SetMethod("GET")
	rc.Request.SetRequestURI("/accounts/44/hi")

	rctx := NewRouteContext()
	rc.SetUserValue(routeUserValueKey, rctx)

	router.ServeHTTP(context.Background(), &rc)

	expected = "account2"
	if string(rc.Response.Body()) != expected {
		t.Fatalf("expected:%s got:%s", expected, string(rc.Response.Body()))
	}

	routePatterns := rctx.RoutePatterns
	if len(rctx.RoutePatterns) != 3 {
		t.Fatalf("expected 3 routing patterns, got:%d", len(rctx.RoutePatterns))
	}
	expected = "/accounts/{accountID}/*"
	if routePatterns[0] != expected {
		t.Fatalf("routePattern, expected:%s got:%s", expected, routePatterns[0])
	}
	expected = "/*"
	if routePatterns[1] != expected {
		t.Fatalf("routePattern, expected:%s got:%s", expected, routePatterns[1])
	}
	expected = "/hi"
	if routePatterns[2] != expected {
		t.Fatalf("routePattern, expected:%s got:%s", expected, routePatterns[2])
	}

}

func TestSingleHandler(t *testing.T) {
	h := HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		name := URLParam(rc, "name")
		rc.Write([]byte("hi " + name))
	})

	rc := fasthttp.RequestCtx{}
	rc.Request.Header.SetMethod("GET")
	rc.Request.SetRequestURI("/")

	rctx := NewRouteContext()
	rc.SetUserValue(routeUserValueKey, rctx)
	rctx.URLParams.Add("name", "joe")

	h.ServeHTTP(context.Background(), &rc)

	expected := "hi joe"
	if string(rc.Response.Body()) != expected {
		t.Fatalf("expected:%s got:%s", expected, string(rc.Response.Body()))
	}
}

// TODO: a Router wrapper test..
//
// type ACLMux struct {
// 	*Mux
// 	XX string
// }
//
// func NewACLMux() *ACLMux {
// 	return &ACLMux{Mux: NewRouter(), XX: "hihi"}
// }
//
// // TODO: this should be supported...
// func TestWoot(t *testing.T) {
// 	var r Router = NewRouter()
//
// 	var r2 Router = NewACLMux() //NewRouter()
// 	r2.Get("/hi", func(ctx context.Context, rc *fasthttp.RequestCtx) {
// 		rc.Write([]byte("hi"))
// 	})
//
// 	r.Mount("/", r2)
// }

func TestServeHTTPExistingContext(t *testing.T) {
	r := NewRouter()
	r.Get("/hi", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		s, _ := ctx.Value(ctxKey{"testCtx"}).(string)
		rc.Write([]byte(s))
	}))
	r.NotFound(HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		s, _ := ctx.Value(ctxKey{"testCtx"}).(string)
		rc.Response.SetStatusCode(404)
		rc.Write([]byte(s))
	}))

	testcases := []struct {
		Ctx            context.Context
		Method         string
		Path           string
		ExpectedBody   string
		ExpectedStatus int
	}{
		{
			Method:         "GET",
			Path:           "/hi",
			Ctx:            context.WithValue(context.Background(), ctxKey{"testCtx"}, "hi ctx"),
			ExpectedStatus: 200,
			ExpectedBody:   "hi ctx",
		},
		{
			Method:         "GET",
			Path:           "/hello",
			Ctx:            context.WithValue(context.Background(), ctxKey{"testCtx"}, "nothing here ctx"),
			ExpectedStatus: 404,
			ExpectedBody:   "nothing here ctx",
		},
	}

	for _, tc := range testcases {
		rc := fasthttp.RequestCtx{}
		rc.Request.SetRequestURI(tc.Path)
		rc.Request.Header.SetMethod(tc.Method)

		r.ServeHTTP(tc.Ctx, &rc)

		if rc.Response.StatusCode() != tc.ExpectedStatus {
			t.Fatalf("%v != %v", tc.ExpectedStatus, rc.Response.StatusCode())
		}
		if string(rc.Response.Body()) != tc.ExpectedBody {
			t.Fatalf("%s != %s", tc.ExpectedBody, string(rc.Response.Body()))
		}
	}
}

func TestNestedGroups(t *testing.T) {
	handlerPrintCounter := func(ctx context.Context, rc *fasthttp.RequestCtx) {
		counter, _ := ctx.Value(ctxKey{"counter"}).(int)
		rc.Write([]byte(fmt.Sprintf("%v", counter)))
	}

	mwIncreaseCounter := func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {

			counter, _ := ctx.Value(ctxKey{"counter"}).(int)
			counter++
			ctx = context.WithValue(ctx, ctxKey{"counter"}, counter)
			next.ServeHTTP(ctx, rc)
		})
	}

	// Each route represents value of its counter (number of applied middlewares).
	r := NewRouter() // counter == 0
	r.Get("/0", HandlerFunc(handlerPrintCounter))
	r.Group(func(r Router) {
		r.Use(mwIncreaseCounter) // counter == 1
		r.Get("/1", HandlerFunc(handlerPrintCounter))

		// r.Handle(GET, "/2", Chain(mwIncreaseCounter).HandlerFunc(handlerPrintCounter))
		r.With(mwIncreaseCounter).Get("/2", HandlerFunc(handlerPrintCounter))

		r.Group(func(r Router) {
			r.Use(mwIncreaseCounter, mwIncreaseCounter) // counter == 3
			r.Get("/3", HandlerFunc(handlerPrintCounter))
		})
		r.Route("/", func(r Router) {
			r.Use(mwIncreaseCounter, mwIncreaseCounter) // counter == 3

			// r.Handle(GET, "/4", Chain(mwIncreaseCounter).HandlerFunc(handlerPrintCounter))
			r.With(mwIncreaseCounter).Get("/4", HandlerFunc(handlerPrintCounter))

			r.Group(func(r Router) {
				r.Use(mwIncreaseCounter, mwIncreaseCounter) // counter == 5
				r.Get("/5", HandlerFunc(handlerPrintCounter))
				// r.Handle(GET, "/6", Chain(mwIncreaseCounter).HandlerFunc(handlerPrintCounter))
				r.With(mwIncreaseCounter).Get("/6", HandlerFunc(handlerPrintCounter))

			})
		})
	})

	ts := NewTestServer(r)
	defer ts.Close()

	for _, route := range []string{"0", "1", "2", "3", "4", "5", "6"} {
		if _, body := testRequest(t, ts, "GET", "/"+route, nil); body != route {
			t.Errorf("expected %v, got %v", route, body)
		}
	}
}

func TestMiddlewarePanicOnLateUse(t *testing.T) {
	handler := func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte("hello\n"))
	}

	mw := func(next Handler) Handler {
		return HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			next.ServeHTTP(ctx, rc)
		})
	}

	defer func() {
		if recover() == nil {
			t.Error("expected panic()")
		}
	}()

	r := NewRouter()
	r.Get("/", HandlerFunc(handler))
	r.Use(mw) // Too late to apply middleware, we're expecting panic().
}

func TestMountingExistingPath(t *testing.T) {
	handler := func(ctx context.Context, rc *fasthttp.RequestCtx) {}

	defer func() {
		if recover() == nil {
			t.Error("expected panic()")
		}
	}()

	r := NewRouter()
	r.Get("/", HandlerFunc(handler))
	r.Mount("/hi", HandlerFunc(handler))
	r.Mount("/hi", HandlerFunc(handler))
}

func TestMountingSimilarPattern(t *testing.T) {
	r := NewRouter()
	r.Get("/hi", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte("bye"))
	}))

	r2 := NewRouter()
	r2.Get("/", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte("foobar"))
	}))

	r3 := NewRouter()
	r3.Get("/", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte("foo"))
	}))

	r.Mount("/foobar", r2)
	r.Mount("/foo", r3)

	ts := NewTestServer(r)
	defer ts.Close()

	if _, body := testRequest(t, ts, "GET", "/hi", nil); body != "bye" {
		t.Fatalf(body)
	}
}

func TestMuxEmptyParams(t *testing.T) {
	r := NewRouter()
	r.Get(`/users/{x}/{y}/{z}`, HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		x := URLParam(rc, "x")
		y := URLParam(rc, "y")
		z := URLParam(rc, "z")
		rc.Write([]byte(fmt.Sprintf("%s-%s-%s", x, y, z)))
	}))

	ts := NewTestServer(r)
	defer ts.Close()

	if _, body := testRequest(t, ts, "GET", "/users/a/b/c", nil); body != "a-b-c" {
		t.Fatalf(body)
	}
	if _, body := testRequest(t, ts, "GET", "/users///c", nil); body != "--c" {
		t.Fatalf(body)
	}
}

func TestMuxMissingParams(t *testing.T) {
	r := NewRouter()
	r.Get(`/user/{userId:\d+}`, HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		userID := URLParam(rc, "userId")
		rc.Write([]byte(fmt.Sprintf("userId = '%s'", userID)))
	}))
	r.NotFound(HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Response.SetStatusCode(404)
		rc.Write([]byte("nothing here"))
	}))

	ts := NewTestServer(r)
	defer ts.Close()

	if _, body := testRequest(t, ts, "GET", "/user/123", nil); body != "userId = '123'" {
		t.Fatalf(body)
	}
	if _, body := testRequest(t, ts, "GET", "/user/", nil); body != "nothing here" {
		t.Fatalf(body)
	}
}

func TestMuxWildcardRoute(t *testing.T) {
	handler := func(ctx context.Context, rc *fasthttp.RequestCtx) {}

	defer func() {
		if recover() == nil {
			t.Error("expected panic()")
		}
	}()

	r := NewRouter()
	r.Get("/*/wildcard/must/be/at/end", HandlerFunc(handler))
}

func TestMuxWildcardRouteCheckTwo(t *testing.T) {
	handler := func(ctx context.Context, rc *fasthttp.RequestCtx) {}

	defer func() {
		if recover() == nil {
			t.Error("expected panic()")
		}
	}()

	r := NewRouter()
	r.Get("/*/wildcard/{must}/be/at/end", HandlerFunc(handler))
}

func TestMuxRegexp(t *testing.T) {
	r := NewRouter()
	r.Route("/{param:[0-9]*}/test", func(r Router) {
		r.Get("/", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			rc.Write([]byte(fmt.Sprintf("Hi: %s", URLParam(rc, "param"))))
		}))
	})

	ts := NewTestServer(r)
	defer ts.Close()

	if _, body := testRequest(t, ts, "GET", "//test", nil); body != "Hi: " {
		t.Fatalf(body)
	}
}

func TestMuxRegexp2(t *testing.T) {
	r := NewRouter()
	r.Get("/foo-{suffix:[a-z]{2,3}}.json", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte(URLParam(rc, "suffix")))
	}))
	ts := NewTestServer(r)
	defer ts.Close()

	if _, body := testRequest(t, ts, "GET", "/foo-.json", nil); body != "" {
		t.Fatalf(body)
	}
	if _, body := testRequest(t, ts, "GET", "/foo-abc.json", nil); body != "abc" {
		t.Fatalf(body)
	}
}

func TestMuxRegexp3(t *testing.T) {
	r := NewRouter()
	r.Get("/one/{firstId:[a-z0-9-]+}/{secondId:[a-z]+}/first", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte("first"))
	}))
	r.Get("/one/{firstId:[a-z0-9-_]+}/{secondId:[0-9]+}/second", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte("second"))
	}))
	r.Delete("/one/{firstId:[a-z0-9-_]+}/{secondId:[0-9]+}/second", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte("third"))
	}))

	r.Route("/one", func(r Router) {
		r.Get("/{dns:[a-z-0-9_]+}", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			rc.Write([]byte("_"))
		}))
		r.Get("/{dns:[a-z-0-9_]+}/info", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			rc.Write([]byte("_"))
		}))
		r.Delete("/{id:[0-9]+}", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			rc.Write([]byte("forth"))
		}))
	})

	ts := NewTestServer(r)
	defer ts.Close()

	if _, body := testRequest(t, ts, "GET", "/one/hello/peter/first", nil); body != "first" {
		t.Fatalf(body)
	}
	if _, body := testRequest(t, ts, "GET", "/one/hithere/123/second", nil); body != "second" {
		t.Fatalf(body)
	}
	if _, body := testRequest(t, ts, "DELETE", "/one/hithere/123/second", nil); body != "third" {
		t.Fatalf(body)
	}
	if _, body := testRequest(t, ts, "DELETE", "/one/123", nil); body != "forth" {
		t.Fatalf(body)
	}
}

func TestMuxSubrouterWildcardParam(t *testing.T) {
	h := HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		fmt.Fprintf(rc, "param:%v *:%v", URLParam(rc, "param"), URLParam(rc, "*"))
	}))

	r := NewRouter()

	r.Get("/bare/{param}", h)
	r.Get("/bare/{param}/*", h)

	r.Route("/case0", func(r Router) {
		r.Get("/{param}", h)
		r.Get("/{param}/*", h)
	})

	ts := httptest.NewServer(r)
	defer ts.Close()

	if _, body := testRequest(t, ts, "GET", "/bare/hi", nil); body != "param:hi *:" {
		t.Fatalf(body)
	}
	if _, body := testRequest(t, ts, "GET", "/bare/hi/yes", nil); body != "param:hi *:yes" {
		t.Fatalf(body)
	}
	if _, body := testRequest(t, ts, "GET", "/case0/hi", nil); body != "param:hi *:" {
		t.Fatalf(body)
	}
	if _, body := testRequest(t, ts, "GET", "/case0/hi/yes", nil); body != "param:hi *:yes" {
		t.Fatalf(body)
	}
}

func TestMuxContextIsThreadSafe(t *testing.T) {
	router := NewRouter()
	router.Get("/{id}", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		ctx, cancel := context.WithTimeout(ctx, 1*time.Millisecond)
		defer cancel()

		<-ctx.Done()
	}))

	wg := sync.WaitGroup{}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10000; j++ {
				rc := fasthttp.RequestCtx{}
				rc.Request.Header.SetMethod("GET")
				rc.Request.SetRequestURI("/ok")

				ctx, cancel := context.WithCancel(context.Background())

				go func() {
					cancel()
				}()
				router.ServeHTTP(ctx, &rc)
			}
		}()
	}
	wg.Wait()
}

func TestEscapedURLParams(t *testing.T) {
	m := NewRouter()
	m.Get("/api/{identifier}/{region}/{size}/{rotation}/*", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Response.SetStatusCode(200)
		rctx := RouteContext(rc)
		if rctx == nil {
			t.Error("no context")
			return
		}
		identifier := URLParam(rc, "identifier")
		if identifier != "http:%2f%2fexample.com%2fimage.png" {
			t.Errorf("identifier path parameter incorrect %s", identifier)
			return
		}
		region := URLParam(rc, "region")
		if region != "full" {
			t.Errorf("region path parameter incorrect %s", region)
			return
		}
		size := URLParam(rc, "size")
		if size != "max" {
			t.Errorf("size path parameter incorrect %s", size)
			return
		}
		rotation := URLParam(rc, "rotation")
		if rotation != "0" {
			t.Errorf("rotation path parameter incorrect %s", rotation)
			return
		}
		rc.Write([]byte("success"))
	}))

	ts := NewTestServer(m)
	defer ts.Close()

	if _, body := testRequest(t, ts, "GET", "/api/http:%2f%2fexample.com%2fimage.png/full/max/0/color.png", nil); body != "success" {
		t.Fatalf(body)
	}
}

func TestCustomHTTPMethod(t *testing.T) {
	// first we must register this method to be accepted, then we
	// can define method handlers on the router below
	RegisterMethod("BOO")

	r := NewRouter()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("."))
	})

	// note the custom BOO method for route /hi
	r.MethodFunc("BOO", "/hi", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("custom method"))
	})

	ts := httptest.NewServer(r)
	defer ts.Close()

	if _, body := testRequest(t, ts, "GET", "/", nil); body != "." {
		t.Fatalf(body)
	}
	if _, body := testRequest(t, ts, "BOO", "/hi", nil); body != "custom method" {
		t.Fatalf(body)
	}
}

func TestMuxMatch(t *testing.T) {
	r := NewRouter()
	r.Get("/hi", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Response.Header.Set("X-Test", "yes")
		rc.Write([]byte("bye"))
	}))
	r.Route("/articles", func(r Router) {
		r.Get("/{id}", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			id := URLParam(rc, "id")
			rc.Response.Header.Set("X-Article", id)
			rc.Write([]byte("article:" + id))
		}))
	})
	r.Route("/users", func(r Router) {
		r.Head("/{id}", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			rc.Response.Header.Set("X-User", "-")
			rc.Write([]byte("user"))
		}))
		r.Get("/{id}", HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			id := URLParam(rc, "id")
			rc.Response.Header.Set("X-User", id)
			rc.Write([]byte("user:" + id))
		}))
	})

	tctx := NewRouteContext()

	tctx.Reset()
	if r.Match(tctx, "GET", "/users/1") == false {
		t.Fatal("expecting to find match for route:", "GET", "/users/1")
	}

	tctx.Reset()
	if r.Match(tctx, "HEAD", "/articles/10") == true {
		t.Fatal("not expecting to find match for route:", "HEAD", "/articles/10")
	}
}

func TestServerBaseContext(t *testing.T) {
	r := NewRouter()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		baseYes := r.Context().Value(ctxKey{"base"}).(string)
		if _, ok := r.Context().Value(http.ServerContextKey).(*http.Server); !ok {
			panic("missing server context")
		}
		if _, ok := r.Context().Value(http.LocalAddrContextKey).(net.Addr); !ok {
			panic("missing local addr context")
		}
		w.Write([]byte(baseYes))
	})

	// Setup http Server with a base context
	ctx := context.WithValue(context.Background(), ctxKey{"base"}, "yes")
	ts := httptest.NewUnstartedServer(r)
	ts.Config.BaseContext = func(_ net.Listener) context.Context {
		return ctx
	}
	ts.Start()

	defer ts.Close()

	if _, body := testRequest(t, ts, "GET", "/", nil); body != "yes" {
		t.Fatalf(body)
	}
}

func testRequest(t *testing.T, ts *TestServer, method, path string, body io.Reader) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, body)
	if err != nil {
		t.Fatal(err)
		return nil, ""
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
		return nil, ""
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
		return nil, ""
	}
	defer resp.Body.Close()

	return resp, string(respBody)
}

func testHandler(h Handler, method, path string) string {
	rc := fasthttp.RequestCtx{}
	rc.Request.Header.SetMethod(method)
	rc.Request.SetRequestURI(path)

	h.ServeHTTP(context.Background(), &rc)
	return string(rc.Response.Body())
}

type testFileSystem struct {
	open func(name string) (http.File, error)
}

func (fs *testFileSystem) Open(name string) (http.File, error) {
	return fs.open(name)
}

type testFile struct {
	name     string
	contents []byte
}

func (tf *testFile) Close() error {
	return nil
}

func (tf *testFile) Read(p []byte) (n int, err error) {
	copy(p, tf.contents)
	return len(p), nil
}

func (tf *testFile) Seek(offset int64, whence int) (int64, error) {
	return 0, nil
}

func (tf *testFile) Readdir(count int) ([]os.FileInfo, error) {
	stat, _ := tf.Stat()
	return []os.FileInfo{stat}, nil
}

func (tf *testFile) Stat() (os.FileInfo, error) {
	return &testFileInfo{tf.name, int64(len(tf.contents))}, nil
}

type testFileInfo struct {
	name string
	size int64
}

func (tfi *testFileInfo) Name() string       { return tfi.name }
func (tfi *testFileInfo) Size() int64        { return tfi.size }
func (tfi *testFileInfo) Mode() os.FileMode  { return 0755 }
func (tfi *testFileInfo) ModTime() time.Time { return time.Now() }
func (tfi *testFileInfo) IsDir() bool        { return false }
func (tfi *testFileInfo) Sys() interface{}   { return nil }

type ctxKey struct {
	name string
}

func (k ctxKey) String() string {
	return "context value " + k.name
}

func BenchmarkMux(b *testing.B) {
	h1 := HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {})
	h2 := HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {})
	h3 := HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {})
	h4 := HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {})
	h5 := HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {})
	h6 := HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {})

	mx := NewRouter()
	mx.Get("/", h1)
	mx.Get("/hi", h2)
	mx.Get("/sup/{id}/and/{this}", h3)
	mx.Get("/sup/{id}/{bar:foo}/{this}", h3)

	mx.Route("/sharing/{x}/{hash}", func(mx Router) {
		mx.Get("/", h4)          // subrouter-1
		mx.Get("/{network}", h5) // subrouter-1
		mx.Get("/twitter", h5)
		mx.Route("/direct", func(mx Router) {
			mx.Get("/", h6) // subrouter-2
			mx.Get("/download", h6)
		})
	})

	routes := []string{
		"/",
		"/hi",
		"/sup/123/and/this",
		"/sup/123/foo/this",
		"/sharing/z/aBc",                 // subrouter-1
		"/sharing/z/aBc/twitter",         // subrouter-1
		"/sharing/z/aBc/direct",          // subrouter-2
		"/sharing/z/aBc/direct/download", // subrouter-2
	}

	for _, path := range routes {
		b.Run("route:"+path, func(b *testing.B) {
			rc := fasthttp.RequestCtx{}
			rc.Request.Header.SetMethod("GET")
			rc.Request.SetRequestURI(path)

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				mx.ServeHTTP(context.Background(), &rc)
			}
		})
	}
}
