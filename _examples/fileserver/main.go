//
// FileServer
// ===========
// This example demonstrates how to serve static files from your filesystem.
//
//
// Boot the server:
// ----------------
// $ go run main.go
//
// Client requests:
// ----------------
// $ curl http://localhost:3333/files/
// <pre>
// <a href="notes.txt">notes.txt</a>
// </pre>
//
// $ curl http://localhost:3333/files/notes.txt
// Notessszzz
//
package main

import (
	"context"
	"github.com/valyala/fasthttp"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/swaggest/fchi"
)

func main() {
	r := fchi.NewRouter()

	// Index handler
	r.Get("/", fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rc.Write([]byte("hi"))
	}))

	// Create a route along /files that will serve contents from
	// the ./data/ folder.
	workDir, _ := os.Getwd()
	filesDir := http.Dir(filepath.Join(workDir, "data"))
	FileServer(r, "/files", filesDir)

	fasthttp.ListenAndServe(":3333", fchi.RequestHandler(r))
}

// FileServer conveniently sets up a http.FileServer handler to serve
// static files from a http.FileSystem.
func FileServer(r fchi.Router, path string, root http.FileSystem) {
	if strings.ContainsAny(path, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}

	if path != "/" && path[len(path)-1] != '/' {
		r.Get(path, fchi.Adapt(http.RedirectHandler(path+"/", 301)))
		path += "/"
	}
	path += "*"

	r.Get(path, fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		rctx := fchi.RouteContext(rc)
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(root))
		fchi.Adapt(fs).ServeHTTP(ctx, rc)
	}))
}
