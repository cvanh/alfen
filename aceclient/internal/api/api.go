// Package api ports the charger HTTPS control/settings channel from
// ICUNetwork.ICUDevice / ICULanDevice (docs/decompiled/ACENetwork.cs).
//
// URI scheme (TryGetUri):     {Protocol}://{IPAddress}:{Port}/api/{command}[?{parameters}]
// Auth (CreateHttpClient):    Bearer <AccessToken> over HTTPS; Basic over HTTP.
// Bodies (SendHttpRequest):   nil -> GET; string -> POST application/json;
//
//	[]byte -> POST multipart/form-data (firmware).
//
// Settings are a CANopen-style object dictionary: propId (u16 index) + subId
// (u8 subindex), wire id "IDX_SUB" in hex.
package api

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// SDT mirrors ICUNetwork.SDT (the object-dictionary data types).
type SDT int

const (
	SDTLowLimit       SDT = 0
	SDTBoolean        SDT = 1
	SDTInteger8       SDT = 2
	SDTInteger16      SDT = 3
	SDTInteger24      SDT = 16
	SDTInteger32      SDT = 4
	SDTInteger40      SDT = 18
	SDTInteger48      SDT = 19
	SDTInteger56      SDT = 20
	SDTInteger64      SDT = 21
	SDTUnsigned8      SDT = 5
	SDTUnsigned16     SDT = 6
	SDTUnsigned24     SDT = 22
	SDTUnsigned32     SDT = 7
	SDTUnsigned40     SDT = 24
	SDTUnsigned48     SDT = 25
	SDTUnsigned56     SDT = 26
	SDTUnsigned64     SDT = 27
	SDTReal32         SDT = 8
	SDTReal64         SDT = 17
	SDTVisibleString  SDT = 9
	SDTOctetString    SDT = 10
	SDTUnicodeString  SDT = 11
	SDTTimeOfDay      SDT = 12
	SDTTimeDifference SDT = 13
	SDTDomain         SDT = 15 // "DOAMIN" in the source (sic)
	SDTByteArray      SDT = 64
	SDTArray16        SDT = 65
	SDTHighLimit      SDT = 65535
)

// Property mirrors the fields of ICUProperty that cross the wire.
type Property struct {
	ID        uint16 // propId (index)
	Sub       byte   // subId (subindex)
	Name      string // ICUName (JSON key on write, "name" on read)
	DataType  SDT
	Value     string // raw string value as the device returns / expects
	ReadOnly  bool
	Category  string
	MaxLength uint64
}

// IDSub returns the "IDX_SUB" hex identifier (e.g. "8317_2"), matching
// string.Format("{0:X}_{1:X}", Id, SubId) — uppercase hex, no padding.
func (p Property) IDSub() string {
	return fmt.Sprintf("%X_%X", p.ID, p.Sub)
}

// LoginData mirrors the username/password/displayname triple.
type LoginData struct {
	Username    string
	Password    string
	DisplayName string
}

// Client is a single charger connection (ICULanDevice over LANConnection).
type Client struct {
	Protocol string // "https" (default) or "http"
	IP       string
	Port     int
	Creds    LoginData

	AccessToken  string
	RefreshToken string

	HTTP *http.Client
}

// New builds a Client. insecureTLS skips certificate verification, which is
// normally required for chargers (self-signed device certs; the installer pins
// them via its own CertificateStore instead).
func New(ip string, port int, insecureTLS bool) *Client {
	tr := &http.Transport{
		MaxConnsPerHost: 1, // HttpClientHandler.MaxConnectionsPerServer = 1
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecureTLS},
	}
	return &Client{
		Protocol: "https",
		IP:       ip,
		Port:     port,
		HTTP:     &http.Client{Transport: tr, Timeout: 15 * time.Minute},
	}
}

func (c *Client) isHTTPS() bool { return strings.EqualFold(c.Protocol, "https") }

// buildURI ports TryGetUri.
func (c *Client) buildURI(command, parameters string) (string, error) {
	u := fmt.Sprintf("%s://%s:%d/api/%s", c.Protocol, c.IP, c.Port, command)
	if parameters != "" {
		u += "?" + parameters
	}
	if _, err := url.ParseRequestURI(u); err != nil {
		return "", fmt.Errorf("incorrect uri: %s", u)
	}
	return u, nil
}

// applyAuth ports the Bearer(HTTPS)/Basic(HTTP) selection in CreateHttpClient.
func (c *Client) applyAuth(req *http.Request) {
	if c.isHTTPS() {
		if c.AccessToken != "" {
			req.Header.Set("Authorization", "Bearer "+c.AccessToken)
		}
		return
	}
	if c.Creds.Username != "" && c.Creds.Password != "" {
		req.SetBasicAuth(c.Creds.Username, c.Creds.Password)
	}
}

// Response mirrors the (state, status, body) tuple ExecuteWebRequest returns.
type Response struct {
	StatusCode int
	Body       string
}

// OK reports a 2xx response.
func (r Response) OK() bool { return r.StatusCode >= 200 && r.StatusCode < 300 }

// Exec ports ExecuteWebRequest/SendHttpRequest dispatch:
//
//	body == nil        -> GET
//	body is string     -> POST application/json
//	body is []byte     -> POST multipart/form-data ("firmwarefile")
func (c *Client) Exec(command, parameters string, body any) (Response, error) {
	uri, err := c.buildURI(command, parameters)
	if err != nil {
		return Response{}, err
	}
	var req *http.Request
	switch b := body.(type) {
	case nil:
		req, err = http.NewRequest(http.MethodGet, uri, nil)
	case string:
		req, err = http.NewRequest(http.MethodPost, uri, strings.NewReader(b))
		if err == nil {
			req.Header.Set("Content-Type", "application/json; charset=utf-8")
		}
	case []byte:
		return c.execMultipart(uri, b)
	default:
		return Response{}, fmt.Errorf("unsupported body type %T", body)
	}
	if err != nil {
		return Response{}, err
	}
	req.Header.Set("Accept", "application/json")
	c.applyAuth(req)
	return c.do(req)
}

func (c *Client) do(req *http.Request) (Response, error) {
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return Response{}, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return Response{}, err
	}
	return Response{StatusCode: resp.StatusCode, Body: string(data)}, nil
}

// Login ports ICUDevice login: POST /api/login with the exact JSON body
// {"username":..,"password":..,"displayname":..}, then stores the tokens.
func (c *Client) Login(ld LoginData) error {
	c.Creds = ld
	body := fmt.Sprintf(`{"username":"%s","password":"%s","displayname":"%s"}`,
		ld.Username, ld.Password, ld.DisplayName)
	resp, err := c.Exec("login", "", body)
	if err != nil {
		return err
	}
	if !resp.OK() {
		return fmt.Errorf("login failed: HTTP %d: %s", resp.StatusCode, resp.Body)
	}
	var tok struct {
		Access  string `json:"access"`
		Token   string `json:"token"`
		Refresh string `json:"refresh"`
	}
	_ = json.Unmarshal([]byte(resp.Body), &tok)
	if tok.Access != "" {
		c.AccessToken = tok.Access
	} else if tok.Token != "" {
		c.AccessToken = tok.Token
	}
	c.RefreshToken = tok.Refresh
	return nil
}

// RefreshSession ports POST /api/token/refresh {"refresh":..}.
func (c *Client) RefreshSession() error {
	body := fmt.Sprintf(`{"refresh":"%s"}`, c.RefreshToken)
	resp, err := c.Exec("token/refresh", "", body)
	if err != nil {
		return err
	}
	if !resp.OK() {
		return fmt.Errorf("refresh failed: HTTP %d", resp.StatusCode)
	}
	var tok struct {
		Access  string `json:"access"`
		Token   string `json:"token"`
		Refresh string `json:"refresh"`
	}
	_ = json.Unmarshal([]byte(resp.Body), &tok)
	if tok.Access != "" {
		c.AccessToken = tok.Access
	} else if tok.Token != "" {
		c.AccessToken = tok.Token
	}
	if tok.Refresh != "" {
		c.RefreshToken = tok.Refresh
	}
	return nil
}

// Logout ports POST /api/logout.
func (c *Client) Logout() error {
	_, err := c.Exec("logout", "", "")
	c.AccessToken = ""
	c.RefreshToken = ""
	return err
}

// propPage is one page of the GET /api/prop response.
type propPage struct {
	Version    int             `json:"version"`
	Total      int             `json:"total"`
	Offset     int             `json:"offset"`
	Count      int             `json:"count"`
	Properties []propEntryJSON `json:"properties"`
}

// propEntryJSON mirrors one entry parsed by ParseProperty.
type propEntryJSON struct {
	ID     string          `json:"id"`     // "IDX_SUB"
	Type   int             `json:"type"`   // SDT
	Access int             `json:"access"` // 1 == read-only
	Cat    string          `json:"cat"`
	Len    uint64          `json:"len"`
	Name   string          `json:"name"`
	Value  json.RawMessage `json:"value"`
}

// ReadProperties ports UpdatePropertiesInternal's paging + ParseProperty. It
// requests GET /api/prop with the given parameters, following &offset= paging
// until total <= offset+count, and returns the flattened property list.
func (c *Client) ReadProperties(parameters string) ([]Property, error) {
	var out []Property
	offset := 0
	for {
		params := parameters
		if offset > 0 {
			if params != "" {
				params += "&"
			}
			params += "offset=" + strconv.Itoa(offset)
		}
		resp, err := c.Exec("prop", params, nil)
		if err != nil {
			return out, err
		}
		if !resp.OK() {
			return out, fmt.Errorf("prop read HTTP %d: %s", resp.StatusCode, resp.Body)
		}
		var page propPage
		if err := json.Unmarshal([]byte(resp.Body), &page); err != nil {
			return out, fmt.Errorf("decode prop page: %w", err)
		}
		count := 0
		for _, e := range page.Properties {
			p, ok := parsePropertyEntry(e)
			if ok {
				out = append(out, p)
			}
			count++
		}
		if page.Total <= page.Offset+count || len(page.Properties) == 0 {
			break
		}
		offset = page.Offset + count
	}
	return out, nil
}

// parsePropertyEntry ports ParseProperty's id-splitting and field mapping.
func parsePropertyEntry(e propEntryJSON) (Property, bool) {
	parts := strings.Split(e.ID, "_")
	if len(parts) != 2 {
		return Property{}, false
	}
	idx, err1 := strconv.ParseUint(parts[0], 16, 16)
	sub, err2 := strconv.ParseUint(parts[1], 16, 8)
	if err1 != nil || err2 != nil {
		return Property{}, false
	}
	val := strings.Trim(string(e.Value), `"`)
	if e.Value == nil {
		val = "0"
	}
	return Property{
		ID:        uint16(idx),
		Sub:       byte(sub),
		Name:      e.Name,
		DataType:  SDT(e.Type),
		Value:     val,
		ReadOnly:  e.Access == 1,
		Category:  e.Cat,
		MaxLength: e.Len,
	}, true
}

// StoreProperties ports ICULanDevice.StoreProperties: batch up to 15 properties
// per POST /api/prop, serialising each per its data type exactly as the source.
func (c *Client) StoreProperties(props ...Property) error {
	for len(props) > 0 {
		n := 15
		if len(props) < n {
			n = len(props)
		}
		batch := props[:n]
		props = props[n:]

		var sb strings.Builder
		sb.WriteString("{")
		for i, p := range batch {
			sb.WriteString(serializeProperty(p))
			if i < len(batch)-1 {
				sb.WriteString(",")
			}
		}
		sb.WriteString("}")
		resp, err := c.Exec("prop", "", sb.String())
		if err != nil {
			return err
		}
		if !resp.OK() {
			return fmt.Errorf("prop store HTTP %d: %s", resp.StatusCode, resp.Body)
		}
	}
	return nil
}

// serializeProperty mirrors the per-type branches of StoreProperties. Value is
// held as a string here; the branches control quoting to match the source.
func serializeProperty(p Property) string {
	id := p.IDSub()
	switch p.DataType {
	case SDTByteArray, SDTArray16, SDTUnicodeString, SDTVisibleString, SDTDomain, SDTBoolean:
		// value quoted (byte/word arrays are hex CSV already in Value)
		return fmt.Sprintf(`"%s":{"id":"%s","value":"%s"}`, p.Name, id, p.Value)
	default:
		// REAL32/REAL64 and integer types: value unquoted/raw
		return fmt.Sprintf(`"%s":{"id":"%s","value":%s}`, p.Name, id, p.Value)
	}
}

// ---- convenience accessors (GetPropertyString et al.) ----

// GetProperty fetches a single property by index/subindex via ids=IDX_SUB.
func (c *Client) GetProperty(id uint16, sub byte) (Property, error) {
	want := fmt.Sprintf("%X_%X", id, sub)
	props, err := c.ReadProperties("ids=" + want)
	if err != nil {
		return Property{}, err
	}
	for _, p := range props {
		if p.ID == id && p.Sub == sub {
			return p, nil
		}
	}
	return Property{}, fmt.Errorf("property %s not found", want)
}
