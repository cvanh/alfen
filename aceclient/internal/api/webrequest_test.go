package api

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestExecuteWebRequestGET(t *testing.T) {
	c, ss := newScripted(t, always(200, `{"a":nan,"b":1}`))
	c.AccessToken = "tok"
	state, resp, err := c.ExecuteWebRequest(context.Background(), "prop", "ids=2059_0,2060_0", nil, ExecOptions{})
	if err != nil || state != ValidResponse {
		t.Fatalf("state=%v err=%v", state, err)
	}
	if resp.StatusCode != 200 || resp.Body != `{"a":null,"b":1}` {
		t.Fatalf("resp = %+v (\":nan\" must become \":null\")", resp)
	}
	r := ss.requests()[0]
	if r.Method != "GET" || r.Path != "/api/prop" || r.Query != "ids=2059_0,2060_0" || r.Body != "" {
		t.Fatalf("request = %+v", r)
	}
	if r.Auth != "Bearer tok" || r.Accept != "application/json" {
		t.Fatalf("headers = %+v", r)
	}
	if c.LastHTTPStatus() != 200 || c.LastCommand() != "prop" || c.LastValidResponse().IsZero() {
		t.Fatalf("bookkeeping: status=%d cmd=%q valid=%v", c.LastHTTPStatus(), c.LastCommand(), c.LastValidResponse())
	}
}

func TestExecuteWebRequestPOST(t *testing.T) {
	c, ss := newScripted(t, always(200, `{"version":1}`))
	body := `{"OD_x":{"id":"2059_0","value":1}}`
	state, _, err := c.ExecuteWebRequest(context.Background(), "prop", "", body, ExecOptions{})
	if err != nil || state != ValidResponse {
		t.Fatalf("state=%v err=%v", state, err)
	}
	r := ss.requests()[0]
	if r.Method != "POST" || r.Path != "/api/prop" || r.Query != "" || r.Body != body {
		t.Fatalf("request = %+v", r)
	}
	if r.ContentType != "application/json; charset=utf-8" || r.Auth != "" {
		t.Fatalf("headers = %+v", r)
	}
}

func TestExecuteWebRequest404IsValidAndEmpty(t *testing.T) {
	c, _ := newScripted(t, always(404, "not here"))
	state, resp, err := c.ExecuteWebRequest(context.Background(), "nope", "", nil, ExecOptions{})
	if err != nil || state != ValidResponse || resp.StatusCode != 404 || resp.Body != "" {
		t.Fatalf("state=%v resp=%+v err=%v", state, resp, err)
	}
}

func TestExecuteWebRequestStatusHandling(t *testing.T) {
	const id = "ACE0012345"
	tests := []struct {
		name       string
		status     int
		maxRetries int
		attempts   int
		state      RequestState
		class      error
		message    string // "%IP%" is replaced by the client IP
		loggedOut  bool
	}{
		{"400 not last retry", 400, 3, 1, Unsuccessful, ErrUnsupportedRequest, "", false},
		{"400 last retry", 400, 1, 1, Unsuccessful, ErrUnsupportedRequest, "Device '" + id + "' does not support the request, please contact support.", false},
		{"501 last retry", 501, 1, 1, Unsuccessful, ErrUnsupportedRequest, "Device '" + id + "' does not support the request, please contact support.", false},
		{"406 last retry", 406, 1, 1, Unsuccessful, ErrInvalidCertificate, "Device '" + id + "' has an invalid or expired certificate, please contact support.", false},
		{"500 last retry", 500, 1, 1, Unsuccessful, ErrConnectionLost, "Device '" + id + "'/%IP% connection lost.", true},
		{"503 not last retry", 503, 2, 1, Unsuccessful, ErrConnectionLost, "", true},
		{"409", 409, 1, 1, Unsuccessful, ErrConnectionLost, "Device '" + id + "'/%IP% connection lost.", true},
		{"300", 300, 1, 1, Unsuccessful, ErrConnectionLost, "Device '" + id + "'/%IP% connection lost.", true},
		{"504 retries then gives up", 504, 2, 2, Unsuccessful, ErrRequestTimeout, "", true},
		{"429 no re-login", 429, 4, 1, Unsuccessful, ErrUnauthorized, "", true},
		{"418 default keeps retrying", 418, 3, 3, RetryRequest, ErrRetriesExhausted, "", false},
		{"502 default keeps retrying", 502, 2, 2, RetryRequest, ErrRetriesExhausted, "", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c, ss := newScripted(t, always(tc.status, "x"))
			c.AccessToken = "tok"
			c.SetLoggedIn(true)
			c.SetDefaultCategoryCollected(true)
			state, resp, err := c.ExecuteWebRequest(context.Background(), "prop", "", nil, ExecOptions{MaxRetries: tc.maxRetries})
			if got := len(ss.requests()); got != tc.attempts {
				t.Errorf("attempts = %d, want %d", got, tc.attempts)
			}
			if state != tc.state || resp.StatusCode != tc.status {
				t.Errorf("state=%v status=%d, want %v %d", state, resp.StatusCode, tc.state, tc.status)
			}
			var re *RequestError
			if !errors.As(err, &re) || !errors.Is(err, tc.class) {
				t.Fatalf("err = %v, want RequestError of class %v", err, tc.class)
			}
			want := strings.ReplaceAll(tc.message, "%IP%", c.IP)
			if re.Message != want {
				t.Errorf("message = %q, want %q", re.Message, want)
			}
			if c.IsLoggedIn() == tc.loggedOut {
				t.Errorf("IsLoggedIn = %v, want %v", c.IsLoggedIn(), !tc.loggedOut)
			}
			if tc.loggedOut && c.IsDefaultCategoryCollected() {
				t.Errorf("IsDefaultCategoryCollected not reset")
			}
			if tc.status == 429 && c.AccessToken != "" {
				t.Errorf("AccessToken not cleared on 429")
			}
			if c.LastHTTPStatus() != tc.status {
				t.Errorf("LastHTTPStatus = %d", c.LastHTTPStatus())
			}
		})
	}
}

func TestExecuteWebRequestTimeoutThenSuccess(t *testing.T) {
	lowerTimeouts(t)
	c, ss := newScripted(t, func(n int, _ seenRequest) reply {
		if n < 2 {
			return reply{block: true}
		}
		return reply{status: 200, body: "{}"}
	})
	var pings int32
	c.SetNetworkCheck(func(context.Context, string, int) (bool, string) { atomic.AddInt32(&pings, 1); return true, "" })
	c.SetLoggedIn(true)
	state, resp, err := c.ExecuteWebRequest(context.Background(), "prop", "", nil, ExecOptions{Timeout: 50 * time.Millisecond})
	if err != nil || state != ValidResponse || resp.StatusCode != 200 {
		t.Fatalf("state=%v resp=%+v err=%v", state, resp, err)
	}
	if n := len(ss.requests()); n != 3 {
		t.Fatalf("attempts = %d, want 3", n)
	}
	if pings != 2 {
		t.Fatalf("network checks = %d, want 2", pings)
	}
	if c.IsLoggedIn() {
		t.Fatal("a 408 must reset IsLoggedIn")
	}
}

func TestExecuteWebRequestTimeoutExhausted(t *testing.T) {
	lowerTimeouts(t)
	c, ss := newScripted(t, func(int, seenRequest) reply { return reply{block: true} })
	c.SetNetworkCheck(func(context.Context, string, int) (bool, string) { return false, "PING FAILED" })
	state, resp, err := c.ExecuteWebRequest(context.Background(), "prop", "", nil, ExecOptions{Timeout: 30 * time.Millisecond, MaxRetries: 2})
	if state != Unsuccessful || resp.StatusCode != http.StatusRequestTimeout {
		t.Fatalf("state=%v resp=%+v", state, resp)
	}
	var re *RequestError
	if !errors.As(err, &re) || !errors.Is(err, ErrRequestTimeout) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v", err)
	}
	if re.Message != "PING FAILED" {
		t.Fatalf("message = %q (the failed ping text is shown on the last retry)", re.Message)
	}
	if n := len(ss.requests()); n != 2 {
		t.Fatalf("attempts = %d", n)
	}
}

func TestExecuteWebRequestUploadingSkipsNetworkCheck(t *testing.T) {
	lowerTimeouts(t)
	c, ss := newScripted(t, func(int, seenRequest) reply { return reply{block: true} })
	c.SetNetworkCheck(func(context.Context, string, int) (bool, string) {
		t.Error("network check while uploading")
		return true, ""
	})
	c.SetUploading(true)
	state, _, err := c.ExecuteWebRequest(context.Background(), "firmware", "", nil, ExecOptions{Timeout: 30 * time.Millisecond})
	if state != Unsuccessful || !errors.Is(err, ErrRequestTimeout) || len(ss.requests()) != 1 {
		t.Fatalf("state=%v err=%v attempts=%d", state, err, len(ss.requests()))
	}
}

func TestExecuteWebRequestRebootingAfterTimeout(t *testing.T) {
	c, ss := newScripted(t, always(500, ""))
	c.SetRebooting(true)
	c.SetLoggedIn(true)
	c.sess.lastStatus = http.StatusRequestTimeout // previous request timed out
	state, _, err := c.ExecuteWebRequest(context.Background(), "prop", "", nil, ExecOptions{})
	if state != Unsuccessful || !errors.Is(err, ErrDeviceRebooting) || len(ss.requests()) != 1 {
		t.Fatalf("state=%v err=%v attempts=%d", state, err, len(ss.requests()))
	}
	if !c.IsLoggedIn() {
		t.Fatal("the rebooting shortcut must not touch IsLoggedIn")
	}
	c.FinishedRebooting()
	if c.IsRebooting() || c.LastHTTPStatus() != 200 {
		t.Fatal("FinishedRebooting")
	}
}

func TestExecuteWebRequestReloginOn401(t *testing.T) {
	c, ss := newScripted(t, func(n int, r seenRequest) reply {
		switch n {
		case 0:
			return reply{status: 401, body: "{}"}
		case 1:
			return reply{status: 200, body: `{"access":"new","refresh":"r2"}`}
		default:
			return reply{status: 200, body: `{"ok":true}`}
		}
	})
	c.Creds = LoginData{Username: "admin", Password: "secret-for-test", DisplayName: "Installer"}
	c.AccessToken = "old"
	state, resp, err := c.ExecuteWebRequest(context.Background(), "prop", "cat=generic", nil, ExecOptions{})
	if err != nil || state != ValidResponse || resp.Body != `{"ok":true}` {
		t.Fatalf("state=%v resp=%+v err=%v", state, resp, err)
	}
	reqs := ss.requests()
	if len(reqs) != 3 {
		t.Fatalf("requests = %+v", reqs)
	}
	if reqs[0].Auth != "Bearer old" {
		t.Errorf("first auth = %q", reqs[0].Auth)
	}
	login := reqs[1]
	if login.Method != "POST" || login.Path != "/api/login" || login.Auth != "" ||
		login.Body != `{"username":"admin","password":"secret-for-test","displayname":"Installer"}` {
		t.Errorf("login request = %+v", login)
	}
	if reqs[2].Auth != "Bearer new" || reqs[2].Query != "cat=generic" {
		t.Errorf("retry = %+v", reqs[2])
	}
	if c.AccessToken != "new" || c.RefreshToken != "r2" || !c.IsLoggedIn() {
		t.Errorf("tokens %q %q loggedIn=%v", c.AccessToken, c.RefreshToken, c.IsLoggedIn())
	}
}

func TestExecuteWebRequestRefreshOn403(t *testing.T) {
	c, ss := newScripted(t, func(n int, r seenRequest) reply {
		switch n {
		case 0:
			return reply{status: 403}
		case 1:
			return reply{status: 200, body: `{"access":"a3","refresh":"r3"}`}
		default:
			return reply{status: 200, body: "{}"}
		}
	})
	c.AccessToken, c.RefreshToken = "old", "r1"
	state, _, err := c.ExecuteWebRequest(context.Background(), "prop", "", nil, ExecOptions{})
	if err != nil || state != ValidResponse {
		t.Fatalf("state=%v err=%v", state, err)
	}
	reqs := ss.requests()
	if len(reqs) != 3 || reqs[1].Path != "/api/token/refresh" || reqs[1].Body != `{"refresh":"r1"}` || reqs[1].Auth != "" {
		t.Fatalf("requests = %+v", reqs)
	}
	if reqs[2].Auth != "Bearer a3" || c.LastLoginStatus() != 200 {
		t.Fatalf("retry auth %q, LastLoginStatus %d", reqs[2].Auth, c.LastLoginStatus())
	}
}

func TestExecuteWebRequestNoReloginForExcludedCommands(t *testing.T) {
	for _, cmd := range []string{"login", "logout", "prc"} {
		c, ss := newScripted(t, always(401, ""))
		c.Creds = LoginData{Username: "admin", Password: "x"}
		state, _, err := c.ExecuteWebRequest(context.Background(), cmd, "", "", ExecOptions{})
		if state != Unsuccessful || !errors.Is(err, ErrUnauthorized) || len(ss.requests()) != 1 {
			t.Errorf("%s: state=%v err=%v requests=%d", cmd, state, err, len(ss.requests()))
		}
	}
}

func TestExecuteWebRequestReloginFails(t *testing.T) {
	c, ss := newScripted(t, func(n int, r seenRequest) reply {
		if n == 0 {
			return reply{status: 401}
		}
		return reply{status: 403, body: "{}"}
	})
	c.Creds = LoginData{Username: "admin", Password: "x"}
	state, _, err := c.ExecuteWebRequest(context.Background(), "prop", "", nil, ExecOptions{})
	if state != Unsuccessful || !errors.Is(err, ErrUnauthorized) || len(ss.requests()) != 2 {
		t.Fatalf("state=%v err=%v requests=%d", state, err, len(ss.requests()))
	}
}

func TestExecuteWebRequestConnectionRefused(t *testing.T) {
	srv := httptest.NewTLSServer(http.NotFoundHandler())
	c := clientFor(t, srv)
	srv.Close()
	state, resp, err := c.ExecuteWebRequest(context.Background(), "prop", "", nil, ExecOptions{MaxRetries: 1})
	// net481: HttpRequestException(WebException) hits the generic catch, the
	// status stays at its initial InternalServerError.
	if state != Unsuccessful || resp.StatusCode != 500 || !errors.Is(err, ErrConnectionLost) {
		t.Fatalf("state=%v resp=%+v err=%v", state, resp, err)
	}
	var ue *url.Error
	var re *RequestError
	if !errors.As(err, &ue) || !errors.As(err, &re) || re.Message == "" {
		t.Fatalf("err = %v", err)
	}
}

func TestExecuteWebRequestParentCancelled(t *testing.T) {
	c, _ := newScripted(t, always(200, "{}"))
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	state, resp, err := c.ExecuteWebRequest(ctx, "prop", "", nil, ExecOptions{})
	if state != Unsuccessful || resp.StatusCode != 408 || !errors.Is(err, ErrRequestTimeout) || !errors.Is(err, context.Canceled) {
		t.Fatalf("state=%v resp=%+v err=%v", state, resp, err)
	}
}

func TestExecuteWebRequestMultipart(t *testing.T) {
	var parts [][]byte
	var ctype string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctype = r.Header.Get("Content-Type")
		_, params, _ := mime.ParseMediaType(ctype)
		mr := multipart.NewReader(r.Body, params["boundary"])
		for {
			p, err := mr.NextPart()
			if err != nil {
				break
			}
			if p.FormName() != "firmwarefile" || p.FileName() != "filename" {
				t.Errorf("part %q %q", p.FormName(), p.FileName())
			}
			b, _ := io.ReadAll(p)
			parts = append(parts, b)
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()
	c := clientFor(t, srv)
	data := bytes.Repeat([]byte{0xA5}, 5000)
	state, _, err := c.ExecuteWebRequest(context.Background(), "firmware", "", data, ExecOptions{Timeout: FirmwareUploadTimeout, MaxRetries: 1})
	if err != nil || state != ValidResponse {
		t.Fatalf("state=%v err=%v", state, err)
	}
	if !strings.HasPrefix(ctype, "multipart/form-data; boundary=") {
		t.Fatalf("content type %q", ctype)
	}
	if len(parts) != 2 || len(parts[0]) != 4096 || !bytes.Equal(bytes.Join(parts, nil), data) {
		t.Fatalf("parts: %d", len(parts))
	}
}

func TestExecuteWebRequestUnsupportedBody(t *testing.T) {
	c, ss := newScripted(t, always(200, ""))
	state, resp, err := c.ExecuteWebRequest(context.Background(), "prop", "", 42, ExecOptions{MaxRetries: 1})
	if state != Unsuccessful || resp.StatusCode != 500 || !errors.Is(err, ErrConnectionLost) || len(ss.requests()) != 0 {
		t.Fatalf("state=%v resp=%+v err=%v", state, resp, err)
	}
}

func TestLogoutContext(t *testing.T) {
	c, ss := newScripted(t, always(200, "{}"))
	c.AccessToken, c.RefreshToken = "a", "r"
	c.SetLoggedIn(true)
	if c.LogoutContext(context.Background(), false) {
		t.Fatal("no request made yet: the C# skips the logout call (m_httpClient == null)")
	}
	if c.AccessToken != "a" || c.IsLoggedIn() {
		t.Fatal("tokens kept, logged out")
	}
	if st, _, _ := c.ExecuteWebRequest(context.Background(), "info", "", nil, ExecOptions{}); st != ValidResponse {
		t.Fatal(st)
	}
	if !c.LogoutContext(context.Background(), false) {
		t.Fatal("logout request should succeed")
	}
	reqs := ss.requests()
	last := reqs[len(reqs)-1]
	if last.Method != "POST" || last.Path != "/api/logout" || last.Body != "" || last.Auth != "Bearer a" {
		t.Fatalf("logout request = %+v", last)
	}
	if c.AccessToken != "" || c.RefreshToken != "" || c.IsLoggedIn() {
		t.Fatal("tokens must be cleared")
	}
	if c.LogoutContext(context.Background(), true) || len(ss.requests()) != len(reqs) {
		t.Fatal("second logout must not send a request")
	}
}

func TestRequestStateString(t *testing.T) {
	want := map[RequestState]string{ValidResponse: "VALID_RESPONSE", InvalidResponse: "INVALID_RESPONSE", RetryRequest: "RETRY_REQUEST", Unsuccessful: "UNSUCCESSFUL"}
	for s, w := range want {
		if s.String() != w || int(s) != map[string]int{"VALID_RESPONSE": 0, "INVALID_RESPONSE": 1, "RETRY_REQUEST": 2, "UNSUCCESSFUL": 3}[w] {
			t.Errorf("%d -> %s", s, s.String())
		}
	}
	if ProtocolForPort(443) != "https" || ProtocolForPort(80) != "http" {
		t.Error("ProtocolForPort")
	}
}
