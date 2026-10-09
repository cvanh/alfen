package discovery

import (
	"fmt"
	"net/netip"
	"strconv"
	"strings"
)

// DeviceKey identifies a registry entry for its lifetime (the C# compares
// ICULanDevice object references; Go hands out value snapshots instead).
type DeviceKey uint64

// EndUserAccessType ports ICUNetwork.EndUserAccessType.
type EndUserAccessType int

const (
	EndUserAccessNotAvailable EndUserAccessType = 0
	EndUserAccessDisabled     EndUserAccessType = 1
	EndUserAccessEnabled      EndUserAccessType = 2
	EndUserAccessConfigured   EndUserAccessType = 3
)

// String returns the C# enum member name.
func (t EndUserAccessType) String() string {
	switch t {
	case EndUserAccessNotAvailable:
		return "NotAvailable"
	case EndUserAccessDisabled:
		return "Disabled"
	case EndUserAccessEnabled:
		return "Enabled"
	case EndUserAccessConfigured:
		return "Configured"
	}
	return strconv.Itoa(int(t))
}

// Description ports EnumExtensions.GetEnumDescription: the [Description]
// attribute, else the member name.
func (t EndUserAccessType) Description() string {
	switch t {
	case EndUserAccessDisabled:
		return "Disabled"
	case EndUserAccessEnabled:
		return "Enabled (without PIN)"
	case EndUserAccessConfigured:
		return "Enabled (with PIN)"
	}
	return t.String()
}

// Version ports System.Version as used for ICULanDevice.m_vFirmwareVersion.
// Build and Revision are -1 when the source string did not have them.
type Version struct {
	Major, Minor, Build, Revision int
}

// String ports Version.ToString().
func (v Version) String() string {
	s := fmt.Sprintf("%d.%d", v.Major, v.Minor)
	if v.Build >= 0 {
		s += fmt.Sprintf(".%d", v.Build)
		if v.Revision >= 0 {
			s += fmt.Sprintf(".%d", v.Revision)
		}
	}
	return s
}

// ParseVersion ports System.Version.TryParse (.NET Framework 4.8): two to
// four '.'-separated components, each an Int32 in NumberStyles.Integer
// (surrounding ASCII white space and a leading sign allowed) that is >= 0.
func ParseVersion(s string) (*Version, bool) {
	parts := strings.Split(s, ".")
	if len(parts) < 2 || len(parts) > 4 {
		return nil, false
	}
	vals := []int{-1, -1, -1, -1}
	for i, p := range parts {
		n, ok := netTryParseInt(p)
		if !ok || n < 0 {
			return nil, false
		}
		vals[i] = n
	}
	return &Version{Major: vals[0], Minor: vals[1], Build: vals[2], Revision: vals[3]}, true
}

// netTryParseInt ports int.TryParse(string) (NumberStyles.Integer): optional
// leading/trailing white space (U+0009..U+000D, U+0020), optional sign, decimal
// digits, Int32 range.
func netTryParseInt(s string) (int, bool) {
	s = strings.Trim(s, "\t\n\v\f\r ")
	if s == "" {
		return 0, false
	}
	n, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, false
	}
	return int(n), true
}

// Device is the discovery-owned part of ICUNetwork.ICULanDevice/ICUDevice:
// everything LANConnection and ReInitialize set from an mDNS announcement or a
// manual add. Session state (tokens, properties, login) lives elsewhere.
// Values returned by LANConnection are snapshots.
type Device struct {
	Key DeviceKey

	IPAddress netip.Addr // ICULanDevice.IPAddress (Announcement.Addresses.First())
	Port      int        // ICULanDevice.Port (SRV port; 443 => https)

	HostName     string // announcement Hostname ("" when not discovered)
	Identity     string // TXT identity=, else HostName
	Name         string // ICUDevice.Name (= Identity when discovered)
	SerialNumber string // last '-' part of the host name
	SCNNetwork   string // TXT scnnetwork=

	NumberOfSockets int    // TXT type=<n>.x.y (1 by default)
	SocketTypes     [2]int // TXT type=n.<t0>.<t1>

	RawFirmwareVersion string   // TXT fwversion= (untrimmed)
	FirmwareVersion    *Version // parsed fwversion; nil when absent/unparsable. Never mutated in place.

	EndUserAccessType EndUserAccessType // TXT euaenabled= / euaconfigured=

	Discovered          bool // ReInitialize saw TXT records
	IsManuallyAdded     bool
	AllowObjectIDUpdate bool
	IsRebooting         bool // owned by the device-session layer; blocks removal
	HasCentralMeter     bool // set by AddManualDevice
	HasSmartMeter       bool // set by AddManualDevice

	Announcement *ServiceAnnouncement // last announcement (nil for manual devices). Never mutated in place.
}

// IsHTTPS ports ICULanDevice.IsHTTPS (Port == 443).
func (d Device) IsHTTPS() bool { return d.Port == 443 }

// Protocol ports ICULanDevice.Protocol ("https" or "http").
func (d Device) Protocol() string {
	if !d.IsHTTPS() {
		return "http"
	}
	return "https"
}

// Address ports ICULanDevice.Address ($"{IPAddress}").
func (d Device) Address() string {
	if !d.IPAddress.IsValid() {
		return ""
	}
	return d.IPAddress.String()
}

// Identification ports ICULanDevice.Identification (=> Identity).
func (d Device) Identification() string { return d.Identity }

// DisplayNameLine2 ports ICULanDevice.DisplayNameLine2 (=> HostName).
func (d Device) DisplayNameLine2() string { return d.HostName }

// HasSCNNetwork ports ICULanDevice.HasSCNNetwork.
func (d Device) HasSCNNetwork() bool { return d.SCNNetwork != "" }

// OnDeviceRemoved ports ICULanDevice.OnDeviceRemoved: a rebooting device is
// never dropped from the registry.
func (d Device) OnDeviceRemoved() bool { return !d.IsRebooting }

// newLanDevice ports the ICULanDevice constructor (discovery part):
// AllowObjectIDUpdate = true, ReInitialize(newDevice: true), IsManuallyAdded.
func newLanDevice(addr netip.Addr, port int, ann *ServiceAnnouncement, manual bool) *Device {
	d := &Device{AllowObjectIDUpdate: true}
	d.reInitialize(addr, port, ann, true)
	d.IsManuallyAdded = manual
	return d
}

// SetHostInfo ports ICULanDevice.SetHostInfo.
func (d *Device) SetHostInfo(hostName string, numberOfSockets int) {
	d.HostName = hostName
	d.NumberOfSockets = numberOfSockets
}

// SetUniquePasswordRequired ports the IsUniquePasswordRequired setter with
// value true: it pins the firmware version to 5.0.0 (false is a no-op).
func (d *Device) SetUniquePasswordRequired(v bool) {
	if v {
		d.FirmwareVersion = &Version{Major: 5, Minor: 0, Build: 0, Revision: -1}
	}
}

// reInitialize ports ICULanDevice.ReInitialize up to (not including) the
// IsRebooting re-login tail. It reports whether the endpoint changed, i.e.
// whether the C# reset the session (IsLoggedIn = false,
// IsDefaultCategoryCollected = false, new log).
//
// Not ported here: NumberOfFeederCables (depends on ICUDeviceModel) and the
// reboot completion check (session layer).
func (d *Device) reInitialize(addr netip.Addr, port int, ann *ServiceAnnouncement, newDevice bool) (endpointChanged bool) {
	if newDevice || d.IPAddress != addr || d.Port != port {
		d.IPAddress = addr
		d.Port = port
		d.AllowObjectIDUpdate = newDevice
		endpointChanged = true
	}
	d.Announcement = ann
	d.NumberOfSockets = 1
	d.SocketTypes = [2]int{0, 0}
	if ann != nil && len(ann.Txt) > 0 {
		for _, item := range ann.Txt {
			array := strings.Split(item, "=")
			if len(array) <= 1 {
				continue
			}
			switch strings.ToLower(array[0]) {
			case "identity":
				d.Identity = strings.Join(array[1:], "=")
			case "scnnetwork":
				d.SCNNetwork = strings.TrimSpace(array[1])
			case "type":
				array3 := strings.Split(array[1], ".")
				if len(array3) > 2 {
					if v, ok := netTryParseInt(array3[0]); ok {
						d.NumberOfSockets = v
					}
					if v, ok := netTryParseInt(array3[1]); ok {
						d.SocketTypes[0] = v
					}
					if v, ok := netTryParseInt(array3[2]); ok {
						d.SocketTypes[1] = v
					}
				}
			case "fwversion":
				array2 := strings.Split(strings.TrimSpace(array[1]), "-")
				d.RawFirmwareVersion = array[1]
				text := strings.ReplaceAll(strings.ToLower(array2[0]), "x", "99")
				// Version.TryParse assigns null to the out field on failure.
				d.FirmwareVersion, _ = ParseVersion(text)
			case "euaenabled":
				// Kept as in the C#: "euaenabled=1" after "euaconfigured=1"
				// downgrades Configured to Disabled.
				if array[1] == "1" && d.EndUserAccessType != EndUserAccessConfigured {
					d.EndUserAccessType = EndUserAccessEnabled
				} else {
					d.EndUserAccessType = EndUserAccessDisabled
				}
			case "euaconfigured":
				if array[1] == "1" {
					d.EndUserAccessType = EndUserAccessConfigured
				}
			}
		}
		if array4 := strings.Split(ann.Hostname, "-"); len(array4) > 1 {
			d.SerialNumber = array4[len(array4)-1]
		}
		d.Name = d.Identity
		d.Discovered = true
		d.HostName = ann.Hostname
	} else {
		d.Discovered = false
		d.HostName = ""
	}
	if d.Identity == "" {
		d.Identity = d.HostName
	}
	return endpointChanged
}

// lastDashPart ports s.Split('-').Last().
func lastDashPart(s string) string {
	if i := strings.LastIndexByte(s, '-'); i >= 0 {
		return s[i+1:]
	}
	return s
}
