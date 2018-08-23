package http

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestServer_withAccessTokenInvalidToken(t *testing.T) {
	cases := []struct {
		name          string
		status        int
		contentType   string
		token         string
		expectedToken string
		message       string
		fields        []zapcore.Field
	}{
		{
			name:          "no token",
			status:        http.StatusUnauthorized,
			contentType:   "",
			token:         "",
			expectedToken: "foo",
			message:       "unauthorized",
			fields: []zapcore.Field{
				{
					Key:    "reason",
					Type:   zapcore.StringType,
					String: "no token",
				},
			},
		},
		{
			name:          "wrong token",
			status:        http.StatusUnauthorized,
			contentType:   "",
			token:         "bar",
			expectedToken: "foo",
			message:       "unauthorized",
			fields: []zapcore.Field{
				{
					Key:    "reason",
					Type:   zapcore.StringType,
					String: "invalid token",
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(*testing.T) {
			logger, logs := observer.New(zapcore.InfoLevel)
			server := New(tc.expectedToken, zap.New(logger))
			rr := httptest.NewRecorder()
			handlerF := server.withAccessToken(func(w http.ResponseWriter, r *http.Request) {
				panic("this shouldn't be called")
			})
			req := httptest.NewRequest("GET", "/", nil)
			if tc.token != "" {
				req.Header.Set("X-Access-Token", tc.token)
			}
			handlerF(rr, req)

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

			if logs.Len() != 1 {
				t.Fatalf("incorrect number of log entries: got %d, want %d", logs.Len(), 1)
			}
			entry := logs.All()[0]
			if got, want := entry.Message, tc.message; got != want {
				t.Fatalf("got: %s, want: %s", got, want)
			}
			for i, f := range entry.Context {
				if got, want := tc.fields[i], f; !reflect.DeepEqual(got, want) {
					t.Fatalf("got %v, want %v", got, want)
				}
			}
		})
	}
}

func TestServer_withLog(t *testing.T) {
	fields := []zapcore.Field{
		{
			Key:    "host",
			Type:   zapcore.StringType,
			String: "",
		},
		{
			Key:    "path",
			Type:   zapcore.StringType,
			String: "/",
		},
		{
			Key:     "status",
			Type:    zapcore.Int64Type,
			Integer: http.StatusOK,
		},
		{
			Key:    "method",
			Type:   zapcore.StringType,
			String: http.MethodGet,
		},
	}
	logger, logs := observer.New(zapcore.InfoLevel)
	server := New("foo", zap.New(logger))
	rr := httptest.NewRecorder()
	handlerF := server.withLog(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handlerF(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	if logs.Len() != 1 {
		t.Fatalf("incorrect number of log entries: got %d, want %d", logs.Len(), 1)
	}
	entry := logs.All()[0]
	if got, want := entry.Message, "request"; got != want {
		t.Fatalf("got: %s, want: %s", got, want)
	}
	for i, f := range entry.Context {
		if got, want := fields[i], f; !reflect.DeepEqual(got, want) {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
