package http

func (s *Server) routes() {
	s.router.HandleFunc("/healthz", s.LivenessProbeHandler())
	s.router.HandleFunc("/", s.withAccessToken(s.RPCCallHandler()))
}
