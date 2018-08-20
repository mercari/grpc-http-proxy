package http

import (
	"encoding/json"
	"net/http"
)

func (s *Server) withAccessToken(next http.HandlerFunc) http.HandlerFunc {
	type response struct {
		Status int    `json:"status"`
		Msg    string `json:"msg"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		providedToken := r.Header.Get("X-Access-Token")
		if providedToken != s.accessToken {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			unauthorized := response{
				Status: http.StatusUnauthorized,
				Msg:    "Unauthorized",
			}
			json.NewEncoder(w).Encode(unauthorized)
			return
		}
		next(w, r)
	}
}
