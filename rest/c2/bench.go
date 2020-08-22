package main

import (
	"context"
	"encoding/json"
	"github.com/swaggest/fchi"
	"github.com/valyala/fasthttp"
	"log"
	"net/http"
	"time"
)

func main() {
	log.Println("Starting server at http://localhost:8000, http://localhost:8000/something serves dummy data")
	log.Fatal(fasthttp.ListenAndServe(":8000", fchi.RequestHandler(newRouter())))
}

func newRouter() fchi.Handler {
	router := fchi.NewRouter()

	router.Method(http.MethodGet, "/something", fchi.HandlerFunc(func(ctx context.Context, rc *fasthttp.RequestCtx) {
		type Entity struct {
			Name string    `json:"name"`
			Time time.Time `json:"time"`
		}

		j, err := json.Marshal(Entity{
			Name: "Foo",
			Time: time.Now(),
		})

		if err != nil {
			rc.Error(err.Error(), http.StatusInternalServerError)
		}

		rc.SetContentType("application/json; charset: utf-8")
		_, _ = rc.Write(j)
		//_, _ = rc.Write([]byte(`{"name":"foo","time":"bar"}`))
	}))

	return router
}
