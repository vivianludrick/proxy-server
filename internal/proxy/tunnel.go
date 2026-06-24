package proxy

import (
	"io"
	"net"
	"time"
)

// dialTimeout bounds how long we wait when opening a connection to a target, so
// a slow or unreachable host fails fast with a clear error instead of hanging.
const dialTimeout = 10 * time.Second

// dialTarget opens a TCP connection to host:port. Using net.DialTimeout (rather
// than net.Dial) means an unreachable target returns an error promptly, which
// the caller turns into a 502 response.
func dialTarget(host, port string) (net.Conn, error) {
	return net.DialTimeout("tcp", net.JoinHostPort(host, port), dialTimeout)
}

// tunnel pipes data in both directions between the client and the target until
// both sides are closed. Using io.Copy means data is streamed in fixed-size
// chunks rather than buffered in full, so large downloads and video streams
// flow through without being held in memory.
//
// clientSrc is the reader used for the client -> target direction. We pass it
// separately (instead of reading from client directly) because the client's
// bytes are buffered behind a bufio.Reader, and reading from the raw connection
// would skip whatever is already sitting in that buffer.
func tunnel(client net.Conn, target net.Conn, clientSrc io.Reader) {
	done := make(chan struct{}, 2)

	go copyData(target, clientSrc, done) // client -> target
	go copyData(client, target, done)    // target -> client

	// Wait for both directions to finish before returning (which closes the
	// connections via the caller's deferred Close calls).
	<-done
	<-done
}

// copyData streams everything from src to dst, then half-closes the write side
// of dst so the other end sees EOF, and finally signals completion.
func copyData(dst net.Conn, src io.Reader, done chan struct{}) {
	io.Copy(dst, src)
	closeWrite(dst)
	done <- struct{}{}
}

// closeWrite shuts down only the writing half of a TCP connection. This lets us
// signal "no more data from this side" while still allowing the other half to
// keep streaming.
func closeWrite(conn net.Conn) {
	if tcp, ok := conn.(*net.TCPConn); ok {
		tcp.CloseWrite()
	}
}
