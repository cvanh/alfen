package discovery

import (
	"context"
	"errors"
	"net"
	"slices"
	"sync"
	"testing"
	"time"
)

func iface(index int, name string, flags net.Flags) net.Interface {
	return net.Interface{Index: index, Name: name, MTU: 1500, Flags: flags}
}

const upMC = net.FlagUp | net.FlagRunning | net.FlagMulticast

func connByName(conns []*fakeConn, name string) *fakeConn {
	for _, c := range conns {
		if c.name == name {
			return c
		}
	}
	return nil
}

func connNames(conns []*fakeConn) []string {
	var out []string
	for _, c := range conns {
		out = append(out, c.name)
	}
	return out
}

func TestBrowserInterfaceSelection(t *testing.T) {
	b, fc, log, conns := newTestBrowser(t)
	ifs := []ifaceInfo{
		{Interface: iface(1, "lo0", upMC|net.FlagLoopback), HasIPv4: true, HasIPv6: true},
		{Interface: iface(4, "en0", upMC|net.FlagBroadcast), HasIPv4: true, HasIPv6: true},
		{Interface: iface(9, "utun0", upMC|net.FlagPointToPoint), HasIPv6: true}, // tunnel
		{Interface: iface(5, "en5", net.FlagUp|net.FlagRunning), HasIPv4: true},  // no multicast
		{Interface: iface(6, "en6", upMC)},                                       // no IPv4/IPv6
		{Interface: iface(8, "en8", net.FlagMulticast), HasIPv4: true},           // down
	}
	var mu sync.Mutex
	b.listInterfaces = func() ([]ifaceInfo, error) {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(ifs), nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := b.StartBrowse(ctx, alfen); err != nil {
		t.Fatal(err)
	}
	defer b.StopBrowse()

	var added []string
	for _, ev := range log.take() {
		if ev.Kind != NetworkInterfaceAdded {
			t.Fatalf("unexpected %v", ev.Kind)
		}
		added = append(added, ev.NetworkInterface.Name)
	}
	if !slices.Equal(added, []string{"lo0", "en0", "en8"}) {
		t.Fatalf("interfaces added %q", added)
	}
	if got := connNames(*conns); !slices.Equal(got, []string{"lo0/v4", "lo0/v6", "en0/v4", "en0/v6"}) {
		t.Fatalf("sockets %q (down interface must stay closed)", got)
	}
	fc.advance(0)
	if len(connByName(*conns, "en0/v4").sentPackets()) != 1 {
		t.Fatal("expected immediate query on en0")
	}

	// en0 learns a service, then disappears; en8 comes up.
	en0 := connByName(*conns, "en0/v4")
	b.mu.Lock()
	h := b.handlers[4]
	b.mu.Unlock()
	h.onReceive(en0, announce(alfen, "ACE", host1, 443, "192.168.1.10", "identity=A"))
	if k := log.kinds(); !slices.Equal(k, []ServiceEventKind{ServiceAdded}) {
		t.Fatalf("events %v", k)
	}
	mu.Lock()
	ifs = slices.Delete(ifs, 1, 2)
	ifs[len(ifs)-1].Interface.Flags = upMC
	mu.Unlock()
	b.mu.Lock()
	b.checkNetworkInterfaceStatuses()
	b.mu.Unlock()
	evs := log.take()
	if len(evs) != 2 || evs[0].Kind != ServiceRemoved || evs[1].Kind != NetworkInterfaceRemoved || evs[1].NetworkInterface.Name != "en0" {
		t.Fatalf("events %+v", evs)
	}
	if !en0.isClosed() || connByName(*conns, "en8/v4") == nil {
		t.Fatalf("sockets %q", connNames(*conns))
	}
}

func TestBrowserStartStop(t *testing.T) {
	b, _, log, conns := newTestBrowser(t)
	ctx := context.Background()
	if err := b.StartBrowse(ctx, alfen, lolo3); err != nil {
		t.Fatal(err)
	}
	if !b.IsBrowsing() {
		t.Fatal("not browsing")
	}
	if err := b.StartBrowse(ctx, alfen); !errors.Is(err, ErrAlreadyBrowsing) {
		t.Fatalf("second StartBrowse: %v", err)
	}
	b.mu.Lock()
	h := b.handlers[testIface.Index]
	b.mu.Unlock()
	c := (*conns)[0]
	h.onReceive(c, announce(lolo3, "L", "lolo3-99", 80, "192.168.1.30", "identity=L"))
	log.take()
	if len(b.Services()) != 1 {
		t.Fatal("service not tracked")
	}

	b.StopBrowse()
	if k := log.kinds(); !slices.Equal(k, []ServiceEventKind{ServiceRemoved, NetworkInterfaceRemoved}) {
		t.Fatalf("events %v", k)
	}
	if b.IsBrowsing() || len(b.Services()) != 0 || !c.isClosed() {
		t.Fatal("StopBrowse must disable everything")
	}
	b.StopBrowse() // idempotent

	// Restart works and re-reads the service type list (cleared on stop).
	if err := b.StartBrowse(ctx, "_ALFEN._tcp"); err != nil {
		t.Fatal(err)
	}
	defer b.StopBrowse()
	b.mu.Lock()
	h = b.handlers[testIface.Index]
	b.mu.Unlock()
	if h.serviceHandlers.len() != 1 || !h.serviceHandlers.has(parseName("_alfen._tcp.local.").key()) {
		t.Fatal("service type must be lower-cased + \".local.\"")
	}
}

func TestBrowserContextCancel(t *testing.T) {
	b, _, _, _ := newTestBrowser(t)
	pre, cancelPre := context.WithCancel(context.Background())
	cancelPre()
	if err := b.StartBrowse(pre, alfen); err == nil {
		t.Fatal("cancelled context must be rejected")
	}

	ctx, cancel := context.WithCancel(context.Background())
	if err := b.StartBrowse(ctx, alfen); err != nil {
		t.Fatal(err)
	}
	cancel()
	deadline := time.Now().Add(2 * time.Second)
	for b.IsBrowsing() {
		if time.Now().After(deadline) {
			t.Fatal("ctx cancel did not stop browsing")
		}
		time.Sleep(5 * time.Millisecond)
	}
	// A stale watcher of the old session must not stop a new one.
	if err := b.StartBrowse(context.Background(), alfen); err != nil {
		t.Fatal(err)
	}
	time.Sleep(20 * time.Millisecond)
	if !b.IsBrowsing() {
		t.Fatal("new session stopped by old context")
	}
	b.StopBrowse()
}

func TestBrowserEndToEndAsync(t *testing.T) {
	// Real dispatcher and receive loop; only sockets and interfaces are fake.
	b := NewBrowser()
	var mu sync.Mutex
	var conns []*fakeConn
	b.openConn = func(ifi net.Interface, v6 bool) (packetConn, error) {
		mu.Lock()
		defer mu.Unlock()
		c := newFakeConn(ifi.Name)
		conns = append(conns, c)
		return c, nil
	}
	b.listInterfaces = func() ([]ifaceInfo, error) { return []ifaceInfo{{Interface: testIface, HasIPv4: true}}, nil }
	got := make(chan ServiceEvent, 8)
	b.Subscribe(func(ev ServiceEvent) {
		if ev.Kind == ServiceAdded {
			got <- ev
		}
	})
	if err := b.StartBrowse(context.Background(), ServiceTypes...); err != nil {
		t.Fatal(err)
	}
	defer b.StopBrowse()
	mu.Lock()
	c := conns[0]
	mu.Unlock()
	c.in <- announce(alfen, "ACE", host1, 443, "192.168.1.10", "identity=ACE")
	select {
	case ev := <-got:
		if ev.Announcement.Hostname != host1 || ev.Announcement.Port != 443 {
			t.Fatalf("%+v", ev.Announcement)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no ServiceAdded")
	}
	// The real query timer sent the start-up query right away.
	deadline := time.Now().Add(2 * time.Second)
	for len(c.sentPackets()) == 0 {
		if time.Now().After(deadline) {
			t.Fatal("no query sent")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestBrowserListInterfacesError(t *testing.T) {
	b, _, log, conns := newTestBrowser(t)
	b.listInterfaces = func() ([]ifaceInfo, error) { return nil, errors.New("boom") }
	if err := b.StartBrowse(context.Background(), alfen); err != nil {
		t.Fatal(err)
	}
	defer b.StopBrowse()
	if len(log.take()) != 0 || len(*conns) != 0 {
		t.Fatal("no interfaces expected")
	}
}

func TestBrowserOpenConnErrorIsolated(t *testing.T) {
	b, _, _, _ := newTestBrowser(t)
	var opened []string
	b.openConn = func(ifi net.Interface, v6 bool) (packetConn, error) {
		if v6 {
			return nil, errors.New("no v6")
		}
		opened = append(opened, ifi.Name)
		return newFakeConn(ifi.Name), nil
	}
	b.listInterfaces = func() ([]ifaceInfo, error) {
		return []ifaceInfo{
			{Interface: iface(2, "en0", upMC), HasIPv4: true, HasIPv6: true},
			{Interface: iface(3, "en1", upMC), HasIPv4: true},
		}, nil
	}
	if err := b.StartBrowse(context.Background(), alfen); err != nil {
		t.Fatal(err)
	}
	defer b.StopBrowse()
	if !slices.Equal(opened, []string{"en0", "en1"}) {
		t.Fatalf("opened %q: an IPv6 failure must not stop other sockets/interfaces", opened)
	}
}
