package proxytest

import (
	"net/url"
	"testing"

	"context"
	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	TestService     = "grpc.testing.TestService"
	NotFoundService = "not.found.NoService"
	EmptyCall       = "EmptyCall"
	UnaryCall       = "UnaryCall"
	NotFoundCall    = "NotFoundCall"
	File            = "grpc_testing/test.proto"
)

var (
	TestError = errors.Errorf("an error")
)

// ParseURL is a test helper that parses URLs into *url.URL, and fails the test on parse failure
func ParseURL(t *testing.T, rawurl string) *url.URL {
	u, err := url.Parse(rawurl)
	if err != nil {
		t.Fatal(err.Error())
	}
	return u
}

// NewFileDescriptor is a test helper that parses a .proto file into a file descriptor
func NewFileDescriptor(t *testing.T, file string) *desc.FileDescriptor {
	t.Helper()
	desc, err := desc.LoadFileDescriptor(file)
	if err != nil {
		t.Fatal(err.Error())
	}
	return desc
}

type FakeGrpcreflectClient struct {
	*desc.ServiceDescriptor
}

func (m *FakeGrpcreflectClient) ResolveService(serviceName string) (*desc.ServiceDescriptor, error) {
	if serviceName != TestService {
		return nil, errors.Errorf("service not found")
	}
	return m.ServiceDescriptor, nil
}

type FakeGrpcdynamicStub struct {
}

func (m *FakeGrpcdynamicStub) InvokeRpc(ctx context.Context, method *desc.MethodDescriptor, request proto.Message, opts ...grpc.CallOption) (proto.Message, error) {
	if method.GetName() == "UnaryCall" {
		return nil, status.Error(codes.Unimplemented, "unary unimplemented")
	}
	output := dynamic.NewMessage(method.GetOutputType())
	return output, nil
}
