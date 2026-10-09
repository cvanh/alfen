package core

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// fakeRequest is one request the fake charger received.
type fakeRequest struct {
	Method, Path, Query, Body, Auth string
}

// fakeProp is one object-dictionary entry served by the fake charger in the
// /api/prop page shape ParseProperty expects.
type fakeProp struct {
	ID     string `json:"id"`
	Type   int    `json:"type"`
	Access int    `json:"access"`
	Cat    string `json:"cat"`
	Len    uint64 `json:"len"`
	Name   string `json:"name"`
	Value  any    `json:"value"`
}

// fakeDevice is an httptest TLS charger: /api/login, /api/logout,
// /api/categories, GET/POST /api/prop and /api/firmware.
type fakeDevice struct {
	t   *testing.T
	srv *httptest.Server

	mu        sync.Mutex
	reqs      []fakeRequest
	props     []fakeProp
	password  string
	loginCode int    // forced /api/login status (0 = check password)
	loginBody string // body for a forced status
	propCode  []int  // queued statuses for GET /api/prop (consumed first)
	storeCode int    // status for POST /api/prop (0 = 200)
	fwCode    []int  // queued statuses for /api/firmware
	token     int
}

func newFakeDevice(t *testing.T, props []fakeProp) *fakeDevice {
	t.Helper()
	d := &fakeDevice{t: t, props: props, password: "secret"}
	d.srv = httptest.NewTLSServer(http.HandlerFunc(d.serve))
	t.Cleanup(d.srv.Close)
	return d
}

func (d *fakeDevice) hostPort() (string, int) {
	host, port, _ := net.SplitHostPort(strings.TrimPrefix(d.srv.URL, "https://"))
	p, _ := strconv.Atoi(port)
	return host, p
}

func (d *fakeDevice) session() *Session {
	host, port := d.hostPort()
	return NewSession(host, port, true, "https")
}

func (d *fakeDevice) requests() []fakeRequest {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]fakeRequest(nil), d.reqs...)
}

func (d *fakeDevice) reset() {
	d.mu.Lock()
	d.reqs = nil
	d.mu.Unlock()
}

func (d *fakeDevice) serve(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	d.mu.Lock()
	d.reqs = append(d.reqs, fakeRequest{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Body: string(body), Auth: r.Header.Get("Authorization")})
	d.mu.Unlock()
	switch r.URL.Path {
	case "/api/login":
		d.mu.Lock()
		code, cbody := d.loginCode, d.loginBody
		d.mu.Unlock()
		if code != 0 {
			w.WriteHeader(code)
			_, _ = io.WriteString(w, cbody)
			return
		}
		var ld struct{ Password string }
		_ = json.Unmarshal(body, &ld)
		if ld.Password != d.password {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		d.mu.Lock()
		d.token++
		tok := d.token
		d.mu.Unlock()
		fmt.Fprintf(w, `{"access":"acc%d","refresh":"ref%d"}`, tok, tok)
	case "/api/token/refresh":
		w.WriteHeader(http.StatusUnauthorized)
	case "/api/logout":
		w.WriteHeader(http.StatusOK)
	case "/api/categories":
		d.mu.Lock()
		seen := map[string]bool{}
		var cats []string
		for _, p := range d.props {
			if !seen[p.Cat] {
				seen[p.Cat] = true
				cats = append(cats, p.Cat)
			}
		}
		d.mu.Unlock()
		b, _ := json.Marshal(map[string]any{"version": 1, "categories": cats})
		_, _ = w.Write(b)
	case "/api/prop":
		d.servePropLocked(w, r)
	case "/api/firmware":
		d.mu.Lock()
		code := http.StatusOK
		if len(d.fwCode) > 0 {
			code, d.fwCode = d.fwCode[0], d.fwCode[1:]
		}
		d.mu.Unlock()
		w.WriteHeader(code)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (d *fakeDevice) servePropLocked(w http.ResponseWriter, r *http.Request) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if r.Method == http.MethodPost {
		if d.storeCode != 0 {
			w.WriteHeader(d.storeCode)
			return
		}
		w.WriteHeader(http.StatusOK)
		return
	}
	if len(d.propCode) > 0 {
		code := d.propCode[0]
		d.propCode = d.propCode[1:]
		w.WriteHeader(code)
		return
	}
	q := r.URL.Query()
	var out []fakeProp
	for _, p := range d.props {
		switch {
		case q.Get("cat") != "":
			if p.Cat == q.Get("cat") {
				out = append(out, p)
			}
		case q.Get("ids") != "":
			for _, id := range strings.Split(q.Get("ids"), ",") {
				if strings.EqualFold(id, p.ID) {
					out = append(out, p)
				}
			}
		default:
			out = append(out, p)
		}
	}
	b, _ := json.Marshal(map[string]any{"version": 2, "total": len(out), "offset": 0, "count": len(out), "properties": out})
	_, _ = w.Write(b)
}

// sampleProps is a small NG9xx-like dictionary.
func sampleProps() []fakeProp {
	return []fakeProp{
		{ID: "2050_0", Type: 9, Access: 1, Cat: "generic", Len: 31, Name: "sysChargePointModel", Value: "NG910-60023"},
		{ID: "2051_0", Type: 9, Access: 0, Cat: "generic", Len: 31, Name: "sysChargePointSerialNumber", Value: "ACE0123456"},
		{ID: "2053_0", Type: 9, Access: 0, Cat: "generic", Len: 21, Name: "sysChargeBoxIdentity", Value: "MYCHARGER"},
		{ID: "100A_0", Type: 9, Access: 1, Cat: "generic", Len: 31, Name: "Manufacturer software version", Value: "7.4.6-4416"},
		{ID: "2062_0", Type: 8, Access: 0, Cat: "generic2", Len: 0, Name: "sysMaxStationCurrent", Value: 16.0},
		{ID: "2129_0", Type: 5, Access: 0, Cat: "generic2", Len: 2, Name: "flag", Value: 1},
		{ID: "2118_0", Type: 9, Access: 1, Cat: "comm", Len: 64, Name: "commModemManufacturer", Value: "Quectel"},
	}
}
