package proxy

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"

	"github.com/mercari/grpc-http-proxy"
	"github.com/mercari/grpc-http-proxy/internal/testservice"
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
	cases := []struct {
		name string
		err  error
	}{
		{
			name: "no preexisting error",
			err:  nil,
		},
		{
			name: "preexisting error",
			err:  testError,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := NewProxy(log.NewDiscard())
			p.err = tc.err
			p.Connect(context.Background(), parseURL(t, "localhost:5000"))
			if p.err != nil {
				t.Log(p.err.Error())
			}
		})
	}
}

func TestProxy_ResolveService(t *testing.T) {
	t.Run("should work", func(t *testing.T) {
		ctrl, ctx := gomock.WithContext(context.Background(), t)
		defer ctrl.Finish()
		p := NewProxy(log.NewDiscard())
		rc := mock_reflection.NewMockReflectionClient(ctrl)
		mockSvcDesc := mock_reflection.NewMockServiceDescriptor(ctrl)
		p.reflectionClient = rc
		rc.EXPECT().ResolveService(ctx, testService).
			Return(mockSvcDesc, nil)

		serviceDescriptor := p.resolveService(ctx, testService)
		if serviceDescriptor == nil {
			t.Error("service descriptor is nil")
		}
	})

	t.Run("already failed", func(t *testing.T) {
		ctx := context.Background()
		p := NewProxy(log.NewDiscard())
		p.err = testError

		serviceDescriptor := p.resolveService(ctx, testService)
		if serviceDescriptor != nil {
			t.Fatal("service descriptor should be nil")
		}
	})
}

func TestProxy_LoadDescriptors(t *testing.T) {
	t.Run("should work", func(t *testing.T) {
		ctrl, ctx := gomock.WithContext(context.Background(), t)
		defer ctrl.Finish()
		p := NewProxy(log.NewDiscard())
		rc := mock_reflection.NewMockReflectionClient(ctrl)
		mockSvcDesc := mock_reflection.NewMockServiceDescriptor(ctrl)
		p.reflectionClient = rc
		rc.EXPECT().ResolveService(ctx, testService).
			Return(mockSvcDesc, nil)
		mockMethodDesc := mock_reflection.NewMockMethodDescriptor(ctrl)
		mockSvcDesc.EXPECT().FindMethodByName(method).
			Return(mockMethodDesc, error(nil))

		p.loadDescriptors(ctx, testService, method)
		if p.methodDescriptor == nil {
			t.Error("method descriptor is nil")
		}
	})

	t.Run("already failed", func(t *testing.T) {
		ctx := context.Background()
		p := NewProxy(log.NewDiscard())
		p.err = testError

		p.loadDescriptors(ctx, testService, method)
		if p.methodDescriptor != nil {
			t.Fatal("method descriptor should be nil")
		}
	})
}

func TestProxy_CreateMessages(t *testing.T) {
	t.Run("should work", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		p := NewProxy(log.NewDiscard())
		mockMethodDesc := mock_reflection.NewMockMethodDescriptor(ctrl)
		p.methodDescriptor = mockMethodDesc
		mockInputMsgDesc := mock_reflection.NewMockMessageDescriptor(ctrl)
		mockOutputMsgDesc := mock_reflection.NewMockMessageDescriptor(ctrl)
		mockMethodDesc.EXPECT().GetInputType().
			Return(mockInputMsgDesc)
		mockMethodDesc.EXPECT().GetOutputType().
			Return(mockOutputMsgDesc)
		mockInputMsg := mock_reflection.NewMockMessage(ctrl)
		mockOutputMsg := mock_reflection.NewMockMessage(ctrl)
		mockInputMsgDesc.EXPECT().NewMessage().
			Return(mockInputMsg)
		mockOutputMsgDesc.EXPECT().NewMessage().
			Return(mockOutputMsg)

		p.createMessages()
		if p.InputMessage == nil {
			t.Error("input message is nil")
		}
		if p.OutputMessage == nil {
			t.Error("output message is nil")
		}
	})

	t.Run("already failed", func(t *testing.T) {
		p := NewProxy(log.NewDiscard())
		p.err = testError

		p.createMessages()
		if p.methodDescriptor != nil {
			t.Fatal("service descriptor should be nil")
		}
	})
}

func TestProxy_UnmarshalInputMessage(t *testing.T) {
	t.Run("should work", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		p := NewProxy(log.NewDiscard())
		mockInputMsg := mock_reflection.NewMockMessage(ctrl)
		p.InputMessage = mockInputMsg
		mockInputMsg.EXPECT().UnmarshalJSON([]byte("message body")).
			Return(error(nil))

		p.unmarshalInputMessage([]byte("message body"))
	})

	t.Run("already failed", func(t *testing.T) {
		p := NewProxy(log.NewDiscard())
		p.err = testError

		p.unmarshalInputMessage([]byte("foo"))
		if p.methodDescriptor != nil {
			t.Fatal("service descriptor should be nil")
		}
	})
}

func TestProxy_MarshalOutputMessage(t *testing.T) {
	t.Run("should work", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		p := NewProxy(log.NewDiscard())
		mockOutputMsg := mock_reflection.NewMockMessage(ctrl)
		p.OutputMessage = mockOutputMsg
		mockOutputMsg.EXPECT().MarshalJSON().
			Return([]byte("message body"), error(nil))

		p.marshalOutputMessage()
	})

	t.Run("already failed", func(t *testing.T) {
		p := NewProxy(log.NewDiscard())
		p.err = testError

		p.marshalOutputMessage()
		if p.methodDescriptor != nil {
			t.Fatal("service descriptor should be nil")
		}
	})
}

func TestProxy_InvokeRPC(t *testing.T) {
	t.Run("should work", func(t *testing.T) {
		ctrl, ctx := gomock.WithContext(context.Background(), t)
		defer ctrl.Finish()
		p := NewProxy(log.NewDiscard())
		mockMethodDesc := mock_reflection.NewMockMethodDescriptor(ctrl)
		p.methodDescriptor = mockMethodDesc
		mockInputMsg := mock_reflection.NewMockMessage(ctrl)
		p.InputMessage = mockInputMsg
		mockOutputMsg := mock_reflection.NewMockMessage(ctrl)
		md := make(proxy.Metadata)
		mockStub := mock_stub.NewMockStub(ctrl)
		p.stub = mockStub
		mockStub.EXPECT().InvokeRPC(ctx, p.methodDescriptor, p.InputMessage, &md).
			Return(mockOutputMsg, error(nil))

		p.invokeRPC(ctx, &md)
	})

	t.Run("already failed", func(t *testing.T) {
		p := NewProxy(log.NewDiscard())
		p.err = testError

		md := make(proxy.Metadata)
		p.invokeRPC(context.Background(), &md)
		if p.methodDescriptor != nil {
			t.Fatal("service descriptor should be nil")
		}
	})
}

func TestProxy_Call(t *testing.T) {
	const target = "localhost:5000"

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
	p := NewProxy(log.NewDiscard())
	ctx := context.Background()
	md := make(proxy.Metadata)

	p.Connect(ctx, parseURL(t, target))
	response, err := p.Call(ctx, testService, method, []byte("{}"), &md)
	if response == nil {
		t.Fatal("response was nil")
	}
	if err != nil {
		t.Fatal(err.Error())
	}
}
