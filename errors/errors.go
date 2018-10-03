package errors

import (
	"encoding/json"
	"io"
	"net/http"

	any "github.com/golang/protobuf/ptypes/any"
	"google.golang.org/grpc/codes"
)

type Error interface {
	error
	HTTPStatusCode() int
	WriteJSON(w io.Writer) error
}

// ProxyError represents internal errors
type ProxyError struct {
	Code
	Message string
	Err     error
}

// Code represents type of internal error
type Code int

const (
	// UpstreamConnFailure represents failure to connect to the upstream gRPC service
	UpstreamConnFailure Code = 1
	// ServiceUnresolvable represents failure to resolve a gRPC service to its upstream FQDN
	ServiceUnresolvable Code = 2
	// ServiceNotFound represents a missing gRPC service in an upstream, even though the service resolved to that upstream
	ServiceNotFound Code = 3
	// MethodNotFound represents a missing gRPC method in an upstream
	MethodNotFound Code = 4
	// MessageTypeMismatch represents user provided JSON not matching the message's type
	MessageTypeMismatch Code = 5
	// Unknown represents an unknown internal error
	Unknown Code = 6
	// VersionNotSpecified represents the user not specifying the upstream version when it is required.
	VersionNotSpecified Code = 7
	// VersionUndecidable represents there being multiple upstreams that match the specified (service, version) pair
	VersionUndecidable Code = 8
)

// Error satisfies the error interface
func (e *ProxyError) Error() string {
	switch e.Code {
	case UpstreamConnFailure:
		return "could not connect to backend gRPC service"
	case ServiceUnresolvable:
		return "could not resolve service"
	case ServiceNotFound:
		return "service not found; service discovery error"
	case MethodNotFound:
		return "no such gRPC method"
	case MessageTypeMismatch:
		return "message type mismatch"
	case VersionNotSpecified:
		return "multiple versions of this service exist. specify version in request"
	case VersionUndecidable:
		return "multiple backends exist. add version annotations"
	default:
		return "unknown failure"
	}
}

// HTTPStatusCode returns the HTTP status code for a internal error
func (e *ProxyError) HTTPStatusCode() int {
	switch e.Code {
	case UpstreamConnFailure:
		return http.StatusBadGateway
	case ServiceUnresolvable:
		return http.StatusNotFound
	case ServiceNotFound:
		return http.StatusInternalServerError
	case MethodNotFound:
		return http.StatusNotFound
	case MessageTypeMismatch:
		return http.StatusBadRequest
	case VersionNotSpecified:
		return http.StatusBadRequest
	case VersionUndecidable:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// WriteJSON writes an JSON representation of the internal error for responses
func (e *ProxyError) WriteJSON(w io.Writer) error {
	type JSONSchema struct {
		Status  int    `json:"status"`
		Message string `json:"message"`
	}
	return json.NewEncoder(w).Encode(&JSONSchema{
		Status:  e.HTTPStatusCode(),
		Message: e.Message,
	})
}

// GRPCError is an error returned by gRPC upstream
type GRPCError struct {
	StatusCode int        `json:"code"`
	Message    string     `json:"message"`
	Details    []*any.Any `json:"details,omitempty"`
}

// HTTPStatusCode converts gRPC status codes to HTTP status codes
// https://github.com/grpc-ecosystem/grpc-gateway/blob/7951e5b80744558ae3363fd792806e1db15e91a4/runtime/errors.go
func (e *GRPCError) HTTPStatusCode() int {
	c := codes.Code(e.StatusCode)
	switch c {
	case codes.OK:
		return http.StatusOK
	case codes.Canceled:
		return http.StatusRequestTimeout
	case codes.Unknown:
		return http.StatusInternalServerError
	case codes.InvalidArgument:
		return http.StatusBadRequest
	case codes.DeadlineExceeded:
		return http.StatusRequestTimeout
	case codes.NotFound:
		return http.StatusNotFound
	case codes.AlreadyExists:
		return http.StatusConflict
	case codes.PermissionDenied:
		return http.StatusForbidden
	case codes.Unauthenticated:
		return http.StatusUnauthorized
	case codes.ResourceExhausted:
		return http.StatusServiceUnavailable
	case codes.FailedPrecondition:
		return http.StatusPreconditionFailed
	case codes.Aborted:
		return http.StatusConflict
	case codes.OutOfRange:
		return http.StatusBadRequest
	case codes.Unimplemented:
		return http.StatusNotImplemented
	case codes.Internal:
		return http.StatusInternalServerError
	case codes.Unavailable:
		return http.StatusServiceUnavailable
	case codes.DataLoss:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}

// Error satisfies the error interface
func (e *GRPCError) Error() string {
	return e.Message
}

// WriteJSON writes an JSON representation of the gRPC error for responses
func (e *GRPCError) WriteJSON(w io.Writer) error {
	return json.NewEncoder(w).Encode(e)
}
