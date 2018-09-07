//go:generate mockgen -destination mock/reflection_mock.go github.com/mercari/grpc-http-proxy/proxy/reflection ReflectionClient,ServiceDescriptor,MethodDescriptor,MessageDescriptor,Message

package reflection

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"

	"github.com/mercari/grpc-http-proxy/errors"
)

// ReflectionClient performs reflection to obtain descriptors
type ReflectionClient interface {
	// ResolveService gets the service descriptor from the service the client is connected to
	ResolveService(ctx context.Context, serviceName string) (ServiceDescriptor, error)
}

type reflectionClientImpl struct {
	grpcdynamicClient
}

type grpcdynamicClient interface {
	ResolveService(serviceName string) (*desc.ServiceDescriptor, error)
}

// NewReflectionClient creates a new ReflectionClient
func NewReflectionClient(rc grpcdynamicClient) ReflectionClient {
	return &reflectionClientImpl{
		grpcdynamicClient: rc,
	}
}

func (c *reflectionClientImpl) ResolveService(ctx context.Context,
	serviceName string) (ServiceDescriptor, error) {
	d, err := c.grpcdynamicClient.ResolveService(serviceName)
	if err != nil {
		return nil, &errors.Error{
			Code:    errors.ServiceNotFound,
			Message: fmt.Sprintf("service %s was not found upstream", serviceName),
		}
	}
	return &serviceDescriptorImpl{
		desc: d,
	}, nil
}

// ServiceDescriptor represents a service type
type ServiceDescriptor interface {
	// FindMethodByName finds the method descriptor with the name
	FindMethodByName(name string) (MethodDescriptor, error)
}
type serviceDescriptorImpl struct {
	desc serviceDescriptor
}

// serviceDescriptor is the interface for message.serviceDescriptorImpl
type serviceDescriptor interface {
	FindMethodByName(name string) *desc.MethodDescriptor
}

// ServiceDescriptorFromFileDescriptor finds the service descriptor from a file descriptor
// This can be useful in tests that don't connect to a real server
func ServiceDescriptorFromFileDescriptor(fd *desc.FileDescriptor, service string) ServiceDescriptor {
	d := fd.FindService(service)
	if d == nil {
		return nil
	}
	return &serviceDescriptorImpl{
		desc: d,
	}
}

func (s *serviceDescriptorImpl) FindMethodByName(name string) (MethodDescriptor, error) {
	d := s.desc.FindMethodByName(name)
	if d == nil {
		return nil, &errors.Error{
			Code:    errors.MethodNotFound,
			Message: fmt.Sprintf("the method %s was not found", name),
		}
	}
	return &methodDescriptorImpl{
		desc: d,
	}, nil
}

// MethodDescriptor represents a method type
type MethodDescriptor interface {
	// GetInputType gets the messageImpl descriptor for the input type for the method
	GetInputType() MessageDescriptor
	// GetOutputType gets the messageImpl descriptor for the output type for the method
	GetOutputType() MessageDescriptor
	// AsProtoreflectDescriptor returns the underlying protoreflect method descriptor
	AsProtoreflectDescriptor() *desc.MethodDescriptor
}

type methodDescriptorImpl struct {
	desc methodDescriptor
}

// methodDescriptor is an interface for message.methodDescriptorImpl
type methodDescriptor interface {
	GetInputType() *desc.MessageDescriptor
	GetOutputType() *desc.MessageDescriptor
}

func (m *methodDescriptorImpl) GetInputType() MessageDescriptor {
	return &messageDescriptorImpl{
		desc: m.desc.GetInputType(),
	}
}

func (m *methodDescriptorImpl) GetOutputType() MessageDescriptor {
	return &messageDescriptorImpl{
		desc: m.desc.GetOutputType(),
	}
}

func (m *methodDescriptorImpl) AsProtoreflectDescriptor() *desc.MethodDescriptor {
	d, ok := m.desc.(*desc.MethodDescriptor)
	if !ok {
		return &desc.MethodDescriptor{}
	}
	return d
}

// MessageDescriptor represents a message type
type MessageDescriptor interface {
	// NewMessage creates a new messageImpl from the receiver
	NewMessage() Message
}

type messageDescriptorImpl struct {
	desc *desc.MessageDescriptor
}

func (m *messageDescriptorImpl) NewMessage() Message {
	return &messageImpl{
		message: dynamic.NewMessage(m.desc),
	}
}

// Message is an message value
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

type messageImpl struct {
	message
}

// message is an interface for dynamic.messageImpl
type message interface {
	MarshalJSON() ([]byte, error)
	UnmarshalJSON([]byte) error
	ConvertFrom(target proto.Message) error
	SetField(fd *desc.FieldDescriptor, val interface{})
	FindFieldDescriptorByName(name string) *desc.FieldDescriptor
}

func (m *messageImpl) MarshalJSON() ([]byte, error) {
	b, err := m.message.MarshalJSON()
	if err != nil {
		return nil, &errors.Error{
			Code:    errors.Unknown,
			Message: "could not marshal backend response into JSON",
		}
	}
	return b, nil
}

func (m *messageImpl) UnmarshalJSON(b []byte) error {
	err := m.message.UnmarshalJSON(b)
	if err != nil {
		return &errors.Error{
			Code:    errors.MessageTypeMismatch,
			Message: "input JSON does not match messageImpl type",
		}
	}
	return nil
}

func (m *messageImpl) ConvertFrom(target proto.Message) error {
	return m.message.ConvertFrom(target)
}

func (m *messageImpl) AsProtoreflectMessage() *dynamic.Message {
	msg, ok := m.message.(*dynamic.Message)
	if !ok {
		return &dynamic.Message{}
	}
	return msg
}
