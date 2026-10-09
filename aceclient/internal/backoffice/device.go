package backoffice

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"alfen/aceclient/internal/api"
)

// Lookup ports ICUDevice.GetProperty(propId, subId) over the device's cached
// property dictionary (filled by api.Client.ReadProperties). ok=false means
// the device does not expose that object.
type Lookup func(id uint16, sub byte) (api.Property, bool)

// LookupFrom builds a Lookup over a property list (later duplicates win, as
// ParseProperty updates the existing ICUProperty).
func LookupFrom(props []api.Property) Lookup {
	m := make(map[[2]uint16]api.Property, len(props))
	for _, p := range props {
		m[[2]uint16{p.ID, uint16(p.Sub)}] = p
	}
	return func(id uint16, sub byte) (api.Property, bool) {
		p, ok := m[[2]uint16{id, uint16(sub)}]
		return p, ok
	}
}

// has ports ICULanDevice.HasProperty(id, subid).
func (l Lookup) has(id uint16, sub byte) bool {
	if l == nil {
		return false
	}
	_, ok := l(id, sub)
	return ok
}

// typedString returns ICUProperty.Value.ToString() for a device property
// (the raw wire value run through ICUProperty.SetValue).
func typedString(p api.Property) string { return coerceValue(p.DataType, p.Value, p.Value) }

// propInt ports ICUDevice.GetPropertyInt(propId, subId): Convert.ToInt32 of
// the typed value, 0 when missing or not convertible.
func (l Lookup) propInt(id uint16, sub byte) int {
	if l == nil {
		return 0
	}
	p, ok := l(id, sub)
	if !ok {
		return 0
	}
	s := typedString(p)
	switch p.DataType {
	case api.SDTBoolean:
		if s == "True" {
			return 1
		}
		if s == "False" {
			return 0
		}
	case api.SDTReal32, api.SDTReal64:
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return 0
		}
		n, err := roundToInt32(f)
		if err != nil {
			return 0
		}
		return n
	}
	n, err := csParseInt32(s)
	if err != nil {
		return 0
	}
	return n
}

// coerceValue ports ICUProperty.SetValue followed by the Value.ToString() that
// ICULanDevice.StoreProperties puts on the wire: text is newValue.ToString(),
// previous is the property's current value string (kept for data types the
// C# switch does not handle). Byte/word arrays come back as the
// comma-separated "X2"/"X4" hex StoreProperties emits.
func coerceValue(dt api.SDT, text, previous string) string {
	trimW := func(s string) string { return strings.TrimFunc(s, isNumWhite) }
	parseU := func(bits int) string {
		u, err := strconv.ParseUint(trimW(text), 10, bits)
		if err != nil {
			return "0"
		}
		return strconv.FormatUint(u, 10)
	}
	parseI := func(bits int) string {
		n, err := strconv.ParseInt(trimW(text), 10, bits)
		if err != nil {
			return "0"
		}
		return strconv.FormatInt(n, 10)
	}
	boolish := func() (string, bool) {
		switch {
		case strings.EqualFold(text, "false"):
			return "0", true
		case strings.EqualFold(text, "true"):
			return "1", true
		}
		return "", false
	}
	switch dt {
	case api.SDTVisibleString:
		return text
	case api.SDTUnsigned8:
		if s, ok := boolish(); ok {
			return s
		}
		return parseU(8)
	case api.SDTInteger8:
		if s, ok := boolish(); ok {
			return s
		}
		return parseI(8)
	case api.SDTUnsigned16:
		return parseU(16)
	case api.SDTUnsigned32:
		return parseU(32)
	case api.SDTUnsigned64:
		return parseU(64)
	case api.SDTInteger16:
		return parseI(16)
	case api.SDTInteger32:
		return parseI(32)
	case api.SDTInteger64:
		return parseI(64)
	case api.SDTReal32, api.SDTReal64:
		bits, prec := 64, 15
		if dt == api.SDTReal32 {
			bits, prec = 32, 7
		}
		f, err := strconv.ParseFloat(trimW(strings.ReplaceAll(text, ",", ".")), bits)
		if err != nil {
			f = 0
		}
		return formatG(f, prec, bits)
	case api.SDTBoolean:
		if b, err := csParseBool(text); err == nil {
			return csToString(b)
		}
		return "0" // (byte)0
	case api.SDTByteArray, api.SDTArray16:
		bits, width := 8, "%02X"
		if dt == api.SDTArray16 {
			bits, width = 16, "%04X"
		}
		parts := strings.Split(text, ",")
		out := make([]string, len(parts))
		for i, s := range parts {
			u, err := strconv.ParseUint(trimW(s), 16, bits)
			if err != nil {
				u = 0
			}
			out[i] = fmt.Sprintf(width, u)
		}
		return strings.Join(out, ",")
	case api.SDTLowLimit:
		return ""
	default: // unknown data type: logged, value unchanged
		return previous
	}
}

// formatG mirrors float.ToString()/double.ToString() under en-US ("G" with 7
// resp. 15 significant digits, "E+XX" exponents).
func formatG(f float64, prec, bits int) string {
	if f == 0 {
		return "0"
	}
	return strconv.FormatFloat(f, 'G', prec, bits)
}

// writeSet ports the PanelConnectivity.SetProperty(ref propList, propId,
// subId, newValue) pattern: a value is only set when the device exposes the
// property, and the last value set for a property wins.
type writeSet struct {
	lookup Lookup
	keys   [][2]uint16
	vals   map[[2]uint16]api.Property
}

func newWriteSet(l Lookup) *writeSet {
	return &writeSet{lookup: l, vals: map[[2]uint16]api.Property{}}
}

// set ports PanelConnectivity.SetProperty; value is the C# newValue.ToString().
func (w *writeSet) set(id uint16, sub byte, value string) {
	if w.lookup == nil {
		return
	}
	p, ok := w.lookup(id, sub)
	if !ok {
		return
	}
	k := [2]uint16{id, uint16(sub)}
	prev := p.Value
	if q, seen := w.vals[k]; seen {
		prev = q.Value
	} else {
		w.keys = append(w.keys, k)
	}
	p.Value = coerceValue(p.DataType, value, prev)
	w.vals[k] = p
}

// all returns every property set, in first-set order (the propList).
func (w *writeSet) all() []api.Property {
	out := make([]api.Property, 0, len(w.keys))
	for _, k := range w.keys {
		out = append(out, w.vals[k])
	}
	return out
}

// changed returns the properties whose new value differs from the device
// value (ICUProperty.IsChanged, which StoreChangedProperties filters on).
func (w *writeSet) changed() []api.Property {
	var out []api.Property
	for _, k := range w.keys {
		p := w.vals[k]
		dev, _ := w.lookup(uint16(k[0]), byte(k[1]))
		if p.Value != typedString(dev) {
			out = append(out, p)
		}
	}
	return out
}

// Version mirrors System.Version (Build/Revision are -1 when not specified).
type Version struct {
	Major, Minor, Build, Revision int
}

// V builds a Version like `new Version(major, minor[, build[, revision]])`.
func V(major, minor int, rest ...int) Version {
	v := Version{Major: major, Minor: minor, Build: -1, Revision: -1}
	if len(rest) > 0 {
		v.Build = rest[0]
	}
	if len(rest) > 1 {
		v.Revision = rest[1]
	}
	return v
}

// ParseVersion ports `new Version(string)`: 2..4 dot-separated non-negative
// Int32 components.
func ParseVersion(s string) (Version, error) {
	parts := strings.Split(s, ".")
	if len(parts) < 2 || len(parts) > 4 {
		return Version{}, errors.New("version string portion was too short or too long")
	}
	n := make([]int, len(parts))
	for i, p := range parts {
		x, err := csParseInt32(p)
		if err != nil || x < 0 {
			return Version{}, fmt.Errorf("invalid version component %q", p)
		}
		n[i] = x
	}
	return V(n[0], n[1], n[2:]...), nil
}

// FirmwareVersion ports ICULanDevice.FirmwareVersionNumber from the raw value
// of property 4106 (e.g. "6.6.2-4227"): without a '-' the device is assumed to
// run 4.10.0 over HTTPS, 4.9.0 over HTTP; "X" is read as "99"; an unparsable
// version becomes `new Version()` (0.0).
func FirmwareVersion(raw string, https bool) Version {
	parts := strings.Split(raw, "-")
	if len(parts) < 2 {
		if !https {
			return V(4, 9, 0)
		}
		return V(4, 10, 0)
	}
	v, err := ParseVersion(strings.ReplaceAll(parts[0], "X", "99"))
	if err != nil {
		return V(0, 0)
	}
	return v
}

// Compare ports Version.CompareTo (-1, 0, 1).
func (v Version) Compare(o Version) int {
	for _, d := range [][2]int{{v.Major, o.Major}, {v.Minor, o.Minor}, {v.Build, o.Build}, {v.Revision, o.Revision}} {
		if d[0] != d[1] {
			if d[0] > d[1] {
				return 1
			}
			return -1
		}
	}
	return 0
}

// AtLeast reports v >= o.
func (v Version) AtLeast(o Version) bool { return v.Compare(o) >= 0 }

// String ports Version.ToString().
func (v Version) String() string {
	s := fmt.Sprintf("%d.%d", v.Major, v.Minor)
	if v.Build >= 0 {
		s += fmt.Sprintf(".%d", v.Build)
		if v.Revision >= 0 {
			s += fmt.Sprintf(".%d", v.Revision)
		}
	}
	return s
}

// HasNetworkProfiles ports PanelConnectivity's
// `m_bHasNetworkProfiles = HasProperty(8432, 14)` (firmware with OCPP network
// profiles 8432..8435).
func HasNetworkProfiles(l Lookup) bool { return l.has(PropNetworkProfile, NPSubPriority) }

// HasExtendedFieldLengths ports PanelConnectivity's
// `m_hasExtendedFieldLengths = m_bHasNetworkProfiles && HasProperty(8432, 15)`.
func HasExtendedFieldLengths(l Lookup) bool {
	return HasNetworkProfiles(l) && l.has(PropNetworkProfile, NPSubAPNUsername)
}

// HasBackOfficeConfigured ports ICULanDevice.HasBackOfficeConfigured (and
// HasBackOfficeConfiguredInProfiles): with network profiles, any profile with
// a priority (sub 14) and an interface (sub 6); otherwise connect method 8311
// != 0.
func HasBackOfficeConfigured(l Lookup) bool {
	if l.has(PropNetworkProfile, NPSubPriority) {
		for i := 0; i < NetworkProfileCount; i++ {
			id := PropNetworkProfile + uint16(i)
			if l.propInt(id, NPSubPriority) > 0 && l.propInt(id, NPSubInterface) > 0 {
				return true
			}
		}
		return false
	}
	return l.has(PropConnectMethod, 0) && l.propInt(PropConnectMethod, 0) != 0
}
