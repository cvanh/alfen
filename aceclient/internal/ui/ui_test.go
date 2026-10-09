package ui

import (
	"encoding/json"
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

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"alfen/aceclient/internal/fwi"
	"alfen/aceclient/internal/ui/core"
)

// ----------------------------------------------------------- fake charger

type fakeReq struct{ Method, Path, Query, Body string }

type fakeCharger struct {
	srv   *httptest.Server
	mu    sync.Mutex
	reqs  []fakeReq
	props []map[string]any
}

func newFakeCharger(t *testing.T) *fakeCharger {
	t.Helper()
	f := &fakeCharger{props: []map[string]any{
		{"id": "2050_0", "type": 9, "access": 1, "cat": "generic", "len": 31, "name": "sysChargePointModel", "value": "NG910-60023"},
		{"id": "2051_0", "type": 9, "access": 0, "cat": "generic", "len": 31, "name": "sysChargePointSerialNumber", "value": "ACE0123456"},
		{"id": "2053_0", "type": 9, "access": 0, "cat": "generic", "len": 21, "name": "sysChargeBoxIdentity", "value": "MYCHARGER"},
		{"id": "100A_0", "type": 9, "access": 1, "cat": "generic", "len": 31, "name": "swVersion", "value": "7.4.6-4416"},
		{"id": "2062_0", "type": 8, "access": 0, "cat": "generic2", "len": 0, "name": "sysMaxStationCurrent", "value": 16.0},
		{"id": "2118_0", "type": 9, "access": 1, "cat": "comm", "len": 64, "name": "modemManufacturer", "value": "Quectel"},
	}}
	f.srv = httptest.NewTLSServer(http.HandlerFunc(f.serve))
	t.Cleanup(f.srv.Close)
	return f
}

func (f *fakeCharger) addr() (string, int) {
	h, p, _ := net.SplitHostPort(strings.TrimPrefix(f.srv.URL, "https://"))
	n, _ := strconv.Atoi(p)
	return h, n
}

func (f *fakeCharger) requests() []fakeReq {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]fakeReq(nil), f.reqs...)
}

func (f *fakeCharger) reset() {
	f.mu.Lock()
	f.reqs = nil
	f.mu.Unlock()
}

func (f *fakeCharger) serve(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)
	f.mu.Lock()
	f.reqs = append(f.reqs, fakeReq{r.Method, r.URL.Path, r.URL.RawQuery, string(body)})
	f.mu.Unlock()
	switch r.URL.Path {
	case "/api/login":
		var ld struct{ Password string }
		_ = json.Unmarshal(body, &ld)
		if ld.Password != "secret" {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		_, _ = io.WriteString(w, `{"access":"A1","refresh":"R1"}`)
	case "/api/logout":
	case "/api/categories":
		_, _ = io.WriteString(w, `{"version":1,"categories":["generic","generic2","comm"]}`)
	case "/api/prop":
		if r.Method == http.MethodPost {
			return
		}
		cat := r.URL.Query().Get("cat")
		var out []map[string]any
		for _, p := range f.props {
			if cat == "" || p["cat"] == cat {
				out = append(out, p)
			}
		}
		b, _ := json.Marshal(map[string]any{"version": 2, "total": len(out), "offset": 0, "count": len(out), "properties": out})
		_, _ = w.Write(b)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (f *fakeCharger) summary() string {
	var s []string
	for _, r := range f.requests() {
		e := r.Method + " " + r.Path
		if r.Query != "" {
			e += "?" + r.Query
		}
		s = append(s, e)
	}
	return strings.Join(s, "|")
}

// ---------------------------------------------------------------- helpers

func newTestState(t *testing.T) *State {
	t.Helper()
	a := test.NewTempApp(t)
	s := newMainWindow(a, Options{}, defaultRegistry)
	s.protocol = "https" // httptest serves TLS on a random port (C#: https only on 443)
	return s
}

func connectFake(t *testing.T, s *State, f *fakeCharger, password string) (core.LoginResult, error) {
	t.Helper()
	host, port := f.addr()
	var res core.LoginResult
	var err error
	called := false
	s.Connect(ConnectParams{Address: host, Port: port, Insecure: true, Username: core.UserLevelAdmin, Password: password},
		func(r core.LoginResult, e error) { res, err, called = r, e, true })
	s.Wait()
	if !called {
		t.Fatal("Connect never completed")
	}
	return res, err
}

// ------------------------------------------------------------------ tests

func TestRegistryOrdering(t *testing.T) {
	r := newRegistry()
	noop := func(*State) fyne.CanvasObject { return widget.NewLabel("") }
	for _, p := range []PanelSpec{
		{ID: "allprops", Order: OrderAllProperties, Build: noop},
		{ID: "info", Order: OrderInformation, Build: noop},
		{ID: "sockets", Order: OrderSockets, Build: noop},
		{ID: "b-same", Order: OrderLog, Build: noop},
		{ID: "a-same", Order: OrderLog, Build: noop},
		{ID: "devices", Order: OrderDiscover, Build: noop},
	} {
		r.registerPanel(p)
	}
	var ids []string
	for _, p := range r.panelList() {
		ids = append(ids, p.ID)
	}
	if got := strings.Join(ids, ","); got != "devices,sockets,info,a-same,b-same,allprops" {
		t.Fatalf("order = %s", got)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("duplicate ID must panic")
		}
	}()
	r.registerPanel(PanelSpec{ID: "info", Build: noop})
}

func TestFeatureOrderFollowsFillPanelList(t *testing.T) {
	// MainWindow constructor order of m_allPanels then m_allSCNPanels.
	want := []int{OrderSockets, OrderInformation, OrderPower, OrderNetwork, OrderWhitelist, OrderTransactions,
		OrderBackoffice, OrderUI, OrderAlerts, OrderLog, OrderStates, OrderAllProperties, OrderSCNOverview, OrderSCNSettings}
	for i := 1; i < len(want); i++ {
		if want[i] <= want[i-1] {
			t.Fatalf("order constants not increasing at %d", i)
		}
	}
	var ids []string
	for _, p := range Panels() {
		ids = append(ids, p.ID)
	}
	got := strings.Join(ids, ",")
	for _, pair := range [][2]string{{PanelIDDiscover, PanelIDInformation}, {PanelIDInformation, PanelIDProperties}, {PanelIDProperties, PanelIDFirmware}} {
		if strings.Index(got, pair[0]) > strings.Index(got, pair[1]) {
			t.Fatalf("%s must precede %s in %s", pair[0], pair[1], got)
		}
	}
	for _, p := range Panels() {
		if p.ID == PanelIDInformation && p.FeatureID != FeatureInformation {
			t.Fatal("information feature id")
		}
		if p.ID == PanelIDProperties && p.FeatureID != FeatureAllProperties {
			t.Fatal("all properties feature id")
		}
	}
}

func TestMenuRegistryOrder(t *testing.T) {
	r := newRegistry()
	act := func(*State) {}
	r.registerMenuItem(MenuItemSpec{Menu: "Help", Label: "About...", Action: act})
	r.registerMenuItem(MenuItemSpec{Menu: "Tools", Label: "X", Action: act})
	r.registerMenuItem(MenuItemSpec{Menu: "Device", Label: "Refresh", Order: 700, Action: act})
	r.registerMenuItem(MenuItemSpec{Menu: "Device", Label: "Upload new firmware...", Order: 500, Action: act})
	r.registerMenuItem(MenuItemSpec{Menu: "File", Label: "Close", Action: act})
	names, by := r.menuList()
	if strings.Join(names, ",") != "File,Device,Help,Tools" {
		t.Fatalf("menus = %v", names)
	}
	if by["Device"][0].Label != "Upload new firmware..." {
		t.Fatalf("device items = %+v", by["Device"])
	}
}

func TestRightsHideTabs(t *testing.T) {
	s := newTestState(t)
	has := func(title string) bool {
		for _, it := range s.tabs.Items {
			if it.Text == title {
				return true
			}
		}
		return false
	}
	if !has("Information") || !has("All Properties") || !has("Devices") {
		t.Fatal("default rights show every page")
	}
	s.SetRights(func(id string) Rights {
		if id == FeatureAllProperties {
			return RightsNone
		}
		return RightsFull
	})
	if has("All Properties") || !has("Information") {
		t.Fatal("RightsNone must hide the page")
	}
	s.ShowSCN("NET1")
	if has("Information") || !has("Devices") {
		t.Fatal("SCN mode hides device pages")
	}
}

func TestConnectDisconnect(t *testing.T) {
	f := newFakeCharger(t)
	s := newTestState(t)
	info := s.hosts[PanelIDInformation]
	if info.stack.Objects[0] != info.placeholder {
		t.Fatal("pages needing a device show PanelNotLoggedIn while disconnected")
	}
	connected, disconnected := 0, 0
	s.OnConnect(func(*State) { connected++ })
	s.OnDisconnect(func(*State) { disconnected++ })

	if _, err := connectFake(t, s, f, "secret"); err != nil {
		t.Fatal(err)
	}
	if !s.Connected() || connected != 1 || s.Client() == nil || s.Client().AccessToken != "A1" {
		t.Fatalf("connected=%v hooks=%d", s.Connected(), connected)
	}
	reqs := f.requests()
	if reqs[0].Path != "/api/login" || reqs[0].Body != `{"username":"admin","password":"secret","displayname":""}` {
		t.Fatalf("login request = %+v", reqs[0])
	}
	if got := f.summary(); got != "POST /api/login|GET /api/prop?cat=generic|GET /api/prop?cat=generic2|GET /api/prop?cat=comm" {
		t.Fatalf("requests = %s", got)
	}
	if info.stack.Objects[0] == info.placeholder {
		t.Fatal("content must replace the placeholder after login")
	}
	if !strings.Contains(s.connInfo.Text, "MYCHARGER") || s.bar.connect.Disabled() == false {
		t.Fatalf("conn info = %q", s.connInfo.Text)
	}
	iv := s.view(PanelIDInformation).(*informationView)
	if iv.head.Text != "MYCHARGER" || len(iv.body.Objects) == 0 {
		t.Fatalf("information view not rendered: %q", iv.head.Text)
	}

	f.reset()
	s.Disconnect(nil)
	s.Wait()
	if s.Connected() || disconnected != 1 || s.Session() != nil {
		t.Fatalf("after disconnect: connected=%v hooks=%d", s.Connected(), disconnected)
	}
	if got := f.summary(); got != "POST /api/logout" {
		t.Fatalf("requests = %s", got)
	}
	if info.stack.Objects[0] != info.placeholder {
		t.Fatal("placeholder must return after logout")
	}
}

func TestConnectWrongPassword(t *testing.T) {
	f := newFakeCharger(t)
	s := newTestState(t)
	res, err := connectFake(t, s, f, "nope")
	if err == nil || res.StatusCode != 403 || s.Connected() {
		t.Fatalf("res=%+v err=%v", res, err)
	}
	if want := "The password for device '' is incorrect! Please provide a valid password."; err.Error() != want || !strings.Contains(s.status.Text, want) {
		t.Fatalf("err=%q status=%q", err, s.status.Text)
	}
	if _, err := connectFake(t, s, f, ""); err == nil {
		t.Fatal("empty password is rejected by the device")
	}
	var gotErr error
	s.Connect(ConnectParams{Address: "not-an-ip", Port: 443}, func(_ core.LoginResult, e error) { gotErr = e })
	if gotErr == nil || !strings.HasPrefix(gotErr.Error(), "Invalid IP addres!") {
		t.Fatalf("validation err = %v", gotErr)
	}
}

func TestPropertyEditStoresBody(t *testing.T) {
	f := newFakeCharger(t)
	s := newTestState(t)
	if _, err := connectFake(t, s, f, "secret"); err != nil {
		t.Fatal(err)
	}
	f.reset()
	s.SelectPanel(PanelIDProperties) // OnPanelShown → PanelAllProperties.OnChangeDevice
	s.Wait()
	if got := f.summary(); got != "GET /api/categories|GET /api/categories|GET /api/prop?cat=generic|GET /api/prop?cat=generic2|GET /api/prop?cat=comm" {
		t.Fatalf("load requests = %s", got)
	}
	v := s.view(PanelIDProperties).(*propertiesView)
	if len(v.groups) != 3 || v.category.Selected != allCategories {
		t.Fatalf("groups = %d sel=%q", len(v.groups), v.category.Selected)
	}
	rowOf := func(id uint16) int {
		for i, r := range v.rows {
			if r.ID == id {
				return i
			}
		}
		t.Fatalf("row %04X not shown", id)
		return -1
	}
	if !s.saveBtn.Disabled() {
		t.Fatal("Save must be disabled without changes")
	}

	v.table.Select(widget.TableCellID{Row: rowOf(0x2053), Col: 0})
	if v.entry == nil || v.entry.Text != "MYCHARGER" || v.apply.Disabled() {
		t.Fatal("text editor for the identity")
	}
	v.entry.SetText("NEWID")
	test.Tap(v.apply)

	v.table.Select(widget.TableCellID{Row: rowOf(0x2062), Col: 0})
	if v.entry == nil || v.entry.Text != "16" {
		t.Fatal("number editor for the REAL32")
	}
	v.entry.SetText("25,5")
	test.Tap(v.apply)

	v.table.Select(widget.TableCellID{Row: rowOf(0x2050), Col: 0})
	if !v.apply.Disabled() {
		t.Fatal("8272 is forced read-only")
	}
	if s.saveBtn.Disabled() || s.revertBtn.Disabled() {
		t.Fatal("Save/Revert must be enabled after an edit")
	}

	f.reset()
	test.Tap(s.saveBtn)
	s.Wait()
	reqs := f.requests()
	if len(reqs) != 1 || reqs[0].Method != "POST" || reqs[0].Path != "/api/prop" {
		t.Fatalf("save requests = %s", f.summary())
	}
	if want := `{"2053_0":{"id":"2053_0","value":"NEWID"},"2062_0":{"id":"2062_0","value":25.5}}`; reqs[0].Body != want {
		t.Fatalf("body = %s\nwant  %s", reqs[0].Body, want)
	}
	if !s.saveBtn.Disabled() || s.Props().HasChanges() {
		t.Fatal("changes must be committed after Save")
	}
	if !strings.Contains(v.cell(v.rows[rowOf(0x2053)], 2), "NEWID") {
		t.Fatal("table shows the stored value")
	}
}

func TestInvalidIdentityBlocksSave(t *testing.T) {
	f := newFakeCharger(t)
	s := newTestState(t)
	if _, err := connectFake(t, s, f, "secret"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Props().SetValue(0x2053, 0, "bad id"); err != nil {
		t.Fatal(err)
	}
	s.NotifyPropertiesChanged()
	f.reset()
	var gotErr error
	s.SaveAllChanges(func(_ core.SaveResult, err error) { gotErr = err })
	s.Wait()
	if gotErr == nil || !strings.HasPrefix(gotErr.Error(), "The field \"Customer Ident. Number\" is invalid.") {
		t.Fatalf("err = %v", gotErr)
	}
	if n := len(f.requests()); n != 0 {
		t.Fatalf("no request may be sent, got %s", f.summary())
	}
}

func TestFirmwareClassificationView(t *testing.T) {
	s := newTestState(t)
	v := s.view(PanelIDFirmware).(*firmwareView)
	data, err := fwi.WrapDisplayPayload([]byte("objects"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "logo_1.0.0_A.fwi")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	v.file.SetText(path)
	s.Wait()
	for _, want := range []string{"File: logo_1.0.0_A.fwi", "Type: NG9xx firmware file (.fwi)", "Variant: display resource (installer-built, known key); unwrap OK, 7 bytes"} {
		if !strings.Contains(v.report.Text, want) {
			t.Errorf("report lacks %q:\n%s", want, v.report.Text)
		}
	}
	if !v.upload.Disabled() {
		t.Fatal("upload needs a logged-in device")
	}
	fixture := "../../../firmware/Firmware/NG9xx 7.4.6-4416.fwi"
	if _, err := os.Stat(fixture); err != nil {
		t.Skipf("fixture missing: %v", err)
	}
	v.file.SetText(fixture)
	s.Wait()
	if !strings.Contains(v.report.Text, "Variant: controller firmware (device-held key)") {
		t.Fatalf("fixture report:\n%s", v.report.Text)
	}
}

func TestDiscoverFillsConnectionBar(t *testing.T) {
	s := newTestState(t)
	v := s.view(PanelIDDiscover).(*discoverView)
	if !v.empty.Visible() {
		t.Fatal("PanelNoDevice text when the list is empty")
	}
	s.Devices.Upsert(core.Device{Identity: "CP-B", Address: "10.1.1.2", SCNNetwork: "NET", Discovered: true})
	s.Devices.Upsert(core.Device{Identity: "CP-A", Address: "10.1.1.3", Discovered: true})
	v.refreshList()
	if v.empty.Visible() || len(v.rows) != 2 || v.rows[0].Identity != "CP-B" {
		t.Fatalf("rows = %+v", v.rows)
	}
	v.table.Select(widget.TableCellID{Row: 1, Col: 0})
	if s.bar.ip.Text != "10.1.1.3" || s.bar.port.Text != "443" || v.connect.Disabled() {
		t.Fatalf("bar ip=%q port=%q", s.bar.ip.Text, s.bar.port.Text)
	}
	v.filter.SetText("cp-b")
	if len(v.rows) != 1 {
		t.Fatalf("filtered rows = %d", len(v.rows))
	}
	if discoverCell(core.Device{Address: "1.2.3.4"}, 1) != "1.2.3.4 (manual)" {
		t.Fatal("manual marker")
	}
}

func TestRunAsyncMarshalsBack(t *testing.T) {
	s := newTestState(t)
	got := ""
	s.RunAsync(func() { got = "work" }, func() { got += "+done"; s.SetStatus("%s", got) })
	s.Wait()
	if got != "work+done" || s.status.Text != "work+done" || s.activity.Visible() {
		t.Fatalf("got=%q status=%q", got, s.status.Text)
	}
}

func ExampleRegisterPanel() {
	// A later panel file only needs an init():
	_ = PanelSpec{
		ID: "power", Title: "Power", Order: OrderPower, FeatureID: FeaturePower, RequiresConnection: true,
		Build: func(s *State) fyne.CanvasObject { return widget.NewLabel("…") },
	}
	fmt.Println("ok")
	// Output: ok
}
