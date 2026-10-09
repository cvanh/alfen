package discovery

import (
	"context"
	"log/slog"
	"net"
	"net/netip"
	"slices"
	"strings"
	"sync"
	"time"
)

// ServiceTypes ports LANConnection.serviceTypes.
var ServiceTypes = []string{"_lolo3._http._tcp", "_alfen._tcp"}

// Query parameter overrides applied by LANConnection.StartSearch.
const (
	SearchRobustness   = 10
	SearchResponseTime = 2000 * time.Millisecond
)

// SearchTimerInterval ports MainWindow.m_timSearch (Interval = 2000,
// AutoReset): see LANConnection.RunSearchTimer.
const SearchTimerInterval = 2000 * time.Millisecond

// DeviceEventKind identifies a LANConnection event.
type DeviceEventKind int

const (
	// DeviceRegistered ports LANConnection.DeviceRegistered: a new device
	// was discovered and appended to Devices.
	DeviceRegistered DeviceEventKind = iota + 1
	// DeviceReRegistered ports LANConnection.DeviceReRegistered: a known
	// device was announced again (ReInitialize ran).
	DeviceReRegistered
	// DeviceUnregistered ports LANConnection.DeviceUnregistered.
	DeviceUnregistered
	// DevicesChanged ports Devices (ObservableCollection).CollectionChanged,
	// which MainWindow uses to refresh its device tree.
	DevicesChanged
	// ConnectionError ports LANConnection.ErrorHandler (CallErrorHandler).
	ConnectionError
)

func (k DeviceEventKind) String() string {
	switch k {
	case DeviceRegistered:
		return "DeviceRegistered"
	case DeviceReRegistered:
		return "DeviceReRegistered"
	case DeviceUnregistered:
		return "DeviceUnregistered"
	case DevicesChanged:
		return "DevicesChanged"
	case ConnectionError:
		return "ConnectionError"
	}
	return "DeviceEventKind(?)"
}

// CollectionAction ports the NotifyCollectionChangedAction values raised by
// the operations LANConnection performs on Devices.
type CollectionAction int

const (
	CollectionAdd    CollectionAction = 0
	CollectionRemove CollectionAction = 1
	CollectionReset  CollectionAction = 4
)

// DeviceEvent ports DeviceEventArgs (plus the CollectionChanged and
// ErrorHandler payloads).
type DeviceEvent struct {
	Kind   DeviceEventKind
	Device Device // snapshot at the time of the event (zero for Reset/ConnectionError)
	// EndpointChanged is set on DeviceReRegistered when ReInitialize saw a new
	// IP address or port; the C# then dropped the session (IsLoggedIn = false),
	// so the session layer must log in again.
	EndpointChanged bool
	// Action is set on DevicesChanged.
	Action CollectionAction
	// Message is set on ConnectionError.
	Message string
}

// ManualDeviceLogin is the device-session half of AddManualDevice:
// iCULanDevice2.Login().IsLoggedIn followed by
// UpdateCategories("generic", "generic2"). It may update d (e.g. Identity /
// HostName derived from properties) before the device is published, and
// returns a GetPropertyInt(propId, subId) reader over the fetched properties
// (0 for a missing property). ok=false means the login failed.
type ManualDeviceLogin func(ctx context.Context, d *Device) (getPropertyInt func(propID uint16, subID uint8) int, ok bool)

// EMeterTypes values used by AddManualDevice (ICUNetwork.EMeterTypes).
const (
	meterModbusCentral = 0
	meterFKN           = 2
	meterTCPIPCentral  = 3
	meterTCPIPSmart    = 4
	meterP1            = 5
)

// LANConnection ports ICUNetwork.LANConnection (and BaseConnection): the LAN
// device registry fed by the mDNS browser.
//
// Events are delivered serially, in order, on an internal goroutine, never
// while a registry lock is held (MainWindow received them on the UI thread);
// handlers may call back into the LANConnection.
type LANConnection struct {
	// Logger receives the C# Logger.Debug/Information/Error output. Nil
	// discards it. Set before StartSearch.
	Logger *slog.Logger

	browserMu   sync.Mutex // lock (m_serviceBrowser)
	browser     *Browser   // Lazy<ServiceBrowser>
	unsubscribe func()
	ctx         context.Context

	mu          sync.Mutex // Monitor on Devices
	devices     []*Device
	doNotRemove []*Device
	ifaces      []net.Interface
	nextKey     DeviceKey

	subs subscribers[DeviceEvent]
	disp *dispatcher

	newBrowser func() *Browser // test hook
}

// NewLANConnection ports the LANConnection constructor (with an empty Devices
// collection).
func NewLANConnection() *LANConnection {
	return &LANConnection{disp: newDispatcher(), newBrowser: NewBrowser}
}

// Name ports BaseConnection.m_sName as set by LANConnection ("Ethernet").
func (c *LANConnection) Name() string { return "Ethernet" }

// DeviceFound ports LANConnection.DeviceFound (a getter that is never set).
func (c *LANConnection) DeviceFound() bool { return false }

// Subscribe adds an event handler (C# "+="); the returned func removes it.
func (c *LANConnection) Subscribe(fn func(DeviceEvent)) (unsubscribe func()) {
	return c.subs.add(fn)
}

// Events returns a channel carrying every event until ctx is done, after
// which the channel is closed. Delivery blocks while the channel is full.
func (c *LANConnection) Events(ctx context.Context) <-chan DeviceEvent {
	ch := make(chan DeviceEvent, 64)
	var mu sync.Mutex
	closed := false
	unsub := c.Subscribe(func(ev DeviceEvent) {
		mu.Lock()
		defer mu.Unlock()
		if closed {
			return
		}
		select {
		case ch <- ev:
		case <-ctx.Done():
		}
	})
	go func() {
		<-ctx.Done()
		unsub()
		mu.Lock()
		closed = true
		close(ch)
		mu.Unlock()
	}()
	return ch
}

func (c *LANConnection) emit(evs ...DeviceEvent) {
	for _, ev := range evs {
		c.disp.post(func() { c.subs.deliver(ev) })
	}
}

// WaitEvents blocks until every event emitted so far has been delivered.
func (c *LANConnection) WaitEvents() { c.disp.wait() }

func (c *LANConnection) serviceBrowser() *Browser {
	if c.browser == nil {
		c.browser = c.newBrowser()
		c.browser.Logger = c.Logger
	}
	return c.browser
}

// StartSearch ports LANConnection.StartSearch: set Robustness = 10 and
// ResponseTime = 2000 ms, subscribe to the browser and start browsing
// ServiceTypes unless already browsing. ctx bounds every browse session
// started through this connection (cancelling it stops browsing, like
// StopBrowsing; call StopSearch to also clear the registry).
//
// Unlike the C# "+=", calling StartSearch twice does not subscribe twice.
func (c *LANConnection) StartSearch(ctx context.Context) error {
	c.log(slog.LevelDebug-4, "Start browsing for types", "ServiceTypes", ServiceTypes)
	c.browserMu.Lock()
	defer c.browserMu.Unlock()
	b := c.serviceBrowser()
	p := b.QueryParameters()
	p.Robustness = SearchRobustness
	p.ResponseTime = SearchResponseTime
	b.SetQueryParameters(p)
	if c.unsubscribe == nil {
		c.unsubscribe = b.Subscribe(c.onBrowserEvent)
	}
	c.ctx = ctx
	if !b.IsBrowsing() {
		return b.StartBrowse(ctx, ServiceTypes...)
	}
	return nil
}

// StopSearch ports LANConnection.StopSearch: unsubscribe, stop browsing,
// clear Devices and NetworkInterfaces.
func (c *LANConnection) StopSearch() {
	c.log(slog.LevelDebug-4, "Stop browsing for types", "ServiceTypes", ServiceTypes)
	c.browserMu.Lock()
	b := c.serviceBrowser()
	if c.unsubscribe != nil {
		c.unsubscribe()
		c.unsubscribe = nil
	}
	if b.IsBrowsing() {
		b.StopBrowse()
	}
	c.browserMu.Unlock()
	// In the installer both the browser events and StopSearch ran on the UI
	// thread; wait for a delivery already in flight so it cannot re-add a
	// device after the clear below.
	b.waitIdle()

	c.mu.Lock()
	c.devices = nil
	c.ifaces = nil
	c.mu.Unlock()
	c.emit(DeviceEvent{Kind: DevicesChanged, Action: CollectionReset})
}

// StartBrowsing ports LANConnection.StartBrowsing (BaseConnection override):
// remember the current devices as "do not remove" (so the removals raised by
// restarting the browser do not drop them), then restart browsing. The C#
// logs and swallows errors; the error is also returned here.
func (c *LANConnection) StartBrowsing() error {
	c.browserMu.Lock()
	defer c.browserMu.Unlock()
	c.mu.Lock()
	c.doNotRemove = slices.Clone(c.devices)
	c.mu.Unlock()
	b := c.serviceBrowser()
	if b.IsBrowsing() {
		b.StopBrowse()
	}
	ctx := c.ctx
	if ctx == nil {
		ctx = context.Background()
	}
	if err := b.StartBrowse(ctx, ServiceTypes...); err != nil {
		c.log(slog.LevelError, err.Error())
		return err
	}
	return nil
}

// StopBrowsing ports LANConnection.StopBrowsing.
func (c *LANConnection) StopBrowsing() {
	c.browserMu.Lock()
	defer c.browserMu.Unlock()
	c.serviceBrowser().StopBrowse()
}

// RunSearchTimer ports MainWindow.m_timSearch / OnSearchTimerElapsed (the
// installer starts it at window creation): every SearchTimerInterval, while
// no device is known, restart browsing. It returns when ctx is done.
func (c *LANConnection) RunSearchTimer(ctx context.Context) {
	t := time.NewTicker(SearchTimerInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			c.mu.Lock()
			n := len(c.devices)
			c.mu.Unlock()
			if n == 0 {
				_ = c.StartBrowsing()
			}
		}
	}
}

// Devices ports BaseConnection.Devices: snapshots in collection order.
func (c *LANConnection) Devices() []Device {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]Device, len(c.devices))
	for i, d := range c.devices {
		out[i] = *d
	}
	return out
}

// NetworkInterfaces ports BaseConnection.NetworkInterfaces as maintained by
// OnNetworkInterfaceAdded/Removed.
func (c *LANConnection) NetworkInterfaces() []net.Interface {
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.Clone(c.ifaces)
}

// Device returns the current snapshot of the device with the given key.
func (c *LANConnection) Device(key DeviceKey) (Device, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if d := c.find(key); d != nil {
		return *d, true
	}
	return Device{}, false
}

// UpdateDevice applies fn to the registry's device under the registry lock and
// returns the new snapshot. It is how other layers mutate ICULanDevice fields
// the registry reads (IsRebooting, Identity/HostName after a property read,
// ...). fn must not call back into the LANConnection.
func (c *LANConnection) UpdateDevice(key DeviceKey, fn func(d *Device)) (Device, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	d := c.find(key)
	if d == nil {
		return Device{}, false
	}
	fn(d)
	d.Key = key
	return *d, true
}

func (c *LANConnection) find(key DeviceKey) *Device {
	if key == 0 {
		return nil
	}
	for _, d := range c.devices {
		if d.Key == key {
			return d
		}
	}
	return nil
}

func (c *LANConnection) add(d *Device) {
	c.nextKey++
	d.Key = c.nextKey
	c.devices = append(c.devices, d)
}

func (c *LANConnection) remove(d *Device) bool {
	if i := slices.Index(c.devices, d); i >= 0 {
		c.devices = slices.Delete(c.devices, i, i+1)
		return true
	}
	return false
}

// CallErrorHandler ports LANConnection.CallErrorHandler: the message is
// raised (ConnectionError) only while a device with the same host name is
// registered.
func (c *LANConnection) CallErrorHandler(msg string, device Device) {
	c.mu.Lock()
	found := slices.ContainsFunc(c.devices, func(a *Device) bool { return a.HostName == device.HostName })
	c.mu.Unlock()
	if found {
		c.log(slog.LevelError, "Error: "+msg, "HostName", device.HostName, "IPAddress", device.Address())
		c.emit(DeviceEvent{Kind: ConnectionError, Device: device, Message: msg})
	}
}

func (c *LANConnection) onBrowserEvent(ev ServiceEvent) {
	switch ev.Kind {
	case ServiceAdded, ServiceChanged:
		c.onServiceAdded(ev.Announcement)
	case ServiceRemoved:
		c.onServiceRemoved(ev.Announcement)
	case NetworkInterfaceAdded:
		c.onNetworkInterfaceAdded(ev.NetworkInterface)
	case NetworkInterfaceRemoved:
		c.onNetworkInterfaceRemoved(ev.NetworkInterface)
	}
}

// onServiceAdded ports LANConnection.OnServiceAdded (also wired to
// ServiceChanged): match an existing device by exact host name, else by the
// object id (last '-' part of the host name); re-initialise it
// (DeviceReRegistered) or create and append a new one (DeviceRegistered).
// Announcements without addresses are ignored.
//
// The C# guards with Monitor.TryEnter(Devices, 5000) and gives up silently on
// timeout; no I/O happens under the Go mutex, so it is taken unconditionally.
func (c *LANConnection) onServiceAdded(ann ServiceAnnouncement) {
	if len(ann.Addresses) == 0 {
		return
	}
	a := ann
	addr := ann.Addresses[0]
	c.mu.Lock()
	newHostName := ann.Hostname
	var dev *Device
	for _, d := range c.devices {
		if d.HostName == newHostName {
			dev = d
			break
		}
	}
	if dev == nil {
		objectID := lastDashPart(newHostName)
		for _, d := range c.devices {
			if lastDashPart(d.HostName) == objectID {
				dev = d
				break
			}
		}
	}
	var evs []DeviceEvent
	if dev == nil {
		dev = newLanDevice(addr, int(ann.Port), &a, false)
		c.add(dev)
		c.log(slog.LevelDebug, "Added Device", "SerialNumber", lastDashPart(ann.Hostname), "IpAddress", addr.String(), "SCNNetwork", dev.SCNNetwork, "Protocol", dev.Protocol())
		evs = append(evs,
			DeviceEvent{Kind: DevicesChanged, Action: CollectionAdd, Device: *dev},
			DeviceEvent{Kind: DeviceRegistered, Device: *dev})
	} else {
		changed := dev.reInitialize(addr, int(ann.Port), &a, false)
		c.log(slog.LevelDebug, "Reinitialized Device", "DeviceName", lastDashPart(ann.Hostname), "IpAddress", addr.String(), "SCNNetwork", dev.SCNNetwork, "Protocol", dev.Protocol())
		evs = append(evs, DeviceEvent{Kind: DeviceReRegistered, Device: *dev, EndpointChanged: changed})
	}
	c.mu.Unlock()
	c.emit(evs...)
}

// onServiceRemoved ports LANConnection.OnServiceRemoved: unless the host is in
// the do-not-remove snapshot taken by StartBrowsing, the device with exactly
// this host name is unregistered and removed — if it is not rebooting.
func (c *LANConnection) onServiceRemoved(ann ServiceAnnouncement) {
	if len(ann.Addresses) == 0 {
		return
	}
	c.mu.Lock()
	var evs []DeviceEvent
	protected := slices.ContainsFunc(c.doNotRemove, func(a *Device) bool { return a.HostName == ann.Hostname })
	if !protected {
		var dev *Device
		for _, d := range c.devices {
			if d.HostName == ann.Hostname {
				dev = d
				break
			}
		}
		if dev != nil && dev.OnDeviceRemoved() {
			c.log(slog.LevelDebug, "Device removed", "DeviceName", lastDashPart(ann.Hostname), "IpAddress", ann.Addresses[0].String(), "SCNNetwork", dev.SCNNetwork)
			evs = append(evs, DeviceEvent{Kind: DeviceUnregistered, Device: *dev})
			c.remove(dev)
			evs = append(evs, DeviceEvent{Kind: DevicesChanged, Action: CollectionRemove, Device: *dev})
		}
	}
	c.mu.Unlock()
	c.emit(evs...)
}

// AddManualDevice ports LANConnection.AddManualDevice(address, port,
// hostName = "", numberOfSockets = 2, LoginRequired = false). The new device
// is published (DevicesChanged/Add, no DeviceRegistered) only when login
// succeeds; it then takes NumberOfSockets (8286_0), SocketTypes (8485_0,
// 12581_0 when it has more than one socket) and the meter flags from the
// central (16919_0) and smart (21015_0) meter types. A nil login counts as a
// failed login.
//
// The C# first loops over Devices comparing IPAddress with "==", which for
// System.Net.IPAddress is reference equality, so a freshly parsed address
// never matches and that early return is dead code; it is not ported. The
// installer's duplicate check is MainWindow's (see HasDeviceWithAddress).
func (c *LANConnection) AddManualDevice(ctx context.Context, address netip.Addr, port int, hostName string, numberOfSockets int, loginRequired bool, login ManualDeviceLogin) (Device, bool) {
	d := newLanDevice(address, port, nil, true)
	if hostName != "" {
		d.SetHostInfo(hostName, numberOfSockets)
	}
	if loginRequired {
		d.SetUniquePasswordRequired(true)
	}
	if login == nil {
		return Device{}, false
	}
	get, ok := login(ctx, d)
	if !ok {
		return Device{}, false
	}
	if get == nil {
		get = func(uint16, uint8) int { return 0 }
	}
	d.NumberOfSockets = get(8286, 0)
	d.SocketTypes[0] = get(8485, 0)
	if d.NumberOfSockets > 1 {
		d.SocketTypes[1] = get(12581, 0)
	} else {
		d.SocketTypes[1] = 0
	}
	central := get(16919, 0)
	smart := get(21015, 0)
	d.HasCentralMeter = central == meterModbusCentral || central == meterP1 || central == meterTCPIPCentral || central == meterFKN
	d.HasSmartMeter = smart == meterP1 || smart == meterTCPIPSmart
	c.mu.Lock()
	c.add(d)
	snap := *d
	c.mu.Unlock()
	c.emit(DeviceEvent{Kind: DevicesChanged, Action: CollectionAdd, Device: snap})
	return snap, true
}

// RemoveManualDevice ports LANConnection.RemoveManualDevice: drop the device
// from Devices without raising DeviceUnregistered (Deallocate of the
// session-side sub-objects is the caller's business). A zero Device is
// ignored (the C# null check).
func (c *LANConnection) RemoveManualDevice(device Device) {
	if device.Key == 0 {
		return
	}
	c.mu.Lock()
	d := c.find(device.Key)
	removed := d != nil && c.remove(d)
	c.mu.Unlock()
	if removed {
		c.emit(DeviceEvent{Kind: DevicesChanged, Action: CollectionRemove, Device: *d})
	}
}

// RemoveDevice ports LANConnection.RemoveDevice: unless the device is
// rebooting, raise DeviceUnregistered and remove it (used by MainWindow's
// Refresh button before StartBrowsing).
func (c *LANConnection) RemoveDevice(device Device) {
	c.mu.Lock()
	d := c.find(device.Key)
	cur := device
	if d != nil {
		cur = *d
	}
	if !cur.OnDeviceRemoved() {
		c.mu.Unlock()
		return
	}
	c.log(slog.LevelDebug, "Device removed", "DeviceName", cur.Identification(), "IpAddress", cur.Address(), "SCNNetwork", cur.SCNNetwork)
	evs := []DeviceEvent{{Kind: DeviceUnregistered, Device: cur}}
	if d != nil && c.remove(d) {
		evs = append(evs, DeviceEvent{Kind: DevicesChanged, Action: CollectionRemove, Device: cur})
	}
	c.mu.Unlock()
	c.emit(evs...)
}

// FindLanDevice ports LANConnection.FindLanDevice: the first device whose
// Address equals ipAddress.ToString(), case-insensitively.
func (c *LANConnection) FindLanDevice(ipAddress netip.Addr) (Device, bool) {
	want := strings.ToLower(ipAddress.String())
	c.mu.Lock()
	defer c.mu.Unlock()
	for _, d := range c.devices {
		if strings.ToLower(d.Address()) == want {
			return *d, true
		}
	}
	return Device{}, false
}

// FindDevicesInSCN ports LANConnection.FindDevicesInSCN: devices whose
// SCNNetwork equals scnName case-insensitively. (The C# never matches a
// device whose SCNNetwork is null; Go has no null string, so an empty scnName
// also matches devices that never announced an SCN.)
func (c *LANConnection) FindDevicesInSCN(scnName string) []Device {
	want := strings.ToLower(scnName)
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []Device
	for _, d := range c.devices {
		if strings.ToLower(d.SCNNetwork) == want {
			out = append(out, *d)
		}
	}
	return out
}

// HasDeviceWithAddress ports MainWindow.OnAddManualDeviceClicked's guard
// (m_colDevices.FirstOrDefault(a => a.Address == dlg.IPAddress.ToString())):
// an exact, port-agnostic address match. When it is true the installer shows
// "Device with IP: {ip} is already present in the overview and cannot be
// added." and does not call AddManualDevice.
func (c *LANConnection) HasDeviceWithAddress(ip netip.Addr) bool {
	want := ip.String()
	c.mu.Lock()
	defer c.mu.Unlock()
	return slices.ContainsFunc(c.devices, func(d *Device) bool { return d.Address() == want })
}

// DeviceMatchesFilter ports MainWindow.DeviceMatchesFilter: an empty filter
// matches everything, otherwise HostName, Identification or Address must
// contain it (OrdinalIgnoreCase).
func DeviceMatchesFilter(d Device, filter string) bool {
	if filter == "" {
		return true
	}
	for _, s := range []string{d.HostName, d.Identification(), d.Address()} {
		if s != "" && strings.Contains(strings.ToUpper(s), strings.ToUpper(filter)) {
			return true
		}
	}
	return false
}

// onNetworkInterfaceAdded ports LANConnection.OnNetworkInterfaceAdded
// (replace an entry with the same Id, then add).
func (c *LANConnection) onNetworkInterfaceAdded(ifi net.Interface) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if i := slices.IndexFunc(c.ifaces, func(a net.Interface) bool { return a.Name == ifi.Name }); i >= 0 {
		c.ifaces = slices.Delete(c.ifaces, i, i+1)
	}
	c.ifaces = append(c.ifaces, ifi)
}

// onNetworkInterfaceRemoved ports LANConnection.OnNetworkInterfaceRemoved.
func (c *LANConnection) onNetworkInterfaceRemoved(ifi net.Interface) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if i := slices.IndexFunc(c.ifaces, func(a net.Interface) bool { return a.Name == ifi.Name && a.Index == ifi.Index }); i >= 0 {
		c.ifaces = slices.Delete(c.ifaces, i, i+1)
	}
}

func (c *LANConnection) log(level slog.Level, msg string, args ...any) {
	if c.Logger != nil {
		c.Logger.Log(context.Background(), level, msg, args...)
	}
}
