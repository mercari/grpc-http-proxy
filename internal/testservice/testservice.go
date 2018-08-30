package testservice

import (
	"context"
	"net"

	"google.golang.org/grpc/codes"
	grpc_reflection "google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"google.golang.org/grpc"
	pb "google.golang.org/grpc/test/grpc_testing"
)

type TestService struct{}

func (TestService) EmptyCall(context.Context, *pb.Empty) (*pb.Empty, error) {
	return &pb.Empty{}, nil
}

func (TestService) UnaryCall(ctx context.Context, req *pb.SimpleRequest) (*pb.SimpleResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unary unimplemented")
}

func (TestService) StreamingOutputCall(req *pb.StreamingOutputCallRequest, ss pb.TestService_StreamingOutputCallServer) error {
	return status.Error(codes.Unimplemented, "streaming unimplemented")
}

func (TestService) StreamingInputCall(ss pb.TestService_StreamingInputCallServer) error {
	return status.Error(codes.Unimplemented, "streaming unimplemented")
}

func (_ TestService) FullDuplexCall(ss pb.TestService_FullDuplexCallServer) error {
	return status.Error(codes.Unimplemented, "streaming unimplemented")
}

func (_ TestService) HalfDuplexCall(ss pb.TestService_HalfDuplexCallServer) error {
	return status.Error(codes.Unimplemented, "streaming unimplemented")
}

func StartTestService(stopCh chan struct{}) error {
	s := TestService{}
	lis, err := net.Listen("tcp", ":5000")
	if err != nil {
		return err
	}
	grpcServer := grpc.NewServer()
	pb.RegisterTestServiceServer(grpcServer, &s)
	grpc_reflection.Register(grpcServer)
	go func() {
		<-stopCh
		grpcServer.GracefulStop()
	}()
	if err := grpcServer.Serve(lis); err != nil {
		return err
	}
	return nil
}
