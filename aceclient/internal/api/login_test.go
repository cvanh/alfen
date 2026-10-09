package api

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLoginRequestSuccess(t *testing.T) {
	c, ss := newScripted(t, always(200, `{"access":"A1","refresh":"R1"}`))
	c.Creds = LoginData{Username: UserAdmin, Password: `"test-pass"`, DisplayName: "Installer"}
	res := c.LoginRequest(context.Background(), 0)
	if !res.LoggedIn || res.StatusCode != 200 || res.Content != `{"access":"A1","refresh":"R1"}` {
		t.Fatalf("res = %+v", res)
	}
	r := ss.requests()[0]
	// LoginData.Password.Trim('"') — the surrounding quotes are dropped.
	if r.Method != "POST" || r.Path != "/api/login" || r.ContentType != "application/json; charset=utf-8" ||
		r.Body != `{"username":"admin","password":"test-pass","displayname":"Installer"}` {
		t.Fatalf("request = %+v", r)
	}
	if c.AccessToken != "A1" || c.RefreshToken != "R1" || !c.IsLoggedIn() {
		t.Fatalf("tokens %q %q loggedIn %v", c.AccessToken, c.RefreshToken, c.IsLoggedIn())
	}
}

func TestLoginRequestRejected(t *testing.T) {
	c, _ := newScripted(t, always(403, `{"version":1}`))
	c.Creds = LoginData{Username: UserAdmin, Password: "wrong"}
	res := c.LoginRequest(context.Background(), 0)
	if res.LoggedIn || res.StatusCode != 403 || res.Content != `{"version":1}` || c.IsLoggedIn() {
		t.Fatalf("res = %+v", res)
	}
}

func TestLoginRequestCachedToken(t *testing.T) {
	c, ss := newScripted(t, always(500, ""))
	c.AccessToken = "cached"
	c.SetLastLoginStatus(200)
	res := c.LoginRequest(context.Background(), 0)
	if !res.LoggedIn || res.StatusCode != 200 || res.Content != "" || len(ss.requests()) != 0 || !c.IsLoggedIn() {
		t.Fatalf("res = %+v, requests %d", res, len(ss.requests()))
	}
}

func TestLoginRequestRefreshToken(t *testing.T) {
	c, ss := newScripted(t, always(200, `{"access":"A2"}`))
	c.RefreshToken = "R1"
	res := c.LoginRequest(context.Background(), 0)
	if !res.LoggedIn || res.StatusCode != 200 || res.Content != "" {
		t.Fatalf("res = %+v", res)
	}
	reqs := ss.requests()
	if len(reqs) != 1 || reqs[0].Path != "/api/token/refresh" || reqs[0].Body != `{"refresh":"R1"}` {
		t.Fatalf("requests = %+v", reqs)
	}
	// UpdateTokenInfo clears both tokens before reading the body.
	if c.AccessToken != "A2" || c.RefreshToken != "" || c.LastLoginStatus() != 200 {
		t.Fatalf("tokens %q %q status %d", c.AccessToken, c.RefreshToken, c.LastLoginStatus())
	}
}

func TestLoginRequestRefreshFailsThenLogin(t *testing.T) {
	c, ss := newScripted(t, func(n int, r seenRequest) reply {
		if r.Path == "/api/token/refresh" {
			return reply{status: 401}
		}
		return reply{status: 200, body: `{"access":"A3","refresh":"R3"}`}
	})
	c.Creds = LoginData{Username: UserService, Password: "pw"}
	c.RefreshToken = "stale"
	res := c.LoginRequest(context.Background(), 0)
	reqs := ss.requests()
	if !res.LoggedIn || len(reqs) != 2 || reqs[1].Path != "/api/login" || c.AccessToken != "A3" {
		t.Fatalf("res = %+v requests = %+v", res, reqs)
	}
}

func TestLoginRequestTimeout(t *testing.T) {
	c, _ := newScripted(t, func(int, seenRequest) reply { return reply{block: true} })
	res := c.LoginRequest(context.Background(), 50*time.Millisecond)
	if res != (LoginResult{StatusCode: 408, Content: "RequestTimeout"}) {
		t.Fatalf("res = %+v", res)
	}
}

func TestLoginRequestConnectFailure(t *testing.T) {
	srv := httptest.NewTLSServer(http.NotFoundHandler())
	c := clientFor(t, srv)
	srv.Close()
	res := c.LoginRequest(context.Background(), 0)
	if res != (LoginResult{StatusCode: 408}) {
		t.Fatalf("res = %+v", res)
	}
}

func TestClassicLogin(t *testing.T) {
	lowerTimeouts(t)
	ss := &scriptServer{reply: func(n int, r seenRequest) reply {
		if n < 2 {
			return reply{status: 401}
		}
		return reply{status: 200, body: "{}"}
	}}
	srv := httptest.NewServer(ss)
	defer srv.Close()
	c := clientFor(t, srv)
	c.Creds = LoginData{Username: "user-a", Password: "pass-a", DisplayName: "d"}
	c.SetClassicLoginFallback(LoginData{Username: "legacy-user", Password: "legacy-pass"})
	res := c.LoginRequest(context.Background(), 0)
	if res != (LoginResult{LoggedIn: true, StatusCode: 200, Content: "OK"}) {
		t.Fatalf("res = %+v", res)
	}
	reqs := ss.requests()
	if len(reqs) != 3 {
		t.Fatalf("attempts = %d", len(reqs))
	}
	basic := func(u, p string) string { return "Basic " + base64.StdEncoding.EncodeToString([]byte(u+":"+p)) }
	if reqs[0].Auth != basic("user-a", "pass-a") || reqs[0].Body != `{"username":"user-a","password":"pass-a","displayname":"d"}` {
		t.Errorf("attempt 1 = %+v", reqs[0])
	}
	if reqs[2].Auth != basic("legacy-user", "legacy-pass") || !strings.Contains(reqs[2].Body, `"username":"legacy-user"`) {
		t.Errorf("attempt 3 = %+v", reqs[2])
	}
	if c.IsLoggedIn() {
		t.Error("ClassicLogin does not set IsLoggedIn in the C#")
	}
}

func TestClassicLoginAllFail(t *testing.T) {
	lowerTimeouts(t)
	ss := &scriptServer{reply: always(401, "")}
	srv := httptest.NewServer(ss)
	defer srv.Close()
	c := clientFor(t, srv)
	c.Creds = LoginData{Username: "u", Password: "p"}
	res := c.LoginRequest(context.Background(), 0)
	if res != (LoginResult{StatusCode: 401}) || len(ss.requests()) != 3 {
		t.Fatalf("res = %+v attempts %d", res, len(ss.requests()))
	}
}

func TestHandleUnsuccessfulLoginRequest(t *testing.T) {
	stale := time.Now().Add(-time.Hour)
	tests := []struct {
		name      string
		status    int
		content   string
		prev      time.Time
		unique    bool
		rebooting bool
		pingOK    bool
		class     error
		message   string
		lockout   time.Duration
	}{
		{"401", 401, "", time.Now(), true, false, true, ErrServiceAccessDisabled, "Secure Service Access is disabled for device 'ACE1'", 0},
		{"403", 403, "", time.Now(), true, false, true, ErrWrongPassword, "The password for device 'ACE1' is incorrect! Please provide a valid password.", 0},
		{"429 90s", 429, `{"version":1,"lockout_remaining_seconds":90}`, time.Now(), true, false, true, ErrLoginLockout,
			"Locked out of device 'ACE1' for 1.5 minutes due to multiple incorrect password attempts.", 90 * time.Second},
		{"429 60s", 429, `{"lockout_remaining_seconds":60}`, time.Now(), true, false, true, ErrLoginLockout,
			"Locked out of device 'ACE1' for 1.0 minute due to multiple incorrect password attempts.", time.Minute},
		{"429 3s", 429, `{"lockout_remaining_seconds":3}`, time.Now(), true, false, true, ErrLoginLockout,
			"Locked out of device 'ACE1' for 0.1 minutes due to multiple incorrect password attempts.", 3 * time.Second},
		{"429 no field", 429, `{"version":1}`, time.Now(), true, false, true, ErrLoginLockout,
			"Locked out of device 'ACE1' due to multiple incorrect password attempts.", 0},
		{"429 empty", 429, "", time.Now(), true, false, true, ErrLoginLockout,
			"Locked out of device 'ACE1' due to multiple incorrect password attempts.", 0},
		{"408 ping ok", 408, "", time.Now(), true, false, true, ErrLoginTimeout, "Login request timeout for ACE1 and IP 10.0.0.5", 0},
		{"408 ping fails", 408, "", time.Now(), true, false, false, ErrLoginTimeout, "PING FAILED", 0},
		{"503", 503, "", time.Now(), true, false, true, ErrDeviceUnreachable, "Device ACE1/10.0.0.5 is not reachable", 0},
		{"500", 500, "", time.Now(), true, false, true, ErrLoginUnknown, "Unknown error", 0},
		{"classic recent", 403, "", time.Now(), false, false, true, ErrLoginFailed, "", 0},
		{"classic stale", 403, "", stale, false, false, true, ErrNoValidResponse,
			"Login failed: Device 'ACE1' has given no valid response since " + stale.Local().Format("01/02/2006 15:04:05") + ", please reboot the device.", 0},
		{"classic stale rebooting", 403, "", stale, false, true, true, ErrLoginFailed, "", 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := New("10.0.0.5", 443, true)
			c.Identity = "ACE1"
			c.SetRebooting(tc.rebooting)
			c.SetNetworkCheck(func(context.Context, string, int) (bool, string) { return tc.pingOK, "PING FAILED" })
			le := c.HandleUnsuccessfulLoginRequest(context.Background(), tc.status, tc.content, tc.prev, LoginTimeout, tc.unique)
			var err error = le
			if !errors.Is(err, tc.class) {
				t.Fatalf("class = %v, want %v", le.Err, tc.class)
			}
			var as *LoginError
			if !errors.As(err, &as) || as.Message != tc.message || as.StatusCode != tc.status || as.LockoutRemaining != tc.lockout {
				t.Fatalf("got %+v\nwant message %q lockout %v", le, tc.message, tc.lockout)
			}
		})
	}
}

func TestAuthenticate(t *testing.T) {
	c, _ := newScripted(t, always(403, `{"version":1}`))
	res, err := c.Authenticate(context.Background(), LoginData{Username: UserAdmin, Password: "wrong"})
	if res.LoggedIn || !errors.Is(err, ErrWrongPassword) || res.Content != `{"version":1}` || c.IsLoggedIn() {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if !strings.Contains(err.Error(), "is incorrect") {
		t.Fatalf("err text %q", err)
	}

	c2, _ := newScripted(t, always(200, `{"access":"A","refresh":"R"}`))
	res, err = c2.Authenticate(context.Background(), LoginData{Username: UserTemp, Password: "tmp"})
	if err != nil || !res.LoggedIn || !c2.IsLoggedIn() || c2.LastLoginStatus() != 200 || !c2.Creds.IsTempUser() {
		t.Fatalf("res=%+v err=%v", res, err)
	}
}

func TestUniquePasswordRequired(t *testing.T) {
	tests := []struct {
		major int
		ahp   bool
		want  bool
	}{{0, false, false}, {4, false, false}, {4, true, true}, {5, false, true}, {6, true, true}}
	for _, tc := range tests {
		if got := UniquePasswordRequired(tc.major, tc.ahp); got != tc.want {
			t.Errorf("UniquePasswordRequired(%d,%v) = %v", tc.major, tc.ahp, got)
		}
	}
	c := New("1.2.3.4", 443, true)
	if !c.ShouldLoginUsingUniquePassword(5, false) {
		t.Error("fw 5 should use unique password")
	}
	c.SetUniquePasswordGracePeriod(true)
	if c.ShouldLoginUsingUniquePassword(5, false) {
		t.Error("grace period")
	}
	ld := LoginData{Username: UserService}
	if !ld.IsServiceUser() || ld.IsAdminUser() || ld.IsTempUser() {
		t.Error("user helpers")
	}
}

func TestValidateNewPassword(t *testing.T) {
	tests := []struct {
		p1, p2 string
		want   error
	}{
		{"short", "short", ErrPasswordTooShort},
		{strings.Repeat("a", 41), strings.Repeat("a", 41), ErrPasswordTooLong},
		{"abcdefghij,", "abcdefghij,", ErrPasswordInvalidChar},
		{`abcdefghij"`, `abcdefghij"`, ErrPasswordInvalidChar},
		{`abcdefghij\`, `abcdefghij\`, ErrPasswordInvalidChar},
		{"abcdefghij", "abcdefghik", ErrPasswordMismatch},
		{"abcdefghij", "abcdefghij", nil},
		{strings.Repeat("x", 40), strings.Repeat("x", 40), nil},
		{"😀😀😀😀😀", "😀😀😀😀😀", nil}, // 10 UTF-16 code units
	}
	for _, tc := range tests {
		if got := ValidateNewPassword(tc.p1, tc.p2); got != tc.want {
			t.Errorf("ValidateNewPassword(%q) = %v, want %v", tc.p1, got, tc.want)
		}
	}
	if ErrPasswordInvalidChar.Error() != `The new password contains an invalid character, the characters '\', '"' and ',' are not allowed. Please enter another password.` {
		t.Errorf("message %q", ErrPasswordInvalidChar)
	}
}

func TestNewPasswordRequiredForUpgrade(t *testing.T) {
	tests := []struct {
		ahp   bool
		major int
		file  string
		want  bool
	}{
		{false, 4, `C:\Firmware\NG9xx_5.1.0-3105_release.fwi`, true},
		{false, 4, "/tmp/ACE Service 5.10.2.fwi", true},
		{false, 4, "fw_4.9.0-1.fwi", false},
		{true, 4, "fw_5.1.0_x.tfw", false},
		{false, 5, "fw_6.0.0_x.fwi", false},
		{false, 4, "firmware.fwi", false},
		{false, 4, "fw_5x1x0_.fwi", false}, // Version.Parse would throw in the C#
	}
	for _, tc := range tests {
		if got := NewPasswordRequiredForUpgrade(tc.ahp, tc.major, tc.file); got != tc.want {
			t.Errorf("%q: got %v", tc.file, got)
		}
	}
}
