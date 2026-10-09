package backoffice

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
)

// xnode is a minimal XElement: name, attributes and mixed content.
type xnode struct {
	name    string
	attrs   []xml.Attr
	content []any // string (text) or *xnode
}

// attr ports XElement.Attribute(name) for an un-namespaced name.
func (n *xnode) attr(name string) (string, bool) {
	for _, a := range n.attrs {
		if a.Name.Space == "" && a.Name.Local == name {
			return a.Value, true
		}
	}
	return "", false
}

// value ports XElement.Value (all descendant text, document order).
func (n *xnode) value() string {
	var b strings.Builder
	var walk func(*xnode)
	walk = func(x *xnode) {
		for _, c := range x.content {
			switch v := c.(type) {
			case string:
				b.WriteString(v)
			case *xnode:
				walk(v)
			}
		}
	}
	walk(n)
	return b.String()
}

// descendants ports XElement.Descendants(name) (pre-order, self excluded).
func (n *xnode) descendants(name string) []*xnode {
	var out []*xnode
	var walk func(*xnode)
	walk = func(x *xnode) {
		for _, c := range x.content {
			if e, ok := c.(*xnode); ok {
				if e.name == name {
					out = append(out, e)
				}
				walk(e)
			}
		}
	}
	walk(n)
	return out
}

// parseXElement reads the first element of data into an xnode tree.
func parseXElement(data []byte) (*xnode, error) {
	dec := xml.NewDecoder(bytes.NewReader(data))
	var stack []*xnode
	var root *xnode
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Space != "" {
				// XName with a namespace never matches the plain names the C# asks for.
				t.Name.Local = "{" + t.Name.Space + "}" + t.Name.Local
			}
			n := &xnode{name: t.Name.Local, attrs: t.Copy().Attr}
			if len(stack) > 0 {
				p := stack[len(stack)-1]
				p.content = append(p.content, n)
			} else if root == nil {
				root = n
			}
			stack = append(stack, n)
		case xml.EndElement:
			stack = stack[:len(stack)-1]
			if len(stack) == 0 && root != nil {
				return root, nil
			}
		case xml.CharData:
			if len(stack) > 0 {
				p := stack[len(stack)-1]
				p.content = append(p.content, string(t))
			}
		}
	}
	if root == nil {
		return nil, errors.New("no element")
	}
	return root, nil
}

// reConnectionID is the Regex ParseElement applies to <Connection Id="..">.
var reConnectionID = regexp.MustCompile(`([0-9]).([0-9a-fA-F]+)(sub([0-9a-fA-F]+))?`)

// csHexInt ports int.TryParse(s, NumberStyles.HexNumber, ...) (at most 8 hex
// digits, two's-complement wrap like the C#).
func csHexInt(s string) (int32, bool) {
	if s == "" || len(s) > 8 {
		return 0, false
	}
	u, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return 0, false
	}
	return int32(uint32(u)), true
}

// ParseElement ports the ICUBackOffice(EDSParameterOption option) constructor
// and ICUBackOffice.ParseElement: it builds a backoffice from an EDS option
// element (Value/Group attributes, <Title Lang=".."> children and
// <Connection Id="1.XXXXsubYY"><Property Value=".."/></Connection> children).
// data must hold that element. The shipped EDS.xml contains no <Connection>
// elements, so in v4.4.1 this path is effectively unused. Errors are returned
// where the C# would throw (missing Lang/Id/Value attributes).
func ParseElement(data []byte) (*BackOffice, error) {
	x, err := parseXElement(data)
	if err != nil {
		return nil, fmt.Errorf("backoffice element: %w", err)
	}
	b := newBackOffice(false)
	v, _ := x.attr("Value")
	b.setProperty("Title", v)
	titles := x.descendants("Title")
	if len(titles) > 0 {
		getTitle := func(lang string) (string, error) {
			for _, t := range titles {
				l, ok := t.attr("Lang")
				if !ok {
					return "", errors.New("backoffice element: <Title> without Lang attribute")
				}
				if strings.ToLower(csTrim(l)) == lang {
					return t.value(), nil
				}
			}
			return "", nil
		}
		for _, f := range []struct{ prop, lang string }{
			{"Title", ""}, {"TitleNL", "nl"}, {"TitleDE", "de"}, {"TitleFR", "fr"},
		} {
			s, err := getTitle(f.lang)
			if err != nil {
				return nil, err
			}
			b.setProperty(f.prop, s)
		}
		for _, item := range x.descendants("Connection") {
			idAttr, ok := item.attr("Id")
			if !ok {
				return nil, errors.New("backoffice element: <Connection> without Id attribute")
			}
			var fullID uint32
			if m := reConnectionID.FindStringSubmatch(strings.ToLower(csTrim(idAttr))); m != nil {
				if r, ok := csHexInt(m[2]); ok {
					fullID |= uint32(r << 8)
				}
				if r, ok := csHexInt(m[4]); ok {
					fullID |= uint32(r & 0xFF)
				}
			}
			propValue := func() (string, error) {
				for _, p := range item.descendants("Property") {
					if s, ok := p.attr("Value"); ok {
						return s, nil
					}
				}
				return "", fmt.Errorf("backoffice element: <Connection Id=%q> without <Property Value>", idAttr)
			}
			if p := b.GetPropertyByNumber(fullID); p != nil {
				text, err := propValue()
				if err != nil {
					return nil, err
				}
				if fullID == 2127616 && text == "99" {
					text = "3"
				}
				b.setProperty(p.Name, text)
			} else if fullID == 2120192 {
				text, err := propValue()
				if err != nil {
					return nil, err
				}
				num, err := csParseInt32(text)
				if err != nil {
					return nil, fmt.Errorf("backoffice element: timezone %q: %w", text, err)
				}
				if tz := b.GetPropertyByNumber(2125312); tz != nil {
					b.setProperty(tz.Name, num*6)
				}
			}
			// Other ids (0x207800, 0x207F00, 0x205400, 0x206400, 0x206401 or
			// unknown) are ignored (Logger.Debug only).
		}
	}
	g, _ := x.attr("Group")
	b.setProperty("Groups", g)
	b.snapshot()
	return b, nil
}
