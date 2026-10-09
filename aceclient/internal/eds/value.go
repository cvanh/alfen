package eds

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"alfen/aceclient/internal/api"
)

// ValueKind is the CLR type ICUProperty.m_objValue ends up holding.
type ValueKind int

const (
	ValueNull    ValueKind = iota // null: SDT not handled by SetValue (the property is never shown)
	ValueString                   // string (VISIBLE_STRING, LOW_LIMIT)
	ValueBool                     // bool (BOOLEAN)
	ValueInt                      // sbyte/short/int/long (INTEGER8/16/32/64)
	ValueUint                     // byte/ushort/uint/ulong (UNSIGNED8/16/32/64, BOOLEAN parse failure)
	ValueFloat32                  // float (REAL32)
	ValueFloat64                  // double (REAL64)
	ValueBytes                    // byte[] (BYTEARRAY)
	ValueWords                    // ushort[] (ARRAY_16)
)

// Value is the typed value an ICUProperty holds (ICUProperty.Value), produced
// by ParseValue. Only the field matching Kind is meaningful.
type Value struct {
	Kind  ValueKind
	Str   string
	Bool  bool
	Int   int64
	Uint  uint64
	Float float64 // REAL32 values are float32-exact
	Bytes []byte
	Words []uint16
}

// IsNull reports ICUProperty.Value == null.
func (v Value) IsNull() bool { return v.Kind == ValueNull }

// ValueSupported reports whether ICUProperty.SetValue stores a value for sdt;
// for any other type (OCTET_STRING, UNICODE_STRING, DOMAIN, TIME_*, the 24/40/
// 48/56-bit integers) the value stays null and the C# hides the property.
func ValueSupported(sdt api.SDT) bool {
	switch sdt {
	case api.SDTVisibleString, api.SDTUnsigned8, api.SDTInteger8, api.SDTUnsigned16,
		api.SDTUnsigned32, api.SDTUnsigned64, api.SDTInteger16, api.SDTInteger32,
		api.SDTInteger64, api.SDTReal32, api.SDTReal64, api.SDTBoolean,
		api.SDTByteArray, api.SDTArray16, api.SDTLowLimit:
		return true
	}
	return false
}

// ParseValue ports ICUProperty.SetValue for a property that has no value yet
// (SetInitialValue after ParseProperty, or a UI write): text is
// newValue.ToString(), coerced per data type exactly like the C#:
//
//	VISIBLE_STRING            stored verbatim
//	UNSIGNED8 / INTEGER8      "false"/"true" (case-insensitive) -> 0/1, else TryParse, else 0
//	UNSIGNED16/32/64          TryParse (white space, no sign), else 0
//	INTEGER16/32/64           TryParse (NumberStyles.Integer), else 0
//	REAL32 / REAL64           ',' -> '.', TryParse (NumberStyles.Float), else 0
//	BOOLEAN                   bool.TryParse, else (byte)0
//	BYTEARRAY / ARRAY_16      split on ',', each hex-parsed, else 0
//	LOW_LIMIT                 "" (and the property becomes read-only, see Merge)
//	anything else             null
func ParseValue(sdt api.SDT, text string) Value {
	switch sdt {
	case api.SDTVisibleString:
		return Value{Kind: ValueString, Str: text}
	case api.SDTUnsigned8:
		if strings.EqualFold(text, "false") {
			return Value{Kind: ValueUint, Uint: 0}
		}
		if strings.EqualFold(text, "true") {
			return Value{Kind: ValueUint, Uint: 1}
		}
		v, _ := netParseUint(text, 8)
		return Value{Kind: ValueUint, Uint: v}
	case api.SDTInteger8:
		if strings.EqualFold(text, "false") {
			return Value{Kind: ValueInt, Int: 0}
		}
		if strings.EqualFold(text, "true") {
			return Value{Kind: ValueInt, Int: 1}
		}
		v, _ := netParseInt(text, 8)
		return Value{Kind: ValueInt, Int: v}
	case api.SDTUnsigned16:
		v, _ := netParseUint(text, 16)
		return Value{Kind: ValueUint, Uint: v}
	case api.SDTUnsigned32:
		v, _ := netParseUint(text, 32)
		return Value{Kind: ValueUint, Uint: v}
	case api.SDTUnsigned64:
		v, _ := netParseUint(text, 64)
		return Value{Kind: ValueUint, Uint: v}
	case api.SDTInteger16:
		v, _ := netParseInt(text, 16)
		return Value{Kind: ValueInt, Int: v}
	case api.SDTInteger32:
		v, _ := netParseInt(text, 32)
		return Value{Kind: ValueInt, Int: v}
	case api.SDTInteger64:
		v, _ := netParseInt(text, 64)
		return Value{Kind: ValueInt, Int: v}
	case api.SDTReal32:
		f, ok := netParseSingle(strings.ReplaceAll(text, ",", "."), styleFloat)
		if !ok {
			f = 0
		}
		return Value{Kind: ValueFloat32, Float: float64(f)}
	case api.SDTReal64:
		f, ok := netParseDouble(strings.ReplaceAll(text, ",", "."), styleFloat)
		if !ok {
			f = 0
		}
		return Value{Kind: ValueFloat64, Float: f}
	case api.SDTBoolean:
		if b, ok := netParseBool(text); ok {
			return Value{Kind: ValueBool, Bool: b}
		}
		return Value{Kind: ValueUint, Uint: 0}
	case api.SDTByteArray:
		parts := strings.Split(text, ",")
		out := make([]byte, len(parts))
		for i, s := range parts {
			if b, ok := netParseHex(s, 8); ok {
				out[i] = byte(b)
			}
		}
		return Value{Kind: ValueBytes, Bytes: out}
	case api.SDTArray16:
		parts := strings.Split(text, ",")
		out := make([]uint16, len(parts))
		for i, s := range parts {
			if w, ok := netParseHex(s, 16); ok {
				out[i] = uint16(w)
			}
		}
		return Value{Kind: ValueWords, Words: out}
	case api.SDTLowLimit:
		return Value{Kind: ValueString, Str: ""}
	}
	return Value{Kind: ValueNull}
}

// ValueOf is ParseValue(p.DataType, p.Value): the ICUProperty.Value the C#
// holds after reading p from /api/prop.
func ValueOf(p api.Property) Value { return ParseValue(p.DataType, p.Value) }

// String ports Value.ToString() on .NET Framework (invariant formatting):
// bool -> "True"/"False", float -> G7, double -> G15, byte[] ->
// "System.Byte[]", ushort[] -> "System.UInt16[]", null -> "".
func (v Value) String() string {
	switch v.Kind {
	case ValueString:
		return v.Str
	case ValueBool:
		if v.Bool {
			return "True"
		}
		return "False"
	case ValueInt:
		return strconv.FormatInt(v.Int, 10)
	case ValueUint:
		return strconv.FormatUint(v.Uint, 10)
	case ValueFloat32:
		return netFormatG(v.Float, 7)
	case ValueFloat64:
		return netFormatG(v.Float, 15)
	case ValueBytes:
		return "System.Byte[]"
	case ValueWords:
		return "System.UInt16[]"
	}
	return ""
}

// HexString ports the "X2" (byte[]) / "X4" (ushort[]) comma-joined form used
// by ICULanDevice.StoreProperties, ICUProperty.Element and the All Properties
// custom text rows; other kinds return String().
func (v Value) HexString() string {
	switch v.Kind {
	case ValueBytes:
		s := make([]string, len(v.Bytes))
		for i, b := range v.Bytes {
			s[i] = fmt.Sprintf("%02X", b)
		}
		return strings.Join(s, ",")
	case ValueWords:
		s := make([]string, len(v.Words))
		for i, w := range v.Words {
			s[i] = fmt.Sprintf("%04X", w)
		}
		return strings.Join(s, ",")
	}
	return v.String()
}

// Wire ports the value text ICULanDevice.StoreProperties writes for this
// value: X2/X4 hex CSV for arrays, Convert.ToString(v, en-US) for REAL32/64
// (same as G7/G15 with "."), Value.ToString() otherwise. api.StoreProperties
// adds the per-SDT quoting.
func (v Value) Wire() string { return v.HexString() }

// Equal ports object.Equals between two boxed values, as ICUProperty uses for
// IsChanged (!m_objValue.Equals(DeviceValue)): same CLR type and value; arrays
// compare by reference, so two array values are never equal (every array
// write marks the property changed). Two nulls are reported equal. Values are
// assumed to come from the same property (same SDT): the byte/ushort/uint/
// ulong (and sbyte..long) widths share one Kind.
func (v Value) Equal(o Value) bool {
	if v.Kind != o.Kind {
		return false
	}
	switch v.Kind {
	case ValueNull:
		return true
	case ValueString:
		return v.Str == o.Str
	case ValueBool:
		return v.Bool == o.Bool
	case ValueInt:
		return v.Int == o.Int
	case ValueUint:
		return v.Uint == o.Uint
	case ValueFloat32, ValueFloat64:
		// float.Equals/double.Equals: NaN equals NaN.
		if math.IsNaN(v.Float) && math.IsNaN(o.Float) {
			return true
		}
		return v.Float == o.Float
	}
	return false
}

// Changed ports ICUProperty.IsChanged after a write of v over the device
// value dev: !v.Equals(dev) (false when v is null).
func Changed(dev, v Value) bool {
	if v.IsNull() {
		return false
	}
	return !v.Equal(dev)
}

var errNotConvertible = errors.New("eds: value not convertible")

// ToInt32 ports Convert.ToInt32(object) on the boxed value (round half to
// even for floats; int.Parse for strings).
func (v Value) ToInt32() (int32, error) {
	switch v.Kind {
	case ValueInt:
		if v.Int < math.MinInt32 || v.Int > math.MaxInt32 {
			return 0, errNotConvertible
		}
		return int32(v.Int), nil
	case ValueUint:
		if v.Uint > math.MaxInt32 {
			return 0, errNotConvertible
		}
		return int32(v.Uint), nil
	case ValueFloat32, ValueFloat64:
		if r, ok := netConvertToInt32(v.Float); ok {
			return r, nil
		}
		return 0, errNotConvertible
	case ValueBool:
		if v.Bool {
			return 1, nil
		}
		return 0, nil
	case ValueString:
		if r, ok := netParseInt32(v.Str); ok {
			return r, nil
		}
	}
	return 0, errNotConvertible
}

// ToFloat64 ports Convert.ToDouble(object) on the boxed value.
func (v Value) ToFloat64() (float64, error) {
	switch v.Kind {
	case ValueInt:
		return float64(v.Int), nil
	case ValueUint:
		return float64(v.Uint), nil
	case ValueFloat32, ValueFloat64:
		return v.Float, nil
	case ValueBool:
		if v.Bool {
			return 1, nil
		}
		return 0, nil
	case ValueString:
		if f, ok := netParseDouble(v.Str, styleFloat); ok {
			return f, nil
		}
	}
	return 0, errNotConvertible
}
