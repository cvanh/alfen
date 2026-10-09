package api

import (
	"context"
	"strconv"
	"sync"
	"time"
)

// Timing constants of ICULanDevice (ACENetwork/ICUNetwork/ICULanDevice.cs).
const (
	// RequestTimeout ports s_nRequestTimeout (5000 ms), the default and the
	// minimum per-attempt timeout of ExecuteWebRequest.
	RequestTimeout = 5000 * time.Millisecond
	// LoginTimeout ports s_nLoginTimeoutInMs / LoginTimeoutInSeconds (15 s).
	LoginTimeout = 15000 * time.Millisecond
	// MaxRequestRetries ports s_maxRequestRetries (4).
	MaxRequestRetries = 4
	// RequestLongTimeout ports s_nRequestLongTimeOut (10000 ms).
	RequestLongTimeout = 10000 * time.Millisecond
	// FirmwareUploadTimeout ports s_nFirmwareUploadTimeout (900000 ms).
	FirmwareUploadTimeout = 900000 * time.Millisecond
	// FirmwareUploadResetTimeout ports s_nFirmwareUploadResetTimeout (170000 ms).
	FirmwareUploadResetTimeout = 170000 * time.Millisecond
	// ClassicLoginTimeout is the loginTimeoutInMs LoginRequest passes to
	// ClassicLogin (5000 ms).
	ClassicLoginTimeout = 5000 * time.Millisecond
)

// HTTP status codes ICULanDevice uses that net/http does not name.
const (
	// StatusUnused ports System.Net.HttpStatusCode.Unused (306), which
	// ExecuteWebRequest never assigns on net481 but HandleUnsuccessfulRequest
	// still groups with the timeouts.
	StatusUnused = 306
	// StatusLoginLockout ports CustomHttpStatusCode.LoginLockout (429).
	StatusLoginLockout = 429
)

// minRequestTimeout is the clamp `if (requestTimeoutInMs < 5000)
// requestTimeoutInMs = 5000;` in ExecuteWebRequestInternal (tests lower it).
var minRequestTimeout = RequestTimeout

// classicLoginRetryDelay is the `await Task.Delay(1000)` between ClassicLogin
// attempts (tests lower it).
var classicLoginRetryDelay = time.Second

// RequestState ports ICUNetwork.EWebRequestState
// (ACENetwork/ICUNetwork/EWebRequestState.cs).
type RequestState int

// EWebRequestState values, same order and numbering as the C# enum.
const (
	ValidResponse   RequestState = 0 // VALID_RESPONSE
	InvalidResponse RequestState = 1 // INVALID_RESPONSE
	RetryRequest    RequestState = 2 // RETRY_REQUEST
	Unsuccessful    RequestState = 3 // UNSUCCESSFUL
)

// String returns the C# enum member name.
func (s RequestState) String() string {
	switch s {
	case ValidResponse:
		return "VALID_RESPONSE"
	case InvalidResponse:
		return "INVALID_RESPONSE"
	case RetryRequest:
		return "RETRY_REQUEST"
	case Unsuccessful:
		return "UNSUCCESSFUL"
	}
	return "EWebRequestState(" + strconv.Itoa(int(s)) + ")"
}

// ProtocolForPort ports ICULanDevice.IsHTTPS/Protocol: "https" when the port
// is 443, "http" otherwise. Assign it to Client.Protocol for devices that are
// not on 443.
func ProtocolForPort(port int) string {
	if port == 443 {
		return "https"
	}
	return "http"
}

// NetworkCheck is the signature of CheckNetworkAndPing, injectable per Client
// via SetNetworkCheck.
type NetworkCheck func(ctx context.Context, ip string, port int) (ok bool, message string)

// session holds the ICULanDevice fields that ExecuteWebRequest, LoginRequest
// and HandleUnsuccessfulRequest read and write. The C# guards some of them
// with ReadLocked/WriteLocked (ReaderWriterLockSlim); here one mutex guards
// all of them plus the token/credential fields the new code touches.
type session struct {
	mu sync.Mutex

	loggedIn                 bool // LoginData.IsLoggedIn (IsLoggedIn)
	rebooting                bool // IsRebooting
	uploading                bool // IsUploading
	defaultCategoryCollected bool // IsDefaultCategoryCollected
	uniquePasswordGrace      bool // m_fUniquePasswordGracePeriod
	httpActive               bool // m_httpClient != null

	lastStatus        int       // LastHttpStatusCode
	lastLoginStatus   int       // LoginData.LastHttpStatusCode
	lastValidResponse time.Time // LastValidResponse
	lastCommand       string    // LastCommand

	classicFallback *LoginData   // ICUNetworkConfig.HTTPUsernameOld/HTTPPasswordOld, caller supplied
	netCheck        NetworkCheck // CheckNetworkAndPing override
}

func (c *Client) lock() func() {
	c.sess.mu.Lock()
	return c.sess.mu.Unlock
}

// IsLoggedIn ports ICULanDevice.IsLoggedIn (LoginData.IsLoggedIn).
func (c *Client) IsLoggedIn() bool { defer c.lock()(); return c.sess.loggedIn }

// SetLoggedIn ports the IsLoggedIn / LoginData.IsLoggedIn setter used by
// DlgDeviceLogin, MainWindow.OnLoginRequest and the firmware-upload loop.
func (c *Client) SetLoggedIn(v bool) { defer c.lock()(); c.sess.loggedIn = v }

// IsRebooting ports ICULanDevice.IsRebooting.
func (c *Client) IsRebooting() bool { defer c.lock()(); return c.sess.rebooting }

// SetRebooting ports the IsRebooting setter (Reboot / StartUpload).
func (c *Client) SetRebooting(v bool) { defer c.lock()(); c.sess.rebooting = v }

// IsUploading ports ICULanDevice.IsUploading.
func (c *Client) IsUploading() bool { defer c.lock()(); return c.sess.uploading }

// SetUploading ports the IsUploading setter (StartUpload).
func (c *Client) SetUploading(v bool) { defer c.lock()(); c.sess.uploading = v }

// IsDefaultCategoryCollected ports ICULanDevice.IsDefaultCategoryCollected.
func (c *Client) IsDefaultCategoryCollected() bool {
	defer c.lock()()
	return c.sess.defaultCategoryCollected
}

// SetDefaultCategoryCollected ports the IsDefaultCategoryCollected setter.
func (c *Client) SetDefaultCategoryCollected(v bool) {
	defer c.lock()()
	c.sess.defaultCategoryCollected = v
}

// LastHTTPStatus ports ICULanDevice.LastHttpStatusCode: the status of the
// last ExecuteWebRequest (synthetic 408/406/500 when no response arrived).
// Zero until the first request, like default(HttpStatusCode).
func (c *Client) LastHTTPStatus() int { defer c.lock()(); return c.sess.lastStatus }

// LastValidResponse ports ICULanDevice.LastValidResponse (UTC).
func (c *Client) LastValidResponse() time.Time { defer c.lock()(); return c.sess.lastValidResponse }

// LastCommand ports ICULanDevice.LastCommand.
func (c *Client) LastCommand() string { defer c.lock()(); return c.sess.lastCommand }

// LastLoginStatus ports ACEWebLoginData.LastHttpStatusCode.
func (c *Client) LastLoginStatus() int { defer c.lock()(); return c.sess.lastLoginStatus }

// SetLastLoginStatus ports the ACEWebLoginData.LastHttpStatusCode setter
// (DlgDeviceLogin stores the login status after a successful login).
func (c *Client) SetLastLoginStatus(status int) { defer c.lock()(); c.sess.lastLoginStatus = status }

// SetUniquePasswordGracePeriod ports m_fUniquePasswordGracePeriod, which
// StartUpload raises while a 4.x -> 5.x firmware upload sets a new password.
func (c *Client) SetUniquePasswordGracePeriod(v bool) {
	defer c.lock()()
	c.sess.uniquePasswordGrace = v
}

// ShouldLoginUsingUniquePassword ports ICULanDevice.ShouldLoginUsingUniquePassword:
// false during the grace period, otherwise UniquePasswordRequired.
func (c *Client) ShouldLoginUsingUniquePassword(fwMajor int, isAHP bool) bool {
	c.sess.mu.Lock()
	grace := c.sess.uniquePasswordGrace
	c.sess.mu.Unlock()
	if grace {
		return false
	}
	return UniquePasswordRequired(fwMajor, isAHP)
}

// SetClassicLoginFallback supplies the credentials ClassicLogin switches to
// on its third attempt (ICUNetworkConfig.HTTPUsernameOld/HTTPPasswordOld in
// the C#). They are deliberately not compiled into this package; without
// them the third attempt reuses the current credentials.
func (c *Client) SetClassicLoginFallback(ld LoginData) {
	defer c.lock()()
	c.sess.classicFallback = &ld
}

// SetNetworkCheck overrides CheckNetworkAndPing for this client (nil restores
// the default).
func (c *Client) SetNetworkCheck(fn NetworkCheck) { defer c.lock()(); c.sess.netCheck = fn }

// FinishedRebooting ports the connection-state part of
// ICULanDevice.FinishedRebooting: IsRebooting = false, LastHttpStatusCode = OK.
func (c *Client) FinishedRebooting() {
	defer c.lock()()
	c.sess.rebooting = false
	c.sess.lastStatus = 200
}

// Allocate ports the connection-state part of ICULanDevice.Allocate:
// LastValidResponse = now, LastHttpStatusCode = OK.
func (c *Client) Allocate() {
	defer c.lock()()
	c.sess.lastValidResponse = time.Now().UTC()
	c.sess.lastStatus = 200
}

// ClearTokens ports ICULanDevice.ClearTokens.
func (c *Client) ClearTokens() {
	defer c.lock()()
	c.AccessToken = ""
	c.RefreshToken = ""
}

func (c *Client) tokens() (access, refresh string) {
	defer c.lock()()
	return c.AccessToken, c.RefreshToken
}

func (c *Client) creds() LoginData {
	defer c.lock()()
	return c.Creds
}

func (c *Client) checkNetwork(ctx context.Context) (bool, string) {
	c.sess.mu.Lock()
	fn := c.sess.netCheck
	c.sess.mu.Unlock()
	if fn == nil {
		fn = CheckNetworkAndPing
	}
	return fn(ctx, c.IP, c.Port)
}

// resetConnectionFlags is the `IsDefaultCategoryCollected = false;
// IsLoggedIn = false;` pair HandleUnsuccessfulRequest repeats.
func (c *Client) resetConnectionFlags() {
	defer c.lock()()
	c.sess.defaultCategoryCollected = false
	c.sess.loggedIn = false
}
