package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// Categories ports ICULanDevice.RequestCategories: GET /api/categories and
// collect every non-null element of the top-level "categories" value,
// converted with Convert.ToString and trimmed of '"' and ' '. A failed
// request or an empty body yields an empty list (with the request error);
// a body the C# cannot walk yields the entries gathered so far plus an error
// (the C# logs it and returns the partial list).
func (c *Client) Categories(ctx context.Context) ([]string, error) {
	list := []string{}
	state, resp, err := c.ExecuteWebRequest(ctx, "categories", "", nil, ExecOptions{})
	if state != ValidResponse {
		return list, err
	}
	if resp.Body == "" {
		return list, nil
	}
	top, err := decodeJSONValue(resp.Body)
	if err != nil {
		return list, fmt.Errorf("categories: %w", err)
	}
	m, ok := top.(map[string]any)
	if !ok {
		return list, fmt.Errorf("categories: top-level JSON is %s, not an object", netTypeName(top))
	}
	v, ok := m["categories"]
	if !ok {
		return list, nil
	}
	switch x := v.(type) {
	case []any:
		for _, e := range x {
			if e != nil {
				list = append(list, strings.Trim(netString(e), `" `))
			}
		}
	case string: // foreach over a string walks its characters
		for _, r := range x {
			list = append(list, strings.Trim(string(r), `" `))
		}
	default:
		return list, fmt.Errorf("categories: \"categories\" is %s, not enumerable", netTypeName(v))
	}
	return list, nil
}

// UseLimit500 ports the flag of ICULanDevice.UpdateCategories:
// isAHP && FirmwareVersionNumber >= new Version(2, 2).
func UseLimit500(isAHP bool, fwMajor, fwMinor int) bool {
	return isAHP && (fwMajor > 2 || (fwMajor == 2 && fwMinor >= 2))
}

// UpdateCategories ports ICULanDevice.UpdateCategories. limit500 is
// UseLimit500(...). Without categories and with limit500 it reads everything
// with "limit=500"; otherwise it reads each given category (or every
// category from Categories when none are given) with "cat=<c>[&limit=500]",
// stopping at the first failure. It returns all properties read; a failing
// Categories call is returned as the error (the C# then reports success with
// nothing read).
func (c *Client) UpdateCategories(ctx context.Context, limit500 bool, categories ...string) ([]Property, error) {
	if c.IP == "" {
		return nil, errors.New("api: no IP address")
	}
	if len(categories) == 0 && limit500 {
		return c.FetchProperties(ctx, "limit=500")
	}
	cats := categories
	var catErr error
	if len(cats) == 0 {
		cats, catErr = c.Categories(ctx)
	}
	var all []Property
	for _, cat := range cats {
		params := "cat=" + cat
		if limit500 {
			params += "&limit=500"
		}
		props, err := c.FetchProperties(ctx, params)
		all = append(all, props...)
		if err != nil {
			return all, err
		}
	}
	return all, catErr
}

// UpdateProperties ports ICULanDevice.UpdateProperties(string sIds):
// FetchProperties("ids=" + ids). Build ids with CombinedIDs or ODIndexes.
func (c *Client) UpdateProperties(ctx context.Context, ids string) ([]Property, error) {
	if c.IP == "" {
		return nil, errors.New("api: no IP address")
	}
	return c.FetchProperties(ctx, "ids="+ids)
}

// FetchProperties ports ICULanDevice.UpdatePropertiesInternal + ParseProperty.
//
// The first page is GET /api/prop?<parameters> with ExecuteWebRequest(5 s, 2
// attempts); further pages append "&offset=<offset+count>" and use the
// defaults (5 s, 4 attempts). Top-level keys are walked in document order:
// "version" is ignored, "count"/"total"/"offset" must be int32 integers,
// every "properties" element is parsed under the name of its "id" and also
// increments count (so a "count" key that precedes "properties" is added
// to, exactly like the C#), and any other non-bool key is parsed as a legacy
// top-level property named by its key. Paging stops when
// total <= offset + count.
//
// Per property: "value" must be present (null -> "0") and is converted with
// .NET ToString semantics; "type", "id" and "access" must be present and
// convertible, "cat"/"len" are optional; the id must split into exactly two
// hex parts (truncated to 16/8 bits). A property that fails these checks is
// skipped; a structural problem (non-object item, missing "value", bad
// paging field) aborts with an error, as the C# returns false.
//
// Name is the C# propName ("IDX_SUB" for paged entries). ICUPropertyDictionary
// keeps EDS names ("OD_" + ParameterName) instead; that merge, the 8273
// AllowObjectIDUpdate rule and the IsChanged guard belong to the caller.
// When c.Identity is empty it is set from property 8275 (0x2053) as the C#
// does with GetPropertyString(8275).
func (c *Client) FetchProperties(ctx context.Context, parameters string) ([]Property, error) {
	state, resp, err := c.ExecuteWebRequest(ctx, "prop", parameters, nil, ExecOptions{Timeout: RequestTimeout, MaxRetries: 2})
	if state != ValidResponse {
		return nil, err
	}
	var out []Property
	total, offset := 0, 0
	for {
		fields, err := decodeOrderedObject(resp.Body)
		if err != nil {
			return out, fmt.Errorf("prop: %w", err)
		}
		count := 0
		for _, f := range fields {
			switch f.key {
			case "version":
				if _, err := strictInt32(f.val); err != nil {
					return out, fmt.Errorf("prop: version: %w", err)
				}
			case "count":
				if count, err = strictInt32(f.val); err != nil {
					return out, fmt.Errorf("prop: count: %w", err)
				}
			case "total":
				if total, err = strictInt32(f.val); err != nil {
					return out, fmt.Errorf("prop: total: %w", err)
				}
			case "offset":
				if offset, err = strictInt32(f.val); err != nil {
					return out, fmt.Errorf("prop: offset: %w", err)
				}
			case "properties":
				items, ok := f.val.([]any)
				if !ok {
					return out, fmt.Errorf("prop: properties is %s, not an array", netTypeName(f.val))
				}
				for _, it := range items {
					m, ok := it.(map[string]any)
					if !ok {
						return out, fmt.Errorf("prop: property item is %s", netTypeName(it))
					}
					idv, ok := m["id"]
					if !ok || idv == nil {
						return out, errors.New(`prop: property item without "id"`)
					}
					p, ok, err := parsePropertyItem(netString(idv), m)
					if err != nil {
						return out, err
					}
					if ok {
						out = append(out, p)
					}
					count++
				}
			default:
				if _, isBool := f.val.(bool); isBool {
					continue
				}
				if f.val == nil {
					return out, fmt.Errorf("prop: %q is null", f.key)
				}
				p, ok, err := parsePropertyItem(f.key, f.val)
				if err != nil {
					return out, err
				}
				if ok {
					out = append(out, p)
				}
			}
		}
		if total <= offset+count {
			break
		}
		params := fmt.Sprintf("%s&offset=%d", parameters, offset+count)
		state, resp, err = c.ExecuteWebRequest(ctx, "prop", params, nil, ExecOptions{})
		if state != ValidResponse {
			return out, err
		}
	}
	if c.Identity == "" {
		for _, p := range out {
			if p.ID == 8275 && p.Sub == 0 {
				c.Identity = p.ToString()
			}
		}
	}
	return out, nil
}

// parsePropertyItem ports ICULanDevice.ParseProperty. err aborts the whole
// update (exceptions outside the C# try block); ok=false skips the item.
func parsePropertyItem(propName string, item any) (Property, bool, error) {
	m, isMap := item.(map[string]any)
	if !isMap {
		return Property{}, false, fmt.Errorf("prop: %q is %s, not an object", propName, netTypeName(item))
	}
	raw, has := m["value"]
	if !has {
		return Property{}, false, fmt.Errorf(`prop: %q has no "value"`, propName)
	}
	value := "0"
	if raw != nil {
		value = netString(raw)
	}
	tv, has := m["type"]
	if !has {
		return Property{}, false, nil
	}
	dataType, err := netToInt32(jsonToNet(tv))
	if err != nil {
		return Property{}, false, nil
	}
	idv, has := m["id"]
	if !has {
		return Property{}, false, nil
	}
	av, has := m["access"]
	if !has {
		return Property{}, false, nil
	}
	access, err := netToInt32(jsonToNet(av))
	if err != nil {
		return Property{}, false, nil
	}
	category := ""
	if cv, has := m["cat"]; has {
		category = netString(cv)
	}
	var maxLength uint64
	if lv, has := m["len"]; has {
		if maxLength, err = netToUInt64(jsonToNet(lv)); err != nil {
			return Property{}, false, nil
		}
	}
	parts := strings.Split(netString(idv), "_")
	if len(parts) != 2 {
		return Property{}, false, nil
	}
	idx, ok1 := parseNetHexInt32(parts[0])
	sub, ok2 := parseNetHexInt32(parts[1])
	if !ok1 || !ok2 {
		return Property{}, false, nil
	}
	return Property{
		ID:        uint16(idx),
		Sub:       byte(sub),
		Name:      propName,
		DataType:  SDT(dataType),
		Value:     value,
		ReadOnly:  access == 1,
		Category:  category,
		MaxLength: maxLength,
	}, true, nil
}

// StorePropertiesContext ports ICULanDevice.StoreProperties on top of
// ExecuteWebRequest: batches of 15 posted to /api/prop as
// {"<Name>":{"id":"<X>_<X>","value":<v>},...}, each value first normalised
// through ParseValue the way ICUProperty.Value holds it (e.g. BOOLEAN
// "true" -> "True", REAL32 "1,5" -> 1.5, BYTEARRAY "a,1" -> "0A,01").
// Data types without a SetValue branch fail like the C# null dereference
// (that batch is not sent). Earlier batches stay stored when a later one
// fails. The original StoreProperties keeps sending p.Value verbatim.
func (c *Client) StorePropertiesContext(ctx context.Context, props ...Property) error {
	for len(props) > 0 {
		n := min(15, len(props))
		batch := props[:n]
		props = props[n:]
		var sb strings.Builder
		sb.WriteString("{")
		for i, p := range batch {
			text, err := storeValueText(p)
			if err != nil {
				return err
			}
			q := p
			q.Value = text
			sb.WriteString(serializeProperty(q))
			if i < len(batch)-1 {
				sb.WriteString(",")
			}
		}
		sb.WriteString("}")
		state, _, err := c.ExecuteWebRequest(ctx, "prop", "", sb.String(), ExecOptions{})
		if state != ValidResponse {
			return err
		}
	}
	return nil
}

// StoreProperty ports ICULanDevice.storeProperty(propId, subId, newValue)
// for a property p whose Value holds the current device value: newValue is
// parsed per p.DataType; when it differs from the device value
// (ICUProperty.IsChanged) the property is stored and stored=true is
// returned regardless of the outcome (err carries it; the C# ignores it).
// Unchanged values, and data types SetValue cannot hold, return false.
func (c *Client) StoreProperty(ctx context.Context, p Property, newValue string) (stored bool, err error) {
	nv, ok := ParseValue(p.DataType, newValue)
	if !ok {
		return false, nil
	}
	if dv, dok := ParseValue(p.DataType, p.Value); dok && netEquals(nv, dv) {
		return false, nil
	}
	p.Value = newValue
	return true, c.StorePropertiesContext(ctx, p)
}

// SplitCombinedID ports ICUDevice.GetProperty(uint combinedPropId): values up
// to 0xFFFF are a bare index (sub 0), larger ones are index<<8 | sub.
func SplitCombinedID(combined uint32) (id uint16, sub byte) {
	if combined <= 0xFFFF {
		return uint16(combined), 0
	}
	return uint16(combined >> 8), byte(combined)
}

// CombinedIDs ports the id list of ICUDevice.UpdateProperties(params uint[]):
// "{a >> 8:X}_{a & 0xFF:X}" joined by ',' (no bare-index special case).
func CombinedIDs(combined ...uint32) string {
	parts := make([]string, len(combined))
	for i, a := range combined {
		parts[i] = fmt.Sprintf("%X_%X", a>>8, a&0xFF)
	}
	return strings.Join(parts, ",")
}

// ODIndexes ports the id list of ICULanDevice.UpdateProperties(params
// ICUProperty[]): each ODIndex joined by ','.
func ODIndexes(props ...Property) string {
	parts := make([]string, len(props))
	for i, p := range props {
		parts[i] = p.ODIndex()
	}
	return strings.Join(parts, ",")
}

// ParseIDSub ports the id parsing of ICUPropertyDictionary.GetProperty(string
// id_sub): split on '_', at least two parts, each hex (unparsable -> 0); ok
// is false for fewer parts or values beyond 16/8 bits (Convert.ToUInt16/
// ToByte would throw).
func ParseIDSub(s string) (id uint16, sub byte, ok bool) {
	parts := strings.Split(s, "_")
	if len(parts) <= 1 {
		return 0, 0, false
	}
	if r, okp := parseNetHexInt32(parts[0]); okp {
		if r < 0 || r > 0xFFFF {
			return 0, 0, false
		}
		id = uint16(r)
	}
	if r, okp := parseNetHexInt32(parts[1]); okp {
		if r < 0 || r > 0xFF {
			return 0, 0, false
		}
		sub = byte(r)
	}
	return id, sub, true
}

// ---- JavaScriptSerializer helpers ----------------------------------------

type jsonField struct {
	key string
	val any
}

// decodeJSONValue decodes like JavaScriptSerializer.Deserialize<object>,
// keeping numbers as json.Number.
func decodeJSONValue(body string) (any, error) {
	dec := json.NewDecoder(strings.NewReader(body))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	return v, nil
}

// decodeOrderedObject decodes a top-level JSON object keeping key order
// (JavaScriptSerializer's dictionary enumerates in document order).
func decodeOrderedObject(body string) ([]jsonField, error) {
	dec := json.NewDecoder(strings.NewReader(body))
	dec.UseNumber()
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	if d, ok := tok.(json.Delim); !ok || d != '{' {
		return nil, fmt.Errorf("top-level JSON is not an object (%v)", tok)
	}
	var out []jsonField
	for dec.More() {
		kt, err := dec.Token()
		if err != nil {
			return nil, err
		}
		key, _ := kt.(string)
		var v any
		if err := dec.Decode(&v); err != nil {
			return nil, err
		}
		out = append(out, jsonField{key, v})
	}
	if _, err := dec.Token(); err != nil {
		return nil, err
	}
	return out, nil
}

// strictInt32 ports the `(int)item.Value` unboxing: only JSON integers that
// JavaScriptSerializer stores as Int32 pass.
func strictInt32(v any) (int, error) {
	n, ok := v.(json.Number)
	if !ok {
		return 0, fmt.Errorf("%s is not an Int32", netTypeName(v))
	}
	i, err := strconv.ParseInt(n.String(), 10, 32)
	if err != nil {
		return 0, fmt.Errorf("%s is not an Int32", n)
	}
	return int(i), nil
}

var jsonIntRe = regexp.MustCompile(`^-?[0-9]+$`)

// jsonToNet maps a decoded JSON value to the CLR value JavaScriptSerializer
// produces (Int32, Int64, else decimal/double approximated as float64), for
// the Convert.ToXxx ports.
func jsonToNet(v any) any {
	n, ok := v.(json.Number)
	if !ok {
		return v
	}
	s := n.String()
	if jsonIntRe.MatchString(s) {
		if i, err := strconv.ParseInt(s, 10, 32); err == nil {
			return int32(i)
		}
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return i
		}
	}
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// netString ports Convert.ToString / ToString() on a JavaScriptSerializer
// value (invariant culture).
func netString(v any) string {
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
	case json.Number:
		return netNumberText(x.String())
	case map[string]any:
		return "System.Collections.Generic.Dictionary`2[System.String,System.Object]"
	case []any:
		return "System.Object[]"
	}
	return fmt.Sprint(v)
}

// netNumberText renders a JSON number as the CLR value JavaScriptSerializer
// picks would print: Int32/Int64 normalised, decimals verbatim (scale kept,
// "-0.0" -> "0.0"), exponent forms as double "G15".
func netNumberText(s string) string {
	if jsonIntRe.MatchString(s) {
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return strconv.FormatInt(i, 10)
		}
		return s // beyond Int64: decimal, printed verbatim
	}
	if strings.ContainsAny(s, "eE") {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil && !(errors.Is(err, strconv.ErrRange) && !math.IsInf(f, 0)) {
			return s
		}
		return formatNetG(f, 15, 64)
	}
	if strings.HasPrefix(s, "-") && strings.Trim(s, "-0.") == "" {
		return s[1:]
	}
	return s
}

// netTypeName names a decoded JSON value for error messages.
func netTypeName(v any) string {
	switch v.(type) {
	case nil:
		return "null"
	case string:
		return "a string"
	case bool:
		return "a bool"
	case json.Number:
		return "a number"
	case map[string]any:
		return "an object"
	case []any:
		return "an array"
	}
	return fmt.Sprintf("%T", v)
}
