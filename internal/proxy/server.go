package proxy

import (
	"bufio"
	"log"
	"net"
	"strconv"
)

// Server is a basic HTTP/HTTPS proxy. It listens for client connections,
// parses each request, and dispatches it to the right handling strategy.
type Server struct {
	Addr   string      // address to listen on, e.g. ":8888"
	Logger *log.Logger // where to write log lines
}

// NewServer creates a Server listening on the given address and logging to
// logger.
func NewServer(addr string, logger *log.Logger) *Server {
	return &Server{Addr: addr, Logger: logger}
}

// ListenAndServe starts accepting connections and blocks until the listener
// fails. Each connection is handled in its own goroutine so many clients can
// be served at once.
func (s *Server) ListenAndServe() error {
	listener, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	defer listener.Close()

	s.Logger.Printf("proxy listening on %s", s.Addr)

	for {
		conn, err := listener.Accept()
		if err != nil {
			s.Logger.Printf("accept error: %v", err)
			continue
		}
		go s.handleConnection(conn)
	}
}

// handleConnection serves a single client connection from start to finish.
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)

	req, err := ParseRequest(reader)
	if err != nil {
		s.Logger.Printf("parse error from %s: %v", conn.RemoteAddr(), err)
		return
	}

	s.Logger.Printf("%s %s -> %s:%s", req.Method, req.Target, req.Host, req.Port)

	// Pick the strategy (HTTP forward vs. HTTPS tunnel) and run it.
	handler := SelectHandler(req)
	if err := handler.Handle(conn, reader, req); err != nil {
		s.Logger.Printf("handle error for %s:%s: %v", req.Host, req.Port, err)
	}
}

// writeError sends a minimal HTTP error response back to the client. status
// should be a full status string like "502 Bad Gateway". detail is included in
// the response body so the cause (e.g. the underlying dial error) is visible
// in the browser, not just a blank error page.
func writeError(client net.Conn, status, detail string) {
	body := status + "\n\n" + detail + "\n"
	client.Write([]byte(
		"HTTP/1.1 " + status + "\r\n" +
			"Content-Type: text/plain; charset=utf-8\r\n" +
			"Content-Length: " + strconv.Itoa(len(body)) + "\r\n" +
			"Connection: close\r\n\r\n" +
			body,
	))
}
