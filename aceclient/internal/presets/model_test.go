package presets

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestDeviceModelEnumMatchesCSharp(t *testing.T) {
	src, err := os.ReadFile(filepath.Join("..", "..", "..", "firmware", "decompiled", "ACENetwork", "ICUNetwork", "ICUDeviceModel.cs"))
	if err != nil {
		t.Skip("decompiled ICUDeviceModel.cs not present")
	}
	names := regexp.MustCompile(`(?m)^\s+([A-Za-z0-9_]+),?\s*$`).FindAllStringSubmatch(string(src), -1)
	if len(names) != len(deviceModelNames) {
		t.Fatalf("C# has %d members, Go %d", len(names), len(deviceModelNames))
	}
	for i, m := range names {
		if deviceModelNames[i] != m[1] {
			t.Errorf("value %d: Go %q, C# %q", i, deviceModelNames[i], m[1])
		}
	}
}

func TestDeviceModelValues(t *testing.T) {
	tests := []struct {
		m    DeviceModel
		v    int
		name string
	}{
		{ModelUnknown, 0, "Unknown"},
		{ModelTwin_4_XL, 19, "Twin_4_XL"},
		{ModelNG900_60503, 20, "NG900_60503"},
		{ModelAHPDC_30001, 171, "AHPDC_30001"},
		{DeviceModel(500), 500, "500"},
	}
	for _, tt := range tests {
		if int(tt.m) != tt.v || tt.m.String() != tt.name {
			t.Errorf("%d %q", tt.m, tt.m.String())
		}
	}
}

func TestParseDeviceModel(t *testing.T) {
	tests := []struct {
		host    string
		sockets int
		fixed   bool
		want    DeviceModel
	}{
		{"Twin 4.0", 1, false, ModelTwin_4_0},
		{"twin-4-0-1234", 1, false, ModelTwin_4_0}, // second pass after cutting the suffix
		{"twin-4-0-a-b", 1, false, ModelUnknown},   // only one suffix is cut
		{"Twin 4 XL", 1, false, ModelTwin_4_XL},
		{"twin-4-xl", 1, false, ModelTwin_4_XL},
		{"Twin 4.0 Single", 1, false, ModelTwin_4_0_Single},
		{"Twin 5", 1, false, ModelTwin_5_0},
		{"twin", 1, false, ModelTwin_3_0},
		{"ICU Eve Mini", 1, false, ModelEve_Mini},
		{"ICU Eve Mini", 1, true, ModelEve_Mini_FC},
		{"icu-EVe-single-x", 1, false, ModelEve_Single},
		{"  ICU  eve_dual ", 1, false, ModelEve_Dual},
		{"Compact", 1, true, ModelCompact_FC},
		{"LOLO3", 1, false, ModelLolo3},
		{"TUBE", 1, false, ModelTube_1},
		{"TUBE", 2, false, ModelTube_2},
		{"NG910-60023", 1, false, ModelNG910_60023},
		{"ng910-60023-ace0123456", 1, false, ModelNG910_60023},
		{"AHWP01-52599-x", 1, false, ModelAHWP01_52599},
		{"AHP02-63021", 1, false, ModelAHP02_63021},
		{"AHPDC-30001", 1, false, ModelAHPDC_30001},
		{"ng9", 1, false, ModelUnknown},
		{"", 1, false, ModelUnknown},
		{"something", 1, false, ModelUnknown},
	}
	for _, tt := range tests {
		if got := ParseDeviceModel(tt.host, tt.sockets, tt.fixed); got != tt.want {
			t.Errorf("ParseDeviceModel(%q,%d,%v) = %v, want %v", tt.host, tt.sockets, tt.fixed, got, tt.want)
		}
	}
}

func TestDeviceModelDisplay(t *testing.T) {
	tests := []struct {
		m    DeviceModel
		host string
		want string
	}{
		{ModelTwin_4_XL, "", "Twin 4 XL"},
		{ModelEve_Mini_FC, "", "ICU Eve Mini"},
		{ModelTube_2, "", "TUBE"},
		{ModelNG910_60023, "", "NG910-60023"},
		{ModelUnknown, "", "Unknown"},
		{ModelUnknown, "ICU-Foo-Bar-12", "foo-bar-12"},          // last '-' at 7: kept
		{ModelUnknown, "somemodel-abc-123", "somemodel-abc"},    // last '-' > 8: cut
		{ModelUnknown, "ICU my_model 2000-SN", "my-model-2000"}, // normalised
	}
	for _, tt := range tests {
		if got := tt.m.Display(tt.host); got != tt.want {
			t.Errorf("%v.Display(%q) = %q, want %q", tt.m, tt.host, got, tt.want)
		}
	}
}

// TestModelRoundTrip documents which models survive the Model string written
// by SaveProperties and read back by LoadProperties' From(): the *_FC
// variants, Tube/Tube_2 and the old Twin label map to a sibling, so loading a file
// saved from such a device asks the model question.
func TestModelRoundTrip(t *testing.T) {
	differs := map[DeviceModel]DeviceModel{
		ModelCompact_FC:  ModelCompact,
		ModelLolo3_FC:    ModelLolo3,
		ModelEve_Mini_FC: ModelEve_Mini,
		ModelTube:        ModelTube_1,
		ModelTube_2:      ModelTube_1,
		ModelTwin:        ModelTwin_3_0, // "Twin" -> "twin"
	}
	for m := DeviceModel(1); int(m) < len(deviceModelNames); m++ {
		got := ParseDeviceModel(m.Display(""), 1, false)
		want := m
		if d, ok := differs[m]; ok {
			want = d
		}
		if got != want {
			t.Errorf("%v: Display %q -> %v, want %v", m, m.Display(""), got, want)
		}
	}
	if !strings.HasPrefix(ModelAHWP01_52599.Display(""), "AHWP01-") {
		t.Error("AHWP display")
	}
}

func TestNewDeviceInfo(t *testing.T) {
	d := NewDeviceInfo("ng910-60023-ace0123456", 2, 1, "ACE0123456", "192.168.1.5", 443)
	if d.ModelType != ModelNG910_60023 || d.Model != "NG910-60023" || d.NumberOfSockets != 2 || d.Identification != "ACE0123456" {
		t.Errorf("%+v", d)
	}
}
