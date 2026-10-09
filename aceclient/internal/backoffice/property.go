package backoffice

import "fmt"

// PropertyType ports ICUSettings.ICUBackOfficePropetyTypes (sic)
// (ACESettings/ICUSettings/ICUBackOfficePropetyTypes.cs).
type PropertyType int

const (
	// Hidden properties (titles, groups) are serialized as top-level JSON
	// fields, not in "Values"/"ValuesEx".
	Hidden PropertyType = 0
	// Normal properties are serialized in the "Values" array.
	Normal PropertyType = 1
	// Extended properties are serialized in the "ValuesEx" array.
	Extended PropertyType = 2
)

// String returns the C# enum member name.
func (t PropertyType) String() string {
	switch t {
	case Hidden:
		return "Hidden"
	case Normal:
		return "Normal"
	case Extended:
		return "Extended"
	}
	return fmt.Sprintf("ICUBackOfficePropetyTypes(%d)", int(t))
}

// Property ports ICUSettings.ICUBackOfficeProperty
// (ACESettings/ICUSettings/ICUBackOfficeProperty.cs).
type Property struct {
	Name         string // OD name, the JSON "Key" (e.g. "OD_commConnectMethod")
	PropertyName string // the C# property it backs (e.g. "ConnectMethod")
	// Value is the boxed C# object: string, int or bool.
	Value          any
	PropertyNumber uint32 // (propId << 8) | subId; 0 for hidden fields
	Type           PropertyType
}

// ValueString returns Value.ToString() as the C# would ("True"/"False" for
// bools).
func (p *Property) ValueString() string { return csToString(p.Value) }

// PropID returns the object-dictionary index (PropertyNumber >> 8), as
// ICUBackOffice.GetValue(uint) splits it.
func (p *Property) PropID() uint16 { return uint16(p.PropertyNumber >> 8) }

// SubID returns the object-dictionary subindex (PropertyNumber & 0xFF).
func (p *Property) SubID() byte { return byte(p.PropertyNumber & 0xFF) }

// PMProperty ports ICUSettings.ICUPMProperty
// (ACESettings/ICUSettings/ICUPMProperty.cs).
type PMProperty struct {
	Name           string // OD name, the JSON "Key"
	PropertyName   string // the C# property it backs
	Value          any    // string, int or bool
	PropertyNumber uint32 // (propId << 8) | subId; 0 for "Title"
}

// ValueString returns Value.ToString() as the C# would.
func (p *PMProperty) ValueString() string { return csToString(p.Value) }

// Clone ports ICUPMProperty.Clone (shallow; values are immutable).
func (p *PMProperty) Clone() *PMProperty {
	c := *p
	return &c
}

// ObjectElement is the data of the <Object Id=".." Value="..">PropertyName</Object>
// element ICUPMProperty.Element builds.
type ObjectElement struct {
	ID    string // "1.<index hex>[sub<subindex hex>]", lowercase, unpadded
	Value string // bools become "1"/"0"
	Text  string // the PropertyName
}

// Element ports ICUPMProperty.Element. ok=false where the C# returns null
// (PropertyNumber == 0, i.e. "Title", or an empty value).
func (p *PMProperty) Element() (ObjectElement, bool) {
	if p.PropertyNumber == 0 || p.ValueString() == "" {
		return ObjectElement{}, false
	}
	id := fmt.Sprintf("1.%x", p.PropertyNumber>>8) // Convert.ToString(long, 16)
	if b := byte(p.PropertyNumber & 0xFF); b != 0 {
		id += fmt.Sprintf("sub%x", b)
	}
	val := p.ValueString()
	if bv, isBool := p.Value.(bool); isBool {
		val = "0"
		if bv {
			val = "1"
		}
	}
	return ObjectElement{ID: id, Value: val, Text: p.PropertyName}, true
}
