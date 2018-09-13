package metadata

import "strings"

// This is from an old grpc-gateway (https://github.com/grpc-ecosystem/grpc-gateway) specification
const metadataHeaderPrefix = "Grpc-Metadata-"

// Metadata is gRPC metadata sent to and from upstream
type Metadata map[string][]string

func MetadataFromHeaders(raw map[string][]string) Metadata {
	m := make(map[string][]string, len(raw))
	for rawK, v := range raw {
		if k := extractGrpcMetadataKey(rawK); k != "" {
			k = strings.ToLower(k)
			m[k] = v
		}
	}
	return m
}

func extractGrpcMetadataKey(rawKey string) string {
	if !strings.HasPrefix(rawKey, metadataHeaderPrefix) {
		return ""
	}
	return strings.TrimPrefix(rawKey, metadataHeaderPrefix)
}

func (m Metadata) ToHeaders() map[string][]string {
	h := make(map[string][]string, len(m))
	for k, v := range m {
		h[metadataHeaderPrefix+k] = v
	}
	return h
}
