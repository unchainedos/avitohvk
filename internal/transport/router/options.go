package router

import "net/http"

type Middleware = func(http.Handler) http.Handler

type group struct {
	registrators []RouteRegistrator
	middlewares  []Middleware
}

type GroupConfig struct {
	groups []group
}

type Option func(*GroupConfig)

func WithGroup(registrators []RouteRegistrator, mw ...Middleware) Option {
	return func(c *GroupConfig) {
		c.groups = append(c.groups, group{registrators: registrators, middlewares: mw})
	}
}
