package backend

import (
	"context"
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/jhump/protoreflect/grpcreflect"
	"google.golang.org/grpc"
	rpb "google.golang.org/grpc/reflection/grpc_reflection_v1alpha"

	"github.com/mercari/grpc-http-proxy/errors"
)

type ReflectionClient struct {
	cc *grpc.ClientConn
}

func NewReflectionClient(c *ClientConn) *ReflectionClient {
	return &ReflectionClient{
		cc: c.cc,
	}
}

// ResolveService gets the service descriptor from the service the client is connected to
func (c *ReflectionClient) ResolveService(ctx context.Context, serviceName string) (*ServiceDescriptor, error) {
	reflectClient := grpcreflect.NewClient(ctx, rpb.NewServerReflectionClient(c.cc))
	d, err := reflectClient.ResolveService(serviceName)
	if err != nil {
		return nil, &errors.Error{
			Code:    errors.ServiceNotFound,
			Message: fmt.Sprintf("service %s was not found", serviceName),
		}
	}
	return &ServiceDescriptor{
		desc: d,
	}, nil
}

// ServiceDescriptor represents a service type
type ServiceDescriptor struct {
	desc *desc.ServiceDescriptor
}

func serviceDescriptorFromFileDescriptor(fd *desc.FileDescriptor, service string) *ServiceDescriptor {
	d := fd.FindService(service)
	if d == nil {
		return nil
	}
	return &ServiceDescriptor{
		desc: d,
	}
}

// FindMethodByName finds the method descriptor with the name
func (s *ServiceDescriptor) FindMethodByName(name string) (*MethodDescriptor, error) {
	d := s.desc.FindMethodByName(name)
	if d == nil {
		return nil, &errors.Error{
			Code:    errors.MethodNotFound,
			Message: fmt.Sprintf("the method %s was not found", name),
		}
	}
	return &MethodDescriptor{
		desc: d,
	}, nil
}

// MethodDescriptor represents a method type
type MethodDescriptor struct {
	desc *desc.MethodDescriptor
}

// GetInputType gets the message descriptor for the input type for the method
func (m *MethodDescriptor) GetInputType() *MessageDescriptor {
	return &MessageDescriptor{
		desc: m.desc.GetInputType(),
	}
}

// GetInputType gets the message descriptor for the output type for the method
func (m *MethodDescriptor) GetOutputType() *MessageDescriptor {
	return &MessageDescriptor{
		desc: m.desc.GetOutputType(),
	}
}

// MessageDescriptor represents a message type
type MessageDescriptor struct {
	desc *desc.MessageDescriptor
}

// NewMessage creates a new Message from the receiver
func (m *MessageDescriptor) NewMessage() *Message {
	return &Message{
		desc: dynamic.NewMessage(m.desc),
	}
}

// Message is an message value
type Message struct {
	desc *dynamic.Message
}

func (m *Message) MarshalJSON() ([]byte, error) {
	b, err := m.desc.MarshalJSON()
	if err != nil {
		return nil, &errors.Error{
			Code:    errors.Unknown,
			Message: "could not marshal backend response into JSON",
		}
	}
	return b, nil
}

func (m *Message) UnmarshalJSON(b []byte) error {
	err := m.desc.UnmarshalJSON(b)
	if err != nil {
		return &errors.Error{
			Code:    errors.MessageTypeMismatch,
			Message: "input JSON does not match message type",
		}
	}
	return nil
}

func (m *Message) convertFrom(target proto.Message) error {
	return m.desc.ConvertFrom(target)
}
