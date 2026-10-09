package settings

import (
	"errors"
	"fmt"
	"net/netip"
	"slices"
	"strconv"
	"strings"
)

// DlgManualIP texts and limits (ACEServiceInstaller/ICUServiceInstaller/DlgManualIP.cs).
const (
	// DlgManualIPTitle is "Manual IP address " + AppProperties.AppName.
	DlgManualIPTitle = "Manual IP address " + AppName
	// LabelManualIPHeader is the bold header label (sic "a IP").
	LabelManualIPHeader = "Please enter a IP address."
	// Field labels, placeholder and the link-local search button caption.
	LabelManualIPAddress       = "IP address:"
	LabelManualPort            = "Port:"
	LabelManualModelType       = "Model type:"
	LabelManualNumberOfSockets = "Number of sockets:"
	LabelManualLoginRequired   = "Login required:"
	PlaceholderManualIPAddress = "xxx.xxx.xxx.xxx"
	LabelManualSearch          = "Search"

	// ManualPortMin..ManualPortDefault are the m_spbPort SpinButton limits.
	ManualPortMin     = 0
	ManualPortMax     = 65535
	ManualPortDefault = 443
	// ManualSocketsMin..ManualSocketsDefault are the m_spbNumberOfSockets limits.
	ManualSocketsMin     = 1
	ManualSocketsMax     = 2
	ManualSocketsDefault = 1

	// MsgInvalidIPTitle / MsgInvalidIPDetail are the MessageDialog.ShowError
	// texts for an unparsable address (sic "addres").
	MsgInvalidIPTitle  = "Invalid IP addres!"
	MsgInvalidIPDetail = "Please enter an ip address in the following form: 192.168.1.10"

	// MsgNoLinkLocalDevice is shown when the link-local search finds nothing.
	MsgNoLinkLocalDevice = "No device found on the local subnet"
)

// ValidationError is a DlgManualIP input error with the C# dialog texts.
//
// ports DlgManualIP.OnCommandActivated MessageDialog.ShowError (ACEServiceInstaller/ICUServiceInstaller/DlgManualIP.cs)
type ValidationError struct {
	Title  string
	Detail string
}

func (e *ValidationError) Error() string {
	if e.Detail == "" {
		return e.Title
	}
	return e.Title + " " + e.Detail
}

// ErrInvalidIP is the DlgManualIP "Invalid IP addres!" error.
var ErrInvalidIP = &ValidationError{Title: MsgInvalidIPTitle, Detail: MsgInvalidIPDetail}

// ErrNoModelType is returned when no model type is selected. The C# reads
// m_cmbType.SelectedItem.ToString() without a null check, so Ok with an empty
// combo throws a NullReferenceException (logged as unhandled; the dialog
// stays open, nothing is saved). Go reports it as an error instead.
var ErrNoModelType = errors.New("settings: no model type selected")

// ParseIPAddress ports System.Net.IPAddress.TryParse (.NET Framework 4.8) as
// DlgManualIP uses it. FormatIPAddress gives the string IPAddress.ToString()
// would print, which is what the C# stores.
//
// Strings without ':' go through IPv4AddressHelper.ParseNonCanonical, so the
// legacy inet_addr forms are accepted exactly like the C#: 1–4 dotted parts,
// each decimal, octal (leading 0) or hex (0x), the last part filling the
// remaining bytes ("10" = 0.0.0.10, "192.168.1" = 192.168.0.1,
// "010.0.0.1" = 8.0.0.1). Whitespace or trailing characters are rejected.
//
// Strings containing ':' are IPv6. The C# hands them to the OS
// (WSAStringToAddress, i.e. RtlIpv6StringToAddressEx), which accepts these
// forms: "addr", "addr%scope", "[addr]", "[addr%scope]" and the same in
// brackets followed by ":port". The port is checked (decimal 1..65535 here)
// and then dropped, because IPAddress keeps only the bytes and the scope ID.
// The scope must be a decimal number (no leading zero). Scope 0 means "no
// scope". Interface names such as "%eth0" are rejected. The address text
// itself is parsed with netip.ParseAddr.
//
// ports System.Net.IPAddress.TryParse / IPv4AddressHelper.ParseNonCanonical (.NET Framework 4.8 System.dll, as called from ACEServiceInstaller/ICUServiceInstaller/DlgManualIP.cs)
func ParseIPAddress(s string) (netip.Addr, error) {
	if strings.IndexByte(s, ':') >= 0 {
		a, ok := parseIPv6(s)
		if !ok {
			return netip.Addr{}, ErrInvalidIP
		}
		return a, nil
	}
	v, ok := parseIPv4NonCanonical(s)
	if !ok {
		return netip.Addr{}, ErrInvalidIP
	}
	return netip.AddrFrom4([4]byte{byte(v >> 24), byte(v >> 16), byte(v >> 8), byte(v)}), nil
}

// parseIPv6 handles the bracket, port and scope syntax of
// RtlIpv6StringToAddressEx and leaves the address text to netip.ParseAddr.
// A non-zero scope ID is kept as the decimal zone string.
func parseIPv6(s string) (netip.Addr, bool) {
	if strings.HasPrefix(s, "[") {
		end := strings.IndexByte(s, ']')
		if end < 0 {
			return netip.Addr{}, false
		}
		if rest := s[end+1:]; rest != "" {
			if rest[0] != ':' {
				return netip.Addr{}, false
			}
			port, ok := parseDecimalStrict(rest[1:])
			if !ok || port == 0 || port > 0xFFFF {
				return netip.Addr{}, false
			}
		}
		s = s[1:end]
	}
	text, zone, hasZone := strings.Cut(s, "%")
	var scope uint32
	if hasZone {
		v, ok := parseDecimalStrict(zone)
		if !ok {
			return netip.Addr{}, false
		}
		scope = v
	}
	a, err := netip.ParseAddr(text)
	if err != nil || !a.Is6() {
		return netip.Addr{}, false
	}
	if scope != 0 {
		a = a.WithZone(strconv.FormatUint(uint64(scope), 10))
	}
	return a, true
}

// parseDecimalStrict parses a non-empty run of decimal digits without a
// leading zero (except "0" itself) that fits a uint32.
func parseDecimalStrict(s string) (uint32, bool) {
	if s == "" || (len(s) > 1 && s[0] == '0') {
		return 0, false
	}
	var v uint64
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, false
		}
		v = v*10 + uint64(c-'0')
		if v > 0xFFFFFFFF {
			return 0, false
		}
	}
	return uint32(v), true
}

// FormatIPAddress ports IPAddress.ToString(). IPv4 addresses print as dotted
// decimal. IPv6 addresses are formatted the way Windows WSAAddressToString
// does (the .NET Framework 4.8 path), using the algorithm .NET reimplemented
// for it (IPAddressParser.IPv6AddressToString /
// IPv6AddressHelper.ShouldHaveIpv4Embedded):
//
//   - lower-case hex groups without leading zeros;
//   - the longest run (first one on a tie) of two or more zero groups becomes "::";
//   - the last 32 bits print as dotted IPv4 for IPv4-compatible (::a.b.c.d,
//     when group 6 is non-zero), IPv4-mapped (::ffff:a.b.c.d), SIIT
//     (::ffff:0:a.b.c.d) and ISATAP (…:0:5efe:a.b.c.d) addresses;
//   - a scope ID is appended as "%<n>".
//
// So "::1.2.3.4" stays "::1.2.3.4", where netip.Addr.String() would print
// "::102:304".
//
// ports System.Net.IPAddress.ToString (.NET Framework 4.8 System.dll, as called from ACEServiceInstaller/ICUServiceInstaller/DlgManualIP.cs)
func FormatIPAddress(a netip.Addr) string {
	if !a.Is6() {
		return a.String()
	}
	b := a.As16()
	var w [8]uint16
	for i := range w {
		w[i] = uint16(b[2*i])<<8 | uint16(b[2*i+1])
	}
	var sb strings.Builder
	if ipv6ShouldHaveIPv4Embedded(w) {
		appendIPv6Sections(&sb, w[:6])
		if !strings.HasSuffix(sb.String(), ":") {
			sb.WriteByte(':')
		}
		fmt.Fprintf(&sb, "%d.%d.%d.%d", b[12], b[13], b[14], b[15])
	} else {
		appendIPv6Sections(&sb, w[:])
	}
	if z := a.Zone(); z != "" {
		sb.WriteString("%" + z)
	}
	return sb.String()
}

// ipv6ShouldHaveIPv4Embedded ports IPv6AddressHelper.ShouldHaveIpv4Embedded.
func ipv6ShouldHaveIPv4Embedded(w [8]uint16) bool {
	if w[0] == 0 && w[1] == 0 && w[2] == 0 && w[3] == 0 && w[6] != 0 {
		if w[4] == 0 && (w[5] == 0 || w[5] == 0xFFFF) {
			return true // IPv4-compatible / IPv4-mapped
		}
		if w[4] == 0xFFFF && w[5] == 0 {
			return true // SIIT
		}
	}
	return w[4] == 0 && w[5] == 0x5EFE // ISATAP
}

// appendIPv6Sections ports IPAddressParser.AppendSections: hex groups joined
// by ':' with the longest run of two or more zero groups replaced by "::".
func appendIPv6Sections(sb *strings.Builder, w []uint16) {
	zeroStart, zeroEnd := -1, 0
	best, cur := 0, 0
	for i, v := range w {
		if v != 0 {
			cur = 0
			continue
		}
		cur++
		if cur > best {
			best = cur
			zeroStart = i - cur + 1
		}
	}
	if best > 1 {
		zeroEnd = zeroStart + best
	} else {
		zeroStart, zeroEnd = -1, 0
	}
	needColon := false
	if zeroStart >= 0 {
		for i := 0; i < zeroStart; i++ {
			if needColon {
				sb.WriteByte(':')
			}
			needColon = true
			sb.WriteString(strconv.FormatUint(uint64(w[i]), 16))
		}
		sb.WriteString("::")
		needColon = false
	}
	for i := zeroEnd; i < len(w); i++ {
		if needColon {
			sb.WriteByte(':')
		}
		needColon = true
		sb.WriteString(strconv.FormatUint(uint64(w[i]), 16))
	}
}

// parseIPv4NonCanonical ports IPv4AddressHelper.ParseNonCanonical(name, 0,
// ref end, notImplicitFile: true) plus IPAddress.InternalParse's
// "end == ipString.Length" check. It returns the address as a host-order
// uint32 (first octet in the top byte).
func parseIPv4NonCanonical(name string) (uint32, bool) {
	const (
		maxIPv4 = 0xFFFFFFFF
		decimal = 10
		octal   = 8
		hex     = 16
	)
	var parts [4]int64
	var currentValue int64
	atLeastOneChar := false
	dotCount := 0
	end := len(name)
	current := 0
	for ; current < end; current++ {
		ch := name[current]
		currentValue = 0
		numberBase := int64(decimal)
		if ch == '0' {
			numberBase = octal
			current++
			atLeastOneChar = true
			if current < end {
				ch = name[current]
				if ch == 'x' || ch == 'X' {
					numberBase = hex
					current++
					atLeastOneChar = false
				}
			}
		}
	digits:
		for ; current < end; current++ {
			ch = name[current]
			var digit int64
			switch {
			case (numberBase == decimal || numberBase == hex) && '0' <= ch && ch <= '9':
				digit = int64(ch - '0')
			case numberBase == octal && '0' <= ch && ch <= '7':
				digit = int64(ch - '0')
			case numberBase == hex && 'a' <= ch && ch <= 'f':
				digit = int64(ch-'a') + 10
			case numberBase == hex && 'A' <= ch && ch <= 'F':
				digit = int64(ch-'A') + 10
			default:
				break digits // invalid character or terminator
			}
			currentValue = currentValue*numberBase + digit
			if currentValue > maxIPv4 {
				return 0, false
			}
			atLeastOneChar = true
		}
		if current < end && name[current] == '.' {
			if dotCount >= 3 || !atLeastOneChar || currentValue > 0xFF {
				return 0, false
			}
			parts[dotCount] = currentValue
			dotCount++
			atLeastOneChar = false
			continue
		}
		break
	}
	if !atLeastOneChar {
		return 0, false
	}
	if current < end {
		// '/', '\\', ':', '?', '#' terminate the address in ParseNonCanonical,
		// but InternalParse then rejects the string because end != Length;
		// any other character is invalid outright. Either way: not an address.
		return 0, false
	}
	parts[dotCount] = currentValue
	var v int64
	switch dotCount {
	case 0:
		if parts[0] > maxIPv4 {
			return 0, false
		}
		v = parts[0]
	case 1:
		if parts[1] > 0xffffff {
			return 0, false
		}
		v = parts[0]<<24 | parts[1]&0xffffff
	case 2:
		if parts[2] > 0xffff {
			return 0, false
		}
		v = parts[0]<<24 | (parts[1]&0xff)<<16 | parts[2]&0xffff
	case 3:
		if parts[3] > 0xff {
			return 0, false
		}
		v = parts[0]<<24 | (parts[1]&0xff)<<16 | (parts[2]&0xff)<<8 | parts[3]&0xff
	default:
		return 0, false
	}
	return uint32(v), true
}

// IsLinkLocalSearchCandidate ports the filter of DlgManualIP.OnSearchLinkLocalClicked:
// GetAddressBytes()[0] == 169 && GetAddressBytes()[1] == 254. The ARP table
// enumeration (ArpList.GetAllDevicesOnLAN / IpHlpApi) is not ported; callers
// feed addresses from another source.
//
// ports DlgManualIP.OnSearchLinkLocalClicked (ACEServiceInstaller/ICUServiceInstaller/DlgManualIP.cs)
func IsLinkLocalSearchCandidate(a netip.Addr) bool {
	b := a.AsSlice()
	return len(b) >= 2 && b[0] == 169 && b[1] == 254
}

// MsgLinkLocalFound is the question OnSearchLinkLocalClicked asks per
// candidate (Yes fills the address field and stops the search).
//
// ports DlgManualIP.OnSearchLinkLocalClicked (ACEServiceInstaller/ICUServiceInstaller/DlgManualIP.cs)
func MsgLinkLocalFound(ip string) string {
	return fmt.Sprintf("A new device is found on ip: %s do you want to add that manually?", ip)
}

// ManualIPForm is the DlgManualIP input state.
//
// ports DlgManualIP widgets (ACEServiceInstaller/ICUServiceInstaller/DlgManualIP.cs)
type ManualIPForm struct {
	IPAddress       string // m_txtIPAddress.Text
	Port            int    // m_spbPort.Value
	ModelType       string // m_cmbType selection (a DeviceModelNames entry; "" = none)
	NumberOfSockets int    // m_spbNumberOfSockets.Value
	LoginRequired   bool   // m_chkLogin
}

// ManualDevice is an accepted DlgManualIP entry: the arguments MainWindow
// passes to LANConnection.AddManualDevice(IPAddress, Port, Hostname,
// NumberOfSockets, LoginRequired). Hostname is the model-type name (it ends
// up in ICULanDevice.SetHostInfo); LoginRequired sets IsUniquePasswordRequired.
//
// ports DlgManualIP result properties (ACEServiceInstaller/ICUServiceInstaller/DlgManualIP.cs)
type ManualDevice struct {
	IPAddress       string `json:"IPAddress"` // IPAddress.ToString() form
	Port            int    `json:"Port"`
	Hostname        string `json:"Hostname"`
	NumberOfSockets int    `json:"NumberOfSockets"`
	LoginRequired   bool   `json:"LoginRequired"`
}

// Addr parses d.IPAddress.
func (d ManualDevice) Addr() (netip.Addr, error) { return ParseIPAddress(d.IPAddress) }

// ManualIPDefaults ports the DlgManualIP constructor: port 443, 1 socket, no
// model type, "Login required" ticked. When Settings.LastManualIPAddress is
// non-empty, the address, port, socket count and model type come from the
// LastManual* settings. A model name that is not a combo item selects nothing.
//
// Port and socket count are taken as stored, without clamping. The Xwt.WPF
// SpinButton (WindowsSpinButton) setter only rounds the value; it clamps only
// text the user types. So an out-of-range stored value (e.g. a hand-edited
// file) is shown and later returned by `(int)m_spbPort.Value` as is.
//
// The login box is ticked unconditionally: LastManualLoginRequired is never read.
//
// ports DlgManualIP..ctor (ACEServiceInstaller/ICUServiceInstaller/DlgManualIP.cs)
func (st *Store) ManualIPDefaults() ManualIPForm {
	st.mu.RLock()
	defer st.mu.RUnlock()
	f := ManualIPForm{Port: ManualPortDefault, NumberOfSockets: ManualSocketsDefault}
	if st.s.LastManualIPAddress != "" {
		f.IPAddress = st.s.LastManualIPAddress
		f.Port = st.s.LastManualIPPort
		f.NumberOfSockets = st.s.LastManualNumberOfSockets
		if IsDeviceModelName(st.s.LastManualModelType) {
			f.ModelType = st.s.LastManualModelType
		}
	}
	f.LoginRequired = true
	return f
}

// ValidateManualIP ports the validation in DlgManualIP.OnCommandActivated(Ok).
// Port and NumberOfSockets are used as the SpinButtons hold them
// (`(int)m_spbPort.Value`), without clamping. The view must clamp text the
// user types into [ManualPortMin, ManualPortMax] and
// [ManualSocketsMin, ManualSocketsMax], as WindowsSpinButton.parseTextBox
// does (unparsable text becomes the minimum). Pre-filled values are not
// clamped (see ManualIPDefaults).
//
// The address must pass IPAddress.TryParse (else ErrInvalidIP, shown as
// ShowError(title, detail) with the dialog left open), and a model type must
// be selected (else ErrNoModelType). The returned device carries the
// normalised address string (IPAddress.ToString(), see FormatIPAddress).
//
// ports DlgManualIP.OnCommandActivated (ACEServiceInstaller/ICUServiceInstaller/DlgManualIP.cs)
func ValidateManualIP(f ManualIPForm) (ManualDevice, error) {
	addr, err := ParseIPAddress(f.IPAddress)
	if err != nil {
		return ManualDevice{}, err
	}
	if !IsDeviceModelName(f.ModelType) {
		return ManualDevice{}, ErrNoModelType
	}
	return ManualDevice{
		IPAddress:       FormatIPAddress(addr),
		Port:            f.Port,
		Hostname:        f.ModelType,
		NumberOfSockets: f.NumberOfSockets,
		LoginRequired:   f.LoginRequired,
	}, nil
}

// AcceptManualIP ports DlgManualIP.OnCommandActivated(Ok) as a whole:
// ValidateManualIP, then LastManualIPAddress / LastManualIPPort /
// LastManualNumberOfSockets / LastManualLoginRequired / LastManualModelType are
// stored and saved. On a validation error nothing is saved. Returns the
// device for AddManualDevice (the save error, if any, is returned with it).
//
// ports DlgManualIP.OnCommandActivated (ACEServiceInstaller/ICUServiceInstaller/DlgManualIP.cs)
func (st *Store) AcceptManualIP(f ManualIPForm) (ManualDevice, error) {
	d, err := ValidateManualIP(f)
	if err != nil {
		return ManualDevice{}, err
	}
	err = st.Update(func(s *Settings) {
		s.LastManualIPAddress = d.IPAddress
		s.LastManualIPPort = d.Port
		s.LastManualNumberOfSockets = d.NumberOfSockets
		s.LastManualLoginRequired = d.LoginRequired
		s.LastManualModelType = d.Hostname
	})
	return d, err
}

// MsgDeviceAlreadyPresent is MainWindow.OnAddManualDeviceClicked's refusal.
//
// ports MainWindow.OnAddManualDeviceClicked (ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs)
func MsgDeviceAlreadyPresent(ip string) string {
	return fmt.Sprintf("Device with IP: %s is already present in the overview and cannot be added.", ip)
}

// MsgManualDeviceAdded is shown after a successful add (sic "sucessfully");
// identification is the logged-in device's ICULanDevice.Identification.
//
// ports MainWindow.OnAddManualDeviceClicked (ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs)
func MsgManualDeviceAdded(identification, ip string) string {
	return fmt.Sprintf("Device '%s' with IP: %s sucessfully added to the overview.", identification, ip)
}

// MsgManualDeviceNotAdded is shown when AddManualDevice returns null (login failed).
//
// ports MainWindow.OnAddManualDeviceClicked (ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs)
func MsgManualDeviceNotAdded(ip string) string {
	return fmt.Sprintf("Device with IP: %s could not be added to the overview.", ip)
}

// MsgDeviceReAdded is shown after saving changed the identity or model of the
// current device, which is then removed with RemoveManualDevice
// (MainWindow.SaveAllChanges).
const MsgDeviceReAdded = "The Identity or model has changed, the device will be re-added automatically to the Service Installer"

// DeviceAlreadyPresentError is returned by CheckManualDuplicate and
// AddManualDevice for a duplicate address; its text is MsgDeviceAlreadyPresent.
//
// ports MainWindow.OnAddManualDeviceClicked (ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs)
type DeviceAlreadyPresentError struct{ IPAddress string }

func (e *DeviceAlreadyPresentError) Error() string { return MsgDeviceAlreadyPresent(e.IPAddress) }

// ManualDevices returns the persisted manual devices in insertion order.
func (st *Store) ManualDevices() []ManualDevice {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return slices.Clone(st.s.ManualDevices)
}

// checkDuplicateLocked reports a device whose Address string equals ip.
func (st *Store) checkDuplicateLocked(ip string, presentAddresses []string) error {
	if slices.Contains(presentAddresses, ip) ||
		slices.ContainsFunc(st.s.ManualDevices, func(m ManualDevice) bool { return m.IPAddress == ip }) {
		return &DeviceAlreadyPresentError{IPAddress: ip}
	}
	return nil
}

// CheckManualDuplicate ports the duplicate check of
// MainWindow.OnAddManualDeviceClicked, which runs BEFORE any login:
//
//	if (m_colDevices.FirstOrDefault(a => a.Address == dlg.IPAddress.ToString()) != null)
//	    DlgInfo("Device with IP: ... is already present in the overview and cannot be added.")
//
// A device whose Address string equals ip, on any port, is already in the
// overview, so *DeviceAlreadyPresentError is returned and nothing else may
// happen (no login). presentAddresses are the addresses of the other devices
// in the overview (e.g. discovered chargers). The persisted manual devices are
// always checked as well. Nothing is written.
//
// LANConnection.AddManualDevice has its own IP+port check, but it compares
// IPAddress objects with == (reference equality; IPAddress has no
// operator==), so it never matches a freshly parsed address and is not
// reproduced.
//
// ports MainWindow.OnAddManualDeviceClicked (ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs)
func (st *Store) CheckManualDuplicate(ip string, presentAddresses ...string) error {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return st.checkDuplicateLocked(ip, presentAddresses)
}

// AddManualDevice persists a manually added device whose login succeeded.
// The C# order, which the caller must keep, is:
//
//  1. AcceptManualIP (dialog validation + LastManual* save);
//  2. CheckManualDuplicate; stop on *DeviceAlreadyPresentError;
//  3. LANConnection.AddManualDevice: device login + property reads;
//  4. on success AddManualDevice and MsgManualDeviceAdded; on failure
//     nothing is stored and MsgManualDeviceNotAdded is shown (the C#
//     returns null and the device is never kept).
//
// AddManualDevice repeats the duplicate check of step 2 under the store lock,
// so that the persisted list cannot get two entries for one address.
// Persisting the list across restarts is a Go-native addition (the C# keeps
// it in memory only).
//
// ports LANConnection.AddManualDevice success path (Devices.Add) (ACENetwork/ICUNetwork/LANConnection.cs)
func (st *Store) AddManualDevice(d ManualDevice, presentAddresses ...string) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	if err := st.checkDuplicateLocked(d.IPAddress, presentAddresses); err != nil {
		return err
	}
	st.s.ManualDevices = append(st.s.ManualDevices, d)
	return st.saveLocked()
}

// RemoveManualDevice ports LANConnection.RemoveManualDevice for the persisted
// list: the entry with this address is removed (MainWindow's "Remove device
// from this list" button, or SaveAllChanges after an identity/model change).
// removed is false when no manual device has the address.
//
// ports LANConnection.RemoveManualDevice (ACENetwork/ICUNetwork/LANConnection.cs)
func (st *Store) RemoveManualDevice(ipAddress string) (removed bool, err error) {
	st.mu.Lock()
	defer st.mu.Unlock()
	for i, m := range st.s.ManualDevices {
		if m.IPAddress == ipAddress {
			st.s.ManualDevices = slices.Delete(st.s.ManualDevices, i, i+1)
			return true, st.saveLocked()
		}
	}
	return false, nil
}
