package decoder

import (
	"github.com/valyala/fasthttp"
	"net/url"
)

func queryToURLValues(rc *fasthttp.RequestCtx) (url.Values, error) {
	var params url.Values

	rc.Request.URI().QueryArgs().VisitAll(func(key, value []byte) {
		if params == nil {
			params = make(url.Values, 1)
		}

		params[string(key)] = []string{string(value)}
	})

	return params, nil
}
