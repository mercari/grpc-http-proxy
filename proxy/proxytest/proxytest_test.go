package proxytest

import (
	"testing"

	"context"
	"github.com/jhump/protoreflect/dynamic"
	_ "google.golang.org/grpc/test/grpc_testing"
)

func TestNewFileDescriptor(t *testing.T) {
	fd := NewFileDescriptor(t, File)
	if fd == nil {
		t.Fatal("file descriptor was nil")
	}
}

func TestParseURL(t *testing.T) {
	u := ParseURL(t, "localhost:5000")
	if u == nil {
		t.Fatal("file descriptor was nil")
	}
}

func TestFakeGrpcdynamicStub_InvokeRpc(t *testing.T) {
	cases := []struct {
		name        string
		serviceName string
		isDescNil   bool
		isErrNil    bool
	}{
		{
			name:        "found",
			serviceName: TestService,
			isDescNil:   false,
			isErrNil:    true,
		},
		{
			name:        "found",
			serviceName: NotFoundService,
			isDescNil:   true,
			isErrNil:    false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fd := NewFileDescriptor(t, File)
			rc := &FakeGrpcreflectClient{
				ServiceDescriptor: fd.FindService(TestService),
			}
			sd, err := rc.ResolveService(tc.serviceName)
			if got, want := sd == nil, tc.isDescNil; got != want {
				t.Fatalf("got %t, want %t", got, want)
			}
			if got, want := err == nil, tc.isErrNil; got != want {
				t.Fatalf("got %t, want %t", got, want)
			}
		})
	}
}

func TestFakeGrpcreflectClient_ResolveService(t *testing.T) {
	cases := []struct {
		name         string
		methodName   string
		isMessageNil bool
		isErrNil     bool
	}{
		{
			name:         "found",
			methodName:   EmptyCall,
			isMessageNil: false,
			isErrNil:     true,
		},
		{
			name:         "found",
			methodName:   UnaryCall,
			isMessageNil: true,
			isErrNil:     false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			fd := NewFileDescriptor(t, File)
			methodDesc := fd.FindService(TestService).
				FindMethodByName(tc.methodName)
			message := dynamic.NewMessage(methodDesc.GetInputType())

			s := FakeGrpcdynamicStub{}
			msg, err := s.InvokeRpc(ctx, methodDesc, message)
			if got, want := msg == nil, tc.isMessageNil; got != want {
				t.Fatalf("got %t, want %t", got, want)
			}
			if got, want := err == nil, tc.isErrNil; got != want {
				t.Fatalf("got %t, want %t", got, want)
			}
		})
	}
}
