package presets

import (
	"math"
	"strconv"
	"strings"
	"unicode"

	"alfen/aceclient/internal/api"
)

// valueKind is the CLR type boxed in ICUProperty.m_objValue.
type valueKind int

const (
	kindString  valueKind = iota // string (VISIBLE_STRING, LOW_LIMIT)
	kindByte                     // byte (UNSIGNED8, BOOLEAN parse failure)
	kindSByte                    // sbyte (INTEGER8)
	kindUInt16                   // ushort (UNSIGNED16)
	kindInt16                    // short (INTEGER16)
	kindUInt32                   // uint (UNSIGNED32)
	kindInt32                    // int (INTEGER32)
	kindUInt64                   // ulong (UNSIGNED64)
	kindInt64                    // long (INTEGER64)
	kindFloat32                  // float (REAL32)
	kindFloat64                  // double (REAL64)
	kindBool                     // bool (BOOLEAN)
	kindBytes                    // byte[] (BYTEARRAY)
	kindWords                    // ushort[] (ARRAY_16)
)

// propValue is a boxed ICUProperty value. A nil *propValue is the C# null.
// Arrays compare by reference in the C# (object.Equals on byte[]/ushort[]),
// which is modelled by pointer identity of the *propValue holding them.
type propValue struct {
	kind  valueKind
	s     string
	u     uint64  // unsigned kinds and kindByte
	i     int64   // signed kinds
	f     float64 // kindFloat64, or a float32 widened exactly
	b     bool
	bytes []byte
	words []uint16
}

// equals ports object.Equals between the boxed values (m_objValue.Equals(DeviceValue)).
func (v *propValue) equals(o *propValue) bool {
	if v == nil || o == nil {
		return v == o
	}
	if v.kind != o.kind {
		return false // different CLR types never compare equal
	}
	switch v.kind {
	case kindString:
		return v.s == o.s
	case kindByte, kindUInt16, kindUInt32, kindUInt64:
		return v.u == o.u
	case kindSByte, kindInt16, kindInt32, kindInt64:
		return v.i == o.i
	case kindFloat32, kindFloat64:
		// Single/Double.Equals: == or both NaN
		return v.f == o.f || (math.IsNaN(v.f) && math.IsNaN(o.f))
	case kindBool:
		return v.b == o.b
	default: // arrays: reference equality
		return v == o
	}
}

// parseResult is the outcome of ICUProperty.SetValue for a text value.
// keep == true means the default branch: the data type is not handled and
// m_objValue is left unchanged.
type parseResult struct {
	v        *propValue
	keep     bool
	readOnly bool // LOW_LIMIT sets ReadOnly = true
}

// parseValue ports the switch in ICUProperty.SetValue for a string newValue
// (ACENetwork/ICUNetwork/ICUProperty.cs), with .NET Framework parsing rules:
// failed parses fall back to zero, BOOLEAN falls back to (byte)0.
func parseValue(dt api.SDT, text string) parseResult {
	switch dt {
	case api.SDTVisibleString:
		return parseResult{v: &propValue{kind: kindString, s: text}}
	case api.SDTUnsigned8:
		v := &propValue{kind: kindByte}
		switch {
		case strings.EqualFold(text, "false"):
		case strings.EqualFold(text, "true"):
			v.u = 1
		default:
			if n, ok := netParseUnsigned(text, math.MaxUint8); ok {
				v.u = n
			}
		}
		return parseResult{v: v}
	case api.SDTInteger8:
		v := &propValue{kind: kindSByte}
		switch {
		case strings.EqualFold(text, "false"):
		case strings.EqualFold(text, "true"):
			v.i = 1
		default:
			if n, ok := netParseSigned(text, math.MinInt8, math.MaxInt8); ok {
				v.i = n
			}
		}
		return parseResult{v: v}
	case api.SDTUnsigned16:
		n, _ := netParseUnsigned(text, math.MaxUint16)
		return parseResult{v: &propValue{kind: kindUInt16, u: n}}
	case api.SDTUnsigned32:
		n, _ := netParseUnsigned(text, math.MaxUint32)
		return parseResult{v: &propValue{kind: kindUInt32, u: n}}
	case api.SDTUnsigned64:
		n, _ := netParseUnsigned(text, math.MaxUint64)
		return parseResult{v: &propValue{kind: kindUInt64, u: n}}
	case api.SDTInteger16:
		n, _ := netParseSigned(text, math.MinInt16, math.MaxInt16)
		return parseResult{v: &propValue{kind: kindInt16, i: n}}
	case api.SDTInteger32:
		n, _ := netParseSigned(text, math.MinInt32, math.MaxInt32)
		return parseResult{v: &propValue{kind: kindInt32, i: n}}
	case api.SDTInteger64:
		n, _ := netParseSigned(text, math.MinInt64, math.MaxInt64)
		return parseResult{v: &propValue{kind: kindInt64, i: n}}
	case api.SDTReal32:
		f, _ := netParseSingle(strings.ReplaceAll(text, ",", "."))
		return parseResult{v: &propValue{kind: kindFloat32, f: float64(f)}}
	case api.SDTReal64:
		f, _ := netParseDouble(strings.ReplaceAll(text, ",", "."))
		return parseResult{v: &propValue{kind: kindFloat64, f: f}}
	case api.SDTBoolean:
		if b, ok := netParseBool(text); ok {
			return parseResult{v: &propValue{kind: kindBool, b: b}}
		}
		return parseResult{v: &propValue{kind: kindByte}}
	case api.SDTByteArray:
		parts := strings.Split(text, ",")
		out := make([]byte, len(parts))
		for i, p := range parts {
			if n, ok := netParseHex(p, math.MaxUint8); ok {
				out[i] = byte(n)
			}
		}
		return parseResult{v: &propValue{kind: kindBytes, bytes: out}}
	case api.SDTArray16:
		parts := strings.Split(text, ",")
		out := make([]uint16, len(parts))
		for i, p := range parts {
			if n, ok := netParseHex(p, math.MaxUint16); ok {
				out[i] = uint16(n)
			}
		}
		return parseResult{v: &propValue{kind: kindWords, words: out}}
	case api.SDTLowLimit:
		return parseResult{v: &propValue{kind: kindString}, readOnly: true}
	default:
		// "SDT {Name} has unknown datatype" — value left unchanged
		return parseResult{keep: true}
	}
}

// toString ports object.ToString() on the boxed value (current culture is
// assumed to use '-' and '.' like the invariant culture; SetValue maps ',' to
// '.' before parsing reals anyway). Floats use the .NET Framework default "G"
// format (7 digits for float, 15 for double).
func (v *propValue) toString() string {
	switch v.kind {
	case kindString:
		return v.s
	case kindByte, kindUInt16, kindUInt32, kindUInt64:
		return strconv.FormatUint(v.u, 10)
	case kindSByte, kindInt16, kindInt32, kindInt64:
		return strconv.FormatInt(v.i, 10)
	case kindFloat32:
		return netFormatG(v.f, 7, "Infinity", "-Infinity")
	case kindFloat64:
		return netFormatG(v.f, 15, "Infinity", "-Infinity")
	case kindBool:
		if v.b {
			return "True"
		}
		return "False"
	case kindBytes:
		return "System.Byte[]"
	default:
		return "System.UInt16[]"
	}
}

// joinHex ports string.Join(",", array.Select(a => a.ToString("X2"/"X4"))).
func (v *propValue) joinHex() string {
	var sb strings.Builder
	switch v.kind {
	case kindBytes:
		for i, b := range v.bytes {
			if i > 0 {
				sb.WriteByte(',')
			}
			sb.WriteString(strings.ToUpper(strconv.FormatUint(uint64(b)|0x100, 16)[1:]))
		}
	case kindWords:
		for i, w := range v.words {
			if i > 0 {
				sb.WriteByte(',')
			}
			sb.WriteString(strings.ToUpper(strconv.FormatUint(uint64(w)|0x10000, 16)[1:]))
		}
	}
	return sb.String()
}

// elementString is the "Value" attribute text of ICUProperty.Element:
// arrays joined as hex, everything else through XAttribute's conversion
// (XContainer.GetStringValue: XmlConvert for float/double/bool, ToString
// otherwise).
func (v *propValue) elementString() string {
	switch v.kind {
	case kindBytes, kindWords:
		return v.joinHex()
	case kindFloat32:
		return xmlConvertFloat(v.f, 7, 9, true)
	case kindFloat64:
		return xmlConvertFloat(v.f, 15, 17, false)
	case kindBool:
		if v.b {
			return "true"
		}
		return "false"
	default:
		return v.toString()
	}
}

// storeString is the JSON value text ICULanDevice.StoreProperties writes for
// the boxed value: arrays as hex lists, REAL32/REAL64 via
// Convert.ToString(value, en-US) (default "G"), everything else
// Value.ToString() (so BOOLEAN sends "True"/"False").
func (v *propValue) storeString() string {
	switch v.kind {
	case kindBytes, kindWords:
		return v.joinHex()
	default:
		return v.toString()
	}
}

// ---- .NET Framework number parsing (System.Number.ParseNumber) ----

// isNetWhite is Number.IsWhite: U+0020 and U+0009..U+000D.
func isNetWhite(c byte) bool { return c == ' ' || (c >= 0x09 && c <= 0x0D) }

// trailingOK ports TryStringToNumber's acceptance of trailing NUL characters.
func trailingOK(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] != 0 {
			return false
		}
	}
	return true
}

// netParseDigits parses [white][sign]digits[white] (sign only if allowSign)
// into a magnitude. ok is false for a syntax error or uint64 overflow.
func netParseDigits(s string, allowSign bool) (mag uint64, neg, ok bool) {
	i := 0
	for i < len(s) && isNetWhite(s[i]) {
		i++
	}
	if allowSign && i < len(s) && (s[i] == '+' || s[i] == '-') {
		neg = s[i] == '-'
		i++
	}
	start := i
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		d := uint64(s[i] - '0')
		if mag > (math.MaxUint64-d)/10 {
			return 0, false, false
		}
		mag = mag*10 + d
		i++
	}
	if i == start {
		return 0, false, false
	}
	for i < len(s) && isNetWhite(s[i]) {
		i++
	}
	if !trailingOK(s[i:]) {
		return 0, false, false
	}
	return mag, neg, true
}

// netParseUnsigned ports byte/ushort/uint/ulong.TryParse with
// AllowLeadingWhite | AllowTrailingWhite (no sign) and the type's range.
func netParseUnsigned(s string, max uint64) (uint64, bool) {
	mag, _, ok := netParseDigits(s, false)
	if !ok || mag > max {
		return 0, false
	}
	return mag, true
}

// netParseSigned ports sbyte/short/int/long.TryParse with NumberStyles.Integer.
func netParseSigned(s string, min, max int64) (int64, bool) {
	mag, neg, ok := netParseDigits(s, true)
	if !ok {
		return 0, false
	}
	if neg {
		if mag > uint64(-(min+1))+1 {
			return 0, false
		}
		if mag == 0 {
			return 0, true
		}
		return -int64(mag-1) - 1, true
	}
	if mag > uint64(max) {
		return 0, false
	}
	return int64(mag), true
}

// netParseHex ports the NumberStyles.HexNumber parse of byte/ushort/int
// TryParse: white space around hex digits (no "0x"), parsed as a 32-bit
// value; the result must lie in [0, max] (int.TryParse itself accepts any
// 32-bit pattern, which callers reinterpret as signed).
func netParseHex(s string, max uint64) (uint64, bool) {
	i := 0
	for i < len(s) && isNetWhite(s[i]) {
		i++
	}
	start := i
	var v uint64
	for i < len(s) {
		c := s[i]
		var d uint64
		switch {
		case c >= '0' && c <= '9':
			d = uint64(c - '0')
		case c >= 'a' && c <= 'f':
			d = uint64(c-'a') + 10
		case c >= 'A' && c <= 'F':
			d = uint64(c-'A') + 10
		default:
			goto done
		}
		v = v<<4 | d
		if v > math.MaxUint32 {
			return 0, false
		}
		i++
	}
done:
	if i == start {
		return 0, false
	}
	for i < len(s) && isNetWhite(s[i]) {
		i++
	}
	if !trailingOK(s[i:]) || v > max {
		return 0, false
	}
	return v, true
}

// netParseFloatSyntax validates NumberStyles.Float syntax ([white][sign]
// digits[.digits][(e|E)[sign]digits][white]) and returns a strconv-ready
// string. An exponent without digits is not consumed (and then fails as
// trailing garbage), like ParseNumber.
func netParseFloatSyntax(s string) (string, bool) {
	i := 0
	for i < len(s) && isNetWhite(s[i]) {
		i++
	}
	var sb strings.Builder
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		if s[i] == '-' {
			sb.WriteByte('-')
		}
		i++
	}
	digits := 0
	for i < len(s) && s[i] >= '0' && s[i] <= '9' {
		sb.WriteByte(s[i])
		i++
		digits++
	}
	if i < len(s) && s[i] == '.' {
		sb.WriteByte('.')
		i++
		for i < len(s) && s[i] >= '0' && s[i] <= '9' {
			sb.WriteByte(s[i])
			i++
			digits++
		}
	}
	if digits == 0 {
		return "", false
	}
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		j := i + 1
		negExp := false
		if j < len(s) && (s[j] == '+' || s[j] == '-') {
			negExp = s[j] == '-'
			j++
		}
		if j < len(s) && s[j] >= '0' && s[j] <= '9' {
			exp := 0
			for j < len(s) && s[j] >= '0' && s[j] <= '9' {
				exp = exp*10 + int(s[j]-'0')
				j++
				if exp > 1000 { // ParseNumber clamps and skips the rest
					exp = 9999
					for j < len(s) && s[j] >= '0' && s[j] <= '9' {
						j++
					}
				}
			}
			if negExp {
				exp = -exp
			}
			sb.WriteString("e" + strconv.Itoa(exp))
			i = j
		}
	}
	for i < len(s) && isNetWhite(s[i]) {
		i++
	}
	if !trailingOK(s[i:]) {
		return "", false
	}
	return sb.String(), true
}

// netParseDouble ports double.TryParse(s, NumberStyles.Float, InvariantCulture)
// on .NET Framework: overflow fails, -0 becomes +0, and the invariant symbols
// "Infinity", "-Infinity" and "NaN" (after Trim) are accepted.
func netParseDouble(s string) (float64, bool) {
	if clean, ok := netParseFloatSyntax(s); ok {
		d, err := strconv.ParseFloat(clean, 64)
		if err == nil || !math.IsInf(d, 0) {
			if d == 0 {
				d = 0 // NumberBufferToDouble drops the sign of zero
			}
			return d, true
		}
	}
	return netParseFloatSymbols(s)
}

// netParseSingle ports float.TryParse (Framework): parsed as double, cast to
// float; a cast that overflows to infinity fails.
func netParseSingle(s string) (float32, bool) {
	if clean, ok := netParseFloatSyntax(s); ok {
		d, err := strconv.ParseFloat(clean, 64)
		if err == nil || !math.IsInf(d, 0) {
			f := float32(d)
			if !math.IsInf(float64(f), 0) {
				if f == 0 {
					f = 0
				}
				return f, true
			}
		}
	}
	d, ok := netParseFloatSymbols(s)
	return float32(d), ok
}

func netParseFloatSymbols(s string) (float64, bool) {
	switch strings.TrimSpace(s) {
	case "Infinity":
		return math.Inf(1), true
	case "-Infinity":
		return math.Inf(-1), true
	case "NaN":
		return math.NaN(), true
	}
	return 0, false
}

// netParseBool ports bool.TryParse: "True"/"False" ignoring case, after
// trimming white space and NUL characters.
func netParseBool(s string) (bool, bool) {
	t := strings.TrimFunc(s, func(r rune) bool { return r == 0 || unicode.IsSpace(r) })
	switch {
	case strings.EqualFold(t, "true"):
		return true, true
	case strings.EqualFold(t, "false"):
		return false, true
	}
	return false, false
}

// ---- .NET Framework number formatting ----

// netDigits returns the decimal digits (trailing zeros removed) and the
// decimal exponent "scale" (value = 0.d1d2... * 10^scale) of |f| rounded to
// prec significant digits, like DoubleToNumber + RoundNumber. Zero yields
// ("", 0).
func netDigits(f float64, prec int) (string, int) {
	if f == 0 {
		return "", 0
	}
	s := strconv.FormatFloat(math.Abs(f), 'e', prec-1, 64) // d.ddde±XX
	mant, expPart, _ := strings.Cut(s, "e")
	exp, _ := strconv.Atoi(expPart)
	digits := strings.TrimRight(strings.Replace(mant, ".", "", 1), "0")
	return digits, exp + 1
}

// netFormatGeneral ports NumberToString 'G' / FormatGeneral: scientific when
// scale > maxDigits or scale < -3, exponent as E+dd / E-dd (at least two
// digits).
func netFormatGeneral(neg bool, digits string, scale, maxDigits int) string {
	var sb strings.Builder
	if neg && digits != "" {
		sb.WriteByte('-')
	}
	digPos := scale
	scientific := false
	if digPos > maxDigits || digPos < -3 {
		digPos = 1
		scientific = true
	}
	d := 0
	if digPos > 0 {
		for ; digPos > 0; digPos-- {
			if d < len(digits) {
				sb.WriteByte(digits[d])
				d++
			} else {
				sb.WriteByte('0')
			}
		}
	} else {
		sb.WriteByte('0')
	}
	if d < len(digits) || digPos < 0 {
		sb.WriteByte('.')
		for ; digPos < 0; digPos++ {
			sb.WriteByte('0')
		}
		sb.WriteString(digits[d:])
	}
	if scientific {
		e := scale - 1
		sb.WriteByte('E')
		if e < 0 {
			sb.WriteByte('-')
			e = -e
		} else {
			sb.WriteByte('+')
		}
		if e < 10 {
			sb.WriteByte('0')
		}
		sb.WriteString(strconv.Itoa(e))
	}
	return sb.String()
}

// netFormatG ports float/double.ToString() on .NET Framework ("G" with 7 or
// 15 digits); -0 prints as "0".
func netFormatG(f float64, prec int, posInf, negInf string) string {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return posInf
	case math.IsInf(f, -1):
		return negInf
	}
	digits, scale := netDigits(f, prec)
	return netFormatGeneral(f < 0, digits, scale, prec)
}

// netFormatR ports the .NET Framework "R" format: format with prec digits;
// if that does not parse back to the same value use fallback digits.
// single selects the float round-trip test ((float)double).
func netFormatR(f float64, prec, fallback int, single bool) string {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	}
	digits, scale := netDigits(f, prec)
	back, _ := strconv.ParseFloat(netFormatGeneral(f < 0, digits, scale, prec), 64)
	if single {
		back = float64(float32(back))
	}
	if back != f {
		prec = fallback
		digits, scale = netDigits(f, prec)
	}
	return netFormatGeneral(f < 0, digits, scale, prec)
}

// xmlConvertFloat ports XmlConvert.ToString(float|double) (.NET Framework):
// "-INF", "INF", "-0" for negative zero, otherwise ToString("R").
func xmlConvertFloat(f float64, prec, fallback int, single bool) string {
	switch {
	case math.IsInf(f, -1):
		return "-INF"
	case math.IsInf(f, 1):
		return "INF"
	case f == 0 && math.Signbit(f):
		return "-0"
	}
	return netFormatR(f, prec, fallback, single)
}
