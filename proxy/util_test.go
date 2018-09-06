package proxy

import (
	"net/url"
	"testing"

	"github.com/jhump/protoreflect/desc"
)

func serviceDescriptorFromFileDescriptor(fd *desc.FileDescriptor, service string) *serviceDescriptor {
	d := fd.FindService(service)
	if d == nil {
		return nil
	}
	return &serviceDescriptor{
		desc: d,
	}
}

func newFileDescriptor(t *testing.T, file string) *desc.FileDescriptor {
	t.Helper()
	desc, err := desc.LoadFileDescriptor(file)
	if err != nil {
		t.Fatal(err.Error())
	}
	return desc
}

func parseURL(t *testing.T, rawurl string) *url.URL {
	u, err := url.Parse(rawurl)
	if err != nil {
		t.Fatal(err.Error())
	}
	return u
}
