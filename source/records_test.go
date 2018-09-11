package source

import (
	"fmt"
	"net/url"
	"reflect"
	"sync"
	"testing"

	"github.com/mercari/grpc-http-proxy/errors"
)

func TestNewRecords(t *testing.T) {
	want := &Records{
		m:     make(map[string]versions),
		mutex: sync.RWMutex{},
	}
	got := NewRecords()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got: %v, want %v", got, want)
	}
}

func TestRecords_GetRecord(t *testing.T) {
	cases := []struct {
		name    string
		service string
		version string
		url     *url.URL
		err     *errors.Error
	}{
		{
			name:    "resolved (multi-version)",
			service: "a",
			version: "v1",
			url:     parseURL(t, "a.v1"),
			err:     nil,
		},
		{
			name:    "version not specified",
			service: "a",
			version: "",
			url:     nil,
			err: &errors.Error{
				Code: errors.VersionNotSpecified,
				Message: fmt.Sprintf("There are multiple version of the gRPC service %s available. "+
					"You must specify one", "a"),
			},
		},
		{
			name:    "version not found",
			service: "a",
			version: "v3",
			url:     nil,
			err: &errors.Error{
				Code:    errors.ServiceUnresolvable,
				Message: fmt.Sprintf("Version %s of the gRPC service %s is unresolvable", "v3", "a"),
			},
		},
		{
			name:    "resolved (single version)",
			service: "b",
			version: "",
			url:     parseURL(t, "b.v1"),
			err:     nil,
		},
		{
			name:    "service not found",
			service: "c",
			version: "",
			url:     nil,
			err: &errors.Error{
				Code:    errors.ServiceUnresolvable,
				Message: fmt.Sprintf("The gRPC service %s is unresolvable", "c"),
			},
		},
		{
			name:    "service undecidable (unversioned)",
			service: "d",
			version: "",
			url:     nil,
			err: &errors.Error{
				Code: errors.VersionUndecidable,
				Message: fmt.Sprintf("Multiple possible backends found for the gRPC service %s. "+
					"Add annotations to distinguish versions", "d"),
			},
		},
		{
			name:    "service undecidable (versioned)",
			service: "e",
			version: "v1",
			url:     nil,
			err: &errors.Error{
				Code: errors.VersionUndecidable,
				Message: fmt.Sprintf("Multiple possible backends found for the gRPC service %s. "+
					"Add annotations to distinguish versions", "e"),
			},
		},
	}

	r := Records{
		m: map[string]versions{
			"a": {
				"v1": []*url.URL{parseURL(t, "a.v1")},
				"v2": []*url.URL{parseURL(t, "a.v2")},
			},
			"b": {
				"v1": []*url.URL{parseURL(t, "b.v1")},
			},
			"d": {
				"": []*url.URL{parseURL(t, "d.v1"), parseURL(t, "d.v2")},
			},
			"e": {
				"v1": []*url.URL{parseURL(t, "e.v1"), parseURL(t, "e.v2")},
			},
		},
		mutex: sync.RWMutex{},
	}
	for _, tc := range cases {
		t.Run(string(tc.name), func(t *testing.T) {
			u, err := r.GetRecord(tc.service, tc.version)
			if got, want := u, tc.url; !reflect.DeepEqual(got, want) {
				t.Fatalf("got: %s, want %s", got.String(), want.String())
			}
			switch err.(type) {
			case nil:
				if tc.err != nil {
					t.Fatalf("got: %v, want %v", nil, tc.err)
				}
			case *errors.Error:
				err2, ok := err.(*errors.Error)
				if !ok {
					t.Fatalf("err was not *proxy.Error")
				}
				if got, want := err2, tc.err; !reflect.DeepEqual(got, want) {
					t.Fatalf("got: %v, want %v", got, want)
				}
			}
		})
	}
}

func TestRecords_SetRecord(t *testing.T) {
	cases := []struct {
		name     string
		service  string
		version  string
		url      *url.URL
		m        map[string]versions
		expected map[string]versions
	}{
		{
			name:    "add version",
			service: "a",
			version: "v2",
			url:     parseURL(t, "a.v2"),
			m: map[string]versions{
				"a": {
					"v1": []*url.URL{parseURL(t, "a.v1")},
				},
			},
			expected: map[string]versions{
				"a": {
					"v1": []*url.URL{parseURL(t, "a.v1")},
					"v2": []*url.URL{parseURL(t, "a.v2")},
				},
			},
		},
		{
			name:    "add service",
			service: "b",
			version: "v1",
			url:     parseURL(t, "b.v1"),
			m:       map[string]versions{},
			expected: map[string]versions{
				"b": {
					"v1": []*url.URL{parseURL(t, "b.v1")},
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(string(tc.name), func(t *testing.T) {
			r := Records{
				m:     tc.m,
				mutex: sync.RWMutex{},
			}
			r.SetRecord(tc.service, tc.version, tc.url)
			if got, want := r.m, tc.expected; !reflect.DeepEqual(got, want) {
				t.Fatalf("got: %v, want %v", got, want)
			}
		})
	}
}

func TestRecords_IsServiceUnique(t *testing.T) {
	cases := []struct {
		name    string
		service string
		b       bool
	}{
		{
			name:    "not decidable",
			service: "a",
			b:       false,
		},
		{
			name:    "decidable",
			service: "b",
			b:       true,
		},
	}

	r := Records{
		m: map[string]versions{
			"a": {
				"v1": []*url.URL{parseURL(t, "a.v1")},
				"v2": []*url.URL{parseURL(t, "a.v2")},
			},
			"b": {
				"v1": []*url.URL{parseURL(t, "b.v1")},
			},
		},
		mutex: sync.RWMutex{},
	}
	for _, tc := range cases {
		t.Run(string(tc.name), func(t *testing.T) {
			b := r.IsServiceUnique(tc.service)
			if got, want := b, tc.b; !reflect.DeepEqual(got, want) {
				t.Fatalf("got: %t, want %t", got, want)
			}
		})
	}
}

func TestRecords_RemoveRecord(t *testing.T) {
	cases := []struct {
		name     string
		service  string
		version  string
		url      *url.URL
		m        map[string]versions
		expected map[string]versions
	}{
		{
			name:    "delete version",
			service: "a",
			version: "v1",
			url:     parseURL(t, "a.v1"),
			m: map[string]versions{
				"a": {
					"v1": []*url.URL{parseURL(t, "a.v1")},
					"v2": []*url.URL{parseURL(t, "a.v2")},
				},
			},
			expected: map[string]versions{
				"a": {
					"v2": []*url.URL{parseURL(t, "a.v2")},
				},
			},
		},
		{
			name:    "no version",
			service: "c",
			version: "v1",
			url:     parseURL(t, "c.v1"),
			m: map[string]versions{
				"c": {
					"v2": []*url.URL{parseURL(t, "c.v2")},
				},
			},
			expected: map[string]versions{
				"c": {
					"v2": []*url.URL{parseURL(t, "c.v2")},
				},
			},
		},
		{
			name:    "delete service",
			service: "b",
			version: "v1",
			url:     parseURL(t, "b.v1"),
			m: map[string]versions{
				"b": {
					"v1": []*url.URL{parseURL(t, "b.v1")},
				},
			},
			expected: map[string]versions{},
		},
		{
			name:     "no service",
			service:  "c",
			version:  "v1",
			url:      parseURL(t, "c.v1"),
			m:        map[string]versions{},
			expected: map[string]versions{},
		},
		{
			name:    "delete version (duplicate)",
			service: "a",
			version: "",
			url:     parseURL(t, "a.v1"),
			m: map[string]versions{
				"a": {
					"": []*url.URL{
						parseURL(t, "a.v1"),
						parseURL(t, "a.v2"),
					},
				},
			},
			expected: map[string]versions{
				"a": {
					"": []*url.URL{parseURL(t, "a.v2")},
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(string(tc.name), func(t *testing.T) {
			r := Records{
				m:     tc.m,
				mutex: sync.RWMutex{},
			}
			r.RemoveRecord(tc.service, tc.version, tc.url)
			if got, want := r.m, tc.expected; !reflect.DeepEqual(got, want) {
				t.Fatalf("got: %v, want %v", got, want)
			}
		})
	}
}

func TestRecords_RecordExists(t *testing.T) {
	cases := []struct {
		name    string
		service string
		version string
		b       bool
	}{
		{
			name:    "exists",
			service: "a",
			version: "v1",
			b:       true,
		},
		{
			name:    "service does not exist",
			service: "b",
			version: "v1",
			b:       false,
		},
		{
			name:    "version does not exist",
			service: "a",
			version: "v2",
			b:       false,
		},
	}

	r := Records{
		m: map[string]versions{
			"a": {
				"v1": []*url.URL{parseURL(t, "a.v1")},
			},
		},
		mutex: sync.RWMutex{},
	}
	for _, tc := range cases {
		t.Run(string(tc.name), func(t *testing.T) {
			b := r.RecordExists(tc.service, tc.version)
			if got, want := b, tc.b; !reflect.DeepEqual(got, want) {
				t.Fatalf("got: %t, want %t", got, want)
			}
		})
	}
}
