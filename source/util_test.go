package source

import (
	"net/url"
	"testing"

	"github.com/mercari/grpc-http-proxy"
)

func parseURL(t *testing.T, rawurl string) proxy.ServiceURL {
	u, err := url.Parse(rawurl)
	if err != nil {
		t.Fatal(err.Error())
	}
	return u
}
