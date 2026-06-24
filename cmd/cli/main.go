// Command cli starts the HTTP/HTTPS web proxy server.
//
// Example:
//
//	go run ./cmd/cli -port 8888
//
// Then configure your browser or application to use http://localhost:8888 as
// its HTTP and HTTPS proxy.
package main

import (
	"flag"
	"log"
	"os"

	"github.com/vivianludrick/proxy-server/internal/proxy"
)

func main() {
	port := flag.String("port", "8888", "port for the proxy to listen on")
	host := flag.String("host", "", "host/interface to bind to (empty = all interfaces)")
	flag.Parse()

	logger := log.New(os.Stdout, "[proxy] ", log.LstdFlags)

	addr := *host + ":" + *port
	server := proxy.NewServer(addr, logger)

	if err := server.ListenAndServe(); err != nil {
		logger.Fatalf("server stopped: %v", err)
	}
}
