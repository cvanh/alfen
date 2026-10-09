package isah

import (
	"reflect"
	"strings"
	"testing"
)

func TestFeatureValues(t *testing.T) {
	cases := []struct {
		f    Features
		want uint32
		name string
	}{
		{FeatureNone, 0x0, "None"},
		{FeatureLoadBalancingSCN, 0x1, "LoadBalancing_SCN"},
		{FeatureLoadBalancingStatic, 0x2, "LoadBalancing_Static"},
		{FeatureLoadBalancingActive, 0x4, "LoadBalancing_Active"},
		{FeatureHighPowerSockets, 0x10, "HighPowerSockets"},
		{FeatureRFIDReader, 0x100, "RFIDReader"},
		{FeatureISO15118, 0x200, "ISO15118"},
		{FeaturePersonalizedDisplay, 0x1000, "PersonalizedDisplay"},
		{FeatureMobile3G4G, 0x10000, "Mobile3G4G"},
		{FeaturePaymentOptions, 0x100000, "Payment_Options"},
		{FeatureExposeSmartMeterData, 0x1000000, "Expose_SmartMeterData"},
		{FeatureObjectID, 0x80000000, "ObjectID"},
	}
	if len(cases) != len(featureEntries) {
		t.Fatalf("featureEntries has %d members, test has %d", len(featureEntries), len(cases))
	}
	for i, c := range cases {
		if uint32(c.f) != c.want {
			t.Errorf("%s = %#x, want %#x", c.name, uint32(c.f), c.want)
		}
		if got := c.f.String(); got != c.name {
			t.Errorf("Features(%#x).String() = %q, want %q", c.want, got, c.name)
		}
		if featureEntries[i].value != c.f || featureEntries[i].name != c.name {
			t.Errorf("featureEntries[%d] = %+v, want ascending order entry %s", i, featureEntries[i], c.name)
		}
		back, err := ParseFeatures(c.name, false)
		if err != nil || back != c.f {
			t.Errorf("ParseFeatures(%q) = %v, %v", c.name, back, err)
		}
	}
	if FeaturesAll != Features(^uint32(0)) {
		t.Errorf("FeaturesAll = %#x", uint32(FeaturesAll))
	}
}

func TestFeaturesString(t *testing.T) {
	cases := []struct {
		f    Features
		want string
	}{
		{0, "None"},
		{FeatureLoadBalancingSCN | FeatureLoadBalancingStatic | FeatureRFIDReader, "LoadBalancing_SCN, LoadBalancing_Static, RFIDReader"},
		{FeatureObjectID | FeatureLoadBalancingSCN, "LoadBalancing_SCN, ObjectID"},
		{0x8, "8"},                       // undefined bit -> number
		{FeatureRFIDReader | 0x8, "264"}, // any undefined bit -> whole number
		{FeaturesAll, "4294967295"},      // uint.MaxValue as PanelInformation uses it
	}
	for _, c := range cases {
		if got := c.f.String(); got != c.want {
			t.Errorf("Features(%#x).String() = %q, want %q", uint32(c.f), got, c.want)
		}
	}
	var all Features
	var names []string
	for _, e := range featureEntries[1:] {
		all |= e.value
		names = append(names, e.name)
	}
	if got, want := all.String(), strings.Join(names, ", "); got != want {
		t.Errorf("all named flags = %q, want %q", got, want)
	}
	if back, err := ParseFeatures(all.String(), false); err != nil || back != all {
		t.Errorf("round trip of all named flags = %#x, %v", uint32(back), err)
	}
}

func TestParseFeatures(t *testing.T) {
	cases := []struct {
		in         string
		ignoreCase bool
		want       Features
		errSub     string
	}{
		{"RFIDReader", false, FeatureRFIDReader, ""},
		{"  LoadBalancing_SCN ,ISO15118  ", false, FeatureLoadBalancingSCN | FeatureISO15118, ""},
		{" 257 ", false, 257, ""},
		{"+5", false, 5, ""},
		{"-0", false, 0, ""},
		{"4294967295", false, FeaturesAll, ""},
		{"rfidreader", true, FeatureRFIDReader, ""},
		{"rfidreader", false, 0, "Requested value 'rfidreader' was not found."},
		{"", false, 0, "Must specify valid information"},
		{"   ", false, 0, "Must specify valid information"},
		{"Foo", false, 0, "Requested value 'Foo' was not found."},
		{"RFIDReader,", false, 0, "was not found"}, // empty part after ','
		{"-1", false, 0, "too large or too small for a UInt32"},
		{"4294967296", false, 0, "too large or too small for a UInt32"},
		{"3D", false, 0, "Requested value '3D' was not found."}, // FormatException falls through to names
	}
	for _, c := range cases {
		got, err := ParseFeatures(c.in, c.ignoreCase)
		if c.errSub != "" {
			if err == nil || !strings.Contains(err.Error(), c.errSub) {
				t.Errorf("ParseFeatures(%q, %v) error = %v, want containing %q", c.in, c.ignoreCase, err, c.errSub)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("ParseFeatures(%q, %v) = %#x, %v; want %#x", c.in, c.ignoreCase, uint32(got), err, uint32(c.want))
		}
	}
}

func TestGetFeatureTextLongList(t *testing.T) {
	allTexts := []string{
		"Smart Charging Network", "Active load balancing", "Static Load balancing",
		"32A output per socket", "RFID reader", "ISO15118", "Personalized display",
		"Mobile Technology 3G & 4G", "Direct Payment Solutions",
	}
	cases := []struct {
		name      string
		f         Features
		ahp, isDC bool
		want      []string
	}{
		{"none", FeatureNone, false, false, []string{"None"}},
		{"scn implies active+static", FeatureLoadBalancingSCN, false, false, allTexts[:3]},
		{"active implies static", FeatureLoadBalancingActive, false, false, allTexts[1:3]},
		{"static", FeatureLoadBalancingStatic, false, false, allTexts[2:3]},
		{"high power", FeatureHighPowerSockets, false, false, []string{"32A output per socket"}},
		{"high power hidden on DC", FeatureHighPowerSockets, false, true, []string{}},
		{"mobile", FeatureMobile3G4G, false, false, []string{"Mobile Technology 3G & 4G"}},
		{"mobile hidden on AHP", FeatureMobile3G4G, true, false, []string{}},
		{"no text for smart meter / object id", FeatureExposeSmartMeterData | FeatureObjectID, false, false, []string{}},
		{"all", FeaturesAll, false, false, allTexts},
		{"all on AHP DC", FeaturesAll, true, true, []string{
			"Smart Charging Network", "Active load balancing", "Static Load balancing",
			"RFID reader", "ISO15118", "Personalized display", "Direct Payment Solutions",
		}},
	}
	for _, c := range cases {
		got := GetFeatureTextLongList(c.f, c.ahp, c.isDC)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: GetFeatureTextLongList(%#x, %v, %v) = %q, want %q", c.name, uint32(c.f), c.ahp, c.isDC, got, c.want)
		}
	}
	if got := GetFeatureTextLong(FeatureRFIDReader|FeaturePaymentOptions, false, false, DefaultFeatureTextJoin); got != "RFID reader, Direct Payment Solutions" {
		t.Errorf("GetFeatureTextLong(\", \") = %q", got)
	}
	if got := GetFeatureTextLong(FeatureLoadBalancingActive, false, false, "\n"); got != "Active load balancing\nStatic Load balancing" {
		t.Errorf("GetFeatureTextLong(\"\\n\") = %q", got)
	}
	if got := GetFeatureTextLong(FeatureNone, false, false, "\n"); got != "None" {
		t.Errorf("GetFeatureTextLong(None) = %q", got)
	}
}

func TestIsFeatureUnlocked(t *testing.T) {
	v := func(s string) Version {
		t.Helper()
		x, err := ParseVersion(s)
		if err != nil {
			t.Fatal(err)
		}
		return x
	}
	cases := []struct {
		name    string
		fw      Version
		flags   uint32
		feature Features
		ahp     bool
		want    bool
	}{
		{"ng9xx pre-3.4: scn needs flag", v("3.3.9"), 0, FeatureLoadBalancingSCN, false, false},
		{"ng9xx pre-3.4: scn with flag", v("3.3.9"), 1, FeatureLoadBalancingSCN, false, true},
		{"ng9xx pre-3.4: others free", v("3.3.9"), 0, FeatureRFIDReader, false, true},
		{"ng9xx pre-3.4: None free", v("2.0.0"), 0, FeatureNone, false, true},
		{"ng9xx 3.4 (no build) < 3.4.0", v("3.4"), 0, FeatureRFIDReader, false, true},
		{"ng9xx 3.4.0 licensed: missing", v("3.4.0"), 0, FeatureRFIDReader, false, false},
		{"ng9xx 3.4.0 licensed: present", v("3.4.0"), 0x100, FeatureRFIDReader, false, true},
		{"ng9xx licensed: None never", v("6.4.0"), 0xFFFFFFFF, FeatureNone, false, false},
		{"ng9xx licensed: object id bit", v("6.4.0"), 0x80000000, FeatureObjectID, false, true},
		{"ahp pre-1.4 everything", v("1.3.9"), 0, FeatureLoadBalancingSCN, true, true},
		{"ahp 1.4.0 licensed: missing", v("1.4.0"), 0, FeatureISO15118, true, false},
		{"ahp 1.4.0 licensed: present", v("1.4.0"), 0x200, FeatureISO15118, true, true},
		{"ahp 1.0 is not ng9xx rule", v("1.0.0"), 0, FeatureLoadBalancingSCN, true, true},
		{"ng9xx 1.0 uses ng9xx rule", v("1.0.0"), 0, FeatureLoadBalancingSCN, false, false},
		{"zero version ng9xx", ZeroVersion, 0, FeaturePaymentOptions, false, true},
	}
	for _, c := range cases {
		if got := IsFeatureUnlocked(c.fw, c.flags, c.feature, c.ahp); got != c.want {
			t.Errorf("%s: IsFeatureUnlocked(%s, %#x, %s, ahp=%v) = %v, want %v", c.name, c.fw, c.flags, c.feature, c.ahp, got, c.want)
		}
		d := DeviceFeatures{FirmwareVersion: c.fw, FeatureFlags: c.flags, IsAHP: c.ahp}
		if got := d.IsFeatureUnlocked(c.feature); got != c.want {
			t.Errorf("%s: DeviceFeatures.IsFeatureUnlocked = %v, want %v", c.name, got, c.want)
		}
	}
}
