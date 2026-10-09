package backoffice

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"alfen/aceclient/internal/api"
)

func prop(id uint16, sub byte, name string, dt api.SDT, value string) api.Property {
	return api.Property{ID: id, Sub: sub, Name: name, DataType: dt, Value: value}
}

// legacyDevice is an NG9xx without network profiles (8311 connect method).
func legacyDevice() []api.Property {
	return []api.Property{
		prop(8305, 1, "commBackOfficeURLwired_serverDomainAndPort", api.SDTVisibleString, "ws://wired.example"),
		prop(8305, 2, "commBackOfficeURLwired_serverPath", api.SDTVisibleString, ""),
		prop(8310, 0, "commBackOfficeShortName", api.SDTVisibleString, "acme,ABB"),
		prop(8311, 0, "commConnectMethod", api.SDTInteger8, "3"),
		prop(8312, 1, "commBackOfficeURL_serverDomainAndPort", api.SDTVisibleString, "ws://gprs.example"),
		prop(8312, 2, "commBackOfficeURL_serverPath", api.SDTVisibleString, "p"),
		prop(8321, 0, "commProtocolName", api.SDTVisibleString, "ocpp/json"),
		prop(8448, 0, "gprsAPNname", api.SDTVisibleString, "apn.example"),
		prop(8449, 0, "gprsAPNuser", api.SDTVisibleString, ""),
		prop(8450, 0, "gprsAPNpassword", api.SDTVisibleString, ""),
		prop(8471, 0, "commProxyEnabled", api.SDTUnsigned8, "1"),
	}
}

// profileDevice is a device with the four network profiles (8432..8435).
func profileDevice(extended bool) []api.Property {
	ps := []api.Property{
		prop(8310, 0, "commBackOfficeShortName", api.SDTVisibleString, "acme"),
		prop(8321, 0, "commProtocolName", api.SDTVisibleString, "ocpp/json"),
		prop(8448, 0, "gprsAPNname", api.SDTVisibleString, ""),
	}
	for i := 0; i < 4; i++ {
		id := uint16(8432 + i)
		n := func(s string) string { return fmt.Sprintf("commProfile%d_%s", i+1, s) }
		ps = append(ps,
			prop(id, 1, n("boVersion"), api.SDTUnsigned8, "3"),
			prop(id, 3, n("CSMSUrl"), api.SDTVisibleString, "ws://csms.example"),
			prop(id, 4, n("msgTimeout"), api.SDTUnsigned16, "10"),
			prop(id, 5, n("SecurityProfile"), api.SDTUnsigned8, "0"),
			prop(id, 6, n("boInterface"), api.SDTUnsigned8, strconv.Itoa(i%2)),
			prop(id, 7, n("APNName"), api.SDTVisibleString, ""),
			prop(id, 8, n("APNUsernameOld"), api.SDTVisibleString, ""),
			prop(id, 9, n("APNPassword"), api.SDTVisibleString, ""),
			prop(id, 10, n("simPinOld"), api.SDTVisibleString, ""),
			prop(id, 14, n("priority"), api.SDTUnsigned8, strconv.Itoa(i)),
		)
		if extended {
			ps = append(ps,
				prop(id, 15, n("APNUsername"), api.SDTVisibleString, ""),
				prop(id, 16, n("simPin"), api.SDTVisibleString, ""))
		}
	}
	return ps
}

func ids(ps []api.Property) string {
	var s []string
	for _, p := range ps {
		s = append(s, fmt.Sprintf("%d_%d=%s", p.ID, p.Sub, p.Value))
	}
	return strings.Join(s, " ")
}

func TestClearAllBackOfficeSettings(t *testing.T) {
	got := ids(ClearAllBackOfficeSettings(LookupFrom(legacyDevice()), false))
	want := "8305_1= 8305_2= 8312_1= 8312_2= 8448_0= 8449_0= 8450_0="
	if got != want {
		t.Errorf("legacy:\n got %s\nwant %s", got, want)
	}
	all := ClearAllBackOfficeSettings(LookupFrom(profileDevice(false)), false)
	if len(all) != 1+4*10 {
		t.Fatalf("profile clear: %d entries", len(all))
	}
	if got := ids(all[1:11]); got != "8432_14=0 8432_6=0 8432_1=1 8432_3= 8432_5=0 8432_7= 8432_8= 8432_9= 8432_10= 8432_4=10" {
		t.Errorf("profile 1 clear: %s", got)
	}
	ext := ClearAllBackOfficeSettings(LookupFrom(profileDevice(true)), true)
	if got := ids(ext[1:11]); got != "8432_14=0 8432_6=0 8432_1=1 8432_3= 8432_5=0 8432_7= 8432_15= 8432_9= 8432_16= 8432_4=10" {
		t.Errorf("extended field lengths: %s", got)
	}
	if got := ClearAllBackOfficeSettings(nil, false); len(got) != 0 {
		t.Error("no device, no writes")
	}
}

func TestDeviceCapabilities(t *testing.T) {
	legacy := LookupFrom(legacyDevice())
	if HasNetworkProfiles(legacy) || HasExtendedFieldLengths(legacy) || !HasBackOfficeConfigured(legacy) {
		t.Error("legacy device with connect method 3")
	}
	none := LookupFrom([]api.Property{prop(8311, 0, "c", api.SDTInteger8, "0")})
	if HasBackOfficeConfigured(none) || HasBackOfficeConfigured(LookupFrom(nil)) {
		t.Error("connect method 0 / missing 8311 -> not configured")
	}
	pd := LookupFrom(profileDevice(true))
	if !HasNetworkProfiles(pd) || !HasExtendedFieldLengths(pd) {
		t.Error("profile capabilities")
	}
	// profile 2 (8433) has priority 1 and interface 1
	if !HasBackOfficeConfigured(pd) {
		t.Error("profile with priority and interface -> configured")
	}
	off := profileDevice(false)
	for i := range off {
		if off[i].Sub == 6 {
			off[i].Value = "0"
		}
	}
	if HasBackOfficeConfigured(LookupFrom(off)) {
		t.Error("no interface -> not configured")
	}
}

func TestPlanSave(t *testing.T) {
	cat := Catalog{"acme-b": "/p/acme-b.fwi", "acme-a": "/p/acme-a.fwi", "newbo": "/p/newbo.fwi"}
	legacy := LookupFrom(legacyDevice())
	cases := []struct {
		name                   string
		in                     SaveInput
		sel, upload            string
		clear, after, writes   string
		disableProxy, profiles bool
	}{
		{
			name:   "standalone legacy",
			in:     SaveInput{Device: legacy, Selection: KeyStandAlone, SelectionChanged: true, CurrentPreset: "acme", MeterName: "ABB", ConnectMode: "0"},
			sel:    KeyStandAlone,
			writes: "8310_0=,ABB 8305_1= 8312_1= 8312_2= 8448_0= 8471_0=0 8311_0=0", disableProxy: true,
		},
		{
			name:   "manual legacy wired",
			in:     SaveInput{Device: legacy, Selection: KeyManual, SelectionChanged: true, CurrentPreset: "acme", ConnectMode: "1", ConnectModeChanged: true},
			sel:    KeyManual,
			writes: "8310_0= 8311_0=1",
		},
		{
			name: "same preset legacy only connect mode",
			in:   SaveInput{Device: legacy, Selection: "acme", CurrentPreset: "acme", Presets: cat, Firmware: V(6, 0, 0), ConnectMode: "3"},
			sel:  "acme", writes: "8471_0=0", disableProxy: true,
		},
		{
			name: "changed preset new firmware -> -b",
			in: SaveInput{Device: legacy, Selection: "acme", SelectionChanged: true, CurrentPreset: "acme", MeterName: "ABB",
				Presets: cat, Firmware: V(4, 12, 0), ConnectMode: "1"},
			sel: "acme", upload: "/p/acme-b.fwi",
			clear:  "8305_1= 8305_2= 8312_1= 8312_2= 8448_0= 8449_0= 8450_0=",
			after:  "", // 8310 already "acme,ABB"
			writes: "8311_0=1",
		},
		{
			name: "different preset old firmware falls back to exact name",
			in:   SaveInput{Device: legacy, Selection: "newbo", CurrentPreset: "acme", MeterName: "ABB", Presets: cat, Firmware: V(4, 11, 0), ConnectMode: "3"},
			sel:  "newbo", upload: "/p/newbo.fwi",
			clear: "8305_1= 8305_2= 8312_1= 8312_2= 8448_0= 8449_0= 8450_0=",
			after: "8310_0=newbo,ABB", writes: "8471_0=0", disableProxy: true,
		},
		{
			name: "preset file missing -> nothing uploaded",
			in:   SaveInput{Device: legacy, Selection: "gone", CurrentPreset: "acme", Presets: cat, Firmware: V(6, 0, 0), ConnectMode: "3"},
			sel:  "gone", writes: "8471_0=0", disableProxy: true,
		},
		{
			name: "profiles all off -> standalone",
			in: SaveInput{Device: LookupFrom(profileDevice(false)), Selection: "acme", CurrentPreset: "acme",
				HasNetworkProfiles: true},
			sel: KeyStandAlone,
			writes: "8433_14=0 8433_6=0 8433_1=1 8433_3= 8432_1=1 8432_3= 8434_14=0 8434_1=1 8434_3= 8435_14=0 8435_6=0 8435_1=1 8435_3=" +
				"",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			plan, err := PlanSave(c.in)
			if err != nil {
				t.Fatal(err)
			}
			if plan.Selection != c.sel || plan.UploadFile != c.upload || plan.DisableProxy != c.disableProxy || plan.ProfilesChanged != c.profiles {
				t.Errorf("plan: sel %q upload %q proxy %v profiles %v", plan.Selection, plan.UploadFile, plan.DisableProxy, plan.ProfilesChanged)
			}
			if got := ids(plan.ClearFirst); got != c.clear {
				t.Errorf("ClearFirst:\n got %s\nwant %s", got, c.clear)
			}
			if got := ids(plan.AfterUpload); got != c.after {
				t.Errorf("AfterUpload:\n got %s\nwant %s", got, c.after)
			}
			if c.name == "profiles all off -> standalone" {
				// order is first-set order; only changed values are written
				got := ids(plan.Writes)
				for _, w := range []string{"8433_14=0", "8432_1=1", "8432_3=", "8435_6=0"} {
					if !strings.Contains(got, w) {
						t.Errorf("missing %s in %s", w, got)
					}
				}
				for _, w := range []string{"8432_14=", "8321_0", "8432_4="} {
					if strings.Contains(got, w) {
						t.Errorf("unchanged %s must not be written: %s", w, got)
					}
				}
				return
			}
			if got := ids(plan.Writes); got != c.writes {
				t.Errorf("Writes:\n got %s\nwant %s", got, c.writes)
			}
		})
	}
}

func TestPlanSaveNetworkProfileValidation(t *testing.T) {
	dev := LookupFrom(profileDevice(true))
	base := SaveInput{Device: dev, Selection: KeyManual, CurrentPreset: "acme", HasNetworkProfiles: true, HasExtendedFieldLengths: true}

	in := base
	in.ProfilePriorities = [4]byte{1, 1, 0, 0}
	if _, err := PlanSave(in); !errors.Is(err, ErrUniquePriorities) || err.Error() != "Network profiles must have unique priorities." {
		t.Errorf("duplicate priorities: %v", err)
	}

	in = base
	in.IsAHP = true
	in.ProfilePriorities = [4]byte{1, 2, 0, 0}
	in.ProfileOCPPVersions = [4]NPVersion{NPVersionOCPP16, NPVersionOCPP201, NPVersionOCPP201, 0}
	in.ProfileSecurityProfiles = [4]string{"0", "0", "2", "0"}
	_, err := PlanSave(in)
	var spe SecurityProfileError
	if !errors.As(err, &spe) || spe.Profile != 2 ||
		err.Error() != `You cannot use "0: Default" as security profile in Network Profile 2 when using OCPP 2.0.1 or higher` {
		t.Errorf("security profile: %v", err)
	}
	in.IsAHP = false
	if _, err := PlanSave(in); err != nil {
		t.Errorf("non-AHP skips the security-profile rule: %v", err)
	}

	in = base
	in.ProfilePriorities = [4]byte{1, 0, 2, 0}
	in.ProfilePriorityChanged = [4]bool{false, false, true, false}
	plan, err := PlanSave(in)
	if err != nil || !plan.ProfilesChanged {
		t.Errorf("priority change: %v %+v", err, plan)
	}
	if got := ids(plan.Writes); got != "8310_0=" {
		t.Errorf("manual with profiles writes only the short name (8321 unchanged): %s", got)
	}
}

type recorded struct {
	method, path, query, ctype, auth string
	body                             []byte
}

func newDeviceServer(t *testing.T, status map[string]int) (*api.Client, func() []recorded) {
	t.Helper()
	var mu sync.Mutex
	var reqs []recorded
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		mu.Lock()
		reqs = append(reqs, recorded{r.Method, r.URL.Path, r.URL.RawQuery, r.Header.Get("Content-Type"), r.Header.Get("Authorization"), b})
		mu.Unlock()
		if s, ok := status[r.URL.Path]; ok {
			w.WriteHeader(s)
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)
	host, port, err := net.SplitHostPort(strings.TrimPrefix(srv.URL, "https://"))
	if err != nil {
		t.Fatal(err)
	}
	p, _ := strconv.Atoi(port)
	c := api.New(host, p, true)
	c.AccessToken = "tok"
	return c, func() []recorded { mu.Lock(); defer mu.Unlock(); return append([]recorded(nil), reqs...) }
}

func TestExecutePresetUpload(t *testing.T) {
	dir := t.TempDir()
	preset := filepath.Join(dir, "newbo.fwi")
	payload := []byte("PRESET-BYTES\x00\x01")
	if err := os.WriteFile(preset, payload, 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := PlanSave(SaveInput{Device: LookupFrom(legacyDevice()), Selection: "newbo", CurrentPreset: "acme", MeterName: "ABB",
		Presets: Catalog{"newbo": preset}, Firmware: V(6, 6, 2), ConnectMode: "3"})
	if err != nil {
		t.Fatal(err)
	}
	c, got := newDeviceServer(t, nil)
	res := Execute(c, plan, nil)
	if res.ClearErr != nil || res.UploadErr != nil || res.StoreErr != nil || !res.Uploaded || !res.AskReboot {
		t.Fatalf("result: %+v", res)
	}
	reqs := got()
	if len(reqs) != 3 {
		t.Fatalf("%d requests", len(reqs))
	}
	clear := `{"commBackOfficeURLwired_serverDomainAndPort":{"id":"2071_1","value":""},` +
		`"commBackOfficeURLwired_serverPath":{"id":"2071_2","value":""},` +
		`"commBackOfficeURL_serverDomainAndPort":{"id":"2078_1","value":""},` +
		`"commBackOfficeURL_serverPath":{"id":"2078_2","value":""},` +
		`"gprsAPNname":{"id":"2100_0","value":""},` +
		`"gprsAPNuser":{"id":"2101_0","value":""},` +
		`"gprsAPNpassword":{"id":"2102_0","value":""}}`
	if r := reqs[0]; r.method != "POST" || r.path != "/api/prop" || r.query != "" || string(r.body) != clear || r.auth != "Bearer tok" {
		t.Errorf("clear request: %s %s?%s %q", r.method, r.path, r.query, r.body)
	}
	if r := reqs[1]; r.method != "POST" || r.path != "/api/firmware" || !strings.HasPrefix(r.ctype, "multipart/form-data; boundary=") ||
		!strings.Contains(string(r.body), `Content-Disposition: form-data; name="firmwarefile"; filename="filename"`+"\r\n\r\n"+string(payload)+"\r\n") {
		t.Errorf("upload request: %s %s %q %q", r.method, r.path, r.ctype, r.body)
	}
	final := `{"commProxyEnabled":{"id":"2117_0","value":0},"commBackOfficeShortName":{"id":"2076_0","value":"newbo,ABB"}}`
	if r := reqs[2]; r.method != "POST" || r.path != "/api/prop" || string(r.body) != final {
		t.Errorf("final store: %s %s %q", r.method, r.path, r.body)
	}
}

func TestExecuteFailures(t *testing.T) {
	plan, _ := PlanSave(SaveInput{Device: LookupFrom(legacyDevice()), Selection: "newbo", CurrentPreset: "acme",
		Presets: Catalog{"newbo": "/nonexistent/newbo.fwi"}, Firmware: V(6, 6, 2), ConnectMode: "1"})

	// Upload fails: the clear was stored, the remaining changes (8311) are
	// still stored, but the short name (8310) is not written.
	c, got := newDeviceServer(t, nil)
	res := Execute(c, plan, func(string) error { return errors.New("boom") })
	if res.Uploaded || res.UploadErr == nil || res.AskReboot {
		t.Errorf("result: %+v", res)
	}
	reqs := got()
	if len(reqs) != 2 || reqs[0].path != "/api/prop" || reqs[1].path != "/api/prop" {
		t.Fatalf("clear + final store expected, got %d requests", len(reqs))
	}
	if b := string(reqs[1].body); b != `{"commConnectMethod":{"id":"2077_0","value":1}}` {
		t.Errorf("final store after failed upload: %s", b)
	}

	// Clear store fails: the changed cleared values are retried with the rest.
	c, got = newDeviceServer(t, map[string]int{"/api/prop": 500})
	uploaded := ""
	res = Execute(c, plan, func(p string) error { uploaded = p; return nil }, prop(8322, 0, "commProtocolVersion", api.SDTVisibleString, "1.6"))
	if res.ClearErr == nil || !res.Uploaded || res.StoreErr == nil || uploaded != "/nonexistent/newbo.fwi" {
		t.Errorf("result: %+v uploaded %q", res, uploaded)
	}
	reqs = got()
	if len(reqs) != 2 {
		t.Fatalf("%d requests", len(reqs))
	}
	body := string(reqs[1].body)
	for _, want := range []string{`"id":"2071_1"`, `"id":"2078_2"`, `"id":"2100_0"`, `"id":"2076_0","value":"newbo"`, `"id":"2082_0","value":"1.6"`} {
		if !strings.Contains(body, want) {
			t.Errorf("retry body missing %s: %s", want, body)
		}
	}
	if strings.Contains(body, `"id":"2071_2"`) {
		t.Error("an unchanged cleared value is not retried")
	}
}

func TestSelectionChanged(t *testing.T) {
	tr, fa := true, false
	cases := []struct {
		name     string
		sel, cur string
		prev     *bool
		profiles bool
		prio     [4]byte
		want     SelectionEffects
	}{
		{"same as device", "ACME", "acme", &tr, false, [4]byte{}, SelectionEffects{Ignored: true}},
		{"first change to manual", KeyManual, "acme", nil, false, [4]byte{}, SelectionEffects{}},
		{"preset -> manual clears APN", KeyManual, "acme", &tr, false, [4]byte{}, SelectionEffects{ClearAPNFields: true}},
		{"manual with profiles off", KeyManual, "acme", &fa, true, [4]byte{}, SelectionEffects{SetFirstProfilePriority: true}},
		{"manual with a profile on", KeyManual, "acme", &fa, true, [4]byte{0, 1}, SelectionEffects{}},
		{"standalone legacy", KeyStandAlone, "acme", &tr, false, [4]byte{}, SelectionEffects{SetConnectModeNone: true, ClearAPNFields: true}},
		{"standalone profiles", KeyStandAlone, "acme", &fa, true, [4]byte{1}, SelectionEffects{ClearProfilePriorities: true}},
		{"to a preset", "other", "acme", &fa, false, [4]byte{}, SelectionEffects{WasPreset: true}},
	}
	for _, c := range cases {
		if got := SelectionChanged(c.sel, c.cur, c.prev, c.profiles, c.prio); got != c.want {
			t.Errorf("%s: got %+v; want %+v", c.name, got, c.want)
		}
	}
}

func TestControls(t *testing.T) {
	keys := func(o []Option) string {
		var s []string
		for _, x := range o {
			s = append(s, x.Key)
		}
		return strings.Join(s, ",")
	}
	s := Controls(ControlsInput{Selection: KeyStandAlone, SupportsModem: true, Firmware: V(6, 0, 0), Protocol: "1.6", ConnectMode: "3"})
	if keys(s.ConnectMethodOptions) != "0" || s.ConnectMethodEnabled || !s.ForceConnectMethodNone || !s.WiredURLHidden ||
		s.URLFieldsEnabled || !s.APNFieldsHidden || s.SimPinHidden || s.ShowMobile || s.ShowProxy || s.ShowSecurityExtensions {
		t.Errorf("standalone: %+v", s)
	}
	s = Controls(ControlsInput{Selection: KeyManual, SupportsModem: false, Firmware: V(6, 0, 0), Protocol: "1.5", ConnectMode: "1"})
	if keys(s.ConnectMethodOptions) != "0,1" || !s.ConnectMethodEnabled || !s.URLFieldsEnabled || s.APNFieldsHidden ||
		!s.SimPinHidden || !s.ShowProxy || s.ShowMobile || !s.ShowSmartChargingProfiles || s.ShowSecurityExtensions {
		t.Errorf("manual: %+v", s)
	}
	s = Controls(ControlsInput{Selection: "acme", SupportsModem: true, Firmware: V(4, 9, 0), HasNetworkProfiles: true})
	if keys(s.ConnectMethodOptions) != "0,1,2,3" || !s.URLFieldsEnabled || !s.ShowMobile || !s.ShowProxy || !s.ShowSecurityExtensions {
		t.Errorf("preset on 4.x with profiles: %+v", s)
	}
	if RebootQuestion("ACE0001") != "You have changed the Backoffice settings. The new settings will only be active after a restart. Do you want to restart ACE0001 now?" {
		t.Error("reboot question text")
	}
}
