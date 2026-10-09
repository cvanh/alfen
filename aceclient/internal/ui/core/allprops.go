package core

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"alfen/aceclient/internal/api"
)

// AdvancedSettingsWarning is the warning PanelAllProperties puts at the top
// of every category.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/PanelAllProperties.cs:46-86
const AdvancedSettingsWarning = "These settings are advanced settings, only change them when you know what you are doing. Incorrect values might damage your charging station!"

// EditorKind is the control PanelAllProperties.AddProperties picks for a
// property (AddText, AddCustomNumber/AddNumber, AddCheckBox, AddSelect,
// AddCustomText/AddReadOnlyText).
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/PanelAllProperties.cs:238-355
type EditorKind int

const (
	EditorReadOnly EditorKind = iota // AddCustomText / AddReadOnlyText
	EditorText                       // AddText (UIPropertyString)
	EditorNumber                     // AddNumber / AddCustomNumber (UIPropertyNumber)
	EditorCheck                      // AddCheckBox (UIPropertyCheckbox)
	EditorSelect                     // AddSelect (EDS options)
)

func (k EditorKind) String() string {
	switch k {
	case EditorText:
		return "text"
	case EditorNumber:
		return "number"
	case EditorCheck:
		return "checkbox"
	case EditorSelect:
		return "select"
	}
	return "read-only"
}

// ValueFormat selects how a read-only value is rendered.
type ValueFormat int

const (
	FormatRaw      ValueFormat = iota
	FormatDateTime             // UIPropertyStringType.DateTime
)

// PropertyRow is one control of the All Properties panel.
type PropertyRow struct {
	Property
	Kind     EditorKind
	Label    string // the control's label text
	Display  string // the rendered current value
	Format   ValueFormat
	Digits   int  // UIPropertyNumber digits (3 for REAL32/REAL64)
	ReadOnly bool // effective read-only state (device access, forced, kind)
	Options  []EDSOption
}

// PropertyGroup is one UIConfigCategory of the panel.
type PropertyGroup struct {
	Category string // the device category ("generic", …)
	Name     string // char.ToUpper(category[0]) + category.Substring(1)
	Rows     []PropertyRow
}

// forcedReadOnly are the ids PanelAllProperties.AddProperties forces
// read-only unless Shift+Ctrl was held when the panel was populated
// (specialWritePermission): 8271, 8272 (model), 8273 (object number).
var forcedReadOnly = map[uint16]bool{8271: true, 8272: true, 8273: true}

// AllPropertiesGroups ports PanelAllProperties.OnChangeDevice: for each
// category returned by RequestCategories (distinct, in order) the properties
// of that category sorted by ID_SUB; a category name longer than one
// character opens a new group (whose first entry is the advanced-settings
// warning), a shorter one appends to the previous group; a group that ends up
// with nothing but the warning is hidden (omitted here).
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/PanelAllProperties.cs:46-86
func AllPropertiesGroups(pc *PropertyCache, categories []string, specialWrite bool) []PropertyGroup {
	seen := map[string]bool{}
	var cats []string
	for _, c := range categories {
		if !seen[c] {
			seen[c] = true
			cats = append(cats, c)
		}
	}
	all := pc.All()
	var groups []PropertyGroup
	var cur *PropertyGroup
	for _, cat := range cats {
		var list []Property
		for _, p := range all {
			if p.Category == cat {
				list = append(list, p)
			}
		}
		if len(list) == 0 {
			continue
		}
		sort.SliceStable(list, func(i, j int) bool { return IDSub(list[i].ID, list[i].Sub) < IDSub(list[j].ID, list[j].Sub) })
		if len(cat) > 1 {
			r, n := utf8.DecodeRuneInString(cat)
			groups = append(groups, PropertyGroup{Category: cat, Name: string(unicode.ToUpper(r)) + cat[n:]})
			cur = &groups[len(groups)-1]
		}
		if cur == nil {
			continue // AddProperties(null, …) adds nothing
		}
		cur.Rows = append(cur.Rows, PropertyRows(pc, list, specialWrite)...)
	}
	out := groups[:0]
	for _, g := range groups {
		if len(g.Rows) > 0 { // Hide = m_lstProperties.Count == 1 (warning only)
			out = append(out, g)
		}
	}
	return out
}

// PropertyRows ports PanelAllProperties.AddProperties for a list of
// properties (also used for search results).
func PropertyRows(pc *PropertyCache, list []Property, specialWrite bool) []PropertyRow {
	var rows []PropertyRow
	for _, p := range list {
		if r, ok := PropertyRowFor(pc, p, specialWrite); ok {
			rows = append(rows, r)
		}
	}
	return rows
}

// PropertyRowFor ports one iteration of PanelAllProperties.AddProperties.
// ok is false when the C# adds no control (Value == null or a byte array
// in the "leds" category).
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/PanelAllProperties.cs:238-355
func PropertyRowFor(pc *PropertyCache, p Property, specialWrite bool) (PropertyRow, bool) {
	if forcedReadOnly[p.ID] {
		p.ReadOnly = !specialWrite
	}
	if !p.HasValue {
		return PropertyRow{}, false
	}
	eds, hasEDS := pc.Lookup(p.ID, p.Sub)
	row := PropertyRow{Property: p, Display: p.Value, ReadOnly: p.ReadOnly}
	icuName := pc.ICUName(p)
	switch p.DataType {
	case api.SDTVisibleString, api.SDTOctetString, api.SDTUnicodeString:
		row.Kind = EditorText
		row.Label = pc.LabelText(p)
		row.Display = TrimDisplayString(p.Value)
	case api.SDTInteger8, api.SDTInteger16, api.SDTInteger32, api.SDTUnsigned8,
		api.SDTUnsigned16, api.SDTUnsigned32, api.SDTReal32, api.SDTReal64, api.SDTInteger64:
		if p.DataType == api.SDTReal32 || p.DataType == api.SDTReal64 {
			row.Digits = 3
		}
		switch {
		case !hasEDS:
			if p.MaxLength == 2 {
				row.Kind = EditorCheck
			} else {
				row.Kind = EditorNumber
			}
			row.Label = icuName // AddCheckBox(…, item.ICUName) / AddCustomNumber(…, item.ICUName)
		case len(eds.Options) == 1:
			row.Kind = EditorCheck
			row.Label = pc.LabelText(p)
		case len(eds.Options) > 1:
			row.Kind = EditorSelect
			row.Label = pc.LabelText(p)
			row.Options = eds.Options
		default:
			row.Kind = EditorNumber
			row.Label = pc.LabelText(p)
		}
	case api.SDTUnsigned64:
		row.Kind = EditorReadOnly
		if p.ReadOnly {
			row.Label = pc.Title(p) // AddCustomText(item.Title, item.Value.ToString())
		} else {
			row.Label = pc.LabelText(p) // AddReadOnlyText(…, UIPropertyStringType.DateTime)
			row.Format = FormatDateTime
		}
	case api.SDTBoolean:
		row.Kind = EditorCheck
		row.Label = pc.LabelText(p)
	case api.SDTByteArray:
		if strings.ToLower(p.Category) == "leds" {
			return PropertyRow{}, false
		}
		row.Kind = EditorReadOnly
		row.Label = icuName
	case api.SDTArray16:
		row.Kind = EditorReadOnly
		row.Label = icuName
	default:
		row.Kind = EditorReadOnly
		row.Label = icuName
	}
	if row.Kind == EditorReadOnly {
		row.ReadOnly = true
	}
	if row.Kind == EditorCheck {
		if CheckState(p.Value) {
			row.Display = "[x]"
		} else {
			row.Display = "[ ]"
		}
	}
	if row.Kind == EditorSelect {
		for _, o := range row.Options {
			if o.Value == p.Value {
				row.Display = o.Title
			}
		}
	}
	return row, true
}

// TrimDisplayString ports UIPropertyString.OnRefreshDisplay's
// Value.ToString().Trim('\u0003', '\u001e').
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/UIPropertyString.cs:180-235
func TrimDisplayString(s string) string { return strings.Trim(s, "\u0003\u001e") }

// CheckState ports UIPropertyCheckbox.OnRefreshDisplay (no value mask):
// a parsable boolean gives its value, otherwise Convert.ToInt32(Value) != 0.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/UIPropertyCheckbox.cs:97-140
func CheckState(value string) bool {
	if b, ok := parseDotNetBool(value); ok {
		return b
	}
	if n, ok := parseDotNetInt(value, 64, true); ok {
		return n != 0
	}
	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return math.RoundToEven(f) != 0
	}
	return false
}

// CheckboxValue ports UIPropertyCheckbox.GetValue (no value mask) followed by
// ICUProperty.SetValue: INTEGER8/UNSIGNED8 → 1/0, BOOLEAN → True/False, any
// other type receives the boolean as text, which its SetValue branch fails to
// parse — so e.g. an UNSIGNED16 shown as a checkbox always stores 0. This
// mirrors the installer exactly.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/UIPropertyCheckbox.cs:160-196
func CheckboxValue(t api.SDT, active bool) string {
	switch t {
	case api.SDTInteger8, api.SDTUnsigned8:
		if active {
			return "1"
		}
		return "0"
	}
	v, _ := ConvertValue(t, dotNetBool(active))
	return v
}

// NumberRange ports UIPropertyNumber.OnRefreshDisplay's SetValueMinMax per
// SDT, followed by `if (MaxLength != 0) m_spnValue.MaximumValue = MaxLength`.
// REAL64 has no explicit range in the C#; it is left unbounded here.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/UIPropertyNumber.cs:158-249
func NumberRange(t api.SDT, maxLength uint64) (min, max float64) {
	switch t {
	case api.SDTUnsigned8:
		min, max = 0, 255
	case api.SDTUnsigned16:
		min, max = 0, 65535
	case api.SDTUnsigned32:
		min, max = 0, 4294967295
	case api.SDTUnsigned64:
		min, max = 0, 1.8446744073709552e19
	case api.SDTInteger8:
		min, max = -128, 127
	case api.SDTInteger16:
		min, max = -32768, 32767
	case api.SDTInteger32:
		min, max = -2147483648, 2147483647
	case api.SDTInteger64:
		min, max = -9.223372036854776e18, 9.223372036854776e18
	case api.SDTReal32:
		min, max = -3.4028234663852886e38, 3.4028234663852886e38
	default:
		min, max = math.Inf(-1), math.Inf(1)
	}
	if maxLength != 0 {
		max = float64(maxLength)
	}
	return min, max
}

// NumberInput ports editing a UIPropertyNumber: the spin button parses the
// text, clamps it to NumberRange and rounds to Digits; OnSpinValueChanged then
// assigns the double to ICUProperty.Value (SetValue(double.ToString())).
// clamped reports that the entered value was outside the range.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/UIPropertyNumber.cs:134-140,250-266
func NumberInput(t api.SDT, maxLength uint64, digits int, text string) (value string, clamped bool, err error) {
	f, ok := parseDotNetFloat(strings.ReplaceAll(text, ",", "."), 64)
	if !ok || math.IsNaN(f) || math.IsInf(f, 0) {
		return "", false, fmt.Errorf("%q is not a number", text)
	}
	min, max := NumberRange(t, maxLength)
	if f < min {
		f, clamped = min, true
	}
	if f > max {
		f, clamped = max, true
	}
	p := math.Pow(10, float64(digits))
	f = math.Round(f*p) / p
	v, _ := ConvertValue(t, FormatDotNetFloat(f, 64))
	return v, clamped, nil
}

// TextMaxLength ports UIPropertyString's length limit: 256 by default, the EDS
// MaxLength when known, and MaxLength-1 once the device reported len > 1.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/UIPropertyString.cs:141-178,180-235
func TextMaxLength(p Property, eds *EDSParam) int {
	n := uint64(256)
	if eds != nil {
		n = eds.MaxLength
	}
	if p.MaxLength > 1 {
		n = p.MaxLength - 1
	}
	return int(n)
}

// TextInput ports UIPropertyString.onTxtChanged: truncate to the max length.
func TextInput(p Property, eds *EDSParam, text string) string {
	max := TextMaxLength(p, eds)
	if r := []rune(text); len(r) > max {
		return string(r[:max])
	}
	return text
}

// ---------------------------------------------------------------- search

// SearchOptions are PanelAllProperties' search check boxes (Name and ID are
// on by default).
type SearchOptions struct {
	Name, Value, ID, Regex bool
}

// DefaultSearchOptions mirrors AddSearchFunction (m_chkName/m_chkId active).
var DefaultSearchOptions = SearchOptions{Name: true, ID: true}

// SearchEmptyTermError is OnSearchClicked's error for a blank term.
const SearchEmptyTermError = "Please enter a search term."

// SearchNoResults ports the "No results found for: {term}" header.
func SearchNoResults(term string) string { return "No results found for: " + term }

// SearchResult carries the matches plus an optional warning (the invalid
// regex message shown before falling back to plain text search).
type SearchResult struct {
	Matches []Property
	Warning string
}

// Search ports PanelAllProperties.OnSearchClicked: every property with a
// value whose name (Title or EDSParameter.Name), value or id
// ("0x{Id:X4}_{SubId:X}") matches the term — a case-insensitive substring, or
// a case-insensitive regular expression when opts.Regex (Go RE2 syntax
// instead of .NET's; an invalid pattern falls back to text search with the
// C# warning).
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/PanelAllProperties.cs:122-236
func Search(pc *PropertyCache, term string, opts SearchOptions) (SearchResult, error) {
	if strings.TrimSpace(term) == "" {
		return SearchResult{}, fmt.Errorf("%s", SearchEmptyTermError)
	}
	var res SearchResult
	var re *regexp.Regexp
	if opts.Regex {
		r, err := regexp.Compile("(?i)" + term)
		if err != nil {
			res.Warning = fmt.Sprintf("Invalid regular expression: %s\nNormal text search will be used instead.", err.Error())
		} else {
			re = r
		}
	}
	match := func(text string) bool {
		if strings.TrimSpace(text) == "" {
			return false
		}
		if re != nil {
			return re.MatchString(text)
		}
		return strings.Contains(strings.ToLower(text), strings.ToLower(term))
	}
	for _, p := range pc.All() {
		if !p.HasValue {
			continue
		}
		hit := false
		if opts.Name && (match(pc.Title(p)) || match(pc.ParameterName(p))) {
			hit = true
		}
		if !hit && opts.Value && match(p.Value) {
			hit = true
		}
		if !hit && opts.ID && match(fmt.Sprintf("0x%04X_%X", p.ID, p.Sub)) {
			hit = true
		}
		if hit {
			res.Matches = append(res.Matches, p)
		}
	}
	return res, nil
}

// SDTName returns the ICUNetwork.SDT enum name of t (as the installer spells
// it, including "DOAMIN").
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/SDT.cs
func SDTName(t api.SDT) string {
	switch t {
	case api.SDTLowLimit:
		return "LOW_LIMIT"
	case api.SDTBoolean:
		return "BOOLEAN"
	case api.SDTInteger8:
		return "INTEGER8"
	case api.SDTInteger16:
		return "INTEGER16"
	case api.SDTInteger24:
		return "INTEGER24"
	case api.SDTInteger32:
		return "INTEGER32"
	case api.SDTInteger40:
		return "INTEGER40"
	case api.SDTInteger48:
		return "INTEGER48"
	case api.SDTInteger56:
		return "INTEGER56"
	case api.SDTInteger64:
		return "INTEGER64"
	case api.SDTUnsigned8:
		return "UNSIGNED8"
	case api.SDTUnsigned16:
		return "UNSIGNED16"
	case api.SDTUnsigned24:
		return "UNSIGNED24"
	case api.SDTUnsigned32:
		return "UNSIGNED32"
	case api.SDTUnsigned40:
		return "UNSIGNED40"
	case api.SDTUnsigned48:
		return "UNSIGNED48"
	case api.SDTUnsigned56:
		return "UNSIGNED56"
	case api.SDTUnsigned64:
		return "UNSIGNED64"
	case api.SDTReal32:
		return "REAL32"
	case api.SDTReal64:
		return "REAL64"
	case api.SDTVisibleString:
		return "VISIBLE_STRING"
	case api.SDTOctetString:
		return "OCTET_STRING"
	case api.SDTUnicodeString:
		return "UNICODE_STRING"
	case api.SDTTimeOfDay:
		return "TIME_OF_DAY"
	case api.SDTTimeDifference:
		return "TIME_DIFFERENCE"
	case api.SDTDomain:
		return "DOAMIN"
	case api.SDTByteArray:
		return "BYTEARRAY"
	case api.SDTArray16:
		return "ARRAY_16"
	case api.SDTHighLimit:
		return "HIGH_LIMIT"
	}
	return "SDT(" + strconv.Itoa(int(t)) + ")"
}

// RowDisplay renders a row's value for the table: DateTime rows through
// FormatDeviceDateTime with the device offset.
func RowDisplay(pc *PropertyCache, r PropertyRow) string {
	if r.Format == FormatDateTime {
		return FormatDeviceDateTime(r.Value, DeviceUTCOffsetMinutes(pc))
	}
	return r.Display
}
