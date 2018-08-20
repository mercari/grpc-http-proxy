package http

func (s *Server) routes() {
	s.router.HandleFunc("/healthz", s.LivenessProbeHandler()).
		Methods("GET")
	s.router.HandleFunc("/{service}/{method}", s.withAccessToken(s.RPCCallHandler())).
		Methods("POST")
	s.router.HandleFunc("/{service}/{method}", s.withAccessToken(s.MethodNowAllowedHandler()))
	s.router.HandleFunc("/{service}/{version}/{method}", s.withAccessToken(s.VersionedRPCCallHandler())).
		Methods("POST")
	s.router.HandleFunc("/{service}/{version}/{method}", s.withAccessToken(s.MethodNowAllowedHandler()))
	s.router.PathPrefix("/").HandlerFunc(s.NotFoundHandler())
}
