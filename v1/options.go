package eloverblik

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
)

// Option configures a client created with NewCustomer or NewThirdParty.
type Option func(*client)

// WithResponseHeaderOutput writes the HTTP response headers of every API call to w.
//
// It is intended for debugging. Headers are written as one block per response,
// prefixed with the request method, the request URL and the status line, so calls
// can be told apart:
//
//	< GET https://api.eloverblik.dk/customerapi/api/token -> 200 OK
//	< Content-Type: application/json; charset=utf-8
//	< Date: Mon, 01 Jan 2024 00:00:00 GMT
//
// Example:
//
//	customerClient := eloverblik.NewCustomer(refreshToken, eloverblik.WithResponseHeaderOutput(os.Stderr))
func WithResponseHeaderOutput(w io.Writer) Option {
	return func(c *client) {
		if w == nil {
			return
		}

		// Wrap the transport instead of using an after-response middleware: requests
		// made with SetDoNotParseResponse(true), such as the export endpoints, skip
		// all resty after-response middlewares. A transport wrapper sees every response.
		transport := c.resty.GetClient().Transport
		if transport == nil {
			transport = http.DefaultTransport
		}

		c.resty.SetTransport(&responseHeaderPrinter{
			transport: transport,
			out:       w,
		})
	}
}

// responseHeaderPrinter is an http.RoundTripper that writes the response headers of
// every response it sees to out. The response is returned untouched and its body is
// neither buffered nor consumed, so streamed responses keep working.
type responseHeaderPrinter struct {
	transport http.RoundTripper
	out       io.Writer
	mu        sync.Mutex
}

func (p *responseHeaderPrinter) RoundTrip(req *http.Request) (*http.Response, error) {
	res, err := p.transport.RoundTrip(req)
	if err != nil || res == nil {
		return res, err
	}
	p.write(req, res)
	return res, nil
}

// write renders a single header block and writes it to out. Write errors are ignored:
// debug output must never break an API call.
func (p *responseHeaderPrinter) write(req *http.Request, res *http.Response) {
	var block strings.Builder

	fmt.Fprintf(&block, "< %s %s -> %s\n", req.Method, req.URL, res.Status)

	keys := make([]string, 0, len(res.Header))
	for key := range res.Header {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		for _, value := range res.Header[key] {
			fmt.Fprintf(&block, "< %s: %s\n", key, value)
		}
	}
	block.WriteString("\n")

	// Guard the writer so parallel requests cannot interleave partial lines.
	p.mu.Lock()
	defer p.mu.Unlock()
	_, _ = io.WriteString(p.out, block.String())
}
