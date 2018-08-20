package main

import (
	"fmt"
	"net"
	"os"

	"github.com/mercari/grpc-http-proxy/http"
	"github.com/mercari/grpc-http-proxy/log"
)

func main() {
	logger, err := log.NewLogger()
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to create logger: %s\n", err)
		os.Exit(1)
	}
	s := http.New("foo", logger)
	logger.Info("starting grpc-http-proxy")

	port := 3000
	addr := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to listen HTTP port %s\n", err)
		os.Exit(1)
	}
	s.Serve(ln)
}
