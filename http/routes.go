package http

func (s *Server) registerHandlers() {

	s.router.HandleFunc("/healthz", s.withLog(s.LivenessProbeHandler()))
	s.router.HandleFunc("/v1/", apply(s.RPCCallHandler(), []Adapter{
		s.withAccessToken,
		s.withLog,
	}...))
	s.router.HandleFunc("/", apply(s.CatchAllHandler(), []Adapter{
		s.withAccessToken,
		s.withLog,
	}...))
}
