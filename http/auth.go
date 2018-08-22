package http

import (
	"net/http"
)

func (s *Server) withAccessToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		providedToken := r.Header.Get("X-Access-Token")
		if providedToken != s.accessToken {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}
