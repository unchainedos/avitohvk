package server

import "time"

type Option func(srv *Server)

func SetReadTimeout(duration time.Duration) Option {
	return func(srv *Server) {
		srv.internalServer.ReadTimeout = duration
	}
}

func SetWriteTimeout(duration time.Duration) Option {
	return func(srv *Server) {
		srv.internalServer.WriteTimeout = duration
	}
}

func SetAddr(addr string) Option {
	return func(srv *Server) {
		if env := addr; env != "" {
			srv.internalServer.Addr = env
			return
		}

		srv.internalServer.Addr = defaultAddr
	}
}

func SetShutdownTimeout(duration time.Duration) Option {
	return func(srv *Server) {
		srv.shutdownTimeout = duration
	}
}
