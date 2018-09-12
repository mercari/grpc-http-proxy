package proxy

import (
	"context"
	"testing"

	_ "google.golang.org/grpc/test/grpc_testing"

	"github.com/mercari/grpc-http-proxy/metadata"
	"github.com/mercari/grpc-http-proxy/proxy/proxytest"
	"github.com/mercari/grpc-http-proxy/proxy/reflection"
	pstub "github.com/mercari/grpc-http-proxy/proxy/stub"
)

func TestNewProxy(t *testing.T) {
	p := NewProxy()
	if p == nil {
		t.Fatalf("proxy was nil")
	}
}

func TestProxy_Connect(t *testing.T) {
	p := NewProxy()
	p.Connect(context.Background(), proxytest.ParseURL(t, "localhost:5000"))
}

func TestProxy_Call(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		p := NewProxy()
		ctx := context.Background()
		md := make(metadata.Metadata)

		p.stub = pstub.NewStub(&proxytest.FakeGrpcdynamicStub{})
		fd := proxytest.NewFileDescriptor(t, proxytest.File)
		sd := reflection.ServiceDescriptorFromFileDescriptor(fd, proxytest.TestService)
		p.reflector = reflection.NewReflector(&proxytest.FakeGrpcreflectClient{ServiceDescriptor: sd.ServiceDescriptor})

		_, err := p.Call(ctx, proxytest.TestService, proxytest.EmptyCall, []byte("{}"), &md)
		if err != nil {
			t.Fatalf("err should be nil, got %s", err.Error())
		}
	})

	t.Run("reflector fails", func(t *testing.T) {
		p := NewProxy()
		ctx := context.Background()
		md := make(metadata.Metadata)

		p.stub = pstub.NewStub(&proxytest.FakeGrpcdynamicStub{})
		p.reflector = reflection.NewReflector(&proxytest.FakeGrpcreflectClient{})

		_, err := p.Call(ctx, proxytest.NotFoundService, proxytest.EmptyCall, []byte("{}"), &md)
		if err == nil {
			t.Fatalf("err should be not nil")
		}
	})

	t.Run("invoking RPC returns error", func(t *testing.T) {
		p := NewProxy()
		ctx := context.Background()
		md := make(metadata.Metadata)

		p.stub = pstub.NewStub(&proxytest.FakeGrpcdynamicStub{})
		fd := proxytest.NewFileDescriptor(t, proxytest.File)
		sd := reflection.ServiceDescriptorFromFileDescriptor(fd, proxytest.TestService)
		p.reflector = reflection.NewReflector(&proxytest.FakeGrpcreflectClient{ServiceDescriptor: sd.ServiceDescriptor})

		_, err := p.Call(ctx, proxytest.TestService, proxytest.UnaryCall, []byte("{}"), &md)
		if err == nil {
			t.Fatalf("err should be not nil")
		}
	})
}
