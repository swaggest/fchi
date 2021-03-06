package main

import (
	"context"

	"github.com/swaggest/fchi"
	"github.com/valyala/fasthttp"
)

type usersResource struct{}

// Routes creates a REST router for the todos resource
func (rs usersResource) Routes() fchi.Router {
	r := fchi.NewRouter()
	// r.Use() // some middleware..

	r.Get("/", fchi.HandlerFunc(rs.List))    // GET /todos - read a list of todos
	r.Post("/", fchi.HandlerFunc(rs.Create)) // POST /todos - create a new todo and persist it
	r.Put("/", fchi.HandlerFunc(rs.Delete))

	r.Route("/{id}", func(r fchi.Router) {
		// r.Use(rs.TodoCtx) // lets have a todos map, and lets actually load/manipulate
		r.Get("/", fchi.HandlerFunc(rs.Get))       // GET /todos/{id} - read a single todo by :id
		r.Put("/", fchi.HandlerFunc(rs.Update))    // PUT /todos/{id} - update a single todo by :id
		r.Delete("/", fchi.HandlerFunc(rs.Delete)) // DELETE /todos/{id} - delete a single todo by :id
	})

	return r
}

func (rs usersResource) List(ctx context.Context, rc *fasthttp.RequestCtx) {
	rc.Write([]byte("aaa list of stuff.."))
}

func (rs usersResource) Create(ctx context.Context, rc *fasthttp.RequestCtx) {
	rc.Write([]byte("aaa create"))
}

func (rs usersResource) Get(ctx context.Context, rc *fasthttp.RequestCtx) {
	rc.Write([]byte("aaa get"))
}

func (rs usersResource) Update(ctx context.Context, rc *fasthttp.RequestCtx) {
	rc.Write([]byte("aaa update"))
}

func (rs usersResource) Delete(ctx context.Context, rc *fasthttp.RequestCtx) {
	rc.Write([]byte("aaa delete"))
}
