package http

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mercari/grpc-http-proxy/log"
)

func TestServer_withAccessToken(t *testing.T) {
	cases := []struct {
		name        string
		status      int
		contentType string
		token       string
	}{
		{
			name:        "unauthorized",
			status:      http.StatusUnauthorized,
			contentType: "",
			token:       "foo",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(*testing.T) {
			server := New(tc.token, log.NewDiscard())
			rr := httptest.NewRecorder()
			handlerF := server.withAccessToken(func(w http.ResponseWriter, r *http.Request) {
				panic("this shouldn't be called")
			})
			handlerF(rr, httptest.NewRequest("GET", "/", nil))

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
		})
	}
}

func TestServer_withLog(t *testing.T) {
	server := New("foo", log.NewDiscard())
	rr := httptest.NewRecorder()
	handlerF := server.withLog(func(w http.ResponseWriter, r *http.Request) {})
	handlerF(rr, httptest.NewRequest("GET", "/", nil))
}
