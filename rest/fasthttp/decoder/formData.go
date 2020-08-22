package decoder

import (
	"github.com/valyala/fasthttp"
	"net/url"
)

func formDataToURLValues(rc *fasthttp.RequestCtx) (url.Values, error) {
	args := rc.Request.PostArgs()

	if args.Len() == 0 {
		return nil, nil
	}

	var params url.Values
	args.VisitAll(func(key, value []byte) {
		if params == nil {
			params = make(url.Values, 1)
		}

		params[string(key)] = []string{string(value)}
	})

	return params, nil
}
