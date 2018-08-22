package http

func (s *Server) registerHandlers() {
	s.router.HandleFunc("/healthz", s.LivenessProbeHandler())
	s.router.HandleFunc("/v1/", s.withAccessToken(s.RPCCallHandler()))
	s.router.HandleFunc("/", s.withAccessToken(s.CatchAllHandler()))
}
