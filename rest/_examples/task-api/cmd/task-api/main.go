package main

import (
	"github.com/swaggest/fchi"
	"github.com/swaggest/rest/_examples/task-api/internal/infrastructure"
	http3 "github.com/swaggest/rest/_examples/task-api/internal/infrastructure/fasthttp"
	http2 "github.com/swaggest/rest/_examples/task-api/internal/infrastructure/http"
	"github.com/valyala/fasthttp"
	"log"
	"net/http"
)

func main() {
	// Initialize application resources.
	l, err := infrastructure.NewServiceLocator()
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		fr := http3.NewRouter(l)
		log.Println("starting fast HTTP server at http://localhost:8011/docs/")
		//srv := fasthttp.Server{
		//	Handler: fr.ServeHTTP,
		//}
		//psrv := prefork.New(&srv)
		//
		//err = psrv.ListenAndServe(":8011")
		err := fasthttp.ListenAndServe(":8011", fchi.RequestHandler(fr))
		if err != nil {
			log.Fatal(err)
		}
	}()

	// Initialize HTTP router.
	r := http2.NewRouter(l)

	// Start HTTP server.
	log.Println("starting HTTP server at http://localhost:8010/docs/")
	err = http.ListenAndServe(":8010", r)
	if err != nil {
		log.Fatal(err)
	}
}
