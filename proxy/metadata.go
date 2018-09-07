package proxy

import (
	"context"

	"google.golang.org/grpc/metadata"

	"github.com/mercari/grpc-http-proxy"
)

// NewOutgoingContext adds gRPC metadata to the context
func NewOutgoingContext(ctx context.Context, md proxy.Metadata) context.Context {
	return metadata.NewOutgoingContext(ctx, metadata.MD(md))
}
