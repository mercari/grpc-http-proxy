package main

import (
	"fmt"
	"net"
	"os"

	"go.uber.org/zap"

	"github.com/mercari/grpc-http-proxy/config"
	"github.com/mercari/grpc-http-proxy/http"
	"github.com/mercari/grpc-http-proxy/log"
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

	s := http.New(env.Token, logger)
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
