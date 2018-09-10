package proxy

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"

	"github.com/mercari/grpc-http-proxy"
	"github.com/mercari/grpc-http-proxy/log"
	"github.com/mercari/grpc-http-proxy/proxy/reflection/mock"
	"github.com/mercari/grpc-http-proxy/proxy/stub/mock"
)

const testService = "grpc.testing.TestService"
const method = "EmptyCall"

var testError = errors.Errorf("an error")

func TestNewProxy(t *testing.T) {
	p := NewProxy(log.NewDiscard())
	if p == nil {
		t.Fatalf("proxy was nil")
	}
}

func TestProxy_Connect(t *testing.T) {
	p := NewProxy(log.NewDiscard())
	p.Connect(context.Background(), parseURL(t, "localhost:5000"))
}

func TestProxy_Call(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		p := NewProxy(log.NewDiscard())
		ctx := context.Background()
		md := make(proxy.Metadata)

		ctrl, ctx := gomock.WithContext(context.Background(), t)
		defer ctrl.Finish()
		rc := mock_reflection.NewMockReflectionClient(ctrl)
		mockSvcDesc := mock_reflection.NewMockServiceDescriptor(ctrl)
		p.reflectionClient = rc
		rc.EXPECT().ResolveService(ctx, testService).
			Return(mockSvcDesc, nil)

		mockMethodDesc := mock_reflection.NewMockMethodDescriptor(ctrl)
		mockSvcDesc.EXPECT().FindMethodByName(method).
			Return(mockMethodDesc, error(nil))

		mockInputMsgDesc := mock_reflection.NewMockMessageDescriptor(ctrl)
		mockMethodDesc.EXPECT().GetInputType().
			Return(mockInputMsgDesc)
		mockInputMsg := mock_reflection.NewMockMessage(ctrl)
		mockOutputMsg := mock_reflection.NewMockMessage(ctrl)
		mockInputMsgDesc.EXPECT().NewMessage().
			Return(mockInputMsg)

		mockInputMsg.EXPECT().UnmarshalJSON([]byte("message body")).
			Return(error(nil))

		mockOutputMsg.EXPECT().MarshalJSON().
			Return([]byte("message body"), error(nil))

		mockStub := mock_stub.NewMockStub(ctrl)
		p.stub = mockStub
		mockStub.EXPECT().InvokeRPC(ctx, mockMethodDesc, mockInputMsg, &md).
			Return(mockOutputMsg, error(nil))

		p.Call(ctx, testService, method, []byte("message body"), &md)
	})

	t.Run("service not found (upstream)", func(t *testing.T) {
		p := NewProxy(log.NewDiscard())
		ctx := context.Background()
		md := make(proxy.Metadata)

		ctrl, ctx := gomock.WithContext(context.Background(), t)
		defer ctrl.Finish()
		rc := mock_reflection.NewMockReflectionClient(ctrl)
		p.reflectionClient = rc
		rc.EXPECT().ResolveService(ctx, testService).
			Return(nil, testError)

		p.Call(ctx, testService, method, []byte("message body"), &md)
	})

	t.Run("method not found", func(t *testing.T) {
		p := NewProxy(log.NewDiscard())
		ctx := context.Background()
		md := make(proxy.Metadata)

		ctrl, ctx := gomock.WithContext(context.Background(), t)
		defer ctrl.Finish()
		rc := mock_reflection.NewMockReflectionClient(ctrl)
		mockSvcDesc := mock_reflection.NewMockServiceDescriptor(ctrl)
		p.reflectionClient = rc
		rc.EXPECT().ResolveService(ctx, testService).
			Return(mockSvcDesc, nil)

		mockSvcDesc.EXPECT().FindMethodByName(method).
			Return(nil, testError)

		p.Call(ctx, testService, method, []byte("message body"), &md)
	})

	t.Run("message type mismatch between JSON and proto", func(t *testing.T) {
		p := NewProxy(log.NewDiscard())
		ctx := context.Background()
		md := make(proxy.Metadata)

		ctrl, ctx := gomock.WithContext(context.Background(), t)
		defer ctrl.Finish()
		rc := mock_reflection.NewMockReflectionClient(ctrl)
		mockSvcDesc := mock_reflection.NewMockServiceDescriptor(ctrl)
		p.reflectionClient = rc
		rc.EXPECT().ResolveService(ctx, testService).
			Return(mockSvcDesc, nil)

		mockMethodDesc := mock_reflection.NewMockMethodDescriptor(ctrl)
		mockSvcDesc.EXPECT().FindMethodByName(method).
			Return(mockMethodDesc, error(nil))

		mockInputMsgDesc := mock_reflection.NewMockMessageDescriptor(ctrl)
		mockMethodDesc.EXPECT().GetInputType().
			Return(mockInputMsgDesc)
		mockInputMsg := mock_reflection.NewMockMessage(ctrl)
		mockInputMsgDesc.EXPECT().NewMessage().
			Return(mockInputMsg)

		mockInputMsg.EXPECT().UnmarshalJSON([]byte("message body")).
			Return(testError)

		p.Call(ctx, testService, method, []byte("message body"), &md)
	})

	t.Run("invokeRPC returned error", func(t *testing.T) {
		p := NewProxy(log.NewDiscard())
		ctx := context.Background()
		md := make(proxy.Metadata)

		ctrl, ctx := gomock.WithContext(context.Background(), t)
		defer ctrl.Finish()
		rc := mock_reflection.NewMockReflectionClient(ctrl)
		mockSvcDesc := mock_reflection.NewMockServiceDescriptor(ctrl)
		p.reflectionClient = rc
		rc.EXPECT().ResolveService(ctx, testService).
			Return(mockSvcDesc, nil)

		mockMethodDesc := mock_reflection.NewMockMethodDescriptor(ctrl)
		mockSvcDesc.EXPECT().FindMethodByName(method).
			Return(mockMethodDesc, error(nil))

		mockInputMsgDesc := mock_reflection.NewMockMessageDescriptor(ctrl)
		mockMethodDesc.EXPECT().GetInputType().
			Return(mockInputMsgDesc)
		mockInputMsg := mock_reflection.NewMockMessage(ctrl)
		mockInputMsgDesc.EXPECT().NewMessage().
			Return(mockInputMsg)

		mockInputMsg.EXPECT().UnmarshalJSON([]byte("message body")).
			Return(error(nil))

		mockStub := mock_stub.NewMockStub(ctrl)
		p.stub = mockStub
		mockStub.EXPECT().InvokeRPC(ctx, mockMethodDesc, mockInputMsg, &md).
			Return(nil, testError)

		p.Call(ctx, testService, method, []byte("message body"), &md)
	})

	t.Run("marshaling failed", func(t *testing.T) {
		p := NewProxy(log.NewDiscard())
		ctx := context.Background()
		md := make(proxy.Metadata)

		ctrl, ctx := gomock.WithContext(context.Background(), t)
		defer ctrl.Finish()
		rc := mock_reflection.NewMockReflectionClient(ctrl)
		mockSvcDesc := mock_reflection.NewMockServiceDescriptor(ctrl)
		p.reflectionClient = rc
		rc.EXPECT().ResolveService(ctx, testService).
			Return(mockSvcDesc, nil)

		mockMethodDesc := mock_reflection.NewMockMethodDescriptor(ctrl)
		mockSvcDesc.EXPECT().FindMethodByName(method).
			Return(mockMethodDesc, error(nil))

		mockInputMsgDesc := mock_reflection.NewMockMessageDescriptor(ctrl)
		mockMethodDesc.EXPECT().GetInputType().
			Return(mockInputMsgDesc)
		mockInputMsg := mock_reflection.NewMockMessage(ctrl)
		mockOutputMsg := mock_reflection.NewMockMessage(ctrl)
		mockInputMsgDesc.EXPECT().NewMessage().
			Return(mockInputMsg)

		mockInputMsg.EXPECT().UnmarshalJSON([]byte("message body")).
			Return(error(nil))

		mockOutputMsg.EXPECT().MarshalJSON().
			Return(nil, testError)

		mockStub := mock_stub.NewMockStub(ctrl)
		p.stub = mockStub
		mockStub.EXPECT().InvokeRPC(ctx, mockMethodDesc, mockInputMsg, &md).
			Return(mockOutputMsg, error(nil))

		p.Call(ctx, testService, method, []byte("message body"), &md)
	})
}
