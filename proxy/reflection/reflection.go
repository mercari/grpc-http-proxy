package reflection

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/pkg/errors"

	perrors "github.com/mercari/grpc-http-proxy/errors"
)

// MethodInvocation contains a method and a message used to invoke an RPC
type MethodInvocation struct {
	*MethodDescriptor
	Message
}

// Reflector performs reflection on the gRPC service to obtain the method type
type Reflector interface {
	CreateInvocation(ctx context.Context, serviceName, methodName string, input []byte) (*MethodInvocation, error)
}

// NewReflector creates a new Reflector from the reflection client
func NewReflector(rc grpcreflectClient) Reflector {
	return &reflectorImpl{
		rc: newReflectionClient(rc),
	}
}

type reflectorImpl struct {
	rc *reflectionClient
}

// CreateInvocation creates a MethodInvocation by performing reflection
func (r *reflectorImpl) CreateInvocation(ctx context.Context,
	serviceName,
	methodName string,
	input []byte,
) (*MethodInvocation, error) {
	serviceDesc, err := r.rc.resolveService(ctx, serviceName)
	if err != nil {
		return nil, errors.Wrap(err, "service was not found upstream even though it should have been there")
	}
	methodDesc, err := serviceDesc.FindMethodByName(methodName)
	if err != nil {
		return nil, errors.Wrap(err, "method not found upstream")
	}
	inputMessage := methodDesc.GetInputType().NewMessage()
	err = inputMessage.UnmarshalJSON(input)
	if err != nil {
		return nil, err
	}
	return &MethodInvocation{
		MethodDescriptor: methodDesc,
		Message:          inputMessage,
	}, nil
}

// reflectionClient performs reflection to obtain descriptors
type reflectionClient struct {
	grpcreflectClient
}

type grpcreflectClient interface {
	ResolveService(serviceName string) (*desc.ServiceDescriptor, error)
}

// newReflectionClient creates a new ReflectionClient
func newReflectionClient(rc grpcreflectClient) *reflectionClient {
	return &reflectionClient{
		grpcreflectClient: rc,
	}
}

func (c *reflectionClient) resolveService(ctx context.Context,
	serviceName string) (*ServiceDescriptor, error) {
	d, err := c.grpcreflectClient.ResolveService(serviceName)
	if err != nil {
		return nil, &perrors.ProxyError{
			Code:    perrors.ServiceNotFound,
			Message: fmt.Sprintf("service %s was not found upstream", serviceName),
		}
	}
	return &ServiceDescriptor{
		ServiceDescriptor: d,
	}, nil
}

// ServiceDescriptor represents a service type
type ServiceDescriptor struct {
	*desc.ServiceDescriptor
}

// ServiceDescriptorFromFileDescriptor finds the service descriptor from a file descriptor
// This can be useful in tests that don't connect to a real server
func ServiceDescriptorFromFileDescriptor(fd *desc.FileDescriptor, service string) *ServiceDescriptor {
	d := fd.FindService(service)
	if d == nil {
		return nil
	}
	return &ServiceDescriptor{
		ServiceDescriptor: d,
	}
}

// FindMethodByName finds the method descriptor by name from the service descriptor
func (s *ServiceDescriptor) FindMethodByName(name string) (*MethodDescriptor, error) {
	d := s.ServiceDescriptor.FindMethodByName(name)
	if d == nil {
		return nil, &perrors.ProxyError{
			Code:    perrors.MethodNotFound,
			Message: fmt.Sprintf("the method %s was not found", name),
		}
	}
	return &MethodDescriptor{
		MethodDescriptor: d,
	}, nil
}

// MethodDescriptor represents a method type
type MethodDescriptor struct {
	*desc.MethodDescriptor
}

// GetInputType gets the MessageDescriptor for the method input type
func (m *MethodDescriptor) GetInputType() *MessageDescriptor {
	return &MessageDescriptor{
		desc: m.MethodDescriptor.GetInputType(),
	}
}

// GetOutputType gets the MessageDescriptor for the method output type
func (m *MethodDescriptor) GetOutputType() *MessageDescriptor {
	return &MessageDescriptor{
		desc: m.MethodDescriptor.GetOutputType(),
	}
}

// AsProtoreflectDescriptor returns the underlying protoreflect method descriptor
func (m *MethodDescriptor) AsProtoreflectDescriptor() *desc.MethodDescriptor {
	return m.MethodDescriptor
}

// MessageDescriptor represents a message type
type MessageDescriptor struct {
	desc *desc.MessageDescriptor
}

// NewMessage creates a new message from the message descriptor
func (m *MessageDescriptor) NewMessage() *messageImpl {
	return &messageImpl{
		Message: dynamic.NewMessage(m.desc),
	}
}

type Message interface {
	// MarshalJSON marshals the Message into JSON
	MarshalJSON() ([]byte, error)
	// UnmarshalJSON unmarshals JSON into a Message
	UnmarshalJSON(b []byte) error
	// ConvertFrom converts a raw protobuf message into a Message
	ConvertFrom(target proto.Message) error
	// AsProtoreflectMessage returns the underlying protoreflect message
	AsProtoreflectMessage() *dynamic.Message
}

// messageImpl is an message value
type messageImpl struct {
	*dynamic.Message
}

func (m *messageImpl) MarshalJSON() ([]byte, error) {
	b, err := m.Message.MarshalJSON()
	if err != nil {
		return nil, &perrors.ProxyError{
			Code:    perrors.Unknown,
			Message: "could not marshal backend response into JSON",
		}
	}
	return b, nil
}

func (m *messageImpl) UnmarshalJSON(b []byte) error {
	if err := m.Message.UnmarshalJSON(b); err != nil {
		return &perrors.ProxyError{
			Code:    perrors.MessageTypeMismatch,
			Message: "input JSON does not match messageImpl type",
		}
	}
	return nil
}

func (m *messageImpl) ConvertFrom(target proto.Message) error {
	return m.Message.ConvertFrom(target)
}

func (m *messageImpl) AsProtoreflectMessage() *dynamic.Message {
	return m.Message
}
