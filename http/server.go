package http

import (
	"net"
	"net/http"

	"github.com/gorilla/mux"
)

type Server struct {
	router      *mux.Router
	accessToken string
}

func New(token string) *Server {
	s := &Server{
		router:      mux.NewRouter(),
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
