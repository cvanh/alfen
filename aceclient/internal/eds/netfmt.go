package eds

import (
	"math"
	"strconv"
	"strings"
	"unicode"
)

// .NET Framework number parsing/formatting semantics the ported C# relies on
// (ICUProperty.SetValue's TryParse calls, double/float.ToString(), "F"/"N"
// formats, Math.Round and Convert.ToInt32). The installer targets .NET
// Framework 4.8, whose double.ToString() is "G15" and float.ToString() "G7"
// (not the shortest round-trip form of .NET Core 3.0+).

// isNetWhite reports the characters NumberStyles.AllowLeading/TrailingWhite
// accept: U+0009..U+000D and U+0020.
func isNetWhite(c byte) bool { return c == ' ' || (c >= '\t' && c <= '\r') }

func trimNetWhite(s string) string {
	i, j := 0, len(s)
	for i < j && isNetWhite(s[i]) {
		i++
	}
	for j > i && isNetWhite(s[j-1]) {
		j--
	}
	return s[i:j]
}

// numStyle is the subset of System.Globalization.NumberStyles used here.
type numStyle struct {
	leadWhite, trailWhite bool
	leadSign, trailSign   bool
	decimal, exponent     bool
}

var (
	// NumberStyles.AllowLeadingWhite | AllowTrailingWhite (unsigned TryParse in ICUProperty.SetValue)
	styleUnsigned = numStyle{leadWhite: true, trailWhite: true}
	// NumberStyles.Integer
	styleInteger = numStyle{leadWhite: true, trailWhite: true, leadSign: true}
	// NumberStyles.Float
	styleFloat = numStyle{leadWhite: true, trailWhite: true, leadSign: true, decimal: true, exponent: true}
	// NumberStyles.Number (WindowsSpinButton.parseTextBox); AllowThousands is
	// moot there because ',' was already split away.
	styleNumber = numStyle{leadWhite: true, trailWhite: true, leadSign: true, trailSign: true, decimal: true}
)

// netParseNumber ports the shape checks of System.Number.ParseNumber with the
// invariant culture ("+"/"-" signs, "." decimal separator). It returns a
// canonical string accepted by strconv ("-123.45e6") and whether the input
// matched.
func netParseNumber(s string, st numStyle) (string, bool) {
	i, n := 0, len(s)
	neg, sign := false, false
	for i < n {
		c := s[i]
		if st.leadWhite && isNetWhite(c) && !sign {
			i++
			continue
		}
		if st.leadSign && !sign && (c == '+' || c == '-') {
			sign, neg = true, c == '-'
			i++
			continue
		}
		break
	}
	var mant strings.Builder
	digits := 0
	for i < n && s[i] >= '0' && s[i] <= '9' {
		mant.WriteByte(s[i])
		i++
		digits++
	}
	if st.decimal && i < n && s[i] == '.' {
		i++
		frac := 0
		var fb strings.Builder
		for i < n && s[i] >= '0' && s[i] <= '9' {
			fb.WriteByte(s[i])
			i++
			frac++
		}
		if frac > 0 {
			if digits == 0 {
				mant.WriteByte('0')
			}
			mant.WriteByte('.')
			mant.WriteString(fb.String())
		}
		digits += frac
	}
	if digits == 0 {
		return "", false
	}
	if st.exponent && i < n && (s[i] == 'e' || s[i] == 'E') {
		j := i + 1
		esign := ""
		if j < n && (s[j] == '+' || s[j] == '-') {
			esign = s[j : j+1]
			j++
		}
		k := j
		for k < n && s[k] >= '0' && s[k] <= '9' {
			k++
		}
		if k > j {
			mant.WriteString("e" + esign + s[j:k])
			i = k
		}
	}
	for i < n {
		c := s[i]
		if st.trailWhite && isNetWhite(c) {
			i++
			continue
		}
		if st.trailSign && !sign && (c == '+' || c == '-') {
			sign, neg = true, c == '-'
			i++
			continue
		}
		break
	}
	// .NET tolerates trailing NUL characters.
	for i < n && s[i] == 0 {
		i++
	}
	if i != n {
		return "", false
	}
	out := mant.String()
	if neg {
		out = "-" + out
	}
	return out, true
}

func netParseUint(s string, bits int) (uint64, bool) {
	c, ok := netParseNumber(s, styleUnsigned)
	if !ok {
		return 0, false
	}
	v, err := strconv.ParseUint(c, 10, bits)
	if err != nil {
		return 0, false
	}
	return v, true
}

func netParseInt(s string, bits int) (int64, bool) {
	c, ok := netParseNumber(s, styleInteger)
	if !ok {
		return 0, false
	}
	v, err := strconv.ParseInt(c, 10, bits)
	if err != nil {
		return 0, false
	}
	return v, true
}

// netParseInt32 ports int.TryParse(s, out v) (NumberStyles.Integer).
func netParseInt32(s string) (int32, bool) {
	v, ok := netParseInt(s, 32)
	return int32(v), ok
}

// netParseHexInt32 ports int.TryParse(s, NumberStyles.HexNumber, ...): up to
// 32 bits of hex digits (no "0x"), wrapped into int32 like the C#.
func netParseHexInt32(s string) (int32, bool) {
	v, ok := netParseHex(s, 32)
	return int32(uint32(v)), ok
}

// netParseHex ports byte/ushort/int.TryParse with NumberStyles.HexNumber.
func netParseHex(s string, bits int) (uint64, bool) {
	t := trimNetWhite(s)
	if t == "" {
		return 0, false
	}
	for i := 0; i < len(t); i++ {
		if !isHexDigit(t[i]) {
			return 0, false
		}
	}
	v, err := strconv.ParseUint(t, 16, bits)
	if err != nil {
		return 0, false
	}
	return v, true
}

func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// netParseDouble ports double.TryParse(s, style, InvariantCulture) on .NET
// Framework, including the trimmed "Infinity"/"-Infinity"/"NaN" symbol
// fallback and failure on overflow.
func netParseDouble(s string, st numStyle) (float64, bool) {
	if c, ok := netParseNumber(s, st); ok {
		v, err := strconv.ParseFloat(c, 64)
		if err == nil {
			return v, true
		}
		if ne, ok := err.(*strconv.NumError); ok && ne.Err == strconv.ErrRange && !math.IsInf(v, 0) {
			return v, true // underflow to (sub)normal/zero succeeds in .NET
		}
	}
	return netInfNaNSymbol(s)
}

// netParseSingle ports float.TryParse(s, style, InvariantCulture) on .NET
// Framework: parse as double, cast, and fail when the cast overflows.
func netParseSingle(s string, st numStyle) (float32, bool) {
	if c, ok := netParseNumber(s, st); ok {
		d, err := strconv.ParseFloat(c, 64)
		ok := err == nil
		if ne, isNE := err.(*strconv.NumError); isNE && ne.Err == strconv.ErrRange && !math.IsInf(d, 0) {
			ok = true
		}
		if ok {
			f := float32(d)
			if !math.IsInf(float64(f), 0) {
				return f, true
			}
		}
	}
	v, ok := netInfNaNSymbol(s)
	return float32(v), ok
}

func netInfNaNSymbol(s string) (float64, bool) {
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

// netParseBool ports bool.TryParse: "True"/"False" case-insensitively, after
// trimming white space and NUL characters.
func netParseBool(s string) (bool, bool) {
	t := strings.TrimFunc(s, func(r rune) bool { return unicode.IsSpace(r) || r == 0 })
	switch {
	case strings.EqualFold(t, "True"):
		return true, true
	case strings.EqualFold(t, "False"):
		return false, true
	}
	return false, false
}

// netFormatG ports double.ToString() / float.ToString() on .NET Framework:
// the "G" format with prec significant digits (15 for double, 7 for float).
func netFormatG(v float64, prec int) string {
	switch {
	case math.IsNaN(v):
		return "NaN"
	case math.IsInf(v, 1):
		return "Infinity"
	case math.IsInf(v, -1):
		return "-Infinity"
	case v == 0:
		return "0"
	}
	return strconv.FormatFloat(v, 'G', prec, 64)
}

// netFormatFixed ports .NET Framework's fixed-point formatting ("F<n>", "N<n>"
// without group separators, and custom "0.000"): the value is first reduced to
// 15 significant digits, then rounded half away from zero at <decimals>, and a
// result that rounds to zero loses its sign. The invariant "." separator is
// used.
func netFormatFixed(v float64, decimals int) string {
	switch {
	case math.IsNaN(v):
		return "NaN"
	case math.IsInf(v, 1):
		return "Infinity"
	case math.IsInf(v, -1):
		return "-Infinity"
	}
	neg := v < 0
	var dig []byte
	scale := 0
	if v != 0 {
		s := strconv.FormatFloat(math.Abs(v), 'e', 14, 64) // d.dddddddddddddde±XX
		ePos := strings.IndexByte(s, 'e')
		exp, _ := strconv.Atoi(s[ePos+1:])
		dig = []byte(s[:1] + s[2:ePos])
		for len(dig) > 0 && dig[len(dig)-1] == '0' {
			dig = dig[:len(dig)-1]
		}
		scale = exp + 1
	}
	// RoundNumber(ref number, scale + decimals)
	pos := scale + decimals
	i := 0
	for i < pos && i < len(dig) {
		i++
	}
	if i == pos && i < len(dig) && dig[i] >= '5' {
		for i > 0 && dig[i-1] == '9' {
			i--
		}
		if i > 0 {
			dig[i-1]++
		} else {
			scale++
			dig = append(dig[:0], '1')
			i = 1
		}
	} else {
		for i > 0 && dig[i-1] == '0' {
			i--
		}
	}
	if i == 0 {
		scale = 0
		neg = false
	}
	if i < len(dig) {
		dig = dig[:i]
	}
	digitAt := func(k int) byte {
		if k >= 0 && k < len(dig) {
			return dig[k]
		}
		return '0'
	}
	var sb strings.Builder
	if neg {
		sb.WriteByte('-')
	}
	if scale > 0 {
		for k := 0; k < scale; k++ {
			sb.WriteByte(digitAt(k))
		}
	} else {
		sb.WriteByte('0')
	}
	if decimals > 0 {
		sb.WriteByte('.')
		for k := 0; k < decimals; k++ {
			sb.WriteByte(digitAt(scale + k))
		}
	}
	return sb.String()
}

// netMathRound ports Math.Round(double value, int digits) on .NET Framework:
// scale, round half to even, unscale — skipped for |value| >= 1e16.
func netMathRound(v float64, digits int) float64 {
	if digits < 0 {
		digits = 0
	}
	if digits > 15 {
		digits = 15
	}
	if math.Abs(v) < 1e16 {
		p := math.Pow10(digits)
		return math.RoundToEven(v*p) / p
	}
	return v
}

// netConvertToInt32 ports Convert.ToInt32(double): round half to even, fail on
// overflow.
func netConvertToInt32(v float64) (int32, bool) {
	if math.IsNaN(v) || v >= 2147483647.5 || v < -2147483648.5 {
		return 0, false
	}
	return int32(math.RoundToEven(v)), true
}

// utf16Len is .NET string.Length.
func utf16Len(s string) int {
	n := 0
	for _, r := range s {
		if r >= 0x10000 {
			n += 2
		} else {
			n++
		}
	}
	return n
}

// utf16Prefix ports s.Substring(0, n) in UTF-16 code units; a surrogate pair
// cut in half is dropped (Go strings cannot hold a lone surrogate).
func utf16Prefix(s string, n int) string {
	units := 0
	for i, r := range s {
		w := 1
		if r >= 0x10000 {
			w = 2
		}
		if units+w > n {
			return s[:i]
		}
		units += w
	}
	return s
}
