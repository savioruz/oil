// Package router provides HTTP routing utilities.
package router

import (
	"oil/internal/handlers/gallery"
	"oil/internal/handlers/todo"
	"oil/internal/handlers/userprofile"

	"github.com/go-chi/chi/v5"
)

type DomainHandlers struct {
	Todo        todo.Handler
	Gallery     gallery.Handler
	Userprofile userprofile.Handler
}

type Router struct {
	DomainHandlers DomainHandlers
}

func (r *Router) SetupRoutes(router chi.Router) {
	router.Route("/api", func(routerGroup chi.Router) {
		r.DomainHandlers.Todo.Router(routerGroup)
		r.DomainHandlers.Gallery.Router(routerGroup)
		r.DomainHandlers.Userprofile.Router(routerGroup)
	})
}

func New(domainHandlers DomainHandlers) Router {
	return Router{
		DomainHandlers: domainHandlers,
	}
}
