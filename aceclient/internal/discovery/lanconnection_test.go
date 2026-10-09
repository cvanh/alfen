package discovery

import (
	"context"
	"net"
	"net/netip"
	"slices"
	"sync"
	"testing"
	"time"
)

type devLog struct {
	mu  sync.Mutex
	evs []DeviceEvent
}

func collect(c *LANConnection) *devLog {
	l := &devLog{}
	c.Subscribe(func(ev DeviceEvent) {
		l.mu.Lock()
		l.evs = append(l.evs, ev)
		l.mu.Unlock()
	})
	return l
}

func (l *devLog) take(c *LANConnection) []DeviceEvent {
	c.WaitEvents()
	l.mu.Lock()
	defer l.mu.Unlock()
	out := l.evs
	l.evs = nil
	return out
}

func (l *devLog) kinds(c *LANConnection) []DeviceEventKind {
	var ks []DeviceEventKind
	for _, e := range l.take(c) {
		ks = append(ks, e.Kind)
	}
	return ks
}

// svc builds an announcement; without explicit TXT strings it carries one
// ("type=1.0.0"), since ReInitialize only adopts the host name when Txt is
// non-empty (see TestLANEmptyTXTQuirk).
func svc(host string, port uint16, ip string, txt ...string) ServiceAnnouncement {
	if len(txt) == 0 {
		txt = []string{"type=1.0.0"}
	}
	a := ServiceAnnouncement{Instance: "i", Type: "_alfen._tcp", Domain: "local.", Hostname: host, Port: port, Txt: txt, NetworkInterface: testIface}
	if ip != "" {
		a.Addresses = []netip.Addr{netip.MustParseAddr(ip)}
	}
	return a
}

func added(a ServiceAnnouncement) ServiceEvent {
	return ServiceEvent{Kind: ServiceAdded, Announcement: a}
}
func changed(a ServiceAnnouncement) ServiceEvent {
	return ServiceEvent{Kind: ServiceChanged, Announcement: a}
}
func removed(a ServiceAnnouncement) ServiceEvent {
	a.IsRemoved = true
	return ServiceEvent{Kind: ServiceRemoved, Announcement: a}
}

func TestLANRegisterAndReRegister(t *testing.T) {
	c := NewLANConnection()
	log := collect(c)

	c.onBrowserEvent(added(svc(host1, 443, "192.168.1.10", "identity=ACE0012345", "scnnetwork=Site")))
	evs := log.take(c)
	if len(evs) != 2 || evs[0].Kind != DevicesChanged || evs[0].Action != CollectionAdd || evs[1].Kind != DeviceRegistered {
		t.Fatalf("events %+v", evs)
	}
	d := evs[1].Device
	if d.Key == 0 || d.HostName != host1 || d.Identity != "ACE0012345" || d.SerialNumber != "1234567" ||
		d.Address() != "192.168.1.10" || d.Port != 443 || !d.Discovered || d.IsManuallyAdded || d.SCNNetwork != "Site" {
		t.Fatalf("device %+v", d)
	}

	// ServiceChanged, same endpoint.
	c.onBrowserEvent(changed(svc(host1, 443, "192.168.1.10", "identity=ACE0012345")))
	evs = log.take(c)
	if len(evs) != 1 || evs[0].Kind != DeviceReRegistered || evs[0].EndpointChanged || evs[0].Device.Key != d.Key {
		t.Fatalf("events %+v", evs)
	}

	// New address.
	c.onBrowserEvent(added(svc(host1, 443, "192.168.1.77", "identity=ACE0012345")))
	evs = log.take(c)
	if len(evs) != 1 || evs[0].Kind != DeviceReRegistered || !evs[0].EndpointChanged || evs[0].Device.Address() != "192.168.1.77" {
		t.Fatalf("events %+v", evs)
	}

	// Different host name, same object id (serial): same device, host name updated.
	c.onBrowserEvent(added(svc("eve-single-s-line-1234567", 443, "192.168.1.77", "identity=ACE0012345")))
	evs = log.take(c)
	if len(evs) != 1 || evs[0].Kind != DeviceReRegistered || evs[0].Device.HostName != "eve-single-s-line-1234567" {
		t.Fatalf("events %+v", evs)
	}

	// Announcements without addresses are ignored.
	c.onBrowserEvent(added(svc("other-1", 443, "")))
	if evs := log.take(c); len(evs) != 0 {
		t.Fatalf("events %+v", evs)
	}

	// A second device.
	c.onBrowserEvent(added(svc("ng910-60023-7654321", 80, "192.168.1.11")))
	if k := log.kinds(c); !slices.Equal(k, []DeviceEventKind{DevicesChanged, DeviceRegistered}) {
		t.Fatalf("kinds %v", k)
	}
	devs := c.Devices()
	if len(devs) != 2 || devs[0].Key != d.Key || devs[1].Protocol() != "http" {
		t.Fatalf("devices %+v", devs)
	}
}

func TestLANUnregister(t *testing.T) {
	c := NewLANConnection()
	log := collect(c)
	c.onBrowserEvent(added(svc(host1, 443, "192.168.1.10")))
	c.onBrowserEvent(added(svc("ng910-60023-2", 443, "192.168.1.12")))
	log.take(c)

	// Removal matches the exact host name only (no object-id fallback).
	c.onBrowserEvent(removed(svc("eve-single-1234567", 443, "192.168.1.10")))
	// Removal without addresses is ignored.
	c.onBrowserEvent(removed(svc(host1, 443, "")))
	if evs := log.take(c); len(evs) != 0 {
		t.Fatalf("events %+v", evs)
	}

	// A rebooting device is kept.
	devs := c.Devices()
	if _, ok := c.UpdateDevice(devs[0].Key, func(d *Device) { d.IsRebooting = true; d.Key = 999 }); !ok {
		t.Fatal("UpdateDevice")
	}
	if d, _ := c.Device(devs[0].Key); d.Key != devs[0].Key || !d.IsRebooting {
		t.Fatalf("UpdateDevice must not change the key: %+v", d)
	}
	c.onBrowserEvent(removed(svc(host1, 443, "192.168.1.10")))
	if evs := log.take(c); len(evs) != 0 {
		t.Fatalf("rebooting device removed: %+v", evs)
	}
	c.UpdateDevice(devs[0].Key, func(d *Device) { d.IsRebooting = false })

	c.onBrowserEvent(removed(svc(host1, 443, "192.168.1.10")))
	evs := log.take(c)
	if len(evs) != 2 || evs[0].Kind != DeviceUnregistered || evs[1].Kind != DevicesChanged || evs[1].Action != CollectionRemove || evs[0].Device.HostName != host1 {
		t.Fatalf("events %+v", evs)
	}
	if devs := c.Devices(); len(devs) != 1 || devs[0].HostName != "ng910-60023-2" {
		t.Fatalf("devices %+v", devs)
	}
}

// newLANWithTestBrowser wires a LANConnection to a fake-socket browser.
func newLANWithTestBrowser(t *testing.T) (*LANConnection, *Browser, *fakeClock, *[]*fakeConn) {
	t.Helper()
	b, fc, _, conns := newTestBrowser(t)
	c := NewLANConnection()
	c.newBrowser = func() *Browser { return b }
	return c, b, fc, conns
}

func TestLANStartBrowsingProtectsKnownDevices(t *testing.T) {
	c, b, _, _ := newLANWithTestBrowser(t)
	defer b.StopBrowse()
	log := collect(c)
	c.onBrowserEvent(added(svc(host1, 443, "192.168.1.10")))
	log.take(c)

	if err := c.StartBrowsing(); err != nil {
		t.Fatal(err)
	}
	c.onBrowserEvent(added(svc("ng910-60023-2", 443, "192.168.1.12"))) // discovered after the snapshot
	log.take(c)

	c.onBrowserEvent(removed(svc(host1, 443, "192.168.1.10")))
	if evs := log.take(c); len(evs) != 0 {
		t.Fatalf("protected device removed: %+v", evs)
	}
	c.onBrowserEvent(removed(svc("ng910-60023-2", 443, "192.168.1.12")))
	if k := log.kinds(c); !slices.Equal(k, []DeviceEventKind{DeviceUnregistered, DevicesChanged}) {
		t.Fatalf("kinds %v", k)
	}
	// StartBrowsing restarts an active browse.
	if !b.IsBrowsing() {
		t.Fatal("not browsing")
	}
	if err := c.StartBrowsing(); err != nil || !b.IsBrowsing() {
		t.Fatalf("restart: %v", err)
	}
	c.StopBrowsing()
	if b.IsBrowsing() {
		t.Fatal("StopBrowsing")
	}
}

func TestLANStartSearchEndToEnd(t *testing.T) {
	c, b, fc, conns := newLANWithTestBrowser(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := c.Events(ctx)

	if err := c.StartSearch(ctx); err != nil {
		t.Fatal(err)
	}
	if err := c.StartSearch(ctx); err != nil { // second call: no double subscription
		t.Fatal(err)
	}
	p := b.QueryParameters()
	if p.Robustness != 10 || p.ResponseTime != 2*time.Second || p.QueryInterval != 10*time.Second || p.StartQueryCount != 2 {
		t.Fatalf("query parameters %+v", p)
	}
	if !b.IsBrowsing() {
		t.Fatal("not browsing")
	}
	if ifs := c.NetworkInterfaces(); len(ifs) != 1 || ifs[0].Name != testIface.Name {
		t.Fatalf("interfaces %+v", ifs)
	}
	fc.advance(0)
	q := (*conns)[0].sentPackets()
	if len(q) != 1 || !slices.Equal(parseQuestions(t, q[0].data), []string{"_lolo3._http._tcp.local./12", "_alfen._tcp.local./12"}) {
		t.Fatalf("start-up query %v", q)
	}

	(*conns)[0].in <- announce(alfen, "ACE0012345", host1, 443, "192.168.1.10", "identity=ACE0012345", "type=2.1.1", "fwversion=6.1.0-4180")
	var reg []DeviceEvent
	deadline := time.After(2 * time.Second)
	for len(reg) < 2 {
		select {
		case ev := <-events:
			reg = append(reg, ev)
		case <-deadline:
			t.Fatalf("got %+v", reg)
		}
	}
	if reg[0].Kind != DevicesChanged || reg[1].Kind != DeviceRegistered {
		t.Fatalf("events %+v", reg)
	}
	d := reg[1].Device
	if d.Identity != "ACE0012345" || d.NumberOfSockets != 2 || d.SocketTypes != [2]int{1, 1} ||
		!sameVersion(d.FirmwareVersion, ver(6, 1, 0, -1)) || d.Address() != "192.168.1.10" || !d.IsHTTPS() ||
		d.Announcement == nil || d.Announcement.NetworkInterface.Name != testIface.Name {
		t.Fatalf("device %+v", d)
	}

	c.StopSearch()
	select {
	case ev := <-events:
		if ev.Kind != DevicesChanged || ev.Action != CollectionReset {
			t.Fatalf("event %+v", ev)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no reset event")
	}
	if b.IsBrowsing() || len(c.Devices()) != 0 || len(c.NetworkInterfaces()) != 0 || !(*conns)[0].isClosed() {
		t.Fatal("StopSearch must stop browsing and clear the registry")
	}
	// Unsubscribed: browser events no longer reach the registry.
	b.postEvent(added(svc(host1, 443, "192.168.1.10")))
	c.WaitEvents()
	if len(c.Devices()) != 0 {
		t.Fatal("events after StopSearch must be ignored")
	}

	cancel()
	for range events {
	} // closed after cancel
}

func TestLANAddManualDevice(t *testing.T) {
	ip := netip.MustParseAddr("192.168.1.50")
	props := func(m map[uint16]int, calls *[]uint16) func(uint16, uint8) int {
		return func(id uint16, sub uint8) int {
			if sub != 0 {
				t.Fatalf("sub %d", sub)
			}
			*calls = append(*calls, id)
			return m[id]
		}
	}

	t.Run("login failure", func(t *testing.T) {
		c := NewLANConnection()
		log := collect(c)
		if _, ok := c.AddManualDevice(context.Background(), ip, 443, "", 2, false, nil); ok {
			t.Fatal("nil login must fail")
		}
		fail := func(context.Context, *Device) (func(uint16, uint8) int, bool) { return nil, false }
		if _, ok := c.AddManualDevice(context.Background(), ip, 443, "", 2, false, fail); ok {
			t.Fatal("failed login must not add")
		}
		if len(c.Devices()) != 0 || len(log.take(c)) != 0 {
			t.Fatal("nothing must be added")
		}
	})

	t.Run("success", func(t *testing.T) {
		c := NewLANConnection()
		log := collect(c)
		var calls []uint16
		var seen Device
		login := func(_ context.Context, d *Device) (func(uint16, uint8) int, bool) {
			seen = *d
			d.Identity = "FROM-8275"
			return props(map[uint16]int{8286: 2, 8485: 1, 12581: 3, 16919: meterP1, 21015: meterTCPIPSmart}, &calls), true
		}
		d, ok := c.AddManualDevice(context.Background(), ip, 443, "NG910_60023", 1, true, login)
		if !ok {
			t.Fatal("not added")
		}
		if seen.HostName != "NG910_60023" || seen.NumberOfSockets != 1 || !sameVersion(seen.FirmwareVersion, ver(5, 0, 0, -1)) ||
			seen.Discovered || !seen.IsManuallyAdded || seen.Identity != "" {
			t.Fatalf("device at login %+v", seen)
		}
		if d.Key == 0 || d.NumberOfSockets != 2 || d.SocketTypes != [2]int{1, 3} || !d.HasCentralMeter || !d.HasSmartMeter || d.Identity != "FROM-8275" {
			t.Fatalf("device %+v", d)
		}
		if !slices.Equal(calls, []uint16{8286, 8485, 12581, 16919, 21015}) {
			t.Fatalf("property reads %v", calls)
		}
		evs := log.take(c)
		if len(evs) != 1 || evs[0].Kind != DevicesChanged || evs[0].Action != CollectionAdd {
			t.Fatalf("events %+v (no DeviceRegistered for manual devices)", evs)
		}
		if !c.HasDeviceWithAddress(ip) {
			t.Fatal("HasDeviceWithAddress")
		}
		// The C# early-return compares IPAddress by reference and never fires.
		d2, ok := c.AddManualDevice(context.Background(), ip, 443, "", 2, false, login)
		if !ok || d2.Key == d.Key || len(c.Devices()) != 2 {
			t.Fatal("same address added twice, as in the C#")
		}
	})

	t.Run("single socket skips 12581", func(t *testing.T) {
		c := NewLANConnection()
		var calls []uint16
		login := func(context.Context, *Device) (func(uint16, uint8) int, bool) {
			return props(map[uint16]int{8286: 1, 8485: 0, 12581: 9}, &calls), true
		}
		d, _ := c.AddManualDevice(context.Background(), ip, 443, "", 2, false, login)
		if d.SocketTypes[1] != 0 || slices.Contains(calls, 12581) {
			t.Fatalf("device %+v calls %v", d, calls)
		}
		if d.HasCentralMeter != true { // 16919 missing => 0 => MODBUS_CENTRAL
			t.Fatal("GetPropertyInt default 0 is ENERGYMETER_MODBUS_CENTRAL")
		}
	})

	meters := []struct {
		central, smart        int
		wantCentral, wantSmrt bool
	}{
		{meterModbusCentral, -1, true, false},
		{meterFKN, 6, true, false},
		{meterTCPIPCentral, 7, true, false},
		{meterP1, meterP1, true, true},
		{meterTCPIPSmart, meterTCPIPSmart, false, true},
		{6, 0, false, false},
		{7, 3, false, false},
		{-1, 2, false, false},
	}
	for _, m := range meters {
		c := NewLANConnection()
		var calls []uint16
		login := func(context.Context, *Device) (func(uint16, uint8) int, bool) {
			return props(map[uint16]int{8286: 1, 16919: m.central, 21015: m.smart}, &calls), true
		}
		d, _ := c.AddManualDevice(context.Background(), ip, 443, "", 2, false, login)
		if d.HasCentralMeter != m.wantCentral || d.HasSmartMeter != m.wantSmrt {
			t.Fatalf("central %d smart %d -> %v %v", m.central, m.smart, d.HasCentralMeter, d.HasSmartMeter)
		}
	}
}

func TestLANRemoveManualAndRemoveDevice(t *testing.T) {
	c := NewLANConnection()
	log := collect(c)
	c.onBrowserEvent(added(svc(host1, 443, "192.168.1.10")))
	c.onBrowserEvent(added(svc("ng910-60023-2", 443, "192.168.1.12")))
	log.take(c)
	devs := c.Devices()

	c.RemoveManualDevice(Device{}) // null
	c.RemoveManualDevice(devs[0])
	evs := log.take(c)
	if len(evs) != 1 || evs[0].Kind != DevicesChanged || evs[0].Action != CollectionRemove {
		t.Fatalf("RemoveManualDevice events %+v", evs)
	}

	c.UpdateDevice(devs[1].Key, func(d *Device) { d.IsRebooting = true })
	c.RemoveDevice(devs[1])
	if evs := log.take(c); len(evs) != 0 {
		t.Fatalf("rebooting device removed %+v", evs)
	}
	c.UpdateDevice(devs[1].Key, func(d *Device) { d.IsRebooting = false })
	c.RemoveDevice(devs[1])
	if k := log.kinds(c); !slices.Equal(k, []DeviceEventKind{DeviceUnregistered, DevicesChanged}) {
		t.Fatalf("RemoveDevice kinds %v", k)
	}
	// Not registered (any more): the C# still raises DeviceUnregistered.
	c.RemoveDevice(devs[1])
	if k := log.kinds(c); !slices.Equal(k, []DeviceEventKind{DeviceUnregistered}) {
		t.Fatalf("kinds %v", k)
	}
	if len(c.Devices()) != 0 {
		t.Fatal("registry not empty")
	}
}

func TestLANFind(t *testing.T) {
	c := NewLANConnection()
	c.onBrowserEvent(added(svc(host1, 443, "192.168.1.10", "scnnetwork=Parking-A", "identity=Pole 1")))
	c.onBrowserEvent(added(svc("ng910-60023-2", 443, "fe80::1", "scnnetwork=parking-a")))
	c.onBrowserEvent(added(svc("ng910-60023-3", 443, "192.168.1.30")))

	if d, ok := c.FindLanDevice(netip.MustParseAddr("192.168.1.10")); !ok || d.HostName != host1 {
		t.Fatalf("FindLanDevice %+v %v", d, ok)
	}
	if _, ok := c.FindLanDevice(netip.MustParseAddr("FE80::1")); !ok {
		t.Fatal("IPv6 lookup")
	}
	if _, ok := c.FindLanDevice(netip.MustParseAddr("192.168.1.99")); ok {
		t.Fatal("unexpected match")
	}
	if got := c.FindDevicesInSCN("PARKING-A"); len(got) != 2 {
		t.Fatalf("FindDevicesInSCN %+v", got)
	}
	if got := c.FindDevicesInSCN("other"); len(got) != 0 {
		t.Fatalf("FindDevicesInSCN %+v", got)
	}
	if !c.HasDeviceWithAddress(netip.MustParseAddr("192.168.1.30")) || c.HasDeviceWithAddress(netip.MustParseAddr("192.168.1.31")) {
		t.Fatal("HasDeviceWithAddress")
	}

	d, _ := c.FindLanDevice(netip.MustParseAddr("192.168.1.10"))
	for filter, want := range map[string]bool{"": true, "60023": true, "POLE": true, "168.1.10": true, "nope": false} {
		if got := DeviceMatchesFilter(d, filter); got != want {
			t.Fatalf("filter %q = %v", filter, got)
		}
	}
}

func TestLANCallErrorHandler(t *testing.T) {
	c := NewLANConnection()
	log := collect(c)
	c.onBrowserEvent(added(svc(host1, 443, "192.168.1.10")))
	log.take(c)
	d, _ := c.FindLanDevice(netip.MustParseAddr("192.168.1.10"))
	c.CallErrorHandler("Device 'x' connection lost.", d)
	evs := log.take(c)
	if len(evs) != 1 || evs[0].Kind != ConnectionError || evs[0].Message != "Device 'x' connection lost." {
		t.Fatalf("events %+v", evs)
	}
	c.CallErrorHandler("ignored", Device{HostName: "unknown-1"})
	if evs := log.take(c); len(evs) != 0 {
		t.Fatalf("events %+v", evs)
	}
}

func TestLANNetworkInterfaces(t *testing.T) {
	c := NewLANConnection()
	en0 := net.Interface{Index: 4, Name: "en0"}
	c.onBrowserEvent(ServiceEvent{Kind: NetworkInterfaceAdded, NetworkInterface: en0})
	en0b := en0
	en0b.MTU = 9000
	c.onBrowserEvent(ServiceEvent{Kind: NetworkInterfaceAdded, NetworkInterface: en0b}) // same Id: replaced
	c.onBrowserEvent(ServiceEvent{Kind: NetworkInterfaceAdded, NetworkInterface: net.Interface{Index: 5, Name: "en1"}})
	ifs := c.NetworkInterfaces()
	if len(ifs) != 2 || ifs[0].Name != "en0" || ifs[0].MTU != 9000 || ifs[1].Name != "en1" {
		t.Fatalf("interfaces %+v", ifs)
	}
	c.onBrowserEvent(ServiceEvent{Kind: NetworkInterfaceRemoved, NetworkInterface: en0})
	if ifs := c.NetworkInterfaces(); len(ifs) != 1 || ifs[0].Name != "en1" {
		t.Fatalf("interfaces %+v", ifs)
	}
}

func TestLANEmptyTXTQuirk(t *testing.T) {
	// ReInitialize sets HostName = "" when the announcement has no TXT
	// strings, so such a device can neither be matched on re-announcement
	// (it is registered again) nor removed by host name. Ported as is.
	c := NewLANConnection()
	a := ServiceAnnouncement{Hostname: host1, Port: 443, Addresses: []netip.Addr{netip.MustParseAddr("192.168.1.10")}, Txt: []string{}}
	c.onBrowserEvent(added(a))
	c.onBrowserEvent(added(a))
	devs := c.Devices()
	if len(devs) != 2 || devs[0].HostName != "" || devs[0].Discovered {
		t.Fatalf("devices %+v", devs)
	}
	c.onBrowserEvent(removed(a))
	if len(c.Devices()) != 2 {
		t.Fatal("cannot be removed by host name")
	}
	// A single empty TXT string is enough to adopt the host name.
	c2 := NewLANConnection()
	a.Txt = []string{""}
	c2.onBrowserEvent(added(a))
	c2.onBrowserEvent(added(a))
	if devs := c2.Devices(); len(devs) != 1 || devs[0].HostName != host1 || !devs[0].Discovered {
		t.Fatalf("devices %+v", devs)
	}
}

func TestLANSubscribeUnsubscribe(t *testing.T) {
	c := NewLANConnection()
	n := 0
	unsub := c.Subscribe(func(DeviceEvent) { n++ })
	c.onBrowserEvent(added(svc(host1, 443, "192.168.1.10")))
	c.WaitEvents()
	unsub()
	c.onBrowserEvent(added(svc("x-2", 443, "192.168.1.11")))
	c.WaitEvents()
	if n != 2 {
		t.Fatalf("delivered %d, want 2 (Add + Registered before unsubscribe)", n)
	}
	if c.Name() != "Ethernet" || c.DeviceFound() {
		t.Fatal("BaseConnection members")
	}
}
