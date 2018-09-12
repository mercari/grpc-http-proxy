package reflection

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/jhump/protoreflect/dynamic"
	_ "google.golang.org/grpc/test/grpc_testing"

	perrors "github.com/mercari/grpc-http-proxy/errors"
	"github.com/mercari/grpc-http-proxy/proxy/proxytest"
)

const messageName = "grpc.testing.Payload"

func TestNewReflector(t *testing.T) {
	r := NewReflector(&proxytest.FakeGrpcreflectClient{})
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
			serviceName:     proxytest.TestService,
			methodName:      proxytest.EmptyCall,
			message:         []byte("{}"),
			invocationIsNil: false,
			errorIsNil:      true,
		},
		{
			name:            "service not found",
			serviceName:     proxytest.NotFoundService,
			methodName:      proxytest.EmptyCall,
			message:         []byte("{}"),
			invocationIsNil: true,
			errorIsNil:      false,
		},
		{
			name:            "method not found",
			serviceName:     proxytest.TestService,
			methodName:      proxytest.NotFoundCall,
			message:         []byte("{}"),
			invocationIsNil: true,
			errorIsNil:      false,
		},
		{
			name:            "unmarshal failed",
			serviceName:     proxytest.TestService,
			methodName:      proxytest.EmptyCall,
			message:         []byte("{"),
			invocationIsNil: true,
			errorIsNil:      false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			fd := proxytest.NewFileDescriptor(t, proxytest.File)
			sd := ServiceDescriptorFromFileDescriptor(fd, proxytest.TestService)
			r := NewReflector(&proxytest.FakeGrpcreflectClient{sd.ServiceDescriptor})
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
		error       *perrors.ProxyError
	}{
		{
			name:        "found",
			serviceName: proxytest.TestService,
			descIsNil:   false,
			error:       nil,
		},
		{
			name:        "not found",
			serviceName: proxytest.NotFoundService,
			descIsNil:   true,
			error: &perrors.ProxyError{
				Code:    perrors.ServiceNotFound,
				Message: fmt.Sprintf("service %s was not found upstream", "not.found.NoService"),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			c := newReflectionClient(&proxytest.FakeGrpcreflectClient{})
			serviceDesc, err := c.resolveService(ctx, tc.serviceName)
			if got, want := serviceDesc == nil, tc.descIsNil; got != want {
				t.Fatalf("got %t, want %t", got, want)
			}
			{
				err, ok := err.(*perrors.ProxyError)
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
	cases := []struct {
		name       string
		methodName string
		descIsNil  bool
		error      *perrors.ProxyError
	}{
		{
			name:       "method found",
			methodName: proxytest.EmptyCall,
			descIsNil:  false,
			error:      nil,
		},
		{
			name:       "method not found",
			methodName: proxytest.NotFoundCall,
			descIsNil:  true,
			error: &perrors.ProxyError{
				Code:    perrors.MethodNotFound,
				Message: fmt.Sprintf("the method %s was not found", proxytest.NotFoundCall),
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			file := proxytest.NewFileDescriptor(t, proxytest.File)
			serviceDesc := ServiceDescriptorFromFileDescriptor(file, proxytest.TestService)
			if serviceDesc == nil {
				t.Fatalf("service descriptor is nil")
			}
			methodDesc, err := serviceDesc.FindMethodByName(tc.methodName)
			if got, want := methodDesc == nil, tc.descIsNil; got != want {
				t.Fatalf("got %t, want %t", got, want)
			}
			{
				err, ok := err.(*perrors.ProxyError)
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
	cases := []struct {
		name        string
		serviceName string
		methodName  string
		descIsNil   bool
	}{
		{
			name:        "input type found",
			serviceName: proxytest.TestService,
			methodName:  proxytest.UnaryCall,
			descIsNil:   false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			file := proxytest.NewFileDescriptor(t, proxytest.File)
			serviceDesc := ServiceDescriptorFromFileDescriptor(file, tc.serviceName)
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
	cases := []struct {
		name        string
		serviceName string
		methodName  string
		descIsNil   bool
	}{
		{
			name:        "output type found",
			serviceName: proxytest.TestService,
			methodName:  proxytest.EmptyCall,
			descIsNil:   false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			file := proxytest.NewFileDescriptor(t, proxytest.File)
			serviceDesc := ServiceDescriptorFromFileDescriptor(file, tc.serviceName)
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
	file := proxytest.NewFileDescriptor(t, proxytest.File)
	serviceDesc := ServiceDescriptorFromFileDescriptor(file, proxytest.TestService)
	if serviceDesc == nil {
		t.Fatal("service descriptor is nil")
	}
	methodDesc, err := serviceDesc.FindMethodByName(proxytest.EmptyCall)
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
	file := proxytest.NewFileDescriptor(t, proxytest.File)
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
			messageDesc := file.FindMessage(messageName)
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
	file := proxytest.NewFileDescriptor(t, proxytest.File)
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
			error: &perrors.ProxyError{
				Code:    perrors.MessageTypeMismatch,
				Message: "input JSON does not match messageImpl type",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			messageDesc := file.FindMessage(messageName)
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
