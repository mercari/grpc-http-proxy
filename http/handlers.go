package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

// This struct is for a dummy response. It will be removed in the future.
type placeholderResponse struct {
	ServiceVersion string `json:"serviceVersion"`
	Service        string `json:"service"`
	Method         string `json:"method"`
}

func (s *Server) LivenessProbeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			defer s.logger.Sync()
			s.logger.Info("Method not allowed",
				zap.Int("status", http.StatusMethodNotAllowed),
				zap.String("path", r.URL.Path))

			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		defer s.logger.Sync()
		s.logger.Info("IT WORKS!",
			zap.Int("status", http.StatusOK),
			zap.String("path", r.URL.Path))
		w.WriteHeader(http.StatusOK)
	}
}

func (s *Server) CatchAllHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}
}

func (s *Server) RPCCallHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			defer s.logger.Sync()
			s.logger.Info("Method not allowed",
				zap.Int("status", http.StatusMethodNotAllowed),
				zap.String("path", r.URL.Path))

			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// example path and query parameter:
		// example.com/v1/svc/method?version=v1
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) != 4 {
			s.logger.Info("Not found",
				zap.Int("status", http.StatusNotFound),
				zap.String("path", r.URL.Path))
			w.WriteHeader(http.StatusNotFound)
			return
		}
		resp := placeholderResponse{
			Service: parts[2],
			Method:  parts[3],
		}
		if v, ok := r.URL.Query()["version"]; ok {
			if len(v) != 1 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			resp.ServiceVersion = v[0]
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		defer s.logger.Sync()
		/* TODO(tomoyat1) emit logs based on results */
		return
	}
}
