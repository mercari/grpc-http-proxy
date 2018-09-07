package proxy

import (
	"context"
	"net/url"

	"go.uber.org/zap"

	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc"
	rpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"

	"github.com/mercari/grpc-http-proxy"
	"github.com/mercari/grpc-http-proxy/proxy/reflection"
	pstub "github.com/mercari/grpc-http-proxy/proxy/stub"
)

// Proxy is a dynamic gRPC client that performs reflection
type Proxy struct {
	logger           *zap.Logger
	cc               *grpc.ClientConn
	reflectionClient reflection.ReflectionClient
	methodDescriptor reflection.MethodDescriptor
	InputMessage     reflection.Message
	OutputMessage    reflection.Message
	stub             pstub.Stub
	err              error
}

// Err returns the error that Proxy aborted on
func (p *Proxy) Err() error {
	return p.err
}

// NewProxy creates a new client
func NewProxy(l *zap.Logger) *Proxy {
	return &Proxy{
		logger: l,
	}
}

// Connect opens a connection to target
func (p *Proxy) Connect(ctx context.Context, target *url.URL) {
	if p.err != nil {
		return
	}
	cc, err := grpc.DialContext(ctx, target.String(), grpc.WithInsecure())
	p.cc = cc
	p.err = err
	return
}

// CloseConn closes the underlying connection
func (p *Proxy) CloseConn() {
	if p.err != nil {
		return
	}
	p.err = p.cc.Close()
	return
}

func (p *Proxy) newReflectionClient(ctx context.Context) {
	if p.err != nil {
		return
	}
	rc := grpcreflect.NewClient(ctx, rpb.NewServerReflectionClient(p.cc))
	p.reflectionClient = reflection.NewReflectionClient(rc)
}

func (p *Proxy) resolveService(ctx context.Context, serviceName string) reflection.ServiceDescriptor {
	if p.err != nil {
		return nil
	}
	sd, err := p.reflectionClient.ResolveService(ctx, serviceName)
	p.err = err
	return sd
}

func (p *Proxy) loadDescriptors(ctx context.Context, serviceName, methodName string) {
	if p.err != nil {
		return
	}
	s := p.resolveService(ctx, serviceName)
	if p.err != nil {
		return
	}
	p.methodDescriptor, p.err = s.FindMethodByName(methodName)
}

func (p *Proxy) createMessages() {
	if p.err != nil {
		return
	}
	p.InputMessage = p.methodDescriptor.GetInputType().NewMessage()
	p.OutputMessage = p.methodDescriptor.GetOutputType().NewMessage()
}

func (p *Proxy) unmarshalInputMessage(b []byte) {
	if p.err != nil {
		return
	}
	err := p.InputMessage.UnmarshalJSON(b)
	p.err = err
	return
}

func (p *Proxy) marshalOutputMessage() proxy.GRPCResponse {
	if p.err != nil {
		return nil
	}
	b, err := p.OutputMessage.MarshalJSON()
	p.err = err
	return b
}

func (p *Proxy) newStub() {
	if p.err != nil {
		return
	}
	p.stub = pstub.NewStub(p.cc)
	return
}

func (p *Proxy) invokeRPC(ctx context.Context, md *proxy.Metadata) {
	if p.err != nil {
		return
	}
	m, err := p.stub.InvokeRPC(ctx, p.methodDescriptor, p.InputMessage, md)
	p.err = err
	p.OutputMessage = m
	return
}

// Call performs the gRPC call after doing reflection to obtain type information
func (p *Proxy) Call(ctx context.Context,
	serviceName, methodName string,
	message []byte,
	md *proxy.Metadata,
) (proxy.GRPCResponse, error) {

	p.newReflectionClient(ctx)
	p.loadDescriptors(ctx, serviceName, methodName)
	p.createMessages()
	p.unmarshalInputMessage(message)
	p.newStub()
	p.invokeRPC(ctx, md)
	response := p.marshalOutputMessage()
	return response, p.err
}
