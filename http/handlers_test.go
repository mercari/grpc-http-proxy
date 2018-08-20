package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServer_NotFoundHandler(t *testing.T) {
	type response struct {
		Status int    `json:"status"`
		Msg    string `json:"msg"`
	}
	cases := []struct {
		status      int
		contentType string
		r           response
	}{
		{
			status:      http.StatusNotFound,
			contentType: "application/json",
			r: response{
				Status: http.StatusNotFound,
				Msg:    "Not found",
			},
		},
	}
	server := New("foo")
	for _, tc := range cases {
		rr := httptest.NewRecorder()
		handlerF := server.NotFoundHandler()
		handlerF(rr, httptest.NewRequest("GET", "/notfound", nil))

		if got, want := rr.Result().StatusCode, tc.status; got != want {
			t.Fatalf("got %d, want %d", got, want)
		}
		contentType := rr.Result().Header["Content-Type"][0]
		if got, want := contentType, tc.contentType; got != want {
			t.Fatalf("got %s, want %s", got, want)
		}
		r := response{}
		err := json.Unmarshal(rr.Body.Bytes(), &r)
		if err != nil {
			t.Fatal(err.Error())
		}
		if got, want := r.Status, tc.status; got != want {
			t.Fatalf("got %d, want %d", got, want)
		}
		if got, want := r.Msg, tc.r.Msg; got != want {
			t.Fatalf("got %s, want %s", got, want)
		}
	}
}

func TestServer_MethodNowAllowedHandler(t *testing.T) {
	type response struct {
		Status int    `json:"status"`
		Msg    string `json:"msg"`
	}
	cases := []struct {
		status      int
		contentType string
		r           response
	}{
		{
			status:      http.StatusMethodNotAllowed,
			contentType: "application/json",
			r: response{
				Status: http.StatusMethodNotAllowed,
				Msg:    "Method not allowed",
			},
		},
	}
	server := New("foo")
	for _, tc := range cases {
		rr := httptest.NewRecorder()
		handlerF := server.MethodNowAllowedHandler()
		handlerF(rr, httptest.NewRequest("GET", "/svc/method", nil))

		if got, want := rr.Result().StatusCode, tc.status; got != want {
			t.Fatalf("got %d, want %d", got, want)
		}
		contentType := rr.Result().Header["Content-Type"][0]
		if got, want := contentType, tc.contentType; got != want {
			t.Fatalf("got %s, want %s", got, want)
		}
		r := response{}
		err := json.Unmarshal(rr.Body.Bytes(), &r)
		if err != nil {
			t.Fatal(err.Error())
		}
		if got, want := r.Status, tc.status; got != want {
			t.Fatalf("got %d, want %d", got, want)
		}
		if got, want := r.Msg, tc.r.Msg; got != want {
			t.Fatalf("got %s, want %s", got, want)
		}
	}
}

func TestServer_LivenessProbeHandler(t *testing.T) {
	type response struct {
		Status int    `json:"status"`
		Msg    string `json:"msg"`
	}
	cases := []struct {
		status      int
		contentType string
		r           response
	}{
		{
			status:      http.StatusOK,
			contentType: "application/json",
			r: response{
				Status: http.StatusOK,
				Msg:    "IT WORKS!",
			},
		},
	}
	server := New("foo")
	for _, tc := range cases {
		rr := httptest.NewRecorder()
		handlerF := server.LivenessProbeHandler()
		handlerF(rr, httptest.NewRequest("GET", "/healthz", nil))

		if got, want := rr.Result().StatusCode, tc.status; got != want {
			t.Fatalf("got %d, want %d", got, want)
		}
		contentType := rr.Result().Header["Content-Type"][0]
		if got, want := contentType, tc.contentType; got != want {
			t.Fatalf("got %s, want %s", got, want)
		}
		r := response{}
		err := json.Unmarshal(rr.Body.Bytes(), &r)
		if err != nil {
			t.Fatal(err.Error())
		}
		if got, want := r.Status, tc.status; got != want {
			t.Fatalf("got %d, want %d", got, want)
		}
		if got, want := r.Msg, tc.r.Msg; got != want {
			t.Fatalf("got %s, want %s", got, want)
		}
	}
}

func TestServer_RPCCallHandler(t *testing.T) {
	cases := []struct {
		status      int
		contentType string
		msg         string
	}{
		{
			status:      http.StatusOK,
			contentType: "application/json",
			msg:         "{\"msg\":\"unimplemented\"}",
		},
	}
	server := New("foo")
	for _, tc := range cases {
		rr := httptest.NewRecorder()
		handlerF := server.RPCCallHandler()
		handlerF(rr, httptest.NewRequest("GET", "/healthz", nil))

		if got, want := rr.Result().StatusCode, tc.status; got != want {
			t.Fatalf("got %d, want %d", got, want)
		}
		contentType := rr.Result().Header["Content-Type"][0]
		if got, want := contentType, tc.contentType; got != want {
			t.Fatalf("got %s, want %s", got, want)
		}
		if got, want := string(rr.Body.Bytes()), tc.msg; got != want {
			t.Fatalf("got %s, want %s", got, want)
		}
	}
}

func TestServer_VersionedRPCCallHandler(t *testing.T) {
	cases := []struct {
		status      int
		contentType string
		msg         string
	}{
		{
			status:      http.StatusOK,
			contentType: "application/json",
			msg:         "{\"msg\":\"unimplemented\"}",
		},
	}
	server := New("foo")
	for _, tc := range cases {
		rr := httptest.NewRecorder()
		handlerF := server.VersionedRPCCallHandler()
		handlerF(rr, httptest.NewRequest("GET", "/healthz", nil))

		if got, want := rr.Result().StatusCode, tc.status; got != want {
			t.Fatalf("got %d, want %d", got, want)
		}
		contentType := rr.Result().Header["Content-Type"][0]
		if got, want := contentType, tc.contentType; got != want {
			t.Fatalf("got %s, want %s", got, want)
		}
		if got, want := string(rr.Body.Bytes()), tc.msg; got != want {
			t.Fatalf("got %s, want %s", got, want)
		}
	}
}
