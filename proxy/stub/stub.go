package stub

import (
	"context"
	"fmt"
	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/mercari/grpc-http-proxy/errors"
	"github.com/mercari/grpc-http-proxy/metadata"
	"github.com/mercari/grpc-http-proxy/proxy/reflection"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpc_metadata "google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Stub performs gRPC calls based on descriptors obtained through reflection
type Stub interface {
	// InvokeRPC calls the backend gRPC method with the message provided in JSON.
	// This performs reflection against the backend every time it is called.
	InvokeRPC(
		ctx context.Context,
		invocation *reflection.MethodInvocation,
		md *metadata.Metadata) (reflection.Message, grpc_metadata.MD, error)
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
	md *metadata.Metadata) (reflection.Message, grpc_metadata.MD, error) {

	var responseTrailer grpc_metadata.MD // variable to store responseTrailer

	message, err := s.stub.InvokeRpc(ctx,
		invocation.MethodDescriptor.AsProtoreflectDescriptor(),
		invocation.Message.AsProtoreflectMessage(),
		grpc.Header((*grpc_metadata.MD)(md)),
		grpc.Trailer(&responseTrailer))

	if err != nil {
		stat := status.Convert(err)

		if err != nil && stat.Code() == codes.Unavailable {
			return nil, nil, &errors.ProxyError{
				Code:    errors.UpstreamConnFailure,
				Message: fmt.Sprintf("could not connect to backend"),
			}
		}

		// When InvokeRPC returns an error, it should always be a gRPC error, so this should not panic
		return nil, nil, &errors.GRPCError{
			StatusCode: int(stat.Code()),
			Message:    stat.Message(),
			Details:    stat.Proto().Details,
		}
	}

  	outputMsg := invocation.MethodDescriptor.GetOutputType().NewMessage()
	err = outputMsg.ConvertFrom(message)

	if err != nil {
		return nil, nil, &errors.ProxyError{
			Code:    errors.Unknown,
			Message: "response from backend could not be converted internally; this is a bug",
		}
	}

	return outputMsg, responseTrailer, nil
}
