package stub

import (
	"context"
	"reflect"
	"testing"

	"github.com/golang/protobuf/proto"
	"github.com/jhump/protoreflect/desc"
	"github.com/jhump/protoreflect/dynamic"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	_ "google.golang.org/grpc/test/grpc_testing"

	"github.com/mercari/grpc-http-proxy"
	"github.com/mercari/grpc-http-proxy/errors"
	"github.com/mercari/grpc-http-proxy/proxy/reflection"
)

type mockGrpcdynamicStub struct{}

func (m *mockGrpcdynamicStub) InvokeRpc(ctx context.Context, method *desc.MethodDescriptor, request proto.Message, opts ...grpc.CallOption) (proto.Message, error) {
	if method.GetName() == "UnaryCall" {
		return nil, status.Error(codes.Unimplemented, "unary unimplemented")
	}
	output := dynamic.NewMessage(method.GetOutputType())
	return output, nil
}

func TestNewStub(t *testing.T) {
	cc, err := grpc.Dial("localhost:5000", grpc.WithInsecure())
	if err != nil {
		t.Fatal(err.Error())
	}
	NewStub(cc)
}

func TestStub_InvokeRPC(t *testing.T) {
	cases := []struct {
		name           string
		methodName     string
		outputMsgIsNil bool
		error
	}{
		{
			name:           "success",
			methodName:     "EmptyCall",
			outputMsgIsNil: false,
			error:          nil,
		},
		{
			name:           "grpc error",
			methodName:     "UnaryCall",
			outputMsgIsNil: true,
			error: &errors.GRPCError{
				StatusCode: int(codes.Unimplemented),
				Message:    "unary unimplemented",
			},
		},
	}
	const fileName = "grpc_testing/test.proto"
	const target = "localhost:5000"
	const serviceName = "grpc.testing.TestService"
	fileDesc := newFileDescriptor(t, fileName)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			serviceDesc := reflection.ServiceDescriptorFromFileDescriptor(fileDesc, serviceName)
			if serviceDesc == nil {
				t.Fatal("service descriptor is nil")
			}
			methodDesc, err := serviceDesc.FindMethodByName(tc.methodName)
			if err != nil {
				t.Fatal(err.Error())
			}
			inputMsgDesc := methodDesc.GetInputType()
			inputMsg := inputMsgDesc.NewMessage()
			ctx := context.Background()

			stub := &stubImpl{
				stub: &mockGrpcdynamicStub{},
			}
			invocation := &reflection.MethodInvocation{
				MethodDescriptor: methodDesc,
				Message:          inputMsg,
			}
			outputMsg, err := stub.InvokeRPC(ctx, invocation, (*proxy.Metadata)(&map[string][]string{}))
			if err != nil {
				switch v := err.(type) {
				case *errors.Error:
					expected := tc.error.(*errors.Error)
					if got, want := v, expected; !reflect.DeepEqual(got, want) {
						t.Fatalf("got %v, want %v", got, want)
					}
				case *errors.GRPCError:
					expected := tc.error.(*errors.GRPCError)
					if got, want := v, expected; !reflect.DeepEqual(got, want) {
						t.Fatalf("got %#v, want %#v", got, want)
					}
				}
			}
			if got, want := outputMsg == nil, tc.outputMsgIsNil; got != want {
				t.Fatalf("got %t, want %t", got, want)
			}
		})
	}
}
