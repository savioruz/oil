package router

import (
	"oil/internal/handlers/auth"
	"oil/internal/handlers/gallery"
	"oil/internal/handlers/todo"

	"github.com/go-chi/chi/v5"
)

type DomainHandlers struct {
	Todo    todo.Handler
	Auth    auth.Handler
	Gallery gallery.Handler
}

type Router struct {
	DomainHandlers DomainHandlers
}

func (r *Router) SetupRoutes(router chi.Router) {
	router.Route("/api", func(routerGroup chi.Router) {
		r.DomainHandlers.Todo.Router(routerGroup)
		r.DomainHandlers.Auth.Router(routerGroup)
		r.DomainHandlers.Gallery.Router(routerGroup)
	})
}

func New(domainHandlers DomainHandlers) Router {
	return Router{
		DomainHandlers: domainHandlers,
	}
}
