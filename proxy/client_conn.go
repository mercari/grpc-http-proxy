package proxy

import (
	"context"
	"fmt"
	"net/url"

	"google.golang.org/grpc"

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
