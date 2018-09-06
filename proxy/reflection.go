package proxy

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

type reflectionClient struct {
	cc *grpc.ClientConn
}

func newReflectionClient(c *clientConn) *reflectionClient {
	return &reflectionClient{
		cc: c.cc,
	}
}

// resolveService gets the service descriptor from the service the client is connected to
func (c *reflectionClient) resolveService(ctx context.Context, serviceName string) (*serviceDescriptor, error) {
	reflectClient := grpcreflect.NewClient(ctx, rpb.NewServerReflectionClient(c.cc))
	d, err := reflectClient.ResolveService(serviceName)
	if err != nil {
		return nil, &errors.Error{
			Code:    errors.ServiceNotFound,
			Message: fmt.Sprintf("service %s was not found upstream", serviceName),
		}
	}
	return &serviceDescriptor{
		desc: d,
	}, nil
}

// serviceDescriptor represents a service type
type serviceDescriptor struct {
	desc *desc.ServiceDescriptor
}

// findMethodByName finds the method descriptor with the name
func (s *serviceDescriptor) findMethodByName(name string) (*methodDescriptor, error) {
	d := s.desc.FindMethodByName(name)
	if d == nil {
		return nil, &errors.Error{
			Code:    errors.MethodNotFound,
			Message: fmt.Sprintf("the method %s was not found", name),
		}
	}
	return &methodDescriptor{
		desc: d,
	}, nil
}

// methodDescriptor represents a method type
type methodDescriptor struct {
	desc *desc.MethodDescriptor
}

// getInputType gets the message descriptor for the input type for the method
func (m *methodDescriptor) getInputType() *messageDescriptor {
	return &messageDescriptor{
		desc: m.desc.GetInputType(),
	}
}

// getInputType gets the message descriptor for the output type for the method
func (m *methodDescriptor) getOutputType() *messageDescriptor {
	return &messageDescriptor{
		desc: m.desc.GetOutputType(),
	}
}

// messageDescriptor represents a message type
type messageDescriptor struct {
	desc *desc.MessageDescriptor
}

// newMessage creates a new message from the receiver
func (m *messageDescriptor) newMessage() *message {
	return &message{
		desc: dynamic.NewMessage(m.desc),
	}
}

// message is an message value
type message struct {
	desc *dynamic.Message
}

func (m *message) marshalJSON() ([]byte, error) {
	b, err := m.desc.MarshalJSON()
	if err != nil {
		return nil, &errors.Error{
			Code:    errors.Unknown,
			Message: "could not marshal backend response into JSON",
		}
	}
	return b, nil
}

func (m *message) unmarshalJSON(b []byte) error {
	err := m.desc.UnmarshalJSON(b)
	if err != nil {
		return &errors.Error{
			Code:    errors.MessageTypeMismatch,
			Message: "input JSON does not match message type",
		}
	}
	return nil
}

func (m *message) convertFrom(target proto.Message) error {
	return m.desc.ConvertFrom(target)
}
