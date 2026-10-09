package config

import "testing"

func TestParseRights(t *testing.T) {
	tests := []struct {
		in      string
		want    Rights
		wantErr bool
	}{
		{"None", RightsNone, false},
		{"ReadOnly", RightsReadOnly, false},
		{"Full", RightsFull, false},
		{" Full ", RightsFull, false},             // Enum.Parse trims
		{"1", RightsReadOnly, false},              // numeric form
		{"7", Rights(7), false},                   // undefined values are accepted
		{"None, ReadOnly", RightsReadOnly, false}, // comma list is OR-ed
		{"full", 0, true},                         // case-sensitive
		{"", 0, true},
		{"Admin", 0, true},
	}
	for _, tc := range tests {
		got, err := ParseRights(tc.in)
		if (err != nil) != tc.wantErr || (!tc.wantErr && got != tc.want) {
			t.Errorf("ParseRights(%q) = %v, %v", tc.in, got, err)
		}
	}
}

func TestParseFeatureType(t *testing.T) {
	for in, want := range map[string]FeatureType{
		"Normal": FeatureTypeNormal, "Page": FeatureTypePage, "Property": FeatureTypeProperty, "Backoffice": FeatureTypeBackoffice, "3": FeatureTypeBackoffice,
	} {
		if got, err := ParseFeatureType(in); err != nil || got != want {
			t.Errorf("ParseFeatureType(%q) = %v, %v", in, got, err)
		}
	}
	if _, err := ParseFeatureType("BackOffice"); err == nil {
		t.Error("expected case-sensitive failure")
	}
}

func TestEnumStrings(t *testing.T) {
	tests := []struct{ name, got, want string }{
		{"RightsNone", RightsNone.String(), "None"},
		{"RightsReadOnly", RightsReadOnly.String(), "ReadOnly"},
		{"RightsFull", RightsFull.String(), "Full"},
		{"Rights(5)", Rights(5).String(), "5"},
		{"FeatureTypeBackoffice", FeatureTypeBackoffice.String(), "Backoffice"},
		{"EncryptNone", EncryptNone.String(), "encryptNone"},
		{"EncryptRijndaelHashed", EncryptRijndaelHashed.String(), "encryptRijndaelHashed"},
	}
	for _, tc := range tests {
		if tc.got != tc.want {
			t.Errorf("%s.String() = %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}
