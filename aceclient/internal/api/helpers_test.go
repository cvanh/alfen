package api

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"testing"
	"time"
)

// seenRequest is what the scripted server records for every request.
type seenRequest struct {
	Method, Path, Query, Body string
	Auth, ContentType, Accept string
}

// reply is one scripted response; block makes the handler wait for the
// client to give up (timeouts).
type reply struct {
	status int
	body   string
	block  bool
}

// scriptServer answers requests with a per-index reply function.
type scriptServer struct {
	mu    sync.Mutex
	reqs  []seenRequest
	reply func(n int, r seenRequest) reply
}

func (s *scriptServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data, _ := io.ReadAll(r.Body)
	sr := seenRequest{
		Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Body: string(data),
		Auth: r.Header.Get("Authorization"), ContentType: r.Header.Get("Content-Type"), Accept: r.Header.Get("Accept"),
	}
	s.mu.Lock()
	n := len(s.reqs)
	s.reqs = append(s.reqs, sr)
	s.mu.Unlock()
	rp := s.reply(n, sr)
	if rp.block {
		<-r.Context().Done()
		return
	}
	w.WriteHeader(rp.status)
	_, _ = io.WriteString(w, rp.body)
}

func (s *scriptServer) requests() []seenRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]seenRequest(nil), s.reqs...)
}

// clientFor points a Client at srv the way the GUI would (api.New with
// insecure TLS), with a fixed identity and a network check that succeeds.
func clientFor(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	u, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	host, portStr, _ := net.SplitHostPort(u.Host)
	port, _ := strconv.Atoi(portStr)
	c := New(host, port, true)
	if u.Scheme == "http" {
		c.Protocol = "http"
	}
	c.Identity = "ACE0012345"
	c.SetNetworkCheck(func(context.Context, string, int) (bool, string) { return true, "" })
	return c
}

// newScripted starts a TLS server driven by fn and returns a client for it.
func newScripted(t *testing.T, fn func(n int, r seenRequest) reply) (*Client, *scriptServer) {
	t.Helper()
	ss := &scriptServer{reply: fn}
	srv := httptest.NewTLSServer(ss)
	t.Cleanup(srv.Close)
	return clientFor(t, srv), ss
}

// always returns the same reply for every request.
func always(status int, body string) func(int, seenRequest) reply {
	return func(int, seenRequest) reply { return reply{status: status, body: body} }
}

// lowerTimeouts shrinks the C# minimum timeout and ClassicLogin delay for
// the duration of a test.
func lowerTimeouts(t *testing.T) {
	t.Helper()
	oldMin, oldDelay := minRequestTimeout, classicLoginRetryDelay
	minRequestTimeout = 10 * time.Millisecond
	classicLoginRetryDelay = time.Millisecond
	t.Cleanup(func() { minRequestTimeout, classicLoginRetryDelay = oldMin, oldDelay })
}
