package proxy

import (
	"context"

	"google.golang.org/grpc/metadata"

	"github.com/mercari/grpc-http-proxy"
)

func NewOutgoingContext(ctx context.Context, md proxy.Metadata) context.Context {
	return metadata.NewOutgoingContext(ctx, metadata.MD(md))
}
