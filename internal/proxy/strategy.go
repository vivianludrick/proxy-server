package proxy

import (
	"bufio"
	"net"
)

// Handler is the strategy interface for serving a parsed proxy request. Each
// kind of request (plain HTTP vs. HTTPS tunnelling) gets its own implementation,
// and the server just calls Handle without caring which one it picked.
type Handler interface {
	// Handle serves the request. client is the connection back to the caller,
	// reader is the buffered reader sitting on top of client (it may already
	// hold some bytes the client sent), and req is the parsed request.
	Handle(client net.Conn, reader *bufio.Reader, req *Request) error
}

// SelectHandler chooses the right strategy for a request. CONNECT means the
// client wants an opaque tunnel (used for HTTPS); anything else is a normal
// HTTP request we forward on the client's behalf.
func SelectHandler(req *Request) Handler {
	if req.Method == "CONNECT" {
		return &ConnectHandler{}
	}
	return &HTTPHandler{}
}
