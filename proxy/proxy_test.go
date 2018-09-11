package proxy

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"

	"github.com/mercari/grpc-http-proxy"
	"github.com/mercari/grpc-http-proxy/proxy/reflection"
	"github.com/mercari/grpc-http-proxy/proxy/reflection/mock"
	"github.com/mercari/grpc-http-proxy/proxy/stub/mock"
)

const testService = "grpc.testing.TestService"
const method = "EmptyCall"

var testError = errors.Errorf("an error")

func TestNewProxy(t *testing.T) {
	p := NewProxy()
	if p == nil {
		t.Fatalf("proxy was nil")
	}
}

func TestProxy_Connect(t *testing.T) {
	p := NewProxy()
	p.Connect(context.Background(), parseURL(t, "localhost:5000"))
}

func TestProxy_Call(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		p := NewProxy()
		ctx := context.Background()
		md := make(proxy.Metadata)

		ctrl, ctx := gomock.WithContext(context.Background(), t)
		defer ctrl.Finish()
		foo := mock_reflection.NewMockReflector(ctrl)
		p.reflector = foo

		mockInputMsg := mock_reflection.NewMockMessage(ctrl)
		mockOutputMsg := mock_reflection.NewMockMessage(ctrl)
		mockOutputMsg.EXPECT().MarshalJSON().
			Return([]byte("message body"), error(nil))

		mockStub := mock_stub.NewMockStub(ctrl)
		p.stub = mockStub
		invocation := &reflection.MethodInvocation{
			MethodDescriptor: &reflection.MethodDescriptor{},
			Message:          mockInputMsg,
		}
		mockStub.EXPECT().InvokeRPC(ctx, invocation, &md).
			Return(mockOutputMsg, error(nil))

		foo.EXPECT().CreateInvocation(ctx, testService, method, []byte("message body")).
			Return(invocation, nil)

		p.Call(ctx, testService, method, []byte("message body"), &md)
	})

	t.Run("service not found (upstream)", func(t *testing.T) {
		p := NewProxy()
		ctx := context.Background()
		md := make(proxy.Metadata)

		ctrl, ctx := gomock.WithContext(context.Background(), t)
		defer ctrl.Finish()
		foo := mock_reflection.NewMockReflector(ctrl)
		p.reflector = foo

		mockStub := mock_stub.NewMockStub(ctrl)
		p.stub = mockStub

		foo.EXPECT().CreateInvocation(ctx, testService, method, []byte("message body")).
			Return(nil, testError)

		p.Call(ctx, testService, method, []byte("message body"), &md)
	})
}
