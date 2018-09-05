package http

import (
	"context"
	"go.uber.org/zap"
	"net"
	"net/http"

	"github.com/mercari/grpc-http-proxy/metadata"
	"net/url"
)

type Server struct {
	router      *http.ServeMux
	accessToken string
	client      Client
	discoverer  Discoverer
	logger      *zap.Logger
}

func New(token string,
	discoverer Discoverer,
	logger *zap.Logger,
) *Server {
	s := &Server{
		router:      http.NewServeMux(),
		accessToken: token,
		discoverer:  discoverer,
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

// Client is a dynamic gRPC client that performs reflection
type Client interface {
	Connect(context.Context, *url.URL) error
	CloseConn() error
	Call(context.Context,
		string,
		string,
		[]byte,
		*metadata.Metadata,
	) ([]byte, error)
}

// Discoverer performs service discover
type Discoverer interface {
	Resolve(svc, version string) (*url.URL, error)
}
