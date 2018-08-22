package http

import (
	"encoding/json"
	"net/http"
	"strings"
)

type placeholderResponse struct {
	Service string `json:"service"`
	Version string `json:"version"`
	Method  string `json:"method"`
}

func (s *Server) LivenessProbeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
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
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) != 4 {
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
			resp.Version = v[0]
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
		return
	}
}
