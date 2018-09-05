package stub

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpc_metadata "google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/mercari/grpc-http-proxy/errors"
	"github.com/mercari/grpc-http-proxy/metadata"
	"github.com/mercari/grpc-http-proxy/proxy/reflection"
)

// Stub performs gRPC calls based on descriptors obtained through reflection
type Stub interface {
	// InvokeRPC calls the backend gRPC method with the message provided in JSON.
	// This performs reflection against the backend every time it is called.
	InvokeRPC(
		ctx context.Context,
		invocation *reflection.MethodInvocation,
		md *metadata.Metadata) (reflection.Message, error)
}

type stubImpl struct {
	stub grpcdynamicStub
}

type grpcdynamicStub interface {
	// This must be InvokeRpc with lower-case 'p' and 'c', because that is how the protoreflect library
	InvokeRpc(ctx context.Context, method *desc.MethodDescriptor, request proto.Message, opts ...grpc.CallOption) (proto.Message, error)
}

// NewStub creates a new Stub with the passed connection
func NewStub(s grpcdynamicStub) Stub {
	return &stubImpl{
		stub: s,
	}
}

func (s *stubImpl) InvokeRPC(
	ctx context.Context,
	invocation *reflection.MethodInvocation,
	md *metadata.Metadata) (reflection.Message, error) {

	o, err := s.stub.InvokeRpc(ctx,
		invocation.MethodDescriptor.AsProtoreflectDescriptor(),
		invocation.Message.AsProtoreflectMessage(),
		grpc.Header((*grpc_metadata.MD)(md)))
	if err != nil {
		stat := status.Convert(err)
		if err != nil && stat.Code() == codes.Unavailable {
			return nil, &errors.InternalError{
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
	outputMsg := invocation.MethodDescriptor.GetOutputType().NewMessage()
	err = outputMsg.ConvertFrom(o)

	if err != nil {
		return nil, &errors.InternalError{
			Code:    errors.Unknown,
			Message: "response from backend could not be converted internally; this is a bug",
		}
	}

	return outputMsg, nil
}
