package core

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"alfen/aceclient/internal/api"
)

// EDSOption mirrors ICUSettings.EDSParameterOption (Value + Title).
type EDSOption struct {
	Value string
	Title string
}

// EDSParam is the subset of ICUSettings.EDSParameter the views consume.
//
// Source: firmware/decompiled/ACESettings/ICUSettings/EDSParameter.cs
type EDSParam struct {
	Name      string // ParameterName; ICUProperty.ICUName becomes "OD_" + Name
	Title     string
	Units     string
	MaxLength uint64
	Options   []EDSOption
}

// EDSLookup resolves EDS.xml metadata for an object-dictionary entry. It is
// nil until internal/eds (Phase 2) is wired in; every consumer here falls back
// to the C# behaviour for a property that is not in the EDS.
type EDSLookup func(id uint16, sub byte) (EDSParam, bool)

// Key is the combined property id the installer uses in UpdateProperties
// (uint)((id << 8) | sub), e.g. 0x360001 for 0x3600_01.
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICULanDevice.cs:2814-2822 (HasProperty's combined id)
func Key(id uint16, sub byte) uint32 { return uint32(id)<<8 | uint32(sub) }

// IDSub mirrors ICUProperty.ID_SUB: "{Id:X4}_{SubId:X2}" (the sort key).
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICUProperty.cs:57
func IDSub(id uint16, sub byte) string { return fmt.Sprintf("%04X_%02X", id, sub) }

// ODIndex mirrors ICUProperty.ODIndex: "{Id:X4}_{SubId:X}" (the wire id).
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICUProperty.cs:59
func ODIndex(id uint16, sub byte) string { return fmt.Sprintf("%04X_%X", id, sub) }

// Property is the installer-side state of one object-dictionary entry
// (ICUNetwork.ICUProperty): the device-reported metadata plus the converted
// device value and the current (possibly edited) value.
//
// The embedded api.Property carries the metadata; its Value field is the
// CURRENT value (ICUProperty.Value) in canonical string form, i.e. exactly
// what StoreProperties will serialise.
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICUProperty.cs:13-409
type Property struct {
	api.Property
	DeviceValue string // ICUProperty.DeviceValue (canonical string)
	HasValue    bool   // ICUProperty.Value != null
	Changed     bool   // ICUProperty.IsChanged
}

// Key returns the combined id of p.
func (p Property) Key() uint32 { return Key(p.ID, p.Sub) }

// ConvertValue ports ICUProperty.SetValue followed by Value.ToString(): it
// converts text the way SetValue's per-SDT switch does and renders the result
// the way the installer later serialises it (StoreProperties uses
// Value.ToString(), en-US for REAL32/REAL64).
//
// ok is false for the data types SetValue has no branch for (OCTET_STRING,
// UNICODE_STRING, TIME_OF_DAY, TIME_DIFFERENCE, DOMAIN, the 24/40/48/56-bit
// integers, HIGH_LIMIT): the installer leaves Value unchanged (null for a
// freshly read property) in that case.
//
// Fidelity notes: unparsable numbers become 0 (not an error); BOOLEAN renders
// as .NET's "True"/"False" and an unparsable boolean becomes (byte)0 → "0";
// REAL32 renders with .NET Framework's "G" (7 significant digits), REAL64 with
// "G" (15 digits); BYTEARRAY/ARRAY_16 parse a hex CSV and render "X2"/"X4".
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICUProperty.cs:97-341 (SetValue)
func ConvertValue(t api.SDT, text string) (string, bool) {
	switch t {
	case api.SDTVisibleString:
		return text, true
	case api.SDTUnsigned8:
		if strings.EqualFold(text, "false") {
			return "0", true
		}
		if strings.EqualFold(text, "true") {
			return "1", true
		}
		return uintOrZero(text, 8), true
	case api.SDTInteger8:
		if strings.EqualFold(text, "false") {
			return "0", true
		}
		if strings.EqualFold(text, "true") {
			return "1", true
		}
		return intOrZero(text, 8), true
	case api.SDTUnsigned16:
		return uintOrZero(text, 16), true
	case api.SDTUnsigned32:
		return uintOrZero(text, 32), true
	case api.SDTUnsigned64:
		return uintOrZero(text, 64), true
	case api.SDTInteger16:
		return intOrZero(text, 16), true
	case api.SDTInteger32:
		return intOrZero(text, 32), true
	case api.SDTInteger64:
		return intOrZero(text, 64), true
	case api.SDTReal32:
		f, ok := parseDotNetFloat(strings.ReplaceAll(text, ",", "."), 32)
		if !ok {
			return "0", true
		}
		return FormatDotNetFloat(f, 32), true
	case api.SDTReal64:
		f, ok := parseDotNetFloat(strings.ReplaceAll(text, ",", "."), 64)
		if !ok {
			return "0", true
		}
		return FormatDotNetFloat(f, 64), true
	case api.SDTBoolean:
		if b, ok := parseDotNetBool(text); ok {
			return dotNetBool(b), true
		}
		return "0", true // m_objValue = (byte)0
	case api.SDTByteArray:
		return hexCSV(text, 8), true
	case api.SDTArray16:
		return hexCSV(text, 16), true
	case api.SDTLowLimit:
		return "", true // m_objValue = string.Empty; ReadOnly = true (caller)
	default:
		return "", false
	}
}

func uintOrZero(text string, bits int) string {
	n, ok := parseDotNetUint(text, bits)
	if !ok {
		return "0"
	}
	return strconv.FormatUint(n, 10)
}

func intOrZero(text string, bits int) string {
	n, ok := parseDotNetInt(text, bits, true)
	if !ok {
		return "0"
	}
	return strconv.FormatInt(n, 10)
}

// hexCSV ports the BYTEARRAY/ARRAY_16 branches: Split(','), each element
// TryParse(NumberStyles.HexNumber) or 0, rendered "X2"/"X4" joined by ','.
func hexCSV(text string, bits int) string {
	parts := strings.Split(text, ",")
	out := make([]string, len(parts))
	for i, p := range parts {
		n, ok := parseDotNetHex(p, bits)
		if !ok {
			n = 0
		}
		if bits == 8 {
			out[i] = fmt.Sprintf("%02X", n)
		} else {
			out[i] = fmt.Sprintf("%04X", n)
		}
	}
	return strings.Join(out, ",")
}

// dotNetFloatRe is NumberStyles.Float: [ws][sign]digits[.digits][e[sign]digits][ws]
// (either side of the decimal point may be empty, but not both).
var dotNetFloatRe = regexp.MustCompile(`^[+-]?(\d+\.?\d*|\.\d+)([eE][+-]?\d+)?$`)

// parseDotNetFloat ports Single/Double.TryParse(s, NumberStyles.Float,
// CultureInfo.InvariantCulture) as of .NET Framework 4.8: out-of-range values
// fail (no ±Infinity result), the invariant "NaN"/"Infinity"/"-Infinity"
// symbols are accepted.
func parseDotNetFloat(s string, bits int) (float64, bool) {
	t := trimDotNetWhite(s)
	switch t {
	case "NaN":
		return math.NaN(), true
	case "Infinity", "+Infinity":
		return math.Inf(1), true
	case "-Infinity":
		return math.Inf(-1), true
	}
	if !dotNetFloatRe.MatchString(t) {
		return 0, false
	}
	f, err := strconv.ParseFloat(t, bits)
	if err != nil {
		var ne *strconv.NumError
		if errors.As(err, &ne) && ne.Err == strconv.ErrRange && !math.IsInf(f, 0) {
			return f, true // underflow rounds to (denormal) zero like .NET
		}
		return 0, false
	}
	return f, true
}

// FormatDotNetFloat renders f like .NET Framework's Single/Double.ToString()
// ("G", 7 resp. 15 significant digits, invariant culture).
func FormatDotNetFloat(f float64, bits int) string {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	}
	prec := 15
	if bits == 32 {
		prec = 7
		f = float64(float32(f))
	}
	s := strconv.FormatFloat(f, 'G', prec, bits)
	if s == "-0" {
		s = "0" // .NET Framework prints negative zero as "0"
	}
	return s
}

// parseDotNetBool ports Boolean.TryParse: "True"/"False" case-insensitively,
// ignoring surrounding white space and trailing NULs.
func parseDotNetBool(s string) (bool, bool) {
	t := strings.TrimRight(strings.TrimSpace(s), "\x00")
	t = strings.TrimSpace(t)
	switch {
	case strings.EqualFold(t, "True"):
		return true, true
	case strings.EqualFold(t, "False"):
		return false, true
	}
	return false, false
}

func dotNetBool(b bool) string {
	if b {
		return "True"
	}
	return "False"
}

// PropertyCache ports the ICUPropertyDictionary of one ICULanDevice: every
// property ever read, in first-seen order, with IsChanged tracking.
// It is safe for concurrent use.
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICUPropertyDictionary.cs:9-70
type PropertyCache struct {
	mu    sync.RWMutex
	order []uint32
	m     map[uint32]*Property
	eds   EDSLookup
}

// NewPropertyCache returns an empty cache.
func NewPropertyCache() *PropertyCache {
	return &PropertyCache{m: map[uint32]*Property{}}
}

// SetEDS installs (or clears, with nil) the EDS lookup used for names/titles.
func (pc *PropertyCache) SetEDS(l EDSLookup) {
	pc.mu.Lock()
	pc.eds = l
	pc.mu.Unlock()
}

// EDS returns the installed lookup (may be nil).
func (pc *PropertyCache) EDS() EDSLookup {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return pc.eds
}

// Lookup resolves EDS metadata for (id, sub), false when no EDS is wired or
// the entry is unknown (DataSheet.FindParameter == null).
func (pc *PropertyCache) Lookup(id uint16, sub byte) (EDSParam, bool) {
	l := pc.EDS()
	if l == nil {
		return EDSParam{}, false
	}
	return l(id, sub)
}

// Merge ports ICULanDevice.ParseProperty for each entry of a /api/prop read:
// the entry is created on first sight, DataType/ReadOnly/Category/MaxLength
// are refreshed, and the value is re-initialised (SetInitialValue) unless the
// user has a pending change (IsChanged). A JSON null value counts as "0"
// (`if (obj == null) obj = "0"`), which also covers the ":nan" → ":null"
// rewrite of HandleIncomingResponse.
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICULanDevice.cs:1805-1860 (ParseProperty)
func (pc *PropertyCache) Merge(props []api.Property) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	for _, in := range props {
		k := Key(in.ID, in.Sub)
		p, ok := pc.m[k]
		if !ok {
			p = &Property{}
			p.ID, p.Sub = in.ID, in.Sub
			pc.m[k] = p
			pc.order = append(pc.order, k)
		}
		p.DataType = in.DataType
		if p.Name == "" {
			p.Name = in.Name
		}
		p.ReadOnly = in.ReadOnly
		p.Category = in.Category
		p.MaxLength = in.MaxLength
		raw := in.Value
		if raw == "null" {
			raw = "0"
		}
		if !p.Changed {
			p.setInitialValue(raw)
		}
	}
}

// setInitialValue ports ICUProperty.SetInitialValue.
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICUProperty.cs:382-391
func (p *Property) setInitialValue(text string) {
	if v, ok := ConvertValue(p.DataType, text); ok {
		p.Value = v
		p.HasValue = true
		if p.DataType == api.SDTLowLimit {
			p.ReadOnly = true
		}
	}
	p.DeviceValue = p.Value
	p.Changed = false
}

// Get returns a copy of the cached property.
func (pc *PropertyCache) Get(id uint16, sub byte) (Property, bool) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	p, ok := pc.m[Key(id, sub)]
	if !ok {
		return Property{}, false
	}
	return *p, true
}

// Has ports ICULanDevice.HasProperty(id, subid): present AND Value != null.
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICULanDevice.cs:2814-2822
func (pc *PropertyCache) Has(id uint16, sub byte) bool {
	p, ok := pc.Get(id, sub)
	return ok && p.HasValue
}

// String ports ICUDevice.GetPropertyString(id, sub): the EDS option title
// when the value matches an option, else the value, else "".
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICUDevice.cs:92-113
func (pc *PropertyCache) String(id uint16, sub byte) string {
	p, ok := pc.Get(id, sub)
	if !ok || !p.HasValue {
		return ""
	}
	if e, ok := pc.Lookup(id, sub); ok {
		for _, o := range e.Options {
			if o.Value == p.Value {
				return o.Title
			}
		}
	}
	return p.Value
}

// Int ports ICUDevice.GetPropertyInt(id, sub, mask): Convert.ToInt32(Value)
// (0 on conversion failure/overflow), optionally masked.
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICUDevice.cs:142-162
func (pc *PropertyCache) Int(id uint16, sub byte, mask ...int) int {
	p, ok := pc.Get(id, sub)
	if !ok || !p.HasValue {
		return 0
	}
	n, ok := toInt64(&p)
	if !ok || n < math.MinInt32 || n > math.MaxInt32 {
		return 0
	}
	v := int(n)
	if len(mask) > 0 && mask[0] != 0 {
		v &= mask[0]
	}
	return v
}

// UInt ports ICUDevice.GetPropertyUInt(id, sub): Convert.ToUInt32(Value).
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICUDevice.cs:164-184
func (pc *PropertyCache) UInt(id uint16, sub byte) uint32 {
	p, ok := pc.Get(id, sub)
	if !ok || !p.HasValue {
		return 0
	}
	n, ok := toInt64(&p)
	if !ok || n < 0 || n > math.MaxUint32 {
		return 0
	}
	return uint32(n)
}

// Bool ports ICUDevice.GetPropertyBool(id, sub, default): Convert.ToBoolean
// (numbers: != 0; strings: "True"/"False"; anything else: the default).
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICUDevice.cs:186-201
func (pc *PropertyCache) Bool(id uint16, sub byte, def bool) bool {
	p, ok := pc.Get(id, sub)
	if !ok || !p.HasValue {
		return def
	}
	if b, ok := parseDotNetBool(p.Value); ok && (p.DataType == api.SDTBoolean || p.DataType == api.SDTVisibleString) {
		return b
	}
	switch p.DataType {
	case api.SDTVisibleString, api.SDTByteArray, api.SDTArray16, api.SDTLowLimit:
		return def
	}
	f, err := strconv.ParseFloat(p.Value, 64)
	if err != nil {
		return def
	}
	return f != 0
}

// toInt64 mirrors Convert.ToInt32/ToUInt32's source-type handling: integer
// values directly, reals rounded to even, booleans as 1/0, strings parsed.
func toInt64(p *Property) (int64, bool) {
	switch p.DataType {
	case api.SDTBoolean:
		if b, ok := parseDotNetBool(p.Value); ok {
			if b {
				return 1, true
			}
			return 0, true
		}
		n, err := strconv.ParseInt(p.Value, 10, 64) // (byte)0 fallback
		return n, err == nil
	case api.SDTReal32, api.SDTReal64:
		f, err := strconv.ParseFloat(p.Value, 64)
		if err != nil || math.IsNaN(f) || math.IsInf(f, 0) {
			return 0, false
		}
		r := math.RoundToEven(f)
		if r < math.MinInt64 || r > math.MaxInt64 {
			return 0, false
		}
		return int64(r), true
	case api.SDTByteArray, api.SDTArray16:
		return 0, false // InvalidCastException
	case api.SDTUnsigned64:
		u, err := strconv.ParseUint(p.Value, 10, 64)
		if err != nil || u > math.MaxInt64 {
			return 0, false
		}
		return int64(u), true
	default:
		return parseDotNetInt(p.Value, 64, true)
	}
}

// ErrUnsupportedType is returned by SetValue for data types ICUProperty.SetValue
// does not convert (the value stays unchanged in the installer).
var ErrUnsupportedType = errors.New("data type not supported by the installer")

// SetValue ports assigning ICUProperty.Value: convert text, then
// IsChanged = !Value.Equals(DeviceValue).
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICUProperty.cs:43-53,97-341
func (pc *PropertyCache) SetValue(id uint16, sub byte, text string) (Property, error) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	p, ok := pc.m[Key(id, sub)]
	if !ok {
		return Property{}, fmt.Errorf("property %s not read from the device", ODIndex(id, sub))
	}
	v, ok := ConvertValue(p.DataType, text)
	if !ok {
		return *p, ErrUnsupportedType
	}
	p.Value = v
	p.HasValue = true
	p.Changed = p.Value != p.DeviceValue
	return *p, nil
}

// Changed lists the properties with IsChanged set, in dictionary order
// (ICUDevice.StoreChangedProperties' PropertyDictionary.Where(a => a.IsChanged)).
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICUDevice.cs:291-299
func (pc *PropertyCache) Changed() []Property {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	var out []Property
	for _, k := range pc.order {
		if p := pc.m[k]; p.Changed {
			out = append(out, *p)
		}
	}
	return out
}

// HasChanges reports whether any property is changed (MainWindow
// CheckIfPropertiesHasChanged → Save/Revert sensitivity).
func (pc *PropertyCache) HasChanges() bool { return len(pc.Changed()) > 0 }

// Commit ports ICUProperty.CommitChange for the given keys: DeviceValue =
// Value, IsChanged = false.
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICUProperty.cs:393-398
func (pc *PropertyCache) Commit(keys ...uint32) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	for _, k := range keys {
		if p, ok := pc.m[k]; ok {
			p.DeviceValue = p.Value
			p.Changed = false
		}
	}
}

// Revert ports ICUDevice.RevertChanges: Rollback (Value = DeviceValue) for
// every changed property. With keys, only those properties are rolled back.
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICUDevice.cs:301-307, ICUProperty.cs:400-403
func (pc *PropertyCache) Revert(keys ...uint32) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	sel := map[uint32]bool{}
	for _, k := range keys {
		sel[k] = true
	}
	for _, k := range pc.order {
		p := pc.m[k]
		if !p.Changed || (len(keys) > 0 && !sel[k]) {
			continue
		}
		p.Value = p.DeviceValue
		p.Changed = false
	}
}

// All returns copies of every cached property in dictionary order.
func (pc *PropertyCache) All() []Property {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	out := make([]Property, 0, len(pc.order))
	for _, k := range pc.order {
		out = append(out, *pc.m[k])
	}
	return out
}

// Len returns the number of cached properties.
func (pc *PropertyCache) Len() int {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	return len(pc.order)
}

// Clear drops every property (a new ICUPropertyDictionary).
func (pc *PropertyCache) Clear() {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.m = map[uint32]*Property{}
	pc.order = nil
}

// ICUName ports how ICUProperty.ICUName is set: "OD_" + EDS ParameterName for
// properties the EDS pre-populates (ICUProperty.Initialize), otherwise the
// propName ParseProperty receives, which is the device's "id" string
// (ParseProperty(item2["id"].ToString(), …)). It is the JSON key
// StoreProperties writes.
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICUProperty.cs:348-360, ICULanDevice.cs:1834-1840
func (pc *PropertyCache) ICUName(p Property) string {
	if e, ok := pc.Lookup(p.ID, p.Sub); ok && e.Name != "" {
		return "OD_" + e.Name
	}
	return p.IDSub()
}

// Title ports ICUProperty.Title: the EDS Title, else the device "id" string.
func (pc *PropertyCache) Title(p Property) string {
	if e, ok := pc.Lookup(p.ID, p.Sub); ok && e.Title != "" {
		return e.Title
	}
	return p.IDSub()
}

// LabelText ports UIPropertyBase.LabelText for a property-bound control: EDS
// Title (with " (Units)"), else ICUProperty.Title (the m_fUnknownTitle path).
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/UIPropertyBase.cs:85-108
func (pc *PropertyCache) LabelText(p Property) string {
	if e, ok := pc.Lookup(p.ID, p.Sub); ok {
		if e.Units != "" {
			return fmt.Sprintf("%s (%s)", e.Title, e.Units)
		}
		return e.Title
	}
	return pc.Title(p)
}

// ParameterName returns EDSParameter.Name for search. When no EDS is wired
// the device-reported "name" stands in (plan risk note: "fall back to
// device-reported type/cat from /api/prop"); it is never used as a write key.
func (pc *PropertyCache) ParameterName(p Property) string {
	if e, ok := pc.Lookup(p.ID, p.Sub); ok {
		return e.Name
	}
	if pc.EDS() == nil {
		return p.Name
	}
	return ""
}

// WireProperty returns the api.Property StoreProperties should send for p:
// the current value under the ICUName key.
func (pc *PropertyCache) WireProperty(p Property) api.Property {
	w := p.Property
	w.Name = pc.ICUName(p)
	return w
}
