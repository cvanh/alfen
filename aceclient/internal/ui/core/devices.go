package core

import (
	"net"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"alfen/aceclient/internal/scn"
)

// NoDevicesText is PanelNoDevice's label.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/PanelNoDevice.cs
const NoDevicesText = "No devices found on the current network, please make sure the Charging Station and this PC are connected to each other over ethernet."

// NoDeviceSelectedText is PanelNoDeviceSelected's label.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/PanelNoDeviceSelected.cs
const NoDeviceSelectedText = "No charging station selected.\nPlease select a charging station in the list."

// NotLoggedInText is PanelNotLoggedIn's label (its button reads "Login...").
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/PanelNotLoggedIn.cs
const NotLoggedInText = "You are logged out.\nPlease retry to login."

// DeviceFilterTooltip is MainWindow.CreateDeviceFilterUI's tooltip.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs:715-735
const DeviceFilterTooltip = "Filter the device list by S/N, CPID, IP address or model name"

// DeviceSocket is the SCN telemetry of one socket (ICUNetwork.SCNSocket).
type DeviceSocket struct {
	Index    byte
	State    scn.ChargingState
	Mode3    byte
	SetPoint float64
	LastSeen time.Time
}

// Device mirrors the ICULanDevice fields the installer's device tree
// (UIListLanDevice) renders, plus the SCN telemetry that discovery fills.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/UIListLanDevice.cs:69-95, firmware/decompiled/ACENetwork/ICUNetwork/ICULanDevice.cs:113-125
type Device struct {
	Identity        string // ICULanDevice.Identity (Identification, "CPID")
	HostName        string // DisplayNameLine2 (mDNS hostname; empty from SCN)
	Address         string // ICULanDevice.Address (IP)
	Port            int    // 443 unless discovery says otherwise
	SCNNetwork      string // ICULanDevice.SCNNetwork (SCN group name)
	FirmwareVersion string // RawFirmwareVersion (mDNS "fwversion"; empty from SCN)
	SerialNumber    string // from the mDNS hostname suffix; empty from SCN
	Discovered      bool   // false renders "(manual)"
	Source          string // "scn", "mdns", "manual"
	NumberOfSockets int    // mDNS "type" TXT (ReInitialize); 0 = unknown

	// SCN telemetry (SCNSocket fields), keyed by socket index.
	TotalSockets  int
	UniqueID      uint64
	ScnLibVersion int
	Sockets       map[byte]DeviceSocket
	LastSeen      time.Time
}

// HasSCNNetwork ports ICULanDevice.HasSCNNetwork.
func (d Device) HasSCNNetwork() bool { return d.SCNNetwork != "" }

// SocketSummary renders the per-socket charging states in SocketIndex order
// using the raw SCN socket index, e.g. "0: Idle, 1: Charging".
func (d Device) SocketSummary() string {
	idx := make([]int, 0, len(d.Sockets))
	for i := range d.Sockets {
		idx = append(idx, int(i))
	}
	sort.Ints(idx)
	parts := make([]string, 0, len(idx))
	for _, i := range idx {
		parts = append(parts, strconv.Itoa(i)+": "+d.Sockets[byte(i)].State.String())
	}
	return strings.Join(parts, ", ")
}

// DeviceMatchesFilter ports MainWindow.DeviceMatchesFilter: an empty filter
// matches; otherwise a case-insensitive substring of HostName,
// Identification or Address.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs:915-934
func DeviceMatchesFilter(d Device, filter string) bool {
	if filter == "" {
		return true
	}
	f := strings.ToLower(filter)
	for _, s := range []string{d.HostName, d.Identity, d.Address} {
		if s != "" && strings.Contains(strings.ToLower(s), f) {
			return true
		}
	}
	return false
}

// DeviceList is the discovered-device collection (MainWindow.m_colDevices),
// keyed by IP address. It is safe for concurrent use.
type DeviceList struct {
	mu sync.RWMutex
	m  map[string]*Device
}

// NewDeviceList returns an empty list.
func NewDeviceList() *DeviceList { return &DeviceList{m: map[string]*Device{}} }

// UpsertSCN folds one decoded SCN datagram into the list. The station name
// (SCNSocket.Name, max 20 chars + NUL) is taken as the identity and the
// sender IP as the address; the port defaults to 443 since SCN carries none.
func (l *DeviceList) UpsertSCN(s *scn.Socket, from net.IP, now time.Time) {
	if s == nil || from == nil {
		return
	}
	addr := from.String()
	l.mu.Lock()
	defer l.mu.Unlock()
	d, ok := l.m[addr]
	if !ok {
		d = &Device{Address: addr, Port: 443, Discovered: true, Source: "scn", Sockets: map[byte]DeviceSocket{}}
		l.m[addr] = d
	}
	if d.Identity == "" || d.Source == "scn" {
		d.Identity = s.Name
	}
	d.SCNNetwork = s.NetworkName
	d.TotalSockets = s.TotalNumberOfSockets
	d.UniqueID = s.UniqueID
	d.ScnLibVersion = s.ScnLibVersion
	if d.Sockets == nil {
		d.Sockets = map[byte]DeviceSocket{}
	}
	d.Sockets[s.SocketIndex] = DeviceSocket{
		Index: s.SocketIndex, State: s.State, Mode3: s.Mode3State,
		SetPoint: s.SetPointCurrent, LastSeen: now,
	}
	d.LastSeen = now
}

// Upsert adds or replaces a device by address (mDNS / manual add).
// SCN telemetry already collected for that address is kept.
func (l *DeviceList) Upsert(d Device) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if old, ok := l.m[d.Address]; ok && d.Sockets == nil {
		d.Sockets = old.Sockets
		if d.SCNNetwork == "" {
			d.SCNNetwork = old.SCNNetwork
		}
	}
	if d.Port == 0 {
		d.Port = 443
	}
	cp := d
	l.m[d.Address] = &cp
}

// Remove drops a device (LANConnection.RemoveDevice / RemoveManualDevice).
func (l *DeviceList) Remove(address string) {
	l.mu.Lock()
	delete(l.m, address)
	l.mu.Unlock()
}

// Clear ports MainWindow.OnRefreshLAN: every device is removed.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs:1553-1560
func (l *DeviceList) Clear() {
	l.mu.Lock()
	l.m = map[string]*Device{}
	l.mu.Unlock()
}

// Len returns the number of devices (before filtering).
func (l *DeviceList) Len() int {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return len(l.m)
}

// Get returns a copy of the device at address.
func (l *DeviceList) Get(address string) (Device, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	d, ok := l.m[address]
	if !ok {
		return Device{}, false
	}
	return copyDevice(d), true
}

// Rows ports MainWindow.RefreshDeviceList's ordering and filter:
// orderby HasSCNNetwork descending, SCNNetwork, Identification; then
// DeviceMatchesFilter. Address breaks remaining ties for a stable table.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs:1404-1502
func (l *DeviceList) Rows(filter string) []Device {
	l.mu.RLock()
	out := make([]Device, 0, len(l.m))
	for _, d := range l.m {
		if DeviceMatchesFilter(*d, filter) {
			out = append(out, copyDevice(d))
		}
	}
	l.mu.RUnlock()
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.HasSCNNetwork() != b.HasSCNNetwork() {
			return a.HasSCNNetwork()
		}
		if a.SCNNetwork != b.SCNNetwork {
			return a.SCNNetwork < b.SCNNetwork
		}
		if a.Identity != b.Identity {
			return a.Identity < b.Identity
		}
		return a.Address < b.Address
	})
	return out
}

func copyDevice(d *Device) Device {
	c := *d
	if d.Sockets != nil {
		c.Sockets = make(map[byte]DeviceSocket, len(d.Sockets))
		for k, v := range d.Sockets {
			c.Sockets[k] = v
		}
	}
	return c
}

// ParseIPAddress ports DlgManualIP's IPAddress.TryParse check. The error text
// is the dialog's ("Invalid IP addres!" sic, "Please enter an ip address in
// the following form: 192.168.1.10"). Go's parser accepts dotted-quad IPv4
// and IPv6 only (not .NET's shorthand forms such as "10.1").
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/DlgManualIP.cs:132-166
func ParseIPAddress(text string) (net.IP, *UserError) {
	ip := net.ParseIP(strings.TrimSpace(text))
	if ip == nil {
		return nil, &UserError{Primary: "Invalid IP addres!", Secondary: "Please enter an ip address in the following form: 192.168.1.10"}
	}
	return ip, nil
}

// ValidatePort mirrors DlgManualIP's port SpinButton range (0..65535). The
// spin button cannot hold an invalid value, so the C# has no message for it;
// the text here is the port's own (needed because the Go bar uses a free text
// entry). 0 is accepted by the spin button but can never connect, so it is
// rejected too.
func ValidatePort(port int) *UserError {
	if port <= 0 || port > 65535 {
		return &UserError{Primary: "Invalid port!", Secondary: "Please enter a port between 1 and 65535 (default 443)."}
	}
	return nil
}

// UserError is a MessageDialog.ShowError(primary, secondary) pair.
type UserError struct {
	Primary   string
	Secondary string
}

func (e *UserError) Error() string {
	if e.Secondary == "" {
		return e.Primary
	}
	return e.Primary + "\n" + e.Secondary
}
