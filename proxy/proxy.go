package proxy

import (
	"context"
	"net/url"

	"go.uber.org/zap"

	"github.com/mercari/grpc-http-proxy"
)

// Proxy is a dynamic gRPC client that performs reflection
type Proxy struct {
	logger *zap.Logger
	*clientConn
	*reflectionClient
	*serviceDescriptor
	*methodDescriptor
	InputMessage  *message
	OutputMessage *message
	*stub
	err error
}

// Err returns the error that Proxy aborted on
func (c *Proxy) Err() error {
	return c.err
}

// NewProxy creates a new client
func NewProxy(l *zap.Logger) *Proxy {
	return &Proxy{
		logger:           l,
		clientConn:       &clientConn{},
		reflectionClient: &reflectionClient{},
		methodDescriptor: &methodDescriptor{},
		InputMessage:     &message{},
		OutputMessage:    &message{},
		stub:             &stub{},
	}
}

// Connect opens a connection to target
func (c *Proxy) Connect(ctx context.Context, target *url.URL) {
	if c.err != nil {
		return
	}
	cc, err := newClientConn(ctx, target)
	c.clientConn = cc
	c.err = err
	return
}

// CloseConn closes the underlying connection
func (c *Proxy) CloseConn() {
	if c.err != nil {
		return
	}
	c.err = c.clientConn.close()
	return
}

func (c *Proxy) newReflectionClient() {
	if c.err != nil {
		return
	}
	c.reflectionClient = newReflectionClient(c.clientConn)
	return
}

func (c *Proxy) resolveService(ctx context.Context, serviceName string) *serviceDescriptor {
	c.newReflectionClient()
	if c.err != nil {
		return nil
	}
	sd, err := c.reflectionClient.resolveService(ctx, serviceName)
	c.err = err
	return sd
}

func (c *Proxy) loadDescriptors(ctx context.Context, serviceName, methodName string) {
	if c.err != nil {
		return
	}
	c.methodDescriptor, c.err = c.resolveService(ctx, serviceName).findMethodByName(methodName)
	if c.err != nil {
		return
	}
	c.InputMessage = c.methodDescriptor.getInputType().newMessage()
	c.OutputMessage = c.methodDescriptor.getOutputType().newMessage()
	return
}

func (c *Proxy) unmarshalInputMessage(b []byte) {
	if c.err != nil {
		return
	}
	err := c.InputMessage.unmarshalJSON(b)
	c.err = err
	return
}

func (c *Proxy) marshalOutputMessage() proxy.GRPCResponse {
	if c.err != nil {
		return nil
	}
	b, err := c.InputMessage.marshalJSON()
	c.err = err
	return b
}

func (c *Proxy) newStub() {
	if c.err != nil {
		return
	}
	c.stub = newStub(c.clientConn)
	return
}

func (c *Proxy) invokeRPC(
	ctx context.Context,
	md *proxy.Metadata) {

	c.newStub()
	if c.err != nil {
		return
	}

	m, err := c.stub.invokeRPC(ctx, c.methodDescriptor, c.InputMessage, md)
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
	c.loadDescriptors(ctx, serviceName, methodName)
	c.unmarshalInputMessage(message)
	c.invokeRPC(ctx, md)
	response := c.marshalOutputMessage()
	return response, c.err
}
