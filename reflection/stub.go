package reflection

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

// ClientConn is a connection to a backend, with the ability to perform reflection
type ClientConn struct {
	cc *grpc.ClientConn
}

// NewClientConn creates a new ClientConn
func NewClientConn(
	ctx context.Context,
	target *url.URL) (*ClientConn, error) {
	cc, err := grpc.DialContext(ctx, target.String(), grpc.WithInsecure())
	if err != nil {
		return nil, &errors.Error{
			Code:    errors.UpstreamConnFailure,
			Message: fmt.Sprintf("failed to connect to backend %s", target),
		}
	}
	return &ClientConn{
		cc: cc,
	}, nil
}

func (c *ClientConn) Target() string {
	return c.cc.Target()
}

type Stub struct {
	stub grpcdynamic.Stub
}

func NewStub(c *ClientConn) *Stub {
	stub := grpcdynamic.NewStub(c.cc)
	return &Stub{
		stub: stub,
	}
}

// InvokeRPC calls the backend gRPC method with the message provided in JSON.
// This performs reflection against the backend every time it is called.
// TODO(tomoyat1) split this function up
func (s *Stub) InvokeRPC(
	ctx context.Context,
	method *MethodDescriptor,
	inputMessage *Message,
	md *proxy.Metadata) (*Message, error) {

	o, err := s.stub.InvokeRpc(ctx, method.desc, inputMessage.desc, grpc.Header((*metadata.MD)(md)))
	if err != nil {
		stat := status.Convert(err)
		if err != nil && stat.Code() == codes.Unavailable {
			return nil, &errors.Error{
				Code:    errors.UpstreamConnFailure,
				Message: fmt.Sprintf("could not connect to backend"),
			}
		}

		// When InvokeRPC returns an error, it should always be a gRPC error, so this should not panic
		return nil, &errors.GRPCError{
			StatusCode: int(stat.Code()),
			Message:    stat.Message(),
		}
	}
	outputMsg := method.GetOutputType().NewMessage()
	err = outputMsg.convertFrom(o)

	if err != nil {
		return nil, &errors.Error{
			Code:    errors.Unknown,
			Message: "response from backend could not be converted internally; this is a bug",
		}
	}

	return outputMsg, nil
}
