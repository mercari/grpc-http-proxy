package proxy

// Metadata is gRPC metadata sent to and from upstream
type Metadata map[string][]string

// GRPCResponse is the response gRPC message marshaled to JSON
type GRPCResponse []byte
