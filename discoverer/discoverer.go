package discoverer

import (
	"github.com/mercari/grpc-http-proxy"
)

// Discoverer does service discovery and provides name resolution of gRPC services
type Discoverer interface {
	Resolve(string, string) (proxy.ServiceURL, error)
}
