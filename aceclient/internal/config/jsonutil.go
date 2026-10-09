package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

// jsonObject is the Dictionary<string, object> JavaScriptSerializer.DeserializeObject
// yields for a JSON object (ordinal, case-sensitive keys); values stay raw so
// presence checks (ContainsKey / KeyNotFoundException) can be mirrored exactly.
type jsonObject map[string]json.RawMessage

func isNull(raw json.RawMessage) bool {
	return len(raw) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}

func decodeObject(raw json.RawMessage, what string) (jsonObject, error) {
	if isNull(raw) {
		return nil, fmt.Errorf("config: %s: null object", what)
	}
	var o jsonObject
	if err := json.Unmarshal(raw, &o); err != nil {
		return nil, fmt.Errorf("config: %s: %w", what, err)
	}
	return o, nil
}

func (o jsonObject) has(key string) bool {
	_, ok := o[key]
	return ok
}

// str is obj[key] used as a string (e.g. obj["Name"].Trim()): a missing key is
// the KeyNotFoundException, null / non-string the runtime binder failure.
func (o jsonObject) str(key, what string) (string, error) {
	raw, ok := o[key]
	if !ok {
		return "", fmt.Errorf("config: %s: the given key %q was not present in the dictionary", what, key)
	}
	var s *string
	if err := json.Unmarshal(raw, &s); err != nil || s == nil {
		return "", fmt.Errorf("config: %s: %q is not a string", what, key)
	}
	return *s, nil
}

// array is foreach (dynamic x in obj[key]) for a loop whose body indexes every
// item as a dictionary (see foreachItems).
func (o jsonObject) array(key, what string) ([]json.RawMessage, error) {
	raw, ok := o[key]
	if !ok {
		return nil, fmt.Errorf("config: %s: the given key %q was not present in the dictionary", what, key)
	}
	a, err := foreachItems(raw)
	if err != nil {
		return nil, fmt.Errorf("config: %s: %q: %w", what, key, err)
	}
	return a, nil
}

// foreachItems mirrors `foreach (dynamic item in value)` over a value
// JavaScriptSerializer.DeserializeObject produced, for loops whose body
// indexes each item as a dictionary (item["Key"], new ICUUser(groups, item),
// ...). An array (object[]) yields its elements. A string and a
// Dictionary<string, object> are IEnumerable too: when empty they enumerate
// nothing, otherwise the first item (a char / a KeyValuePair) has no string
// indexer and the body throws before touching any state, so that is reported
// here. null throws NullReferenceException (GetEnumerator on null), numbers
// and booleans have no IEnumerable conversion (RuntimeBinderException).
func foreachItems(raw json.RawMessage) ([]json.RawMessage, error) {
	b := bytes.TrimSpace(raw)
	if len(b) == 0 {
		return nil, fmt.Errorf("no value")
	}
	switch b[0] {
	case '[':
		var a []json.RawMessage
		if err := json.Unmarshal(b, &a); err != nil {
			return nil, err
		}
		return a, nil
	case '"':
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return nil, err
		}
		if s == "" {
			return nil, nil
		}
		return nil, fmt.Errorf("a string item (char) cannot be indexed")
	case '{':
		var o map[string]json.RawMessage
		if err := json.Unmarshal(b, &o); err != nil {
			return nil, err
		}
		if len(o) == 0 {
			return nil, nil
		}
		return nil, fmt.Errorf("an object item (KeyValuePair) cannot be indexed")
	case 'n':
		return nil, fmt.Errorf("null is not enumerable")
	}
	return nil, fmt.Errorf("not enumerable")
}

// Type names object.ToString() returns for the non-scalar values
// JavaScriptSerializer.DeserializeObject produces.
const (
	csArrayTypeName = "System.Object[]"
	csDictTypeName  = "System.Collections.Generic.Dictionary`2[System.String,System.Object]"
)

// convertToString mirrors Convert.ToString(object) on a JavaScriptSerializer
// value. It never fails: null -> "", scalars as scalarString renders them
// (numbers by their JSON literal; the C# formats them with the current
// culture), an array -> "System.Object[]", an object -> the
// Dictionary<string, object> type name.
func convertToString(raw json.RawMessage) string {
	if s, err := scalarString(raw, true); err == nil {
		return s
	}
	if b := bytes.TrimSpace(raw); len(b) > 0 {
		switch b[0] {
		case '[':
			return csArrayTypeName
		case '{':
			return csDictTypeName
		}
	}
	return ""
}

// convertToBoolean mirrors Convert.ToBoolean(object) on a JavaScriptSerializer
// value: null -> false (ToBoolean(string null)), a bool as is, any number
// (int/long/decimal/double) -> value != 0, a string through Boolean.Parse
// ("True"/"False" ignoring case, then again after trimming white space and
// NUL; anything else is a FormatException). Arrays and objects are not
// IConvertible (InvalidCastException).
func convertToBoolean(raw json.RawMessage) (bool, error) {
	if isNull(raw) {
		return false, nil
	}
	var v any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if err := d.Decode(&v); err != nil {
		return false, err
	}
	switch t := v.(type) {
	case bool:
		return t, nil
	case json.Number:
		f, _ := strconv.ParseFloat(t.String(), 64) // out of range -> ±Inf, still != 0
		return f != 0, nil
	case string:
		for _, s := range []string{t, strings.TrimFunc(t, func(r rune) bool { return unicode.IsSpace(r) || r == 0 })} {
			switch {
			case strings.EqualFold(s, "True"):
				return true, nil
			case strings.EqualFold(s, "False"):
				return false, nil
			}
		}
		return false, fmt.Errorf("string %q was not recognized as a valid Boolean", t)
	}
	typeName := csDictTypeName
	if _, ok := v.([]any); ok {
		typeName = csArrayTypeName
	}
	return false, fmt.Errorf("unable to cast object of type '%s' to type 'System.IConvertible'", typeName)
}

// scalarString mirrors calling .ToString() / Convert.ToString() on a value
// JavaScriptSerializer produced: strings as-is, numbers by their literal,
// booleans as "True"/"False". nullOK selects Convert.ToString (null -> "")
// over obj.ToString() (NullReferenceException).
func scalarString(raw json.RawMessage, nullOK bool) (string, error) {
	if isNull(raw) {
		if nullOK {
			return "", nil
		}
		return "", fmt.Errorf("null value")
	}
	var v any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if err := d.Decode(&v); err != nil {
		return "", err
	}
	switch t := v.(type) {
	case string:
		return t, nil
	case json.Number:
		return t.String(), nil
	case bool:
		if t {
			return "True", nil
		}
		return "False", nil
	}
	return "", fmt.Errorf("not a scalar value")
}

// escapeControlChars re-encodes raw control characters (< 0x20) found inside
// JSON strings as \u00XX. JavaScriptSerializer accepts them literally (its
// DeserializeString appends any non-quote, non-backslash char), and the C#
// writer (ICUBaseObject.ValidString) never escapes them - only CR/LF become
// "<br>". encoding/json rejects them, so this is applied as a fallback.
func escapeControlChars(b []byte) []byte {
	var out bytes.Buffer
	out.Grow(len(b) + 64)
	inString, escaped := false, false
	for _, c := range b {
		switch {
		case inString && escaped:
			escaped = false
		case inString && c == '\\':
			escaped = true
		case c == '"':
			inString = !inString
		case inString && c < 0x20:
			out.WriteString(`\u00`)
			out.WriteString(strconv.FormatUint(uint64(c)>>4, 16))
			out.WriteString(strconv.FormatUint(uint64(c)&0xF, 16))
			continue
		}
		out.WriteByte(c)
	}
	return out.Bytes()
}
