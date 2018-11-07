package http

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/mercari/grpc-http-proxy/log"
	"github.com/mercari/grpc-http-proxy/metadata"
)

type fakeDiscoverer struct {
	t *testing.T
}

func newFakeDiscoverer(t *testing.T) *fakeDiscoverer {
	return &fakeDiscoverer{
		t: t,
	}
}

func (d *fakeDiscoverer) Resolve(service, version string) (*url.URL, error) {
	var rawurl string
	if version == "" {
		rawurl = service + ":5000"
	} else {
		rawurl = fmt.Sprintf("%s.%s:5000", version, service)
	}
	u, _ := url.Parse(rawurl)
	return u, nil
}

func (d *fakeDiscoverer) All() []byte {
	return []byte{}
}

type fakeClient struct {
	t       *testing.T
	service string
	version string
	err     error
}

func newFakeClient(t *testing.T) *fakeClient {
	return &fakeClient{
		t: t,
	}
}

func (c *fakeClient) Connect(ctx context.Context, target *url.URL) error {
	parts := strings.Split(target.String(), ".")
	if len(parts) == 2 {
		c.version = parts[0]
		c.service = strings.TrimSuffix(parts[1], ":5000")
	} else {
		c.version = ""
		c.service = strings.TrimSuffix(target.String(), ":5000")
	}
	return nil
}

func (c *fakeClient) CloseConn() error {
	return nil
}

func (c *fakeClient) Call(ctx context.Context,
	serviceName, methodName string,
	message []byte,
	md *metadata.Metadata,
) ([]byte, error) {
	response := fmt.Sprintf("{\"serviceVersion\":\"%s\",\"service\":\"%s\",\"method\":\"%s\"}\n",
		c.version,
		c.service,
		methodName)
	return []byte(response), nil
}

func TestServer_LivenessProbeHandler(t *testing.T) {
	cases := []struct {
		name   string
		method string
		status int
	}{
		{
			name:   "get method",
			method: http.MethodGet,
			status: http.StatusOK,
		},
		{
			name:   "post method",
			method: http.MethodPost,
			status: http.StatusMethodNotAllowed,
		},
	}
	d := newFakeDiscoverer(t)
	server := New("foo", d, log.NewDiscard())
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			handlerF := server.LivenessProbeHandler()
			handlerF(rr, httptest.NewRequest(tc.method, "/healthz", nil))

			if got, want := rr.Result().StatusCode, tc.status; got != want {
				t.Fatalf("got %d, want %d", got, want)
			}
		})
	}
}

func TestServer_CatchAllHandler(t *testing.T) {
	cases := []struct {
		name   string
		status int
		path   string
	}{
		{
			name:   "not found",
			status: http.StatusNotFound,
			path:   "/notfound",
		},
	}
	d := newFakeDiscoverer(t)
	for _, tc := range cases {
		server := New("foo", d, log.NewDiscard())
		t.Run(tc.name, func(*testing.T) {
			rr := httptest.NewRecorder()
			handlerF := server.CatchAllHandler()
			handlerF(rr, httptest.NewRequest(http.MethodGet, tc.path, nil))

			if got, want := rr.Result().StatusCode, tc.status; got != want {
				t.Fatalf("got %d, want %d", got, want)
			}
		})
	}
}

func TestServer_RPCCallHandler(t *testing.T) {
	cases := []struct {
		name        string
		status      int
		contentType string
		path        string
		method      string
		resp        string
	}{
		{
			name:        "success (without version)",
			status:      http.StatusOK,
			contentType: "application/json",
			path:        "/v1/svc/method",
			method:      http.MethodPost,
			resp:        "{\"serviceVersion\":\"\",\"service\":\"svc\",\"method\":\"method\"}\n",
		},
		{
			name:        "success (with version)",
			status:      http.StatusOK,
			contentType: "application/json",
			path:        "/v1/svc/method?version=v1",
			method:      http.MethodPost,
			resp:        "{\"serviceVersion\":\"v1\",\"service\":\"svc\",\"method\":\"method\"}\n",
		},
		{
			name:        "multiple versions specified",
			status:      http.StatusBadRequest,
			contentType: "",
			path:        "/v1/svc/method?version=v1&version=v2",
			method:      http.MethodPost,
			resp:        "",
		},
		{
			name:        "invalid path",
			status:      http.StatusNotFound,
			contentType: "",
			path:        "/v1/svc/method/toolong",
			method:      http.MethodPost,
			resp:        "",
		},
		{
			name:        "method not allowed",
			status:      http.StatusMethodNotAllowed,
			contentType: "",
			path:        "/v1/svc/version",
			method:      http.MethodGet,
			resp:        "",
		},
	}
	d := newFakeDiscoverer(t)
	server := New("foo", d, log.NewDiscard())
	newClient := func() Client {
		return newFakeClient(t)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(*testing.T) {
			rr := httptest.NewRecorder()
			handlerF := server.RPCCallHandler(newClient)
			handlerF(rr, httptest.NewRequest(tc.method, tc.path, nil))

			if got, want := rr.Result().StatusCode, tc.status; got != want {
				t.Fatalf("got %d, want %d", got, want)
			}

			var contentType string
			if len(rr.Result().Header["Content-Type"]) == 1 {
				contentType = rr.Result().Header["Content-Type"][0]
			}
			if got, want := contentType, tc.contentType; got != want {
				t.Fatalf("got %s, want %s", got, want)
			}
			if got, want := rr.Body.String(), tc.resp; got != want {
				t.Fatalf("got %s, want %s", got, want)
			}
		})
	}
}
