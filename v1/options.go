package eloverblik

import (
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
)

// Option configures a client created with NewCustomer or NewThirdParty.
type Option func(*client)

// Defaults of the retry policy. The API restricts calls to /token to 2 per minute per IP
// and all calls to 120 per minute per IP (1200 per minute across all users), and answers
// 429 when a limit is exceeded, or 503 when DataHub cannot keep up. Both are transient,
// so a request is retried a couple of times before the error reaches the caller. The
// numbers are deliberately modest: a CLI must not appear to hang.
const (
	// DefaultRetryCount is the number of retries after the initial attempt.
	DefaultRetryCount = 2
	// DefaultRetryWait is the base backoff between attempts. resty jitters it and
	// doubles it per attempt, so the waits are roughly 5s and then 5-10s.
	DefaultRetryWait = 5 * time.Second
	// DefaultRetryMaxWait caps a single wait, including one asked for by a Retry-After
	// response header.
	DefaultRetryMaxWait = 60 * time.Second
)

// WithRetry overrides the default retry policy. Only the transient statuses 429 (rate
// limit exceeded) and 503 (DataHub unavailable) are retried; a 401 or any other 4xx is
// returned to the caller immediately. A Retry-After response header is honoured, capped
// at maxWait.
//
// count is the number of retries after the initial attempt, and is clamped to zero when
// negative. maxWait falls back to DefaultRetryMaxWait when it is zero or negative, and
// also caps the base backoff, so a short maxWait makes the whole policy short.
//
// Example:
//
//	customerClient := eloverblik.NewCustomer(refreshToken, eloverblik.WithRetry(4, 30*time.Second))
func WithRetry(count int, maxWait time.Duration) Option {
	return func(c *client) {
		setRetryPolicy(c.resty, count, maxWait)
	}
}

// WithoutRetry disables retrying: every response, including 429 and 503, is returned to
// the caller as it arrives.
//
// Example:
//
//	customerClient := eloverblik.NewCustomer(refreshToken, eloverblik.WithoutRetry())
func WithoutRetry() Option {
	return func(c *client) {
		setRetryPolicy(c.resty, 0, DefaultRetryMaxWait)
	}
}

// setRetryPolicy configures retrying on a resty client. It is idempotent: the retry
// condition is assigned rather than appended, so calling it again from an option
// replaces the policy instead of stacking a second condition on top of it.
func setRetryPolicy(client *resty.Client, count int, maxWait time.Duration) *resty.Client {
	if count < 0 {
		count = 0
	}
	if maxWait <= 0 {
		maxWait = DefaultRetryMaxWait
	}

	// The base backoff can never exceed the cap
	wait := DefaultRetryWait
	if wait > maxWait {
		wait = maxWait
	}

	client.RetryConditions = []resty.RetryConditionFunc{retryCondition}

	return client.
		SetRetryCount(count).
		SetRetryWaitTime(wait).
		SetRetryMaxWaitTime(maxWait).
		SetRetryAfter(retryAfter)
}

// retryCondition decides whether resty retries a response. See isRetryableError.
func retryCondition(res *resty.Response, err error) bool {
	if res == nil {
		return false
	}
	return isRetryableError(res.StatusCode(), err)
}

// retryAfter honours the Retry-After response header the API can send along with 429 and
// 503. Returning (0, nil) tells resty to fall back to its own jittered backoff. resty
// caps the returned wait at the client's max wait time.
func retryAfter(_ *resty.Client, res *resty.Response) (time.Duration, error) {
	if res == nil {
		return 0, nil
	}
	return parseRetryAfter(res.Header().Get("Retry-After")), nil
}

// parseRetryAfter reads both forms of the Retry-After header: a delay in seconds ("60")
// and an HTTP-date ("Wed, 21 Oct 2015 07:28:00 GMT"). It returns 0 when the header is
// absent, unparseable or already in the past.
func parseRetryAfter(value string) time.Duration {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0
	}

	if seconds, err := strconv.Atoi(value); err == nil {
		if seconds <= 0 {
			return 0
		}
		return time.Duration(seconds) * time.Second
	}

	if date, err := http.ParseTime(value); err == nil {
		if wait := time.Until(date); wait > 0 {
			return wait
		}
	}

	return 0
}

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
