package proxy

import (
	"context"
	"fmt"
	"net/url"

	"github.com/jhump/protoreflect/dynamic/grpcdynamic"
	"github.com/jhump/protoreflect/grpcreflect"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	rpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"

	"github.com/mercari/grpc-http-proxy/metadata"
	"github.com/mercari/grpc-http-proxy/proxy/reflection"
	pstub "github.com/mercari/grpc-http-proxy/proxy/stub"
)

// Proxy is a dynamic gRPC client that performs reflection
type Proxy struct {
	cc        *grpc.ClientConn
	reflector reflection.Reflector
	stub      pstub.Stub
	rc        *grpcreflect.Client
}

// NewProxy creates a new client
func NewProxy() *Proxy {
	return &Proxy{}
}

// Connect opens a connection to target.
func (p *Proxy) Connect(ctx context.Context, target *url.URL) error {
	cc, err := grpc.DialContext(ctx, target.String(), grpc.WithInsecure())
	if err != nil {
		return err
	}
	p.cc = cc
	rc := grpcreflect.NewClient(ctx, rpb.NewServerReflectionClient(p.cc))
	p.rc = rc
	p.reflector = reflection.NewReflector(rc)
	p.stub = pstub.NewStub(grpcdynamic.NewStub(p.cc))
	return err
}

// ListServices is
func (p *Proxy) ListServices() ([]string, error) {
	return p.rc.ListServices()
}

// ListMethods is
func (p *Proxy) ListMethods(service string) ([]string, error) {
	d, err := p.rc.ResolveService(service)
	if err != nil {
		return nil, err
	}
	md := d.GetMethods()
	methods := make([]string, len(md))

	for i, m := range md {
		methods[i] = m.GetName()
	}

	return methods, nil
}

// ListFields is
func (p *Proxy) ListFields(service, method string) ([]string, error) {
	d, err := p.rc.ResolveService(service)
	if err != nil {
		return nil, err
	}

	m := d.FindMethodByName(method)
	fs := m.GetInputType().GetFields()

	fields := make([]string, len(fs))

	for i, f := range fs {
		fmt.Println(f.AsProto())
		fields[i] = f.GetJSONName()
	}

	return fields, nil
}

// CloseConn closes the underlying connection
func (p *Proxy) CloseConn() error {
	return p.cc.Close()
}

// Call performs the gRPC call after doing reflection to obtain type information
func (p *Proxy) Call(ctx context.Context,
	serviceName, methodName string,
	message []byte,
	md *metadata.Metadata,
) ([]byte, error) {
	invocation, err := p.reflector.CreateInvocation(ctx, serviceName, methodName, message)
	if err != nil {
		return nil, err
	}

	outputMsg, err := p.stub.InvokeRPC(ctx, invocation, md)
	if err != nil {
		return nil, err
	}
	m, err := outputMsg.MarshalJSON()
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal output JSON")
	}
	return m, err
}
