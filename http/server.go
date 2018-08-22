package http

import (
	"net"
	"net/http"
)

type Server struct {
	router      *http.ServeMux
	accessToken string
}

func New(token string) *Server {
	s := &Server{
		router:      http.NewServeMux(),
		accessToken: token,
	}
	s.routes()

	return s
}

func (s *Server) Serve(ln net.Listener) error {
	srv := &http.Server{
		Handler: s.router,
	}
	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
