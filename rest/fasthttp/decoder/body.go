package decoder

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/swaggest/rest"
	"github.com/valyala/fasthttp"
)

func decodeJSONBody(rc *fasthttp.RequestCtx, input interface{}, validator rest.Validator) error {
	if rc.Request.Header.ContentLength() == 0 {
		return errors.New("missing request body to decode json")
	}

	contentType := rc.Request.Header.ContentType()
	if len(contentType) > 0 {
		if len(contentType) < 16 || !bytes.Equal(contentType[0:16], []byte("application/json")) { // allow 'application/json;charset=UTF-8'
			return fmt.Errorf(`request with "application/json" content type expected %q received`, contentType)
		}
	}

	err := json.Unmarshal(rc.Request.Body(), &input)
	if err != nil {
		return fmt.Errorf("failed to decode json: %w", err)
	}

	rawBody := json.RawMessage(rc.Request.Body())

	if validator != nil {
		err = validator.ValidateRequestData(rest.ParamInBody, map[string]interface{}{`body`: rawBody})
		if err != nil {
			return err
		}
	}

	return nil
}
