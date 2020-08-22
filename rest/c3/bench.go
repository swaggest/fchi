package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi"
)

func main() {
	log.Println("Starting server at http://localhost:8000, http://localhost:8000/something serves dummy data")
	log.Fatal(http.ListenAndServe(":8000", newRouter()))
}

func newRouter() http.Handler {
	router := chi.NewRouter()

	router.Method(http.MethodGet, "/something", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type Entity struct {
			Name string    `json:"name"`
			Time time.Time `json:"time"`
		}

		j, err := json.Marshal(Entity{
			Name: "Foo",
			Time: time.Now(),
		})

		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		w.Header().Set("Content-Type", "application/json; charset: utf-8")
		_, _ = w.Write(j)
	}))

	return router
}
