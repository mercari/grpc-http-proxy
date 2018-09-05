package proxy

// ServiceURL represents an service upstream url
type ServiceURL interface {
	String() string
}
