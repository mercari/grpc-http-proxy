package http

import (
	"github.com/mercari/grpc-http-proxy/proxy"
)

func (s *Server) registerHandlers() {
	newClient := func() Client {
		return proxy.NewProxy()
	}

	s.router.HandleFunc("/healthz", s.LivenessProbeHandler())
	s.router.HandleFunc("/debug", s.withLog(s.DebugHandler()))
	s.router.HandleFunc("/v1/", apply(s.RPCCallHandler(newClient), []Adapter{
		s.withAccessToken,
		s.withLog,
	}...))
	s.router.HandleFunc("/services", s.withLog(s.ListServicesHandler()))
	// s.router.HandleFunc("/v1/list/", func(arg1 http.ResponseWriter, arg2 *http.Request) {
	//
	// })
	s.router.HandleFunc("/", apply(s.CatchAllHandler(), []Adapter{
		s.withAccessToken,
		s.withLog,
	}...))
}
