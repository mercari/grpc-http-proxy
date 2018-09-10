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
	stub             pstub.Stub
}

// NewProxy creates a new client
func NewProxy(l *zap.Logger) *Proxy {
	return &Proxy{
		logger: l,
	}
}

// Connect opens a connection to target.
func (p *Proxy) Connect(ctx context.Context, target *url.URL) error {
	cc, err := grpc.DialContext(ctx, target.String(), grpc.WithInsecure())
	p.cc = cc
	rc := grpcreflect.NewClient(ctx, rpb.NewServerReflectionClient(p.cc))
	p.reflectionClient = reflection.NewReflectionClient(rc)
	p.stub = pstub.NewStub(p.cc)
	return err
}

// CloseConn closes the underlying connection
func (p *Proxy) CloseConn() error {
	return p.cc.Close()
}

// Call performs the gRPC call after doing reflection to obtain type information
func (p *Proxy) Call(ctx context.Context,
	serviceName, methodName string,
	message []byte,
	md *proxy.Metadata,
) (proxy.GRPCResponse, error) {
	serviceDesc, err := p.reflectionClient.ResolveService(ctx, serviceName)
	if err != nil {
		p.logger.Error("service was not found upstream",
			zap.String("service", serviceName),
		)
		return nil, err
	}

	methodDesc, err := serviceDesc.FindMethodByName(methodName)
	if err != nil {
		p.logger.Error("method was not found in service",
			zap.String("service", serviceName),
			zap.String("method", methodName),
		)
		return nil, err
	}

	inputMsg := methodDesc.GetInputType().NewMessage()

	err = inputMsg.UnmarshalJSON(message)
	if err != nil {
		p.logger.Error("failed to unmarshal input JSON",
			zap.String("err", err.Error()),
		)
		return nil, err
	}

	outputMsg, err := p.stub.InvokeRPC(ctx, methodDesc, inputMsg, md)
	if err != nil {
		return nil, err
	}
	m, err := outputMsg.MarshalJSON()
	if err != nil {
		p.logger.Error("failed to marshal output to JSON",
			zap.String("err", err.Error()),
		)
		return nil, err
	}
	return m, err
}
