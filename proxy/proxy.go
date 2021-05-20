package proxy

import (
	"context"
	"github.com/jhump/protoreflect/dynamic/grpcdynamic"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/mercari/grpc-http-proxy/metadata"
	"github.com/mercari/grpc-http-proxy/proxy/reflection"
	pstub "github.com/mercari/grpc-http-proxy/proxy/stub"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	grpc_metadata "google.golang.org/grpc/metadata"
	rpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"
	"net/url"
)

// Proxy is a dynamic gRPC client that performs reflection
type Proxy struct {
	conn      *grpc.ClientConn
	reflector reflection.Reflector
	stub      pstub.Stub
}

// NewProxy creates a new client
func NewProxy() *Proxy {
	return &Proxy{}
}

// Connect opens a connection to target.
func (p *Proxy) Connect(ctx context.Context, target *url.URL) error {
	conn, err := grpc.DialContext(ctx, target.String(), grpc.WithInsecure())
	if err != nil {
		return err
	}
	p.conn = conn
	rc := grpcreflect.NewClient(ctx, rpb.NewServerReflectionClient(p.conn))
	p.reflector = reflection.NewReflector(rc)
	p.stub = pstub.NewStub(grpcdynamic.NewStub(p.conn))
	return err
}

// CloseConn closes the underlying connection
func (p *Proxy) CloseConn() error {
	return p.conn.Close()
}

// Call performs the gRPC call after doing reflection to obtain type information
func (p *Proxy) Call(ctx context.Context,
	serviceName, methodName string,
	message []byte,
	md *metadata.Metadata,
) ([]byte, grpc_metadata.MD, error) {
	invocation, err := p.reflector.CreateInvocation(ctx, serviceName, methodName, message)
	if err != nil {
		return nil, nil, err
	}

	outputMsg, responseTrailer, err := p.stub.InvokeRPC(ctx, invocation, md)

	if err != nil {
		return nil, nil, err
	}

	m, err := outputMsg.MarshalJSON()

	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to marshal output JSON")
	}
	return m, responseTrailer, err
}
