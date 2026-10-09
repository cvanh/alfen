package discovery

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/netip"
	"sort"
	"strings"
	"sync"
	"time"
)

// ServiceAnnouncement ports Tmds.MDns.ServiceAnnouncement. Values are
// snapshots: the browser never mutates an announcement it has handed out.
type ServiceAnnouncement struct {
	// Instance is the first label of the service instance name.
	Instance string
	// Type is labels 1..2 of the instance name (Name.SubName(1, 2)), e.g.
	// "_alfen._tcp". For "_lolo3._http._tcp" this is "_lolo3._http", as in Tmds.
	Type string
	// Domain is the SRV target minus its first label, e.g. "local.".
	Domain string
	// Hostname is the first label of the SRV target. The registry keys
	// devices on it; its last '-' separated part is the serial number.
	Hostname string
	// Port is the SRV port.
	Port uint16
	// Addresses are the host's A then AAAA addresses (IPv6 carry the
	// interface as zone). LANConnection uses Addresses[0].
	Addresses []netip.Addr
	// NetworkInterface is the interface the service was seen on.
	NetworkInterface net.Interface
	// Txt are the raw TXT strings in record order.
	Txt []string
	// IsRemoved is set on the announcement carried by a ServiceRemoved event.
	IsRemoved bool
}

// ServiceEventKind identifies a Browser event (the ServiceBrowser C# events).
type ServiceEventKind int

const (
	ServiceAdded ServiceEventKind = iota + 1
	ServiceChanged
	ServiceRemoved
	NetworkInterfaceAdded
	NetworkInterfaceRemoved
)

func (k ServiceEventKind) String() string {
	switch k {
	case ServiceAdded:
		return "ServiceAdded"
	case ServiceChanged:
		return "ServiceChanged"
	case ServiceRemoved:
		return "ServiceRemoved"
	case NetworkInterfaceAdded:
		return "NetworkInterfaceAdded"
	case NetworkInterfaceRemoved:
		return "NetworkInterfaceRemoved"
	}
	return "ServiceEventKind(?)"
}

// ServiceEvent ports ServiceAnnouncementEventArgs / NetworkInterfaceEventArgs.
type ServiceEvent struct {
	Kind ServiceEventKind
	// Announcement is set for ServiceAdded/ServiceChanged/ServiceRemoved.
	Announcement ServiceAnnouncement
	// NetworkInterface is set for NetworkInterfaceAdded/Removed.
	NetworkInterface net.Interface
}

// ErrAlreadyBrowsing ports the "Already browsing" exception of StartBrowse.
var ErrAlreadyBrowsing = errors.New("Already browsing")

// DefaultInterfacePollInterval is how often the browser re-checks the host's
// network interfaces. Tmds.MDns reacts to NetworkChange.NetworkAddressChanged /
// NetworkAvailabilityChanged; Go has no portable equivalent, so the same
// CheckNetworkInterfaceStatuses logic runs on this period instead.
const DefaultInterfacePollInterval = 5 * time.Second

// ifaceInfo is what CheckNetworkInterfaceStatuses reads per interface:
// NetworkInterface plus Supports(IPv4)/Supports(IPv6) (approximated by the
// interface having an address of that family).
type ifaceInfo struct {
	Interface net.Interface
	HasIPv4   bool
	HasIPv6   bool
}

// Browser ports Tmds.MDns.ServiceBrowser: it browses a set of DNS-SD service
// types on every multicast-capable interface and raises ServiceEvents. Events
// are delivered serially on an internal goroutine (the C# posts them to the
// SynchronizationContext of the StartBrowse caller, i.e. the UI thread).
type Browser struct {
	// Logger receives socket/interface errors. Nil discards them. Set it
	// before StartBrowse.
	Logger *slog.Logger
	// InterfacePollInterval overrides DefaultInterfacePollInterval when > 0.
	// Set it before StartBrowse.
	InterfacePollInterval time.Duration

	paramsMu sync.Mutex
	params   QueryParameters

	mu           sync.Mutex
	browsing     bool
	serviceTypes []string
	handlers     map[int]*ifaceHandler
	stopCh       chan struct{}
	gen          uint64

	annMu sync.Mutex
	anns  *netDict[ServiceAnnouncement] // _serviceAnnouncements (and _services)

	subs subscribers[ServiceEvent]
	disp *dispatcher

	// injectable for tests
	clock          clock
	listInterfaces func() ([]ifaceInfo, error)
	openConn       func(ifi net.Interface, v6 bool) (packetConn, error)
	post           func(func())
}

// NewBrowser ports the ServiceBrowser constructor (default QueryParameters).
func NewBrowser() *Browser {
	b := &Browser{
		params:         DefaultQueryParameters(),
		anns:           newNetDict[ServiceAnnouncement](),
		disp:           newDispatcher(),
		clock:          realClock{},
		listInterfaces: systemInterfaces,
		openConn:       listenMDNS,
	}
	b.post = b.disp.post
	return b
}

// QueryParameters returns the current query parameters.
func (b *Browser) QueryParameters() QueryParameters {
	b.paramsMu.Lock()
	defer b.paramsMu.Unlock()
	return b.params
}

// SetQueryParameters replaces the query parameters (ServiceBrowser.QueryParameters
// is a mutable object in the C#; LANConnection.StartSearch edits it in place).
// Running handlers pick the new values up on their next timer tick.
func (b *Browser) SetQueryParameters(p QueryParameters) {
	b.paramsMu.Lock()
	b.params = p
	b.paramsMu.Unlock()
}

// Subscribe adds an event handler (C# "+="); the returned func removes it
// ("-="). Handlers are invoked serially, never while a browser lock is held,
// so they may call back into the Browser.
func (b *Browser) Subscribe(fn func(ServiceEvent)) (unsubscribe func()) {
	return b.subs.add(fn)
}

// IsBrowsing ports ServiceBrowser.IsBrowsing.
func (b *Browser) IsBrowsing() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.browsing
}

// Services ports ServiceBrowser.Services: the currently complete services.
func (b *Browser) Services() []ServiceAnnouncement {
	b.annMu.Lock()
	defer b.annMu.Unlock()
	out := make([]ServiceAnnouncement, 0, b.anns.len())
	b.anns.each(func(_ string, a ServiceAnnouncement) bool {
		out = append(out, a)
		return true
	})
	return out
}

// StartBrowse ports ServiceBrowser.StartBrowse(IEnumerable<string>): browse
// "<type>.local." for every given type (e.g. "_alfen._tcp") on all usable
// interfaces. It fails with ErrAlreadyBrowsing while browsing. Cancelling ctx
// has the effect of StopBrowse.
func (b *Browser) StartBrowse(ctx context.Context, serviceTypes ...string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.browsing {
		return ErrAlreadyBrowsing
	}
	b.serviceTypes = append(b.serviceTypes, serviceTypes...)
	b.browsing = true
	b.gen++
	gen := b.gen
	b.handlers = map[int]*ifaceHandler{}
	stop := make(chan struct{})
	b.stopCh = stop
	b.checkNetworkInterfaceStatuses()
	poll := b.InterfacePollInterval
	if poll <= 0 {
		poll = DefaultInterfacePollInterval
	}
	go b.watch(ctx, gen, stop, poll)
	return nil
}

// watch replaces the NetworkChange subscriptions (periodic re-check) and
// implements ctx cancellation.
func (b *Browser) watch(ctx context.Context, gen uint64, stop chan struct{}, poll time.Duration) {
	t := time.NewTicker(poll)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ctx.Done():
			b.mu.Lock()
			if b.browsing && b.gen == gen {
				b.stopBrowseLocked()
			}
			b.mu.Unlock()
			return
		case <-t.C:
			b.mu.Lock()
			if b.browsing && b.gen == gen {
				b.checkNetworkInterfaceStatuses()
			}
			b.mu.Unlock()
		}
	}
}

// StopBrowse ports ServiceBrowser.StopBrowse: every interface handler is
// disabled (its complete services are reported removed) and reported as a
// removed interface; the service type list is cleared.
func (b *Browser) StopBrowse() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !b.browsing {
		return
	}
	b.stopBrowseLocked()
}

func (b *Browser) stopBrowseLocked() {
	b.browsing = false
	close(b.stopCh)
	b.stopCh = nil
	for _, h := range b.sortedHandlers() {
		h.disable()
		b.postEvent(ServiceEvent{Kind: NetworkInterfaceRemoved, NetworkInterface: h.iface})
	}
	b.handlers = nil
	b.serviceTypes = nil
}

func (b *Browser) sortedHandlers() []*ifaceHandler {
	hs := make([]*ifaceHandler, 0, len(b.handlers))
	for _, h := range b.handlers {
		hs = append(hs, h)
	}
	sort.Slice(hs, func(i, j int) bool { return hs[i].key < hs[j].key })
	return hs
}

// checkNetworkInterfaceStatuses ports ServiceBrowser.CheckNetworkInterfaceStatuses.
// Tunnel interfaces (here: point-to-point) and interfaces without multicast or
// without any IPv4/IPv6 address are skipped; a handler is created per new
// interface (NetworkInterfaceAdded), refreshed while the interface is up and
// disabled while it is down; vanished interfaces are disabled and reported
// removed. Caller holds b.mu.
func (b *Browser) checkNetworkInterfaceStatuses() {
	infos, err := b.listInterfaces()
	if err != nil {
		b.logError("list network interfaces", "", err)
		return
	}
	names := make([]dnsName, 0, len(b.serviceTypes))
	for _, st := range b.serviceTypes {
		names = append(names, parseName(strings.ToLower(st)+".local."))
	}
	remaining := map[*ifaceHandler]bool{}
	for _, h := range b.handlers {
		remaining[h] = true
	}
	for _, info := range infos {
		ifi := info.Interface
		if ifi.Flags&net.FlagPointToPoint != 0 || ifi.Flags&net.FlagMulticast == 0 {
			continue
		}
		if !info.HasIPv4 && !info.HasIPv6 {
			continue
		}
		h := b.handlers[ifi.Index]
		if h == nil {
			h = newIfaceHandler(b, ifi.Index, ifi, names)
			b.handlers[ifi.Index] = h
			b.postEvent(ServiceEvent{Kind: NetworkInterfaceAdded, NetworkInterface: ifi})
		}
		if ifi.Flags&net.FlagUp != 0 && ifi.Flags&net.FlagRunning != 0 {
			h.refresh(info)
		} else {
			h.disable()
		}
		delete(remaining, h)
	}
	gone := make([]*ifaceHandler, 0, len(remaining))
	for h := range remaining {
		gone = append(gone, h)
	}
	sort.Slice(gone, func(i, j int) bool { return gone[i].key < gone[j].key })
	for _, h := range gone {
		delete(b.handlers, h.key)
		h.disable()
		b.postEvent(ServiceEvent{Kind: NetworkInterfaceRemoved, NetworkInterface: h.iface})
	}
}

func annKey(h *ifaceHandler, name dnsName) string {
	return h.iface.Name + "\x00" + name.key()
}

// onServiceAdded ports ServiceBrowser.OnServiceAdded. Caller holds h.mu.
func (b *Browser) onServiceAdded(h *ifaceHandler, s *serviceInfo) {
	a := h.announcement(s)
	b.annMu.Lock()
	b.anns.add(annKey(h, s.name), a)
	b.annMu.Unlock()
	b.postEvent(ServiceEvent{Kind: ServiceAdded, Announcement: a})
}

// onServiceRemoved ports ServiceBrowser.OnServiceRemoved: the stored
// announcement is removed and raised with IsRemoved set. Caller holds h.mu.
func (b *Browser) onServiceRemoved(h *ifaceHandler, s *serviceInfo) {
	k := annKey(h, s.name)
	b.annMu.Lock()
	a, ok := b.anns.get(k)
	b.anns.remove(k)
	b.annMu.Unlock()
	if !ok {
		a = h.announcement(s)
	}
	a.IsRemoved = true
	b.postEvent(ServiceEvent{Kind: ServiceRemoved, Announcement: a})
}

// onServiceChanged ports ServiceBrowser.OnServiceChanged: the stored
// announcement takes the new values. Caller holds h.mu.
func (b *Browser) onServiceChanged(h *ifaceHandler, s *serviceInfo) {
	a := h.announcement(s)
	b.annMu.Lock()
	b.anns.add(annKey(h, s.name), a)
	b.annMu.Unlock()
	b.postEvent(ServiceEvent{Kind: ServiceChanged, Announcement: a})
}

// postEvent ports SynchronizationContextPost + raising the C# event.
func (b *Browser) postEvent(ev ServiceEvent) {
	b.post(func() { b.subs.deliver(ev) })
}

func (b *Browser) logError(what, iface string, err error) {
	if b.Logger != nil {
		b.Logger.Error("mdns: "+what, "interface", iface, "err", err)
	}
}

// waitIdle blocks until all queued events were delivered.
func (b *Browser) waitIdle() { b.disp.wait() }

// systemInterfaces lists the host interfaces with their address families.
func systemInterfaces() ([]ifaceInfo, error) {
	ifs, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	out := make([]ifaceInfo, 0, len(ifs))
	for _, ifi := range ifs {
		info := ifaceInfo{Interface: ifi}
		addrs, err := ifi.Addrs()
		if err == nil {
			for _, a := range addrs {
				var ip net.IP
				switch v := a.(type) {
				case *net.IPNet:
					ip = v.IP
				case *net.IPAddr:
					ip = v.IP
				}
				if ip == nil {
					continue
				}
				if ip.To4() != nil {
					info.HasIPv4 = true
				} else {
					info.HasIPv6 = true
				}
			}
		}
		out = append(out, info)
	}
	return out, nil
}

// listenMDNS ports CreateIpv4Socket / CreateIpv6Socket: a UDP socket bound to
// the wildcard address on port 5353 with address reuse, multicast egress set
// to the interface, joined to 224.0.0.251 / ff02::fb on that interface, with
// the default multicast TTL of 1. net.ListenMulticastUDP does exactly this
// (SO_REUSEADDR, plus SO_REUSEPORT on BSD so it coexists with mDNSResponder);
// unlike the C# it also disables multicast loopback, which only suppresses
// the browser's own queries (the C# ignores those by transaction id anyway).
func listenMDNS(ifi net.Interface, v6 bool) (packetConn, error) {
	network, group := "udp4", net.ParseIP(MDNSIPv4Addr)
	if v6 {
		network, group = "udp6", net.ParseIP(MDNSIPv6Addr)
	}
	c, err := net.ListenMulticastUDP(network, &ifi, &net.UDPAddr{IP: group, Port: MDNSPort})
	if err != nil {
		return nil, err
	}
	restrictToJoinedInterface(c, v6)
	return c, nil
}
