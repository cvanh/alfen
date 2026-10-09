package eds

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"alfen/aceclient/internal/api"
)

// Texts of PanelAllProperties (PanelAllProperties.cs).
const (
	AllPropertiesTitle   = "All Properties" // PanelBase.Title
	AllPropertiesTooltip = "All settings"   // Tooltip, also the configuration panel title
	AllPropertiesIcon    = "settings-gears.png"

	// AdvancedWarning is the AddWarningText row heading every category.
	AdvancedWarning = "These settings are advanced settings, only change them when you know what you are doing. Incorrect values might damage your charging station!"

	SearchCategoryName = "Search"
	SearchButtonText   = "Search"
	SearchFilterLabel  = "Filter: "

	// MsgEnterSearchTerm is shown (MessageDialog.ShowError) for a blank term.
	MsgEnterSearchTerm = "Please enter a search term."
	// MsgInvalidRegexFmt is shown when the regex does not compile; %s is the
	// exception message. The search then falls back to plain text.
	MsgInvalidRegexFmt = "Invalid regular expression: %s\nNormal text search will be used instead."
	// MsgNoResultsFmt is the small header added when nothing matched; %s is
	// the search term.
	MsgNoResultsFmt = "No results found for: %s"

	// AdvancedSettingsLabel is the UIConfigurationPanel advanced-mode toggle.
	AdvancedSettingsLabel = "Advanced Settings"
)

// Search filter captions (check boxes) in PanelAllProperties.AddSearchFunction.
var SearchFilterNames = [...]string{"Name", "Value", "ID", "Regex"}

// Object ids PanelAllProperties.AddProperties forces read-only unless the
// panel was opened with Ctrl+Shift held (specialWritePermission):
// sysChargePointSerialNumber (Object Number), sysChargePointModel and 0x204F.
var specialWriteIDs = map[uint16]bool{0x2051: true, 0x2050: true, 0x204F: true}

// IsSpecialWriteID reports whether id is one of those (8273, 8272, 8271).
func IsSpecialWriteID(id uint16) bool { return specialWriteIDs[id] }

// UIKind is the widget PanelAllProperties.AddProperties creates for a property.
type UIKind int

const (
	KindNone         UIKind = iota // not shown
	KindText                       // UIPropertyString (editable text, AddText)
	KindNumber                     // UIPropertyNumber (spin button)
	KindCheckbox                   // UIPropertyCheckbox
	KindSelect                     // UIPropertySelect (EDS options)
	KindCustomText                 // UIPropertyString without binding (AddCustomText): read-only label/value
	KindReadOnlyText               // UIPropertyReadOnlyString (AddReadOnlyText)
)

var kindNames = [...]string{"None", "Text", "Number", "Checkbox", "Select", "CustomText", "ReadOnlyText"}

// String names the kind after its C# widget role.
func (k UIKind) String() string {
	if k >= 0 && int(k) < len(kindNames) {
		return kindNames[k]
	}
	return fmt.Sprintf("UIKind(%d)", int(k))
}

// Field is one property row: the widget kind plus everything the C# widget
// derives from EDS and the device property.
type Field struct {
	Prop  Property // merged property (ReadOnly after the special-id rule)
	Kind  UIKind
	Label string // label text as displayed after the first refresh

	// ReadOnly is the widget's effective read-only state (m_fReadOnly):
	// ICUProperty.ReadOnly for bound kinds, always true for KindCustomText
	// and KindReadOnlyText. Access-rights forcing (CheckAccessRights) is
	// applied by the caller via FeatureRightID.
	ReadOnly bool

	StringType    UIPropertyStringType // KindText / KindReadOnlyText
	MaxLen        uint64               // KindText: entry truncation length
	Spin          SpinConfig           // KindNumber
	Factor        float64              // KindNumber (AddNumber/AddCustomNumber factor; 1.0 here)
	Options       []Option             // KindSelect
	CaseSensitive bool                 // KindSelect
	ValueMask     int                  // KindCheckbox / KindSelect (0 in All Properties)
	CustomValue   string               // KindCustomText

	FeatureRightID string // "ID_XXXX" for id-bound rows, "" for custom rows
	AlwaysReadOnly bool   // AddPropertyBase(forceReadonly: true) (AddReadOnlyText)

	// UIConfigCategory row flags (all false in All Properties).
	Advanced, Hide, Confidential bool
}

// BuildField ports one iteration of PanelAllProperties.AddProperties plus the
// constructor/first OnRefreshDisplay of the widget it creates. ok is false when
// the C# adds no row: the property has no value (unsupported SDT), or it is a
// BYTEARRAY in the "leds" category.
//
// Kind selection by device data type:
//
//	VISIBLE_STRING, OCTET_STRING, UNICODE_STRING   Text
//	INTEGER8/16/32/64, UNSIGNED8/16/32, REAL32/64  not in EDS: Checkbox if device len == 2, else Number (label ICUName)
//	                                               EDS with 1 option: Checkbox; with >1 options: Select
//	                                               EDS without options: Number (3 digits for REAL)
//	UNSIGNED64                                     read-only: CustomText(Title, value); else ReadOnlyText(DateTime)
//	BOOLEAN                                        Checkbox
//	BYTEARRAY                                      CustomText(ICUName, X2 CSV) unless category "leds"
//	ARRAY_16                                       CustomText(ICUName, X4 CSV)
//	other (LOW_LIMIT)                              CustomText(ICUName, value)
func (d *Dictionary) BuildField(p api.Property, specialWrite bool) (Field, bool) {
	return d.buildField(d.Merge(p), specialWrite)
}

func (d *Dictionary) buildField(m Property, specialWrite bool) (Field, bool) {
	if IsSpecialWriteID(m.ID) {
		m.ReadOnly = !specialWrite
	}
	if m.Typed.IsNull() {
		return Field{}, false
	}
	f := Field{Prop: m, ReadOnly: m.ReadOnly, Factor: 1.0, CaseSensitive: true}
	bound := func(kind UIKind) {
		f.Kind = kind
		f.FeatureRightID = FeatureRightID(m.ID)
	}
	custom := func(label, value string) {
		f.Kind = KindCustomText
		f.Label = label
		f.CustomValue = value
		f.ReadOnly = true
	}
	// labelBound is the label of a widget built from (Id, SubId) without an
	// override: UIPropertyBase.LabelText, replaced by ICUProperty.Title when
	// the id is unknown (m_fUnknownTitle) and a value is present.
	labelBound := func() string {
		l, unknown := d.LabelText(m.ID, m.Sub, "")
		if unknown {
			return m.Title
		}
		return l
	}

	switch m.DataType {
	case api.SDTVisibleString, api.SDTOctetString, api.SDTUnicodeString:
		bound(KindText)
		f.Label = labelBound()
		f.StringType = StringTypeString
		f.MaxLen = TextMaxLen(d.FindParameter(int(m.ID), int(m.Sub)), StringTypeString, m.MaxLength)
	case api.SDTInteger8, api.SDTInteger16, api.SDTInteger32, api.SDTUnsigned8,
		api.SDTUnsigned16, api.SDTUnsigned32, api.SDTReal32, api.SDTReal64, api.SDTInteger64:
		digits := 0
		if m.DataType == api.SDTReal32 || m.DataType == api.SDTReal64 {
			digits = 3
		}
		switch {
		case m.Param == nil:
			if m.MaxLength == 2 {
				bound(KindCheckbox) // AddCheckBox(Id, SubId, 0, ICUName)
				f.Label = m.ICUName
			} else {
				bound(KindNumber) // AddCustomNumber(Id, SubId, digits, ICUName, 1.0)
				f.Label = m.ICUName
				f.Spin = NumberSpin(m.DataType, digits, m.MaxLength)
			}
		case len(m.Param.Options) > 0:
			if len(m.Param.Options) == 1 {
				bound(KindCheckbox) // AddCheckBox(Id, SubId, 0, "")
				f.Label = labelBound()
			} else {
				bound(KindSelect) // AddSelect(Id, SubId, 0, 0, "", caseSensitive: true)
				f.Label = m.Param.Title
				f.Options = SelectOptions(d.FindParameter(int(m.ID), int(m.Sub)), nil)
			}
		default:
			bound(KindNumber) // AddNumber(Id, SubId, digits)
			f.Label = labelBound()
			f.Spin = NumberSpin(m.DataType, digits, m.MaxLength)
		}
	case api.SDTUnsigned64:
		if m.ReadOnly {
			custom(m.Title, m.Typed.String()) // AddCustomText(Title, Value.ToString())
		} else {
			bound(KindReadOnlyText) // AddReadOnlyText(Id, SubId, "", DateTime)
			f.Label = labelBound()
			f.StringType = StringTypeDateTime
			f.AlwaysReadOnly = true
			f.ReadOnly = true
		}
	case api.SDTBoolean:
		bound(KindCheckbox) // AddCheckBox(Id, SubId, 0, "")
		f.Label = labelBound()
	case api.SDTByteArray:
		if strings.ToLower(m.Category) == "leds" {
			return Field{}, false
		}
		custom(m.ICUName, m.Typed.HexString())
	case api.SDTArray16:
		custom(m.ICUName, m.Typed.HexString())
	default:
		custom(m.ICUName, m.Typed.String())
	}
	return f, true
}

// DisplayText is the text the widget shows for the field's current value:
// Text -> TextDisplay, Number -> FormatSpin(SpinValue), Select -> the
// selected option's Title ("" when none), CustomText -> CustomValue,
// ReadOnlyText -> FormatReadOnly (utcOffsetMinutes from
// DeviceUTCOffsetMinutes), Checkbox -> "" (see Checked).
func (f Field) DisplayText(utcOffsetMinutes int) string {
	switch f.Kind {
	case KindText:
		return TextDisplay(f.Prop.Typed)
	case KindNumber:
		return FormatSpin(SpinValue(f.Prop.Typed, f.Factor, f.Spin.Digits), f.Spin.Digits)
	case KindSelect:
		if o, ok := SelectedOption(f.Prop.Typed, f.ValueMask, f.CaseSensitive, f.Options); ok {
			return o.Title
		}
		return ""
	case KindCustomText:
		return f.CustomValue
	case KindReadOnlyText:
		s, _ := FormatReadOnly(f.StringType, f.Prop.Typed, utcOffsetMinutes)
		return s
	}
	return ""
}

// Checked is CheckboxChecked for a KindCheckbox field.
func (f Field) Checked() bool { return CheckboxChecked(f.Prop.Typed, f.ValueMask) }

// Selected is SelectedOption for a KindSelect field.
func (f Field) Selected() (Option, bool) {
	return SelectedOption(f.Prop.Typed, f.ValueMask, f.CaseSensitive, f.Options)
}

// ErrReadOnly is returned by the Commit methods for read-only fields (the C#
// widgets ignore edits when m_fReadOnly/m_fForceReadOnly).
var ErrReadOnly = errors.New("eds: property is read-only")

// ErrWrongKind is returned by a Commit method that does not match Field.Kind.
var ErrWrongKind = errors.New("eds: commit does not match the field kind")

// CommitText ports UIPropertyString.onTxtChanged for a KindText field: the
// filtered/truncated entry text and the value assigned to the property.
func (f Field) CommitText(text string) (shown string, v Value, err error) {
	if f.Kind != KindText {
		return "", Value{}, ErrWrongKind
	}
	shown = FilterTextInput(text, f.StringType, f.MaxLen)
	if f.ReadOnly {
		return shown, f.Prop.Typed, ErrReadOnly
	}
	return shown, ParseValue(f.Prop.DataType, shown), nil
}

// CommitNumber ports WindowsSpinButton.parseTextBox + UIPropertyNumber.
// OnSpinValueChanged for a KindNumber field: the clamped/rounded spin value
// and the value assigned to the property.
func (f Field) CommitNumber(text string) (spin float64, v Value, err error) {
	if f.Kind != KindNumber {
		return 0, Value{}, ErrWrongKind
	}
	spin = ParseSpinText(text, f.Spin)
	if f.ReadOnly {
		return spin, f.Prop.Typed, ErrReadOnly
	}
	return spin, CommitNumber(f.Prop.DataType, spin, f.Factor), nil
}

// CommitSelect ports UIPropertySelect.OnSelectionChanged for a KindSelect
// field; optionValue is the Value of the chosen Option. write is false when
// the C# assigns nothing (also when optionValue is not one of f.Options: the
// combo box cannot select it).
func (f Field) CommitSelect(optionValue string) (v Value, write bool, err error) {
	if f.Kind != KindSelect {
		return Value{}, false, ErrWrongKind
	}
	if f.ReadOnly {
		return f.Prop.Typed, false, ErrReadOnly
	}
	found := false
	for _, o := range f.Options {
		if o.Value == optionValue {
			found = true
			break
		}
	}
	if !found {
		return f.Prop.Typed, false, nil
	}
	v, write = CommitSelect(f.Prop.DataType, f.Prop.Typed, optionValue, f.ValueMask, f.CaseSensitive)
	return v, write, nil
}

// CommitCheckbox ports UIPropertyCheckbox.OnChkClicked for a KindCheckbox
// field.
func (f Field) CommitCheckbox(checked bool) (Value, error) {
	if f.Kind != KindCheckbox {
		return Value{}, ErrWrongKind
	}
	if f.ReadOnly {
		return f.Prop.Typed, ErrReadOnly
	}
	return CommitCheckbox(f.Prop.DataType, f.Prop.Typed, checked, f.ValueMask), nil
}

// Category ports UIConfigCategory (UIConfigCategory.cs) as filled by the All
// Properties panel.
type Category struct {
	Name  string // UIConfigCategory.Name (capitalized device category)
	Title string // UIConfigCategory.Title ("" in All Properties)

	// Warning is the AddWarningText row added first (AdvancedWarning); it
	// counts as a property row. Empty for categories without it.
	Warning string
	Fields  []Field
	Widgets int // AddWidget count (the Search category has 3)

	ShowCategory bool // default true
	Hide         bool // set when no property row besides the warning was added
}

// rowCount is UIConfigCategory.m_lstProperties.Count.
func (c *Category) rowCount() int {
	n := len(c.Fields)
	if c.Warning != "" {
		n++
	}
	return n
}

// Header ports UIConfigurationPanel.ShowContent: Title, else Name.
func (c *Category) Header() string {
	if c.Title == "" {
		return c.Name
	}
	return c.Title
}

// ContainsAdvanced ports UIConfigCategory.ContainsAdvancedProp.
func (c *Category) ContainsAdvanced() bool {
	for _, f := range c.Fields {
		if f.Advanced {
			return true
		}
	}
	return false
}

// hasNonAdvanced is "m_lstProperties.Where(a => !a.IsAdvancedProp).Any()";
// the warning label is a non-advanced row.
func (c *Category) hasNonAdvanced() bool {
	if c.Warning != "" {
		return true
	}
	for _, f := range c.Fields {
		if !f.Advanced {
			return true
		}
	}
	return false
}

// Listed ports the category filter of UIConfigurationPanel.LoadCategories:
// hidden or empty categories are skipped; others are listed when they have a
// non-advanced row or widgets (and ShowCategory), or — in advanced mode —
// when they only hold advanced rows.
func (c *Category) Listed(showAdvanced bool) bool {
	if c.Hide || (c.rowCount() == 0 && c.Widgets == 0) {
		return false
	}
	flag := c.hasNonAdvanced()
	return ((flag || c.Widgets != 0) && c.ShowCategory) || (showAdvanced && !flag)
}

// FieldVisible ports UIConfigCategory.ShowProperties for one field.
func FieldVisible(f Field, show, showAdvanced bool) bool {
	return show && ((!f.Advanced && !f.Hide) || (f.Advanced && showAdvanced)) && !f.Confidential
}

// ListedCategories ports UIConfigurationPanel.LoadCategories: the indexes of
// the categories shown in the category tree, in order.
func ListedCategories(cats []Category, showAdvanced bool) []int {
	var out []int
	for i := range cats {
		if cats[i].Listed(showAdvanced) {
			out = append(out, i)
		}
	}
	return out
}

// SelectCategory ports LoadCategories' re-selection: the listed category whose
// Name equals the previously selected one, else the first (-1 when none is
// listed).
func SelectCategory(cats []Category, listed []int, prevName string) int {
	if prevName != "" {
		for _, i := range listed {
			if cats[i].Name == prevName {
				return i
			}
		}
	}
	if len(listed) == 0 {
		return -1
	}
	return listed[0]
}

// ShowAdvancedToggle ports UIConfigurationPanel.UpdateAdvancedCheckBox: the
// "Advanced Settings" check box is shown (and enabled) only when some category
// contains an advanced row.
func ShowAdvancedToggle(cats []Category) bool {
	for i := range cats {
		if cats[i].ContainsAdvanced() {
			return true
		}
	}
	return false
}

// CapitalizeCategory ports PanelAllProperties' category naming:
// char.ToUpper(category[0]) + category.Substring(1).
func CapitalizeCategory(category string) string {
	r, n := utf8.DecodeRuneInString(category)
	if r == utf8.RuneError && n <= 1 {
		return category
	}
	return string(unicode.ToUpper(r)) + category[n:]
}

// GroupAllProperties ports PanelAllProperties.OnChangeDevice: for every
// distinct device category (in the order /api/categories returned them), the
// properties of that category sorted by ICUProperty.ID_SUB become a Category
// named CapitalizeCategory(category) with the AdvancedWarning row first.
//
// As in the C#, a category name of length <= 1 opens no new Category: its rows
// are appended to the previous one (when there is none the C# throws a
// NullReferenceException; those rows are dropped here). A Category whose
// properties all produced no row is marked Hide. The trailing "Search"
// category is not included (see Search).
//
// props are all properties read from the device (ReadProperties per category,
// or "limit=500" on AHP >= 2.2); specialWrite is the Ctrl+Shift state when the
// panel was opened.
func (d *Dictionary) GroupAllProperties(categories []string, props []api.Property, specialWrite bool) []Category {
	merged := d.MergeAll(props)
	var out []Category
	cur := -1
	seen := map[string]bool{}
	for _, category := range categories {
		if seen[category] {
			continue
		}
		seen[category] = true
		var list []Property
		for _, m := range merged {
			if m.Category == category {
				list = append(list, m)
			}
		}
		if len(list) == 0 {
			continue
		}
		sort.SliceStable(list, func(i, j int) bool {
			return IDSubPadded(list[i].ID, list[i].Sub) < IDSubPadded(list[j].ID, list[j].Sub)
		})
		if category != "" && utf16Len(category) > 1 {
			out = append(out, Category{Name: CapitalizeCategory(category), Warning: AdvancedWarning, ShowCategory: true})
			cur = len(out) - 1
		}
		if cur < 0 {
			continue
		}
		for _, m := range list {
			if f, ok := d.buildField(m, specialWrite); ok {
				out[cur].Fields = append(out[cur].Fields, f)
			}
		}
		out[cur].Hide = out[cur].rowCount() == 1
	}
	return out
}

// SearchOptions are the All Properties search inputs.
type SearchOptions struct {
	Term  string
	Name  bool // match ICUProperty.Title or the EDS ParameterName
	Value bool // match Value.ToString()
	ID    bool // match "0x{Id:X4}_{SubId:X}"
	Regex bool // Term is a regular expression (case-insensitive)
}

// DefaultSearchOptions has the check boxes as AddSearchFunction leaves them:
// Name and ID active.
func DefaultSearchOptions() SearchOptions { return SearchOptions{Name: true, ID: true} }

// SearchResult is what OnSearchClicked puts in the Search category.
type SearchResult struct {
	Matches []Property // matched properties, in PropertyDictionary order
	Fields  []Field    // rows built from Matches (AddProperties)
	// Warning is the MsgInvalidRegexFmt dialog text when Regex was set and the
	// pattern does not compile (a plain search was done instead).
	Warning string
	// NoResults is the MsgNoResultsFmt header when nothing matched, else "".
	NoResults string
}

// Search ports PanelAllProperties.OnSearchClicked over all device-read props:
// a blank term is an error (MsgEnterSearchTerm); each property with a value is
// matched on Name, then Value, then ID per the enabled filters, case-
// insensitively (OrdinalIgnoreCase substring, or the regex).
//
// Regular expressions use Go's RE2 syntax instead of .NET's (a compile error
// message therefore differs from the C#).
func (d *Dictionary) Search(props []api.Property, opt SearchOptions, specialWrite bool) (SearchResult, error) {
	var res SearchResult
	if strings.TrimSpace(opt.Term) == "" {
		return res, errors.New(MsgEnterSearchTerm)
	}
	var re *regexp.Regexp
	if opt.Regex {
		r, err := regexp.Compile("(?i)" + opt.Term)
		if err != nil {
			res.Warning = fmt.Sprintf(MsgInvalidRegexFmt, err.Error())
		} else {
			re = r
		}
	}
	for _, m := range d.MergeAll(props) {
		if m.Typed.IsNull() {
			continue
		}
		hit := false
		if opt.Name && (isTextMatch(m.Title, opt.Term, re) || (m.Param != nil && isTextMatch(m.Param.Name, opt.Term, re))) {
			hit = true
		}
		if !hit && opt.Value && isTextMatch(m.Typed.String(), opt.Term, re) {
			hit = true
		}
		if !hit && opt.ID && isTextMatch(searchID(m.ID, m.Sub), opt.Term, re) {
			hit = true
		}
		if hit {
			res.Matches = append(res.Matches, m)
		}
	}
	for _, m := range res.Matches {
		if f, ok := d.buildField(m, specialWrite); ok {
			res.Fields = append(res.Fields, f)
		}
	}
	if len(res.Matches) == 0 {
		res.NoResults = fmt.Sprintf(MsgNoResultsFmt, opt.Term)
	}
	return res, nil
}

// isTextMatch ports PanelAllProperties.IsTextMatch.
func isTextMatch(text, term string, re *regexp.Regexp) bool {
	if strings.TrimSpace(text) == "" {
		return false
	}
	if re != nil {
		return re.MatchString(text)
	}
	return strings.Contains(strings.ToUpper(text), strings.ToUpper(term))
}

// ParseCategories ports the body parsing of ICULanDevice.RequestCategories
// (GET /api/categories): every non-null element of the top-level
// "categories" array, Convert.ToString'd and trimmed of '"' and ' '. An empty
// or unparseable body yields no categories.
func ParseCategories(body string) []string {
	var out []string
	if body == "" {
		return out
	}
	var top map[string]any
	if err := json.Unmarshal([]byte(body), &top); err != nil {
		return out
	}
	arr, ok := top["categories"].([]any)
	if !ok {
		return out
	}
	for _, item := range arr {
		if item == nil {
			continue
		}
		var s string
		switch v := item.(type) {
		case string:
			s = v
		case bool:
			s = "False"
			if v {
				s = "True"
			}
		case float64:
			s = netFormatG(v, 15)
		default:
			b, _ := json.Marshal(v)
			s = string(b)
		}
		out = append(out, strings.Trim(s, "\" "))
	}
	return out
}
