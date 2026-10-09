package api

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// ParseValue ports ICUProperty.SetValue(newValue) for a textual newValue
// (ACENetwork/ICUNetwork/ICUProperty.cs): the typed value the C# keeps in
// m_objValue for the given data type.
//
//	VISIBLE_STRING          string, verbatim
//	UNSIGNED8 / INTEGER8    uint8 / int8; "true"/"false" (any case) -> 1/0
//	UNSIGNED16/32/64        uint16/uint32/uint64 (digits, surrounding white space)
//	INTEGER16/32/64         int16/int32/int64 (optional sign)
//	REAL32 / REAL64         float32 / float64, ',' accepted as decimal point
//	BOOLEAN                 bool ("True"/"False", any case), else uint8(0)
//	BYTEARRAY / ARRAY_16    []byte / []uint16 from comma separated hex
//	LOW_LIMIT               "" (the C# also forces ReadOnly)
//
// Unparsable input yields the type's zero value, as in the C#. ok is false
// for the data types SetValue has no branch for (INTEGER24/40/48/56,
// UNSIGNED24/40/48/56, OCTET_STRING, UNICODE_STRING, TIME_OF_DAY,
// TIME_DIFFERENCE, DOMAIN, ...): the C# then leaves the value null, so
// such properties read as "" / 0 and cannot be stored.
func ParseValue(t SDT, text string) (v any, ok bool) {
	switch t {
	case SDTVisibleString:
		return text, true
	case SDTUnsigned8:
		if strings.EqualFold(text, "false") {
			return uint8(0), true
		}
		if strings.EqualFold(text, "true") {
			return uint8(1), true
		}
		n, _ := parseNetUint(text, 8)
		return uint8(n), true
	case SDTInteger8:
		if strings.EqualFold(text, "false") {
			return int8(0), true
		}
		if strings.EqualFold(text, "true") {
			return int8(1), true
		}
		n, _ := parseNetInt(text, 8)
		return int8(n), true
	case SDTUnsigned16:
		n, _ := parseNetUint(text, 16)
		return uint16(n), true
	case SDTUnsigned32:
		n, _ := parseNetUint(text, 32)
		return uint32(n), true
	case SDTUnsigned64:
		n, _ := parseNetUint(text, 64)
		return n, true
	case SDTInteger16:
		n, _ := parseNetInt(text, 16)
		return int16(n), true
	case SDTInteger32:
		n, _ := parseNetInt(text, 32)
		return int32(n), true
	case SDTInteger64:
		n, _ := parseNetInt(text, 64)
		return n, true
	case SDTReal32:
		f, ok := parseNetFloat(strings.ReplaceAll(text, ",", "."), 32, false)
		if !ok {
			return float32(0), true
		}
		return float32(f), true
	case SDTReal64:
		f, ok := parseNetFloat(strings.ReplaceAll(text, ",", "."), 64, false)
		if !ok {
			return float64(0), true
		}
		return f, true
	case SDTBoolean:
		if b, ok := parseNetBool(text); ok {
			return b, true
		}
		return uint8(0), true
	case SDTByteArray:
		parts := strings.Split(text, ",")
		out := make([]byte, len(parts))
		for i, s := range parts {
			n, _ := parseNetHex(s, 8)
			out[i] = byte(n)
		}
		return out, true
	case SDTArray16:
		parts := strings.Split(text, ",")
		out := make([]uint16, len(parts))
		for i, s := range parts {
			n, _ := parseNetHex(s, 16)
			out[i] = uint16(n)
		}
		return out, true
	case SDTLowLimit:
		return "", true
	}
	return nil, false
}

// ValueString ports object.ToString() on a value returned by ParseValue,
// i.e. ICUProperty.ToString(): integers in decimal, bool as "True"/"False",
// REAL32 with 7 and REAL64 with 15 significant digits (.NET Framework "G"),
// arrays as their .NET type name, nil as "". Numbers use the invariant '.'
// where the C# would use the current culture.
func ValueString(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case bool:
		if x {
			return "True"
		}
		return "False"
	case uint8:
		return strconv.FormatUint(uint64(x), 10)
	case int8:
		return strconv.FormatInt(int64(x), 10)
	case uint16:
		return strconv.FormatUint(uint64(x), 10)
	case int16:
		return strconv.FormatInt(int64(x), 10)
	case uint32:
		return strconv.FormatUint(uint64(x), 10)
	case int32:
		return strconv.FormatInt(int64(x), 10)
	case uint64:
		return strconv.FormatUint(x, 10)
	case int64:
		return strconv.FormatInt(x, 10)
	case float32:
		return formatNetG(float64(x), 7, 32)
	case float64:
		return formatNetG(x, 15, 64)
	case []byte:
		return "System.Byte[]"
	case []uint16:
		return "System.UInt16[]"
	}
	return fmt.Sprint(v)
}

// Typed returns the C# ICUProperty.Value for p: ParseValue(p.DataType, p.Value).
func (p Property) Typed() (any, bool) { return ParseValue(p.DataType, p.Value) }

// ToString ports ICUProperty.ToString() (and the fallback of
// ICUDevice.GetPropertyString when no EDS option title matches).
func (p Property) ToString() string {
	v, ok := p.Typed()
	if !ok {
		return ""
	}
	return ValueString(v)
}

// Int ports ICUDevice.GetPropertyInt: Convert.ToInt32(Value) & mask (mask 0
// = none); 0 when the value is missing or does not convert.
func (p Property) Int(mask int) int {
	v, ok := p.Typed()
	if !ok {
		return 0
	}
	n, err := netToInt32(v)
	if err != nil {
		return 0
	}
	if mask != 0 {
		n &= int32(mask)
	}
	return int(n)
}

// UInt ports ICUDevice.GetPropertyUInt: Convert.ToUInt32(Value) & mask.
func (p Property) UInt(mask uint32) uint32 {
	v, ok := p.Typed()
	if !ok {
		return 0
	}
	n, err := netToUInt32(v)
	if err != nil {
		return 0
	}
	if mask != 0 {
		n &= mask
	}
	return n
}

// Bool ports ICUDevice.GetPropertyBool: Convert.ToBoolean(Value), def when
// the value is missing or does not convert.
func (p Property) Bool(def bool) bool {
	v, ok := p.Typed()
	if !ok {
		return def
	}
	b, err := netToBoolean(v)
	if err != nil {
		return def
	}
	return b
}

// UInt64 ports ICUDevice.GetPropertyUInt64: Convert.ToUInt64(Value), def on
// failure.
func (p Property) UInt64(def uint64) uint64 {
	v, ok := p.Typed()
	if !ok {
		return def
	}
	n, err := netToUInt64(v)
	if err != nil {
		return def
	}
	return n
}

// Double ports ICUDevice.GetPropertyDouble: Convert.ToDouble(Value), 0 on
// failure.
func (p Property) Double() float64 {
	v, ok := p.Typed()
	if !ok {
		return 0
	}
	f, err := netToDouble(v)
	if err != nil {
		return 0
	}
	return f
}

// DeviceValueString ports ICUDevice.GetDeviceValueString without the EDS
// option lookup: REAL32/REAL64 as Convert.ToDouble(v).ToString("0.000"),
// everything else as ToString(). p.Value is taken as the device value.
func (p Property) DeviceValueString() string {
	v, ok := p.Typed()
	if !ok {
		return ""
	}
	if p.DataType == SDTReal32 || p.DataType == SDTReal64 {
		f, err := netToDouble(v)
		if err != nil {
			return ""
		}
		return formatNetFixed(f, 3)
	}
	return ValueString(v)
}

// ElementValue ports the Value attribute of ICUProperty.Element (the
// <Property Id=".." Value=".."/> line PropertyStorage writes): BYTEARRAY as
// "X2" CSV, ARRAY_16 as "X4" CSV, otherwise XAttribute's conversion (bool
// "true"/"false", floats via XmlConvert "R", others ToString). ok is false
// when the C# value is null (XAttribute would throw).
func (p Property) ElementValue() (string, bool) {
	v, ok := p.Typed()
	if !ok {
		return "", false
	}
	switch x := v.(type) {
	case []byte:
		return hexCSV(x), true
	case []uint16:
		return hexCSV16(x), true
	case string:
		return x, true
	case bool:
		if x {
			return "true", true
		}
		return "false", true
	case float32:
		return xmlConvertFloat(float64(x), 32), true
	case float64:
		return xmlConvertFloat(x, 64), true
	}
	return ValueString(v), true
}

// PaddedIDSub ports ICUProperty.ID_SUB: "{Id:X4}_{SubId:X2}" (e.g. "207D_02").
func (p Property) PaddedIDSub() string { return fmt.Sprintf("%04X_%02X", p.ID, p.Sub) }

// ODIndex ports ICUProperty.ODIndex: "{Id:X4}_{SubId:X}" (e.g. "207D_2").
func (p Property) ODIndex() string { return fmt.Sprintf("%04X_%X", p.ID, p.Sub) }

// storeValueText returns the "value" text StoreProperties writes for p
// (before quoting, which serializeProperty applies per type).
func storeValueText(p Property) (string, error) {
	v, ok := p.Typed()
	switch p.DataType {
	case SDTByteArray:
		return hexCSV(v.([]byte)), nil
	case SDTArray16:
		return hexCSV16(v.([]uint16)), nil
	case SDTReal32, SDTReal64:
		// Convert.ToString(Value, new CultureInfo("en-US"))
		return ValueString(v), nil
	}
	if !ok {
		// iCUProperty.Value.ToString() on a null value
		return "", fmt.Errorf("property %s (%s): value is null for data type %d", p.Name, p.IDSub(), p.DataType)
	}
	return ValueString(v), nil
}

func hexCSV(b []byte) string {
	parts := make([]string, len(b))
	for i, x := range b {
		parts[i] = fmt.Sprintf("%02X", x)
	}
	return strings.Join(parts, ",")
}

func hexCSV16(w []uint16) string {
	parts := make([]string, len(w))
	for i, x := range w {
		parts[i] = fmt.Sprintf("%04X", x)
	}
	return strings.Join(parts, ",")
}

// netEquals ports object.Equals between two boxed SetValue results, used for
// ICUProperty.IsChanged: arrays compare by reference (a freshly parsed array
// never equals the device value), floats treat NaN as equal to NaN.
func netEquals(a, b any) bool {
	switch x := a.(type) {
	case []byte, []uint16:
		return false
	case float32:
		y, ok := b.(float32)
		return ok && (x == y || (x != x && y != y))
	case float64:
		y, ok := b.(float64)
		return ok && (x == y || (math.IsNaN(x) && math.IsNaN(y)))
	}
	return a == b
}

// ---- System.Convert ports -------------------------------------------------

var errInvalidCast = errors.New("invalid cast")
var errOverflow = errors.New("overflow")

// netToInt32 ports Convert.ToInt32(object).
func netToInt32(v any) (int32, error) {
	switch x := v.(type) {
	case nil:
		return 0, nil
	case bool:
		if x {
			return 1, nil
		}
		return 0, nil
	case string:
		n, ok := parseNetInt(x, 32)
		if !ok {
			return 0, errInvalidCast
		}
		return int32(n), nil
	case float32:
		return doubleToInt32(float64(x))
	case float64:
		return doubleToInt32(x)
	}
	n, u, signed, err := netInteger(v)
	if err != nil {
		return 0, err
	}
	if signed {
		if n < math.MinInt32 || n > math.MaxInt32 {
			return 0, errOverflow
		}
		return int32(n), nil
	}
	if u > math.MaxInt32 {
		return 0, errOverflow
	}
	return int32(u), nil
}

// netToUInt32 ports Convert.ToUInt32(object).
func netToUInt32(v any) (uint32, error) {
	switch x := v.(type) {
	case nil:
		return 0, nil
	case bool:
		if x {
			return 1, nil
		}
		return 0, nil
	case string:
		n, ok := parseNetInt(x, 64)
		if !ok || n < 0 || n > math.MaxUint32 {
			return 0, errInvalidCast
		}
		return uint32(n), nil
	case float32:
		return doubleToUInt32(float64(x))
	case float64:
		return doubleToUInt32(x)
	}
	n, u, signed, err := netInteger(v)
	if err != nil {
		return 0, err
	}
	if signed {
		if n < 0 || n > math.MaxUint32 {
			return 0, errOverflow
		}
		return uint32(n), nil
	}
	if u > math.MaxUint32 {
		return 0, errOverflow
	}
	return uint32(u), nil
}

// netToUInt64 ports Convert.ToUInt64(object).
func netToUInt64(v any) (uint64, error) {
	switch x := v.(type) {
	case nil:
		return 0, nil
	case bool:
		if x {
			return 1, nil
		}
		return 0, nil
	case string:
		t := trimNetWhite(x)
		if strings.HasPrefix(t, "+") {
			t = t[1:]
		}
		if t == "-0" {
			return 0, nil
		}
		if !isASCIIDigits(t) {
			return 0, errInvalidCast
		}
		n, err := strconv.ParseUint(t, 10, 64)
		if err != nil {
			return 0, errOverflow
		}
		return n, nil
	case float32:
		return doubleToUInt64(float64(x))
	case float64:
		return doubleToUInt64(x)
	}
	n, u, signed, err := netInteger(v)
	if err != nil {
		return 0, err
	}
	if signed {
		if n < 0 {
			return 0, errOverflow
		}
		return uint64(n), nil
	}
	return u, nil
}

// netToDouble ports Convert.ToDouble(object).
func netToDouble(v any) (float64, error) {
	switch x := v.(type) {
	case nil:
		return 0, nil
	case bool:
		if x {
			return 1, nil
		}
		return 0, nil
	case string:
		f, ok := parseNetFloat(x, 64, true)
		if !ok {
			return 0, errInvalidCast
		}
		return f, nil
	case float32:
		return float64(x), nil
	case float64:
		return x, nil
	}
	n, u, signed, err := netInteger(v)
	if err != nil {
		return 0, err
	}
	if signed {
		return float64(n), nil
	}
	return float64(u), nil
}

// netToBoolean ports Convert.ToBoolean(object).
func netToBoolean(v any) (bool, error) {
	switch x := v.(type) {
	case nil:
		return false, nil
	case bool:
		return x, nil
	case string:
		b, ok := parseNetBool(x)
		if !ok {
			return false, errInvalidCast
		}
		return b, nil
	case float32:
		return x != 0, nil
	case float64:
		return x != 0, nil
	}
	n, u, signed, err := netInteger(v)
	if err != nil {
		return false, err
	}
	if signed {
		return n != 0, nil
	}
	return u != 0, nil
}

// netInteger widens the integral SetValue types.
func netInteger(v any) (n int64, u uint64, signed bool, err error) {
	switch x := v.(type) {
	case int8:
		return int64(x), 0, true, nil
	case int16:
		return int64(x), 0, true, nil
	case int32:
		return int64(x), 0, true, nil
	case int64:
		return x, 0, true, nil
	case int:
		return int64(x), 0, true, nil
	case uint8:
		return 0, uint64(x), false, nil
	case uint16:
		return 0, uint64(x), false, nil
	case uint32:
		return 0, uint64(x), false, nil
	case uint64:
		return 0, x, false, nil
	}
	return 0, 0, false, errInvalidCast
}

// doubleToInt32 ports Convert.ToInt32(double): banker's rounding, overflow
// outside [-2147483648.5, 2147483647.5) and for NaN.
func doubleToInt32(f float64) (int32, error) {
	if math.IsNaN(f) {
		return 0, errOverflow
	}
	r := math.RoundToEven(f)
	if r < math.MinInt32 || r > math.MaxInt32 {
		return 0, errOverflow
	}
	return int32(r), nil
}

// doubleToUInt32 ports Convert.ToUInt32(double).
func doubleToUInt32(f float64) (uint32, error) {
	if math.IsNaN(f) {
		return 0, errOverflow
	}
	r := math.RoundToEven(f)
	if r < 0 || r > math.MaxUint32 {
		return 0, errOverflow
	}
	return uint32(r), nil
}

// doubleToUInt64 ports Convert.ToUInt64(double): checked((ulong)Math.Round(v)).
func doubleToUInt64(f float64) (uint64, error) {
	if math.IsNaN(f) {
		return 0, errOverflow
	}
	r := math.RoundToEven(f)
	if r < 0 || r >= 18446744073709551616.0 {
		return 0, errOverflow
	}
	return uint64(r), nil
}

// ---- .NET Framework number parsing (invariant culture) --------------------

// trimNetWhite trims the white space .NET number parsing accepts
// (U+0009..U+000D, U+0020).
func trimNetWhite(s string) string { return strings.Trim(s, "\t\n\v\f\r ") }

func isASCIIDigits(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// parseNetUint ports {byte,ushort,uint,ulong}.TryParse with
// AllowLeadingWhite|AllowTrailingWhite (no sign).
func parseNetUint(s string, bits int) (uint64, bool) {
	t := trimNetWhite(s)
	if !isASCIIDigits(t) {
		return 0, false
	}
	n, err := strconv.ParseUint(t, 10, bits)
	if err != nil {
		return 0, false // strconv clamps on overflow; TryParse yields 0
	}
	return n, true
}

// parseNetInt ports {sbyte,short,int,long}.TryParse with NumberStyles.Integer
// (surrounding white space, optional leading '+' or '-').
func parseNetInt(s string, bits int) (int64, bool) {
	t := trimNetWhite(s)
	digits := t
	if strings.HasPrefix(digits, "+") || strings.HasPrefix(digits, "-") {
		digits = digits[1:]
	}
	if !isASCIIDigits(digits) {
		return 0, false
	}
	n, err := strconv.ParseInt(t, 10, bits)
	if err != nil {
		return 0, false
	}
	return n, true
}

// parseNetHex ports TryParse with NumberStyles.HexNumber (surrounding white
// space, hex digits without prefix).
func parseNetHex(s string, bits int) (uint64, bool) {
	t := trimNetWhite(s)
	if t == "" {
		return 0, false
	}
	for i := 0; i < len(t); i++ {
		c := t[i]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f' || c >= 'A' && c <= 'F') {
			return 0, false
		}
	}
	n, err := strconv.ParseUint(t, 16, bits)
	if err != nil {
		return 0, false
	}
	return n, true
}

// parseNetHexInt32 ports int.TryParse(s, NumberStyles.HexNumber, ...): up to
// 32 bits, values >= 0x80000000 wrap negative.
func parseNetHexInt32(s string) (int32, bool) {
	n, ok := parseNetHex(s, 32)
	return int32(uint32(n)), ok
}

// parseNetBool ports bool.TryParse: "True"/"False" in any case, retried after
// trimming white space and NUL characters.
func parseNetBool(s string) (bool, bool) {
	for i := 0; i < 2; i++ {
		switch {
		case strings.EqualFold(s, "True"):
			return true, true
		case strings.EqualFold(s, "False"):
			return false, true
		}
		s = strings.TrimFunc(s, func(r rune) bool { return unicode.IsSpace(r) || r == 0 })
	}
	return false, false
}

var (
	netFloatRe          = regexp.MustCompile(`^[+-]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)(?:[eE][+-]?[0-9]+)?$`)
	netFloatThousandsRe = regexp.MustCompile(`^[+-]?(?:[0-9][0-9,]*(?:\.[0-9]*)?|\.[0-9]+)(?:[eE][+-]?[0-9]+)?$`)
)

// parseNetFloat ports float/double.TryParse with NumberStyles.Float and the
// invariant culture as implemented by .NET Framework: overflow fails (no
// infinity), REAL32 is parsed as double then narrowed, and the literal
// symbols "Infinity", "-Infinity" and "NaN" are accepted. thousands adds
// AllowThousands (Convert.ToDouble(string)).
func parseNetFloat(s string, bits int, thousands bool) (float64, bool) {
	t := trimNetWhite(s)
	re := netFloatRe
	if thousands {
		re = netFloatThousandsRe
	}
	if re.MatchString(t) {
		if thousands {
			t = strings.ReplaceAll(t, ",", "")
		}
		f, err := strconv.ParseFloat(t, 64)
		if err == nil || (errors.Is(err, strconv.ErrRange) && !math.IsInf(f, 0)) {
			if bits == 32 {
				f32 := float32(f)
				if math.IsInf(float64(f32), 0) {
					return 0, false
				}
				return float64(f32), true
			}
			return f, true
		}
		return 0, false
	}
	switch strings.TrimFunc(s, unicode.IsSpace) {
	case "Infinity":
		return math.Inf(1), true
	case "-Infinity":
		return math.Inf(-1), true
	case "NaN":
		return math.NaN(), true
	}
	return 0, false
}

// ---- .NET Framework number formatting (invariant culture) -----------------

// formatNetG ports double/float.ToString() ("G", 15 resp. 7 significant
// digits on .NET Framework): exponent as "E+XX", negative zero as "0".
func formatNetG(f float64, prec, bitSize int) string {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	case f == 0:
		return "0"
	}
	return strings.Replace(strconv.FormatFloat(f, 'g', prec, bitSize), "e", "E", 1)
}

// formatNetR ports ToString("R") on .NET Framework: "G" with 15 (7) digits
// when that round-trips, else 17 (9).
func formatNetR(f float64, bitSize int) string {
	p1, p2 := 15, 17
	if bitSize == 32 {
		p1, p2 = 7, 9
	}
	s := formatNetG(f, p1, bitSize)
	if math.IsNaN(f) || math.IsInf(f, 0) || f == 0 {
		return s
	}
	back, err := strconv.ParseFloat(strings.Replace(s, "E", "e", 1), bitSize)
	if err == nil && back == f {
		return s
	}
	return formatNetG(f, p2, bitSize)
}

// xmlConvertFloat ports XmlConvert.ToString(double/float).
func xmlConvertFloat(f float64, bitSize int) string {
	switch {
	case math.IsInf(f, -1):
		return "-INF"
	case math.IsInf(f, 1):
		return "INF"
	case f == 0 && math.Signbit(f):
		return "-0"
	}
	return formatNetR(f, bitSize)
}

// formatNetFixed ports ToString("F<n>") / ToString("0.000") on .NET
// Framework: the double is first reduced to 15 significant digits, then
// rounded half away from zero at n decimals; a result of zero drops its sign.
func formatNetFixed(f float64, decimals int) string {
	switch {
	case math.IsNaN(f):
		return "NaN"
	case math.IsInf(f, 1):
		return "Infinity"
	case math.IsInf(f, -1):
		return "-Infinity"
	}
	neg := f < 0
	e := strconv.FormatFloat(math.Abs(f), 'e', 14, 64)
	mant, expStr, _ := strings.Cut(e, "e")
	exp, _ := strconv.Atoi(expStr)
	digits := []byte(strings.Replace(mant, ".", "", 1))
	scale := exp + 1
	if f == 0 {
		scale = 0
	}
	pos := scale + decimals
	var kept []byte
	switch {
	case pos < 0:
		kept = nil
	case pos >= len(digits):
		kept = digits
	default:
		kept = append([]byte(nil), digits[:pos]...)
		if digits[pos] >= '5' {
			i := len(kept) - 1
			for i >= 0 && kept[i] == '9' {
				kept[i] = '0'
				i--
			}
			if i < 0 {
				kept = append([]byte{'1'}, kept...)
				scale++
			} else {
				kept[i]++
			}
		}
	}
	zero := true
	for _, d := range kept {
		if d != '0' {
			zero = false
			break
		}
	}
	digitAt := func(i int) byte {
		if i >= 0 && i < len(kept) {
			return kept[i]
		}
		return '0'
	}
	var b strings.Builder
	if neg && !zero {
		b.WriteByte('-')
	}
	if scale <= 0 {
		b.WriteByte('0')
	} else {
		for i := 0; i < scale; i++ {
			b.WriteByte(digitAt(i))
		}
	}
	if decimals > 0 {
		b.WriteByte('.')
		for j := 0; j < decimals; j++ {
			b.WriteByte(digitAt(scale + j))
		}
	}
	return b.String()
}
