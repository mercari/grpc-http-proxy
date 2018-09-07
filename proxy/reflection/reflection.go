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

type reflectionClientImpl struct {
	grpcdynamicClient
}

type grpcdynamicClient interface {
	ResolveService(serviceName string) (*desc.ServiceDescriptor, error)
}

func NewReflectionClient(rc grpcdynamicClient) *reflectionClientImpl {
	return &reflectionClientImpl{
		grpcdynamicClient: rc,
	}
}

// ResolveService gets the service descriptor from the service the client is connected to
func (c *reflectionClientImpl) ResolveService(ctx context.Context,
	serviceName string) (ServiceDescriptor, error) {
	d, err := c.grpcdynamicClient.ResolveService(serviceName)
	if err != nil {
		return nil, &errors.Error{
			Code:    errors.ServiceNotFound,
			Message: fmt.Sprintf("service %s was not found upstream", serviceName),
		}
	}
	return &ServiceDescriptorImpl{
		desc: d,
	}, nil
}

// ServiceDescriptorImpl represents a service type
type ServiceDescriptorImpl struct {
	desc serviceDescriptor
}

// serviceDescriptor is the interface for message.ServiceDescriptorImpl
type serviceDescriptor interface {
	FindMethodByName(name string) *desc.MethodDescriptor
}

func ServiceDescriptorFromFileDescriptor(fd *desc.FileDescriptor, service string) *ServiceDescriptorImpl {
	d := fd.FindService(service)
	if d == nil {
		return nil
	}
	return &ServiceDescriptorImpl{
		desc: d,
	}
}

// FindMethodByName finds the method descriptor with the name
func (s *ServiceDescriptorImpl) FindMethodByName(name string) (MethodDescriptor, error) {
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

// methodDescriptorImpl represents a method type
type methodDescriptorImpl struct {
	desc methodDescriptor
}

// methodDescriptor is an interface for message.methodDescriptorImpl
type methodDescriptor interface {
	GetInputType() *desc.MessageDescriptor
	GetOutputType() *desc.MessageDescriptor
}

// GetInputType gets the MessageImpl descriptor for the input type for the method
func (m *methodDescriptorImpl) GetInputType() MessageDescriptor {
	return &MessageDescriptorImpl{
		desc: m.desc.GetInputType(),
	}
}

// GetOutputType gets the MessageImpl descriptor for the output type for the method
func (m *methodDescriptorImpl) GetOutputType() MessageDescriptor {
	return &MessageDescriptorImpl{
		desc: m.desc.GetOutputType(),
	}
}

func (s *methodDescriptorImpl) AsProtoreflectDescriptor() *desc.MethodDescriptor {
	d, ok := s.desc.(*desc.MethodDescriptor)
	if !ok {
		return &desc.MethodDescriptor{}
	}
	return d
}

// MessageDescriptorImpl represents a MessageImpl type
type MessageDescriptorImpl struct {
	desc *desc.MessageDescriptor
}

// NewMessage creates a new MessageImpl from the receiver
func (m *MessageDescriptorImpl) NewMessage() Message {
	return &MessageImpl{
		message: dynamic.NewMessage(m.desc),
	}
}

// MessageImpl is an MessageImpl value
type MessageImpl struct {
	message
}

// message is an interface for dynamic.MessageImpl
type message interface {
	MarshalJSON() ([]byte, error)
	UnmarshalJSON([]byte) error
	ConvertFrom(target proto.Message) error
	SetField(fd *desc.FieldDescriptor, val interface{})
	FindFieldDescriptorByName(name string) *desc.FieldDescriptor
}

func (m *MessageImpl) MarshalJSON() ([]byte, error) {
	b, err := m.message.MarshalJSON()
	if err != nil {
		return nil, &errors.Error{
			Code:    errors.Unknown,
			Message: "could not marshal backend response into JSON",
		}
	}
	return b, nil
}

func (m *MessageImpl) UnmarshalJSON(b []byte) error {
	err := m.message.UnmarshalJSON(b)
	if err != nil {
		return &errors.Error{
			Code:    errors.MessageTypeMismatch,
			Message: "input JSON does not match MessageImpl type",
		}
	}
	return nil
}

func (m *MessageImpl) ConvertFrom(target proto.Message) error {
	return m.message.ConvertFrom(target)
}

func (m *MessageImpl) AsProtoreflectMessage() *dynamic.Message {
	msg, ok := m.message.(*dynamic.Message)
	if !ok {
		return &dynamic.Message{}
	}
	return msg
}

// ReflectionClient performs reflection to obtain descriptors
type ReflectionClient interface {
	ResolveService(ctx context.Context, serviceName string) (ServiceDescriptor, error)
}

type ServiceDescriptor interface {
	FindMethodByName(name string) (MethodDescriptor, error)
}

type MethodDescriptor interface {
	GetInputType() MessageDescriptor
	GetOutputType() MessageDescriptor
	AsProtoreflectDescriptor() *desc.MethodDescriptor
}

type MessageDescriptor interface {
	NewMessage() Message
}

type Message interface {
	MarshalJSON() ([]byte, error)
	UnmarshalJSON(b []byte) error
	ConvertFrom(target proto.Message) error
	AsProtoreflectMessage() *dynamic.Message
}
