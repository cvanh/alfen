package core

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"alfen/aceclient/internal/api"
)

// HTTP status codes the installer switches on (System.Net.HttpStatusCode).
const (
	StatusOK                  = 200
	StatusMultipleChoices     = 300
	StatusUnused              = 306 // HttpStatusCode.Unused: "unhandled exception"
	StatusBadRequest          = 400
	StatusUnauthorized        = 401
	StatusForbidden           = 403
	StatusNotFound            = 404
	StatusNotAcceptable       = 406 // invalid certificate
	StatusRequestTimeout      = 408
	StatusConflict            = 409
	StatusTooManyRequests     = 429
	StatusInternalServerError = 500
	StatusNotImplemented      = 501
	StatusServiceUnavailable  = 503
	StatusGatewayTimeout      = 504
	StatusHTTPVersionNotSupp  = 505
)

// Timeouts and retry counts from ICULanDevice.
const (
	RequestTimeout        = 5000 * time.Millisecond   // s_nRequestTimeout
	LoginTimeout          = 15000 * time.Millisecond  // s_nLoginTimeoutInMs / LoginTimeoutInSeconds
	ClassicLoginTimeout   = 5000 * time.Millisecond   // ClassicLogin(5000)
	MaxRequestRetries     = 4                         // s_maxRequestRetries
	FirmwareUploadTimeout = 900000 * time.Millisecond // s_nFirmwareUploadTimeout
	PropertyBatchSize     = 15                        // StoreProperties list.Take(15)
)

// ProtocolForPort ports ICULanDevice.IsHTTPS/Protocol: https exactly when the
// port is 443, plain http otherwise (Basic auth, ClassicLogin).
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICULanDevice.cs:271-282
func ProtocolForPort(port int) string {
	if port == 443 {
		return "https"
	}
	return "http"
}

// Session is one selected charger (ICUNetwork.ICULanDevice) plus its property
// dictionary. All device I/O is serialised through an internal lock, like
// ICULanDevice.ExecuteAndWait's m_connectionLock.
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICULanDevice.cs (class ICULanDevice)
type Session struct {
	Client *api.Client
	Props  *PropertyCache

	io sync.Mutex // m_connectionLock

	mu                       sync.RWMutex
	address                  string
	port                     int
	login                    api.LoginData
	identity                 string
	serialNumber             string
	hostName                 string
	numberOfSockets          int
	numberOfFeederCables     int
	loggedIn                 bool
	defaultCategoryCollected bool
	uploading                bool
	fw                       *Version
	lastStatus               int
}

// NewSession creates the api client for address:port. protocol "" selects
// ProtocolForPort(port) (the C# rule); tests may force "https" for an
// httptest TLS server on a random port. insecureTLS skips certificate
// verification (the installer instead pins its own CA, see CertificateStore).
func NewSession(address string, port int, insecureTLS bool, protocol string) *Session {
	c := api.New(address, port, insecureTLS)
	if protocol == "" {
		protocol = ProtocolForPort(port)
	}
	c.Protocol = protocol
	return &Session{
		Client:               c,
		Props:                NewPropertyCache(),
		address:              address,
		port:                 port,
		numberOfSockets:      1, // ReInitialize: NumberOfSockets = 1
		numberOfFeederCables: 1, // ReInitialize: NumberOfFeederCables = 1
	}
}

// Address returns the device IP (ICULanDevice.Address).
func (s *Session) Address() string { return s.address }

// Port returns the device port.
func (s *Session) Port() int { return s.port }

// IsHTTPS reports whether the session talks HTTPS.
func (s *Session) IsHTTPS() bool { return strings.EqualFold(s.Client.Protocol, "https") }

// SetIdentity records what discovery knows about the device (mDNS
// identity/serial/hostname, SCN station name); empty values are ignored.
func (s *Session) SetIdentity(identity, serialNumber, hostName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if identity != "" {
		s.identity = identity
	}
	if serialNumber != "" {
		s.serialNumber = serialNumber
	}
	if hostName != "" {
		s.hostName = hostName
	}
}

// SetSockets records NumberOfSockets / NumberOfFeederCables (mDNS "type" TXT).
func (s *Session) SetSockets(numberOfSockets, numberOfFeederCables int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if numberOfSockets > 0 {
		s.numberOfSockets = numberOfSockets
	}
	if numberOfFeederCables > 0 {
		s.numberOfFeederCables = numberOfFeederCables
	}
}

// NumberOfSockets returns ICUDevice.NumberOfSockets (1 unless discovery set it).
func (s *Session) NumberOfSockets() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.numberOfSockets
}

// SetFirmwareVersion records an mDNS "fwversion" (ReInitialize) so
// FirmwareVersion does not fall back to property 0x100A.
func (s *Session) SetFirmwareVersion(v Version) {
	s.mu.Lock()
	s.fw = &v
	s.mu.Unlock()
}

// Identification ports ICULanDevice.Identification (= Identity). When it is
// empty, UpdatePropertiesInternal fills it from property 8275
// (sysChargeBoxIdentity).
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICULanDevice.cs:500, 1929-1932
func (s *Session) Identification() string {
	s.mu.RLock()
	id := s.identity
	s.mu.RUnlock()
	if id == "" {
		return s.Props.String(8275, 0)
	}
	return id
}

// SerialNumber returns ICUDevice.SerialNumber (from the mDNS hostname suffix).
func (s *Session) SerialNumber() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.serialNumber
}

// HostName ports ICULanDevice.HostName. For a device without an mDNS
// announcement UpdatePropertiesInternal derives it once properties are read:
// (8272.Replace(' ','-').Replace('.','-') + "-" + 8273).ToLowerInvariant().
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICULanDevice.cs:1933-1946
func (s *Session) HostName() string {
	s.mu.RLock()
	h := s.hostName
	s.mu.RUnlock()
	if h != "" {
		return h
	}
	model := s.Props.String(8272, 0)
	serial := s.Props.String(8273, 0)
	if model == "" && serial == "" {
		return ""
	}
	model = strings.NewReplacer(" ", "-", ".", "-").Replace(model)
	return strings.ToLower(model + "-" + serial)
}

// IsAHP ports ICULanDevice.isAHP (Model starts with "AHWP" or "AHP").
func (s *Session) IsAHP() bool { return IsAHPHostName(s.HostName()) }

// IsAHPHostName ports ICULanDevice.isAHP via ICUDeviceModelExtension: the
// model string derived from the hostname keeps the hostname's normalised
// prefix (lower-cased, "icu" and a leading '-' stripped), so the AHP test
// reduces to that prefix starting with "ahwp" or "ahp".
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICULanDevice.cs:334-344, ICUDeviceModelExtension.cs
func IsAHPHostName(hostName string) bool {
	t := strings.ToLower(strings.TrimSpace(hostName))
	if strings.HasPrefix(t, "icu") {
		t = strings.TrimSpace(t[3:])
	}
	t = strings.TrimPrefix(t, "-")
	return strings.HasPrefix(t, "ahwp") || strings.HasPrefix(t, "ahp")
}

// IsAHPV2HostName ports ICULanDevice.isAHPV2 (Model starts with "AHP02" or
// "AHPDC").
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICULanDevice.cs:346-358
func IsAHPV2HostName(hostName string) bool {
	t := strings.ToLower(strings.TrimSpace(hostName))
	if strings.HasPrefix(t, "icu") {
		t = strings.TrimSpace(t[3:])
	}
	t = strings.TrimPrefix(t, "-")
	t = strings.ReplaceAll(strings.ReplaceAll(t, " ", "-"), "_", "-")
	return strings.HasPrefix(t, "ahp02") || strings.HasPrefix(t, "ahpdc")
}

// FirmwareVersion ports ICULanDevice.FirmwareVersionNumber.
func (s *Session) FirmwareVersion() Version {
	s.mu.RLock()
	fw := s.fw
	s.mu.RUnlock()
	if fw != nil {
		return *fw
	}
	return FirmwareVersionNumber(s.Props.String(4106, 0), s.IsHTTPS())
}

// IsUniquePasswordRequired ports ICULanDevice.IsUniquePasswordRequired:
// firmware >= 5 (as known from mDNS / a parsed 0x100A) or an AHP model.
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICULanDevice.cs:384-401
func (s *Session) IsUniquePasswordRequired() bool {
	s.mu.RLock()
	fw := s.fw
	s.mu.RUnlock()
	if fw == nil {
		if s.Props.Has(4106, 0) {
			v := s.FirmwareVersion()
			fw = &v
		}
	}
	if fw == nil || fw.Major < 5 {
		return s.IsAHP()
	}
	return true
}

// LoginData returns the credentials of the last login attempt.
func (s *Session) LoginData() api.LoginData {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.login
}

// IsLoggedIn ports ICULanDevice.IsLoggedIn.
func (s *Session) IsLoggedIn() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loggedIn
}

func (s *Session) setLoggedIn(v bool) {
	s.mu.Lock()
	s.loggedIn = v
	if !v {
		s.defaultCategoryCollected = false // IsDefaultCategoryCollected = false
	}
	s.mu.Unlock()
}

// LastHTTPStatus ports ICULanDevice.LastHttpStatusCode.
func (s *Session) LastHTTPStatus() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastStatus
}

// withTimeout runs fn with the client's timeout temporarily set to d. Callers
// hold s.io, so no other request observes the change.
func (s *Session) withTimeout(d time.Duration, fn func() error) error {
	hc := s.Client.HTTP
	if hc == nil {
		return fn()
	}
	old := hc.Timeout
	hc.Timeout = d
	defer func() { hc.Timeout = old }()
	return fn()
}

// ---------------------------------------------------------------- login

// LoginResult mirrors the (bool IsLoggedIn, HttpStatusCode, string Content)
// tuple of ICULanDevice.LoginRequest.
type LoginResult struct {
	LoggedIn   bool
	StatusCode int
	Content    string
}

// apiStatusRe extracts the status code internal/api embeds in its errors
// ("login failed: HTTP 403: …", "prop read HTTP 401: …").
var apiStatusRe = regexp.MustCompile(`(?s)HTTP (\d{3})(?::\s?(.*))?`)

// StatusFromError maps an internal/api error to the HttpStatusCode the C#
// would have observed: the embedded HTTP status for non-2xx replies; for
// transport failures the codes ExecuteWebRequestInternal/LoginRequest assign
// to exceptions — timeouts and socket errors → 408 RequestTimeout, an invalid
// certificate (AuthenticationException) → 406 NotAcceptable, anything else →
// 306 Unused. content is the response body (or exception text).
//
// An error that is neither a transport failure nor an HTTP status (e.g. a JSON
// decode error after a 2xx reply) yields 0: the C# saw a VALID_RESPONSE and
// only the parsing failed.
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICULanDevice.cs:966-1060 (ExecuteWebRequestInternal catch blocks)
func StatusFromError(err error) (status int, content string) {
	if err == nil {
		return StatusOK, ""
	}
	var ue *url.Error
	if errors.As(err, &ue) {
		var ce *tls.CertificateVerificationError
		var ua x509.UnknownAuthorityError
		var hn x509.HostnameError
		var ci x509.CertificateInvalidError
		switch {
		case errors.As(err, &ce), errors.As(err, &ua), errors.As(err, &hn), errors.As(err, &ci):
			return StatusNotAcceptable, err.Error()
		case ue.Timeout():
			return StatusRequestTimeout, "RequestTimeout"
		}
		var oe *net.OpError
		if errors.As(err, &oe) {
			return StatusRequestTimeout, ""
		}
		return StatusUnused, err.Error()
	}
	if m := apiStatusRe.FindStringSubmatch(err.Error()); m != nil {
		n, _ := strconv.Atoi(m[1])
		return n, m[2]
	}
	return 0, err.Error()
}

// loginStatusFromError is LoginRequest's catch mapping: TaskCanceledException
// → (408, "RequestTimeout"); HttpRequestException wrapping a SocketException →
// (408, ""); other HttpRequestException → (503, message).
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICULanDevice.cs:1714-1732
func loginStatusFromError(err error) (int, string) {
	var ue *url.Error
	if errors.As(err, &ue) {
		if ue.Timeout() {
			return StatusRequestTimeout, "RequestTimeout"
		}
		var oe *net.OpError
		if errors.As(err, &oe) {
			return StatusRequestTimeout, ""
		}
		return StatusServiceUnavailable, err.Error()
	}
	st, content := StatusFromError(err)
	if st == 0 {
		// catch (Exception ex4) → (false, InternalServerError, ex4.Message)
		return StatusInternalServerError, content
	}
	return st, content
}

// LoginRequest ports ICULanDevice.LoginRequest (HTTPS) / ClassicLogin (HTTP).
//
// HTTPS: IsAccessTokenAvailableAsync first (refresh when only a refresh token
// is held), else POST /api/login {"username","password"(Trim('"')),
// "displayname"} with a 15 s budget; on 2xx the tokens are stored.
// HTTP: up to three POST /api/login attempts, 5 s each, 1 s apart.
//
// Not ported: ClassicLogin's third attempt swaps in the legacy fleet
// credentials ICUNetworkConfig.HTTPUsernameOld/HTTPPasswordOld; those are
// secrets compiled into the original binary and are deliberately omitted.
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICULanDevice.cs:1663-1768
func (s *Session) LoginRequest(ld api.LoginData) LoginResult {
	s.io.Lock()
	defer s.io.Unlock()
	s.mu.Lock()
	s.login = ld
	s.mu.Unlock()
	res := s.loginLocked(ld)
	s.mu.Lock()
	s.lastStatus = res.StatusCode
	s.mu.Unlock()
	s.setLoggedIn(res.LoggedIn)
	return res
}

func (s *Session) loginLocked(ld api.LoginData) LoginResult {
	c := s.Client
	if s.IsHTTPS() {
		// IsAccessTokenAvailableAsync
		if c.AccessToken == "" && c.RefreshToken != "" {
			_ = s.withTimeout(LoginTimeout, c.RefreshSession)
		}
		if c.AccessToken != "" {
			return LoginResult{LoggedIn: true, StatusCode: s.LastHTTPStatus()}
		}
		ld.Password = strings.Trim(ld.Password, `"`)
		var err error
		_ = s.withTimeout(LoginTimeout, func() error { err = c.Login(ld); return nil })
		if err != nil {
			st, content := loginStatusFromError(err)
			return LoginResult{StatusCode: st, Content: content}
		}
		return LoginResult{LoggedIn: true, StatusCode: StatusOK}
	}
	last := StatusServiceUnavailable
	for retry := 0; retry < 3; retry++ {
		var err error
		_ = s.withTimeout(ClassicLoginTimeout, func() error { err = c.Login(ld); return nil })
		if err == nil {
			return LoginResult{LoggedIn: true, StatusCode: StatusOK, Content: "OK"}
		}
		var ue *url.Error
		if errors.As(err, &ue) {
			// exceptions escape ClassicLogin into LoginRequest's catch blocks
			st, content := loginStatusFromError(err)
			return LoginResult{StatusCode: st, Content: content}
		}
		last, _ = StatusFromError(err)
		time.Sleep(loginRetryDelay)
	}
	return LoginResult{StatusCode: last}
}

// loginRetryDelay is ClassicLogin's Task.Delay(1000); a var for tests.
var loginRetryDelay = time.Second

// LoginErrorMessage ports ICULanDevice.HandleUnsuccessfulLoginRequest for
// isUniquePasswordRequired == true (the only way DlgDeviceLogin calls it): the
// popup text for a failed login. CheckNetworkAndPing (ICMP ping + adapter
// listing) is not ported, so a 408 always yields the timeout text.
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICULanDevice.cs:1264-1350
func LoginErrorMessage(status int, content, identification, ip string) string {
	switch status {
	case StatusUnauthorized:
		return "Secure Service Access is disabled for device '" + identification + "'"
	case StatusForbidden:
		return "The password for device '" + identification + "' is incorrect! Please provide a valid password."
	case StatusTooManyRequests:
		num, ok, failed := lockoutMinutes(content)
		if failed {
			// JsonConvert throws inside the login task; DlgDeviceLogin wraps it
			// in AuthenticationException("Login failed").
			return "Login failed"
		}
		text2 := ""
		if ok && num > 0 {
			arg := "minutes"
			if num == 1.0 {
				arg = "minute"
			}
			text2 = fmt.Sprintf("for %.1f %s ", num, arg)
		}
		return "Locked out of device '" + identification + "' " + text2 + "due to multiple incorrect password attempts."
	case StatusRequestTimeout:
		return fmt.Sprintf("Login request timeout for %s and IP %s", identification, ip)
	case StatusServiceUnavailable:
		return fmt.Sprintf("Device %s/%s is not reachable", identification, ip)
	default:
		return "Unknown error"
	}
}

// lockoutMinutes ports JsonConvert.DeserializeAnonymousType(content,
// new { lockout_remaining_seconds = 0 })?.lockout_remaining_seconds / 60.0.
// ok is false when the result is null (empty content); failed is true when
// Json.NET would throw.
func lockoutMinutes(content string) (minutes float64, ok, failed bool) {
	if strings.TrimSpace(content) == "" || strings.TrimSpace(content) == "null" {
		return 0, false, false
	}
	var v struct {
		Lockout *json.Number `json:"lockout_remaining_seconds"`
	}
	dec := json.NewDecoder(strings.NewReader(content))
	dec.UseNumber()
	if err := dec.Decode(&v); err != nil {
		return 0, false, true
	}
	if v.Lockout == nil {
		return 0, true, false
	}
	n, err := strconv.ParseInt(v.Lockout.String(), 10, 32)
	if err != nil {
		return 0, false, true
	}
	return float64(n) / 60.0, true, false
}

// Logout ports ICULanDevice.Logout: over HTTPS a POST /api/logout (5 s, one
// attempt, skipped while uploading); IsLoggedIn becomes false either way.
// It reports whether the device acknowledged the logout.
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICULanDevice.cs:1770-1794
func (s *Session) Logout() bool {
	s.io.Lock()
	defer s.io.Unlock()
	ok := false
	s.mu.RLock()
	uploading := s.uploading
	s.mu.RUnlock()
	if s.IsHTTPS() && !uploading {
		var err error
		_ = s.withTimeout(RequestTimeout, func() error { err = s.Client.Logout(); return nil })
		ok = err == nil
	}
	s.setLoggedIn(false)
	return ok
}

// --------------------------------------------------- ExecuteWebRequest port

// RequestError is a failed ExecuteWebRequest: the final status code plus the
// popup text HandleUnsuccessfulRequest would raise through
// LANConnection.CallErrorHandler (empty when the C# stays silent).
type RequestError struct {
	Command    string
	StatusCode int
	Message    string
	Err        error
}

func (e *RequestError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("%s: HTTP %d", e.Command, e.StatusCode)
}

func (e *RequestError) Unwrap() error { return e.Err }

// execute ports ExecuteWebRequestInternal's retry loop with
// HandleUnsuccessfulRequest's per-status policy around one internal/api call:
//
//   - 401/403/429: logged out; unless the command is login/logout/prc or the
//     status is 429, re-login with the stored credentials and RETRY on success.
//   - 400/501: "does not support the request" on the last retry.
//   - 406: "invalid or expired certificate" on the last retry.
//   - 300/409/500/503: logged out; "connection lost" on the last retry.
//   - 306/408/504: logged out; retry until the last attempt (skipped while
//     uploading). CheckNetworkAndPing is not ported.
//   - anything else: retry.
//
// The caller's fn runs with the request timeout applied (min 5 s, as the C#).
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICULanDevice.cs:945-1060
func (s *Session) execute(command string, timeout time.Duration, maxRetries int, suppressPopups bool, fn func(c *api.Client) error) error {
	s.io.Lock()
	defer s.io.Unlock()
	if timeout < RequestTimeout {
		timeout = RequestTimeout
	}
	var last *RequestError
	for retry := 0; retry < maxRetries; retry++ {
		var err error
		_ = s.withTimeout(timeout, func() error { err = fn(s.Client); return nil })
		if err == nil {
			s.mu.Lock()
			s.lastStatus = StatusOK
			s.mu.Unlock()
			return nil
		}
		status, _ := StatusFromError(err)
		if status == 0 || status == StatusNotFound {
			// HandleIncomingResponse: 404 and parsable replies are a
			// VALID_RESPONSE; only the caller's parsing failed. No retry.
			return &RequestError{Command: command, StatusCode: status, Err: err}
		}
		s.mu.Lock()
		s.lastStatus = status
		s.mu.Unlock()
		retryAgain, msg := s.handleUnsuccessful(command, status, retry >= maxRetries-1, suppressPopups)
		last = &RequestError{Command: command, StatusCode: status, Message: msg, Err: err}
		if !retryAgain {
			break
		}
	}
	return last
}

// handleUnsuccessful ports HandleUnsuccessfulRequest. It runs with s.io held.
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICULanDevice.cs:1166-1262
func (s *Session) handleUnsuccessful(command string, status int, isLastRetry, suppressPopups bool) (retry bool, msg string) {
	ident := s.Identification()
	popup := func(m string) string {
		if isLastRetry && !suppressPopups {
			return m
		}
		return ""
	}
	switch status {
	case StatusUnauthorized, StatusForbidden, StatusTooManyRequests:
		s.setLoggedIn(false)
		s.Client.AccessToken = ""
		if command != "login" && command != "logout" && command != "prc" && status != StatusTooManyRequests {
			s.mu.RLock()
			ld := s.login
			s.mu.RUnlock()
			if res := s.loginLocked(ld); res.LoggedIn {
				s.setLoggedIn(true)
				return true, ""
			}
		}
		return false, ""
	case StatusBadRequest, StatusNotImplemented:
		return false, popup("Device '" + ident + "' does not support the request, please contact support.")
	case StatusNotAcceptable:
		return false, popup("Device '" + ident + "' has an invalid or expired certificate, please contact support.")
	case StatusMultipleChoices, StatusConflict, StatusInternalServerError, StatusServiceUnavailable:
		s.setLoggedIn(false)
		return false, popup(fmt.Sprintf("Device '%s'/%s connection lost.", ident, s.address))
	case StatusUnused, StatusRequestTimeout, StatusGatewayTimeout:
		s.setLoggedIn(false)
		s.mu.RLock()
		uploading := s.uploading
		s.mu.RUnlock()
		if uploading {
			return false, ""
		}
		return !isLastRetry, ""
	default:
		return true, ""
	}
}

// ------------------------------------------------------------ categories

// RequestCategories ports ICULanDevice.RequestCategories: GET /api/categories
// and the "categories" array, each entry Trim('"', ' '). Failures yield an
// empty list, as in the C#.
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICULanDevice.cs:1464-1500
func (s *Session) RequestCategories() []string {
	var body string
	err := s.execute("categories", RequestTimeout, MaxRequestRetries, false, func(c *api.Client) error {
		resp, err := c.Exec("categories", "", nil)
		if err != nil {
			return err
		}
		if !resp.OK() {
			return fmt.Errorf("categories HTTP %d: %s", resp.StatusCode, resp.Body)
		}
		body = resp.Body
		return nil
	})
	if err != nil || strings.TrimSpace(body) == "" {
		return nil
	}
	return parseCategories(body)
}

func parseCategories(body string) []string {
	var top map[string]json.RawMessage
	if err := json.Unmarshal([]byte(body), &top); err != nil {
		return nil
	}
	raw, ok := top["categories"]
	if !ok {
		return nil
	}
	var items []any
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil
	}
	var out []string
	for _, it := range items {
		if it == nil {
			continue
		}
		out = append(out, strings.Trim(fmt.Sprint(it), `" `))
	}
	return out
}

// UpdateCategories ports ICULanDevice.UpdateCategories: for each category
// (all of RequestCategories when none are given) GET /api/prop?cat=<c>,
// appending "&limit=500" on AHP firmware >= 2.2; an AHP >= 2.2 asked for all
// categories reads "limit=500" in one go. It stops at the first failure.
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICULanDevice.cs:1443-1462
func (s *Session) UpdateCategories(categories ...string) error {
	flag := s.IsAHP() && s.FirmwareVersion().AtLeast(NewVersion(2, 2))
	if len(categories) == 0 && flag {
		return s.updatePropertiesInternal("limit=500")
	}
	if len(categories) == 0 {
		categories = s.RequestCategories()
	}
	for _, c := range categories {
		p := "cat=" + c
		if flag {
			p += "&limit=500"
		}
		if err := s.updatePropertiesInternal(p); err != nil {
			return err
		}
	}
	return nil
}

// UpdateProperties ports ICULanDevice.UpdateProperties(string sIds): GET
// /api/prop?ids=<comma-joined ODIndex list>.
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICULanDevice.cs:1796-1803
func (s *Session) UpdateProperties(ids ...string) error {
	return s.updatePropertiesInternal("ids=" + strings.Join(ids, ","))
}

// updatePropertiesInternal ports UpdatePropertiesInternal: the paged read
// (ExecuteWebRequest("prop", parameters, null, 5000, 2)) merged into the
// dictionary via ParseProperty.
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICULanDevice.cs:1862-1953
func (s *Session) updatePropertiesInternal(parameters string) error {
	var props []api.Property
	err := s.execute("prop", RequestTimeout, 2, false, func(c *api.Client) error {
		var err error
		props, err = c.ReadProperties(parameters)
		return err
	})
	if err != nil {
		return err
	}
	s.Props.Merge(props)
	return nil
}

// --------------------------------------------------------------- storing

// StoreProperties ports ICULanDevice.StoreProperties: batches of 15 per POST
// /api/prop; each batch that the device accepts is committed (CommitChange)
// before the next is sent; the first failing batch aborts.
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICULanDevice.cs:1955-2036
func (s *Session) StoreProperties(props ...Property) error {
	for len(props) > 0 {
		n := PropertyBatchSize
		if len(props) < n {
			n = len(props)
		}
		batch := props[:n]
		props = props[n:]
		wire := make([]api.Property, len(batch))
		keys := make([]uint32, len(batch))
		for i, p := range batch {
			wire[i] = s.Props.WireProperty(p)
			keys[i] = p.Key()
		}
		err := s.execute("prop", RequestTimeout, MaxRequestRetries, false, func(c *api.Client) error {
			return c.StoreProperties(wire...)
		})
		if err != nil {
			return err
		}
		s.Props.Commit(keys...)
	}
	return nil
}

// StoreProperty ports ICULanDevice.storeProperty(id, sub, value): set the
// value and store it only when that changed it. It reports whether a store
// was attempted successfully.
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICULanDevice.cs:2038-2050
func (s *Session) StoreProperty(id uint16, sub byte, value string) (bool, error) {
	p, err := s.Props.SetValue(id, sub, value)
	if err != nil {
		return false, err
	}
	if !p.Changed {
		return false, nil
	}
	if err := s.StoreProperties(p); err != nil {
		return false, err
	}
	return true, nil
}

// MarkUploading flags an in-progress firmware upload (ICULanDevice.IsUploading).
func (s *Session) MarkUploading(v bool) {
	s.mu.Lock()
	s.uploading = v
	s.mu.Unlock()
}

// AfterLogin ports the logged-in branch of MainWindow.OnDeviceSelectionChanged:
// collect the default categories "generic" and "generic2" once per login
// (IsDefaultCategoryCollected).
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs:1038-1240 (OnDeviceSelectionChanged)
func (s *Session) AfterLogin() error {
	s.mu.RLock()
	done := s.defaultCategoryCollected
	s.mu.RUnlock()
	if done {
		return nil
	}
	if err := s.UpdateCategories("generic", "generic2"); err != nil {
		return err
	}
	s.mu.Lock()
	s.defaultCategoryCollected = true
	s.mu.Unlock()
	return nil
}
