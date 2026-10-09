package backoffice

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// The backoffice models hold their values as boxed C# objects (string, int or
// bool). These helpers port the .NET conversions the C# applies to them.

var errFormat = errors.New("input string was not in a correct format")
var errOverflow = errors.New("value was either too large or too small for an Int32")

// csToString ports object.ToString() / Convert.ToString(object) for the value
// kinds a backoffice property can hold (null -> "").
func csToString(v any) string {
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
	case int:
		return strconv.Itoa(x)
	case int32:
		return strconv.FormatInt(int64(x), 10)
	case int64:
		return strconv.FormatInt(x, 10)
	case json.Number:
		return x.String()
	case float64:
		return strconv.FormatFloat(x, 'G', 15, 64)
	default:
		return fmt.Sprint(x)
	}
}

// csTrim ports string.Trim() (char.IsWhiteSpace on both ends).
func csTrim(s string) string { return strings.TrimFunc(s, unicode.IsSpace) }

// isNumWhite is the whitespace NumberStyles.AllowLeading/TrailingWhite
// accepts: U+0009..U+000D and U+0020.
func isNumWhite(r rune) bool { return r == ' ' || (r >= '\t' && r <= '\r') }

// csParseInt32 ports Int32.Parse(s) (NumberStyles.Integer: surrounding
// whitespace and a leading sign allowed, nothing else).
func csParseInt32(s string) (int, error) {
	t := strings.TrimFunc(s, isNumWhite)
	if t == "" {
		return 0, errFormat
	}
	digits := t
	if digits[0] == '+' || digits[0] == '-' {
		digits = digits[1:]
	}
	if digits == "" {
		return 0, errFormat
	}
	for _, r := range digits {
		if r < '0' || r > '9' {
			return 0, errFormat
		}
	}
	n, err := strconv.ParseInt(t, 10, 32)
	if err != nil {
		return 0, errOverflow
	}
	return int(n), nil
}

// roundToInt32 ports Convert.ToInt32(double/decimal): banker's rounding with
// an overflow check.
func roundToInt32(f float64) (int, error) {
	r := math.RoundToEven(f)
	if math.IsNaN(r) || r > math.MaxInt32 || r < math.MinInt32 {
		return 0, errOverflow
	}
	return int(r), nil
}

// csToInt32 ports Convert.ToInt32(object).
func csToInt32(v any) (int, error) {
	switch x := v.(type) {
	case nil:
		return 0, nil
	case int:
		if x > math.MaxInt32 || x < math.MinInt32 {
			return 0, errOverflow
		}
		return x, nil
	case int32:
		return int(x), nil
	case int64:
		if x > math.MaxInt32 || x < math.MinInt32 {
			return 0, errOverflow
		}
		return int(x), nil
	case bool:
		if x {
			return 1, nil
		}
		return 0, nil
	case string:
		return csParseInt32(x)
	case json.Number:
		if n, err := strconv.ParseInt(x.String(), 10, 64); err == nil {
			return csToInt32(n)
		}
		f, err := strconv.ParseFloat(x.String(), 64)
		if err != nil {
			return 0, errFormat
		}
		return roundToInt32(f)
	case float64:
		return roundToInt32(x)
	default:
		return 0, fmt.Errorf("invalid cast from %T to Int32", v)
	}
}

// csParseBool ports Boolean.Parse(string): "True"/"False" case-insensitively,
// tolerating surrounding whitespace (and trailing NULs).
func csParseBool(s string) (bool, error) {
	try := func(t string) (bool, bool) {
		switch {
		case strings.EqualFold(t, "True"):
			return true, true
		case strings.EqualFold(t, "False"):
			return false, true
		}
		return false, false
	}
	if b, ok := try(s); ok {
		return b, nil
	}
	t := strings.TrimLeftFunc(s, unicode.IsSpace)
	t = strings.TrimRightFunc(t, func(r rune) bool { return unicode.IsSpace(r) || r == 0 })
	if b, ok := try(t); ok {
		return b, nil
	}
	return false, errors.New("string was not recognized as a valid Boolean")
}

// csToBool ports Convert.ToBoolean(object).
func csToBool(v any) (bool, error) {
	switch x := v.(type) {
	case nil:
		return false, nil
	case bool:
		return x, nil
	case string:
		return csParseBool(x)
	case int:
		return x != 0, nil
	case int32:
		return x != 0, nil
	case int64:
		return x != 0, nil
	case json.Number:
		f, err := strconv.ParseFloat(x.String(), 64)
		if err != nil {
			return false, errFormat
		}
		return f != 0, nil
	case float64:
		return x != 0, nil
	default:
		return false, fmt.Errorf("invalid cast from %T to Boolean", v)
	}
}

// reNewline is the Regex "\r\n?|\n" both AddVariable implementations use.
var reNewline = regexp.MustCompile(`\r\n?|\n`)

// convertLike ports the body of ICUBackOffice/ICUPMBackOffice.SetProperty: the
// incoming value is converted to the dynamic type of the current value.
// ok=false means the C# threw (caught + logged) and the value is unchanged.
func convertLike(current, value any) (any, bool) {
	switch current.(type) {
	case string:
		return csTrim(csToString(value)), true
	case int:
		n, err := csToInt32(value)
		if err != nil {
			return current, false
		}
		return n, true
	case bool:
		if value == nil { // value.ToString() -> NullReferenceException
			return current, false
		}
		switch csToString(value) {
		case "0":
			return false, true
		case "1":
			return true, true
		}
		b, err := csToBool(value)
		if err != nil {
			return current, false
		}
		return b, true
	default: // "called with unexpected type" -> logged, unchanged
		return current, true
	}
}
