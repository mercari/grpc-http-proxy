package source

import (
	"net/url"
	"testing"
)

func parseURL(t *testing.T, rawurl string) *url.URL {
	u, err := url.Parse(rawurl)
	if err != nil {
		t.Fatal(err.Error())
	}
	return u
}
