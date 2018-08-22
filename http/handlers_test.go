package http

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

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
	server := New("foo")
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
			path:        "/svc/method",
			method:      http.MethodPost,
			resp:        "{\"service\":\"svc\",\"version\":\"\",\"method\":\"method\"}\n",
		},
		{
			name:        "success (with version)",
			status:      http.StatusOK,
			contentType: "application/json",
			path:        "/svc/version/method",
			method:      http.MethodPost,
			resp:        "{\"service\":\"svc\",\"version\":\"version\",\"method\":\"method\"}\n",
		},
		{
			name:        "invalid path",
			status:      http.StatusNotFound,
			contentType: "application/json",
			path:        "/notfound",
			method:      http.MethodPost,
			resp:        "",
		},
		{
			name:        "method not allowed",
			status:      http.StatusMethodNotAllowed,
			contentType: "application/json",
			path:        "/svc/version",
			method:      http.MethodGet,
			resp:        "",
		},
	}
	server := New("foo")
	for _, tc := range cases {
		t.Run(tc.name, func(*testing.T) {
			rr := httptest.NewRecorder()
			handlerF := server.RPCCallHandler()
			handlerF(rr, httptest.NewRequest(tc.method, tc.path, nil))

			if got, want := rr.Result().StatusCode, tc.status; got != want {
				t.Fatalf("got %d, want %d", got, want)
			}
			contentType := rr.Result().Header["Content-Type"][0]
			if got, want := contentType, tc.contentType; got != want {
				t.Fatalf("got %s, want %s", got, want)
			}
			if got, want := rr.Body.String(), tc.resp; got != want {
				t.Fatalf("got %s, want %s", got, want)
			}
		})
	}
}
