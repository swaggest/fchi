// TODO convert middleware to fasthttp.
// +build ignore

package middleware

import (
	"bytes"
	"context"
	"github.com/swaggest/fchi"
	"github.com/valyala/fasthttp"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestContentType(t *testing.T) {
	t.Parallel()

	var tests = []struct {
		name                string
		inputValue          string
		allowedContentTypes []string
		want                int
	}{
		{
			"should accept requests with a matching content type",
			"application/json; charset=UTF-8",
			[]string{"application/json"},
			http.StatusOK,
		},
		{
			"should accept requests with a matching content type no charset",
			"application/json",
			[]string{"application/json"},
			http.StatusOK,
		},
		{
			"should accept requests with a matching content-type with extra values",
			"application/json; foo=bar; charset=UTF-8; spam=eggs",
			[]string{"application/json"},
			http.StatusOK,
		},
		{
			"should accept requests with a matching content type when multiple content types are supported",
			"text/xml; charset=UTF-8",
			[]string{"application/json", "text/xml"},
			http.StatusOK,
		},
		{
			"should not accept requests with a mismatching content type",
			"text/plain; charset=latin-1",
			[]string{"application/json"},
			http.StatusUnsupportedMediaType,
		},
		{
			"should not accept requests with a mismatching content type even if multiple content types are allowed",
			"text/plain; charset=Latin-1",
			[]string{"application/json", "text/xml"},
			http.StatusUnsupportedMediaType,
		},
	}

	for _, tt := range tests {
		var tt = tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			recorder := httptest.NewRecorder()

			r := fchi.NewRouter()
			r.Use(AllowContentType(tt.allowedContentTypes...))
			r.Post("/", fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {}))

			body := []byte("This is my content. There are many like this but this one is mine")
			req := httptest.NewRequest("POST", "/", bytes.NewReader(body))
			req.Header.Set("Content-Type", tt.inputValue)

			r.ServeHTTP(recorder, req)
			res := recorder.Result()

			if res.StatusCode != tt.want {
				t.Errorf("response is incorrect, got %d, want %d", recorder.Code, tt.want)
			}
		})
	}
}
