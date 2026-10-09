package isah

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// capturedRequest is what the fake IWS saw.
type capturedRequest struct {
	Method, Path, RawPath, RawQuery string
	Header                          http.Header
	Body                            []byte
}

// fakeIWS is an httptest TLS server that records requests and replies through
// handler.
type fakeIWS struct {
	srv  *httptest.Server
	mu   sync.Mutex
	reqs []capturedRequest
}

func newFakeIWS(t *testing.T, handler http.HandlerFunc) *fakeIWS {
	t.Helper()
	f := &fakeIWS{}
	f.srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		f.mu.Lock()
		f.reqs = append(f.reqs, capturedRequest{
			Method: r.Method, Path: r.URL.Path, RawPath: r.URL.EscapedPath(), RawQuery: r.URL.RawQuery,
			Header: r.Header.Clone(), Body: body,
		})
		f.mu.Unlock()
		handler(w, r)
	}))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeIWS) conn() *Connection {
	return &Connection{HTTP: NewHTTPClient(f.srv.Client().Transport)}
}

func (f *fakeIWS) requests() []capturedRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]capturedRequest(nil), f.reqs...)
}

func jsonReply(status int, body string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.WriteHeader(status)
		io.WriteString(w, body)
	}
}

func basic(s string) string { return "Basic " + base64.StdEncoding.EncodeToString([]byte(s)) }

func TestConstants(t *testing.T) {
	if DefaultSite != "https://installer.alfen.com" {
		t.Errorf("DefaultSite = %q", DefaultSite)
	}
	if RequestTimeout != 8*time.Second || ClientTimeout != 5*time.Minute {
		t.Errorf("timeouts = %v / %v", RequestTimeout, ClientTimeout)
	}
	c := NewHTTPClient(nil)
	tr, ok := c.Transport.(*http.Transport)
	if !ok || c.Timeout != ClientTimeout || c.CheckRedirect == nil {
		t.Fatalf("NewHTTPClient(nil) = %+v", c)
	}
	if !tr.DisableCompression || tr.Protocols == nil || !tr.Protocols.HTTP1() || tr.Protocols.HTTP2() {
		t.Errorf("default transport: DisableCompression=%v Protocols=%v", tr.DisableCompression, tr.Protocols)
	}
}

func TestRelativeURIFormat(t *testing.T) {
	cases := []struct {
		req  ObjectDataRequest
		want string
	}{
		{NewObjectDataRequest("ace0001234"),
			"api/settings/ace0001234?NumberOfSockets=1&ProcessorId=&AdditionalInfo=&ProductionRequest=False&ObjectCode=&IncludeLogo=True"},
		{ObjectDataRequest{ObjectID: "12345r01", NumberOfSockets: 2, ProcessorID: "42", AdditionalInfo: "user",
			ProductionRequest: true, ObjectCode: "OC", IncludeLogo: false},
			"api/settings/12345r01?NumberOfSockets=2&ProcessorId=42&AdditionalInfo=user&ProductionRequest=True&ObjectCode=OC&IncludeLogo=False"},
	}
	for _, c := range cases {
		if got := c.req.relativeURI(); got != c.want {
			t.Errorf("relativeURI = %q\nwant         %q", got, c.want)
		}
	}
}

func TestGetObjectDataRequestShape(t *testing.T) {
	f := newFakeIWS(t, jsonReply(200, `{"ObjectId":"ace0001234","Version":1,"Properties":[{"Id":8609,"SubId":0,"Value":"KEY"}]}`))
	req := ObjectDataRequest{ObjectID: "ace0001234", NumberOfSockets: 2, ProcessorID: "123456789", AdditionalInfo: "installer",
		ProductionRequest: false, ObjectCode: "", IncludeLogo: true}
	r, err := f.conn().GetObjectData(context.Background(), f.srv.URL, "company-credentials", req)
	if err != nil {
		t.Fatal(err)
	}
	if r.Err != nil || r.HTTPResponse == nil || r.HTTPResponse.StatusCode != 200 || r.HTTPResponse.Status != "OK" {
		t.Fatalf("response = %+v / %+v", r, r.HTTPResponse)
	}
	if r.IsahObject == nil || r.IsahObject.ObjectID != "ace0001234" || r.IsahObject.FindProperty(PropLicenseKey).ValueString() != "KEY" {
		t.Errorf("IsahObject = %+v", r.IsahObject)
	}
	reqs := f.requests()
	if len(reqs) != 1 {
		t.Fatalf("got %d requests", len(reqs))
	}
	got := reqs[0]
	if got.Method != http.MethodGet || got.Path != "/api/settings/ace0001234" {
		t.Errorf("request line = %s %s", got.Method, got.Path)
	}
	if want := "NumberOfSockets=2&ProcessorId=123456789&AdditionalInfo=installer&ProductionRequest=False&ObjectCode=&IncludeLogo=True"; got.RawQuery != want {
		t.Errorf("query = %q\nwant    %q", got.RawQuery, want)
	}
	if h := got.Header.Get("Accept"); h != "application/json" {
		t.Errorf("Accept = %q", h)
	}
	if h := got.Header.Get("Authorization"); h != basic("company-credentials") {
		t.Errorf("Authorization = %q", h)
	}
	if ua, ok := got.Header["User-Agent"]; ok {
		t.Errorf("User-Agent sent: %q", ua)
	}
	if len(got.Body) != 0 {
		t.Errorf("GET had a body: %q", got.Body)
	}
}

func TestGetObjectDataEscapingAndBase(t *testing.T) {
	f := newFakeIWS(t, jsonReply(404, ``))
	cases := []struct {
		name      string
		base      string
		req       ObjectDataRequest
		security  string
		wantPath  string
		wantQuery string
		wantAuth  string
	}{
		{"space and non-ASCII escaped like System.Uri", f.srv.URL,
			ObjectDataRequest{ObjectID: "a b", NumberOfSockets: 1, AdditionalInfo: "Jöhn Doe", IncludeLogo: true}, "c",
			"/api/settings/a%20b", "NumberOfSockets=1&ProcessorId=&AdditionalInfo=J%C3%B6hn%20Doe&ProductionRequest=False&ObjectCode=&IncludeLogo=True", basic("c")},
		{"ampersand is not encoded (C# string.Format)", f.srv.URL,
			ObjectDataRequest{ObjectID: "x", NumberOfSockets: 1, AdditionalInfo: "a&b=c", IncludeLogo: true}, "c",
			"/api/settings/x", "NumberOfSockets=1&ProcessorId=&AdditionalInfo=a&b=c&ProductionRequest=False&ObjectCode=&IncludeLogo=True", basic("c")},
		{"base without trailing slash replaces last segment", f.srv.URL + "/sub",
			NewObjectDataRequest("x"), "c", "/api/settings/x",
			"NumberOfSockets=1&ProcessorId=&AdditionalInfo=&ProductionRequest=False&ObjectCode=&IncludeLogo=True", basic("c")},
		{"base with trailing slash keeps path", f.srv.URL + "/sub/",
			NewObjectDataRequest("x"), "c", "/sub/api/settings/x",
			"NumberOfSockets=1&ProcessorId=&AdditionalInfo=&ProductionRequest=False&ObjectCode=&IncludeLogo=True", basic("c")},
		{"special characters and bad percent", f.srv.URL,
			ObjectDataRequest{ObjectID: `a\b`, NumberOfSockets: 1, AdditionalInfo: `"<{|}>^` + "`%zz%41\\", IncludeLogo: true}, "c",
			"/api/settings/a/b", "NumberOfSockets=1&ProcessorId=&AdditionalInfo=%22%3C%7B%7C%7D%3E%5E%60%25zz%41%5C&ProductionRequest=False&ObjectCode=&IncludeLogo=True", basic("c")},
		{"non-ASCII security becomes '?'", f.srv.URL, NewObjectDataRequest("x"), "cömpany😀",
			"/api/settings/x", "NumberOfSockets=1&ProcessorId=&AdditionalInfo=&ProductionRequest=False&ObjectCode=&IncludeLogo=True", basic("c?mpany?")},
	}
	for i, c := range cases {
		r, err := f.conn().GetObjectData(context.Background(), c.base, c.security, c.req)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if r.HTTPResponse.StatusCode != 404 || r.IsahObject != nil || r.Err != nil {
			t.Errorf("%s: response = %+v", c.name, r)
		}
		got := f.requests()[i]
		if got.RawPath != c.wantPath || got.RawQuery != c.wantQuery {
			t.Errorf("%s:\n path  %q want %q\n query %q\n want  %q", c.name, got.RawPath, c.wantPath, got.RawQuery, c.wantQuery)
		}
		if h := got.Header.Get("Authorization"); h != c.wantAuth {
			t.Errorf("%s: Authorization = %q, want %q", c.name, h, c.wantAuth)
		}
	}
}

func TestGetObjectDataResponses(t *testing.T) {
	cases := []struct {
		name       string
		handler    http.HandlerFunc
		wantStatus int
		wantObj    bool
		wantErr    bool
		wantBody   string
	}{
		{"object", jsonReply(200, `{"ObjectId":"x"}`), 200, true, false, `{"ObjectId":"x"}`},
		{"not found keeps real response", jsonReply(404, `{"message":"unknown"}`), 404, false, false, `{"message":"unknown"}`},
		{"server error not parsed", jsonReply(500, `not json`), 500, false, false, `not json`},
		{"2xx null body", jsonReply(200, `null`), 200, false, false, `null`},
		{"2xx empty body", jsonReply(204, ``), 204, false, false, ``},
		{"2xx invalid JSON becomes 408", jsonReply(200, `{"Version":`), 408, false, true, ``},
		{"2xx wrong type becomes 408", jsonReply(200, `{"Version":"abc"}`), 408, false, true, ``},
		{"2xx array becomes 408", jsonReply(200, `[]`), 408, false, true, ``},
	}
	for _, c := range cases {
		f := newFakeIWS(t, c.handler)
		r, err := f.conn().GetObjectData(context.Background(), f.srv.URL, "c", NewObjectDataRequest("x"))
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if r.HTTPResponse.StatusCode != c.wantStatus || (r.IsahObject != nil) != c.wantObj || (r.Err != nil) != c.wantErr {
			t.Errorf("%s: status=%d obj=%v err=%v", c.name, r.HTTPResponse.StatusCode, r.IsahObject, r.Err)
		}
		if string(r.HTTPResponse.Body) != c.wantBody {
			t.Errorf("%s: body = %q, want %q", c.name, r.HTTPResponse.Body, c.wantBody)
		}
		if c.wantErr && r.HTTPResponse.Status != "Request Timeout" {
			t.Errorf("%s: synthesized status = %q", c.name, r.HTTPResponse.Status)
		}
	}
}

func TestGetObjectDataTransportFailures(t *testing.T) {
	// Deadline: the caller's context bounds the 8 s budget.
	slow := newFakeIWS(t, func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	})
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	r, err := slow.conn().GetObjectData(ctx, slow.srv.URL, "c", NewObjectDataRequest("x"))
	if err != nil || r.HTTPResponse.StatusCode != 408 || r.Err == nil || r.IsahObject != nil {
		t.Errorf("timeout: %+v, %v", r, err)
	}

	// Connection refused.
	closed := httptest.NewTLSServer(http.NotFoundHandler())
	closedURL := closed.URL
	closed.Close()
	r, err = (&Connection{}).GetObjectData(context.Background(), closedURL, "c", NewObjectDataRequest("x"))
	if err != nil || r.HTTPResponse.StatusCode != 408 || r.Err == nil {
		t.Errorf("refused: %+v, %v", r, err)
	}

	// Untrusted certificate with the default client: TLS error -> 408.
	f := newFakeIWS(t, jsonReply(200, `{}`))
	r, err = GetObjectData(context.Background(), f.srv.URL, "c", NewObjectDataRequest("x"))
	if err != nil || r.HTTPResponse.StatusCode != 408 || r.Err == nil {
		t.Errorf("untrusted cert: %+v, %v", r, err)
	}

	// Unsupported scheme: thrown inside the try -> 408.
	r, err = GetObjectData(context.Background(), "ftp://example.invalid/", "c", NewObjectDataRequest("x"))
	if err != nil || r.HTTPResponse.StatusCode != 408 || r.Err == nil {
		t.Errorf("ftp: %+v, %v", r, err)
	}
}

func TestGetObjectDataInvalidSite(t *testing.T) {
	for _, site := range []string{"", "installer.alfen.com", "/relative", "https://", "http://[::1"} {
		r, err := GetObjectData(context.Background(), site, "c", NewObjectDataRequest("x"))
		if err == nil || r != nil {
			t.Errorf("site %q: %+v, %v; want UriFormatException-like error", site, r, err)
		}
		if ok, err := TestIWSConnection(context.Background(), site); ok || err == nil {
			t.Errorf("TestIWSConnection(%q) = %v, %v", site, ok, err)
		}
	}
}

func TestGetObjectDataRedirects(t *testing.T) {
	var f *fakeIWS
	f = newFakeIWS(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/settings/"):
			http.Redirect(w, r, f.srv.URL+"/moved?x=1", http.StatusFound)
		case r.URL.Path == "/moved":
			jsonReply(200, `{"ObjectId":"moved"}`)(w, r)
		case r.URL.Path == "/loop":
			http.Redirect(w, r, "/loop", http.StatusFound)
		}
	})
	r, err := f.conn().GetObjectData(context.Background(), f.srv.URL, "secret", NewObjectDataRequest("x"))
	if err != nil || r.IsahObject == nil || r.IsahObject.ObjectID != "moved" {
		t.Fatalf("redirect: %+v, %v", r, err)
	}
	reqs := f.requests()
	if len(reqs) != 2 || reqs[0].Header.Get("Authorization") == "" {
		t.Fatalf("requests = %+v", reqs)
	}
	if h := reqs[1].Header.Get("Authorization"); h != "" {
		t.Errorf("Authorization must be cleared on redirect, got %q", h)
	}
	if h := reqs[1].Header.Get("Accept"); h != "application/json" {
		t.Errorf("Accept after redirect = %q", h)
	}

	ok, err := f.conn().TestIWSConnection(context.Background(), f.srv.URL+"/loop")
	if ok || err != nil {
		t.Errorf("redirect loop: %v, %v", ok, err)
	}
	if n := len(f.requests()) - 2; n != maxAutomaticRedirections+1 {
		t.Errorf("redirect loop made %d requests, want %d", n, maxAutomaticRedirections+1)
	}
}

func TestTestIWSConnection(t *testing.T) {
	cases := []struct {
		status int
		want   bool
	}{{200, true}, {204, true}, {301, false}, {401, false}, {500, false}}
	for _, c := range cases {
		f := newFakeIWS(t, jsonReply(c.status, `x`))
		ok, err := f.conn().TestIWSConnection(context.Background(), f.srv.URL)
		if err != nil || ok != c.want {
			t.Errorf("status %d: %v, %v", c.status, ok, err)
		}
		got := f.requests()[0]
		if got.Method != http.MethodGet || got.Path != "/" || got.RawQuery != "" {
			t.Errorf("request = %s %s?%s", got.Method, got.Path, got.RawQuery)
		}
		for _, h := range []string{"Authorization", "Accept", "User-Agent"} {
			if v, present := got.Header[h]; present {
				t.Errorf("unexpected %s: %q", h, v)
			}
		}
	}
	// Plain GET of the site URL itself, including its path.
	f := newFakeIWS(t, jsonReply(200, ``))
	if ok, err := f.conn().TestIWSConnection(context.Background(), f.srv.URL+"/health/check"); !ok || err != nil {
		t.Errorf("path site: %v, %v", ok, err)
	}
	if got := f.requests()[0].Path; got != "/health/check" {
		t.Errorf("path = %q", got)
	}
	// Unreachable: false without error (C# catch block).
	closed := httptest.NewTLSServer(http.NotFoundHandler())
	closedURL := closed.URL
	closed.Close()
	if ok, err := TestIWSConnection(context.Background(), closedURL); ok || err != nil {
		t.Errorf("closed: %v, %v", ok, err)
	}
}
