package fasthttp

import (
	"github.com/swaggest/rest"
	"github.com/valyala/fasthttp"
)

// RequestDecoder maps data from fasthttp.RequestCtx into structured Go input value.
type RequestDecoder interface {
	Decode(rc *fasthttp.RequestCtx, input interface{}, validator rest.Validator) error
}

// RequestDecoderFactory creates request decoder for particular structured Go input value.
type RequestDecoderFactory interface {
	MakeDecoder(method string, input interface{}) RequestDecoder
}
