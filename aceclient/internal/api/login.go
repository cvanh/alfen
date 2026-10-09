package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// User names of ACEWebLoginData (ACENetwork/ICUNetwork/ACEWebLoginData.cs).
const (
	UserAdmin   = "admin"   // ACEWebLoginData.AdminUser
	UserService = "service" // ACEWebLoginData.ServiceUser
	UserTemp    = "temp"    // ACEWebLoginData.TempUser
)

// IsTempUser ports ACEWebLoginData.IsTempUser.
func (ld LoginData) IsTempUser() bool { return ld.Username == UserTemp }

// IsAdminUser ports ACEWebLoginData.IsAdminUser.
func (ld LoginData) IsAdminUser() bool { return ld.Username == UserAdmin }

// IsServiceUser ports ACEWebLoginData.IsServiceUser.
func (ld LoginData) IsServiceUser() bool { return ld.Username == UserService }

// UniquePasswordRequired ports ICULanDevice.IsUniquePasswordRequired:
// firmware major >= 5 requires a per-charger password, older firmware only
// on AHP models. Pass fwMajor 0 when the firmware version is unknown
// (m_vFirmwareVersion == null behaves like major < 5).
func UniquePasswordRequired(fwMajor int, isAHP bool) bool {
	if fwMajor < 5 {
		return isAHP
	}
	return true
}

// LoginResult ports the (IsLoggedIn, HttpStatusCode, Content) tuple that
// ICULanDevice.LoginRequest returns.
type LoginResult struct {
	LoggedIn   bool
	StatusCode int
	Content    string
}

// LoginRequest ports ICULanDevice.LoginRequest (ACENetwork/ICUNetwork/ICULanDevice.cs).
//
// HTTPS: when no access token is present but a refresh token is, it first
// POSTs /api/token/refresh {"refresh":..} (IsAccessTokenAvailableAsync, bound
// to ctx only, not to timeout). With a token it reports success without a
// login request, returning LastLoginStatus. Otherwise it POSTs /api/login
// with the hand-built body
//
//	{"username":"<u>","password":"<p trimmed of '"'>","displayname":"<d>"}
//
// within timeout (zero -> LoginTimeout), stores the "access"/"refresh"
// tokens (UpdateTokenInfo) and marks the client logged in.
//
// HTTP: ClassicLogin with a 5 s timeout per attempt.
//
// Errors are folded into the result as on net481: timeout/cancel ->
// (false, 408, "RequestTimeout"); connect failure -> (false, 408, "");
// other transport failure -> (false, 503, "An error occurred while sending
// the request."); anything else -> (false, 500, message).
func (c *Client) LoginRequest(ctx context.Context, timeout time.Duration) LoginResult {
	if ctx == nil {
		ctx = context.Background()
	}
	if timeout <= 0 {
		timeout = LoginTimeout
	}
	return c.loginRequest(ctx, timeout)
}

func (c *Client) loginRequest(ctx context.Context, timeout time.Duration) LoginResult {
	if !c.isHTTPS() {
		res, err := c.classicLogin(ctx, ClassicLoginTimeout)
		if err != nil {
			return loginException(err)
		}
		return res
	}
	tctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	ok, err := c.isAccessTokenAvailable(ctx)
	if err != nil {
		return loginException(err)
	}
	if ok {
		c.SetLoggedIn(true)
		return LoginResult{LoggedIn: true, StatusCode: c.LastLoginStatus()}
	}
	ld := c.creds()
	body := `{"username":"` + ld.Username + `","password":"` + strings.Trim(ld.Password, `"`) +
		`","displayname":"` + ld.DisplayName + `"}`
	uri, uriErr := c.buildURI("login", "")
	status, text, err := c.roundTrip(tctx, uri, uriErr, body)
	if err != nil {
		return loginException(err)
	}
	if status < 200 || status > 299 {
		return LoginResult{LoggedIn: false, StatusCode: status, Content: text}
	}
	c.updateTokenInfo(text)
	c.SetLoggedIn(true)
	return LoginResult{LoggedIn: true, StatusCode: status, Content: text}
}

// classicLogin ports ICULanDevice.ClassicLogin: up to three POST /api/login
// attempts (Basic auth applies over HTTP), one second apart; the third uses
// the legacy credentials (see SetClassicLoginFallback). Success reports the
// status name as content (HttpStatusCode.ToString()). Unlike LoginRequest's
// HTTPS path it does not set IsLoggedIn, exactly like the C#.
func (c *Client) classicLogin(ctx context.Context, timeout time.Duration) (LoginResult, error) {
	lastStatus := http.StatusServiceUnavailable
	uri, uriErr := c.buildURI("login", "")
	for retry := 0; retry < 3; retry++ {
		if retry == 2 {
			c.sess.mu.Lock()
			if fb := c.sess.classicFallback; fb != nil {
				c.Creds.Username = fb.Username
				c.Creds.Password = fb.Password
			}
			c.sess.mu.Unlock()
		}
		ld := c.creds()
		body := `{"username":"` + ld.Username + `","password":"` + ld.Password +
			`","displayname":"` + ld.DisplayName + `"}`
		actx, cancel := context.WithTimeout(ctx, timeout)
		status, _, err := c.roundTrip(actx, uri, uriErr, body)
		cancel()
		if err != nil {
			return LoginResult{}, err
		}
		lastStatus = status
		if status >= 200 && status <= 299 {
			return LoginResult{LoggedIn: true, StatusCode: status, Content: statusName(status)}, nil
		}
		select {
		case <-time.After(classicLoginRetryDelay):
		case <-ctx.Done():
			// The C# delay is not cancellable, but its next attempt fails
			// with TaskCanceledException on the cancelled linked token.
			return LoginResult{}, ctx.Err()
		}
	}
	return LoginResult{LoggedIn: false, StatusCode: lastStatus}, nil
}

// isAccessTokenAvailable ports ICULanDevice.IsAccessTokenAvailableAsync.
func (c *Client) isAccessTokenAvailable(ctx context.Context) (bool, error) {
	access, refresh := c.tokens()
	if access == "" && refresh != "" {
		uri, uriErr := c.buildURI("token/refresh", "")
		status, text, err := c.roundTrip(ctx, uri, uriErr, `{"refresh":"`+refresh+`"}`)
		if err != nil {
			return false, err
		}
		if status >= 200 && status <= 299 {
			c.SetLastLoginStatus(status)
			c.updateTokenInfo(text)
		}
	}
	access, _ = c.tokens()
	return access != "", nil
}

// updateTokenInfo ports ICULanDevice.UpdateTokenInfo: clear both tokens,
// then take "access"/"refresh" from the body (LogingResponse). A body that
// does not deserialize leaves both tokens cleared.
func (c *Client) updateTokenInfo(body string) {
	c.ClearTokens()
	if body == "" {
		return
	}
	var r struct {
		Access  *string `json:"access"`
		Refresh *string `json:"refresh"`
	}
	if err := json.Unmarshal([]byte(body), &r); err != nil {
		return
	}
	defer c.lock()()
	if r.Access != nil {
		c.AccessToken = *r.Access
	}
	if r.Refresh != nil {
		c.RefreshToken = *r.Refresh
	}
}

// loginException ports the catch blocks of LoginRequest (net481 shapes).
func loginException(err error) LoginResult {
	if isTimeout(err) { // TaskCanceledException
		return LoginResult{StatusCode: http.StatusRequestTimeout, Content: "RequestTimeout"}
	}
	var ue *url.Error
	if errors.As(err, &ue) { // HttpRequestException
		var op *net.OpError
		var dns *net.DNSError
		if errors.As(err, &op) && op.Op == "dial" && !errors.As(err, &dns) {
			// WebException wrapping a SocketException
			return LoginResult{StatusCode: http.StatusRequestTimeout}
		}
		return LoginResult{StatusCode: http.StatusServiceUnavailable, Content: "An error occurred while sending the request."}
	}
	return LoginResult{StatusCode: http.StatusInternalServerError, Content: err.Error()}
}

// statusName ports HttpStatusCode.ToString() for the success codes
// ClassicLogin can report.
func statusName(status int) string {
	switch status {
	case 200:
		return "OK"
	case 201:
		return "Created"
	case 202:
		return "Accepted"
	case 203:
		return "NonAuthoritativeInformation"
	case 204:
		return "NoContent"
	case 205:
		return "ResetContent"
	case 206:
		return "PartialContent"
	}
	return strconv.Itoa(status)
}

// Login outcome classes of HandleUnsuccessfulLoginRequest. A *LoginError
// unwraps to exactly one of them.
var (
	// ErrLoginFailed: classic (non unique-password) login failed; the C#
	// shows nothing.
	ErrLoginFailed = errors.New("api: login failed")
	// ErrNoValidResponse: classic login failed and the device has not
	// answered validly for more than three login timeouts.
	ErrNoValidResponse = errors.New("api: no valid response from device")
	// ErrServiceAccessDisabled: 401, Secure Service Access is disabled.
	ErrServiceAccessDisabled = errors.New("api: secure service access disabled")
	// ErrWrongPassword: 403, wrong password.
	ErrWrongPassword = errors.New("api: wrong password")
	// ErrLoginLockout: 429, locked out after repeated wrong passwords.
	ErrLoginLockout = errors.New("api: locked out")
	// ErrLoginTimeout: 408, the login request timed out.
	ErrLoginTimeout = errors.New("api: login request timeout")
	// ErrDeviceUnreachable: 503, the device is not reachable.
	ErrDeviceUnreachable = errors.New("api: device not reachable")
	// ErrLoginUnknown: any other status ("Unknown error").
	ErrLoginUnknown = errors.New("api: unknown login error")
)

// LoginError is the typed outcome of HandleUnsuccessfulLoginRequest. It
// replaces the showErrorCallback popup.
type LoginError struct {
	StatusCode int
	// Message is the exact text the C# passes to showErrorCallback; empty
	// when it shows nothing (ErrLoginFailed).
	Message string
	// LockoutRemaining is lockout_remaining_seconds from a 429 body.
	LockoutRemaining time.Duration
	// Err is one of the Err* login classes above.
	Err error
}

func (e *LoginError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Err.Error()
}

// Unwrap exposes the login class to errors.Is.
func (e *LoginError) Unwrap() error { return e.Err }

// HandleUnsuccessfulLoginRequest ports ICULanDevice.HandleUnsuccessfulLoginRequest.
// The C# always returns UNSUCCESSFUL; here the popup text and class come back
// as a *LoginError (never nil). previousValidResponse and timeout feed the
// classic-login "no valid response" check (DlgDeviceLogin passes now and
// LoginTimeout). A 408 runs CheckNetworkAndPing and, when that fails, uses
// its message instead.
//
// Deviations: dates use the invariant "MM/dd/yyyy HH:mm:ss" layout and
// numbers the invariant '.' (the C# formats with the current culture); a 429
// body that is not valid JSON is treated as having no remaining time (the C#
// throws, which DlgDeviceLogin reports as "Login failed").
func (c *Client) HandleUnsuccessfulLoginRequest(ctx context.Context, status int, content string, previousValidResponse time.Time, timeout time.Duration, isUniquePasswordRequired bool) *LoginError {
	if ctx == nil {
		ctx = context.Background()
	}
	id := c.Identity
	if !isUniquePasswordRequired {
		if !c.IsRebooting() && time.Since(previousValidResponse) > 3*timeout {
			msg := fmt.Sprintf("Login failed: Device '%s' has given no valid response since %s, please reboot the device.",
				id, previousValidResponse.Local().Format("01/02/2006 15:04:05"))
			return &LoginError{StatusCode: status, Message: msg, Err: ErrNoValidResponse}
		}
		return &LoginError{StatusCode: status, Err: ErrLoginFailed}
	}
	switch status {
	case http.StatusUnauthorized:
		return &LoginError{StatusCode: status, Err: ErrServiceAccessDisabled,
			Message: "Secure Service Access is disabled for device '" + id + "'"}
	case http.StatusForbidden:
		return &LoginError{StatusCode: status, Err: ErrWrongPassword,
			Message: "The password for device '" + id + "' is incorrect! Please provide a valid password."}
	case StatusLoginLockout:
		seconds, known := lockoutRemainingSeconds(content)
		text := ""
		if known {
			num := float64(seconds) / 60.0
			if num > 0 {
				arg := "minutes"
				if num == 1.0 {
					arg = "minute"
				}
				text = "for " + formatNetFixed(num, 1) + " " + arg + " "
			}
		}
		return &LoginError{StatusCode: status, Err: ErrLoginLockout,
			LockoutRemaining: time.Duration(seconds) * time.Second,
			Message:          "Locked out of device '" + id + "' " + text + "due to multiple incorrect password attempts."}
	case http.StatusRequestTimeout:
		msg := fmt.Sprintf("Login request timeout for %s and IP %s", id, c.IP)
		if ok, pingMsg := c.checkNetwork(ctx); !ok {
			msg = pingMsg
		}
		return &LoginError{StatusCode: status, Err: ErrLoginTimeout, Message: msg}
	case http.StatusServiceUnavailable:
		return &LoginError{StatusCode: status, Err: ErrDeviceUnreachable,
			Message: fmt.Sprintf("Device %s/%s is not reachable", id, c.IP)}
	default:
		return &LoginError{StatusCode: status, Err: ErrLoginUnknown, Message: "Unknown error"}
	}
}

// lockoutRemainingSeconds ports JsonConvert.DeserializeAnonymousType(content,
// new { lockout_remaining_seconds = 0 }): known is false when the body
// deserializes to null (empty or "null"), 0 when the field is absent.
func lockoutRemainingSeconds(content string) (seconds int64, known bool) {
	if strings.TrimSpace(content) == "" {
		return 0, false
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(content), &m); err != nil || m == nil {
		return 0, false
	}
	for k, raw := range m { // Newtonsoft matches property names case-insensitively
		if !strings.EqualFold(k, "lockout_remaining_seconds") {
			continue
		}
		var n json.Number
		if err := json.Unmarshal(raw, &n); err != nil {
			return 0, true
		}
		if i, err := n.Int64(); err == nil {
			return i, true
		}
		if f, err := n.Float64(); err == nil && !math.IsInf(f, 0) {
			return int64(math.RoundToEven(f)), true
		}
		return 0, true
	}
	return 0, true
}

// Authenticate ports the non-UI part of DlgDeviceLogin.OnOkClicked: store
// the credentials, run LoginRequest (LoginTimeout), and when the status is
// not 200 classify it with HandleUnsuccessfulLoginRequest(now, LoginTimeout,
// isUniquePasswordRequired: true). On success LastLoginStatus is set and the
// client is marked logged in; on failure it is marked logged out and the
// *LoginError is returned (res.Content is the C# LoginData.LoginError).
//
// The C# also pops the classification when a cached token logs in with a
// remembered non-200 status; that warning is not surfaced here.
func (c *Client) Authenticate(ctx context.Context, ld LoginData) (LoginResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	c.sess.mu.Lock()
	c.Creds = ld
	c.sess.mu.Unlock()
	res := c.LoginRequest(ctx, LoginTimeout)
	if res.LoggedIn {
		c.SetLastLoginStatus(res.StatusCode)
		c.SetLoggedIn(true)
		return res, nil
	}
	c.SetLoggedIn(false)
	return res, c.HandleUnsuccessfulLoginRequest(ctx, res.StatusCode, res.Content, time.Now(), LoginTimeout, true)
}

// LogoutContext ports ICULanDevice.Logout(forceClose): over HTTPS, when a
// request was made since the last logout and nothing is uploading, it POSTs
// /api/logout (5 s, 1 attempt). Tokens are cleared and idle connections
// closed (DisposeHttpClient) when that succeeded or forceClose is set. The
// client is always marked logged out; the result is whether the logout
// request succeeded.
func (c *Client) LogoutContext(ctx context.Context, forceClose bool) bool {
	c.sess.mu.Lock()
	c.sess.defaultCategoryCollected = false
	active, last, uploading := c.sess.httpActive, c.sess.lastCommand, c.sess.uploading
	c.sess.mu.Unlock()
	ok := false
	if c.isHTTPS() && active && strings.ToLower(last) != "logout" && !uploading {
		st, _, _ := c.ExecuteWebRequest(ctx, "logout", "", "", ExecOptions{Timeout: RequestTimeout, MaxRetries: 1})
		ok = st == ValidResponse
	}
	if ok || forceClose {
		c.ClearTokens()
		c.HTTP.CloseIdleConnections()
		c.sess.mu.Lock()
		c.sess.httpActive = false
		c.sess.mu.Unlock()
	}
	c.SetLoggedIn(false)
	return ok
}
