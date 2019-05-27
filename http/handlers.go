package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	grpc_metadata "google.golang.org/grpc/metadata"

	perrors "github.com/mercari/grpc-http-proxy/errors"
	"github.com/mercari/grpc-http-proxy/metadata"
	"github.com/mercari/grpc-http-proxy/proxy"
)

type callee struct {
	ServiceVersion string `json:"serviceVersion"`
	Service        string `json:"service"`
	Method         string `json:"method"`
}

// LivenessProbeHandler returns a status code 200 response for liveness probes
func (s *Server) LivenessProbeHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// DebugHandler is
func (s *Server) DebugHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		j := s.discoverer.All()
		fmt.Println(string(j))
		w.Write(j)
	}
}

// ListServicesHandler is
func (s *Server) ListServicesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		urls, ok := r.URL.Query()["url"]
		if !ok || len(urls[0]) < 1 {
			fmt.Println("url params should have url key")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		u, err := url.Parse(urls[0])
		if err != nil {
			fmt.Println("url not good")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		p := proxy.NewProxy()
		p.Connect(context.Background(), u)
		defer p.CloseConn()

		svc, err := p.ListServices()
		if err != nil {
			fmt.Printf("could not list services %s", err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		json.NewEncoder(w).Encode(svc)
	}
}

// CatchAllHandler handles requests for non-existing paths
// This is done explicitly in order to have the logger middleware log the fact
func (s *Server) CatchAllHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}
}

// RPCCallHandler handles requests for making gRPC calls
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
		// TODO: Re-Use connections instead of creating a new connection for each request.
		client := newClient()
		client.Connect(ctx, u)
		defer client.CloseConn()

		md := make(metadata.Metadata)

		inputMessage, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()
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
