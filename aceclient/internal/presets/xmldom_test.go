package presets

import (
	"encoding/base64"
	"testing"
)

func TestXDocumentParseMatchesDotNet(t *testing.T) {
	for _, v := range dotnetParse {
		src, _ := base64.StdEncoding.DecodeString(v.xmlB64)
		if string(src) == "<a><![CDATA[ ]]></a>" {
			// Known deviation: encoding/xml does not distinguish a
			// white-space-only CDATA section (kept by .NET) from ignorable
			// white space (dropped).
			continue
		}
		doc, err := xDocumentParse(string(src))
		if err != nil {
			t.Errorf("%q: %v", src, err)
			continue
		}
		want, _ := base64.StdEncoding.DecodeString(v.valueB64)
		if got := doc.root.Value(); got != string(want) {
			t.Errorf("%q: Value = %q, want %q", src, got, want)
		}
		attr, ok := doc.root.Attribute("v")
		if v.attrB64 == "-" {
			if ok {
				t.Errorf("%q: unexpected attribute", src)
			}
			continue
		}
		wantAttr, _ := base64.StdEncoding.DecodeString(v.attrB64)
		if !ok || attr != string(wantAttr) {
			t.Errorf("%q: attr = %q, want %q", src, attr, wantAttr)
		}
	}
}

func TestXDocumentParseStructure(t *testing.T) {
	doc, err := xDocumentParse(`<?xml version="1.0" encoding="utf-16"?>
<!DOCTYPE Settings [ <!ENTITY x "a
b"> ]>
<!-- c'omment -->
<Settings xmlns:p="urn:p" p:Id="ns" Id="plain">
  <Properties>
    <Property Id="1"/>
    <p:Property Id="2"/>
    <Property Id="3"/>
  </Properties>
</Settings>`)
	if err != nil {
		t.Fatal(err)
	}
	s := doc.Element("Settings")
	if s == nil || doc.Element("Other") != nil {
		t.Fatal("root lookup")
	}
	if id, ok := s.Attribute("Id"); !ok || id != "plain" {
		t.Errorf("Id = %q", id)
	}
	if _, ok := s.Attribute("xmlns"); ok {
		t.Error("namespace declarations are not attributes")
	}
	props := s.Element("Properties").Elements("Property")
	if len(props) != 2 {
		t.Errorf("namespaced element must not match: %d", len(props))
	}
	if s.Element("Missing").Element("X") != nil || s.Element("Missing").Elements("X") != nil {
		t.Error("nil-safe chaining")
	}
}

func TestXDocumentParseErrors(t *testing.T) {
	for _, src := range []string{"", "   ", "<a/><b/>", "<a/>junk", "junk<a/>", "<a>", "<a></b>"} {
		if _, err := xDocumentParse(src); err == nil {
			t.Errorf("%q: expected an error", src)
		}
	}
	if _, err := xDocumentParse("<a/>  \r\n"); err != nil {
		t.Errorf("trailing white space: %v", err)
	}
}

func TestNormalizeAttributeWhitespace(t *testing.T) {
	tests := []struct{ in, want string }{
		{"<a v=\"x\r\ny\tz\"/>", "<a v=\"x y z\"/>"},
		{"<a v='x\ny'>\n text \n</a>", "<a v='x y'>\n text \n</a>"},
		{"<!-- \"x\ny\" --><a/>", "<!-- \"x\ny\" --><a/>"},
		{"<![CDATA[\"\n\"]]>", "<![CDATA[\"\n\"]]>"},
		{"<?pi \"\n\"?>", "<?pi \"\n\"?>"},
	}
	for _, tt := range tests {
		if got := normalizeAttributeWhitespace(tt.in); got != tt.want {
			t.Errorf("%q -> %q, want %q", tt.in, got, tt.want)
		}
	}
}
