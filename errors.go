package proxy

type Error struct {
	Code
	Message string
	Err     error
}

type Code int

const (
	InvalidPath         Code = 0
	MethodNotAllowed    Code = 1
	BackendConnFailure  Code = 2
	ServiceUnresolvable Code = 3
	ServiceNotFound     Code = 4
	MethodNotFound      Code = 5
	MessageTypeMismatch Code = 6
	Unauthorized        Code = 7
	Unknown             Code = 8
	VersionNotSpecified Code = 9
	VersionUndecidable  Code = 10
)

func (e *Error) Error() string {
	switch e.Code {
	case InvalidPath:
		return "nothing here"
	case MethodNotAllowed:
		return "method not allowed"
	case BackendConnFailure:
		return "could not connect to backend gRPC service"
	case ServiceUnresolvable:
		return "could not resolve service"
	case ServiceNotFound:
		return "service not found; service discovery error"
	case MethodNotFound:
		return "no such gRPC method"
	case MessageTypeMismatch:
		return "message type mismatch"
	case Unauthorized:
		return "unauthorized"
	case VersionNotSpecified:
		return "multiple versions of this service exist. specify version in request"
	case VersionUndecidable:
		return "multiple backends exist. add version annotations"
	default:
		return "unknown failure"
	}
}
