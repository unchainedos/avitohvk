package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

type RouteRegistrator interface {
	RegisterRoutes(r chi.Router)
}

type RegistratorFunc func(r chi.Router)

func (f RegistratorFunc) RegisterRoutes(r chi.Router) {
	f(r)
}

func New(opts ...Option) http.Handler {
	cfg := &GroupConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	r := chi.NewRouter()

	r.Route("/api/v1", func(api chi.Router) {
		for _, g := range cfg.groups {
			api.Group(func(sub chi.Router) {
				for _, mw := range g.middlewares {
					sub.Use(mw)
				}
				for _, reg := range g.registrators {
					reg.RegisterRoutes(sub)
				}
			})
		}
	})

	return r
}
