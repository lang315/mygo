// Package httplite is the few names of net/http that the backends use, with
// its definitions: importing net/http itself costs an app 2.2 MB, whatever
// of it the app calls, and a backend is linked into every app of its
// platform. A test compares each name with net/http's.
package httplite

import "net/textproto"

// Header is http.Header: the same map type, which converts to it and to
// map[string][]string, with the methods the backends call.
type Header map[string][]string

// Get returns the first value of the key, or "".
func (h Header) Get(key string) string {
	if v := h[textproto.CanonicalMIMEHeaderKey(key)]; len(v) > 0 {
		return v[0]
	}
	return ""
}

// Set makes value the only value of the key.
func (h Header) Set(key, value string) { h[textproto.CanonicalMIMEHeaderKey(key)] = []string{value} }

// Add appends value to the values of the key.
func (h Header) Add(key, value string) {
	key = textproto.CanonicalMIMEHeaderKey(key)
	h[key] = append(h[key], value)
}

// Clone returns a copy of h, or nil if h is nil.
func (h Header) Clone() Header {
	if h == nil {
		return nil
	}
	out := make(Header, len(h))
	for k, v := range h {
		out[k] = append([]string(nil), v...)
	}
	return out
}

const (
	// MethodGet is the method of a request that reads.
	MethodGet = "GET"

	// TimeFormat is the format of the dates of HTTP headers.
	TimeFormat = "Mon, 02 Jan 2006 15:04:05 GMT"
)

// The status codes the backends use.
const (
	StatusOK                 = 200
	StatusNoContent          = 204
	StatusTemporaryRedirect  = 307
	StatusForbidden          = 403
	StatusBadGateway         = 502
	StatusServiceUnavailable = 503
)

// StatusText returns the text of an HTTP status code, or "" for a code it
// does not know.
func StatusText(code int) string {
	switch code {
	case 100:
		return "Continue"
	case 101:
		return "Switching Protocols"
	case 102:
		return "Processing"
	case 103:
		return "Early Hints"
	case 200:
		return "OK"
	case 201:
		return "Created"
	case 202:
		return "Accepted"
	case 203:
		return "Non-Authoritative Information"
	case 204:
		return "No Content"
	case 205:
		return "Reset Content"
	case 206:
		return "Partial Content"
	case 207:
		return "Multi-Status"
	case 208:
		return "Already Reported"
	case 226:
		return "IM Used"
	case 300:
		return "Multiple Choices"
	case 301:
		return "Moved Permanently"
	case 302:
		return "Found"
	case 303:
		return "See Other"
	case 304:
		return "Not Modified"
	case 305:
		return "Use Proxy"
	case 307:
		return "Temporary Redirect"
	case 308:
		return "Permanent Redirect"
	case 400:
		return "Bad Request"
	case 401:
		return "Unauthorized"
	case 402:
		return "Payment Required"
	case 403:
		return "Forbidden"
	case 404:
		return "Not Found"
	case 405:
		return "Method Not Allowed"
	case 406:
		return "Not Acceptable"
	case 407:
		return "Proxy Authentication Required"
	case 408:
		return "Request Timeout"
	case 409:
		return "Conflict"
	case 410:
		return "Gone"
	case 411:
		return "Length Required"
	case 412:
		return "Precondition Failed"
	case 413:
		return "Request Entity Too Large"
	case 414:
		return "Request URI Too Long"
	case 415:
		return "Unsupported Media Type"
	case 416:
		return "Requested Range Not Satisfiable"
	case 417:
		return "Expectation Failed"
	case 418:
		return "I'm a teapot"
	case 421:
		return "Misdirected Request"
	case 422:
		return "Unprocessable Entity"
	case 423:
		return "Locked"
	case 424:
		return "Failed Dependency"
	case 425:
		return "Too Early"
	case 426:
		return "Upgrade Required"
	case 428:
		return "Precondition Required"
	case 429:
		return "Too Many Requests"
	case 431:
		return "Request Header Fields Too Large"
	case 451:
		return "Unavailable For Legal Reasons"
	case 500:
		return "Internal Server Error"
	case 501:
		return "Not Implemented"
	case 502:
		return "Bad Gateway"
	case 503:
		return "Service Unavailable"
	case 504:
		return "Gateway Timeout"
	case 505:
		return "HTTP Version Not Supported"
	case 506:
		return "Variant Also Negotiates"
	case 507:
		return "Insufficient Storage"
	case 508:
		return "Loop Detected"
	case 510:
		return "Not Extended"
	case 511:
		return "Network Authentication Required"
	}
	return ""
}
