package middleware

import (
	"crypto/subtle"
	"fmt"
	"strings"

	"github.com/swaggest/fchi"
	"github.com/valyala/fasthttp"
)

// BasicAuth implements a simple middleware handler for adding basic http auth to a route.
func BasicAuth(realm string, creds map[string]string) func(next fchi.Handler) fchi.Handler {
	return func(next fchi.Handler) fchi.Handler {
		return fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
			user, pass, ok := basicAuth(rc)
			if !ok {
				basicAuthFailed(rc, realm)
				return
			}

			credPass, credUserOk := creds[user]
			if !credUserOk || subtle.ConstantTimeCompare([]byte(pass), []byte(credPass)) != 1 {
				basicAuthFailed(rc, realm)
				return
			}

			next.ServeHTTP(ctx, rc)
		})
	}
}

func basicAuthFailed(rc *fasthttp.RequestCtx, realm string) {
	rc.Response.Header.Add("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, realm))
	rc.SetStatusCode(fasthttp.StatusUnauthorized)
}

func basicAuth(rc *fasthttp.RequestCtx) (username, password string, ok bool) {
	auth := rc.Request.Header.Peek("Authorization")
	if len(auth) == 0 {
		return
	}
	return parseBasicAuth(string(auth))
}

// parseBasicAuth parses an HTTP Basic Authentication string.
// "Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==" returns ("Aladdin", "open sesame", true).
func parseBasicAuth(auth string) (username, password string, ok bool) {
	const prefix = "Basic "
	// Case insensitive prefix match. See Issue 22736.
	if len(auth) < len(prefix) || !strings.EqualFold(auth[:len(prefix)], prefix) {
		return
	}
	c, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return
	}
	cs := string(c)
	s := strings.IndexByte(cs, ':')
	if s < 0 {
		return
	}
	return cs[:s], cs[s+1:], true
}
