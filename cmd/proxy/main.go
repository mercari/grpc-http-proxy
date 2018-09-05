package main

import (
	"fmt"
	"net"
	"os"

	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/mercari/grpc-http-proxy/config"
	"github.com/mercari/grpc-http-proxy/http"
	"github.com/mercari/grpc-http-proxy/log"
	"github.com/mercari/grpc-http-proxy/source"
)

func main() {
	env, err := config.ReadFromEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to read environment variables: %s\n", err.Error())
		os.Exit(1)
	}
	logger, err := log.NewLogger(env.LogLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to create logger: %s\n", err)
		os.Exit(1)
	}

	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to create k8s config: %s\n", err)
		os.Exit(1)
	}
	k8sClient, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to create k8s client: %s\n", err)
		os.Exit(1)
	}
	d := source.NewService(k8sClient, "", logger)
	s := http.New(env.Token, d, logger)
	logger.Info("starting grpc-http-proxy",
		zap.String("log_level", env.LogLevel),
		zap.Int16("port", env.Port),
	)

	addr := fmt.Sprintf(":%d", env.Port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to listen HTTP port %s\n", err)
		os.Exit(1)
	}
	s.Serve(ln)
}
