package proxy

import (
	"context"
	"fmt"
	"net/url"

	"github.com/jhump/protoreflect/dynamic/grpcdynamic"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/mercari/grpc-http-proxy"
	"github.com/mercari/grpc-http-proxy/errors"
)

// clientConn is a connection to a backend, with the ability to perform reflection
type clientConn struct {
	cc *grpc.ClientConn
}

// newClientConn creates a new clientConn
func newClientConn(
	ctx context.Context,
	target *url.URL) (*clientConn, error) {
	cc, err := grpc.DialContext(ctx, target.String(), grpc.WithInsecure())
	if err != nil {
		return nil, &errors.Error{
			Code:    errors.UpstreamConnFailure,
			Message: fmt.Sprintf("failed to connect to backend %s", target),
		}
	}
	return &clientConn{
		cc: cc,
	}, nil
}

func (c *clientConn) target() string {
	return c.cc.Target()
}

func (c *clientConn) close() error {
	return c.cc.Close()
}

type stub struct {
	stub grpcdynamic.Stub
}

func newStub(c *clientConn) *stub {
	s := grpcdynamic.NewStub(c.cc)
	return &stub{
		stub: s,
	}
}

// invokeRPC calls the backend gRPC method with the message provided in JSON.
// This performs reflection against the backend every time it is called.
func (s *stub) invokeRPC(
	ctx context.Context,
	method *methodDescriptor,
	inputMessage *message,
	md *proxy.Metadata) (*message, error) {

	o, err := s.stub.InvokeRpc(ctx, method.desc, inputMessage.desc, grpc.Header((*metadata.MD)(md)))
	if err != nil {
		stat := status.Convert(err)
		if err != nil && stat.Code() == codes.Unavailable {
			return nil, &errors.Error{
				Code:    errors.UpstreamConnFailure,
				Message: fmt.Sprintf("could not connect to backend"),
			}
		}

		// When invokeRPC returns an error, it should always be a gRPC error, so this should not panic
		return nil, &errors.GRPCError{
			StatusCode: int(stat.Code()),
			Message:    stat.Message(),
		}
	}
	outputMsg := method.getOutputType().newMessage()
	err = outputMsg.convertFrom(o)

	if err != nil {
		return nil, &errors.Error{
			Code:    errors.Unknown,
			Message: "response from backend could not be converted internally; this is a bug",
		}
	}

	return outputMsg, nil
}
