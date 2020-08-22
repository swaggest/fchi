package decoder

import (
	"github.com/swaggest/rest"
	"github.com/valyala/fasthttp"
	"net/url"

	"github.com/swaggest/form"
)

type (
	decoderFunc      func(rc *fasthttp.RequestCtx) (url.Values, error)
	valueDecoderFunc func(rc *fasthttp.RequestCtx, v interface{}, validator rest.Validator) error
)

func decodeValidate(d *form.Decoder, v interface{}, p url.Values, in rest.ParamIn, val rest.Validator) error {
	goValues := make(map[string]interface{}, len(p))

	err := d.Decode(v, p, goValues)
	if err != nil {
		return err
	}

	return val.ValidateRequestData(in, goValues)
}

func makeDecoder(in rest.ParamIn, formDecoder *form.Decoder, decoderFunc decoderFunc) valueDecoderFunc {
	return func(rc *fasthttp.RequestCtx, v interface{}, validator rest.Validator) error {
		values, err := decoderFunc(rc)
		if err != nil {
			return err
		}
		if validator != nil {
			return decodeValidate(formDecoder, v, values, in, validator)
		}
		return formDecoder.Decode(v, values)
	}
}
