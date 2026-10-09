package backoffice

import "testing"

func TestParseElement(t *testing.T) {
	x := `<Option Value="fallback" Group="ICU|ACME">
  <Title Lang="">Acme backoffice</Title>
  <Title Lang=" NL ">Acme NL</Title>
  <Title Lang="de">Acme DE</Title>
  <Connection Id="1.2077"><Property Value="99"/></Connection>
  <Connection Id="1.2078sub1"><Property Value=" ws://gprs.example:9000 "/></Connection>
  <Connection Id="1.2093"><Property Value="1"/></Connection>
  <Connection Id="1.205A"><Property Value="10"/></Connection>
  <Connection Id="1.2078"><Property Value="ignored"/></Connection>
  <Connection Id="1.ABCDEF"><Property Value="unknown"/></Connection>
</Option>`
	b, err := ParseElement([]byte(x))
	if err != nil {
		t.Fatal(err)
	}
	checks := []struct {
		name      string
		got, want any
	}{
		{"Title", b.Title(), "Acme backoffice"},
		{"TitleNL", b.TitleNL(), "Acme NL"},
		{"TitleDE", b.TitleDE(), "Acme DE"},
		{"TitleFR", b.TitleFR(), ""},
		{"Groups", b.Groups(), "ICU|ACME"},
		{"ConnectMethod 99->3", b.ConnectMethod(), 3},
		{"URL trimmed", b.BackOfficeURL_Domain(), "ws://gprs.example:9000"},
		{"SendStationStatus", b.SendStationStatus(), true},
		{"Timezone 0x205A * 6", b.TimezoneMinutes(), 60},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %#v; want %#v", c.name, c.got, c.want)
		}
	}
	if b.Dirty() || b.OriginalValues == nil {
		t.Error("parsed element must be clean with a snapshot")
	}

	// Without <Title> children only the Value/Group attributes are used.
	b, err = ParseElement([]byte(`<Option Value="Only value" Group="G"><Connection Id="1.2077"><Property Value="2"/></Connection></Option>`))
	if err != nil {
		t.Fatal(err)
	}
	if b.Title() != "Only value" || b.Groups() != "G" || b.ConnectMethod() != 0 {
		t.Errorf("no-title element: %q %q %d", b.Title(), b.Groups(), b.ConnectMethod())
	}

	for _, bad := range []string{
		`<Option><Title>no lang</Title></Option>`,
		`<Option><Title Lang="">t</Title><Connection><Property Value="1"/></Connection></Option>`,
		`<Option><Title Lang="">t</Title><Connection Id="1.2077"/></Option>`,
		`<Option><Title Lang="">t</Title><Connection Id="1.205A"><Property Value="x"/></Connection></Option>`,
		`not xml`,
	} {
		if _, err := ParseElement([]byte(bad)); err == nil {
			t.Errorf("ParseElement(%q): expected error", bad)
		}
	}
}
