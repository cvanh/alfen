package eds

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"alfen/aceclient/internal/api"
)

func prop(id uint16, sub byte, sdt api.SDT, value, cat string) api.Property {
	return api.Property{ID: id, Sub: sub, DataType: sdt, Value: value, Category: cat}
}

func TestBuildField(t *testing.T) {
	d := Default()
	ro := func(p api.Property) api.Property { p.ReadOnly = true; return p }
	withLen := func(p api.Property, n uint64) api.Property { p.MaxLength = n; return p }
	tests := []struct {
		name     string
		p        api.Property
		special  bool
		ok       bool
		kind     UIKind
		label    string
		readOnly bool
		display  string
	}{
		{"string in EDS, special id", prop(0x2050, 0, api.SDTVisibleString, "Twin 4.0", "generic"), false, true, KindText, "Model", true, "Twin 4.0"},
		{"special id with Ctrl+Shift", prop(0x2050, 0, api.SDTVisibleString, "Twin 4.0", "generic"), true, true, KindText, "Model", false, "Twin 4.0"},
		{"special id stays writable even if device says ro", ro(prop(0x2051, 0, api.SDTVisibleString, "x", "generic")), true, true, KindText, "Object Number", false, "x"},
		{"select from EDS options", prop(0x205B, 0, api.SDTInteger8, "1", "generic"), false, true, KindSelect, "Daylight Savings", false, "EU"},
		{"number with units", prop(0x205A, 0, api.SDTInteger8, "-5", "generic"), false, true, KindNumber, "Time Zone (hours)", false, "-5"},
		{"REAL32 number 3 digits", prop(0x2221, 0x10, api.SDTReal32, "49.98", "meter"), false, true, KindNumber, "Frequency (Hz)", false, "49.980"},
		{"single-option checkbox", prop(0x2072, 2, api.SDTUnsigned8, "1", "comm"), false, true, KindCheckbox, "Fixed DHCP server Address", false, ""},
		{"unknown numeric len 2 checkbox", withLen(prop(0x7F00, 1, api.SDTUnsigned16, "1", "x"), 2), false, true, KindCheckbox, "7F00_1", false, ""},
		{"unknown REAL32 number", prop(0x7F00, 2, api.SDTReal32, "1.5", "x"), false, true, KindNumber, "7F00_2", false, "1.500"},
		{"unknown string label = device id", prop(0x7F00, 3, api.SDTVisibleString, "v", "x"), false, true, KindText, "7F00_3", false, "v"},
		{"U64 read-only custom text", ro(prop(0x2059, 0, api.SDTUnsigned64, "1700000000000", "generic")), false, true, KindCustomText, "Charger Date/Time", true, "1700000000000"},
		{"U64 writable -> read-only DateTime", prop(0x2059, 0, api.SDTUnsigned64, "1700000000000", "generic"), false, true, KindReadOnlyText, "Charger Date/Time", true, "Tuesday, 14 November 2023 22:13:20"},
		{"boolean checkbox", prop(0x7F00, 4, api.SDTBoolean, "true", "x"), false, true, KindCheckbox, "7F00_4", false, ""},
		{"byte array in leds skipped", prop(0x2300, 0, api.SDTByteArray, "1,2", "LEDs"), false, false, KindNone, "", false, ""},
		{"byte array elsewhere", prop(0x7F00, 5, api.SDTByteArray, "1,2", "x"), false, true, KindCustomText, "7F00_5", true, "01,02"},
		{"array16", prop(0x7F00, 6, api.SDTArray16, "1,ffff", "x"), false, true, KindCustomText, "7F00_6", true, "0001,FFFF"},
		{"low limit", prop(0x7F00, 7, api.SDTLowLimit, "0", "x"), false, true, KindCustomText, "7F00_7", true, ""},
		{"octet string has no value", prop(0x7F00, 8, api.SDTOctetString, "x", "x"), false, false, KindNone, "", false, ""},
		{"domain has no value", prop(0x2701, 0, api.SDTDomain, "x", "security"), false, false, KindNone, "", false, ""},
	}
	for _, tc := range tests {
		f, ok := d.BuildField(tc.p, tc.special)
		if ok != tc.ok {
			t.Errorf("%s: ok = %v", tc.name, ok)
			continue
		}
		if !ok {
			continue
		}
		if f.Kind != tc.kind || f.Label != tc.label || f.ReadOnly != tc.readOnly || f.DisplayText(0) != tc.display {
			t.Errorf("%s: kind %s label %q ro %v display %q; want %s %q %v %q",
				tc.name, f.Kind, f.Label, f.ReadOnly, f.DisplayText(0), tc.kind, tc.label, tc.readOnly, tc.display)
		}
	}
}

func TestBuildFieldDetails(t *testing.T) {
	d := Default()
	f, _ := d.BuildField(withMaxLen(prop(0x2053, 0, api.SDTVisibleString, "abc", "generic"), 21), false)
	if f.MaxLen != 20 || f.FeatureRightID != "ID_2053" || f.StringType != StringTypeString {
		t.Errorf("text field = %+v", f)
	}
	f, _ = d.BuildField(prop(0x205A, 0, api.SDTInteger8, "1", "generic"), false)
	if f.Spin != (SpinConfig{-128, 127, 1, 0}) || f.Factor != 1 {
		t.Errorf("number spin = %+v", f.Spin)
	}
	f, _ = d.BuildField(prop(0x205B, 0, api.SDTInteger8, "1", "generic"), false)
	if len(f.Options) != 4 || !f.CaseSensitive {
		t.Errorf("select = %+v", f)
	}
	f, _ = d.BuildField(prop(0x7F00, 5, api.SDTByteArray, "1", "x"), false)
	if f.FeatureRightID != "" {
		t.Errorf("custom rows have no feature right: %q", f.FeatureRightID)
	}
	// EDS entry without a <Title>: the bound label is "" like the C#.
	f, ok := d.BuildField(prop(0x2300, 1, api.SDTUnsigned8, "5", "leds"), false)
	if !ok || f.Kind != KindNumber || f.Label != "" || f.Prop.Title != "2300_1" {
		t.Errorf("untitled EDS number = %+v", f)
	}
}

func withMaxLen(p api.Property, n uint64) api.Property { p.MaxLength = n; return p }

func TestFieldCommits(t *testing.T) {
	d := Default()
	txt, _ := d.BuildField(withMaxLen(prop(0x2053, 0, api.SDTVisibleString, "abc", "generic"), 4), false)
	shown, v, err := txt.CommitText("abcdef")
	if err != nil || shown != "abc" || v.Wire() != "abc" {
		t.Errorf("CommitText = %q %q %v", shown, v.Wire(), err)
	}
	num, _ := d.BuildField(prop(0x205A, 0, api.SDTInteger8, "1", "generic"), false)
	spin, v, err := num.CommitNumber("200")
	if err != nil || spin != 127 || v.Wire() != "127" {
		t.Errorf("CommitNumber = %v %q %v", spin, v.Wire(), err)
	}
	sel, _ := d.BuildField(prop(0x205B, 0, api.SDTInteger8, "1", "generic"), false)
	v, w, err := sel.CommitSelect("3")
	if err != nil || !w || v.Wire() != "3" || !Changed(sel.Prop.Typed, v) {
		t.Errorf("CommitSelect = %q %v %v", v.Wire(), w, err)
	}
	if _, w, err := sel.CommitSelect("42"); err != nil || w {
		t.Errorf("value outside the options must not be written: %v %v", w, err)
	}
	chk, _ := d.BuildField(prop(0x2072, 2, api.SDTUnsigned8, "0", "comm"), false)
	if v, err := chk.CommitCheckbox(true); err != nil || v.Wire() != "1" {
		t.Errorf("CommitCheckbox = %q %v", v.Wire(), err)
	}
	if !(Field{Kind: KindCheckbox, Prop: Property{Typed: ParseValue(api.SDTBoolean, "true")}}).Checked() {
		t.Error("Checked")
	}
	locked, _ := d.BuildField(prop(0x2050, 0, api.SDTVisibleString, "Twin 4.0", "generic"), false)
	if _, _, err := locked.CommitText("x"); !errors.Is(err, ErrReadOnly) {
		t.Errorf("read-only commit err = %v", err)
	}
	if _, _, err := locked.CommitNumber("1"); !errors.Is(err, ErrWrongKind) {
		t.Errorf("wrong kind err = %v", err)
	}
}

func TestGroupAllProperties(t *testing.T) {
	d := Default()
	props := []api.Property{
		prop(0x205B, 0, api.SDTInteger8, "1", "generic"),
		prop(0x2050, 0, api.SDTVisibleString, "Twin 4.0", "generic"),
		prop(0x205A, 0, api.SDTInteger8, "1", "generic"),
		prop(0x7F00, 1, api.SDTVisibleString, "short-cat", "x"),
		prop(0x2300, 0, api.SDTByteArray, "1,2", "leds"),
		prop(0x2072, 2, api.SDTUnsigned8, "0", "comm"),
		prop(0x7F01, 0, api.SDTUnsigned8, "1", "notrequested"),
		prop(0x205A, 0, api.SDTInteger8, "2", "generic"), // re-read: last value wins
	}
	cats := []string{"generic", "x", "empty", "leds", "generic", "comm"}
	got := d.GroupAllProperties(cats, props, false)
	if len(got) != 3 {
		t.Fatalf("categories = %d: %+v", len(got), got)
	}
	gen := got[0]
	if gen.Name != "Generic" || gen.Warning != AdvancedWarning || gen.Hide || gen.Header() != "Generic" {
		t.Errorf("generic = %+v", gen)
	}
	var ids []string
	for _, f := range gen.Fields {
		ids = append(ids, IDSubPadded(f.Prop.ID, f.Prop.Sub))
	}
	// sorted by ID_SUB, then the 1-char category "x" appended to the previous one
	if strings.Join(ids, " ") != "2050_00 205A_00 205B_00 7F00_01" {
		t.Errorf("generic order = %v", ids)
	}
	if gen.Fields[1].DisplayText(0) != "2" {
		t.Errorf("re-read value = %q", gen.Fields[1].DisplayText(0))
	}
	if got[1].Name != "Leds" || !got[1].Hide || len(got[1].Fields) != 0 {
		t.Errorf("leds = %+v", got[1])
	}
	if got[2].Name != "Comm" || got[2].Hide || len(got[2].Fields) != 1 {
		t.Errorf("comm = %+v", got[2])
	}
	listed := ListedCategories(got, false)
	if len(listed) != 2 || listed[0] != 0 || listed[1] != 2 {
		t.Errorf("listed = %v", listed)
	}
	if SelectCategory(got, listed, "Comm") != 2 || SelectCategory(got, listed, "Gone") != 0 || SelectCategory(got, nil, "") != -1 {
		t.Error("SelectCategory")
	}
	if ShowAdvancedToggle(got) {
		t.Error("All Properties has no advanced rows")
	}
	// A short category before any real one opens nothing (C# would throw).
	if got := d.GroupAllProperties([]string{"x"}, props, false); len(got) != 0 {
		t.Errorf("leading short category = %+v", got)
	}
}

func TestCategoryVisibility(t *testing.T) {
	adv := Category{Name: "A", Fields: []Field{{Advanced: true}}, ShowCategory: true}
	if adv.Listed(false) || !adv.Listed(true) || !adv.ContainsAdvanced() || !ShowAdvancedToggle([]Category{adv}) {
		t.Error("advanced-only category")
	}
	widgets := Category{Name: SearchCategoryName, Widgets: 3, ShowCategory: true}
	if !widgets.Listed(false) {
		t.Error("search category with widgets must be listed")
	}
	hidden := Category{Name: "H", Fields: []Field{{}}, ShowCategory: false}
	if hidden.Listed(false) {
		t.Error("ShowCategory false")
	}
	if (&Category{Name: "n", Title: "T"}).Header() != "T" {
		t.Error("Header prefers Title")
	}
	vis := []struct {
		f                  Field
		show, showAdv, out bool
	}{
		{Field{}, true, false, true},
		{Field{}, false, true, false},
		{Field{Hide: true}, true, true, false},
		{Field{Advanced: true}, true, false, false},
		{Field{Advanced: true, Hide: true}, true, true, true},
		{Field{Confidential: true}, true, true, false},
	}
	for i, tc := range vis {
		if FieldVisible(tc.f, tc.show, tc.showAdv) != tc.out {
			t.Errorf("%d: FieldVisible", i)
		}
	}
	if CapitalizeCategory("generic") != "Generic" || CapitalizeCategory("éa") != "Éa" || CapitalizeCategory("") != "" {
		t.Error("CapitalizeCategory")
	}
}

func TestSearch(t *testing.T) {
	d := Default()
	props := []api.Property{
		prop(0x7F00, 1, api.SDTVisibleString, "Model-like value", "x"),
		prop(0x205B, 0, api.SDTInteger8, "2", "generic"),
		prop(0x2050, 0, api.SDTVisibleString, "Twin 4.0", "generic"),
		prop(0x2701, 0, api.SDTDomain, "model", "security"), // null value: never matched
		prop(0x2300, 0, api.SDTByteArray, "1,2", "leds"),
	}
	if _, err := d.Search(props, SearchOptions{Term: "  ", Name: true}, false); err == nil || err.Error() != MsgEnterSearchTerm {
		t.Fatalf("blank term err = %v", err)
	}
	res, err := d.Search(props, SearchOptions{Term: "MODEL", Name: true}, false)
	if err != nil || len(res.Matches) != 1 || res.Matches[0].ID != 0x2050 {
		t.Fatalf("name search = %+v %v", res.Matches, err)
	}
	// ParameterName is searched too: "sysDaylight" only appears there.
	if res, _ := d.Search(props, SearchOptions{Term: "sysdaylight", Name: true}, false); len(res.Matches) != 1 {
		t.Errorf("ParameterName search = %+v", res.Matches)
	}
	// Value search uses Value.ToString(): EDS order first, then unknown ids.
	res, _ = d.Search(props, SearchOptions{Term: "model", Value: true}, false)
	if len(res.Matches) != 1 || res.Matches[0].ID != 0x7F00 {
		t.Errorf("value search = %+v", res.Matches)
	}
	res, _ = d.Search(props, SearchOptions{Term: "byte[]", Value: true}, false)
	if len(res.Matches) != 1 || len(res.Fields) != 0 || res.NoResults != "" {
		t.Errorf("byte array matches as System.Byte[] but builds no row: %+v", res)
	}
	res, _ = d.Search(props, SearchOptions{Term: "0x205b_0", ID: true}, false)
	if len(res.Matches) != 1 || res.Matches[0].ID != 0x205B || res.Fields[0].Kind != KindSelect {
		t.Errorf("id search = %+v", res.Matches)
	}
	res, _ = d.Search(props, SearchOptions{Term: "^twin [0-9]", Value: true, Regex: true}, false)
	if len(res.Matches) != 1 || res.Warning != "" {
		t.Errorf("regex search = %+v", res)
	}
	res, _ = d.Search(props, SearchOptions{Term: "(", Value: true, Regex: true}, false)
	if !strings.HasPrefix(res.Warning, "Invalid regular expression: ") || !strings.HasSuffix(res.Warning, "\nNormal text search will be used instead.") {
		t.Errorf("invalid regex warning = %q", res.Warning)
	}
	res, _ = d.Search(props, SearchOptions{Term: "nothing-here", Name: true, Value: true, ID: true}, false)
	if res.NoResults != "No results found for: nothing-here" {
		t.Errorf("NoResults = %q", res.NoResults)
	}
	if o := DefaultSearchOptions(); !o.Name || o.Value || !o.ID || o.Regex {
		t.Errorf("DefaultSearchOptions = %+v", o)
	}
}

func TestParseCategories(t *testing.T) {
	tests := []struct {
		body string
		want []string
	}{
		{`{"version":1,"categories":["generic","comm",null," leds ","\"q\""]}`, []string{"generic", "comm", "leds", "q"}},
		{`{"categories":[]}`, nil},
		{``, nil},
		{`not json`, nil},
		{`["generic"]`, nil},
		{`{"categories":[true,5]}`, []string{"True", "5"}},
	}
	for _, tc := range tests {
		got := ParseCategories(tc.body)
		if fmt.Sprint(got) != fmt.Sprint(tc.want) {
			t.Errorf("ParseCategories(%q) = %q, want %q", tc.body, got, tc.want)
		}
	}
}
