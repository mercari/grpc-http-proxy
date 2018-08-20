package main

import (
	"fmt"
	"net"
	"os"

	"github.com/mercari/grpc-http-proxy/http"
)

func main() {
	s := http.New("foo")

	port := 3000
	addr := fmt.Sprintf(":%d", port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to listen HTTP port %s\n", err)
		os.Exit(1)
	}
	s.Serve(ln)
}
