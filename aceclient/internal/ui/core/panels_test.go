package core

import (
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"alfen/aceclient/internal/api"
	"alfen/aceclient/internal/fwi"
	"alfen/aceclient/internal/scn"
	"alfen/aceclient/internal/tvf"
)

func samplePropertyCache() *PropertyCache {
	pc := NewPropertyCache()
	pc.Merge([]api.Property{
		{ID: 0x2050, DataType: api.SDTVisibleString, ReadOnly: false, Category: "generic", Name: "sysChargePointModel", Value: "NG910-60023", MaxLength: 31},
		{ID: 0x2053, DataType: api.SDTVisibleString, Category: "generic", Name: "sysChargeBoxIdentity", Value: "MYCHARGER", MaxLength: 21},
		{ID: 0x2129, DataType: api.SDTUnsigned8, Category: "generic", Name: "flag", Value: "1", MaxLength: 2},
		{ID: 0x2062, DataType: api.SDTReal32, Category: "generic", Name: "sysMaxStationCurrent", Value: "16"},
		{ID: 0x2187, DataType: api.SDTUnsigned64, Category: "generic", Value: "1760000000000"},
		{ID: 0x2188, DataType: api.SDTUnsigned64, ReadOnly: true, Category: "generic", Value: "42"},
		{ID: 0x3001, DataType: api.SDTByteArray, Category: "leds", Value: "01,02"},
		{ID: 0x3002, DataType: api.SDTByteArray, Category: "x", Value: "0a,0b"},
		{ID: 0x3003, DataType: api.SDTUnicodeString, Category: "generic", Value: "ignored"},
		{ID: 0x3004, DataType: api.SDTBoolean, Category: "comm", Value: "true"},
	})
	return pc
}

func TestAllPropertiesGroups(t *testing.T) {
	pc := samplePropertyCache()
	groups := AllPropertiesGroups(pc, []string{"generic", "generic", "leds", "x", "comm", "empty"}, false)
	var names []string
	for _, g := range groups {
		names = append(names, g.Name)
	}
	// "leds" holds only a skipped byte array → hidden; "x" (len 1) appends
	// to the previous group ("Leds" was created but empty, so its rows land
	// there and it becomes visible).
	if strings.Join(names, ",") != "Generic,Leds,Comm" {
		t.Fatalf("groups = %v", names)
	}
	byID := map[uint16]PropertyRow{}
	for _, g := range groups {
		for _, r := range g.Rows {
			byID[r.ID] = r
		}
	}
	if r := byID[0x2050]; !r.ReadOnly || r.Kind != EditorText { // 8272 forced read-only
		t.Errorf("model row = %+v", r)
	}
	if r := byID[0x2053]; r.ReadOnly || r.Kind != EditorText || r.Label != "2053_0" {
		t.Errorf("identity row = %+v", r)
	}
	if r := byID[0x2129]; r.Kind != EditorCheck || r.Label != "2129_0" { // MaxLength == 2, no EDS
		t.Errorf("flag row = %+v", r)
	}
	if r := byID[0x2062]; r.Kind != EditorNumber || r.Digits != 3 {
		t.Errorf("real row = %+v", r)
	}
	if r := byID[0x2187]; r.Kind != EditorReadOnly || r.Format != FormatDateTime {
		t.Errorf("u64 rw row = %+v", r)
	}
	if r := byID[0x2188]; r.Kind != EditorReadOnly || r.Format != FormatRaw {
		t.Errorf("u64 ro row = %+v", r)
	}
	if _, ok := byID[0x3001]; ok {
		t.Error("leds byte array must be skipped")
	}
	if r := byID[0x3002]; r.Display != "0A,0B" || !r.ReadOnly {
		t.Errorf("byte array row = %+v", r)
	}
	if _, ok := byID[0x3003]; ok {
		t.Error("UNICODE_STRING never gets a value in the installer")
	}
	if r := byID[0x3004]; r.Kind != EditorCheck || r.Display != "[x]" {
		t.Errorf("bool row = %+v", r)
	}
	// Shift+Ctrl unlocks 8272.
	g2 := AllPropertiesGroups(pc, []string{"generic"}, true)
	for _, r := range g2[0].Rows {
		if r.ID == 0x2050 && r.ReadOnly {
			t.Error("specialWrite must unlock 8272")
		}
	}
}

func TestAllPropertiesWithEDS(t *testing.T) {
	pc := samplePropertyCache()
	pc.SetEDS(func(id uint16, sub byte) (EDSParam, bool) {
		switch id {
		case 0x2053:
			return EDSParam{Name: "sysChargeBoxIdentity", Title: "Customer Ident. number"}, true
		case 0x2129:
			return EDSParam{Name: "mode", Title: "Mode", Options: []EDSOption{{"0", "Off"}, {"1", "On"}, {"2", "Auto"}}}, true
		case 0x2062:
			return EDSParam{Name: "sysMaxStationCurrent", Title: "Automat", Units: "A"}, true
		}
		return EDSParam{}, false
	})
	rows := PropertyRows(pc, pc.All(), false)
	byID := map[uint16]PropertyRow{}
	for _, r := range rows {
		byID[r.ID] = r
	}
	if r := byID[0x2053]; r.Label != "Customer Ident. number" {
		t.Errorf("label = %q", r.Label)
	}
	if r := byID[0x2129]; r.Kind != EditorSelect || r.Display != "On" {
		t.Errorf("select row = %+v", r)
	}
	if r := byID[0x2062]; r.Label != "Automat (A)" {
		t.Errorf("units label = %q", r.Label)
	}
	p, _ := pc.Get(0x2053, 0)
	if pc.ICUName(p) != "OD_sysChargeBoxIdentity" || pc.WireProperty(p).Name != "OD_sysChargeBoxIdentity" {
		t.Errorf("ICUName = %q", pc.ICUName(p))
	}
}

func TestSearch(t *testing.T) {
	pc := samplePropertyCache()
	if _, err := Search(pc, "  ", DefaultSearchOptions); err == nil || err.Error() != SearchEmptyTermError {
		t.Fatalf("empty term err = %v", err)
	}
	ids := func(r SearchResult) string {
		var s []string
		for _, p := range r.Matches {
			s = append(s, ODIndex(p.ID, p.Sub))
		}
		return strings.Join(s, ",")
	}
	r, _ := Search(pc, "0x2053", DefaultSearchOptions)
	if ids(r) != "2053_0" {
		t.Errorf("id search = %s", ids(r))
	}
	r, _ = Search(pc, "identity", DefaultSearchOptions) // device-name fallback without EDS
	if ids(r) != "2053_0" {
		t.Errorf("name search = %s", ids(r))
	}
	r, _ = Search(pc, "mycharger", SearchOptions{Value: true})
	if ids(r) != "2053_0" {
		t.Errorf("value search = %s", ids(r))
	}
	r, _ = Search(pc, "^0x30", SearchOptions{ID: true, Regex: true})
	if ids(r) != "3001_0,3002_0,3004_0" { // 3003 has no value
		t.Errorf("regex search = %s", ids(r))
	}
	r, _ = Search(pc, "(", SearchOptions{ID: true, Regex: true})
	if !strings.HasPrefix(r.Warning, "Invalid regular expression:") || !strings.HasSuffix(r.Warning, "Normal text search will be used instead.") {
		t.Errorf("warning = %q", r.Warning)
	}
	if SearchNoResults("x") != "No results found for: x" {
		t.Error("no results text")
	}
}

func TestPropertyCacheSemantics(t *testing.T) {
	pc := NewPropertyCache()
	pc.Merge([]api.Property{{ID: 0x2062, DataType: api.SDTReal32, Value: "16.000"}, {ID: 0x2129, DataType: api.SDTUnsigned8, Value: "null"}})
	if p, _ := pc.Get(0x2129, 0); p.Value != "0" || !p.HasValue {
		t.Fatalf("null → 0: %+v", p)
	}
	if _, err := pc.SetValue(0x2062, 0, "16"); err != nil || pc.HasChanges() {
		t.Fatal("same value is not a change")
	}
	pc.SetValue(0x2062, 0, "20")
	pc.Merge([]api.Property{{ID: 0x2062, DataType: api.SDTReal32, Value: "17"}})
	if p, _ := pc.Get(0x2062, 0); p.Value != "20" || p.DeviceValue != "16" {
		t.Fatalf("pending change must survive a re-read: %+v", p)
	}
	pc.Revert()
	if p, _ := pc.Get(0x2062, 0); p.Value != "16" || p.Changed {
		t.Fatalf("revert: %+v", p)
	}
	if pc.Int(0x2062, 0) != 16 || pc.Int(0x9999, 0) != 0 || !pc.Has(0x2062, 0) || pc.Has(0x9999, 0) {
		t.Fatal("accessors")
	}
	if _, err := pc.SetValue(0x9999, 0, "1"); err == nil {
		t.Fatal("unknown property")
	}
}

func TestInformationRows(t *testing.T) {
	pc := NewPropertyCache()
	pc.Merge([]api.Property{
		{ID: 8272, DataType: api.SDTVisibleString, Value: "NG910-60023"},
		{ID: 8273, DataType: api.SDTVisibleString, Value: "ACE0123456"},
		{ID: 8275, DataType: api.SDTVisibleString, Value: "MYCHARGER"},
		{ID: 4106, DataType: api.SDTVisibleString, Value: "7.4.6-4416"},
		{ID: 8269, Sub: 1, DataType: api.SDTUnsigned8, Value: "3"},
		{ID: 8269, Sub: 2, DataType: api.SDTUnsigned8, Value: "3"},
		{ID: 8269, Sub: 3, DataType: api.SDTUnsigned8, Value: "1"},
		{ID: 8269, Sub: 4, DataType: api.SDTUnsigned8, Value: "1"},
		{ID: 12672, DataType: api.SDTVisibleString, Value: "HW:1.2,SW:3.4"},
		{ID: 8472, DataType: api.SDTVisibleString, Value: "Quectel"},
		{ID: 8281, DataType: api.SDTUnsigned64, Value: "1760000000000"},
		{ID: 8302, DataType: api.SDTInteger16, Value: "60"},
	})
	rows := InformationRows(pc, InformationContext{FirmwareVersion: MustVersion("7.4.6"), NumberOfSockets: 1, IsUniquePasswordRequired: true})
	get := func(label string) (InfoRow, bool) {
		for _, r := range rows {
			if r.Label == label && !r.Header {
				return r, true
			}
		}
		return InfoRow{}, false
	}
	checks := map[string]string{
		"Model":                             "NG910-60023",
		"Object Number":                     "ACE0123456",
		"Customer Ident. number":            "MYCHARGER",
		"Software version controller board": "7.4.6-4416",
		"Hardware version controller board": "C-02",
		"Hardware version power board":      "A-00",
		"NFC-RFID reader 1 hardw. version":  "1.2",
		"NFC-RFID reader 1 softw. version":  "3.4",
		"Modem manufacturer":                "Quectel",
		"Time zone":                         "(UTC+01:00)",
		"Charger date and time":             "Thursday, October 9, 2025 9:53:20 AM",
	}
	for l, want := range checks {
		r, ok := get(l)
		if !ok || r.Value != want {
			t.Errorf("%s = %q (found %v), want %q", l, r.Value, ok, want)
		}
	}
	if _, ok := get("Enable Secure Service Access"); !ok {
		t.Error("Station Password category expected for unique-password devices")
	}
	if _, ok := get("Hardware version SCB"); ok {
		t.Error("SCB row is AHP-only")
	}
	ahp := InformationRows(pc, InformationContext{IsAHP: true, NumberOfSockets: 2, FirmwareVersion: MustVersion("2.4.0")})
	n := 0
	for _, r := range ahp {
		if r.Header && strings.HasPrefix(r.Label, "SocketBoard ") {
			n++
		}
	}
	if n != 2 {
		t.Errorf("AHP socket boards = %d", n)
	}
}

func TestInformationHelpers(t *testing.T) {
	for _, c := range []struct {
		rev, asm int
		want     string
	}{{3, 3, "C-02"}, {0, 0, "DEFAULT-DEFAULT"}, {-1, -1, "UNKNOWN-UNKNOWN"}, {16, 16, "R-15"}, {17, 1, "N/A"}} {
		if got := ParseRevision(c.rev, c.asm); got != c.want {
			t.Errorf("ParseRevision(%d,%d) = %q, want %q", c.rev, c.asm, got, c.want)
		}
	}
	pc := NewPropertyCache()
	pc.Merge([]api.Property{{ID: 8276, DataType: api.SDTVisibleString, Value: "abc#N:H1;S2"}})
	if NFCVersion(pc, MustVersion("4.2.0"), 1, false) != "H1" || NFCVersion(pc, MustVersion("4.2.0"), 1, true) != "S2" {
		t.Error("pre-4.3 NFC parse")
	}
	if NFCVersion(pc, MustVersion("7.0.0"), 2, true) != "N/A" {
		t.Error("missing reader 2")
	}
	if got := FormatDeviceDateTime("90061000", 0); got != "1.01:01:01" {
		t.Errorf("timespan = %q", got)
	}
	if got := FormatDeviceDateTime("3000", 0); got != "00:00:03" {
		t.Errorf("timespan = %q", got)
	}
	if FormatUTCOffset(-90) != "(UTC-01:30)" || FormatUTCOffset(0) != "(UTC)" {
		t.Error("FormatUTCOffset")
	}
}

func TestSaveChecks(t *testing.T) {
	if ValidateIdentity("CP-01*:+|@.", false) != "" {
		t.Error("valid identity rejected")
	}
	if got := ValidateIdentity("bad id", false); got != "The field \"Customer Ident. Number\" is invalid.\nOnly characters, digits and *-_=:+|@. are allowed." {
		t.Errorf("msg = %q", got)
	}
	long := strings.Repeat("A", 21)
	if ValidateIdentity(long, false) == "" || ValidateIdentity(long, true) != "" {
		t.Error("length limits 20/48")
	}
	if !strings.HasSuffix(ValidateIdentity(strings.Repeat("A", 49), true), "\nand the maximum length is 48 positions.") {
		t.Error("AHP message")
	}
	pc := NewPropertyCache()
	pc.Merge([]api.Property{{ID: 8275, DataType: api.SDTVisibleString, Value: "CP1"}, {ID: 8626, DataType: api.SDTUnsigned8, Value: "1"}})
	pc.SetValue(8275, 0, "bad id")
	if c := InformationSaveCheck(pc, false, "CP1"); c.Error == "" {
		t.Error("invalid identity must block")
	}
	pc.SetValue(8626, 0, "0")
	c := InformationSaveCheck(pc, false, "CP1")
	if c.Error != "" || !strings.HasPrefix(c.Question, "Are you sure you want to disable Secure Service Access?") {
		t.Errorf("SSA check = %+v (the C# returns before the identity check)", c)
	}
}

func TestValidateCSConfiguration(t *testing.T) {
	pc := NewPropertyCache()
	pc.Merge([]api.Property{
		{ID: 8489, DataType: api.SDTReal32, Value: "32"},
		{ID: 12585, DataType: api.SDTReal32, Value: "32"},
		{ID: 8290, DataType: api.SDTReal32, Value: "40"},
		{ID: 8292, DataType: api.SDTUnsigned8, Value: "0"},
	})
	if ValidateCSConfiguration(pc, 1, 1, "CP") != "" {
		t.Fatal("single socket: no change")
	}
	msg := ValidateCSConfiguration(pc, 2, 1, "CP")
	if !strings.HasPrefix(msg, "The total sum of the connectors maximum current (64A) is more than the station current (40A)!") {
		t.Fatalf("msg = %q", msg)
	}
	if p, _ := pc.Get(8489, 0); p.Value != "20" || !p.Changed {
		t.Fatalf("8489 = %+v", p)
	}
}

func TestUpsertSCN(t *testing.T) {
	l := NewDeviceList()
	ip := net.IPv4(192, 168, 1, 20)
	now := time.Now()
	l.UpsertSCN(&scn.Socket{Name: "CP-A", NetworkName: "NET1", SocketIndex: 0, TotalNumberOfSockets: 2, State: scn.StateCharging, ScnLibVersion: 6}, ip, now)
	l.UpsertSCN(&scn.Socket{Name: "CP-A", NetworkName: "NET1", SocketIndex: 1, TotalNumberOfSockets: 2, State: scn.StateIdle, ScnLibVersion: 6}, ip, now)
	d, ok := l.Get("192.168.1.20")
	if !ok || d.Identity != "CP-A" || d.SCNNetwork != "NET1" || d.Port != 443 || !d.Discovered || d.TotalSockets != 2 {
		t.Fatalf("device = %+v", d)
	}
	if d.SocketSummary() != "0: Charging, 1: Idle" {
		t.Fatalf("summary = %q", d.SocketSummary())
	}
}

func TestFirmwareTexts(t *testing.T) {
	if v, m, ok := VersionFromFileName("/x/NG9xx 7.4.6-4416.fwi"); !m || !ok || v.String() != "7.4.6" {
		t.Errorf("version = %v %v %v", v, m, ok)
	}
	if v, _, _ := VersionFromFileName("AHP_release_FW_2.7.0_FULL.tfw"); v.String() != "2.7.0" {
		t.Errorf("ahp version = %v", v)
	}
	if _, m, _ := VersionFromFileName("firmware.fwi"); m {
		t.Error("no version in name")
	}
	if !NeedsNewPassword(false, MustVersion("4.12.0"), "NG9xx 5.6.1-4414-A.fwi") || NeedsNewPassword(true, MustVersion("4.12.0"), "NG9xx 5.6.1-4414-A.fwi") || NeedsNewPassword(false, MustVersion("6.6.2"), "NG9xx 7.4.6-4416.fwi") {
		t.Error("NeedsNewPassword")
	}
	if !ShowNg9xxUpgradeWarning(false, MustVersion("6.5.9")) || ShowNg9xxUpgradeWarning(false, MustVersion("6.6")) || ShowNg9xxUpgradeWarning(true, MustVersion("1.0")) {
		t.Error("ShowNg9xxUpgradeWarning")
	}
	if ProgressionText(10, false, 0) != "Uploading firmware to the charger, can take 1 to 3 minutes." ||
		ProgressionText(98, false, 0) != "Finishing up firmware update..." ||
		ProgressionText(50, true, 3*time.Minute) != "Installing firmware (can take up to 12 minutes) 3 minutes passed." {
		t.Error("ProgressionText")
	}
	if UploadHeader("CP1", "ACE1") != "Upload firmware to device 'CP1' (serial number: ACE1)" ||
		UploadConfirmQuestion("CP1", "/a/b/fw.fwi") != "Are you sure you want to update the firmware of 'CP1' to 'fw.fwi'" {
		t.Error("texts")
	}
	if strings.Join(FirmwareFileFilters[0].Extensions(), ",") != ".fwi,.fwu,.tfw,.tcf,.tvf" || ActiveFirmwareFilter(true, false).Name != "AHWP firmware files" {
		t.Error("filters")
	}
}

func TestClassifyFirmwareSynthetic(t *testing.T) {
	disp, err := fwi.WrapDisplayPayload([]byte("display objects"))
	if err != nil {
		t.Fatal(err)
	}
	c := ClassifyFirmware("logo_1.2.3_x.fwi", disp, true, false)
	if c.FWI == nil || !c.FWICRCOK || !c.DisplayResource || !c.DisplayUnwrapOK || c.DisplayUnwrapN != len("display objects") || len(c.Warnings) != 0 {
		t.Fatalf("display classification = %+v", c)
	}
	txt := c.Text()
	for _, want := range []string{"File: logo_1.2.3_x.fwi", "Type: NG9xx firmware file (.fwi)", "Version in file name: 1.2.3", "Header CRC: stored", "(OK)", "Variant: display resource (installer-built, known key); unwrap OK, 15 bytes"} {
		if !strings.Contains(txt, want) {
			t.Errorf("text lacks %q:\n%s", want, txt)
		}
	}
	local := append(tvf.BuildHeader(1, "resources", 100), make([]byte, 100)...)
	c = ClassifyFirmware("res.tvf", local, true, false)
	if !c.LocalTVF || c.TVFHeaderLength == 0 {
		t.Fatalf("tvf classification = %+v", c)
	}
	if !strings.Contains(c.Text(), "(installer-built resource TVF)") || !strings.Contains(c.Text(), "Warning: the connected device takes .fwi/.fwu files") {
		t.Errorf("tvf text:\n%s", c.Text())
	}
	c = ClassifyFirmware("notes.txt", []byte("hello"), false, false)
	if !strings.Contains(c.Text(), "Container: not recognised") || !strings.Contains(c.Text(), "Warning: extension is not one of") {
		t.Errorf("unknown text:\n%s", c.Text())
	}
	if c := ClassifyFirmware("empty.fwi", nil, false, false); len(c.Warnings) != 1 || c.Warnings[0] != "file is empty" {
		t.Errorf("empty = %+v", c.Warnings)
	}
}

func TestClassifyFirmwareFixtures(t *testing.T) {
	cases := []struct {
		path string
		want []string
	}{
		{"../../../../firmware/Firmware/NG9xx 7.4.6-4416.fwi", []string{"Type: NG9xx firmware file (.fwi)", "Version in file name: 7.4.6", "header type 0x21", "(OK)", "Variant: controller firmware (device-held key), wrapped key block: yes, section map: yes"}},
		{"../../../../firmware/Firmware/NG9xx 5.6.1-4414-A.fwi", []string{"header type 0x13", "(OK)", "Variant: controller firmware"}},
		{"../../../../firmware/Firmware/AHP_release_FW_2.7.0_FULL.tfw", []string{"Type: AHWP firmware file (.tfw)", "Version in file name: 2.7.0", "Container: TVF, header 1434 bytes", "(Alfen release)"}},
		{"../../../../firmware/BackofficePresets/SwiGo.tcf", []string{"Type: AHWP firmware file (.tcf)", "Container: TVF, header 1355 bytes", "(Alfen release)"}},
		{"../../../../firmware/BackofficePresets/Chargemap.fwi", []string{"Container: .fwi, header type 0x13, payload 1904 bytes", "(OK)"}},
	}
	for _, c := range cases {
		data, err := os.ReadFile(c.path)
		if err != nil {
			t.Logf("skip %s: %v", c.path, err)
			continue
		}
		txt := ClassifyFirmware(c.path, data, false, false).Text()
		for _, w := range c.want {
			if !strings.Contains(txt, w) {
				t.Errorf("%s: text lacks %q:\n%s", c.path, w, txt)
			}
		}
	}
}
