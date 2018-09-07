package stub

import (
	"context"
	"reflect"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	_ "google.golang.org/grpc/test/grpc_testing"

	"github.com/mercari/grpc-http-proxy"
	"github.com/mercari/grpc-http-proxy/errors"
	"github.com/mercari/grpc-http-proxy/internal/testservice"
	"github.com/mercari/grpc-http-proxy/proxy/reflection"
)

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
			conn, err := grpc.DialContext(ctx, target, grpc.WithInsecure())
			if err != nil {
				t.Fatal(err.Error())
			}
			stub := NewStub(conn)
			outputMsg, err := stub.InvokeRPC(ctx, methodDesc, inputMsg, (*proxy.Metadata)(&map[string][]string{}))
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
						t.Fatalf("got %v, want %v", got, want)
					}
				}
			}
			if got, want := outputMsg == nil, tc.outputMsgIsNil; got != want {
				t.Fatalf("got %t, want %t", got, want)
			}
		})
	}
}
