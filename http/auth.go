package http

import (
	"net/http"

	"go.uber.org/zap"
)

func (s *Server) withAccessToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		providedToken := r.Header.Get("X-Access-Token")
		if providedToken != s.accessToken {
			w.WriteHeader(http.StatusUnauthorized)
			defer s.logger.Sync()
			s.logger.Info("Unauthorized",
				zap.Int("status", http.StatusUnauthorized),
				zap.String("path", r.URL.Path))
			return
		}
		next(w, r)
	}
}
