package core

import (
	"strings"
	"testing"
	"time"

	"alfen/aceclient/internal/api"
)

func init() {
	loginRetryDelay = time.Millisecond
	uploadRetryDelay = time.Millisecond
}

func TestLoginRequestSendsInstallerBody(t *testing.T) {
	d := newFakeDevice(t, sampleProps())
	s := d.session()
	res := s.LoginRequest(api.LoginData{Username: "admin", Password: `"secret"`, DisplayName: "Jan"})
	if !res.LoggedIn || res.StatusCode != 200 {
		t.Fatalf("login = %+v", res)
	}
	reqs := d.requests()
	if len(reqs) != 1 || reqs[0].Method != "POST" || reqs[0].Path != "/api/login" {
		t.Fatalf("requests = %+v", reqs)
	}
	// LoginRequest: Password.Trim('"'), hand-built JSON.
	if want := `{"username":"admin","password":"secret","displayname":"Jan"}`; reqs[0].Body != want {
		t.Fatalf("body = %s, want %s", reqs[0].Body, want)
	}
	if !s.IsLoggedIn() || s.Client.AccessToken != "acc1" {
		t.Fatalf("state: loggedIn=%v token=%q", s.IsLoggedIn(), s.Client.AccessToken)
	}
}

func TestLoginRequestFailureStatus(t *testing.T) {
	d := newFakeDevice(t, nil)
	s := d.session()
	res := s.LoginRequest(api.LoginData{Username: "admin", Password: "wrong"})
	if res.LoggedIn || res.StatusCode != 403 {
		t.Fatalf("res = %+v", res)
	}
	if got := LoginErrorMessage(res.StatusCode, res.Content, s.Identification(), s.Address()); !strings.Contains(got, "is incorrect") {
		t.Fatalf("msg = %q", got)
	}
	d.loginCode, d.loginBody = 429, `{"lockout_remaining_seconds":90}`
	res = s.LoginRequest(api.LoginData{Username: "admin", Password: "secret"})
	if res.StatusCode != 429 || LoginErrorMessage(res.StatusCode, res.Content, "X", "") != "Locked out of device 'X' for 1.5 minutes due to multiple incorrect password attempts." {
		t.Fatalf("lockout: %+v", res)
	}
}

func TestAfterLoginReadsDefaultCategories(t *testing.T) {
	d := newFakeDevice(t, sampleProps())
	s := d.session()
	if !s.LoginRequest(api.LoginData{Username: "admin", Password: "secret"}).LoggedIn {
		t.Fatal("login")
	}
	d.reset()
	if err := s.AfterLogin(); err != nil {
		t.Fatal(err)
	}
	if err := s.AfterLogin(); err != nil { // IsDefaultCategoryCollected: no second read
		t.Fatal(err)
	}
	reqs := d.requests()
	if len(reqs) != 2 || reqs[0].Query != "cat=generic" || reqs[1].Query != "cat=generic2" {
		t.Fatalf("requests = %+v", reqs)
	}
	if reqs[0].Auth != "Bearer acc1" {
		t.Fatalf("auth = %q", reqs[0].Auth)
	}
	if got := s.Props.String(0x2053, 0); got != "MYCHARGER" {
		t.Fatalf("identity = %q", got)
	}
	if s.Identification() != "MYCHARGER" { // filled from 8275 when discovery gave none
		t.Fatalf("Identification = %q", s.Identification())
	}
	if got := s.FirmwareVersion().String(); got != "7.4.6" {
		t.Fatalf("fw = %s", got)
	}
	if !s.IsUniquePasswordRequired() || s.IsAHP() {
		t.Fatal("NG9xx 7.x requires a unique password and is not AHP")
	}
	if got := s.HostName(); got != "ng910-60023-ace0123456" {
		t.Fatalf("derived hostname = %q", got)
	}
	if p, _ := s.Props.Get(0x2062, 0); p.Value != "16" {
		t.Fatalf("REAL32 normalised value = %q", p.Value)
	}
}

func TestRequestAndUpdateAllCategories(t *testing.T) {
	d := newFakeDevice(t, sampleProps())
	s := d.session()
	s.LoginRequest(api.LoginData{Username: "admin", Password: "secret"})
	d.reset()
	cats := s.RequestCategories()
	if strings.Join(cats, ",") != "generic,generic2,comm" {
		t.Fatalf("cats = %v", cats)
	}
	if err := s.UpdateCategories(); err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, r := range d.requests() {
		got = append(got, r.Method+" "+r.Path+"?"+r.Query)
	}
	want := "GET /api/categories?|GET /api/categories?|GET /api/prop?cat=generic|GET /api/prop?cat=generic2|GET /api/prop?cat=comm"
	if strings.Join(got, "|") != want {
		t.Fatalf("requests:\n%s\nwant\n%s", strings.Join(got, "|"), want)
	}
	if s.Props.Len() != len(sampleProps()) {
		t.Fatalf("props = %d", s.Props.Len())
	}
}

func TestUpdatePropertiesUsesIDs(t *testing.T) {
	d := newFakeDevice(t, sampleProps())
	s := d.session()
	s.LoginRequest(api.LoginData{Username: "admin", Password: "secret"})
	d.reset()
	if err := s.UpdateProperties(ODIndex(0x2053, 0), ODIndex(0x2051, 0)); err != nil {
		t.Fatal(err)
	}
	if r := d.requests(); len(r) != 1 || r[0].Query != "ids=2053_0,2051_0" {
		t.Fatalf("requests = %+v", r)
	}
}

func TestExecuteReloginOn401(t *testing.T) {
	d := newFakeDevice(t, sampleProps())
	s := d.session()
	s.LoginRequest(api.LoginData{Username: "admin", Password: "secret"})
	d.reset()
	d.propCode = []int{401}
	if err := s.UpdateCategories("generic"); err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, r := range d.requests() {
		got = append(got, r.Method+" "+r.Path)
	}
	// 401 → token/refresh (only a refresh token is left) → login → retry.
	if want := "GET /api/prop|POST /api/token/refresh|POST /api/login|GET /api/prop"; strings.Join(got, "|") != want {
		t.Fatalf("requests = %s, want %s", strings.Join(got, "|"), want)
	}
	if !s.IsLoggedIn() {
		t.Fatal("should be logged in again")
	}
}

func TestExecuteConnectionLost(t *testing.T) {
	d := newFakeDevice(t, sampleProps())
	s := d.session()
	s.LoginRequest(api.LoginData{Username: "admin", Password: "secret"})
	s.Props.Merge([]api.Property{{ID: 8275, DataType: api.SDTVisibleString, Value: "CPX"}})
	d.reset()
	d.propCode = []int{503, 503}
	err := s.UpdateCategories("generic")
	re, ok := err.(*RequestError)
	if !ok || re.StatusCode != 503 {
		t.Fatalf("err = %#v", err)
	}
	// UpdatePropertiesInternal uses maxRetries 2: the first 503 is not the
	// last retry, so HandleUnsuccessfulRequest raises no popup, and the
	// UNSUCCESSFUL state ends the loop.
	if re.Message != "" {
		t.Fatalf("message = %q", re.Message)
	}
	if s.IsLoggedIn() {
		t.Fatal("503 logs the device out")
	}
	if n := len(d.requests()); n != 1 {
		t.Fatalf("503 must not be retried, got %d requests", n)
	}
	// With a single attempt the popup text is produced.
	d.propCode = []int{503}
	err = s.execute("prop", RequestTimeout, 1, false, func(c *api.Client) error {
		_, err := c.ReadProperties("cat=generic")
		return err
	})
	host, _ := d.hostPort()
	if re, ok := err.(*RequestError); !ok || re.Message != "Device 'CPX'/"+host+" connection lost." {
		t.Fatalf("err = %#v", err)
	}
}

func TestStorePropertiesBatchesOf15AndCommits(t *testing.T) {
	d := newFakeDevice(t, nil)
	s := d.session()
	s.LoginRequest(api.LoginData{Username: "admin", Password: "secret"})
	var in []api.Property
	for i := 0; i < 16; i++ {
		in = append(in, api.Property{ID: uint16(0x3000 + i), DataType: api.SDTUnsigned16, Value: "1"})
	}
	s.Props.Merge(in)
	for i := 0; i < 16; i++ {
		if _, err := s.Props.SetValue(uint16(0x3000+i), 0, "2"); err != nil {
			t.Fatal(err)
		}
	}
	d.reset()
	if err := s.StoreProperties(s.Props.Changed()...); err != nil {
		t.Fatal(err)
	}
	reqs := d.requests()
	if len(reqs) != 2 {
		t.Fatalf("POSTs = %d", len(reqs))
	}
	if !strings.HasPrefix(reqs[0].Body, `{"3000_0":{"id":"3000_0","value":2},"3001_0":`) || strings.Count(reqs[0].Body, `"id"`) != 15 {
		t.Fatalf("batch 1 = %s", reqs[0].Body)
	}
	if reqs[1].Body != `{"300F_0":{"id":"300F_0","value":2}}` {
		t.Fatalf("batch 2 = %s", reqs[1].Body)
	}
	if s.Props.HasChanges() {
		t.Fatal("stored properties must be committed")
	}
}

func TestSaveAllChangesOrder(t *testing.T) {
	d := newFakeDevice(t, nil)
	s := d.session()
	s.LoginRequest(api.LoginData{Username: "admin", Password: "secret"})
	s.Props.Merge([]api.Property{
		{ID: 8272, DataType: api.SDTVisibleString, Value: "NG910"},
		{ID: 8273, DataType: api.SDTVisibleString, Value: "ACE1"},
		{ID: 8275, DataType: api.SDTVisibleString, Value: "CP1"},
		{ID: 8583, DataType: api.SDTUnsigned64, Value: "0"},
		{ID: 8290, DataType: api.SDTReal32, Value: "16"},
		{ID: 8609, DataType: api.SDTVisibleString, Value: "KEY"},
	})
	s.Props.SetValue(8290, 0, "25.5")
	s.Props.SetValue(8273, 0, "ACE2")
	d.reset()
	now := time.UnixMilli(1760000000123)
	res, err := s.SaveAllChanges(now)
	if err != nil {
		t.Fatal(err)
	}
	if !res.IdentityOrModelChanged {
		t.Fatal("8273 changed → identity/model flag")
	}
	var bodies []string
	for _, r := range d.requests() {
		bodies = append(bodies, r.Body)
	}
	want := []string{
		`{"2051_0":{"id":"2051_0","value":"ACE2"}}`,
		`{"2187_0":{"id":"2187_0","value":1760000000123}}`,
		`{"2062_0":{"id":"2062_0","value":25.5}}`,
	}
	if strings.Join(bodies, "\n") != strings.Join(want, "\n") {
		t.Fatalf("bodies:\n%s\nwant:\n%s", strings.Join(bodies, "\n"), strings.Join(want, "\n"))
	}
}

func TestUploadFirmwareResults(t *testing.T) {
	d := newFakeDevice(t, nil)
	s := d.session()
	s.LoginRequest(api.LoginData{Username: "admin", Password: "secret"})
	if got := s.UploadFirmware([]byte("abc"), nil); got != "" {
		t.Fatalf("ok upload = %q", got)
	}
	d.fwCode = []int{500}
	if got := s.UploadFirmware([]byte("abc"), nil); got != UploadCommunicationError {
		t.Fatalf("500 upload = %q", got)
	}
	d.reset()
	d.fwCode = []int{403, 200}
	relogins := 0
	got := s.UploadFirmware([]byte("abc"), func() int { relogins++; return 401 })
	if got != "" || relogins != 1 {
		t.Fatalf("403 → relogin(401) → retry: got %q relogins %d", got, relogins)
	}
	if n := len(d.requests()); n != 2 {
		t.Fatalf("uploads = %d", n)
	}
	r := d.requests()[0]
	if r.Path != "/api/firmware" || r.Method != "POST" {
		t.Fatalf("req = %+v", r)
	}
}

func TestLogout(t *testing.T) {
	d := newFakeDevice(t, nil)
	s := d.session()
	s.LoginRequest(api.LoginData{Username: "admin", Password: "secret"})
	d.reset()
	if !s.Logout() {
		t.Fatal("logout not acknowledged")
	}
	if r := d.requests(); len(r) != 1 || r[0].Path != "/api/logout" || r[0].Method != "POST" {
		t.Fatalf("requests = %+v", r)
	}
	if s.IsLoggedIn() || s.Client.AccessToken != "" {
		t.Fatal("logged out state")
	}
}

func TestTamperAlarm(t *testing.T) {
	pc := NewPropertyCache()
	if TamperAlarm(pc) {
		t.Fatal("no property → no alarm")
	}
	pc.Merge([]api.Property{{ID: 8784, DataType: api.SDTUnsigned8, Value: "3"}})
	if !TamperAlarm(pc) {
		t.Fatal("Tampered → alarm")
	}
	pc.Merge([]api.Property{{ID: 8784, DataType: api.SDTUnsigned8, Value: "4"}})
	if TamperAlarm(pc) {
		t.Fatal("Tampered_Ack → no alarm")
	}
}
