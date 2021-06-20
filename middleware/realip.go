package middleware

// Ported from Goji's middleware, source:
// https://github.com/zenazn/goji/tree/master/web/middleware

import (
	"context"
	"net"
	"net/http"
	"strings"

	"github.com/swaggest/fchi"
	"github.com/valyala/fasthttp"
)

var xForwardedFor = http.CanonicalHeaderKey("X-Forwarded-For")
var xRealIP = http.CanonicalHeaderKey("X-Real-IP")

// RealIP is a middleware that sets a http.Request's RemoteAddr to the results
// of parsing either the X-Real-IP header or the X-Forwarded-For header (in that
// order).
//
// This middleware should be inserted fairly early in the middleware stack to
// ensure that subsequent layers (e.g., request loggers) which examine the
// RemoteAddr will see the intended value.
//
// You should only use this middleware if you can trust the headers passed to
// you (in particular, the two headers this middleware uses), for example
// because you have placed a reverse proxy like HAProxy or nginx in front of
// chi. If your reverse proxies are configured to pass along arbitrary header
// values from the client, or if you use this middleware without a reverse
// proxy, malicious clients will be able to make you very sad (or, depending on
// how you're using RemoteAddr, vulnerable to an attack of some sort).
func RealIP(next fchi.Handler) fchi.Handler {
	fn := func(ctx context.Context, rc *fasthttp.RequestCtx) {
		if rip := realIP(rc); rip != "" {
			rc.SetRemoteAddr(&net.IPAddr{IP: net.ParseIP(rip)})
		}
		next.ServeHTTP(ctx, rc)
	}

	return fchi.HandlerFunc(fn)
}

func realIP(rc *fasthttp.RequestCtx) string {
	var ip string

	if xrip := rc.Request.Header.Peek(xRealIP); len(xrip) > 0 {
		ip = string(xrip)
	} else if xff := rc.Request.Header.Peek(xForwardedFor); len(xff) > 0 {
		i := strings.Index(string(xff), ", ")
		if i == -1 {
			i = len(xff)
		}
		ip = string(xff[:i])
	}
	return ip
}
