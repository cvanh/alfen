// Package eds ports the installer's object-dictionary metadata layer and the
// EDS-driven property UI logic, with no GUI dependencies:
//
//   - ICUSettings.DataSheet / EDSParameter / EDSParameterOption
//     (ACESettings/ICUSettings/*.cs): parsing EDS.xml (<CANopen><Object Id
//     ParameterName DataType AccessType [Length]> with <Title>, <Units> and
//     <Option Value><Title/></Option> children), FindParameter and
//     SelectParameterOption.
//   - The EDS merge done by ICUNetwork.ICUPropertyDictionary / ICUProperty /
//     ICULanDevice.ParseProperty (ACENetwork/ICUNetwork/*.cs): friendly Title,
//     ICUName ("OD_" + ParameterName, the JSON key used on write), Units, and
//     the device-reported type/category/access fallback for ids not in EDS.
//   - ICUProperty.SetValue's per-SDT value coercion and .NET ToString, so the
//     values sent through api.StoreProperties are byte-identical to the C#.
//   - The exe's property widgets and the All Properties panel
//     (ACEServiceInstaller/ICUServiceInstaller/PanelAllProperties.cs,
//     UIProperty*.cs, UIConfigCategory.cs, UIConfigurationPanel.cs,
//     DlgObjectID.cs): widget-kind selection, labels, ranges, input
//     filtering/validation, display formatting, category grouping/ordering and
//     search — as plain functions over api.Property that a thin view calls.
//
// The installer loads "eds.xml" from its working directory (DataSheet's static
// constructor); this package embeds the shipped copy (EDS.xml, byte-identical to
// firmware/msi_work/files3/EDS.xml) and can also load an external file.
package eds

import (
	"bytes"
	_ "embed"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"

	"alfen/aceclient/internal/api"
)

//go:embed EDS.xml
var embeddedEDS []byte

// EmbeddedXML returns a copy of the embedded EDS.xml as shipped with the
// installer (firmware/msi_work/files3/EDS.xml).
func EmbeddedXML() []byte { return append([]byte(nil), embeddedEDS...) }

// DefaultMaxLength is EDSParameter's MaxLength when the Object has no Length
// attribute (256uL in the EDSParameter constructor). No Object in the shipped
// EDS.xml carries a Length attribute.
const DefaultMaxLength uint64 = 256

// Option ports ICUSettings.EDSParameterOption (EDSParameterOption.cs).
type Option struct {
	Index    int    // position among the Object's <Option> descendants (0-based)
	Value    string // Value attribute, verbatim
	Title    string // first descendant <Title> value; "" when none
	HasTitle bool   // false when the option has no <Title> (C# Title == null)
}

// Parameter ports ICUSettings.EDSParameter (EDSParameter.cs): one <Object>.
type Parameter struct {
	ID    int // Id before "sub", parsed as hex (0 when unparseable)
	SubID int // Id after "sub", parsed as hex (0 when absent/unparseable)

	Name     string // ParameterName attribute
	Title    string // first descendant <Title> (also inside <Option>!); "" when none
	HasTitle bool   // false when the Object has no <Title> descendant (C# null)
	Units    string // first descendant <Units>; "" when none
	HasUnits bool   // false when the Object has no <Units> descendant (C# null)

	// DataType is the DataType attribute (optional "0x" prefix, hex; 0 when
	// unparseable). ICUProperty starts with it, but ParseProperty overwrites
	// it with the device-reported type, which is what the UI keys on.
	DataType int
	// ReadWrite is AccessType equals "rw" (case-insensitive) — "rww"/"rwr"
	// are false. The installer never uses it: ICUProperty.ReadOnly comes from
	// the device's "access".
	ReadWrite bool
	// MaxLength is the Length attribute, or DefaultMaxLength. Only
	// UIPropertyString uses it (initial truncation length); ICUProperty's
	// MaxLength comes from the device's "len".
	MaxLength uint64

	// Options is nil when the Object has no <Option> descendant, exactly like
	// EDSParameter.Options (only allocated when options exist).
	Options []Option

	// Raw attributes, kept verbatim for display (not interpreted by the C#
	// beyond the parsed fields above).
	RawID       string // Id, e.g. "2052sub1"
	RawDataType string // DataType, e.g. "0x0009"
	AccessType  string // AccessType: ro, rw, rww, rwr, r, wo
}

// SDT returns DataType as the object-dictionary type (ICUProperty.Initialize
// casts it the same way: (SDT)param.DataType).
func (p *Parameter) SDT() api.SDT { return api.SDT(p.DataType) }

// Dictionary ports the static ICUSettings.DataSheet (DataSheet.cs): the parsed
// EDS parameters in file order.
type Dictionary struct {
	params []*Parameter
	// first parameter per truncated (ushort Id, byte SubId), the association
	// ICUPropertyDictionary.GetProperty + ICUProperty.Parameter yields.
	byKey map[propKey]*Parameter
}

type propKey struct {
	id  uint16
	sub byte
}

// Parse ports DataSheet.Parse on in-memory XML: every <Object> descendant of
// the root becomes a Parameter, in document order. Like the C# (which catches
// and logs the exception), a failure part-way returns the parameters parsed so
// far together with the error.
func Parse(data []byte) (*Dictionary, error) {
	d := &Dictionary{byKey: map[propKey]*Parameter{}}
	root, err := parseDOM(data)
	if err != nil {
		return d, err
	}
	for _, obj := range root.descendants("Object") {
		p, err := newParameter(obj)
		if err != nil {
			return d, err
		}
		d.add(p)
	}
	return d, nil
}

// Load reads and parses EDS XML from r (DataSheet.Parse on a stream).
func Load(r io.Reader) (*Dictionary, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return &Dictionary{byKey: map[propKey]*Parameter{}}, err
	}
	return Parse(data)
}

// LoadFile ports DataSheet.Parse(fileName): load an external EDS.xml.
func LoadFile(path string) (*Dictionary, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return &Dictionary{byKey: map[propKey]*Parameter{}}, err
	}
	return Parse(data)
}

var (
	defaultOnce sync.Once
	defaultDict *Dictionary
)

// Default returns the dictionary parsed from the embedded EDS.xml (the
// installer's DataSheet static constructor, Parse("eds.xml")). It is parsed
// once and shared; callers must not mutate it.
func Default() *Dictionary {
	defaultOnce.Do(func() {
		d, err := Parse(embeddedEDS)
		if err != nil {
			panic(fmt.Sprintf("eds: embedded EDS.xml: %v", err))
		}
		defaultDict = d
	})
	return defaultDict
}

func (d *Dictionary) add(p *Parameter) {
	d.params = append(d.params, p)
	k := propKey{uint16(p.ID), byte(p.SubID)}
	if _, ok := d.byKey[k]; !ok {
		d.byKey[k] = p
	}
}

// Parameters returns the parameters in file order (DataSheet.Parameters),
// duplicates included (the shipped file has two 2522sub1 and two 2180sub6).
func (d *Dictionary) Parameters() []*Parameter {
	if d == nil {
		return nil
	}
	return d.params
}

// Len is len(DataSheet.Parameters).
func (d *Dictionary) Len() int {
	if d == nil {
		return 0
	}
	return len(d.params)
}

// FindParameter ports DataSheet.FindParameter: the first parameter with
// Id == id && SubId == sub, or nil (also nil on a nil Dictionary, like the
// C# "Parameters == null" guard).
func (d *Dictionary) FindParameter(id, sub int) *Parameter {
	if d == nil {
		return nil
	}
	for _, p := range d.params {
		if p.ID == id && p.SubID == sub {
			return p
		}
	}
	return nil
}

// paramFor is the EDS parameter an ICUProperty(id, sub) carries: the first
// EDS entry whose truncated (ushort)Id/(byte)SubId match
// (ICUPropertyDictionary ctor + GetProperty first-match).
func (d *Dictionary) paramFor(id uint16, sub byte) *Parameter {
	if d == nil {
		return nil
	}
	return d.byKey[propKey{id, sub}]
}

// SelectParameterOption ports DataSheet.SelectParameterOption: the integer
// Value of the option whose Title equals description (ordinal), parsed with
// int.TryParse (0 when not an integer).
//
// ok is false when the parameter is unknown (C# returns 0) or when it has no
// option with that title (the C# dereferences FirstOrDefault(...) == null and
// throws NullReferenceException); value is 0 in both cases.
func (d *Dictionary) SelectParameterOption(id, sub int, description string) (value int, ok bool) {
	p := d.FindParameter(id, sub)
	if p == nil {
		return 0, false
	}
	for _, o := range p.Options {
		if o.HasTitle && o.Title == description {
			v, _ := netParseInt32(o.Value)
			return int(v), true
		}
	}
	return 0, false
}

// OptionByValue returns the first option whose Value equals value (ordinal),
// the lookup ICUDevice.GetPropertyString does.
func (p *Parameter) OptionByValue(value string) (Option, bool) {
	if p == nil {
		return Option{}, false
	}
	for _, o := range p.Options {
		if o.Value == value {
			return o, true
		}
	}
	return Option{}, false
}

var errMissingAttr = errors.New("eds: missing attribute")

// newParameter ports the EDSParameter(XElement) constructor.
func newParameter(e *element) (*Parameter, error) {
	p := &Parameter{}

	id, ok := e.attr("Id")
	if !ok {
		return nil, fmt.Errorf("%w Id on <Object>", errMissingAttr)
	}
	p.RawID = id
	// Split(new[]{"sub"}, StringSplitOptions.None)
	parts := strings.Split(id, "sub")
	if len(parts) != 0 {
		if v, ok := netParseHexInt32(parts[0]); ok {
			p.ID = int(v)
		}
		if len(parts) > 1 {
			if v, ok := netParseHexInt32(parts[1]); ok {
				p.SubID = int(v)
			}
		}
	}

	name, ok := e.attr("ParameterName")
	if !ok {
		return nil, fmt.Errorf("%w ParameterName on <Object Id=%q>", errMissingAttr, id)
	}
	p.Name = name

	dt, ok := e.attr("DataType")
	if !ok {
		return nil, fmt.Errorf("%w DataType on <Object Id=%q>", errMissingAttr, id)
	}
	p.RawDataType = dt
	text := dt
	if len(text) >= 2 && strings.EqualFold(text[:2], "0x") {
		text = text[2:]
	}
	if v, ok := netParseHexInt32(text); ok {
		p.DataType = int(v)
	}

	at, ok := e.attr("AccessType")
	if !ok {
		return nil, fmt.Errorf("%w AccessType on <Object Id=%q>", errMissingAttr, id)
	}
	p.AccessType = at
	p.ReadWrite = strings.EqualFold(at, "rw")

	if t := e.descendants("Title"); len(t) > 0 {
		p.Title = t[0].value()
		p.HasTitle = true
	}
	if u := e.descendants("Units"); len(u) > 0 {
		p.Units = u[0].value()
		p.HasUnits = true
	}
	if l, ok := e.attr("Length"); ok {
		// Convert.ToUInt64(value, InvariantCulture): throws on bad input.
		v, err := strconv.ParseUint(trimNetWhite(l), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("eds: <Object Id=%q> Length %q: %w", id, l, err)
		}
		p.MaxLength = v
	} else {
		p.MaxLength = DefaultMaxLength
	}

	opts := e.descendants("Option")
	if len(opts) == 0 {
		return p, nil
	}
	p.Options = make([]Option, 0, len(opts))
	for i, oe := range opts {
		v, ok := oe.attr("Value")
		if !ok {
			return nil, fmt.Errorf("%w Value on <Option> of <Object Id=%q>", errMissingAttr, id)
		}
		o := Option{Index: i, Value: v}
		if t := oe.descendants("Title"); len(t) > 0 {
			o.Title = t[0].value()
			o.HasTitle = true
		}
		p.Options = append(p.Options, o)
	}
	return p, nil
}

// ---- minimal XLinq-like DOM (XDocument.Load / Descendants / Value) ----

type element struct {
	name    xml.Name
	attrs   []xml.Attr
	content []any // *element or string, in document order
}

// parseDOM mirrors XDocument.Load with LoadOptions.None: whitespace-only text
// nodes are dropped, other text is kept verbatim (entities decoded).
func parseDOM(data []byte) (*element, error) {
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF})
	dec := xml.NewDecoder(bytes.NewReader(data))
	var root *element
	var stack []*element
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("eds: xml: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			el := &element{name: t.Name, attrs: append([]xml.Attr(nil), t.Attr...)}
			if len(stack) == 0 {
				if root != nil {
					return nil, errors.New("eds: xml: multiple root elements")
				}
				root = el
			} else {
				parent := stack[len(stack)-1]
				parent.content = append(parent.content, el)
			}
			stack = append(stack, el)
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if len(stack) == 0 || isXMLWhitespace(t) {
				continue
			}
			parent := stack[len(stack)-1]
			parent.content = append(parent.content, string(t))
		}
	}
	if root == nil {
		return nil, errors.New("eds: xml: no root element")
	}
	return root, nil
}

func isXMLWhitespace(b []byte) bool {
	for _, c := range b {
		if c != ' ' && c != '\t' && c != '\r' && c != '\n' {
			return false
		}
	}
	return true
}

// descendants ports XContainer.Descendants(name): all descendant elements (not
// self) with that local name and no namespace, in document order.
func (e *element) descendants(local string) []*element {
	var out []*element
	var walk func(*element)
	walk = func(n *element) {
		for _, c := range n.content {
			if ce, ok := c.(*element); ok {
				if ce.name.Local == local && ce.name.Space == "" {
					out = append(out, ce)
				}
				walk(ce)
			}
		}
	}
	walk(e)
	return out
}

// value ports XElement.Value: the concatenated text of all descendants.
func (e *element) value() string {
	var sb strings.Builder
	var walk func(*element)
	walk = func(n *element) {
		for _, c := range n.content {
			switch v := c.(type) {
			case string:
				sb.WriteString(v)
			case *element:
				walk(v)
			}
		}
	}
	walk(e)
	return sb.String()
}

// attr ports XElement.Attribute(name)?.Value for an un-namespaced attribute.
func (e *element) attr(local string) (string, bool) {
	for _, a := range e.attrs {
		if a.Name.Local == local && a.Name.Space == "" {
			return a.Value, true
		}
	}
	return "", false
}
