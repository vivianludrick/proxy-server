package proxy

import (
	"bufio"
	"net"
	"strings"
)

// HTTPHandler serves plain HTTP requests. It connects to the target server,
// forwards the request, and then pipes the response (and any request body)
// straight through.
type HTTPHandler struct{}

// Handle forwards a single HTTP request to its target and streams the response
// back to the client.
func (h *HTTPHandler) Handle(client net.Conn, reader *bufio.Reader, req *Request) error {
	target, err := dialTarget(req.Host, req.Port)
	if err != nil {
		writeError(client, "502 Bad Gateway", "could not connect to "+req.Host+":"+req.Port+": "+err.Error())
		return err
	}
	defer target.Close()

	// Send the rewritten request line and headers to the target server.
	if _, err := target.Write([]byte(buildRequest(req))); err != nil {
		return err
	}

	// Stream both directions. The client -> target copy carries any request
	// body (e.g. for POST); the target -> client copy streams the response.
	tunnel(client, target, reader)
	return nil
}

// buildRequest turns the parsed request back into raw bytes to send to the
// target. The request line is written in origin-form ("GET /path HTTP/1.1"),
// which is what an origin server expects, and hop-by-hop proxy headers are
// dropped.
func buildRequest(req *Request) string {
	var b strings.Builder

	b.WriteString(req.Method)
	b.WriteString(" ")
	b.WriteString(req.Path)
	b.WriteString(" ")
	b.WriteString(req.Version)
	b.WriteString("\r\n")

	for _, header := range req.Headers {
		if skipHeader(header.Name) {
			continue
		}
		b.WriteString(header.Name)
		b.WriteString(": ")
		b.WriteString(header.Value)
		b.WriteString("\r\n")
	}

	b.WriteString("\r\n")
	return b.String()
}

// skipHeader reports whether a header should not be forwarded to the target.
// Proxy-Connection is only meaningful between the client and the proxy.
func skipHeader(name string) bool {
	return strings.EqualFold(name, "Proxy-Connection")
}
