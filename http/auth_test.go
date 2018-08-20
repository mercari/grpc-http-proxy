package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServer_withAccessToken(t *testing.T) {
	type response struct {
		Status int    `json:"status"`
		Msg    string `json:"msg"`
	}
	cases := []struct {
		status      int
		contentType string
		r           response
		token       string
	}{
		{
			status:      http.StatusUnauthorized,
			contentType: "application/json",
			r: response{
				Status: http.StatusUnauthorized,
				Msg:    "Unauthorized",
			},
			token: "foo",
		},
	}

	for _, tc := range cases {
		server := New(tc.token)
		rr := httptest.NewRecorder()
		handlerF := server.withAccessToken(func(w http.ResponseWriter, r *http.Request) {
			panic("this shouldn't be called")
		})
		handlerF(rr, httptest.NewRequest("GET", "/", nil))

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
