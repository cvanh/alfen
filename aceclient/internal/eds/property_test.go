package eds

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"alfen/aceclient/internal/api"
)

func TestMerge(t *testing.T) {
	d := Default()
	tests := []struct {
		p                     api.Property
		icuName, title, units string
		hasParam, readOnly    bool
	}{
		{prop(0x2050, 0, api.SDTVisibleString, "Twin 4.0", "generic"), "OD_sysChargePointModel", "Model", "", true, false},
		{prop(0x205A, 0, api.SDTInteger8, "1", "generic"), "OD_sysTimeZone", "Time Zone", "hours", true, false},
		{prop(0x2300, 1, api.SDTUnsigned8, "1", "leds"), "OD_ledsStateUnknown:red1", "2300_1", "", true, false}, // untitled EDS entry
		{prop(0x7F00, 0xA, api.SDTUnsigned8, "1", "x"), "7F00_A", "7F00_A", "", false, false},
		{prop(0x7F00, 0, api.SDTLowLimit, "0", "x"), "7F00_0", "7F00_0", "", false, true},
	}
	for _, tc := range tests {
		m := d.Merge(tc.p)
		if m.ICUName != tc.icuName || m.Title != tc.title || m.Units != tc.units || (m.Param != nil) != tc.hasParam || m.ReadOnly != tc.readOnly {
			t.Errorf("Merge(%s) = name %q title %q units %q param %v ro %v", tc.p.IDSub(), m.ICUName, m.Title, m.Units, m.Param != nil, m.ReadOnly)
		}
		if d.Title(tc.p) != tc.title || d.ICUName(tc.p) != tc.icuName {
			t.Errorf("Title/ICUName(%s)", tc.p.IDSub())
		}
	}
	// Device type wins over the EDS DataType (ParseProperty overwrites it).
	if m := d.Merge(prop(0x2050, 0, api.SDTUnsigned8, "3", "generic")); m.DataType != api.SDTUnsigned8 || m.Typed.Kind != ValueUint {
		t.Errorf("device type must win: %+v", m)
	}
}

func TestDictionaryOrder(t *testing.T) {
	d := Default()
	in := []api.Property{
		prop(0x7F02, 0, api.SDTUnsigned8, "1", "x"),
		prop(0x205B, 0, api.SDTInteger8, "1", "g"),
		prop(0x7F01, 0, api.SDTUnsigned8, "1", "x"),
		prop(0x2050, 0, api.SDTVisibleString, "a", "g"),
		prop(0x7F02, 0, api.SDTUnsigned8, "9", "x"),
		prop(0x2522, 1, api.SDTVisibleString, "e", "g"), // duplicated in EDS: listed once
	}
	got := d.DictionaryOrder(in)
	want := []string{"2050_0", "205B_0", "2522_1", "7F02_0", "7F01_0"}
	if len(got) != len(want) {
		t.Fatalf("got %d entries", len(got))
	}
	for i, w := range want {
		if got[i].IDSub() != w {
			t.Errorf("%d: %s, want %s", i, got[i].IDSub(), w)
		}
	}
	if got[3].Value != "9" {
		t.Errorf("repeated id must keep the last data, got %q", got[3].Value)
	}
}

func TestLabelText(t *testing.T) {
	d := Default()
	tests := []struct {
		id      uint16
		sub     byte
		custom  string
		want    string
		unknown bool
	}{
		{0x205A, 0, "", "Time Zone (hours)", false},
		{0x2050, 0, "", "Model", false},
		{0x2221, 0x10, "", "Frequency (Hz)", false},
		{0x7F00, 0, "", "7F00", true},
		{0x7F00, 10, "", "7F00_10", true}, // decimal sub
		{0, 0, "Custom", "Custom", false},
		{0x2300, 1, "", "", false},
	}
	for _, tc := range tests {
		got, unk := d.LabelText(tc.id, tc.sub, tc.custom)
		if got != tc.want || unk != tc.unknown {
			t.Errorf("LabelText(%X,%d) = %q,%v want %q,%v", tc.id, tc.sub, got, unk, tc.want, tc.unknown)
		}
	}
}

func TestPropertyStrings(t *testing.T) {
	d := Default()
	tests := []struct {
		p          api.Property
		str, devic string
	}{
		{prop(0x205B, 0, api.SDTInteger8, "2", "g"), "USA", "USA"},
		{prop(0x205B, 0, api.SDTInteger8, "9", "g"), "9", "9"},
		{prop(0x2050, 0, api.SDTVisibleString, "Twin 4.0", "g"), "Twin 4.0", "Twin 4.0"},
		{prop(0x2221, 0x10, api.SDTReal32, "49.98", "m"), "49.98", "49.980"},
		{prop(0x7F00, 0, api.SDTReal64, "2", "m"), "2", "2.000"},
		{prop(0x7F00, 0, api.SDTBoolean, "true", "m"), "True", "True"},
		{prop(0x7F00, 0, api.SDTDomain, "x", "m"), "", ""},
	}
	for _, tc := range tests {
		if got := d.PropertyString(tc.p); got != tc.str {
			t.Errorf("PropertyString(%s) = %q, want %q", tc.p.IDSub(), got, tc.str)
		}
		if got := d.DeviceValueString(tc.p); got != tc.devic {
			t.Errorf("DeviceValueString(%s) = %q, want %q", tc.p.IDSub(), got, tc.devic)
		}
	}
	child := prop(0x7F00, 0, api.SDTInteger8, "1", "g")
	if got := d.PropertyStringParent(child, 0x205B); got != "EU" {
		t.Errorf("PropertyStringParent = %q", got)
	}
}

func TestIDHelpers(t *testing.T) {
	ids := []struct {
		in  string
		id  uint16
		sub byte
		ok  bool
	}{
		{"2050_0", 0x2050, 0, true},
		{"2221_10", 0x2221, 0x10, true},
		{"1_2_3", 1, 2, true},
		{"zz_1", 0, 1, true},
		{"2050", 0, 0, false},
		{"10000_0", 0, 0, false},
		{"1_100", 0, 0, false},
	}
	for _, tc := range ids {
		id, sub, ok := ParseIDSub(tc.in)
		if id != tc.id || sub != tc.sub || ok != tc.ok {
			t.Errorf("ParseIDSub(%q) = %X,%X,%v", tc.in, id, sub, ok)
		}
	}
	if id, sub := SplitCombinedID(2125312); id != 0x206E || sub != 0 {
		t.Errorf("SplitCombinedID(2125312) = %X,%X", id, sub)
	}
	if id, sub := SplitCombinedID(0x218005); id != 0x2180 || sub != 5 {
		t.Errorf("SplitCombinedID(0x218005) = %X,%X", id, sub)
	}
	if id, sub := SplitCombinedID(0x2050); id != 0x2050 || sub != 0 {
		t.Errorf("SplitCombinedID(0x2050) = %X,%X", id, sub)
	}
	if FeatureRightID(0x2050) != "ID_2050" || FeatureRightID(5) != "ID_0005" {
		t.Error("FeatureRightID")
	}
	if IDSubPadded(0x2050, 0xA) != "2050_0A" || ODIndex(0x2050, 0xA) != "2050_A" || searchID(0x50, 1) != "0x0050_1" {
		t.Error("id formats")
	}
}

// TestStorePropertyWire checks that StoreProperty feeds api.StoreProperties a
// body byte-identical to ICULanDevice.StoreProperties: ICUName keys, per-type
// quoting, .NET value text.
func TestStorePropertyWire(t *testing.T) {
	var method, path, query, body string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		method, path, query, body = r.Method, r.URL.Path, r.URL.RawQuery, string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{}`)
	}))
	defer srv.Close()
	host, portStr, err := net.SplitHostPort(srv.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, _ := strconv.Atoi(portStr)
	c := api.New(host, port, true)

	d := Default()
	model := d.Merge(prop(0x2050, 0, api.SDTVisibleString, "Twin 3.0", "generic"))
	tz := d.Merge(prop(0x205A, 0, api.SDTInteger8, "1", "generic"))
	freq := d.Merge(prop(0x2221, 0x10, api.SDTReal32, "50", "meter"))
	flag := d.Merge(prop(0x7F00, 4, api.SDTBoolean, "false", "x"))
	bytesP := d.Merge(prop(0x7F00, 5, api.SDTByteArray, "1,2", "x"))

	tzField, _ := d.buildField(tz, false)
	_, tzVal, err := tzField.CommitNumber("-2")
	if err != nil {
		t.Fatal(err)
	}
	flagField, _ := d.buildField(flag, false)
	flagVal, err := flagField.CommitCheckbox(true)
	if err != nil {
		t.Fatal(err)
	}
	err = c.StoreProperties(
		d.StoreProperty(model, ParseValue(model.DataType, "Twin 4.0")),
		d.StoreProperty(tz, tzVal),
		d.StoreProperty(freq, CommitNumber(freq.DataType, 49.98, 1)),
		d.StoreProperty(flag, flagVal),
		d.StoreProperty(bytesP, ParseValue(bytesP.DataType, "a,ff")),
	)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"OD_sysChargePointModel":{"id":"2050_0","value":"Twin 4.0"},` +
		`"OD_sysTimeZone":{"id":"205A_0","value":-2},` +
		`"OD_sensEnergyMeterValues:Frequency":{"id":"2221_10","value":49.98},` +
		`"7F00_4":{"id":"7F00_4","value":"True"},` +
		`"7F00_5":{"id":"7F00_5","value":"0A,FF"}}`
	if method != http.MethodPost || path != "/api/prop" || query != "" || body != want {
		t.Errorf("request = %s %s?%s\n got %s\nwant %s", method, path, query, body, want)
	}
}

// TestCategoriesEndpointShape feeds ParseCategories the body a GET
// /api/categories returns through api.Exec.
func TestCategoriesEndpointShape(t *testing.T) {
	var path, method string
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, method = r.URL.Path, r.Method
		_, _ = io.WriteString(w, `{"version":2,"categories":["generic","display","leds"]}`)
	}))
	defer srv.Close()
	host, portStr, _ := net.SplitHostPort(srv.Listener.Addr().String())
	port, _ := strconv.Atoi(portStr)
	resp, err := api.New(host, port, true).Exec("categories", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if method != http.MethodGet || path != "/api/categories" {
		t.Errorf("request = %s %s", method, path)
	}
	got := ParseCategories(resp.Body)
	if len(got) != 3 || got[0] != "generic" || got[2] != "leds" {
		t.Errorf("categories = %q", got)
	}
}

func TestObjectID(t *testing.T) {
	ok := []string{"12345r12", "123456r123", "ACE1234567", " ace1234567 ", "12345R12"}
	for _, in := range ok {
		acc, chk, err := ValidateObjectID(in)
		if err != nil || acc != in {
			t.Errorf("ValidateObjectID(%q) = %q,%q,%v", in, acc, chk, err)
		}
	}
	bad := []struct{ in, msg string }{
		{"1234r12", "Invalid Object ID: 1234r12!\nPlease provide a correct one."},
		{"ACE123456", "Invalid Object ID: ace123456!\nPlease provide a correct one."},
		{" 1234567r12 ", "Invalid Object ID: 1234567r12!\nPlease provide a correct one."},
		{"", "Invalid Object ID: !\nPlease provide a correct one."},
	}
	for _, tc := range bad {
		if _, _, err := ValidateObjectID(tc.in); err == nil || err.Error() != tc.msg {
			t.Errorf("ValidateObjectID(%q) err = %v", tc.in, err)
		}
	}

	var asked, shown []string
	yes := func(msg string) bool { asked = append(asked, msg); return true }
	no := func(msg string) bool { asked = append(asked, msg); return false }
	show := func(msg string) { shown = append(shown, msg) }
	reach := func() bool { return true }
	if !CheckObjectIDExists("ace1234567", ObjectIDCheck{LoggedIn: false, Reachable: reach, Ask: yes}) ||
		asked[0] != "Unable to verify the entered Object ID with ISAH. Are you sure to change the Object ID to ace1234567?" {
		t.Errorf("not logged in: %q", asked)
	}
	asked = nil
	if !CheckObjectIDExists("ace1234567", ObjectIDCheck{LoggedIn: true, Reachable: reach, Lookup: func(string) bool { return true }, Ask: no, Show: show}) ||
		len(asked) != 0 || shown[0] != "The Object ID of the Charge Station will be changed to ace1234567." {
		t.Errorf("known object: %q %q", asked, shown)
	}
	if CheckObjectIDExists("ace1234567", ObjectIDCheck{LoggedIn: true, Reachable: reach, Lookup: func(string) bool { return false }, Ask: no}) ||
		asked[0] != "The entered Object ID seems to be invalid. Are you sure to change the Object ID to ace1234567?" {
		t.Errorf("unknown object: %q", asked)
	}
	if CheckObjectIDExists("x", ObjectIDCheck{LoggedIn: true, Reachable: func() bool { return false }}) {
		t.Error("unreachable without Ask must not accept")
	}
}
