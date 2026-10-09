// Package isah ports ACEISAHConnection.dll (namespace ICUIWSConnection), the
// installer's client for the Alfen "IWS" back end (ISAH), together with the
// ICULanDevice members that use it (ACENetwork/ICUNetwork/ICULanDevice.cs):
//
//   - IWSConnection: GetObjectData(Async) and TestIWSConnection
//     (Connection, GetObjectData, TestIWSConnection);
//   - the response model: IWSObject, IWSPropertyValue, Article,
//     CSCommunication, CSInterface, IsahResponse, decoded with Newtonsoft.Json
//     semantics (DecodeIWSObject);
//   - IWSFirmwareFeatures: the Features flag enum, the feature texts, and
//     IsFeatureUnlocked (with a System.Version port, Version);
//   - ICULanDevice.LoadSettingsFromIWS and ICULanDevice.IsFeatureUnlocked
//     (LoadSettingsFromIWS, DeviceFeatures);
//   - the non-UI logic of DlgObjectID (object ID validation and the IWS check).
//
// Wire format (IWSConnection.GetObjectDataAsync):
//
//	GET {site}/api/settings/{objectId}?NumberOfSockets={n}&ProcessorId={p}&AdditionalInfo={a}&ProductionRequest={True|False}&ObjectCode={c}&IncludeLogo={True|False}
//	Accept: application/json
//	Authorization: Basic base64(ASCII(security))
//
// The query values are not URL-encoded, the same as the C# string.Format. Only
// the characters that System.Uri itself escapes are percent-encoded. Each
// request has an 8 s budget (s_nRequestTimeout) for the headers and the whole
// body. Any failure, including a JSON error, becomes a synthesized
// 408 RequestTimeout response. The site is AppProperties.IsahSite
// (DefaultSite). Credentials (ICUUser.Company) always come from the caller.
// This package has no UI dependencies.
package isah

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// DefaultSite ports AppProperties.IsahSite (ACEServiceInstaller/AppProperties.cs).
	DefaultSite = "https://installer.alfen.com"
	// RequestTimeout ports IWSConnection.s_nRequestTimeout (8000 ms), the
	// CancellationTokenSource budget for each request.
	RequestTimeout = 8000 * time.Millisecond
	// ClientTimeout ports HttpClient.Timeout = TimeSpan.FromMinutes(5.0).
	ClientTimeout = 5 * time.Minute
	// maxAutomaticRedirections is the HttpClientHandler default (50).
	maxAutomaticRedirections = 50
)

// Connection ports the static class ICUIWSConnection.IWSConnection
// (ACEISAHConnection/ICUIWSConnection/IWSConnection.cs). Its transport can
// be replaced. The zero value uses a shared client from NewHTTPClient(nil).
type Connection struct {
	// HTTP is the client used for all requests. If it is nil, a shared
	// client from NewHTTPClient(nil) is used.
	HTTP *http.Client
}

var (
	defaultClientOnce sync.Once
	defaultClient     *http.Client
	defaultConnection = &Connection{}
)

func (c *Connection) client() *http.Client {
	if c != nil && c.HTTP != nil {
		return c.HTTP
	}
	defaultClientOnce.Do(func() { defaultClient = NewHTTPClient(nil) })
	return defaultClient
}

// NewHTTPClient returns an *http.Client that behaves like the C#
// `new HttpClient { Timeout = 5 min }` on .NET Framework:
//   - Timeout is ClientTimeout;
//   - redirects are followed up to 50 times (MaxAutomaticRedirections), and
//     the Authorization header is removed on each redirect (HttpWebRequest
//     clears it on automatic redirects);
//   - if rt is nil, a clone of http.DefaultTransport is used with HTTP/1.1
//     only and no transparent gzip (HttpClientHandler sends no
//     Accept-Encoding by default).
//
// Pass rt to use a different transport, for example httptest's.
func NewHTTPClient(rt http.RoundTripper) *http.Client {
	if rt == nil {
		tr := http.DefaultTransport.(*http.Transport).Clone()
		tr.DisableCompression = true
		var p http.Protocols
		p.SetHTTP1(true)
		tr.Protocols = &p
		rt = tr
	}
	return &http.Client{
		Transport:     rt,
		Timeout:       ClientTimeout,
		CheckRedirect: dotNetRedirectPolicy,
	}
}

// dotNetRedirectPolicy ports the HttpWebRequest auto-redirect behavior used
// by .NET Framework's HttpClientHandler.
func dotNetRedirectPolicy(req *http.Request, via []*http.Request) error {
	if len(via) > maxAutomaticRedirections {
		return errors.New("Too many automatic redirections were attempted.")
	}
	req.Header.Del("Authorization")
	return nil
}

// ObjectDataRequest holds the optional parameters of
// IWSConnection.GetObjectData/GetObjectDataAsync. Use NewObjectDataRequest to
// get the C# default values.
type ObjectDataRequest struct {
	ObjectID          string // objectId (path segment)
	NumberOfSockets   int    // numberOfSockets, C# default 1
	ProcessorID       string // processorId, C# default ""
	AdditionalInfo    string // additionalInfo, C# default ""
	ProductionRequest bool   // productionRequest, C# default false
	ObjectCode        string // objectCode, C# default ""
	IncludeLogo       bool   // includeLogo, C# default true
}

// NewObjectDataRequest returns the C# default parameters for objectID:
// numberOfSockets = 1, processorId = "", additionalInfo = "",
// productionRequest = false, objectCode = "", includeLogo = true.
func NewObjectDataRequest(objectID string) ObjectDataRequest {
	return ObjectDataRequest{ObjectID: objectID, NumberOfSockets: 1, IncludeLogo: true}
}

// dotNetBool is bool.ToString() as string.Format writes it.
func dotNetBool(b bool) string {
	if b {
		return "True"
	}
	return "False"
}

// relativeURI is the exact string.Format result in GetObjectDataAsync
// (before System.Uri escaping).
func (r ObjectDataRequest) relativeURI() string {
	return fmt.Sprintf("api/settings/%s?NumberOfSockets=%s&ProcessorId=%s&AdditionalInfo=%s&ProductionRequest=%s&ObjectCode=%s&IncludeLogo=%s",
		r.ObjectID, strconv.Itoa(r.NumberOfSockets), r.ProcessorID, r.AdditionalInfo,
		dotNetBool(r.ProductionRequest), r.ObjectCode, dotNetBool(r.IncludeLogo))
}

// parseAbsoluteURI ports `new Uri(url)` for HttpClient.BaseAddress. It must be
// an absolute URI, and an http(s) URI must have a host. In C# this constructor
// runs before the try block, so a bad site is thrown to the caller instead of
// becoming a 408.
func parseAbsoluteURI(s string) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(s))
	if err != nil {
		return nil, fmt.Errorf("Invalid URI: %w", err)
	}
	if !u.IsAbs() {
		return nil, errors.New("Invalid URI: The format of the URI could not be determined.")
	}
	if (u.Scheme == "http" || u.Scheme == "https") && u.Host == "" {
		return nil, errors.New("Invalid URI: The hostname could not be parsed.")
	}
	return u, nil
}

func isHex(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// escapeLikeSystemURI approximates how System.Uri escapes a relative
// reference that it combines with an http(s) base. It percent-encodes control
// characters, space, '"', '<', '>', '^', '`', '{', '|', '}', non-ASCII (as
// UTF-8), and a '%' that does not start a valid escape. In the path part '\'
// becomes '/', and in the query it is escaped. '?', '&', '=', '#' and '/' are
// kept, so an unescaped '&' or '#' in a value changes the URL, as in C#.
func escapeLikeSystemURI(s string) string {
	const hexDigits = "0123456789ABCDEF"
	var b strings.Builder
	inPath := true
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '?' || c == '#':
			inPath = false
			b.WriteByte(c)
		case c == '\\' && inPath:
			b.WriteByte('/')
		case c == '%' && i+2 < len(s) && isHex(s[i+1]) && isHex(s[i+2]):
			b.WriteByte(c)
		case c <= 0x20 || c >= 0x7f || strings.IndexByte("\"<>\\^`{|}%", c) >= 0:
			b.WriteByte('%')
			b.WriteByte(hexDigits[c>>4])
			b.WriteByte(hexDigits[c&0x0f])
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// resolveRelative ports new Uri(BaseAddress, relativeUri) as HttpClient does it
// for a relative request URI (RFC 3986 resolution, including dot segments).
func resolveRelative(base *url.URL, rel string) (*url.URL, error) {
	ref, err := url.Parse(escapeLikeSystemURI(rel))
	if err != nil {
		return nil, err
	}
	return base.ResolveReference(ref), nil
}

// asciiBytes ports Encoding.ASCII.GetBytes: each character outside 7-bit ASCII
// (and each invalid UTF-8 byte) becomes '?'.
func asciiBytes(s string) []byte {
	out := make([]byte, 0, len(s))
	for _, r := range s {
		if r < 0x80 {
			out = append(out, byte(r))
		} else {
			out = append(out, '?')
		}
	}
	return out
}

// bufferResponse reads the whole body, as HttpClient.GetAsync does with
// HttpCompletionOption.ResponseContentRead (inside the same cancellation
// budget).
func bufferResponse(resp *http.Response) (*HTTPResponse, error) {
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return &HTTPResponse{
		StatusCode: resp.StatusCode,
		Status:     strings.TrimSpace(strings.TrimPrefix(resp.Status, strconv.Itoa(resp.StatusCode))),
		Header:     resp.Header,
		Body:       body,
	}, nil
}

// GetObjectData ports IWSConnection.GetObjectDataAsync (and the synchronous
// GetObjectData wrapper, which only blocks on it). It asks IWS for the
// factory settings of one object number:
//
//	GET {siteURL}/api/settings/{ObjectID}?NumberOfSockets=..&ProcessorId=..&AdditionalInfo=..&ProductionRequest=..&ObjectCode=..&IncludeLogo=..
//	Accept: application/json
//	Authorization: Basic base64(ASCII(security))
//
// It returns an error only when siteURL is not an absolute URI (C# throws
// UriFormatException before its try block). Every other failure (transport
// error, the 8 s timeout, a body read error, or a JSON error on a 2xx
// response) gives an IsahResponse with a synthesized 408 response and Err set.
// IsahObject is set only for a 2xx response whose body decodes to a non-null
// object.
func (c *Connection) GetObjectData(ctx context.Context, siteURL, security string, req ObjectDataRequest) (*IsahResponse, error) {
	base, err := parseAbsoluteURI(siteURL)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()

	res := &IsahResponse{}
	fail := func(err error) (*IsahResponse, error) {
		res.HTTPResponse = requestTimeoutResponse()
		res.Err = err
		return res, nil
	}
	target, err := resolveRelative(base, req.relativeURI())
	if err != nil {
		return fail(err)
	}
	hreq, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return fail(err)
	}
	hreq.Header.Set("Accept", "application/json")
	hreq.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString(asciiBytes(security)))
	hreq.Header["User-Agent"] = []string{""} // HttpClient sends no User-Agent by default
	resp, err := c.client().Do(hreq)
	if err != nil {
		return fail(err)
	}
	hr, err := bufferResponse(resp)
	if err != nil {
		return fail(err)
	}
	res.HTTPResponse = hr
	if hr.IsSuccessStatusCode() {
		obj, err := DecodeIWSObject(hr.Body)
		if err != nil {
			return fail(err)
		}
		res.IsahObject = obj
	}
	return res, nil
}

// TestIWSConnection ports IWSConnection.TestIWSConnection: a plain GET of
// siteURL (no Accept or Authorization header) within the 8 s budget. The result
// is IsSuccessStatusCode. Transport and timeout errors give (false, nil), as
// the C# catch block does. The only error returned is a siteURL that is not
// an absolute URI (thrown by `new Uri(url)` before the try block).
func (c *Connection) TestIWSConnection(ctx context.Context, siteURL string) (bool, error) {
	u, err := parseAbsoluteURI(siteURL)
	if err != nil {
		return false, err
	}
	ctx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return false, nil
	}
	req.Header["User-Agent"] = []string{""}
	resp, err := c.client().Do(req)
	if err != nil {
		return false, nil
	}
	hr, err := bufferResponse(resp)
	if err != nil {
		return false, nil
	}
	return hr.IsSuccessStatusCode(), nil
}

// GetObjectData calls Connection.GetObjectData on the default connection.
func GetObjectData(ctx context.Context, siteURL, security string, req ObjectDataRequest) (*IsahResponse, error) {
	return defaultConnection.GetObjectData(ctx, siteURL, security, req)
}

// TestIWSConnection calls Connection.TestIWSConnection on the default connection.
func TestIWSConnection(ctx context.Context, siteURL string) (bool, error) {
	return defaultConnection.TestIWSConnection(ctx, siteURL)
}
