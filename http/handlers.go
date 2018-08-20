package http

import (
	"encoding/json"
	"net/http"
)

func (s *Server) LivenessProbeHandler() http.HandlerFunc {
	type response struct {
		Status int    `json:"status"`
		Msg    string `json:"msg"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		itWorks := response{
			Status: http.StatusOK,
			Msg:    "IT WORKS!",
		}
		json.NewEncoder(w).Encode(itWorks)
	}
}

func (s *Server) MethodNowAllowedHandler() http.HandlerFunc {
	type response struct {
		Status int    `json:"status"`
		Msg    string `json:"msg"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		methodNotAllowed := response{
			Status: http.StatusMethodNotAllowed,
			Msg:    "Method not allowed",
		}
		json.NewEncoder(w).Encode(methodNotAllowed)
	}
}

func (s *Server) NotFoundHandler() http.HandlerFunc {
	type response struct {
		Status int    `json:"status"`
		Msg    string `json:"msg"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		notFound := response{
			Status: http.StatusNotFound,
			Msg:    "Not found",
		}
		json.NewEncoder(w).Encode(notFound)
	}
}

func (s *Server) RPCCallHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{\"msg\":\"unimplemented\"}"))
	}
}

func (s *Server) VersionedRPCCallHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{\"msg\":\"unimplemented\"}"))
	}
}
