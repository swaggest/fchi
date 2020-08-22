package decoder

import (
	"errors"
	"github.com/swaggest/form"
	"github.com/swaggest/rest"
	fasthttp2 "github.com/swaggest/rest/fasthttp"
	"github.com/valyala/fasthttp"
)

// decoder extracts Go value from *http.Request.
type decoder struct {
	decoders []valueDecoderFunc
	in       []rest.ParamIn
}

var _ fasthttp2.RequestDecoder = &decoder{}

// Decode populates and validates input with data from http request.
func (rm *decoder) Decode(rc *fasthttp.RequestCtx, input interface{}, validator rest.Validator) error {
	for i, decode := range rm.decoders {
		err := decode(rc, input, validator)
		if err != nil {
			if de, ok := err.(form.DecodeErrors); ok {
				errs := make(rest.RequestErrors, len(de))
				for name, e := range de {
					errs[string(rm.in[i])+":"+name] = []string{"#: " + e.Error()}
				}

				// TODO proper error processing.
				return errors.New("request decoding failed")
			}

			return err
		}
	}

	return nil
}
