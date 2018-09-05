package metadata

import (
	"reflect"
	"testing"
)

func TestMetadataFromHeaders(t *testing.T) {
	cases := []struct {
		name     string
		headers  map[string][]string
		metadata Metadata
	}{
		{
			name: "convert",
			headers: map[string][]string{
				"Grpc-Metadata-Foo": {"hoge", "fuga"},
				"Other-Header":      {"this", "should", "not", "be", "here"},
			},
			metadata: Metadata{
				"foo": {"hoge", "fuga"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := MetadataFromHeaders(tc.headers)
			if got, want := m, tc.metadata; !reflect.DeepEqual(got, want) {
				t.Fatalf("got %v, want %v", got, want)
			}
		})
	}
}

func TestMetadata_ToHeaders(t *testing.T) {
	cases := []struct {
		name     string
		headers  map[string][]string
		metadata Metadata
	}{
		{
			name: "convert",
			headers: map[string][]string{
				"Grpc-Metadata-foo": {"hoge", "fuga"},
			},
			metadata: Metadata{
				"foo": {"hoge", "fuga"},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := tc.metadata.ToHeaders()
			if got, want := h, tc.headers; !reflect.DeepEqual(got, want) {
				t.Fatalf("got %v, want %v", got, want)
			}
		})
	}
}
