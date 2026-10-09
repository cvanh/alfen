package eds

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"alfen/aceclient/internal/api"
)

// firmwareEDS is the installer's shipped EDS.xml, relative to this package.
const firmwareEDS = "../../../firmware/msi_work/files3/EDS.xml"

func TestEmbeddedMatchesFirmware(t *testing.T) {
	data, err := os.ReadFile(firmwareEDS)
	if err != nil {
		t.Skipf("fixture not available: %v", err)
	}
	if !bytes.Equal(data, EmbeddedXML()) {
		t.Fatal("embedded EDS.xml differs from firmware/msi_work/files3/EDS.xml")
	}
}

func TestLoadFileMatchesDefault(t *testing.T) {
	if _, err := os.Stat(firmwareEDS); err != nil {
		t.Skipf("fixture not available: %v", err)
	}
	d, err := LoadFile(firmwareEDS)
	if err != nil {
		t.Fatal(err)
	}
	def := Default()
	if d.Len() != def.Len() {
		t.Fatalf("Len %d != %d", d.Len(), def.Len())
	}
	for i, p := range d.Parameters() {
		q := def.Parameters()[i]
		if p.RawID != q.RawID || p.Name != q.Name || p.Title != q.Title || len(p.Options) != len(q.Options) {
			t.Fatalf("param %d differs: %+v vs %+v", i, p, q)
		}
	}
	f, err := os.Open(firmwareEDS)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	d2, err := Load(f)
	if err != nil || d2.Len() != def.Len() {
		t.Fatalf("Load: %v, %d", err, d2.Len())
	}
}

// TestDefaultStructure checks the whole-file shape of the shipped EDS.xml:
// 695 <Object>, 289 <Option> (13 without <Title>), 321 titled objects, 124
// with <Units>, no Length attribute.
func TestDefaultStructure(t *testing.T) {
	d := Default()
	if d.Len() != 695 {
		t.Fatalf("Len = %d, want 695", d.Len())
	}
	var options, untitledOptions, titled, units, withOptions int
	dataTypes := map[int]int{}
	access := map[string]int{}
	for _, p := range d.Parameters() {
		if p.HasTitle {
			titled++
		}
		if p.HasUnits {
			units++
		}
		if p.Options != nil {
			withOptions++
		}
		if p.MaxLength != DefaultMaxLength {
			t.Errorf("%s: MaxLength %d", p.RawID, p.MaxLength)
		}
		for i, o := range p.Options {
			options++
			if !o.HasTitle {
				untitledOptions++
			}
			if o.Index != i {
				t.Errorf("%s option %d has Index %d", p.RawID, i, o.Index)
			}
		}
		dataTypes[p.DataType]++
		access[p.AccessType]++
		if p.ReadWrite != (p.AccessType == "rw") {
			t.Errorf("%s: ReadWrite %v for %q", p.RawID, p.ReadWrite, p.AccessType)
		}
	}
	checks := []struct {
		name      string
		got, want int
	}{
		{"options", options, 289},
		{"untitled options", untitledOptions, 13},
		{"titled objects", titled, 321},
		{"objects with units", units, 124},
		{"objects with options", withOptions, 81},
		{"UNSIGNED8", dataTypes[0x05], 414},
		{"REAL32", dataTypes[0x08], 82},
		{"VISIBLE_STRING", dataTypes[0x09], 67},
		{"INTEGER8", dataTypes[0x02], 55},
		{"UNSIGNED16", dataTypes[0x06], 30},
		{"INTEGER16", dataTypes[0x03], 25},
		{"DOMAIN", dataTypes[0x0F], 11},
		{"UNSIGNED32", dataTypes[0x07], 6},
		{"INTEGER32", dataTypes[0x04], 4},
		{"UNSIGNED64", dataTypes[0x1B], 1},
		{"rw", access["rw"], 578},
		{"ro", access["ro"], 92},
		{"r", access["r"], 11},
		{"rww", access["rww"], 7},
		{"wo", access["wo"], 5},
		{"rwr", access["rwr"], 2},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %d, want %d", c.name, c.got, c.want)
		}
	}
}

func TestSysChargePointModel(t *testing.T) {
	p := Default().FindParameter(0x2050, 0)
	if p == nil {
		t.Fatal("2050 not found")
	}
	if p.Name != "sysChargePointModel" || p.Title != "Model" || p.SDT() != api.SDTVisibleString ||
		!p.ReadWrite || p.AccessType != "rw" || p.RawDataType != "0x0009" || p.HasUnits {
		t.Fatalf("unexpected 2050: %+v", p)
	}
	want := []string{"Twin 3.0", "EVe-dual", "EVe-single", "Compact", "LOLO3", "ICU Eve Mini", "TUBE", "Twin 4.0"}
	if len(p.Options) != len(want) {
		t.Fatalf("options = %d, want %d", len(p.Options), len(want))
	}
	for i, w := range want {
		o := p.Options[i]
		if o.Value != w || o.Title != w || o.Index != i || !o.HasTitle {
			t.Errorf("option %d = %+v, want %q", i, o, w)
		}
	}
}

func TestKnownParameters(t *testing.T) {
	d := Default()
	tests := []struct {
		id, sub  int
		name     string
		title    string
		units    string
		dataType int
		access   string
		options  int
	}{
		{0x1008, 0, "Manufacturer device name", "Device Name", "", 0x09, "ro", 0},
		{0x2052, 1, "sysBoardSerial:Value", "Ethernet MAC address", "", 0x09, "rw", 0},
		{0x205A, 0, "sysTimeZone", "Time Zone", "hours", 0x02, "rww", 0},
		{0x205B, 0, "sysDaylightSavings", "Daylight Savings", "", 0x02, "rww", 4},
		{0x2059, 0, "sysDateTime", "Charger Date/Time", "", 0x1B, "rww", 0},
		{0x206E, 0, "sysTimeZoneMinutes", "Time Zone", "minutes", 0x02, "rw", 0},
		{0x2221, 0x10, "", "", "", -1, "", -1}, // "sub10" is hex: SubId 16
		{0x2072, 2, "commDHCPaddress:isFixed", "Fixed DHCP server Address", "", 0x05, "rw", 1},
		{0x2522, 2, "SlaveType", "Mode", "", 0x09, "rw", 2},
		{0x2701, 0, "securitySSLPreSharedKey", "", "", 0x0F, "wo", 0},
	}
	for _, tc := range tests {
		p := d.FindParameter(tc.id, tc.sub)
		if p == nil {
			t.Errorf("%X sub %X not found", tc.id, tc.sub)
			continue
		}
		if tc.name == "" {
			continue
		}
		if p.Name != tc.name || (tc.title != "" && p.Title != tc.title) || p.Units != tc.units ||
			p.DataType != tc.dataType || p.AccessType != tc.access || len(p.Options) != tc.options {
			t.Errorf("%X sub %X = %+v", tc.id, tc.sub, p)
		}
	}
	// Untitled option and whitespace kept verbatim, entities decoded.
	o := d.FindParameter(0x2072, 2).Options[0]
	if o.Value != "0" || o.HasTitle || o.Title != "" {
		t.Errorf("2072sub2 option = %+v", o)
	}
	if p := d.FindParameter(0x2703, 0); p == nil || p.Title != "Security Features: " || p.Options[0].Title != "AUTH_DOMAIN " {
		t.Errorf("2703 = %+v", p)
	}
	if p := d.FindParameter(0x2902, 0); p == nil || p.Title != "   End Time" {
		t.Errorf("2902 = %+v", p)
	}
	found := false
	for _, p := range d.Parameters() {
		for _, o := range p.Options {
			if o.Title == "Plug & Charge" {
				found = true
			}
		}
	}
	if !found {
		t.Error("&amp; not decoded in option titles")
	}
}

func TestFindParameterDuplicatesFirstWins(t *testing.T) {
	d := Default()
	if p := d.FindParameter(0x2180, 6); p == nil || p.Name != "SCN_UnconnectedSafeCurrent" {
		t.Errorf("2180sub6 = %+v, want the first (SCN_UnconnectedSafeCurrent)", p)
	}
	n := 0
	for _, p := range d.Parameters() {
		if p.ID == 0x2522 && p.SubID == 1 {
			n++
		}
	}
	if n != 2 {
		t.Errorf("2522sub1 occurrences = %d, want 2", n)
	}
	var nilDict *Dictionary
	if nilDict.FindParameter(0x2050, 0) != nil || nilDict.Len() != 0 {
		t.Error("nil dictionary must find nothing")
	}
}

func TestSelectParameterOption(t *testing.T) {
	d := Default()
	tests := []struct {
		id, sub int
		desc    string
		want    int
		ok      bool
	}{
		{9506, 2, "Socomec", 1, true}, // PanelLoadbalancing: SelectParameterOption(9506, 2, "Socomec")
		{9506, 2, "Custom register mapping", 2, true},
		{9506, 2, "socomec", 0, false},              // Title compare is ordinal
		{0x2050, 0, "Twin 4.0", 0, true},            // non-numeric Value -> 0
		{0x205B, 0, "Australia", 3, true},           // INTEGER8 options
		{0x3284, 0, "", 0, false},                   // no option titled ""
		{0x7FFF, 0, "anything", 0, false},           // unknown parameter
		{0x2072, 2, "", 0, false},                   // untitled option is not Title == ""
		{0x3284, 0, firstTitle(0x3284, 2), 0, true}, // Value "0xFF" does not int.TryParse
	}
	for _, tc := range tests {
		got, ok := d.SelectParameterOption(tc.id, tc.sub, tc.desc)
		if got != tc.want || ok != tc.ok {
			t.Errorf("SelectParameterOption(%X,%d,%q) = %d,%v want %d,%v", tc.id, tc.sub, tc.desc, got, ok, tc.want, tc.ok)
		}
	}
}

func firstTitle(id, idx int) string {
	p := Default().FindParameter(id, 0)
	if p == nil || idx >= len(p.Options) {
		return "\x00missing"
	}
	return p.Options[idx].Title
}

func TestParseSemantics(t *testing.T) {
	const xmlDoc = "\xEF\xBB\xBF" + `<?xml version="1.0" encoding="utf-8"?>
<CANopen>
  <Object Id="2abcsub1F" ParameterName="a" DataType="0X1b" AccessType="RW" Length="12">
    <Option Value="7"><Title>first-from-option</Title></Option>
    <Title>direct</Title>
    <Units></Units>
  </Object>
  <Group><Object Id="zz" ParameterName="b" DataType="nothex" AccessType="rww"><Title>   </Title></Object></Group>
  <Object Id="FFFFFFFF" ParameterName="c" DataType="0x0005" AccessType="ro"/>
</CANopen>`
	d, err := Parse([]byte(xmlDoc))
	if err != nil {
		t.Fatal(err)
	}
	if d.Len() != 3 {
		t.Fatalf("Len = %d (nested <Object> must be found by Descendants)", d.Len())
	}
	a := d.Parameters()[0]
	// Descendants("Title").First() is the option's title when it comes first.
	if a.ID != 0x2abc || a.SubID != 0x1F || a.DataType != 27 || !a.ReadWrite || a.MaxLength != 12 ||
		a.Title != "first-from-option" || !a.HasUnits || a.Units != "" || len(a.Options) != 1 {
		t.Errorf("a = %+v", a)
	}
	if l, _ := d.LabelText(0x2abc, 0x1F, ""); l != "first-from-option ()" {
		t.Errorf("LabelText = %q", l)
	}
	b := d.Parameters()[1]
	if b.ID != 0 || b.SubID != 0 || b.DataType != 0 || b.ReadWrite || !b.HasTitle || b.Title != "" || b.Options != nil {
		t.Errorf("b = %+v", b)
	}
	if c := d.Parameters()[2]; c.ID != -1 {
		t.Errorf("FFFFFFFF must wrap to int32 -1, got %d", c.ID)
	}
}

func TestParsePartialOnError(t *testing.T) {
	doc := `<CANopen>
  <Object Id="1" ParameterName="ok" DataType="0x5" AccessType="rw"/>
  <Object Id="2" DataType="0x5" AccessType="rw"/>
  <Object Id="3" ParameterName="never" DataType="0x5" AccessType="rw"/>
</CANopen>`
	d, err := Parse([]byte(doc))
	if !errors.Is(err, errMissingAttr) {
		t.Fatalf("err = %v", err)
	}
	if d.Len() != 1 || d.Parameters()[0].Name != "ok" {
		t.Fatalf("partial parameters = %d", d.Len())
	}
	if _, err := Parse([]byte(`<CANopen><Object Id="1" ParameterName="x" DataType="5" AccessType="rw" Length="-1"/></CANopen>`)); err == nil {
		t.Error("bad Length must fail like Convert.ToUInt64")
	}
	if _, err := Parse([]byte("<CANopen>")); err == nil || !strings.Contains(err.Error(), "xml") {
		t.Errorf("truncated XML err = %v", err)
	}
	if _, err := LoadFile("does-not-exist.xml"); err == nil {
		t.Error("LoadFile on a missing file must fail")
	}
}
