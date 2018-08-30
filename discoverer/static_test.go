package discoverer

import (
	"reflect"
	"sync"
	"testing"

	"github.com/mercari/grpc-http-proxy"
	"github.com/mercari/grpc-http-proxy/log"
)

func TestNewStatic(t *testing.T) {
	cases := []struct {
		name     string
		yamlFile string
		expected map[string]versions
	}{
		{
			name:     "valid yaml",
			yamlFile: "test-fixtures/valid.yaml",
			expected: map[string]versions{
				"a": {
					"v1": entry{
						decidable: true,
						url:       parseURL("a.v1", t),
					},
					"v2": entry{
						decidable: true,
						url:       parseURL("a.v2", t),
					},
				},
				"b": {
					"v1": entry{
						decidable: true,
						url:       parseURL("b.v1", t),
					},
				},
			},
		},
		{
			name:     "invalid yaml",
			yamlFile: "test-fixtures/invalid.yaml",
			expected: map[string]versions{},
		},
		{
			name:     "missing yaml",
			yamlFile: "test-fixtures/does-not-exist.yaml",
			expected: map[string]versions{},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			logger := log.NewDiscard()
			local := NewStatic(logger, tc.yamlFile)
			if got, want := local.(*Static).m, tc.expected; !reflect.DeepEqual(got, want) {
				t.Fatalf("got %v, want %v", got, want)
			}
		})
	}

}

func TestStatic_Resolve(t *testing.T) {
	cases := []struct {
		name    string
		service string
		version string
		url     proxy.ServiceURL
		err     error
	}{
		{
			name:    "resolved",
			service: "a",
			version: "v1",
			url:     parseURL("a.v1", t),
			err:     nil,
		},
		{
			name:    "service not found",
			service: "b",
			version: "",
			url:     nil,
			err:     error(serviceNotFound("b")),
		},
	}
	r := records{
		m: map[string]versions{
			"a": {
				"v1": entry{
					decidable: true,
					url:       parseURL("a.v1", t),
				},
			},
		},
		mutex: sync.RWMutex{},
	}
	logger := log.NewDiscard()
	local := &Static{
		records: &r,
		logger:  logger,
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			u, err := local.Resolve(tc.service, tc.version)
			if got, want := u, tc.url; !reflect.DeepEqual(got, want) {
				t.Fatalf("got %v, want %v", got, want)
			}
			if got, want := err, tc.err; !reflect.DeepEqual(got, want) {
				t.Fatalf("got %v, want %v", got, want)
			}
		})
	}
}
