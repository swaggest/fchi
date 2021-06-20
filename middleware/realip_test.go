package middleware

import (
	"context"
	"testing"

	"github.com/swaggest/fchi"
	"github.com/valyala/fasthttp"
)

func TestXRealIP(t *testing.T) {
	req := &fasthttp.RequestCtx{}
	req.Request.URI().SetPath("/")
	req.Request.Header.Set(xRealIP, "100.100.100.100")

	r := fchi.NewRouter()
	r.Use(RealIP)

	realIP := ""
	r.Get("/", fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		realIP = rc.RemoteAddr().String()
	}))
	r.ServeHTTP(context.TODO(), req)

	if req.Response.StatusCode() != 200 {
		t.Fatal("Response Code should be 200")
	}

	if realIP != "100.100.100.100" {
		t.Fatal("Test get real IP error.")
	}
}

func TestXForwardForIP(t *testing.T) {
	req := &fasthttp.RequestCtx{}
	req.Request.URI().SetPath("/")
	req.Request.Header.Set(xForwardedFor, "100.100.100.100")

	r := fchi.NewRouter()
	r.Use(RealIP)

	realIP := ""
	r.Get("/", fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		realIP = rc.RemoteAddr().String()
	}))
	r.ServeHTTP(context.TODO(), req)

	if req.Response.StatusCode() != 200 {
		t.Fatal("Response Code should be 200")
	}

	if realIP != "100.100.100.100" {
		t.Fatal("Test get real IP error.")
	}
}

func TestXForwardForXRealIPPrecedence(t *testing.T) {
	req := &fasthttp.RequestCtx{}
	req.Request.URI().SetPath("/")
	req.Request.Header.Set(xForwardedFor, "0.0.0.0")
	req.Request.Header.Set(xRealIP, "100.100.100.100")

	r := fchi.NewRouter()
	r.Use(RealIP)

	realIP := ""
	r.Get("/", fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		realIP = rc.RemoteAddr().String()
	}))
	r.ServeHTTP(context.TODO(), req)

	if req.Response.StatusCode() != 200 {
		t.Fatal("Response Code should be 200")
	}

	if realIP != "100.100.100.100" {
		t.Fatal("Test get real IP error.")
	}
}
