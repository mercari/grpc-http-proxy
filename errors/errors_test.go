package errors

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"testing"

	"google.golang.org/grpc/codes"
)

func TestError_Error(t *testing.T) {
	cases := []struct {
		Code
		msg string
	}{
		{
			Code: UpstreamConnFailure,
			msg:  "could not connect to backend gRPC service",
		},
		{
			Code: ServiceUnresolvable,
			msg:  "could not resolve service",
		},
		{
			Code: ServiceNotFound,
			msg:  "service not found; service discovery error",
		},
		{
			Code: MethodNotFound,
			msg:  "no such gRPC method",
		},
		{
			Code: MessageTypeMismatch,
			msg:  "message type mismatch",
		},
		{
			Code: VersionNotSpecified,
			msg:  "multiple versions of this service exist. specify version in request",
		},
		{
			Code: VersionUndecidable,
			msg:  "multiple backends exist. add version annotations",
		},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%d", tc.Code), func(t *testing.T) {
			err := &ProxyError{
				Code: tc.Code,
			}
			if got, want := err.Error(), tc.msg; got != want {
				t.Fatalf("got %s, want %s", got, want)
			}
		})
	}
}

func TestGRPCError_Error(t *testing.T) {
	const msg = "error"
	err := &GRPCError{
		Message: msg,
	}
	if got, want := err.Error(), msg; got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

func TestGRPCError_HTTPStatusCode(t *testing.T) {
	cases := []struct {
		grpcCode codes.Code
		httpCode int
	}{
		{
			codes.OK,
			http.StatusOK,
		},
		{
			codes.Canceled,
			http.StatusRequestTimeout,
		},
		{
			codes.Unknown,
			http.StatusInternalServerError,
		},
		{
			codes.InvalidArgument,
			http.StatusBadRequest,
		},
		{
			codes.DeadlineExceeded,
			http.StatusRequestTimeout,
		},
		{
			codes.NotFound,
			http.StatusNotFound,
		},
		{
			codes.AlreadyExists,
			http.StatusConflict,
		},
		{
			codes.PermissionDenied,
			http.StatusForbidden,
		},
		{
			codes.Unauthenticated,
			http.StatusUnauthorized,
		},
		{
			codes.ResourceExhausted,
			http.StatusServiceUnavailable,
		},
		{
			codes.FailedPrecondition,
			http.StatusPreconditionFailed,
		},
		{
			codes.Aborted,
			http.StatusConflict,
		},
		{
			codes.OutOfRange,
			http.StatusBadRequest,
		},
		{
			codes.Unimplemented,
			http.StatusNotImplemented,
		},
		{
			codes.Internal,
			http.StatusInternalServerError,
		},
		{
			codes.Unavailable,
			http.StatusServiceUnavailable,
		},
		{
			codes.DataLoss,
			http.StatusInternalServerError,
		},
	}
	for _, tc := range cases {
		t.Run(fmt.Sprintf("%d", tc.grpcCode), func(t *testing.T) {
			err := &GRPCError{
				StatusCode: int(tc.grpcCode),
			}
			if got, want := err.HTTPStatusCode(), tc.httpCode; got != want {
				t.Fatalf("got %d, want %d", got, want)
			}
		})
	}
}

func TestInternalError_WriteJSON(t *testing.T) {
	err := &ProxyError{
		Code:    Unknown,
		Message: "test error",
	}
	expected := []byte(fmt.Sprintf("{\"status\":\"%d\",\"message\":\"%s\"}",
		err.HTTPStatusCode(),
		err.Message,
	))

	b := bytes.NewBuffer(make([]byte, 0, 128))
	w := bufio.NewWriter(b)
	if got, want := err.WriteJSON(w), error(nil); got != want {
		t.Fatalf("err was not nil")
	}
	if got, want := b.Bytes(), expected; bytes.Equal(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
