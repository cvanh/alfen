package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ExecOptions carries the optional ExecuteWebRequest arguments. Zero values
// select the C# defaults.
type ExecOptions struct {
	// Timeout is requestTimeoutInMs: the per-attempt timeout. Values below
	// 5 s (including zero) are raised to 5 s, as the C# does.
	Timeout time.Duration
	// MaxRetries is maxRetries: the number of attempts. Zero or negative
	// selects the C# default of 4.
	MaxRetries int
}

// Error classes of a failed ExecuteWebRequest. A *RequestError unwraps to
// exactly one of them (plus the transport error, if any), so callers can
// branch with errors.Is.
var (
	// ErrUnauthorized: 401/403/429 and the automatic re-login failed or was
	// not attempted (commands "login", "logout", "prc", or status 429).
	ErrUnauthorized = errors.New("api: unauthorized")
	// ErrUnsupportedRequest: 400/501 ("does not support the request").
	ErrUnsupportedRequest = errors.New("api: request not supported by device")
	// ErrInvalidCertificate: 406, the device certificate failed validation.
	ErrInvalidCertificate = errors.New("api: invalid or expired device certificate")
	// ErrConnectionLost: 300/409/500/503, or a transport error with no
	// response (status left at 500 on net481).
	ErrConnectionLost = errors.New("api: connection lost")
	// ErrRequestTimeout: 306/408/504, i.e. timeouts and cancellations.
	ErrRequestTimeout = errors.New("api: request timeout")
	// ErrDeviceRebooting: the device is rebooting and the previous request
	// already timed out, so HandleUnsuccessfulRequest gives up immediately.
	ErrDeviceRebooting = errors.New("api: device is rebooting")
	// ErrRetriesExhausted: every attempt got an unclassified status (the
	// C# default branch keeps answering RETRY_REQUEST).
	ErrRetriesExhausted = errors.New("api: retries exhausted")
)

// RequestError reports an ExecuteWebRequest whose final state is not
// ValidResponse. It replaces the LANConnection.CallErrorHandler popups.
type RequestError struct {
	Command    string
	State      RequestState
	StatusCode int
	// Message is the exact text HandleUnsuccessfulRequest passes to
	// LANConnection.CallErrorHandler. The C# only shows it on the last retry
	// and when suppressPopups is false; Message is filled under the same
	// last-retry rule and left to the caller to show or suppress. Empty when
	// the C# shows nothing.
	Message string
	// Err is the transport error of the last attempt, if there was one.
	Err error

	class error
}

func (e *RequestError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "api: %s: %s (HTTP %d)", e.Command, e.State, e.StatusCode)
	if e.Message != "" {
		b.WriteString(": ")
		b.WriteString(e.Message)
	}
	if e.Err != nil {
		b.WriteString(": ")
		b.WriteString(e.Err.Error())
	}
	return b.String()
}

// Unwrap exposes the error class (ErrUnauthorized, ...) and the transport
// error to errors.Is / errors.As.
func (e *RequestError) Unwrap() []error {
	out := make([]error, 0, 2)
	if e.class != nil {
		out = append(out, e.class)
	}
	if e.Err != nil {
		out = append(out, e.Err)
	}
	return out
}

// ExecuteWebRequest ports ICULanDevice.ExecuteWebRequest /
// ExecuteWebRequestInternal (ACENetwork/ICUNetwork/ICULanDevice.cs).
//
// body selects the request like SendHttpRequest: nil -> GET; string -> POST
// application/json; []byte -> POST multipart/form-data ("firmwarefile").
//
// Each attempt gets its own timeout (opt.Timeout, min 5 s) linked to ctx. A
// response goes through HandleIncomingResponse (404 counts as a valid empty
// response; ":nan" becomes ":null"); a non-valid outcome goes through
// HandleUnsuccessfulRequest, which decides between RETRY_REQUEST and
// UNSUCCESSFUL, may log in again (401/403) and may check the network
// (timeouts). Exceptions map like on net481: a cancelled/timed-out attempt
// becomes 408 RequestTimeout, a certificate rejection 406 NotAcceptable, any
// other transport error keeps the previous status (500 on the first attempt).
//
// The returned Response carries the final (possibly synthetic) status and the
// last body received. err is nil exactly when the state is ValidResponse,
// otherwise a *RequestError. LastHTTPStatus, LastCommand and
// LastValidResponse are updated as the C# does after the call.
//
// Deviation: when ctx itself is cancelled the loop stops after the current
// attempt with Unsuccessful/408, the tuple the C# reaches only after running
// through its remaining (immediately failing) retries.
func (c *Client) ExecuteWebRequest(ctx context.Context, command, parameters string, body any, opt ExecOptions) (RequestState, Response, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	timeout := opt.Timeout
	if timeout < minRequestTimeout {
		timeout = minRequestTimeout
	}
	maxRetries := opt.MaxRetries
	if maxRetries <= 0 {
		maxRetries = MaxRequestRetries
	}

	state := Unsuccessful
	status := http.StatusInternalServerError
	data := ""
	message := ""
	var class, lastErr error

	uri, uriErr := c.buildURI(command, parameters)
	for retry := 0; retry < maxRetries; retry++ {
		lastErr = nil
		actx, cancel := context.WithTimeout(ctx, timeout)
		st, text, err := c.roundTrip(actx, uri, uriErr, body)
		if err == nil {
			status = st
			state, data = handleIncomingResponse(st, text)
		} else {
			lastErr = err
			switch {
			case isCertificateError(err):
				status = http.StatusNotAcceptable
			case actx.Err() != nil || isTimeout(err):
				status = http.StatusRequestTimeout
			}
		}
		cancel()
		if state != ValidResponse {
			isLastRetry := retry >= maxRetries-1
			state, message, class = c.handleUnsuccessfulRequest(ctx, command, status, isLastRetry)
		}
		if state != RetryRequest {
			break
		}
		if ctx.Err() != nil {
			// The C# keeps looping: every further attempt fails at once with
			// OperationCanceledException -> 408 -> IsLoggedIn = false, and the
			// last one ends UNSUCCESSFUL. Jump straight to that outcome.
			c.resetConnectionFlags()
			state, status, message, class = Unsuccessful, http.StatusRequestTimeout, "", ErrRequestTimeout
			lastErr = ctx.Err()
			break
		}
	}

	c.sess.mu.Lock()
	c.sess.lastCommand = command
	c.sess.lastStatus = status
	if state == ValidResponse {
		c.sess.lastValidResponse = time.Now().UTC()
	}
	c.sess.mu.Unlock()

	resp := Response{StatusCode: status, Body: data}
	if state == ValidResponse {
		return state, resp, nil
	}
	if class == nil {
		class = ErrConnectionLost
	}
	return state, resp, &RequestError{
		Command:    command,
		State:      state,
		StatusCode: status,
		Message:    message,
		Err:        lastErr,
		class:      class,
	}
}

// handleIncomingResponse ports ICULanDevice.HandleIncomingResponse.
func handleIncomingResponse(status int, body string) (RequestState, string) {
	if status == http.StatusNotFound {
		return ValidResponse, ""
	}
	body = strings.ReplaceAll(body, ":nan", ":null")
	if status >= 200 && status <= 299 {
		return ValidResponse, body
	}
	return InvalidResponse, body
}

// handleUnsuccessfulRequest ports ICULanDevice.HandleUnsuccessfulRequest. It
// returns the new state, the CallErrorHandler text (only on the last retry)
// and the error class for RequestError.
func (c *Client) handleUnsuccessfulRequest(ctx context.Context, command string, status int, isLastRetry bool) (RequestState, string, error) {
	c.sess.mu.Lock()
	rebooting, uploading, previous := c.sess.rebooting, c.sess.uploading, c.sess.lastStatus
	c.sess.mu.Unlock()
	if rebooting && previous == http.StatusRequestTimeout {
		return Unsuccessful, "", ErrDeviceRebooting
	}
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden, http.StatusTooManyRequests:
		c.sess.mu.Lock()
		c.sess.defaultCategoryCollected = false
		c.sess.loggedIn = false
		c.AccessToken = ""
		c.sess.mu.Unlock()
		if command != "login" && command != "logout" && command != "prc" && status != http.StatusTooManyRequests {
			if res := c.loginRequest(ctx, LoginTimeout); res.LoggedIn {
				return RetryRequest, "", nil
			}
		}
		return Unsuccessful, "", ErrUnauthorized
	case http.StatusBadRequest, http.StatusNotImplemented:
		msg := ""
		if isLastRetry {
			msg = "Device '" + c.Identity + "' does not support the request, please contact support."
		}
		return Unsuccessful, msg, ErrUnsupportedRequest
	case http.StatusNotAcceptable:
		msg := ""
		if isLastRetry {
			msg = "Device '" + c.Identity + "' has an invalid or expired certificate, please contact support."
		}
		return Unsuccessful, msg, ErrInvalidCertificate
	case http.StatusMultipleChoices, http.StatusConflict, http.StatusInternalServerError, http.StatusServiceUnavailable:
		c.resetConnectionFlags()
		msg := ""
		if isLastRetry {
			msg = fmt.Sprintf("Device '%s'/%s connection lost.", c.Identity, c.IP)
		}
		return Unsuccessful, msg, ErrConnectionLost
	case StatusUnused, http.StatusRequestTimeout, http.StatusGatewayTimeout:
		c.resetConnectionFlags()
		if uploading || (rebooting && command == "reboot") {
			return Unsuccessful, "", ErrRequestTimeout
		}
		ok, pingMsg := c.checkNetwork(ctx)
		msg := ""
		if !ok && isLastRetry {
			msg = pingMsg
		}
		if !isLastRetry {
			return RetryRequest, "", ErrRequestTimeout
		}
		return Unsuccessful, msg, ErrRequestTimeout
	default:
		return RetryRequest, "", ErrRetriesExhausted
	}
}

// roundTrip ports SendHttpRequest + CreateHttpClient's header selection and
// reads the whole body (the .NET HttpClient buffers it inside SendAsync, so
// a failed body read counts as a failed send).
func (c *Client) roundTrip(ctx context.Context, uri string, uriErr error, body any) (int, string, error) {
	if uriErr != nil {
		return 0, "", uriErr
	}
	req, err := c.newWebRequest(ctx, uri, body)
	if err != nil {
		return 0, "", err
	}
	c.sess.mu.Lock()
	c.sess.httpActive = true
	c.sess.mu.Unlock()
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return 0, "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", &url.Error{Op: req.Method, URL: uri, Err: err}
	}
	return resp.StatusCode, string(data), nil
}

// newWebRequest builds the request exactly like Exec does (same dispatch,
// headers and multipart layout) but bound to ctx, with the auth header read
// under the session lock.
func (c *Client) newWebRequest(ctx context.Context, uri string, body any) (*http.Request, error) {
	var (
		req *http.Request
		err error
	)
	switch b := body.(type) {
	case nil:
		req, err = http.NewRequestWithContext(ctx, http.MethodGet, uri, nil)
	case string:
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, uri, strings.NewReader(b))
		if err == nil {
			req.Header.Set("Content-Type", "application/json; charset=utf-8")
		}
	case []byte:
		boundary := "----aceclientboundary" + randBoundary()
		payload := buildFirmwareMultipart(b, boundary, c.isHTTPS())
		req, err = http.NewRequestWithContext(ctx, http.MethodPost, uri, bytes.NewReader(payload))
		if err == nil {
			req.Header.Set("Content-Type", "multipart/form-data; boundary="+boundary)
		}
	default:
		// SendHttpRequest: throw new NotSupportedException("HttpRequest not supported")
		return nil, fmt.Errorf("HttpRequest not supported (%T)", body)
	}
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	c.sess.mu.Lock()
	access, creds := c.AccessToken, c.Creds
	c.sess.mu.Unlock()
	switch {
	case c.isHTTPS() && access != "": // UseBearerAuthentication
		req.Header.Set("Authorization", "Bearer "+access)
	case !c.isHTTPS() && creds.Username != "" && creds.Password != "": // UseBasicAuthentication
		req.SetBasicAuth(creds.Username, creds.Password)
	}
	return req, nil
}

// isTimeout reports a transport timeout (HttpClient.Timeout / cancellation).
func isTimeout(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	var ue *url.Error
	return errors.As(err, &ue) && ue.Timeout()
}

// isCertificateError reports a rejected server certificate, the Go analogue
// of the AuthenticationException ExecuteWebRequestInternal maps to 406.
func isCertificateError(err error) bool {
	if errors.Is(err, ErrInvalidCertificate) {
		return true
	}
	var (
		cve *tls.CertificateVerificationError
		uae x509.UnknownAuthorityError
		cie x509.CertificateInvalidError
		he  x509.HostnameError
	)
	return errors.As(err, &cve) || errors.As(err, &uae) || errors.As(err, &cie) || errors.As(err, &he)
}
