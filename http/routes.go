package http

import (
	"github.com/mercari/grpc-http-proxy/proxy"
)

func (s *Server) registerHandlers() {
	newClient := func() Client {
		return proxy.NewProxy()
	}

	s.router.HandleFunc("/healthz", s.withLog(s.LivenessProbeHandler()))
	s.router.HandleFunc("/v1/", apply(s.RPCCallHandler(newClient), []Adapter{
		s.withAccessToken,
		s.withLog,
	}...))
	s.router.HandleFunc("/", apply(s.CatchAllHandler(), []Adapter{
		s.withAccessToken,
		s.withLog,
	}...))
}
