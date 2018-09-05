package discoverer

import (
	"net/url"
	"reflect"
	"sync"
	"testing"

	"github.com/mercari/grpc-http-proxy"
)

func parseURL(urlStr string, t *testing.T) proxy.ServiceURL {
	t.Helper()
	u, err := url.Parse(urlStr)
	if err != nil {
		t.Errorf("parsing of url failed: %s", err.Error())
	}
	return u
}

func TestNewRecords(t *testing.T) {
	want := &records{
		m:     make(map[string]versions),
		mutex: sync.RWMutex{},
	}
	got := NewRecords()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got: %v, want %v", got, want)
	}
}

func TestNewRecordsFromYAML(t *testing.T) {
	cases := []struct {
		name        string
		mappingFile string
		expected    map[string]versions
		err         error
	}{
		{
			name:        "valid yaml",
			mappingFile: "test-fixtures/valid.yaml",
			expected: map[string]versions{
				"a": {
					"v1": entry{
						true,
						parseURL("a.v1", t),
					},
					"v2": entry{
						true,
						parseURL("a.v2", t),
					},
				},
				"b": {
					"v1": entry{
						true,
						parseURL("b.v1", t),
					},
				},
			},
			err: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r, err := NewRecordsFromYAML(tc.mappingFile)
			if got, want := r.m, tc.expected; !reflect.DeepEqual(got, want) {
				t.Fatalf("got %v, want %v", got, want)
			}
			if got, want := err, tc.err; !reflect.DeepEqual(got, want) {
				t.Fatalf("got %v, want %v", got, want)
			}
		})
	}
}

func TestRecords_GetRecord(t *testing.T) {
	cases := []struct {
		name    string
		service string
		version string
		url     proxy.ServiceURL
		err     *proxy.Error
	}{
		{
			name:    "resolved (multi-version)",
			service: "a",
			version: "v1",
			url:     parseURL("a.v1", t),
			err:     nil,
		},
		{
			name:    "version not specified",
			service: "a",
			version: "",
			url:     nil,
			err:     versionNotSpecified("a"),
		},
		{
			name:    "version not found",
			service: "a",
			version: "v3",
			url:     nil,
			err:     versionNotFound("a", "v3"),
		},
		{
			name:    "resolved (single version)",
			service: "b",
			version: "",
			url:     parseURL("b.v1", t),
			err:     nil,
		},
		{
			name:    "service not found",
			service: "c",
			version: "",
			url:     nil,
			err:     serviceNotFound("c"),
		},
		{
			name:    "service undecidable (unversioned)",
			service: "d",
			version: "",
			url:     nil,
			err:     versionUndecidable("d"),
		},
		{
			name:    "service undecidable (versioned)",
			service: "e",
			version: "",
			url:     nil,
			err:     versionUndecidable("e"),
		},
	}

	r := records{
		m: map[string]versions{
			"a": {
				"v1": entry{
					true,
					parseURL("a.v1", t),
				},
				"v2": entry{
					true,
					parseURL("a.v2", t),
				},
			},
			"b": {
				"v1": entry{
					true,
					parseURL("b.v1", t),
				},
			},
			"d": {
				"": entry{
					false,
					nil,
				},
			},
			"e": {
				"v1": entry{
					false,
					nil,
				},
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
			case *proxy.Error:
				err2, ok := err.(*proxy.Error)
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
		url      proxy.ServiceURL
		m        map[string]versions
		expected map[string]versions
	}{
		{
			name:    "add version",
			service: "a",
			version: "v2",
			url:     parseURL("a.v2", t),
			m: map[string]versions{
				"a": {
					"v1": entry{
						true,
						parseURL("a.v1", t),
					},
				},
			},
			expected: map[string]versions{
				"a": {
					"v1": entry{
						true,
						parseURL("a.v1", t),
					},
					"v2": entry{
						true,
						parseURL("a.v2", t),
					},
				},
			},
		},
		{
			name:    "add service",
			service: "b",
			version: "v1",
			url:     parseURL("b.v1", t),
			m:       map[string]versions{},
			expected: map[string]versions{
				"b": {
					"v1": entry{
						true,
						parseURL("b.v1", t),
					},
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(string(tc.name), func(t *testing.T) {
			r := records{
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

	r := records{
		m: map[string]versions{
			"a": {
				"v1": entry{
					true,
					parseURL("a.v1", t),
				},
				"v2": entry{
					true,
					parseURL("a.v2", t),
				},
			},
			"b": {
				"v1": entry{
					true,
					parseURL("b.v1", t),
				},
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
		m        map[string]versions
		expected map[string]versions
	}{
		{
			name:    "delete version",
			service: "a",
			version: "v1",
			m: map[string]versions{
				"a": {
					"v1": entry{
						true,
						parseURL("a.v1", t),
					},
					"v2": entry{
						true,
						parseURL("a.v2", t),
					},
				},
			},
			expected: map[string]versions{
				"a": {
					"v2": entry{
						true,
						parseURL("a.v2", t),
					},
				},
			},
		},
		{
			name:    "delete service",
			service: "b",
			version: "v1",
			m: map[string]versions{
				"b": {
					"v1": entry{
						true,
						parseURL("b.v1", t),
					},
				},
			},
			expected: map[string]versions{},
		},
		{
			name:     "no service",
			service:  "c",
			version:  "v1",
			m:        map[string]versions{},
			expected: map[string]versions{},
		},
	}
	for _, tc := range cases {
		t.Run(string(tc.name), func(t *testing.T) {
			r := records{
				m:     tc.m,
				mutex: sync.RWMutex{},
			}
			r.RemoveRecord(tc.service, tc.version)
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

	r := records{
		m: map[string]versions{
			"a": {
				"v1": entry{
					true,
					parseURL("a.v1", t),
				},
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
