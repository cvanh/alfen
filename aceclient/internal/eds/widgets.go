package eds

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"alfen/aceclient/internal/api"
)

// ---- UIPropertyNumber + Xwt.WPFBackend.WindowsSpinButton ----

// SpinConfig is the effective spin-button configuration: range, increment and
// decimal places.
type SpinConfig struct {
	Min, Max  float64
	Increment float64
	Digits    int
}

// DefaultSpin is a fresh Xwt WPF WindowsSpinButton (dependency-property
// defaults): Minimum 0, Maximum 1, Increment 0.1, DecimalPlaces 1.
var DefaultSpin = SpinConfig{Min: 0, Max: 1, Increment: 0.1, Digits: 1}

// NumberSpin ports UIPropertyNumber.OnRefreshDisplay for a control without a
// forced range (m_bForceMinMax == false): forceDigits is the constructor's
// digits (-1 = none; the All Properties panel passes 0, or 3 for REAL32/64),
// sdt the device data type and maxLength ICUProperty.MaxLength (the device
// "len").
//
// Per type the range is SetValueMinMax(min, max, 1, 0) for integers and
// (±3.4028234663852886E+38, 0.001, 3) for REAL32; REAL64 and other types keep
// the spin defaults. A non-zero maxLength then overrides Maximum — the C#
// does this for numbers too.
func NumberSpin(sdt api.SDT, forceDigits int, maxLength uint64) SpinConfig {
	c := DefaultSpin
	if forceDigits != -1 {
		c.Digits = forceDigits
	}
	set := func(min, max, step float64, digits int) { // SetValueMinMax(force: false)
		c.Min, c.Max = min, max
		if forceDigits >= 0 {
			c.Increment = 1.0 / math.Pow(10, float64(forceDigits))
			c.Digits = forceDigits
		} else {
			c.Digits = digits
			c.Increment = step
		}
	}
	switch sdt {
	case api.SDTUnsigned8:
		set(0, 255, 1, 0)
	case api.SDTUnsigned16:
		set(0, 65535, 1, 0)
	case api.SDTUnsigned32:
		set(0, 4294967295, 1, 0)
	case api.SDTUnsigned64:
		set(0, 1.8446744073709552e19, 1, 0)
	case api.SDTInteger8:
		set(-128, 127, 1, 0)
	case api.SDTInteger16:
		set(-32768, 32767, 1, 0)
	case api.SDTInteger32:
		set(-2147483648, 2147483647, 1, 0)
	case api.SDTInteger64:
		set(-9.223372036854776e18, 9.223372036854776e18, 1, 0)
	case api.SDTReal32:
		set(-3.4028234663852886e38, 3.4028234663852886e38, 0.001, 3)
	}
	if maxLength != 0 {
		c.Max = float64(maxLength)
	}
	return c
}

// SpinValue ports the value UIPropertyNumber.OnRefreshDisplay puts in the spin
// (Convert.ToDouble(Value) * factor, rounded by the WindowsSpinButton.Value
// setter to Digits with Math.Round — the setter does not clamp).
func SpinValue(v Value, factor float64, digits int) float64 {
	f, err := v.ToFloat64()
	if err != nil {
		f = 0
	}
	return netMathRound(f*factor, digits)
}

// FormatSpin ports WindowsSpinButton.UpdateTextbox: Value.ToString("N" +
// DecimalPlaces) with en-US and NumberGroupSizes = {0}, i.e. no thousands
// separators and "-" for negatives.
func FormatSpin(v float64, digits int) string { return netFormatFixed(v, digits) }

// ParseSpinText ports WindowsSpinButton.parseTextBox (Enter, Tab, focus loss):
// split the text on '.' and ',', keep "part0.part1", double.TryParse with
// NumberStyles.Number (invariant), clamp to [Min, Max] and round to Digits; an
// unparseable text yields Min.
func ParseSpinText(text string, c SpinConfig) float64 {
	split := splitAny(text, ".,")
	t := split[0]
	if len(split) > 1 {
		t = split[0] + "." + split[1]
	}
	if r, ok := netParseDouble(t, styleNumber); ok {
		if r > c.Max {
			r = c.Max
		} else if r < c.Min {
			r = c.Min
		}
		return netMathRound(r, c.Digits)
	}
	return netMathRound(c.Min, c.Digits)
}

// splitAny ports string.Split(char[]) (empty entries kept).
func splitAny(s, seps string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if strings.IndexByte(seps, s[i]) >= 0 {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}

// CommitNumber ports UIPropertyNumber.OnSpinValueChanged: the property value
// becomes spin / factor, boxed as double and re-parsed by ICUProperty.SetValue
// from its .NET Framework ToString() (G15) — so e.g. 12.5 on an UNSIGNED8
// becomes 0, exactly like the C#.
func CommitNumber(sdt api.SDT, spin, factor float64) Value {
	return ParseValue(sdt, netFormatG(spin/factor, 15))
}

// ---- UIPropertyString ----

// TextMaxLen ports UIPropertyString's m_nMaxLen: 256, the EDS MaxLength when
// the control has an EDS entry (param may be nil), 15 for the IP type, and —
// once a device value is bound — ICUProperty.MaxLength - 1 when that is > 1.
func TextMaxLen(param *Parameter, typ UIPropertyStringType, deviceMaxLength uint64) uint64 {
	n := DefaultMaxLength
	if param != nil {
		n = param.MaxLength
	}
	if typ == StringTypeIP {
		n = 15
	}
	if deviceMaxLength > 1 {
		n = deviceMaxLength - 1
	}
	return n
}

var reNotIPChar = regexp.MustCompile(`[^0-9.]`)

// FilterTextInput ports UIPropertyString.onTxtChanged: IP controls drop every
// character but digits and '.', then the text is cut to maxLen UTF-16 units.
// The result is both the new entry text and the value assigned to the
// property (when the control is writable).
func FilterTextInput(text string, typ UIPropertyStringType, maxLen uint64) string {
	if typ == StringTypeIP {
		text = reNotIPChar.ReplaceAllString(text, "")
	}
	if uint64(utf16Len(text)) > maxLen {
		text = utf16Prefix(text, int(maxLen))
	}
	return text
}

var reLeadingZeros = regexp.MustCompile(`0*([0-9]+)`)

// NormalizeIPText ports UIPropertyString.OnTxtLostFocus for IP controls:
// leading zeros are stripped from every number, then IPAddress.TryParse; a
// valid address is re-rendered dotted-quad, anything else becomes "0.0.0.0".
// Empty text is left as is.
func NormalizeIPText(text string) string {
	if text == "" {
		return text
	}
	if ip, ok := parseIPv4NonCanonical(reLeadingZeros.ReplaceAllString(text, "${1}")); ok {
		return ip
	}
	return "0.0.0.0"
}

// parseIPv4NonCanonical ports .NET Framework IPAddress.TryParse on a string of
// digits and dots (inet_aton rules: 1-4 decimal parts, the last part fills
// the remaining bytes).
func parseIPv4NonCanonical(s string) (string, bool) {
	parts := strings.Split(s, ".")
	if len(parts) < 1 || len(parts) > 4 {
		return "", false
	}
	vals := make([]uint64, len(parts))
	for i, p := range parts {
		if p == "" {
			return "", false
		}
		for j := 0; j < len(p); j++ {
			if p[j] < '0' || p[j] > '9' {
				return "", false
			}
		}
		v, err := strconv.ParseUint(p, 10, 32)
		if err != nil {
			return "", false
		}
		vals[i] = v
	}
	var addr uint64
	n := len(vals)
	for i := 0; i < n-1; i++ {
		if vals[i] > 255 {
			return "", false
		}
		addr |= vals[i] << (8 * uint(3-i))
	}
	last := vals[n-1]
	if last > (uint64(1)<<(8*uint(5-n)))-1 {
		return "", false
	}
	addr |= last
	return fmt.Sprintf("%d.%d.%d.%d", byte(addr>>24), byte(addr>>16), byte(addr>>8), byte(addr)), true
}

// TextDisplay ports UIPropertyString.OnRefreshDisplay for the String type:
// Value.ToString() trimmed of '\u0003' and '\u001e'.
func TextDisplay(v Value) string {
	return strings.Trim(v.String(), "\u0003\u001e")
}

// ---- UIPropertySelect ----

// SelectOptions ports UIPropertySelect.Initialize for EDS-backed selects: the
// EDS options (value, title) in file order, minus values in hide (the
// hideItems constructor); nil when the parameter has no options.
func SelectOptions(param *Parameter, hide map[string]bool) []Option {
	if param == nil || param.Options == nil {
		return nil
	}
	out := make([]Option, 0, len(param.Options))
	for _, o := range param.Options {
		if hide != nil && hide[o.Value] {
			continue
		}
		out = append(out, o)
	}
	return out
}

// SelectedOption ports UIPropertySelect.OnRefreshDisplay: the option the combo
// box shows for value v. With a mask the compared text is (int(v) & mask);
// ComboBox.SelectedItem = text selects the first option whose Value equals it
// ordinally; when not case-sensitive a lower-cased, trimmed match wins. ok is
// false when nothing is selected.
func SelectedOption(v Value, mask int, caseSensitive bool, options []Option) (Option, bool) {
	text := v.String()
	if mask != 0 {
		n, _ := v.ToInt32()
		text = strconv.Itoa(int(n) & mask)
	}
	idx := -1
	for i, o := range options {
		if o.Value == text {
			idx = i
			break
		}
	}
	if !caseSensitive {
		want := strings.TrimSpace(strings.ToLower(text))
		for i, o := range options {
			if strings.TrimSpace(strings.ToLower(o.Value)) == want {
				idx = i
				break
			}
		}
	}
	if idx < 0 {
		return Option{}, false
	}
	return options[idx], true
}

// SelectWriteText ports UIPropertySelect.GetValue: the selected option value,
// "0" when nothing is selected, or with a mask
// ((int(current) & ~mask) | int(selected)).ToString().
func SelectWriteText(current Value, selected string, hasSelection bool, mask int) string {
	if mask != 0 {
		cur, _ := current.ToInt32()
		sel, _ := netParseInt32(selected)
		return strconv.Itoa((int(cur) &^ mask) | int(sel))
	}
	if !hasSelection {
		return "0"
	}
	return selected
}

// CommitSelect ports UIPropertySelect.OnSelectionChanged: the value assigned
// to the property for a new selection; write is false when nothing is
// written (no selection, or a case-insensitive select whose text equals the
// current value ignoring case and surrounding white space).
func CommitSelect(sdt api.SDT, current Value, selected string, mask int, caseSensitive bool) (v Value, write bool) {
	text := SelectWriteText(current, selected, true, mask)
	if !caseSensitive && strings.TrimSpace(strings.ToLower(current.String())) == strings.TrimSpace(strings.ToLower(text)) {
		return current, false
	}
	return ParseValue(sdt, text), true
}

// ---- UIPropertyCheckbox ----

// CheckboxChecked ports UIPropertyCheckbox.OnRefreshDisplay: with a mask,
// (int(v) & mask) == mask; otherwise bool.TryParse(v.ToString()) == true, else
// Convert.ToInt32(v) != 0.
func CheckboxChecked(v Value, mask int) bool {
	if mask != 0 {
		n, _ := v.ToInt32()
		return int(n)&mask == mask
	}
	if b, ok := netParseBool(v.String()); ok && b {
		return true
	}
	n, err := v.ToInt32()
	return err == nil && n != 0
}

// CheckboxWriteText ports UIPropertyCheckbox.GetValue().ToString(): INTEGER8 /
// UNSIGNED8 write 1/0 (or the current value with the mask bits set/cleared),
// every other type writes the bool, i.e. "True"/"False".
func CheckboxWriteText(sdt api.SDT, current Value, checked bool, mask int) string {
	switch sdt {
	case api.SDTInteger8:
		if mask != 0 {
			b := int8(0)
			if n, err := current.ToInt32(); err == nil {
				b = int8(n)
			}
			if checked {
				return strconv.Itoa(int(b) | int(int8(mask)))
			}
			return strconv.Itoa(int(int8(int(b) &^ mask)))
		}
		if checked {
			return "1"
		}
		return "0"
	case api.SDTUnsigned8:
		if mask != 0 {
			b := uint8(0)
			if n, err := current.ToInt32(); err == nil {
				b = uint8(n)
			}
			if checked {
				return strconv.Itoa(int(b) | int(uint8(mask)))
			}
			return strconv.Itoa(int(uint8(int(b) &^ mask)))
		}
		if checked {
			return "1"
		}
		return "0"
	}
	if checked {
		return "True"
	}
	return "False"
}

// CommitCheckbox ports UIPropertyCheckbox.OnChkClicked: the value
// ICUProperty.SetValue makes of GetValue(). Note a checkbox on a 16/32-bit
// integer writes 0 either way ("True"/"False" do not parse), as in the C#.
func CommitCheckbox(sdt api.SDT, current Value, checked bool, mask int) Value {
	return ParseValue(sdt, CheckboxWriteText(sdt, current, checked, mask))
}

// CheckboxFromValue ports UIPropertyCheckbox.SetValue(newValue) (used when a
// panel pushes a value into the control): INTEGER8/UNSIGNED8 compare with the
// mask or with 1; other types accept "1"/"0" or a bool text.
func CheckboxFromValue(sdt api.SDT, text string, mask int) bool {
	switch sdt {
	case api.SDTInteger8:
		n, _ := netParseInt(text, 8)
		if mask != 0 {
			return int(int8(n))&mask == mask
		}
		return n == 1
	case api.SDTUnsigned8:
		n, _ := netParseUint(text, 8)
		if mask != 0 {
			return int(n)&mask == mask
		}
		return n == 1
	}
	switch strings.TrimSpace(text) {
	case "1":
		return true
	case "0":
		return false
	}
	b, _ := netParseBool(text)
	return b
}

// ---- UIPropertyReadOnlyString ----

// DeviceUTCOffsetMinutes ports the offset UIPropertyReadOnlyString /
// UIPropertyString compute for the DateTime type: sysTimeZoneMinutes
// (0x206E) when the device reported it, else sysTimeZone (0x205A) * 6, both
// via ICUDevice.GetPropertyInt (0 when absent or unconvertible).
func DeviceUTCOffsetMinutes(props []api.Property) int {
	get := func(id uint16) (Value, bool) {
		for i := len(props) - 1; i >= 0; i-- {
			if props[i].ID == id && props[i].Sub == 0 {
				return ValueOf(props[i]), true
			}
		}
		return Value{}, false
	}
	if v, ok := get(0x206E); ok && !v.IsNull() {
		n, _ := v.ToInt32()
		return int(n)
	}
	if v, ok := get(0x205A); ok {
		n, _ := v.ToInt32()
		return int(n) * 6
	}
	return 0
}

// windowsBaseOffsets are the distinct TimeZoneInfo.BaseUtcOffset values (in
// minutes) of the Windows time-zone database the C# searches with
// GetSystemTimeZones().FirstOrDefault(x => x.BaseUtcOffset.TotalMinutes == offset).
var windowsBaseOffsets = map[int]bool{
	-720: true, -660: true, -600: true, -570: true, -540: true, -480: true, -420: true,
	-360: true, -300: true, -240: true, -210: true, -180: true, -120: true, -60: true,
	0: true, 60: true, 120: true, 180: true, 210: true, 240: true, 270: true, 300: true,
	330: true, 345: true, 360: true, 390: true, 420: true, 480: true, 525: true, 540: true,
	570: true, 600: true, 630: true, 660: true, 720: true, 765: true, 780: true, 840: true,
}

// unixYear2000 is the threshold below which DateTime values are shown as a
// duration (946684800 = 2000-01-01T00:00:00Z).
const unixYear2000 = 946684800

// FormatDateTime ports the DateTime branch of UIPropertyReadOnlyString /
// UIPropertyString.OnRefreshDisplay: the value is milliseconds since the Unix
// epoch (ulong.TryParse, else 0). Below 2000-01-01 it is shown as a TimeSpan
// ("[d.]hh:mm:ss"); otherwise it is converted to the time zone whose base
// offset equals utcOffsetMinutes and shown as LongDate + " " + LongTime.
// ok is false when no zone has that offset (UIPropertyReadOnlyString leaves
// the text unchanged; UIPropertyString would throw NullReferenceException) or
// the date is beyond DateTime.MaxValue.
//
// Deviations (unverifiable without the Windows host): a fixed-offset zone is
// used (the C# applies the matched Windows zone's DST rules) and the invariant
// culture's patterns "dddd, dd MMMM yyyy" / "HH:mm:ss" (the C# uses the OS
// culture).
func FormatDateTime(v Value, utcOffsetMinutes int) (string, bool) {
	ms, ok := netParseUint(v.String(), 64)
	if !ok {
		ms = 0
	}
	sec := ms / 1000
	if sec < unixYear2000 {
		return formatTimeSpan(sec), true
	}
	if !windowsBaseOffsets[utcOffsetMinutes] {
		return "", false
	}
	const maxDateTimeSeconds = 253402300799 // 9999-12-31T23:59:59Z
	if sec > maxDateTimeSeconds {
		return "", false
	}
	t := time.Unix(int64(sec), 0).In(time.FixedZone("", utcOffsetMinutes*60))
	return t.Format("Monday, 02 January 2006") + " " + t.Format("15:04:05"), true
}

// formatTimeSpan ports TimeSpan.ToString() ("c") for whole seconds.
func formatTimeSpan(sec uint64) string {
	d := sec / 86400
	h := (sec / 3600) % 24
	m := (sec / 60) % 60
	s := sec % 60
	if d > 0 {
		return fmt.Sprintf("%d.%02d:%02d:%02d", d, h, m, s)
	}
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

// FormatReadOnly ports UIPropertyReadOnlyString.OnRefreshDisplay for the
// value-only types (invariant "." decimal separator; the C# uses the OS
// culture):
//
//	DateTime                  FormatDateTime
//	Float / Float2            v.ToString("F1"/"F2")
//	Float_0_001 .. Float2_10  (v * 0.001|0.01|0.1|10).ToString("F1"/"F2")
//	PWM                       "{v*0.01:F1} ({v*0.01*0.6:F1}A)"
//	PublicKey                 lower-cased, a space every 4 characters
//	String (default)          v.ToString()
//
// ok is false when the C# leaves the control text unchanged (empty value for
// the Float*/PWM types, FormatDateTime failure) and for ControllerVersion,
// PowerVersion, NFC*_SW/HW and Features, which need device-level helpers
// (ICULanDevice.GetHWVersion/GetNFCVersion, IWSFirmwareFeatures).
func FormatReadOnly(typ UIPropertyStringType, v Value, utcOffsetMinutes int) (string, bool) {
	scaled := func(factor float64, decimals int) (string, bool) {
		if v.String() == "" {
			return "", false
		}
		f, err := v.ToFloat64()
		if err != nil {
			return "", false
		}
		return netFormatFixed(f*factor, decimals), true
	}
	switch typ {
	case StringTypeDateTime:
		return FormatDateTime(v, utcOffsetMinutes)
	case StringTypeControllerVersion, StringTypePowerVersion, StringTypeNFC1HW,
		StringTypeNFC1SW, StringTypeNFC2HW, StringTypeNFC2SW, StringTypeFeatures:
		return "", false
	case StringTypeFloat:
		return scaled(1, 1)
	case StringTypeFloat2:
		return scaled(1, 2)
	case StringTypeFloat0_001:
		return scaled(0.001, 1)
	case StringTypeFloat2_0_001:
		return scaled(0.001, 2)
	case StringTypeFloat0_01:
		return scaled(0.01, 1)
	case StringTypeFloat2_0_01:
		return scaled(0.01, 2)
	case StringTypeFloat0_1:
		return scaled(0.1, 1)
	case StringTypeFloat2_0_1:
		return scaled(0.1, 2)
	case StringTypeFloat10:
		return scaled(10, 1)
	case StringTypeFloat2_10:
		return scaled(10, 2)
	case StringTypePWM:
		if v.String() == "" {
			return "", false
		}
		f, err := v.ToFloat64()
		if err != nil {
			return "", false
		}
		num := f * 0.01
		return fmt.Sprintf("%s (%sA)", netFormatFixed(num, 1), netFormatFixed(num*0.6, 1)), true
	case StringTypePublicKey:
		return FormatPublicKey(v.String()), true
	}
	return v.String(), true
}

// FormatPublicKey ports the PublicKey branch: lower-case, then a space before
// every 4th character (UTF-16 index; keys are ASCII hex).
func FormatPublicKey(s string) string {
	s = strings.ToLower(s)
	var sb strings.Builder
	for i, r := range []rune(s) {
		if i%4 == 0 && i != 0 {
			sb.WriteByte(' ')
		}
		sb.WriteRune(r)
	}
	return sb.String()
}

// ---- UIPropertyColor / UIPropertyColorHolder ----

// LED pattern spin range of UIPropertyColorHolder.InitializeSpinButton
// ("Period (ms)").
const (
	LEDTimeMin  = 0
	LEDTimeMax  = 2550
	LEDTimeStep = 10
)

// LEDPattern ports UIPropertyColor: a 10-byte LED state
// [R1 G1 B1 x T1 R2 G2 B2 y T2] with times in units of 10 ms; bytes 3 and 8
// are carried over unchanged (OriginalState).
type LEDPattern struct {
	Color1 [3]byte
	Time1  int // ms
	Color2 [3]byte
	Time2  int // ms
	// Original is the array the pattern was read from (OriginalState);
	// Encode copies bytes 3 and 8 from it.
	Original []byte
}

// ParseLEDPattern ports UIPropertyColor.InitializeFromArray: ok is false
// unless the array has more than 9 bytes.
func ParseLEDPattern(b []byte) (LEDPattern, bool) {
	if len(b) <= 9 {
		return LEDPattern{}, false
	}
	return LEDPattern{
		Color1:   [3]byte{b[0], b[1], b[2]},
		Time1:    int(b[4]) * 10,
		Color2:   [3]byte{b[5], b[6], b[7]},
		Time2:    int(b[9]) * 10,
		Original: append([]byte(nil), b...),
	}, true
}

// Bytes ports the array UIPropertyColor.SetValue(col1, time1, col2, time2) /
// GetValue builds: times are divided by 10 and truncated to a byte (unchecked
// cast, so 2560 ms wraps to 0). Bytes 3 and 8 come from Original (zero when it
// is shorter than 9 bytes; the C# would throw on a null OriginalState).
func (p LEDPattern) Bytes() []byte {
	var o3, o8 byte
	if len(p.Original) > 8 {
		o3, o8 = p.Original[3], p.Original[8]
	}
	return []byte{
		p.Color1[0], p.Color1[1], p.Color1[2], o3, byte(p.Time1 / 10),
		p.Color2[0], p.Color2[1], p.Color2[2], o8, byte(p.Time2 / 10),
	}
}

// Encode ports UIPropertyColor.GetValue: Bytes() as "X2" CSV.
func (p LEDPattern) Encode() string {
	return Value{Kind: ValueBytes, Bytes: p.Bytes()}.HexString()
}
