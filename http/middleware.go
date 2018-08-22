package http

import (
	"net/http"

	"go.uber.org/zap"
)

// Adapter represents a middleware adapter
type Adapter func(handlerFunc http.HandlerFunc) http.HandlerFunc

func apply(handler http.HandlerFunc, adapters ...Adapter) http.HandlerFunc {
	for _, m := range adapters {
		handler = m(handler)
	}
	return handler
}

func (s *Server) withAccessToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		providedToken := r.Header.Get("X-Access-Token")
		if providedToken == "" {
			w.WriteHeader(http.StatusUnauthorized)
			s.logger.Info("unauthorized",
				zap.String("reason", "no token"),
			)
			return
		}
		if providedToken != s.accessToken {
			w.WriteHeader(http.StatusUnauthorized)
			s.logger.Info("unauthorized",
				zap.String("reason", "invalid token"),
			)
			return
		}
		next(w, r)
	}
}

func (s *Server) withLog(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		d := newDelegator(w)
		next(d, r)
		s.logger.Info("request",
			zap.String("host", r.URL.Host),
			zap.String("path", r.URL.Path),
			zap.Int("status", d.status),
			zap.String("method", r.Method),
		)
	}
}

type responseWriterDelegator struct {
	status int
	http.ResponseWriter
}

func newDelegator(w http.ResponseWriter) *responseWriterDelegator {
	return &responseWriterDelegator{
		ResponseWriter: w,
	}
}

// WriteHeader saves the status code while writing it to the response header
func (w *responseWriterDelegator) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}
