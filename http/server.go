package http

import (
	"context"
	"net"
	"net/http"
	"net/url"

	"go.uber.org/zap"

	"github.com/mercari/grpc-http-proxy/metadata"
)

// Server is an grpc-http-proxy server
type Server struct {
	router      *http.ServeMux
	accessToken string
	client      Client
	discoverer  Discoverer
	logger      *zap.Logger
}

// New creates a new Server
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

// Serve starts the Server
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
