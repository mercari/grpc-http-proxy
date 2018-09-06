package proxy

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/jhump/protoreflect/dynamic"

	"github.com/mercari/grpc-http-proxy/errors"
	"github.com/mercari/grpc-http-proxy/internal/testservice"
)

func TestReflectionClient_ResolveService(t *testing.T) {
	cases := []struct {
		name        string
		serviceName string
		descIsNil   bool
		error       *errors.Error
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
			error: &errors.Error{
				Code:    errors.ServiceNotFound,
				Message: fmt.Sprintf("service %s was not found upstream", "not.found.NoService"),
			},
		},
	}
	stopCh := make(chan struct{})
	defer func() { stopCh <- struct{}{} }()
	go func() {
		t.Log("starting test service")
		err := testservice.StartTestService(stopCh)
		if err != nil {
			t.Fatal(err.Error())
		}
	}()
	time.Sleep(time.Second)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cc, err := newClientConn(context.Background(), parseURL(t, "localhost:5000"))
			if err != nil {
				t.Fatal(err.Error())
			}
			c := newReflectionClient(cc)
			serviceDesc, err := c.resolveService(context.Background(), tc.serviceName)
			if got, want := serviceDesc == nil, tc.descIsNil; got != want {
				t.Fatalf("got %t, want %t", got, want)
			}
			{
				err, ok := err.(*errors.Error)
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
		error       *errors.Error
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
			error: &errors.Error{
				Code:    errors.MethodNotFound,
				Message: fmt.Sprintf("the method %s was not found", "ThisMethodDoesNotExist"),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fileDesc := newFileDescriptor(t, file)
			serviceDesc := serviceDescriptorFromFileDescriptor(fileDesc, serviceName)
			if serviceDesc == nil {
				t.Fatalf("service descriptor is nil")
			}
			methodDesc, err := serviceDesc.findMethodByName(tc.methodName)
			if got, want := methodDesc == nil, tc.descIsNil; got != want {
				t.Fatalf("got %t, want %t", got, want)
			}
			{
				err, ok := err.(*errors.Error)
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
			serviceDesc := serviceDescriptorFromFileDescriptor(fileDesc, serviceName)
			methodDesc, err := serviceDesc.findMethodByName(tc.methodName)
			if err != nil {
				t.Fatalf(err.Error())
			}
			inputMsgDesc := methodDesc.getInputType()
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
			serviceDesc := serviceDescriptorFromFileDescriptor(fileDesc, serviceName)
			methodDesc, err := serviceDesc.findMethodByName(tc.methodName)
			if err != nil {
				t.Fatalf(err.Error())
			}
			inputMsgDesc := methodDesc.getOutputType()
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
	serviceDesc := serviceDescriptorFromFileDescriptor(fileDesc, serviceName)
	if serviceDesc == nil {
		t.Fatal("service descriptor is nil")
	}
	methodDesc, err := serviceDesc.findMethodByName(methodName)
	if err != nil {
		t.Fatalf(err.Error())
	}
	inputMsgDesc := methodDesc.getInputType()
	inputMsg := inputMsgDesc.newMessage()
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
				t.Fatal("message descriptor is nil")
			}
			message := message{
				desc: dynamic.NewMessage(messageDesc),
			}
			message.desc.SetField(message.desc.FindFieldDescriptorByName("body"), []byte("hello"))
			j, err := message.marshalJSON()
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
			error: &errors.Error{
				Code:    errors.MessageTypeMismatch,
				Message: "input JSON does not match message type",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			messageDesc := fileDesc.FindMessage(messageName)
			if messageDesc == nil {
				t.Fatal("message descriptor is nil")
			}
			message := message{
				desc: dynamic.NewMessage(messageDesc),
			}
			err := message.unmarshalJSON(tc.json)

			expectedMessage := dynamic.NewMessage(messageDesc)
			expectedMessage.SetField(expectedMessage.FindFieldDescriptorByName("body"), []byte("hello!"))

			if got, want := err, tc.error; !reflect.DeepEqual(got, want) {
				t.Fatalf("got %v, want %v", got, want)
			}
		})
	}
}
