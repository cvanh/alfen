package presets

import (
	"errors"
	"fmt"
	"strings"

	"alfen/aceclient/internal/api"
)

// PropKey identifies an object-dictionary entry (ICUProperty.Id / SubId).
type PropKey struct {
	ID  uint16
	Sub byte
}

// DataSheetParameter is the subset of ICUSettings.EDSParameter that
// new ICUProperty(EDSParameter) copies (ACESettings/ICUSettings/EDSParameter.cs,
// ICUProperty.Initialize): one <Object> of EDS.xml in document order.
type DataSheetParameter struct {
	ID       uint16
	Sub      byte
	DataType api.SDT
	Name     string // EDS ParameterName; the property's ICUName becomes "OD_" + Name
}

// dictEntry is one ICUProperty with the state PropertyStorage relies on.
type dictEntry struct {
	id        uint16
	sub       byte
	name      string // ICUName
	dataType  api.SDT
	readOnly  bool
	category  string
	maxLength uint64
	value     *propValue // m_objValue (nil = null)
	device    *propValue // DeviceValue
	changed   bool       // IsChanged
}

// setValue ports ICUProperty.SetValue(newValue) for a parse result.
func (e *dictEntry) setValue(r parseResult) {
	if !r.keep {
		e.value = r.v
	}
	if r.readOnly {
		e.readOnly = true
	}
	if e.value != nil {
		e.changed = !e.value.equals(e.device)
	} else {
		e.changed = false
	}
}

// setText is `property.Value = text`.
func (e *dictEntry) setText(text string) { e.setValue(parseValue(e.dataType, text)) }

// setObject is SetValue with a boxed (non-string) value, as Rollback passes
// DeviceValue: VISIBLE_STRING stores the object itself, BYTEARRAY/ARRAY_16
// store any array as-is (same reference), everything else re-parses
// newValue.ToString() (arrays stringify as "").
func (e *dictEntry) setObject(obj *propValue) {
	isArray := obj.kind == kindBytes || obj.kind == kindWords
	switch {
	case e.dataType == api.SDTVisibleString,
		isArray && (e.dataType == api.SDTByteArray || e.dataType == api.SDTArray16):
		e.setValue(parseResult{v: obj})
	case isArray:
		e.setText("")
	default:
		e.setText(obj.toString())
	}
}

// setInitialValue ports ICUProperty.SetInitialValue.
func (e *dictEntry) setInitialValue(text string) {
	r := parseValue(e.dataType, text)
	if !r.keep {
		e.value = r.v
	}
	if r.readOnly {
		e.readOnly = true
	}
	e.device = e.value
	e.changed = false
}

// property returns the api view of the entry with Value formatted the way
// ICULanDevice.StoreProperties serialises it.
func (e *dictEntry) property() api.Property {
	p := api.Property{
		ID:        e.id,
		Sub:       e.sub,
		Name:      e.name,
		DataType:  e.dataType,
		ReadOnly:  e.readOnly,
		Category:  e.category,
		MaxLength: e.maxLength,
	}
	if e.value != nil {
		p.Value = e.value.storeString()
	}
	return p
}

// Dictionary ports the parts of ICUNetwork.ICUPropertyDictionary and
// ICUProperty (ACENetwork/ICUNetwork/ICUPropertyDictionary.cs, ICUProperty.cs)
// that PropertyStorage reads and mutates: entry order, typed values,
// DeviceValue and IsChanged.
//
// Values are typed exactly like ICUProperty.SetValue (per SDT, with the C#
// parse fallbacks), so "changed" means !Value.Equals(DeviceValue) as in the
// C#. Two consequences are kept on purpose: assigning a BYTEARRAY/ARRAY_16
// value from text always marks the property changed (arrays compare by
// reference), and properties of unhandled data types (e.g. UNICODE_STRING,
// DOMAIN, the 24..56-bit integers) keep a null value and are never saved,
// loaded or changed.
type Dictionary struct {
	entries []*dictEntry
}

// NewDictionary ports `new ICUPropertyDictionary()` (one entry per EDS
// parameter, in DataSheet.Parameters order; may be nil) followed by
// ICULanDevice.ParseProperty for every property read from the device:
// existing entries take the device data type, read-only flag, category and
// max length and get SetInitialValue(value); unknown ones are appended
// (AddProperty). EDS entries the device did not report keep a null value.
//
// ICUName: EDS entries use "OD_" + ParameterName; device-only entries use
// api.Property.Name (falling back to IDSub() when empty).
func NewDictionary(params []DataSheetParameter, deviceProps []api.Property) *Dictionary {
	d := &Dictionary{}
	for _, p := range params {
		d.entries = append(d.entries, &dictEntry{
			id:       p.ID,
			sub:      p.Sub,
			name:     "OD_" + p.Name,
			dataType: p.DataType,
		})
	}
	for _, p := range deviceProps {
		e := d.find(p.ID, p.Sub)
		if e == nil {
			name := p.Name
			if name == "" {
				name = p.IDSub()
			}
			e = &dictEntry{id: p.ID, sub: p.Sub, name: name, dataType: p.DataType, readOnly: p.ReadOnly}
			d.entries = append(d.entries, e)
		}
		e.dataType = p.DataType
		if e.name == "" {
			e.name = p.Name
		}
		e.readOnly = p.ReadOnly
		e.category = p.Category
		e.maxLength = p.MaxLength
		if !e.changed {
			e.setInitialValue(p.Value)
		}
	}
	return d
}

// find ports ICUPropertyDictionary.GetProperty(propId, subId) (first match).
func (d *Dictionary) find(id uint16, sub byte) *dictEntry {
	for _, e := range d.entries {
		if e.id == id && e.sub == sub {
			return e
		}
	}
	return nil
}

// Len returns the number of entries.
func (d *Dictionary) Len() int { return len(d.entries) }

// errOverflow mirrors the OverflowException of Convert.ToUInt16/ToByte.
var errOverflow = errors.New("Value was either too large or too small")

// LookupIDSub ports ICUPropertyDictionary.GetProperty(string id_sub): split
// on '_', parse both parts as hex Int32 (a failed parse leaves 0), then
// Convert.ToUInt16 / Convert.ToByte, which fail with an error on overflow.
// found is false when there is no '_' or no such entry.
func (d *Dictionary) LookupIDSub(idSub string) (key PropKey, found bool, err error) {
	parts := strings.Split(idSub, "_")
	if len(parts) <= 1 {
		return PropKey{}, false, nil
	}
	if n, ok := netParseHex(parts[0], 0xFFFFFFFF); ok {
		v := int32(uint32(n))
		if v < 0 || v > 0xFFFF {
			return PropKey{}, false, fmt.Errorf("%w for a UInt16.", errOverflow)
		}
		key.ID = uint16(v)
	}
	if n, ok := netParseHex(parts[1], 0xFFFFFFFF); ok {
		v := int32(uint32(n))
		if v < 0 || v > 0xFF {
			return PropKey{}, false, fmt.Errorf("%w for an unsigned byte.", errOverflow)
		}
		key.Sub = byte(v)
	}
	return key, d.find(key.ID, key.Sub) != nil, nil
}

// SetValue ports `property.Value = text` (ICUProperty.SetValue) and returns
// the resulting IsChanged. ok is false when the entry does not exist.
func (d *Dictionary) SetValue(id uint16, sub byte, text string) (changed, ok bool) {
	e := d.find(id, sub)
	if e == nil {
		return false, false
	}
	e.setText(text)
	return e.changed, true
}

// IsChanged reports ICUProperty.IsChanged for an entry.
func (d *Dictionary) IsChanged(id uint16, sub byte) bool {
	e := d.find(id, sub)
	return e != nil && e.changed
}

// Property returns the entry as an api.Property whose Value is the current
// (possibly edited) value formatted as StoreProperties would send it; Value is
// "" while the C# value is null.
func (d *Dictionary) Property(id uint16, sub byte) (api.Property, bool) {
	e := d.find(id, sub)
	if e == nil {
		return api.Property{}, false
	}
	return e.property(), true
}

// ChangedProperties ports the selection of ICUDevice.StoreChangedProperties:
// every IsChanged entry in dictionary order, as api.Property writes ready for
// api.Client.StoreProperties (Value formatted like
// ICULanDevice.StoreProperties: hex lists for arrays, "True"/"False" for
// BOOLEAN, .NET "G" for reals, decimal integers).
func (d *Dictionary) ChangedProperties() []api.Property {
	var out []api.Property
	for _, e := range d.entries {
		if e.changed {
			out = append(out, e.property())
		}
	}
	return out
}

// StoreChangedProperties ports ICUDevice.StoreChangedProperties with
// ICULanDevice.StoreProperties' batching: changed properties are POSTed to
// /api/prop in batches of 15; each batch that the device accepts is
// committed (ICUProperty.CommitChange: DeviceValue = Value, not changed);
// the first failing batch stops the store and is returned as the error.
func (d *Dictionary) StoreChangedProperties(c *api.Client) error {
	var list []*dictEntry
	for _, e := range d.entries {
		if e.changed {
			list = append(list, e)
		}
	}
	for len(list) > 0 {
		n := min(15, len(list))
		batch := list[:n]
		list = list[n:]
		props := make([]api.Property, len(batch))
		for i, e := range batch {
			props[i] = e.property()
		}
		if err := c.StoreProperties(props...); err != nil {
			return err
		}
		for _, e := range batch {
			e.device = e.value
			e.changed = false
		}
	}
	return nil
}

// RevertChanges ports ICUDevice.RevertChanges: ICUProperty.Rollback
// (SetValue(DeviceValue)) on every changed entry. As in the C#, an entry whose
// DeviceValue is null (an EDS-only property set from a file) is left as is,
// and a REAL value whose default "G" text does not round-trip stays changed.
func (d *Dictionary) RevertChanges() {
	for _, e := range d.entries {
		if e.changed && e.device != nil {
			e.setObject(e.device)
		}
	}
}
