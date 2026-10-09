package settings

import (
	"errors"
	"net/netip"
	"os"
	"reflect"
	"regexp"
	"testing"
)

func TestParseIPAddress(t *testing.T) {
	cases := []struct {
		in   string
		want string // "" = invalid
	}{
		{"192.168.1.10", "192.168.1.10"},
		{"0.0.0.0", "0.0.0.0"},
		{"255.255.255.255", "255.255.255.255"},
		{"10", "0.0.0.10"},
		{"0", "0.0.0.0"},
		{"192.168.1", "192.168.0.1"},
		{"10.1", "10.0.0.1"},
		{"1.2.65535", "1.2.255.255"},
		{"1.16777215", "1.255.255.255"},
		{"4294967295", "255.255.255.255"},
		{"0xFFFFFFFF", "255.255.255.255"},
		{"0x7f.1", "127.0.0.1"},
		{"0X7F.0.0.1", "127.0.0.1"},
		{"010.0.0.1", "8.0.0.1"},
		{"00.0.0.01", "0.0.0.1"},
		{"192.168.001.010", "192.168.1.8"},
		{"", ""},
		{" 1.2.3.4", ""},
		{"1.2.3.4 ", ""},
		{"1.2.3.4.", ""},
		{"1.2.3.4.5", ""},
		{"1..2", ""},
		{".1.2.3", ""},
		{"256.1.1.1", ""},
		{"1.2.3.256", ""},
		{"1.2.65536", ""},
		{"1.16777216", ""},
		{"4294967296", ""},
		{"08.1.1.1", ""},
		{"0x", ""},
		{"0x.1", ""},
		{"1.2.3.4/24", ""},
		{"1.2.3.4?", ""},
		{"abc", ""},
		{"1.2.3.a", ""},
		{"::1", "::1"},
		{"[::1]", "::1"},
		{"fe80::1%4", "fe80::1%4"},
		{"::ffff:192.168.1.1", "::ffff:192.168.1.1"},
		{"2001:DB8::1", "2001:db8::1"},
		{"1.2.3.4:443", ""},
		{":::", ""},
		// WSAStringToAddress bracket / port / scope forms; the port is dropped.
		{"[::1]:443", "::1"},
		{"[fe80::1%4]:80", "fe80::1%4"},
		{"[2001:db8::1]:65535", "2001:db8::1"},
		{"[::1]:0", ""},
		{"[::1]:65536", ""},
		{"[::1]:", ""},
		{"[::1]:44x", ""},
		{"[::1]x", ""},
		{"[::1", ""},
		{"::1]", ""},
		{"[1.2.3.4]:80", ""},
		// Scope IDs are decimal only; 0 means none.
		{"fe80::1%0", "fe80::1"},
		{"fe80::1%4294967295", "fe80::1%4294967295"},
		{"fe80::1%4294967296", ""},
		{"fe80::1%eth0", ""},
		{"fe80::1%04", ""},
		{"fe80::1%", ""},
		// IPAddress.ToString() prints embedded IPv4 in dotted form.
		{"::1.2.3.4", "::1.2.3.4"},
		{"::0.1.0.0", "::0.1.0.0"},
		{"::0.0.0.1", "::1"},
		{"::ffff:0:1.2.3.4", "::ffff:0:1.2.3.4"},
		{"fe80::5efe:1.2.3.4", "fe80::5efe:1.2.3.4"},
		{"::5efe:1.2.3.4", "::5efe:1.2.3.4"},
		{"1:2:3:4:0:5efe:102:304", "1:2:3:4:0:5efe:1.2.3.4"},
		{"::ffff:1:1.2.3.4", "::ffff:1:102:304"},
		{"0:0:0:0:0:0:0:0", "::"},
		{"1:0:0:2:0:0:0:3", "1:0:0:2::3"},
		{"1:0:0:2:0:0:3:4", "1::2:0:0:3:4"},
		{"1:0:2:3:4:5:6:7", "1:0:2:3:4:5:6:7"},
		{"2001:db8:0:0:1:0:0:1", "2001:db8::1:0:0:1"},
	}
	for _, c := range cases {
		a, err := ParseIPAddress(c.in)
		if c.want == "" {
			if err == nil {
				t.Errorf("ParseIPAddress(%q) = %v, want invalid", c.in, a)
			} else if !errors.Is(err, ErrInvalidIP) {
				t.Errorf("ParseIPAddress(%q) error %v, want ErrInvalidIP", c.in, err)
			}
			continue
		}
		if got := FormatIPAddress(a); err != nil || got != c.want {
			t.Errorf("ParseIPAddress(%q) = %q, %v; want %s", c.in, got, err, c.want)
		}
		// The stored string parses back to the same address.
		if b, err := ParseIPAddress(FormatIPAddress(a)); err != nil || b != a {
			t.Errorf("%q does not round-trip: %v %v", c.in, b, err)
		}
	}
	if ErrInvalidIP.Error() != "Invalid IP addres! Please enter an ip address in the following form: 192.168.1.10" {
		t.Errorf("message %q", ErrInvalidIP.Error())
	}
}

func TestDeviceModelNamesMatchEnum(t *testing.T) {
	src := readFixture(t, "ACENetwork/ICUNetwork/ICUDeviceModel.cs")
	re := regexp.MustCompile(`(?m)^\s+([A-Za-z0-9_]+),?\s*$`)
	var names []string
	for _, m := range re.FindAllStringSubmatch(src, -1) {
		names = append(names, m[1])
	}
	if !reflect.DeepEqual(names, DeviceModelNames) {
		t.Fatalf("DeviceModelNames differs from ICUDeviceModel.cs (%d vs %d)", len(DeviceModelNames), len(names))
	}
	if DeviceModelNames[0] != "Unknown" || !IsDeviceModelName("Eve_Mini") || IsDeviceModelName("eve_mini") || IsDeviceModelName("") {
		t.Error("IsDeviceModelName")
	}
}

func TestManualIPDefaults(t *testing.T) {
	st := openTemp(t)
	want := ManualIPForm{Port: 443, NumberOfSockets: 1, LoginRequired: true}
	if got := st.ManualIPDefaults(); got != want {
		t.Errorf("fresh %+v", got)
	}
	// LastManual* used only when LastManualIPAddress is set; login always on.
	_ = st.Update(func(s *Settings) {
		s.LastManualIPPort = 8443
		s.LastManualModelType = "Eve_Mini"
	})
	if got := st.ManualIPDefaults(); got != want {
		t.Errorf("no address %+v", got)
	}
	_ = st.Update(func(s *Settings) {
		s.LastManualIPAddress = "10.0.0.5"
		s.LastManualNumberOfSockets = 2
		s.LastManualLoginRequired = false
	})
	want = ManualIPForm{IPAddress: "10.0.0.5", Port: 8443, ModelType: "Eve_Mini", NumberOfSockets: 2, LoginRequired: true}
	if got := st.ManualIPDefaults(); got != want {
		t.Errorf("stored %+v", got)
	}
	// Out-of-range stored values are not clamped (the WPF SpinButton setter
	// only rounds); an unknown combo text selects nothing.
	_ = st.Update(func(s *Settings) {
		s.LastManualIPPort = 70000
		s.LastManualNumberOfSockets = 5
		s.LastManualModelType = "NoSuchModel"
	})
	want = ManualIPForm{IPAddress: "10.0.0.5", Port: 70000, NumberOfSockets: 5, LoginRequired: true}
	if got := st.ManualIPDefaults(); got != want {
		t.Errorf("out of range %+v", got)
	}
}

func TestValidateManualIP(t *testing.T) {
	cases := []struct {
		f    ManualIPForm
		want ManualDevice
		err  error
	}{
		{ManualIPForm{"192.168.1.10", 443, "Eve_Single", 1, true}, ManualDevice{"192.168.1.10", 443, "Eve_Single", 1, true}, nil},
		{ManualIPForm{"10.1", 80, "Twin_4_0", 2, false}, ManualDevice{"10.0.0.1", 80, "Twin_4_0", 2, false}, nil},
		// Values are passed through as the SpinButtons hold them (no clamping).
		{ManualIPForm{"1.2.3.4", -1, "Lolo3", 0, true}, ManualDevice{"1.2.3.4", -1, "Lolo3", 0, true}, nil},
		{ManualIPForm{"1.2.3.4", 99999, "Lolo3", 9, true}, ManualDevice{"1.2.3.4", 99999, "Lolo3", 9, true}, nil},
		{ManualIPForm{"[::1.2.3.4]:443", 443, "Lolo3", 1, true}, ManualDevice{"::1.2.3.4", 443, "Lolo3", 1, true}, nil},
		{ManualIPForm{"1.2.3", 443, "", 1, true}, ManualDevice{}, ErrNoModelType},        // address checked first, then model
		{ManualIPForm{"bad", 443, "", 1, true}, ManualDevice{}, ErrInvalidIP},            // address error wins
		{ManualIPForm{"1.2.3.4", 443, "Bogus", 1, true}, ManualDevice{}, ErrNoModelType}, // not a combo item
	}
	for i, c := range cases {
		got, err := ValidateManualIP(c.f)
		if !errors.Is(err, c.err) || got != c.want {
			t.Errorf("case %d: %+v, %v; want %+v, %v", i, got, err, c.want, c.err)
		}
	}
}

func TestAcceptManualIP(t *testing.T) {
	st := openTemp(t)
	if _, err := st.AcceptManualIP(ManualIPForm{IPAddress: "nope", ModelType: "Lolo3"}); !errors.Is(err, ErrInvalidIP) {
		t.Fatalf("invalid: %v", err)
	}
	if st.Get().LastManualIPAddress != "" {
		t.Fatal("saved on validation error")
	}
	d, err := st.AcceptManualIP(ManualIPForm{IPAddress: "192.168.001.010", Port: 444, ModelType: "Eve_Mini", NumberOfSockets: 2, LoginRequired: false})
	if err != nil {
		t.Fatal(err)
	}
	if d != (ManualDevice{"192.168.1.8", 444, "Eve_Mini", 2, false}) {
		t.Errorf("device %+v", d)
	}
	st2, _ := Open(st.Dir())
	s := st2.Get()
	if s.LastManualIPAddress != "192.168.1.8" || s.LastManualIPPort != 444 || s.LastManualNumberOfSockets != 2 ||
		s.LastManualLoginRequired || s.LastManualModelType != "Eve_Mini" || s.LastManualHostname != "" {
		t.Errorf("persisted %+v", s)
	}
	if f := st2.ManualIPDefaults(); f != (ManualIPForm{"192.168.1.8", 444, "Eve_Mini", 2, true}) {
		t.Errorf("reopened defaults %+v", f)
	}
}

func TestManualDevices(t *testing.T) {
	st := openTemp(t)
	a := ManualDevice{"192.168.1.10", 443, "Eve_Mini", 1, true}
	if err := st.AddManualDevice(a); err != nil {
		t.Fatal(err)
	}
	// Same address on another port is still "already present" (MainWindow
	// compares Address strings only).
	err := st.AddManualDevice(ManualDevice{"192.168.1.10", 8443, "Lolo3", 2, false})
	var dup *DeviceAlreadyPresentError
	if !errors.As(err, &dup) || err.Error() != "Device with IP: 192.168.1.10 is already present in the overview and cannot be added." {
		t.Fatalf("dup: %v", err)
	}
	// Discovered devices passed by the caller count as well.
	if err := st.AddManualDevice(ManualDevice{IPAddress: "10.0.0.2", Port: 443}, "10.0.0.9", "10.0.0.2"); !errors.As(err, &dup) {
		t.Fatalf("discovered dup: %v", err)
	}
	// The pre-login check gives the same answers and writes nothing.
	before, _ := os.ReadFile(st.Path())
	for _, c := range []struct {
		ip      string
		present []string
		dup     bool
	}{
		{"192.168.1.10", nil, true},
		{"10.0.0.2", []string{"10.0.0.9", "10.0.0.2"}, true},
		{"10.0.0.3", []string{"10.0.0.9"}, false},
		{"192.168.1.1", nil, false},
	} {
		err := st.CheckManualDuplicate(c.ip, c.present...)
		if got := errors.As(err, &dup); got != c.dup || (err != nil && !got) {
			t.Errorf("CheckManualDuplicate(%s, %v) = %v", c.ip, c.present, err)
		}
	}
	if after, _ := os.ReadFile(st.Path()); string(after) != string(before) {
		t.Error("CheckManualDuplicate wrote the settings file")
	}
	if len(st.ManualDevices()) != 1 {
		t.Errorf("CheckManualDuplicate added a device: %+v", st.ManualDevices())
	}
	b := ManualDevice{"10.0.0.3", 443, "Twin_4_0", 2, false}
	if err := st.AddManualDevice(b, "10.0.0.9"); err != nil {
		t.Fatal(err)
	}
	st2, _ := Open(st.Dir())
	if got := st2.ManualDevices(); !reflect.DeepEqual(got, []ManualDevice{a, b}) {
		t.Errorf("persisted %+v", got)
	}
	if ok, err := st.RemoveManualDevice("192.168.1.10"); !ok || err != nil {
		t.Errorf("remove: %v %v", ok, err)
	}
	if ok, _ := st.RemoveManualDevice("192.168.1.10"); ok {
		t.Error("removed twice")
	}
	st3, _ := Open(st.Dir())
	if got := st3.ManualDevices(); !reflect.DeepEqual(got, []ManualDevice{b}) {
		t.Errorf("after remove %+v", got)
	}
	if addr, err := b.Addr(); err != nil || addr != netip.MustParseAddr("10.0.0.3") {
		t.Errorf("Addr %v %v", addr, err)
	}
	if MsgManualDeviceAdded("ACE1", "1.2.3.4") != "Device 'ACE1' with IP: 1.2.3.4 sucessfully added to the overview." ||
		MsgManualDeviceNotAdded("1.2.3.4") != "Device with IP: 1.2.3.4 could not be added to the overview." {
		t.Error("messages")
	}
}

func TestLinkLocalSearch(t *testing.T) {
	for in, want := range map[string]bool{
		"169.254.1.2": true, "169.254.255.255": true, "169.253.1.1": false, "192.168.1.1": false, "fe80::1": false,
	} {
		if got := IsLinkLocalSearchCandidate(netip.MustParseAddr(in)); got != want {
			t.Errorf("%s: %v", in, got)
		}
	}
	if MsgLinkLocalFound("169.254.0.1") != "A new device is found on ip: 169.254.0.1 do you want to add that manually?" {
		t.Error("message")
	}
	if DlgManualIPTitle != "Manual IP address ACE Service Installer" {
		t.Error("title")
	}
}

func TestProfiles(t *testing.T) {
	st := openTemp(t)
	if _, err := st.SaveProfile(Profile{Name: " ", IP: "1.2.3.4", Port: 443}); !errors.Is(err, ErrProfileName) {
		t.Errorf("empty name: %v", err)
	}
	if _, err := st.SaveProfile(Profile{Name: "a", IP: "x", Port: 443}); !errors.Is(err, ErrInvalidIP) {
		t.Errorf("bad ip: %v", err)
	}
	for _, port := range []int{0, 65536} {
		if _, err := st.SaveProfile(Profile{Name: "a", IP: "1.2.3.4", Port: port}); !errors.Is(err, ErrProfilePort) {
			t.Errorf("port %d: %v", port, err)
		}
	}
	p, err := st.SaveProfile(Profile{Name: " Site A ", IP: "010.0.0.1", Port: 443, User: "admin"})
	if err != nil || p != (Profile{"Site A", "8.0.0.1", 443, "admin"}) {
		t.Fatalf("save: %+v %v", p, err)
	}
	_, _ = st.SaveProfile(Profile{Name: "Site B", IP: "10.0.0.2", Port: 443})
	_, _ = st.SaveProfile(Profile{Name: "Site A", IP: "10.0.0.1", Port: 8443, User: "temp"}) // replace in place
	want := []Profile{{"Site A", "10.0.0.1", 8443, "temp"}, {"Site B", "10.0.0.2", 443, ""}}
	st2, _ := Open(st.Dir())
	if got := st2.Profiles(); !reflect.DeepEqual(got, want) {
		t.Errorf("profiles %+v", got)
	}
	if got, ok := st2.Profile("Site B"); !ok || got != want[1] {
		t.Errorf("lookup %+v %v", got, ok)
	}
	if ok, err := st.DeleteProfile("Site A"); !ok || err != nil {
		t.Errorf("delete %v %v", ok, err)
	}
	if ok, _ := st.DeleteProfile("Site A"); ok {
		t.Error("deleted twice")
	}
	if _, ok := st.LastDevice(); ok {
		t.Error("last device set")
	}
	if _, err := st.SetLastDevice(Profile{IP: "bad", Port: 443}); !errors.Is(err, ErrInvalidIP) {
		t.Errorf("last device bad ip: %v", err)
	}
	if _, err := st.SetLastDevice(Profile{IP: "10.0.0.7", Port: 443, User: "admin"}); err != nil {
		t.Fatal(err)
	}
	st3, _ := Open(st.Dir())
	if got, ok := st3.LastDevice(); !ok || got != (Profile{"", "10.0.0.7", 443, "admin"}) {
		t.Errorf("last device %+v %v", got, ok)
	}
	if len(st3.Profiles()) != 1 {
		t.Errorf("profiles after delete %+v", st3.Profiles())
	}
}
