package proxy

import (
	"bufio"
	"net"
)

// ConnectHandler serves HTTPS (and any other) requests that arrive as a CONNECT.
// The proxy does not inspect the encrypted traffic; it simply opens a raw TCP
// connection to the target and pipes bytes back and forth.
type ConnectHandler struct{}

// Handle opens a tunnel to the target, tells the client the tunnel is ready,
// and then streams data in both directions.
func (h *ConnectHandler) Handle(client net.Conn, reader *bufio.Reader, req *Request) error {
	target, err := dialTarget(req.Host, req.Port)
	if err != nil {
		writeError(client, "502 Bad Gateway", "could not connect to "+req.Host+":"+req.Port+": "+err.Error())
		return err
	}
	defer target.Close()

	// Tell the client the tunnel has been established. After this point the
	// client begins the TLS handshake directly with the target.
	if _, err := client.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n")); err != nil {
		return err
	}

	// Pipe encrypted bytes both ways. We read the client side through reader so
	// any bytes already buffered there are not lost.
	tunnel(client, target, reader)
	return nil
}
