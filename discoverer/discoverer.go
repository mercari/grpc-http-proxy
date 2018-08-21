package discoverer

import (
	"github.com/mercari/grpc-http-proxy"
)

type Discoverer interface {
	Resolve(string, string) (proxy.ServiceURL, error)
}
