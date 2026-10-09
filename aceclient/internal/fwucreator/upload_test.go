package fwucreator

import (
	"bytes"
	"context"

	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"alfen/aceclient/internal/api"
)

type recorded struct {
	Method, Path, Query, Body string
	Parts                     [][]byte
}

// fakeCharger answers the endpoints StartUpload touches. Responses are
// looked up by "METHOD /path"; a queue lets a path answer differently per call.
type fakeCharger struct {
	mu        sync.Mutex
	reqs      []recorded
	responses map[string][]fakeResponse
	delay     map[string]time.Duration
}

type fakeResponse struct {
	status int
	body   string
}

func (f *fakeCharger) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	rec := recorded{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Body: string(body)}
	if mt, params, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err == nil && mt == "multipart/form-data" {
		mr := multipart.NewReader(bytes.NewReader(body), params["boundary"])
		for {
			p, err := mr.NextPart()
			if err != nil {
				break
			}
			if p.FormName() != "firmwarefile" || p.FileName() != "filename" {
				rec.Parts = append(rec.Parts, []byte("BAD PART HEADER"))
			}
			b, _ := io.ReadAll(p)
			rec.Parts = append(rec.Parts, b)
		}
	}
	key := r.Method + " " + r.URL.Path
	f.mu.Lock()
	f.reqs = append(f.reqs, rec)
	resp := fakeResponse{http.StatusOK, "{}"}
	if q := f.responses[key]; len(q) > 0 {
		resp = q[0]
		if len(q) > 1 {
			f.responses[key] = q[1:]
		}
	}
	d := f.delay[key]
	f.mu.Unlock()
	if d > 0 {
		time.Sleep(d)
	}
	w.WriteHeader(resp.status)
	io.WriteString(w, resp.body)
}

func (f *fakeCharger) calls() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []string
	for _, r := range f.reqs {
		s := r.Method + " " + r.Path
		if r.Query != "" {
			s += "?" + r.Query
		}
		out = append(out, s)
	}
	return out
}

func newFakeCharger(t *testing.T, responses map[string][]fakeResponse) (*fakeCharger, *api.Client) {
	t.Helper()
	f := &fakeCharger{responses: responses, delay: map[string]time.Duration{}}
	srv := httptest.NewTLSServer(f)
	t.Cleanup(srv.Close)
	u, _ := url.Parse(srv.URL)
	host, portStr, _ := net.SplitHostPort(u.Host)
	port, _ := strconv.Atoi(portStr)
	c := api.New(host, port, true)
	c.Creds = api.LoginData{Username: "test-user", Password: "test-pass", DisplayName: "test"}
	c.Identity = "TEST-CHARGER"
	return f, c
}

func propBody(id string, value string) string {
	return `{"version":2,"properties":[{"id":"` + id + `","access":0,"type":5,"len":0,"cat":"generic","value":` + value + `}],"offset":0,"total":1}`
}

const statusIdle = `{"uploadInProgress":"false","OD_fileFirmwareUpdateStatus":{"id":"3602_1","value":0},}`

var fixedNow = time.Date(2026, 10, 10, 12, 0, 0, 0, time.UTC)

func TestUploadResourceNonAHP(t *testing.T) {
	f, c := newFakeCharger(t, map[string][]fakeResponse{
		"GET /api/prop":     {{200, propBody("3600_1", "0")}},
		"GET /api/firmware": {{200, statusIdle}},
	})
	data := bytes.Repeat([]byte{0xAB, 0xCD, 0xEF}, 5000) // 15000 bytes -> 4 multipart chunks
	cache := func(id uint16, sub byte) (api.Property, bool) {
		if id == 0x2059 && sub == 0 {
			return api.Property{ID: 0x2059, Sub: 0, Name: "OD_sysDateTime", DataType: api.SDTUnsigned64, Value: "0"}, true
		}
		return api.Property{}, false
	}
	err := UploadResource(context.Background(), c, data, UploadOptions{Property: cache, Now: func() time.Time { return fixedNow }})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"GET /api/prop?ids=3600_1",
		"POST /api/prop",
		"POST /api/cmd",
		"GET /api/firmware",
		"POST /api/firmware",
	}
	if got := f.calls(); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("calls\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	ms := strconv.FormatInt(fixedNow.UnixMilli(), 10)
	if b := f.reqs[1].Body; b != `{"OD_sysDateTime":{"id":"2059_0","value":`+ms+`}}` {
		t.Fatalf("SetDate store body %s", b)
	}
	if b := f.reqs[2].Body; b != `{"command":"date 2026-10-10 12:00:00"}` {
		t.Fatalf("SendDatetime body %s", b)
	}
	up := f.reqs[4]
	if len(up.Parts) != 4 || !bytes.Equal(bytes.Join(up.Parts, nil), data) || len(up.Parts[0]) != 4096 {
		t.Fatalf("multipart: %d parts", len(up.Parts))
	}
	if c.IsUploading() {
		t.Fatal("IsUploading left set")
	}
}

func TestUploadResourceAHP(t *testing.T) {
	// Time sync 3 (NTP): no SetDate at all.
	f, c := newFakeCharger(t, map[string][]fakeResponse{
		"GET /api/prop":     {{200, propBody("3600_1", "3")}},
		"GET /api/firmware": {{200, statusIdle}},
	})
	if err := UploadResource(context.Background(), c, []byte("tvf"), UploadOptions{IsAHP: true}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(f.calls(), ","); got != "GET /api/prop?ids=3600_1,GET /api/firmware,POST /api/firmware" {
		t.Fatalf("calls %s", got)
	}
	// Property absent: AHP sends the quoted datetime to /api/datetime.
	f, c = newFakeCharger(t, map[string][]fakeResponse{
		"GET /api/prop":     {{200, `{"version":2,"properties":[],"offset":0,"total":0}`}},
		"GET /api/firmware": {{200, statusIdle}},
	})
	if err := UploadResource(context.Background(), c, []byte("tvf"), UploadOptions{IsAHP: true, Now: func() time.Time { return fixedNow }}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(f.calls(), ","); got != "GET /api/prop?ids=3600_1,POST /api/datetime,GET /api/firmware,POST /api/firmware" {
		t.Fatalf("calls %s", got)
	}
	if b := f.reqs[1].Body; b != `"2026-10-10 12:00:00"` {
		t.Fatalf("datetime body %s", b)
	}
}

func TestUploadResourceErrors(t *testing.T) {
	tests := []struct {
		name      string
		responses map[string][]fakeResponse
		relogin   int
		wantErr   string
		wantPosts int
		wantSleep int
	}{
		{"upload in progress (string true)", map[string][]fakeResponse{
			"GET /api/firmware": {{200, `{"uploadInProgress":"true"}`}}}, 200, ErrMsgUploadInProgress, 0, 0},
		{"boolean true is not 'true' (C# quirk)", map[string][]fakeResponse{
			"GET /api/firmware": {{200, `{"uploadInProgress":true}`}}}, 200, "", 1, 0},
		{"status read fails", map[string][]fakeResponse{
			"GET /api/firmware": {{500, ""}}}, 200, ErrMsgUploadInProgress, 0, 0},
		{"status 404 is empty", map[string][]fakeResponse{
			"GET /api/firmware": {{404, ""}}}, 200, ErrMsgUploadInProgress, 0, 0},
		{"missing value key throws", map[string][]fakeResponse{
			"GET /api/firmware": {{200, `{"OD_fileFirmwareUpdateStatus":{"id":"x"}}`}}}, 200, "The given key was not present in the dictionary.", 0, 0},
		{"device error", map[string][]fakeResponse{
			"GET /api/firmware":  {{200, statusIdle}},
			"POST /api/firmware": {{500, ""}}}, 200, ErrMsgCouldNotCommunate, 1, 0},
		{"401 then login ok: no resend (C# quirk)", map[string][]fakeResponse{
			"GET /api/firmware":  {{200, statusIdle}},
			"POST /api/firmware": {{401, ""}},
			"POST /api/login":    {{401, ""}}}, 200, "", 1, 0},
		{"401 x3 with failing login ends as success (C# quirk)", map[string][]fakeResponse{
			"GET /api/firmware":  {{200, statusIdle}},
			"POST /api/firmware": {{401, ""}},
			"POST /api/login":    {{401, ""}}}, 401, "", 3, 3},
		{"403 then login 401 then 200", map[string][]fakeResponse{
			"GET /api/firmware":  {{200, statusIdle}},
			"POST /api/firmware": {{403, ""}, {200, ""}},
			"POST /api/login":    {{401, ""}}}, 401, "", 2, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tc.responses["GET /api/prop"] = []fakeResponse{{200, propBody("3600_1", "3")}}
			f, c := newFakeCharger(t, tc.responses)
			sleeps := 0
			err := UploadResource(context.Background(), c, []byte("data"), UploadOptions{
				Relogin: func(context.Context) int { return tc.relogin },
				Sleep:   func(time.Duration) { sleeps++ },
			})
			if tc.wantErr == "" && err != nil || tc.wantErr != "" && (err == nil || err.Error() != tc.wantErr) {
				t.Fatalf("err %v, want %q", err, tc.wantErr)
			}
			posts := 0
			for _, s := range f.calls() {
				if s == "POST /api/firmware" {
					posts++
				}
			}
			if posts != tc.wantPosts || sleeps != tc.wantSleep {
				t.Fatalf("posts %d sleeps %d, want %d %d (%v)", posts, sleeps, tc.wantPosts, tc.wantSleep, f.calls())
			}
			if c.IsUploading() {
				t.Fatal("IsUploading left set")
			}
		})
	}
}

func TestUploadResourceProgress(t *testing.T) {
	f, c := newFakeCharger(t, map[string][]fakeResponse{
		"GET /api/prop":     {{200, propBody("3600_1", "3")}},
		"GET /api/firmware": {{200, statusIdle}},
	})
	f.delay["POST /api/firmware"] = 120 * time.Millisecond
	var mu sync.Mutex
	var got []int
	err := UploadResource(context.Background(), c, []byte("x"), UploadOptions{
		Progress:         func(p int) { mu.Lock(); got = append(got, p); mu.Unlock() },
		ProgressInterval: 10 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(got) < 3 {
		t.Fatalf("progress reported %d times", len(got))
	}
	for i, p := range got {
		if p < 1 || p >= 50 || (i > 0 && p < got[i-1]) {
			t.Fatalf("progress sequence %v", got)
		}
	}
}

func TestGetFirmwareUploadStatus(t *testing.T) {
	tests := []struct {
		status     int
		body       string
		ok, inProg bool
		st         FirmwareUpdateStatus
		err        bool
	}{
		{200, `{"uploadInProgress":"true"}`, true, true, FwNoActiveUpdate, false},
		{200, `{"uploadInProgress":true}`, true, false, FwNoActiveUpdate, false},
		{200, `{"OD_fileFirmwareUpdateStatus":{"value":7}}`, true, false, FwReadyForUpdate, false},
		{200, `{"OD_fileFirmwareUpdateStatus":{"value":" -2 "}}`, true, false, FwErrorDuringUpdate, false},
		{200, `{"OD_fileFirmwareUpdateStatus":{"value":99}}`, false, false, FwNoActiveUpdate, false},
		{200, `{"OD_fileFirmwareUpdateStatus":{"value":7.5}}`, false, false, FwNoActiveUpdate, false},
		{200, `{"OD_fileFirmwareUpdateStatus":{}}`, false, false, FwNoActiveUpdate, false},
		{200, `{"OD_fileFirmwareUpdateStatus":{"value":nan}}`, false, false, FwNoActiveUpdate, true},
		{200, `{"a":1,}`, true, false, FwNoActiveUpdate, false},
		{200, ``, false, false, FwNoActiveUpdate, false},
		{404, `{"uploadInProgress":"true"}`, false, false, FwNoActiveUpdate, false},
	}
	for i, tc := range tests {
		f, c := newFakeCharger(t, map[string][]fakeResponse{"GET /api/firmware": {{tc.status, tc.body}}})
		ok, inProg, st, err := GetFirmwareUploadStatus(context.Background(), c)
		if ok != tc.ok || inProg != tc.inProg || st != tc.st || (err != nil) != tc.err {
			t.Errorf("%d %s: ok=%v in=%v st=%v err=%v", i, tc.body, ok, inProg, st, err)
		}
		if got := f.calls(); len(got) != 1 || got[0] != "GET /api/firmware" {
			t.Errorf("%d: calls %v", i, got)
		}
	}
}

func TestProgressHelper(t *testing.T) {
	p := NewProgressHelper(false)
	v := p.GetProgress(0.9)
	if v != 1.15 {
		t.Fatalf("first %v", v)
	}
	for i := 0; i < 1000; i++ {
		v = p.GetProgress(v)
	}
	if v >= 50 || v < 49.5 {
		t.Fatalf("stage 0 cap %v", v)
	}
	if p.GetNextStage() != 50 || p.GetNextStage() != 97 || p.GetNextStage() != 0 {
		t.Fatal("stages")
	}
	a := NewProgressHelper(true)
	if a.GetProgress(0.9) != 1.0 || a.GetNextStage() != 4 {
		t.Fatal("AHP helper")
	}
}

func TestUnixMillisString(t *testing.T) {
	if got := unixMillisString(time.Date(2026, 10, 10, 12, 0, 0, 1_500_000, time.UTC)); got != strconv.FormatInt(fixedNow.UnixMilli()+2, 10) {
		t.Fatalf("1.5 ms rounds to even: %s", got) // Math.Round(x.5) -> even
	}
	if got := unixMillisString(time.Date(2026, 10, 10, 12, 0, 0, 2_500_000, time.UTC)); got != strconv.FormatInt(fixedNow.UnixMilli()+2, 10) {
		t.Fatalf("2.5 ms rounds to even: %s", got)
	}
}
