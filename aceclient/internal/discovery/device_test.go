package discovery

import (
	"net/netip"
	"testing"
)

func ann(host string, port uint16, txt ...string) *ServiceAnnouncement {
	return &ServiceAnnouncement{
		Instance:  "x",
		Type:      "_alfen._tcp",
		Domain:    "local.",
		Hostname:  host,
		Port:      port,
		Addresses: []netip.Addr{netip.MustParseAddr("192.168.1.10")},
		Txt:       txt,
	}
}

func ver(major, minor, build, rev int) *Version {
	return &Version{Major: major, Minor: minor, Build: build, Revision: rev}
}

func sameVersion(a, b *Version) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}

func TestReInitializeTXT(t *testing.T) {
	ip := netip.MustParseAddr("192.168.1.10")
	tests := []struct {
		name  string
		txt   []string
		check func(t *testing.T, d *Device)
	}{
		{"identity keeps '=' in value", []string{"identity=ACE=01"}, func(t *testing.T, d *Device) {
			if d.Identity != "ACE=01" || d.Name != "ACE=01" {
				t.Fatalf("identity %q name %q", d.Identity, d.Name)
			}
		}},
		{"key is case-insensitive, entries without '=' skipped", []string{"IDENTITY=A", "garbage", "scnnetwork"}, func(t *testing.T, d *Device) {
			if d.Identity != "A" || d.SCNNetwork != "" {
				t.Fatalf("%+v", d)
			}
		}},
		{"scnnetwork trimmed, only first '=' part", []string{"scnnetwork=  Site 1 =x"}, func(t *testing.T, d *Device) {
			if d.SCNNetwork != "Site 1" || !d.HasSCNNetwork() {
				t.Fatalf("scn %q", d.SCNNetwork)
			}
		}},
		{"type with three parts", []string{"type=2.1.3"}, func(t *testing.T, d *Device) {
			if d.NumberOfSockets != 2 || d.SocketTypes != [2]int{1, 3} {
				t.Fatalf("sockets %d types %v", d.NumberOfSockets, d.SocketTypes)
			}
		}},
		{"type needs more than two parts", []string{"type=2.1"}, func(t *testing.T, d *Device) {
			if d.NumberOfSockets != 1 || d.SocketTypes != [2]int{0, 0} {
				t.Fatalf("sockets %d types %v", d.NumberOfSockets, d.SocketTypes)
			}
		}},
		{"type parts parsed independently (int.TryParse)", []string{"type= 2 .x.-1.9"}, func(t *testing.T, d *Device) {
			if d.NumberOfSockets != 2 || d.SocketTypes != [2]int{0, -1} {
				t.Fatalf("sockets %d types %v", d.NumberOfSockets, d.SocketTypes)
			}
		}},
		{"fwversion with build suffix", []string{"fwversion= 5.2.1-4312 "}, func(t *testing.T, d *Device) {
			if !sameVersion(d.FirmwareVersion, ver(5, 2, 1, -1)) || d.RawFirmwareVersion != " 5.2.1-4312 " {
				t.Fatalf("fw %v raw %q", d.FirmwareVersion, d.RawFirmwareVersion)
			}
		}},
		{"fwversion x -> 99", []string{"fwversion=4.X.0-test"}, func(t *testing.T, d *Device) {
			if !sameVersion(d.FirmwareVersion, ver(4, 99, 0, -1)) {
				t.Fatalf("fw %v", d.FirmwareVersion)
			}
		}},
		{"fwversion unparsable -> null", []string{"fwversion=dev-build"}, func(t *testing.T, d *Device) {
			if d.FirmwareVersion != nil || d.RawFirmwareVersion != "dev-build" {
				t.Fatalf("fw %v", d.FirmwareVersion)
			}
		}},
		{"euaenabled=1", []string{"euaenabled=1"}, func(t *testing.T, d *Device) {
			if d.EndUserAccessType != EndUserAccessEnabled {
				t.Fatal(d.EndUserAccessType)
			}
		}},
		{"euaenabled=0", []string{"euaenabled=0"}, func(t *testing.T, d *Device) {
			if d.EndUserAccessType != EndUserAccessDisabled {
				t.Fatal(d.EndUserAccessType)
			}
		}},
		{"enabled then configured", []string{"euaenabled=1", "euaconfigured=1"}, func(t *testing.T, d *Device) {
			if d.EndUserAccessType != EndUserAccessConfigured {
				t.Fatal(d.EndUserAccessType)
			}
		}},
		{"configured then enabled downgrades (C# quirk)", []string{"euaconfigured=1", "euaenabled=1"}, func(t *testing.T, d *Device) {
			if d.EndUserAccessType != EndUserAccessDisabled {
				t.Fatal(d.EndUserAccessType)
			}
		}},
		{"no identity -> host name", []string{"type=1.0.0"}, func(t *testing.T, d *Device) {
			if d.Identity != host1 || d.Name != "" {
				t.Fatalf("identity %q name %q", d.Identity, d.Name)
			}
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := newLanDevice(ip, 443, ann(host1, 443, tc.txt...), false)
			if !d.Discovered || d.HostName != host1 || d.SerialNumber != "1234567" || !d.AllowObjectIDUpdate {
				t.Fatalf("discovered device %+v", d)
			}
			tc.check(t, d)
		})
	}
}

func TestReInitializeWithoutTXT(t *testing.T) {
	ip := netip.MustParseAddr("10.0.0.2")
	for _, a := range []*ServiceAnnouncement{nil, ann(host1, 443)} {
		d := newLanDevice(ip, 80, a, true)
		if d.Discovered || d.HostName != "" || d.Identity != "" || d.SerialNumber != "" || !d.IsManuallyAdded {
			t.Fatalf("%+v", d)
		}
		if d.NumberOfSockets != 1 || d.Protocol() != "http" || d.Address() != "10.0.0.2" {
			t.Fatalf("%+v", d)
		}
	}
	// Host name without '-' leaves SerialNumber empty.
	d := newLanDevice(ip, 443, ann("lolo3", 443, "identity=L"), false)
	if d.SerialNumber != "" || d.HostName != "lolo3" || d.Protocol() != "https" || !d.IsHTTPS() {
		t.Fatalf("%+v", d)
	}
}

func TestReInitializeKeepsAndResets(t *testing.T) {
	ip := netip.MustParseAddr("192.168.1.10")
	d := newLanDevice(ip, 443, ann(host1, 443, "identity=ACE", "scnnetwork=S", "type=2.1.1", "fwversion=5.1.0-1"), false)

	// Same endpoint, fewer TXT keys: identity/SCN/firmware are kept, sockets reset.
	changed := d.reInitialize(ip, 443, ann(host1, 443, "euaenabled=0"), false)
	if changed {
		t.Fatal("endpoint unchanged")
	}
	if d.Identity != "ACE" || d.SCNNetwork != "S" || !sameVersion(d.FirmwareVersion, ver(5, 1, 0, -1)) {
		t.Fatalf("lost state: %+v", d)
	}
	if d.NumberOfSockets != 1 || d.SocketTypes != [2]int{} {
		t.Fatalf("sockets not reset: %+v", d)
	}
	if !d.AllowObjectIDUpdate {
		t.Fatal("AllowObjectIDUpdate only changes with the endpoint")
	}

	// New port: session reset, AllowObjectIDUpdate = newDevice (false).
	if !d.reInitialize(ip, 80, ann(host1, 80, "identity=ACE"), false) {
		t.Fatal("port change must report endpoint change")
	}
	if d.AllowObjectIDUpdate || d.Port != 80 {
		t.Fatalf("%+v", d)
	}
	if !d.reInitialize(netip.MustParseAddr("192.168.1.11"), 80, ann(host1, 80, "identity=ACE"), false) {
		t.Fatal("address change must report endpoint change")
	}
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		in   string
		want *Version
	}{
		{"5.2", ver(5, 2, -1, -1)},
		{"5.2.1", ver(5, 2, 1, -1)},
		{"5.2.1.7", ver(5, 2, 1, 7)},
		{" 5 . +2 ", ver(5, 2, -1, -1)},
		{"4.99.0", ver(4, 99, 0, -1)},
		{"5", nil},
		{"1.2.3.4.5", nil},
		{"5.-1", nil},
		{"5.a", nil},
		{"5.", nil},
		{"", nil},
		{"5.2147483648", nil},
	}
	for _, tc := range tests {
		got, ok := ParseVersion(tc.in)
		if ok != (tc.want != nil) || !sameVersion(got, tc.want) {
			t.Fatalf("ParseVersion(%q) = %v, %v want %v", tc.in, got, ok, tc.want)
		}
	}
	if s := ver(5, 2, 1, -1).String(); s != "5.2.1" {
		t.Fatal(s)
	}
	if s := ver(5, 0, -1, -1).String(); s != "5.0" {
		t.Fatal(s)
	}
}

func TestEndUserAccessTypeDescription(t *testing.T) {
	want := map[EndUserAccessType]string{
		EndUserAccessNotAvailable: "NotAvailable",
		EndUserAccessDisabled:     "Disabled",
		EndUserAccessEnabled:      "Enabled (without PIN)",
		EndUserAccessConfigured:   "Enabled (with PIN)",
	}
	for v, s := range want {
		if v.Description() != s {
			t.Fatalf("%d: %q want %q", v, v.Description(), s)
		}
	}
}

func TestSetHostInfoAndUniquePassword(t *testing.T) {
	d := newLanDevice(netip.MustParseAddr("10.0.0.3"), 443, nil, true)
	d.SetHostInfo("NG910_60023", 2)
	if d.HostName != "NG910_60023" || d.NumberOfSockets != 2 || d.DisplayNameLine2() != "NG910_60023" {
		t.Fatalf("%+v", d)
	}
	d.SetUniquePasswordRequired(false)
	if d.FirmwareVersion != nil {
		t.Fatal("false is a no-op")
	}
	d.SetUniquePasswordRequired(true)
	if !sameVersion(d.FirmwareVersion, ver(5, 0, 0, -1)) {
		t.Fatalf("fw %v", d.FirmwareVersion)
	}
}

func TestLastDashPart(t *testing.T) {
	for in, want := range map[string]string{"ng910-60023-1234567": "1234567", "abc": "abc", "abc-": "", "": ""} {
		if got := lastDashPart(in); got != want {
			t.Fatalf("%q -> %q want %q", in, got, want)
		}
	}
}
