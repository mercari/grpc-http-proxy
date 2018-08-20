package http

import (
	"go.uber.org/zap"
	"net"
	"net/http"
)

type Server struct {
	router      *http.ServeMux
	accessToken string
	logger      *zap.Logger
}

func New(token string, logger *zap.Logger) *Server {
	s := &Server{
		router:      http.NewServeMux(),
		accessToken: token,
		logger:      logger,
	}
	s.registerHandlers()

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
