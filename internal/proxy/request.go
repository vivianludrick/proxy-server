package proxy

import (
	"bufio"
	"errors"
	"net"
	"strings"
)

// Header is a single HTTP header line. We keep headers as an ordered slice
// (instead of a map) so the order the client sent them in is preserved when we
// forward the request to the target server.
type Header struct {
	Name  string
	Value string
}

// Request holds everything we parsed out of the first line and headers of an
// incoming proxy request. The fields below the raw values (Host, Port, Path)
// are filled in by resolveTarget once we know what kind of request this is.
type Request struct {
	Method  string   // e.g. "GET", "POST", "CONNECT"
	Target  string   // the raw target from the request line
	Version string   // e.g. "HTTP/1.1"
	Headers []Header // headers in the order they arrived

	Host string // target host detected from the request
	Port string // target port detected from the request
	Path string // origin-form path used when forwarding plain HTTP
}

// HeaderValue returns the value of the first header matching name
// (case-insensitive), or an empty string if it is not present.
func (r *Request) HeaderValue(name string) string {
	for _, h := range r.Headers {
		if strings.EqualFold(h.Name, name) {
			return h.Value
		}
	}
	return ""
}

// ParseRequest reads and parses a single HTTP request from reader. It reads the
// request line and all header lines, then works out which host and port the
// request is destined for.
func ParseRequest(reader *bufio.Reader) (*Request, error) {
	line, err := readLine(reader)
	if err != nil {
		return nil, err
	}

	// The request line looks like: METHOD TARGET VERSION
	parts := strings.SplitN(line, " ", 3)
	if len(parts) != 3 {
		return nil, errors.New("malformed request line: " + line)
	}

	req := &Request{
		Method:  parts[0],
		Target:  parts[1],
		Version: parts[2],
	}

	if err := parseHeaders(reader, req); err != nil {
		return nil, err
	}

	if err := resolveTarget(req); err != nil {
		return nil, err
	}

	return req, nil
}

// parseHeaders reads header lines until it hits the blank line that separates
// the headers from the body.
func parseHeaders(reader *bufio.Reader, req *Request) error {
	for {
		line, err := readLine(reader)
		if err != nil {
			return err
		}
		if line == "" { // blank line marks the end of the headers
			return nil
		}

		colon := strings.Index(line, ":")
		if colon < 0 {
			continue // ignore anything that is not a valid header
		}

		name := strings.TrimSpace(line[:colon])
		value := strings.TrimSpace(line[colon+1:])
		req.Headers = append(req.Headers, Header{Name: name, Value: value})
	}
}

// resolveTarget figures out the destination host and port from the request.
// This is the "detect target host and port" step.
func resolveTarget(req *Request) error {
	// For HTTPS, browsers send: CONNECT host:port HTTP/1.1
	if req.Method == "CONNECT" {
		host, port := splitHostPort(req.Target, "443")
		req.Host = host
		req.Port = port
		return nil
	}

	// For plain HTTP, the target is usually an absolute URI like
	// "http://example.com/path". Strip the scheme so we are left with the
	// authority and path.
	target := req.Target
	target = strings.TrimPrefix(target, "http://")
	target = strings.TrimPrefix(target, "https://")

	authority := target
	path := "/"
	if slash := strings.Index(target, "/"); slash >= 0 {
		authority = target[:slash]
		path = target[slash:]
	}

	// If there was no authority, the client sent an origin-form request line
	// (just a path). Fall back to the Host header to find the destination.
	if authority == "" {
		authority = req.HeaderValue("Host")
		path = req.Target
	}
	if authority == "" {
		return errors.New("could not determine target host")
	}

	host, port := splitHostPort(authority, "80")
	req.Host = host
	req.Port = port
	req.Path = path
	return nil
}

// splitHostPort separates an authority like "example.com:8080" into its host
// and port. If no port is present, defPort is used. This is what makes
// non-FQDN targets such as "localhost:3000" work correctly.
func splitHostPort(authority, defPort string) (string, string) {
	if host, port, err := net.SplitHostPort(authority); err == nil {
		return host, port
	}
	return authority, defPort
}

// readLine reads one line ending in "\n" and strips the trailing "\r\n" (or
// "\n"). It returns the line without the line terminator.
func readLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}
