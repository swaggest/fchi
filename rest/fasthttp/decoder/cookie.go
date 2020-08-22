package decoder

import (
	"github.com/valyala/fasthttp"
	"net/url"
)

func cookiesToURLValues(rc *fasthttp.RequestCtx) (url.Values, error) {
	var params url.Values

	rc.Request.Header.VisitAllCookie(func(key, value []byte) {
		if params == nil {
			params = make(url.Values, 1)
		}

		params[string(key)] = []string{string(value)}
	})

	return params, nil
}
