package presets

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"

	"alfen/aceclient/internal/api"
)

func TestNewDictionaryOrderAndNames(t *testing.T) {
	params := []DataSheetParameter{
		{ID: 0x2129, Sub: 0, DataType: api.SDTReal32, Name: "mainNormalMaxCurrent"},
		{ID: 0x3000, Sub: 0, DataType: api.SDTUnsigned8, Name: "edsOnly"},
	}
	dev := []api.Property{
		{ID: 0x2200, Sub: 1, Name: "devOnly", DataType: api.SDTUnsigned16, Value: "7"},
		{ID: 0x2129, Sub: 0, Name: "fromDevice", DataType: api.SDTReal32, Value: "16", Category: "cat", MaxLength: 4},
		{ID: 0x2201, Sub: 0, DataType: api.SDTUnsigned8, Value: "1"},
	}
	d := NewDictionary(params, dev)
	if d.Len() != 4 {
		t.Fatalf("Len = %d", d.Len())
	}
	var keys []string
	for _, e := range d.entries {
		keys = append(keys, fmt.Sprintf("%04X_%X:%s", e.id, e.sub, e.name))
	}
	want := "2129_0:OD_mainNormalMaxCurrent 3000_0:OD_edsOnly 2200_1:devOnly 2201_0:2201_0"
	if got := strings.Join(keys, " "); got != want {
		t.Errorf("order/names = %q, want %q", got, want)
	}
	p, ok := d.Property(0x2129, 0)
	if !ok || p.Value != "16" || p.Category != "cat" || p.MaxLength != 4 || p.DataType != api.SDTReal32 {
		t.Errorf("Property(2129_0) = %+v", p)
	}
	if e := d.find(0x3000, 0); e.value != nil || e.device != nil {
		t.Error("EDS-only entry must keep a null value")
	}
}

func TestLookupIDSub(t *testing.T) {
	d := NewDictionary(nil, []api.Property{
		{ID: 0x2129, Sub: 0, DataType: api.SDTReal32, Value: "1"},
		{ID: 0, Sub: 0, DataType: api.SDTUnsigned8, Value: "1"},
	})
	tests := []struct {
		in      string
		key     PropKey
		found   bool
		wantErr bool
	}{
		{"2129_0", PropKey{0x2129, 0}, true, false},
		{"2129_00", PropKey{0x2129, 0}, true, false},
		{" 2129 _0_extra", PropKey{0x2129, 0}, true, false},
		{"2129", PropKey{}, false, false},
		{"zz_zz", PropKey{0, 0}, true, false}, // failed parses leave 0
		{"2129_1", PropKey{0x2129, 1}, false, false},
		{"10000_0", PropKey{}, false, true},
		{"FFFFFFFF_0", PropKey{}, false, true}, // int -1 -> ToUInt16 overflow
		{"2129_100", PropKey{}, false, true},
	}
	for _, tt := range tests {
		key, found, err := d.LookupIDSub(tt.in)
		if (err != nil) != tt.wantErr || (!tt.wantErr && (key != tt.key || found != tt.found)) {
			t.Errorf("LookupIDSub(%q) = %v,%v,%v", tt.in, key, found, err)
		}
	}
}

func TestSetValueChangedAndRevert(t *testing.T) {
	d := NewDictionary(
		[]DataSheetParameter{{ID: 0x3000, DataType: api.SDTUnsigned8, Name: "edsOnly"}},
		[]api.Property{
			{ID: 1, DataType: api.SDTUnsigned8, Value: "5"},
			{ID: 2, DataType: api.SDTReal32, Value: "0.333333343"},
			{ID: 3, DataType: api.SDTByteArray, Value: "01,02"},
			{ID: 4, DataType: api.SDTUnicodeString, Value: "x"},
			{ID: 5, DataType: api.SDTBoolean, Value: "true"},
		})
	steps := []struct {
		id      uint16
		text    string
		changed bool
	}{
		{1, " 5 ", false}, // same typed value
		{1, "6", true},    // different
		{2, "0.3333333", true},
		{3, "01,02", true}, // arrays always differ by reference
		{4, "y", false},    // unhandled type: value stays null-ish/unchanged
		{5, "True", false}, // bool parse is case-insensitive
		{0x3000, "1", true},
	}
	for _, s := range steps {
		changed, ok := d.SetValue(s.id, 0, s.text)
		if !ok || changed != s.changed {
			t.Errorf("SetValue(%d, %q) = %v,%v want %v", s.id, s.text, changed, ok, s.changed)
		}
	}
	if _, ok := d.SetValue(99, 0, "1"); ok {
		t.Error("SetValue on a missing entry must report !ok")
	}
	d.RevertChanges()
	if d.IsChanged(1, 0) || d.IsChanged(3, 0) {
		t.Error("Rollback must restore 1 and 3")
	}
	// Rollback re-parses float.ToString() ("G" = 7 digits): 0.333333343f
	// becomes 0.3333333f and stays changed, as in the C#.
	if !d.IsChanged(2, 0) {
		t.Error("REAL32 rollback quirk not reproduced")
	}
	// DeviceValue null: Rollback is a no-op.
	if !d.IsChanged(0x3000, 0) {
		t.Error("EDS-only entry must stay changed after RevertChanges")
	}
}

// propServer is an HTTPS stand-in for the charger that records POST /api/prop.
type propServer struct {
	mu     sync.Mutex
	bodies []string
	fail   map[int]bool // 1-based request numbers answered with 500
	srv    *httptest.Server
}

func newPropServer(t *testing.T, fail ...int) (*propServer, *api.Client) {
	t.Helper()
	ps := &propServer{fail: map[int]bool{}}
	for _, f := range fail {
		ps.fail[f] = true
	}
	ps.srv = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if r.Method != http.MethodPost || r.URL.Path != "/api/prop" || r.URL.RawQuery != "" {
			t.Errorf("unexpected request %s %s?%s", r.Method, r.URL.Path, r.URL.RawQuery)
		}
		if ct := r.Header.Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
			t.Errorf("Content-Type = %q", ct)
		}
		ps.mu.Lock()
		ps.bodies = append(ps.bodies, string(body))
		n := len(ps.bodies)
		ps.mu.Unlock()
		if ps.fail[n] {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("{}"))
	}))
	t.Cleanup(ps.srv.Close)
	u, _ := url.Parse(ps.srv.URL)
	port, _ := strconv.Atoi(u.Port())
	return ps, api.New(u.Hostname(), port, true)
}

func TestStoreChangedPropertiesBody(t *testing.T) {
	d := NewDictionary(
		[]DataSheetParameter{{ID: 0x2129, DataType: api.SDTReal32, Name: "mainNormalMaxCurrent"}},
		[]api.Property{
			{ID: 0x2129, Sub: 0, Name: "x", DataType: api.SDTReal32, Value: "16"},
			{ID: 0x2068, Sub: 1, Name: "boolProp", DataType: api.SDTBoolean, Value: "false"},
			{ID: 0x206A, Sub: 0, Name: "bytes", DataType: api.SDTByteArray, Value: "00"},
			{ID: 0x2060, Sub: 0, Name: "str", DataType: api.SDTVisibleString, Value: "a"},
			{ID: 0x2064, Sub: 0, Name: "i16", DataType: api.SDTInteger16, Value: "0"},
			{ID: 0x2065, Sub: 0, Name: "same", DataType: api.SDTUnsigned32, Value: "9"},
		})
	for _, s := range []struct {
		id   uint16
		sub  byte
		text string
	}{{0x2129, 0, "20,5"}, {0x2068, 1, "TRUE"}, {0x206A, 0, "a,ff"}, {0x2060, 0, "b"}, {0x2064, 0, "-3"}, {0x2065, 0, "9"}} {
		d.SetValue(s.id, s.sub, s.text)
	}
	ps, c := newPropServer(t)
	if err := d.StoreChangedProperties(c); err != nil {
		t.Fatal(err)
	}
	want := `{"OD_mainNormalMaxCurrent":{"id":"2129_0","value":20.5},` +
		`"boolProp":{"id":"2068_1","value":"True"},` +
		`"bytes":{"id":"206A_0","value":"0A,FF"},` +
		`"str":{"id":"2060_0","value":"b"},` +
		`"i16":{"id":"2064_0","value":-3}}`
	if len(ps.bodies) != 1 || ps.bodies[0] != want {
		t.Errorf("bodies = %q\nwant %q", ps.bodies, want)
	}
	if len(d.ChangedProperties()) != 0 {
		t.Error("stored properties must be committed")
	}
}

func TestStoreChangedPropertiesBatches(t *testing.T) {
	var props []api.Property
	for i := 0; i < 16; i++ {
		props = append(props, api.Property{ID: uint16(0x4000 + i), Name: fmt.Sprintf("p%d", i), DataType: api.SDTUnsigned16, Value: "0"})
	}
	d := NewDictionary(nil, props)
	for i := 0; i < 16; i++ {
		d.SetValue(uint16(0x4000+i), 0, "1")
	}
	ps, c := newPropServer(t, 2)
	if err := d.StoreChangedProperties(c); err == nil {
		t.Fatal("expected the second batch to fail")
	}
	if len(ps.bodies) != 2 || strings.Count(ps.bodies[0], `"id"`) != 15 || strings.Count(ps.bodies[1], `"id"`) != 1 {
		t.Fatalf("batching wrong: %q", ps.bodies)
	}
	changed := d.ChangedProperties()
	if len(changed) != 1 || changed[0].ID != 0x400F || changed[0].Value != "1" {
		t.Errorf("only the failed batch may remain changed, got %+v", changed)
	}
}
