package reflection

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"github.com/pkg/errors"
	_ "google.golang.org/grpc/test/grpc_testing"

	perrors "github.com/mercari/grpc-http-proxy/errors"
)

const (
	testService     = "grpc.testing.TestService"
	notFoundService = "not.found.NoService"
	emptyCall       = "EmptyCall"
	notFoundCall    = "NotFoundCall"
	file            = "grpc_testing/test.proto"
)

type mockGrpcreflectClient struct {
	*desc.ServiceDescriptor
}

func (m *mockGrpcreflectClient) ResolveService(serviceName string) (*desc.ServiceDescriptor, error) {
	if serviceName != testService {
		return nil, errors.Errorf("service not found")
	}
	return m.ServiceDescriptor, nil
}

func TestNewReflector(t *testing.T) {
	r := NewReflector(&mockGrpcreflectClient{})
	if r == nil {
		t.Fatal("reflector should not be nil")
	}
}

func TestReflectorImpl_CreateInvocation(t *testing.T) {
	cases := []struct {
		name            string
		serviceName     string
		methodName      string
		message         []byte
		invocationIsNil bool
		errorIsNil      bool
	}{
		{
			name:            "found",
			serviceName:     testService,
			methodName:      emptyCall,
			message:         []byte("{}"),
			invocationIsNil: false,
			errorIsNil:      true,
		},
		{
			name:            "service not found",
			serviceName:     notFoundService,
			methodName:      emptyCall,
			message:         []byte("{}"),
			invocationIsNil: true,
			errorIsNil:      false,
		},
		{
			name:            "method not found",
			serviceName:     testService,
			methodName:      notFoundCall,
			message:         []byte("{}"),
			invocationIsNil: true,
			errorIsNil:      false,
		},
		{
			name:            "unmarshal failed",
			serviceName:     testService,
			methodName:      emptyCall,
			message:         []byte("{"),
			invocationIsNil: true,
			errorIsNil:      false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			fd := newFileDescriptor(t, file)
			sd := ServiceDescriptorFromFileDescriptor(fd, testService)
			r := NewReflector(&mockGrpcreflectClient{sd.ServiceDescriptor})
			i, err := r.CreateInvocation(ctx, tc.serviceName, tc.methodName, []byte(tc.message))
			if got, want := i == nil, tc.invocationIsNil; got != want {
				t.Fatalf("got %t, want %t", got, want)
			}
			if got, want := err == nil, tc.errorIsNil; got != want {
				t.Fatalf("got %v, want %v", got, want)
			}
		})
	}
}

func TestReflectionClient_ResolveService(t *testing.T) {
	cases := []struct {
		name        string
		serviceName string
		descIsNil   bool
		error       *perrors.Error
	}{
		{
			name:        "found",
			serviceName: "grpc.testing.TestService",
			descIsNil:   false,
			error:       nil,
		},
		{
			name:        "not found",
			serviceName: "not.found.NoService",
			descIsNil:   true,
			error: &perrors.Error{
				Code:    perrors.ServiceNotFound,
				Message: fmt.Sprintf("service %s was not found upstream", "not.found.NoService"),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			c := newReflectionClient(&mockGrpcreflectClient{})
			serviceDesc, err := c.resolveService(ctx, tc.serviceName)
			if got, want := serviceDesc == nil, tc.descIsNil; got != want {
				t.Fatalf("got %t, want %t", got, want)
			}
			{
				err, ok := err.(*perrors.Error)
				if !ok {
					err = nil
				}
				if got, want := err, tc.error; !reflect.DeepEqual(got, want) {
					t.Fatalf("got %v, want %v", got, want)
				}
			}
		})
	}
}

func TestServiceDescriptor_FindMethodByName(t *testing.T) {
	const serviceName = "grpc.testing.TestService"
	const file = "grpc_testing/test.proto"
	cases := []struct {
		name        string
		serviceName string
		methodName  string
		descIsNil   bool
		error       *perrors.Error
	}{
		{
			name:       "method found",
			methodName: "EmptyCall",
			descIsNil:  false,
			error:      nil,
		},
		{
			name:       "method not found",
			methodName: "ThisMethodDoesNotExist",
			descIsNil:  true,
			error: &perrors.Error{
				Code:    perrors.MethodNotFound,
				Message: fmt.Sprintf("the method %s was not found", "ThisMethodDoesNotExist"),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fileDesc := newFileDescriptor(t, file)
			serviceDesc := ServiceDescriptorFromFileDescriptor(fileDesc, serviceName)
			if serviceDesc == nil {
				t.Fatalf("service descriptor is nil")
			}
			methodDesc, err := serviceDesc.FindMethodByName(tc.methodName)
			if got, want := methodDesc == nil, tc.descIsNil; got != want {
				t.Fatalf("got %t, want %t", got, want)
			}
			{
				err, ok := err.(*perrors.Error)
				if !ok {
					err = nil
				}
				if got, want := err, tc.error; !reflect.DeepEqual(got, want) {
					t.Fatalf("got %v, want %v", got, want)
				}
			}
		})
	}
}

func TestServiceDescriptor_GetInputType(t *testing.T) {
	const serviceName = "grpc.testing.TestService"
	const file = "grpc_testing/test.proto"
	cases := []struct {
		name        string
		serviceName string
		methodName  string
		descIsNil   bool
	}{
		{
			name:        "input type found",
			serviceName: "TestService",
			methodName:  "EmptyCall",
			descIsNil:   false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fileDesc := newFileDescriptor(t, file)
			serviceDesc := ServiceDescriptorFromFileDescriptor(fileDesc, serviceName)
			methodDesc, err := serviceDesc.FindMethodByName(tc.methodName)
			if err != nil {
				t.Fatalf(err.Error())
			}
			inputMsgDesc := methodDesc.GetInputType()
			if got, want := inputMsgDesc == nil, tc.descIsNil; got != want {
				t.Fatalf("got %t, want %t", got, want)
			}
		})
	}
}

func TestServiceDescriptor_GetOutputType(t *testing.T) {
	const serviceName = "grpc.testing.TestService"
	const file = "grpc_testing/test.proto"
	cases := []struct {
		name        string
		serviceName string
		methodName  string
		descIsNil   bool
	}{
		{
			name:        "output type found",
			serviceName: "TestService",
			methodName:  "EmptyCall",
			descIsNil:   false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fileDesc := newFileDescriptor(t, file)
			serviceDesc := ServiceDescriptorFromFileDescriptor(fileDesc, serviceName)
			methodDesc, err := serviceDesc.FindMethodByName(tc.methodName)
			if err != nil {
				t.Fatalf(err.Error())
			}
			inputMsgDesc := methodDesc.GetOutputType()
			if got, want := inputMsgDesc == nil, tc.descIsNil; got != want {
				t.Fatalf("got %t, want %t", got, want)
			}
		})
	}
}

func TestMessageDescriptor_NewMessage(t *testing.T) {
	const serviceName = "grpc.testing.TestService"
	const methodName = "EmptyCall"
	const file = "grpc_testing/test.proto"
	fileDesc := newFileDescriptor(t, file)
	serviceDesc := ServiceDescriptorFromFileDescriptor(fileDesc, serviceName)
	if serviceDesc == nil {
		t.Fatal("service descriptor is nil")
	}
	methodDesc, err := serviceDesc.FindMethodByName(methodName)
	if err != nil {
		t.Fatalf(err.Error())
	}
	inputMsgDesc := methodDesc.GetInputType()
	inputMsg := inputMsgDesc.NewMessage()
	if got, want := inputMsg == nil, false; got != want {
		t.Fatalf("got %t, want %t", got, want)
	}
}

func TestMessage_MarshalJSON(t *testing.T) {
	const serviceName = "grpc.testing.TestService"
	const methodName = "EmptyCall"
	const file = "grpc_testing/test.proto"
	const messageName = "grpc.testing.Payload"
	fileDesc := newFileDescriptor(t, file)
	cases := []struct {
		name string
		json []byte
		error
	}{
		{
			name:  "success",
			json:  []byte("{\"body\":\"aGVsbG8=\"}"),
			error: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			messageDesc := fileDesc.FindMessage(messageName)
			if messageDesc == nil {
				t.Fatal("messageImpl descriptor is nil")
			}
			message := messageImpl{
				Message: dynamic.NewMessage(messageDesc),
			}
			message.Message.SetField(message.Message.FindFieldDescriptorByName("body"), []byte("hello"))
			j, err := message.MarshalJSON()
			if got, want := j, tc.json; !reflect.DeepEqual(got, want) {
				t.Fatalf("got %v, want %v", got, want)
			}
			if got, want := err, tc.error; !reflect.DeepEqual(got, want) {
				t.Fatalf("got %v, want %v", got, want)
			}
		})
	}
}

func TestMessage_UnmarshalJSON(t *testing.T) {
	const serviceName = "grpc.testing.TestService"
	const methodName = "EmptyCall"
	const file = "grpc_testing/test.proto"
	const messageName = "grpc.testing.Payload"
	fileDesc := newFileDescriptor(t, file)
	cases := []struct {
		name string
		json []byte
		error
	}{
		{
			name:  "success",
			json:  []byte("{\"body\":\"aGVsbG8=\"}"),
			error: nil,
		},
		{
			name: "type mismatch",
			json: []byte("{\"body\":\"hello!\""),
			error: &perrors.Error{
				Code:    perrors.MessageTypeMismatch,
				Message: "input JSON does not match messageImpl type",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			messageDesc := fileDesc.FindMessage(messageName)
			if messageDesc == nil {
				t.Fatal("messageImpl descriptor is nil")
			}
			message := messageImpl{
				Message: dynamic.NewMessage(messageDesc),
			}
			err := message.UnmarshalJSON(tc.json)

			expectedMessage := dynamic.NewMessage(messageDesc)
			expectedMessage.SetField(expectedMessage.FindFieldDescriptorByName("body"), []byte("hello!"))

			if got, want := err, tc.error; !reflect.DeepEqual(got, want) {
				t.Fatalf("got %v, want %v", got, want)
			}
		})
	}
}
