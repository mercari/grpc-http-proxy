package http

import (
	"io/ioutil"
	"net/http"
	"strings"

	grpc_metadata "google.golang.org/grpc/metadata"

	perrors "github.com/mercari/grpc-http-proxy/errors"
	"github.com/mercari/grpc-http-proxy/metadata"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type callee struct {
	ServiceVersion string `json:"serviceVersion"`
	Service        string `json:"service"`
	Method         string `json:"method"`
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

func (s *Server) RPCCallHandler(newClient func() Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// example path and query parameter:
		// example.com/v1/svc/method?version=v1
		parts := strings.Split(r.URL.Path, "/")
		if len(parts) != 4 {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		c := callee{
			Service: parts[2],
			Method:  parts[3],
		}
		if v, ok := r.URL.Query()["version"]; ok {
			if len(v) != 1 {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			c.ServiceVersion = v[0]
		}
		ctx := grpc_metadata.NewOutgoingContext(r.Context(),
			grpc_metadata.MD(metadata.MetadataFromHeaders(r.Header)))
		u, err := s.discoverer.Resolve(c.Service, c.ServiceVersion)
		if err != nil {
			s.logger.Error("error in handling call",
				zap.String("err", err.Error()))
			returnError(w, errors.Cause(err).(perrors.Error))
			return
		}
		client := newClient()
		client.Connect(ctx, u)
		md := make(metadata.Metadata)

		inputMessage, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		response, err := client.Call(ctx, c.Service, c.Method, inputMessage, &md)
		if err != nil {
			returnError(w, errors.Cause(err).(perrors.Error))
			s.logger.Error("error in handling call",
				zap.String("err", err.Error()))
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(response)
		return
	}
}

func returnError(w http.ResponseWriter, err perrors.Error) {
	w.WriteHeader(err.HTTPStatusCode())
	err.WriteJSON(w)
	return
}
