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
	for _, tc := range cases {
		server := New("foo")
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
			resp:        "{\"service\":\"svc\",\"version\":\"\",\"method\":\"method\"}\n",
		},
		{
			name:        "success (with version)",
			status:      http.StatusOK,
			contentType: "application/json",
			path:        "/v1/svc/method?version=v1",
			method:      http.MethodPost,
			resp:        "{\"service\":\"svc\",\"version\":\"v1\",\"method\":\"method\"}\n",
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
	server := New("foo")
	for _, tc := range cases {
		t.Run(tc.name, func(*testing.T) {
			rr := httptest.NewRecorder()
			handlerF := server.RPCCallHandler()
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
