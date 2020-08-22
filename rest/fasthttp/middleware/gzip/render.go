package gzip

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"github.com/valyala/fasthttp"
	"io"
)

// GzippedJSON exposes compressed JSON payload.
type GzippedJSON interface {
	GzipJSON() []byte
}

// ETagged exposes specific version of resource.
type ETagged interface {
	ETag() string
}

// WriteJSON writes compressed JSON.
//
// If response writer does not support direct gzip writing or rendered value does not expose compressed JSON
// regular JSON marshaling is used.
func WriteJSON(rc *fasthttp.RequestCtx, value interface{}) error {
	rc.SetContentType("application/json; charset=utf-8")

	if e, ok := value.(ETagged); ok {
		rc.Response.Header.Set("Etag", e.ETag())
	}

	if g, ok := value.(GzippedJSON); ok {
		return WriteCompressedBytes(rc, g.GzipJSON())
	}

	j := json.NewEncoder(rc.Response.BodyWriter())

	err := j.Encode(value)
	if err != nil {
		return err
	}

	return err
}

// WriteCompressedBytes writes compressed bytes to response.
//
// Bytes are unpacked if response writer does not support direct gzip writing.
func WriteCompressedBytes(rc *fasthttp.RequestCtx, gzipped []byte) error {
	ae := rc.Request.Header.Peek("Accept-Encoding")
	if len(ae) == 0 {
		return writeUncompressedBytes(rc, gzipped)
	}

	ae = bytes.ToLower(ae)

	n := bytes.Index(ae, []byte("gzip"))
	if n < 0 {
		return writeUncompressedBytes(rc, gzipped)
	}

	rc.Response.Header.Set("Content-Encoding", "gzip")
	_, err := rc.Write(gzipped)

	return err
}

func writeUncompressedBytes(rc *fasthttp.RequestCtx, gzipped []byte) error {
	gzr, err := gzip.NewReader(bytes.NewReader(gzipped))
	if err != nil {
		return err
	}

	// nolint:gosec // Decompression bomb is not relevant since we compress data in app itself.
	_, err = io.Copy(rc, gzr)

	return err
}
