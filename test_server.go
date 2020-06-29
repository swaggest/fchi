package fchi

import (
	"fmt"
	"net"
	"time"

	"github.com/valyala/fasthttp"
)

// TestServer is a test server.
type TestServer struct {
	URL  string
	fsrv fasthttp.Server
}

// NewTestServer spawns fasthttp server on an available localhost port.
func NewTestServer(r Handler) *TestServer {
	ts := TestServer{}

	ts.fsrv.Handler = RequestHandler(r)
	ts.fsrv.IdleTimeout = 10 * time.Millisecond

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if l, err = net.Listen("tcp6", "[::1]:0"); err != nil {
			panic(fmt.Sprintf("httptest: failed to listen on a port: %v", err))
		}
	}
	ts.URL = "http://" + l.Addr().String()

	go func() {
		ts.fsrv.Serve(l)
	}()

	return &ts
}

// Close stops test server.
func (ts *TestServer) Close() {
	err := ts.fsrv.Shutdown()
	if err != nil {
		panic(err)
	}
}
