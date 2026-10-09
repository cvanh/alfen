package eds

import (
	"fmt"
	"strings"

	"alfen/aceclient/internal/api"
)

// Property is the merged ICUProperty view of one device-reported property: the
// api.Property read from /api/prop (DataType, Category, ReadOnly and MaxLength
// always come from the device, as ICULanDevice.ParseProperty overwrites them)
// plus the EDS metadata ICUPropertyDictionary attaches.
type Property struct {
	api.Property

	Param    *Parameter // ICUProperty.Parameter; nil when the id is not in EDS
	Title    string     // ICUProperty.Title
	ICUName  string     // ICUProperty.ICUName: JSON key on write
	Units    string     // ICUProperty.Units (EDS only)
	HasUnits bool
	Typed    Value // ICUProperty.Value / DeviceValue after SetInitialValue
}

// propName is the name ParseProperty receives for entries of the
// "properties" array: item["id"].ToString(), i.e. the device's "IDX_SUB".
// api.Property keeps the parsed numbers, so the id is re-rendered in the wire
// format ("%X_%X").
func propName(p api.Property) string { return p.IDSub() }

// Merge ports what ICUPropertyDictionary + ICULanDevice.ParseProperty build
// for one /api/prop entry:
//
//   - ids in EDS: ICUName = "OD_" + ParameterName, Title = EDS Title,
//     Units = EDS Units (ICUProperty.Initialize);
//   - ids not in EDS (PropertyDictionary.AddProperty): ICUName = Title =
//     the device id string;
//   - an empty Title falls back to the device id string as well;
//   - DataType/Category/MaxLength/ReadOnly are the device's; a LOW_LIMIT
//     value forces ReadOnly (ICUProperty.SetValue).
func (d *Dictionary) Merge(p api.Property) Property {
	m := Property{Property: p}
	if param := d.paramFor(p.ID, p.Sub); param != nil {
		m.Param = param
		m.ICUName = "OD_" + param.Name
		m.Title = param.Title
		m.Units = param.Units
		m.HasUnits = param.HasUnits
	} else {
		m.ICUName = propName(p)
		m.Title = propName(p)
	}
	if m.ICUName == "" {
		m.ICUName = propName(p)
	}
	if m.Title == "" {
		m.Title = propName(p)
	}
	m.Typed = ValueOf(p)
	if p.DataType == api.SDTLowLimit {
		m.ReadOnly = true
	}
	return m
}

// Title is Merge(p).Title: the friendly name of a property (EDS Title, else
// the device id).
func (d *Dictionary) Title(p api.Property) string { return d.Merge(p).Title }

// ICUName is Merge(p).ICUName: the JSON key ICULanDevice.StoreProperties
// writes ("OD_" + EDS ParameterName, else the device id).
func (d *Dictionary) ICUName(p api.Property) string { return d.Merge(p).ICUName }

// StoreProperty returns the api.Property to pass to api.StoreProperties so the
// request body is byte-identical to ICULanDevice.StoreProperties after the C#
// assigned v to the ICUProperty: Name = ICUName, Value = v.Wire().
func (d *Dictionary) StoreProperty(p Property, v Value) api.Property {
	out := p.Property
	out.Name = p.ICUName
	if out.Name == "" {
		out.Name = d.ICUName(p.Property)
	}
	out.Value = v.Wire()
	return out
}

// MergeAll merges props in PropertyDictionary order (see DictionaryOrder).
func (d *Dictionary) MergeAll(props []api.Property) []Property {
	ordered := d.DictionaryOrder(props)
	out := make([]Property, len(ordered))
	for i, p := range ordered {
		out[i] = d.Merge(p)
	}
	return out
}

// DictionaryOrder returns props in ICUPropertyDictionary order — the order the
// C# iterates for search and StoreChangedProperties: first every EDS entry in
// file order that the device reported (each (id, sub) once; duplicate EDS
// entries never receive a value), then the ids not in EDS in the order the
// device first reported them. A repeated (id, sub) keeps its first position
// and its last reported data (ParseProperty updates the same ICUProperty).
func (d *Dictionary) DictionaryOrder(props []api.Property) []api.Property {
	latest := make(map[propKey]api.Property, len(props))
	var unknown []propKey
	for _, p := range props {
		k := propKey{p.ID, p.Sub}
		if _, seen := latest[k]; !seen && d.paramFor(p.ID, p.Sub) == nil {
			unknown = append(unknown, k)
		}
		latest[k] = p
	}
	out := make([]api.Property, 0, len(latest))
	if d != nil {
		emitted := make(map[propKey]bool, len(latest))
		for _, param := range d.params {
			k := propKey{uint16(param.ID), byte(param.SubID)}
			if emitted[k] {
				continue
			}
			emitted[k] = true
			if p, ok := latest[k]; ok {
				out = append(out, p)
			}
		}
	}
	for _, k := range unknown {
		out = append(out, latest[k])
	}
	return out
}

// LabelText ports the UIPropertyBase.LabelText getter for a control bound to
// (id, sub): EDS Title, plus " (Units)" when the EDS entry has Units. Without
// an EDS entry it is custom for id == sub == 0, otherwise "XXXX" (sub 0) or
// "XXXX_n" (hex id, decimal sub) and unknown is true — the bound widgets then
// replace it with ICUProperty.Title once a value is present.
//
// Note an EDS entry without <Title> yields "" (C# string interpolation of null).
func (d *Dictionary) LabelText(id uint16, sub byte, custom string) (label string, unknown bool) {
	if param := d.FindParameter(int(id), int(sub)); param != nil {
		label = param.Title
		if param.HasUnits {
			label = fmt.Sprintf("%s (%s)", label, param.Units)
		}
		return label, false
	}
	if id == 0 && sub == 0 {
		return custom, false
	}
	if sub == 0 {
		return fmt.Sprintf("%04X", id), true
	}
	return fmt.Sprintf("%04X_%d", id, sub), true
}

// PropertyString ports ICUDevice.GetPropertyString(propId, subId, 0): the EDS
// option Title whose Value equals the value text, else the value text; ""
// when the property has no value.
func (d *Dictionary) PropertyString(p api.Property) string {
	return d.PropertyStringParent(p, 0)
}

// PropertyStringParent ports GetPropertyString(propId, subId, parentPropId):
// with parentID != 0 the options are those of the EDS entry (parentID, p.Sub)
// — an ICUProperty exists for every EDS entry whether or not the device
// reported it, so only EDS membership matters.
func (d *Dictionary) PropertyStringParent(p api.Property, parentID uint16) string {
	v := ValueOf(p)
	if v.IsNull() {
		return ""
	}
	id := p.ID
	if parentID != 0 {
		id = parentID
	}
	if param := d.paramFor(id, p.Sub); param != nil && param.Options != nil {
		if o, ok := param.OptionByValue(v.String()); ok {
			return o.Title
		}
	}
	return v.String()
}

// DeviceValueString ports ICUDevice.GetDeviceValueString(propId, subId, 0):
// like PropertyString, but REAL32/REAL64 without a matching option are
// formatted with "0.000" (InvariantCulture).
func (d *Dictionary) DeviceValueString(p api.Property) string {
	v := ValueOf(p)
	if v.IsNull() {
		return ""
	}
	if param := d.paramFor(p.ID, p.Sub); param != nil && param.Options != nil {
		if o, ok := param.OptionByValue(v.String()); ok {
			return o.Title
		}
	}
	if p.DataType == api.SDTReal32 || p.DataType == api.SDTReal64 {
		f, _ := v.ToFloat64()
		return netFormatFixed(f, 3)
	}
	return v.String()
}

// ParseIDSub ports ICUPropertyDictionary.GetProperty(string id_sub): split on
// '_' (at least two parts), hex-parse each part (0 when unparseable); ok is
// false when there are fewer than two parts or a part overflows
// ushort/byte (Convert.ToUInt16/ToByte throw in the C#).
func ParseIDSub(idSub string) (id uint16, sub byte, ok bool) {
	parts := strings.Split(idSub, "_")
	if len(parts) < 2 {
		return 0, 0, false
	}
	var i32, s32 int32
	if v, ok := netParseHexInt32(parts[0]); ok {
		i32 = v
	}
	if v, ok := netParseHexInt32(parts[1]); ok {
		s32 = v
	}
	if i32 < 0 || i32 > 0xFFFF || s32 < 0 || s32 > 0xFF {
		return 0, 0, false
	}
	return uint16(i32), byte(s32), true
}

// SplitCombinedID ports ICUDevice.GetProperty(uint combinedPropId): values up
// to 0xFFFF are a bare index (sub 0), larger ones are index<<8 | sub.
func SplitCombinedID(combined uint32) (id uint16, sub byte) {
	if combined <= 0xFFFF {
		return uint16(combined), 0
	}
	return uint16(combined >> 8), byte(combined & 0xFF)
}

// FeatureRightID ports the FeatureRightID PanelBase.AddPropertyBase(ushort Id,
// ...) assigns to id-bound controls: "ID_" + Id:X4.
func FeatureRightID(id uint16) string { return fmt.Sprintf("ID_%04X", id) }

// IDSubPadded is ICUProperty.ID_SUB ("{Id:X4}_{SubId:X2}"), the All
// Properties sort key.
func IDSubPadded(id uint16, sub byte) string { return fmt.Sprintf("%04X_%02X", id, sub) }

// ODIndex is ICUProperty.ODIndex ("{Id:X4}_{SubId:X}").
func ODIndex(id uint16, sub byte) string { return fmt.Sprintf("%04X_%X", id, sub) }

// searchID is the string PanelAllProperties.TryMatchPropertyId matches:
// $"0x{Id:X4}_{SubId:X}".
func searchID(id uint16, sub byte) string { return "0x" + ODIndex(id, sub) }
