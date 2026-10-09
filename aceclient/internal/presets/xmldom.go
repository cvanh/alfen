package presets

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"
)

// xElement is the part of System.Xml.Linq.XElement the C# readers use:
// Element(name), Elements(name), Attribute(name) and Value. Names match
// only when the element/attribute has no namespace, like an XName without a
// namespace in LINQ to XML.
type xElement struct {
	name     xml.Name
	attrs    []xml.Attr
	children []*xElement
	value    strings.Builder // XElement.Value: concatenated descendant text
}

// Element ports XElement.Element(name): the first child element with that name.
func (e *xElement) Element(name string) *xElement {
	if e == nil {
		return nil
	}
	for _, c := range e.children {
		if c.name.Space == "" && c.name.Local == name {
			return c
		}
	}
	return nil
}

// Elements ports XElement.Elements(name).
func (e *xElement) Elements(name string) []*xElement {
	if e == nil {
		return nil
	}
	var out []*xElement
	for _, c := range e.children {
		if c.name.Space == "" && c.name.Local == name {
			out = append(out, c)
		}
	}
	return out
}

// Attribute ports XElement.Attribute(name) (ok == false is the C# null).
// Namespace declarations are not attributes in LINQ to XML.
func (e *xElement) Attribute(name string) (string, bool) {
	if e == nil || name == "xmlns" {
		return "", false
	}
	for _, a := range e.attrs {
		if a.Name.Space == "" && a.Name.Local == name {
			return a.Value, true
		}
	}
	return "", false
}

// Value ports XElement.Value.
func (e *xElement) Value() string { return e.value.String() }

// xDocument ports the XDocument root handling: Element(name) returns the root
// element when its name matches.
type xDocument struct {
	root *xElement
}

// Element ports XDocument.Element(name).
func (d *xDocument) Element(name string) *xElement {
	if d == nil || d.root == nil || d.root.name.Space != "" || d.root.name.Local != name {
		return nil
	}
	return d.root
}

// isXMLWhitespace reports whether s consists only of XML white space; such
// text nodes are dropped because XDocument.Load/Parse (LoadOptions.None) read
// with XmlReaderSettings.IgnoreWhitespace.
func isXMLWhitespace(s []byte) bool {
	for _, c := range s {
		if c != ' ' && c != '\t' && c != '\r' && c != '\n' {
			return false
		}
	}
	return true
}

// normalizeAttributeWhitespace applies XML attribute-value normalization to
// literal white space (which XmlReader performs and encoding/xml does not):
// inside quoted attribute values of tags, CR LF, CR, LF and TAB each become
// one space. Character references such as &#10; are untouched, so they still
// decode to the real character as in .NET. Comments, CDATA sections,
// processing instructions and DOCTYPE are copied verbatim.
func normalizeAttributeWhitespace(src string) string {
	if !strings.ContainsAny(src, "\t\r\n") {
		return src
	}
	var sb strings.Builder
	sb.Grow(len(src))
	i := 0
	for i < len(src) {
		if src[i] != '<' {
			sb.WriteByte(src[i])
			i++
			continue
		}
		rest := src[i:]
		if n := skipVerbatim(rest); n > 0 {
			sb.WriteString(rest[:n])
			i += n
			continue
		}
		// start or end tag: copy, normalizing inside quoted values only
		sb.WriteByte('<')
		j := 1
		var quote byte
		for j < len(rest) {
			ch := rest[j]
			j++
			if quote != 0 {
				switch ch {
				case quote:
					quote = 0
					sb.WriteByte(ch)
				case '\r':
					sb.WriteByte(' ')
					if j < len(rest) && rest[j] == '\n' {
						j++
					}
				case '\n', '\t':
					sb.WriteByte(' ')
				default:
					sb.WriteByte(ch)
				}
				continue
			}
			sb.WriteByte(ch)
			if ch == '"' || ch == '\'' {
				quote = ch
			} else if ch == '>' {
				break
			}
		}
		i += j
	}
	return sb.String()
}

// skipVerbatim returns the length of a comment, CDATA section, processing
// instruction or DOCTYPE starting at s (0 if s starts a tag).
func skipVerbatim(s string) int {
	for _, d := range [...]struct{ open, close string }{
		{"<!--", "-->"}, {"<![CDATA[", "]]>"}, {"<?", "?>"},
	} {
		if strings.HasPrefix(s, d.open) {
			k := strings.Index(s[len(d.open):], d.close)
			if k < 0 {
				return len(s)
			}
			return len(d.open) + k + len(d.close)
		}
	}
	if !strings.HasPrefix(s, "<!") {
		return 0
	}
	// DOCTYPE, possibly with an internal subset: up to the closing '>',
	// honouring quotes and [ ].
	depth := 0
	var quote byte
	for j := 2; j < len(s); j++ {
		ch := s[j]
		if quote != 0 {
			if ch == quote {
				quote = 0
			}
			continue
		}
		switch ch {
		case '"', '\'':
			quote = ch
		case '[':
			depth++
		case ']':
			depth--
		case '>':
			if depth <= 0 {
				return j + 1
			}
		}
	}
	return len(s)
}

// parseXML builds the tree from already-decoded UTF-8 text. charset is the
// CharsetReader used for a non-UTF-8 encoding declaration.
func parseXML(text string, charset func(string, io.Reader) (io.Reader, error)) (*xDocument, error) {
	dec := xml.NewDecoder(strings.NewReader(normalizeAttributeWhitespace(text)))
	dec.CharsetReader = charset
	doc := &xDocument{}
	var stack []*xElement
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
			el := &xElement{name: t.Name, attrs: append([]xml.Attr(nil), t.Attr...)}
			if len(stack) == 0 {
				if doc.root != nil {
					return nil, errors.New("There are multiple root elements.")
				}
				doc.root = el
			} else {
				parent := stack[len(stack)-1]
				parent.children = append(parent.children, el)
			}
			stack = append(stack, el)
		case xml.EndElement:
			stack = stack[:len(stack)-1]
		case xml.CharData:
			if isXMLWhitespace(t) {
				continue
			}
			if len(stack) == 0 {
				return nil, errors.New("Data at the root level is invalid.")
			}
			for _, el := range stack {
				el.value.Write(t)
			}
		}
	}
	if doc.root == nil {
		return nil, errors.New("Root element is missing.")
	}
	return doc, nil
}

// passthroughCharset accepts any declared encoding without converting: used
// when the input has already been decoded to a string (XDocument.Parse over a
// string ignores the declaration; a BOM overrides it in XDocument.Load).
func passthroughCharset(_ string, r io.Reader) (io.Reader, error) { return r, nil }

// xDocumentParse ports XDocument.Parse(text) (LoadOptions.None).
func xDocumentParse(text string) (*xDocument, error) {
	return parseXML(strings.TrimPrefix(text, "\uFEFF"), passthroughCharset)
}

// xDocumentLoad ports XDocument.Load(path) over the file bytes: a BOM
// (UTF-8/UTF-16) selects the encoding; otherwise the XML declaration does
// (UTF-8 default; ISO-8859-1, Windows-1252 and US-ASCII are converted, any
// other declared encoding is an error).
func xDocumentLoad(b []byte) (*xDocument, error) {
	switch {
	case bytes.HasPrefix(b, []byte{0xEF, 0xBB, 0xBF}):
		return parseXML(string(b[3:]), passthroughCharset)
	case bytes.HasPrefix(b, []byte{0xFF, 0xFE}):
		return parseXML(decodeUTF16(b[2:], false), passthroughCharset)
	case bytes.HasPrefix(b, []byte{0xFE, 0xFF}):
		return parseXML(decodeUTF16(b[2:], true), passthroughCharset)
	}
	return parseXML(string(b), singleByteCharset)
}

// singleByteCharset converts the single-byte encodings .NET supports out of
// the box for XML declarations.
func singleByteCharset(label string, r io.Reader) (io.Reader, error) {
	l := strings.ToLower(strings.TrimSpace(label))
	var table func(byte) rune
	switch l {
	case "iso-8859-1", "iso8859-1", "latin1", "l1", "iso_8859-1", "cp819", "ibm819":
		table = func(c byte) rune { return rune(c) }
	case "windows-1252", "cp1252", "x-ansi":
		table = cp1252
	case "us-ascii", "ascii", "us", "ansi_x3.4-1968", "iso646-us":
		table = func(c byte) rune {
			if c < 0x80 {
				return rune(c)
			}
			return '?'
		}
	default:
		return nil, fmt.Errorf("System does not support '%s' encoding.", label)
	}
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	var sb strings.Builder
	for _, c := range raw {
		sb.WriteRune(table(c))
	}
	return strings.NewReader(sb.String()), nil
}

// cp1252High maps Windows-1252 bytes 0x80..0x9F (others equal Latin-1).
var cp1252High = [32]rune{
	0x20AC, 0x0081, 0x201A, 0x0192, 0x201E, 0x2026, 0x2020, 0x2021,
	0x02C6, 0x2030, 0x0160, 0x2039, 0x0152, 0x008D, 0x017D, 0x008F,
	0x0090, 0x2018, 0x2019, 0x201C, 0x201D, 0x2022, 0x2013, 0x2014,
	0x02DC, 0x2122, 0x0161, 0x203A, 0x0153, 0x009D, 0x017E, 0x0178,
}

func cp1252(c byte) rune {
	if c >= 0x80 && c <= 0x9F {
		return cp1252High[c-0x80]
	}
	return rune(c)
}
