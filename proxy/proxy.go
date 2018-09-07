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
func (c *Proxy) Err() error {
	return c.err
}

// NewProxy creates a new client
func NewProxy(l *zap.Logger) *Proxy {
	return &Proxy{
		logger: l,
	}
}

// Connect opens a connection to target
func (c *Proxy) Connect(ctx context.Context, target *url.URL) {
	if c.err != nil {
		return
	}
	cc, err := grpc.DialContext(ctx, target.String(), grpc.WithInsecure())
	c.cc = cc
	c.err = err
	return
}

// CloseConn closes the underlying connection
func (c *Proxy) CloseConn() {
	if c.err != nil {
		return
	}
	c.err = c.cc.Close()
	return
}

func (c *Proxy) newReflectionClient(ctx context.Context) {
	if c.err != nil {
		return
	}
	rc := grpcreflect.NewClient(ctx, rpb.NewServerReflectionClient(c.cc))
	c.reflectionClient = reflection.NewReflectionClient(rc)
}

func (c *Proxy) resolveService(ctx context.Context, serviceName string) reflection.ServiceDescriptor {
	if c.err != nil {
		return nil
	}
	sd, err := c.reflectionClient.ResolveService(ctx, serviceName)
	c.err = err
	return sd
}

func (c *Proxy) loadDescriptors(ctx context.Context, serviceName, methodName string) {
	if c.err != nil {
		return
	}
	s := c.resolveService(ctx, serviceName)
	if c.err != nil {
		return
	}
	c.methodDescriptor, c.err = s.FindMethodByName(methodName)
}

func (c *Proxy) createMessages() {
	if c.err != nil {
		return
	}
	c.InputMessage = c.methodDescriptor.GetInputType().NewMessage()
	c.OutputMessage = c.methodDescriptor.GetOutputType().NewMessage()
}

func (c *Proxy) unmarshalInputMessage(b []byte) {
	if c.err != nil {
		return
	}
	err := c.InputMessage.UnmarshalJSON(b)
	c.err = err
	return
}

func (c *Proxy) marshalOutputMessage() proxy.GRPCResponse {
	if c.err != nil {
		return nil
	}
	b, err := c.OutputMessage.MarshalJSON()
	c.err = err
	return b
}

func (c *Proxy) newStub() {
	if c.err != nil {
		return
	}
	c.stub = pstub.NewStub(c.cc)
	return
}

func (c *Proxy) invokeRPC(ctx context.Context, md *proxy.Metadata) {
	if c.err != nil {
		return
	}
	m, err := c.stub.InvokeRPC(ctx, c.methodDescriptor, c.InputMessage, md)
	c.err = err
	c.OutputMessage = m
	return
}

// Call performs the gRPC call after doing reflection to obtain type information
func (c *Proxy) Call(ctx context.Context,
	serviceName, methodName string,
	message []byte,
	md *proxy.Metadata,
) (proxy.GRPCResponse, error) {

	c.newReflectionClient(ctx)
	c.loadDescriptors(ctx, serviceName, methodName)
	c.createMessages()
	c.unmarshalInputMessage(message)
	c.newStub()
	c.invokeRPC(ctx, md)
	response := c.marshalOutputMessage()
	return response, c.err
}
