package discovery

import (
	"math/rand"
	"net"
	"net/netip"
	"slices"
	"sync"
	"time"
)

// QueryParameters ports Tmds.MDns.QueryParameters (the C# keeps milliseconds
// in ints; the values here are the same durations).
type QueryParameters struct {
	// StartQueryCount is the number of initial queries sent StartQueryInterval
	// apart before switching to QueryInterval.
	StartQueryCount int
	// StartQueryInterval is the query period during start-up.
	StartQueryInterval time.Duration
	// QueryInterval is the steady-state query period.
	QueryInterval time.Duration
	// ResponseTime is how long after a query a service that reached Robustness
	// unanswered queries is given before it is removed.
	ResponseTime time.Duration
	// Robustness is the number of consecutive unanswered queries after which a
	// service is removed.
	Robustness int
}

// DefaultQueryParameters ports the QueryParameters() constructor defaults.
func DefaultQueryParameters() QueryParameters {
	return QueryParameters{
		StartQueryCount:    2,
		StartQueryInterval: 5000 * time.Millisecond,
		QueryInterval:      10000 * time.Millisecond,
		ResponseTime:       1000 * time.Millisecond,
		Robustness:         2,
	}
}

// serviceInfo ports Tmds.MDns.ServiceInfo. nil hostName/addresses/txt and
// port == -1 stand for the C# nulls; an empty non-nil slice is distinct from
// nil, as List<T> is from null.
type serviceInfo struct {
	name           dnsName
	hostName       *dnsName
	port           int
	addresses      []netip.Addr
	txt            []string
	openQueryCount int
	lastQueryTime  time.Time
}

func newServiceInfo(name dnsName) *serviceInfo {
	return &serviceInfo{name: name, port: -1}
}

// isComplete ports ServiceInfo.IsComplete.
func (s *serviceInfo) isComplete() bool {
	return s.hostName != nil && s.port != -1 && s.addresses != nil && s.txt != nil
}

// hostInfo ports Tmds.MDns.HostInfo.
type hostInfo struct {
	name         dnsName
	serviceInfos []*serviceInfo
	addresses    []netip.Addr
}

// hostAddresses ports Tmds.MDns.HostAddresses.
type hostAddresses struct {
	ipv4 []netip.Addr
	ipv6 []netip.Addr
}

// serviceHandler ports Tmds.MDns.ServiceHandler.
type serviceHandler struct {
	name         dnsName
	serviceInfos []*serviceInfo
}

// packetConn is the subset of *net.UDPConn the handler uses (injectable for
// tests, which must not bind the real mDNS port).
type packetConn interface {
	ReadFrom(b []byte) (int, net.Addr, error)
	WriteTo(b []byte, addr net.Addr) (int, error)
	Close() error
}

// ifaceHandler ports Tmds.MDns.NetworkInterfaceHandler: one per network
// interface, owning that interface's IPv4/IPv6 sockets, its query timer and its
// service/host caches. All state is guarded by mu (the C# lock(this)).
type ifaceHandler struct {
	b     *Browser
	key   int
	iface net.Interface

	mu       sync.Mutex
	conn4    packetConn
	conn6    packetConn
	ipv6Zone string // _ipv6InterfaceIndex != -1 <=> ipv6Zone != ""

	packetServiceInfos  *netDict[packetService]
	packetHostAddresses *netDict[*hostAddresses]
	serviceInfos        *netDict[*serviceInfo]
	hostInfos           *netDict[*hostInfo]
	serviceHandlers     *netDict[*serviceHandler]

	queryCount  int
	queryTimer  stopper
	timerGen    uint64
	rnd         *rand.Rand
	lastQueryID uint16
}

// newIfaceHandler ports the NetworkInterfaceHandler constructor.
func newIfaceHandler(b *Browser, key int, iface net.Interface, names []dnsName) *ifaceHandler {
	h := &ifaceHandler{
		b:                   b,
		key:                 key,
		iface:               iface,
		packetServiceInfos:  newNetDict[packetService](),
		packetHostAddresses: newNetDict[*hostAddresses](),
		serviceInfos:        newNetDict[*serviceInfo](),
		hostInfos:           newNetDict[*hostInfo](),
		serviceHandlers:     newNetDict[*serviceHandler](),
		rnd:                 rand.New(rand.NewSource(time.Now().UnixNano() ^ int64(key))),
	}
	for _, n := range names {
		if !h.serviceHandlers.has(n.key()) {
			h.serviceHandlers.add(n.key(), &serviceHandler{name: n})
		}
	}
	return h
}

// refresh ports NetworkInterfaceHandler.Refresh: open whichever of the IPv4 /
// IPv6 sockets the interface supports and that is not open yet, start a
// receive loop on each, and restart the query sequence. Errors opening a
// socket are reported and leave that family closed (the C# lets the exception
// escape, aborting the interface scan); the next interface check retries.
func (h *ifaceHandler) refresh(info ifaceInfo) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.iface = info.Interface
	if (h.conn4 != nil || !info.HasIPv4) && (h.conn6 != nil || !info.HasIPv6) {
		return
	}
	opened := false
	if info.HasIPv4 && h.conn4 == nil {
		c, err := h.b.openConn(info.Interface, false)
		if err != nil {
			h.b.logError("open IPv4 mDNS socket", info.Interface.Name, err)
		} else {
			h.conn4 = c
			opened = true
			go h.receiveLoop(c)
		}
	}
	if info.HasIPv6 && h.conn6 == nil {
		c, err := h.b.openConn(info.Interface, true)
		if err != nil {
			h.b.logError("open IPv6 mDNS socket", info.Interface.Name, err)
		} else {
			h.conn6 = c
			h.ipv6Zone = info.Interface.Name
			opened = true
			go h.receiveLoop(c)
		}
	}
	if opened {
		h.startQuery()
	}
}

// disable ports NetworkInterfaceHandler.Disable: stop querying, close the
// sockets, report every complete service as removed and drop all caches.
func (h *ifaceHandler) disable() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.conn4 == nil && h.conn6 == nil {
		return
	}
	h.stopQuery()
	h.serviceHandlers.each(func(_ string, sh *serviceHandler) bool {
		sh.serviceInfos = nil
		return true
	})
	if h.conn4 != nil {
		h.conn4.Close()
		h.conn4 = nil
	}
	if h.conn6 != nil {
		h.conn6.Close()
		h.conn6 = nil
	}
	h.ipv6Zone = ""
	h.serviceInfos.each(func(_ string, s *serviceInfo) bool {
		if s.isComplete() {
			h.b.onServiceRemoved(h, s)
		}
		return true
	})
	h.serviceInfos.clear()
	h.hostInfos.clear()
}

// receiveLoop ports StartReceive/OnReceive's re-arm: read datagrams until the
// socket fails. As in the C# (args.SocketError != Success => return without
// re-arming), any read error ends the loop.
func (h *ifaceHandler) receiveLoop(c packetConn) {
	buf := make([]byte, recvBufferSize)
	for {
		n, _, err := c.ReadFrom(buf)
		if err != nil {
			return
		}
		h.onReceive(c, buf[:n])
	}
}

// send ports NetworkInterfaceHandler.Send/SendPackets: the packet goes out on
// both sockets, to the IPv4 and IPv6 group respectively; errors are ignored.
func (h *ifaceHandler) send(pkt []byte) {
	if h.conn4 != nil {
		_, _ = h.conn4.WriteTo(pkt, &net.UDPAddr{IP: net.ParseIP(MDNSIPv4Addr), Port: MDNSPort})
	}
	if h.conn6 != nil {
		_, _ = h.conn6.WriteTo(pkt, &net.UDPAddr{IP: net.ParseIP(MDNSIPv6Addr), Port: MDNSPort, Zone: h.ipv6Zone})
	}
}

// onServiceQuery ports NetworkInterfaceHandler.OnServiceQuery: every service
// of the queried type gets one more open query; those that reached
// Robustness are removed after ResponseTime unless they answer meanwhile
// (an answer resets openQueryCount). Caller holds h.mu.
func (h *ifaceHandler) onServiceQuery(serviceKey string) {
	sh, ok := h.serviceHandlers.get(serviceKey)
	if !ok {
		return
	}
	p := h.b.QueryParameters()
	now := h.b.clock.Now()
	var robustness []*serviceInfo
	for _, s := range sh.serviceInfos {
		s.openQueryCount++
		s.lastQueryTime = now
		if s.openQueryCount >= p.Robustness {
			robustness = append(robustness, s)
		}
	}
	if robustness == nil {
		return
	}
	h.b.clock.AfterFunc(p.ResponseTime, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		r := h.b.QueryParameters().Robustness
		for _, s := range robustness {
			if s.openQueryCount >= r {
				h.removeService(s.name)
			}
		}
	})
}

// onReceive ports NetworkInterfaceHandler.OnReceive. A parse error anywhere in
// the packet discards the whole packet (flag = false), but side effects that
// already happened (query counting) stay, as in the C#.
func (h *ifaceHandler) onReceive(c packetConn, data []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if c == nil || (c != h.conn4 && c != h.conn6) {
		return
	}
	h.packetServiceInfos.clear()
	h.packetHostAddresses.clear()
	if err := h.readPacket(newMsgReader(data)); err != nil {
		return
	}
	h.handlePacketHostAddresses()
	h.handlePacketServiceInfos()
}

// readPacket is the try { ... } body of OnReceive.
func (h *ifaceHandler) readPacket(r *msgReader) error {
	hdr, err := r.readHeader()
	if err != nil {
		return err
	}
	if hdr.isQuery() && hdr.AnswerCount == 0 {
		for i := 0; i < int(hdr.QuestionCount); i++ {
			q, err := r.readQuestion()
			if err != nil {
				return err
			}
			if h.serviceHandlers.has(q.QName.key()) && hdr.TransactionID != h.lastQueryID {
				h.onServiceQuery(q.QName.key())
			}
		}
	}
	if !(hdr.isResponse() && hdr.isNoError()) {
		return nil
	}
	for i := 0; i < int(hdr.QuestionCount); i++ {
		if _, err := r.readQuestion(); err != nil {
			return err
		}
	}
	total := int(hdr.AnswerCount) + int(hdr.AuthorityCount) + int(hdr.AdditionalCount)
	for i := 0; i < total; i++ {
		rh, err := r.readRecordHeader()
		if err != nil {
			return err
		}
		switch rh.Type {
		case typeA, typeAAAA:
			addr, err := r.readARecord()
			if err != nil {
				return err
			}
			if addr.Is6() && h.ipv6Zone != "" {
				addr = addr.WithZone(h.ipv6Zone) // iPAddress.ScopeId = _ipv6InterfaceIndex
			}
			h.onARecord(rh.Name, addr, rh.TTL)
		case typeSRV, typeTXT, typePTR:
			var serviceKey string
			var name dnsName
			if rh.Type == typePTR {
				serviceKey = rh.Name.key()
				if name, err = r.readPtrRecord(); err != nil {
					return err
				}
			} else {
				name = rh.Name
				serviceKey = name.subName(1).key()
			}
			if !h.serviceHandlers.has(serviceKey) {
				continue
			}
			if rh.TTL == 0 {
				h.packetRemovesService(name)
				continue
			}
			s := h.findOrCreatePacketService(name)
			switch rh.Type {
			case typeSRV:
				srv, err := r.readSrvRecord()
				if err != nil {
					return err
				}
				target := srv.Target
				s.hostName = &target
				s.port = int(srv.Port)
			case typeTXT:
				txt, err := r.readTxtRecord()
				if err != nil {
					return err
				}
				s.txt = txt
			}
		}
	}
	return nil
}

// onARecord ports NetworkInterfaceHandler.OnARecord: the per-family list is
// created even for a TTL-0 (goodbye) address, which is then simply not added.
func (h *ifaceHandler) onARecord(name dnsName, addr netip.Addr, ttl uint32) {
	ha, ok := h.packetHostAddresses.get(name.key())
	if !ok {
		ha = &hostAddresses{}
		h.packetHostAddresses.add(name.key(), ha)
	}
	var list *[]netip.Addr
	if addr.Is4() {
		if ha.ipv4 == nil {
			ha.ipv4 = []netip.Addr{}
		}
		list = &ha.ipv4
	} else {
		if ha.ipv6 == nil {
			ha.ipv6 = []netip.Addr{}
		}
		list = &ha.ipv6
	}
	if ttl != 0 {
		*list = append(*list, addr)
	}
}

// packetService is one _packetServiceInfos entry: the Name key the C#
// dictionary holds plus its value; info == nil means "removed by this packet".
type packetService struct {
	name dnsName
	info *serviceInfo
}

// packetRemovesService ports NetworkInterfaceHandler.PacketRemovesService.
func (h *ifaceHandler) packetRemovesService(name dnsName) {
	h.packetServiceInfos.remove(name.key())
	h.packetServiceInfos.add(name.key(), packetService{name: name})
}

// findOrCreatePacketService ports NetworkInterfaceHandler.FindOrCreatePacketService.
func (h *ifaceHandler) findOrCreatePacketService(name dnsName) *serviceInfo {
	e, found := h.packetServiceInfos.get(name.key())
	if e.info == nil {
		if found {
			h.packetServiceInfos.remove(name.key())
		}
		e = packetService{name: name, info: newServiceInfo(name)}
		h.packetServiceInfos.add(name.key(), e)
	}
	return e.info
}

// handlePacketHostAddresses ports NetworkInterfaceHandler.HandlePacketHostAddresses.
//
// Quirk kept from the C#: when an address record's host is not (yet) known the
// loop breaks rather than continues, so later hosts in the same packet are not
// updated. New services pick their packet addresses up in addServiceHostInfo.
func (h *ifaceHandler) handlePacketHostAddresses() {
	h.packetHostAddresses.each(func(key string, ha *hostAddresses) bool {
		hi, ok := h.hostInfos.get(key)
		if !ok {
			return false // break
		}
		list := []netip.Addr{}
		if ha.ipv4 == nil {
			for _, a := range hi.addresses {
				if a.Is4() {
					list = append(list, a)
				}
			}
		} else {
			list = append(list, ha.ipv4...)
		}
		if ha.ipv6 == nil {
			for _, a := range hi.addresses {
				if a.Is6() {
					list = append(list, a)
				}
			}
		} else {
			list = append(list, ha.ipv6...)
		}
		if len(list) == 0 {
			h.hostInfos.remove(key)
			for _, s := range hi.serviceInfos {
				h.packetRemovesService(s.name)
			}
			return true
		}
		if hi.addresses != nil && len(hi.addresses) == len(list) && allContained(hi.addresses, list) {
			return true
		}
		for _, s := range hi.serviceInfos {
			h.findOrCreatePacketService(s.name).addresses = list
		}
		hi.addresses = list
		return true
	})
}

func allContained(a, b []netip.Addr) bool {
	for _, x := range a {
		if !slices.Contains(b, x) {
			return false
		}
	}
	return true
}

// removeService ports NetworkInterfaceHandler.RemoveService.
func (h *ifaceHandler) removeService(name dnsName) {
	s, ok := h.serviceInfos.get(name.key())
	if !ok || s == nil {
		return
	}
	h.serviceInfos.remove(name.key())
	if sh, ok := h.serviceHandlers.get(name.subName(1).key()); ok {
		if i := slices.Index(sh.serviceInfos, s); i >= 0 {
			sh.serviceInfos = slices.Delete(sh.serviceInfos, i, i+1)
		}
	}
	if s.isComplete() {
		h.b.onServiceRemoved(h, s)
	}
	if s.hostName != nil {
		h.clearServiceHostInfo(s)
	}
}

// handlePacketServiceInfos ports NetworkInterfaceHandler.HandlePacketServiceInfos:
// merge the packet's view of each service into the cache and raise
// added/changed/removed depending on how IsComplete moved.
func (h *ifaceHandler) handlePacketServiceInfos() {
	h.packetServiceInfos.each(func(key string, e packetService) bool {
		value := e.info
		if value == nil {
			h.removeService(e.name)
			return true
		}
		changed := false
		wasComplete := false
		existing, ok := h.serviceInfos.get(key)
		if !ok {
			existing = value
			h.serviceInfos.add(key, existing)
			if sh, ok := h.serviceHandlers.get(e.name.subName(1).key()); ok {
				sh.serviceInfos = append(sh.serviceInfos, existing)
			}
			if existing.hostName != nil {
				h.addServiceHostInfo(existing)
			}
			changed = true
		} else {
			existing.openQueryCount = 0
			wasComplete = existing.isComplete()
			if value.port != -1 && existing.port != value.port {
				existing.port = value.port
				changed = true
			}
			if value.name.String() != existing.name.String() {
				existing.name = value.name
				changed = true
			}
			if value.txt != nil && (existing.txt == nil || !slices.Equal(value.txt, existing.txt)) {
				existing.txt = value.txt
				changed = true
			}
			if value.hostName != nil && (existing.hostName == nil || existing.hostName.String() != value.hostName.String()) {
				if existing.hostName != nil {
					h.clearServiceHostInfo(existing)
				}
				existing.hostName = value.hostName
				h.addServiceHostInfo(existing)
				changed = true
			}
			if value.addresses != nil {
				existing.addresses = value.addresses
				changed = true
			}
		}
		if !changed {
			return true
		}
		if wasComplete != existing.isComplete() {
			if wasComplete {
				h.b.onServiceRemoved(h, existing)
			} else {
				h.b.onServiceAdded(h, existing)
			}
		} else if existing.isComplete() {
			h.b.onServiceChanged(h, existing)
		}
		return true
	})
}

// clearServiceHostInfo ports NetworkInterfaceHandler.ClearServiceHostInfo.
// As in the C#, the service keeps its host name when the host is unknown.
func (h *ifaceHandler) clearServiceHostInfo(s *serviceInfo) {
	hk := s.hostName.key()
	hi, ok := h.hostInfos.get(hk)
	if !ok {
		return
	}
	if i := slices.Index(hi.serviceInfos, s); i >= 0 {
		hi.serviceInfos = slices.Delete(hi.serviceInfos, i, i+1)
	}
	if len(hi.serviceInfos) == 0 {
		h.hostInfos.remove(hk)
	}
	s.hostName = nil
	s.addresses = nil
}

// addServiceHostInfo ports NetworkInterfaceHandler.AddServiceHostInfo: a new
// host takes its addresses from the current packet (IPv4 first, then IPv6);
// the service shares the host's address list.
func (h *ifaceHandler) addServiceHostInfo(s *serviceInfo) {
	hk := s.hostName.key()
	hi, ok := h.hostInfos.get(hk)
	if !ok {
		hi = &hostInfo{name: *s.hostName}
		if ha, ok := h.packetHostAddresses.get(hk); ok {
			hi.addresses = ha.ipv4
			if hi.addresses == nil {
				hi.addresses = ha.ipv6
			} else if ha.ipv6 != nil {
				hi.addresses = append(hi.addresses, ha.ipv6...)
			}
		}
		h.hostInfos.add(hk, hi)
	}
	hi.serviceInfos = append(hi.serviceInfos, s)
	s.addresses = hi.addresses
}

// startQuery ports NetworkInterfaceHandler.StartQuery: restart the start-up
// query sequence with an immediate query.
func (h *ifaceHandler) startQuery() {
	h.queryCount = 0
	h.scheduleQueryTimer(0)
}

// stopQuery ports NetworkInterfaceHandler.StopQuery (Timer.Change(-1, -1)).
func (h *ifaceHandler) stopQuery() {
	h.timerGen++
	if h.queryTimer != nil {
		h.queryTimer.Stop()
		h.queryTimer = nil
	}
}

// scheduleQueryTimer ports ScheduleQueryTimer (one-shot Timer.Change(ms, -1)).
// A generation counter makes a callback that was already in flight when the
// timer was re-armed or stopped a no-op.
func (h *ifaceHandler) scheduleQueryTimer(d time.Duration) {
	h.stopQuery()
	gen := h.timerGen
	h.queryTimer = h.b.clock.AfterFunc(d, func() { h.onQueryTimerElapsed(gen) })
}

// onQueryTimerElapsed ports NetworkInterfaceHandler.OnQueryTimerElapsed.
func (h *ifaceHandler) onQueryTimerElapsed(gen uint64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if gen != h.timerGen {
		return
	}
	h.queryTimer = nil
	pkt, sent := h.buildQueryPacket()
	if sent {
		h.send(pkt)
		h.queryCount++
	}
	p := h.b.QueryParameters()
	next := p.StartQueryInterval
	if h.queryCount >= p.StartQueryCount {
		next = p.QueryInterval
	}
	h.scheduleQueryTimer(next)
}

// buildQueryPacket is the packet-building half of OnQueryTimerElapsed: a PTR
// question per service type that is still starting up, has no services, or
// has a service not queried for QueryInterval (each such type counts as a
// query for robustness), plus A and AAAA questions for every known host that
// has no addresses yet. Caller holds h.mu.
func (h *ifaceHandler) buildQueryPacket() ([]byte, bool) {
	p := h.b.QueryParameters()
	now := h.b.clock.Now()
	flag := false
	h.lastQueryID = uint16(h.rnd.Intn(65535)) // Random.Next(0, 65535)
	var qs []question
	h.serviceHandlers.each(func(key string, sh *serviceHandler) bool {
		need := false
		if h.queryCount < p.StartQueryCount {
			need = true
		} else if len(sh.serviceInfos) == 0 {
			need = true
		} else {
			threshold := now.Add(-p.QueryInterval)
			for _, s := range sh.serviceInfos {
				if !s.lastQueryTime.After(threshold) {
					need = true
				}
			}
		}
		if need {
			h.onServiceQuery(key)
			qs = append(qs, question{QName: sh.name, QType: typePTR, QClass: classInternet})
		}
		flag = flag || need
		return true
	})
	h.hostInfos.each(func(_ string, hi *hostInfo) bool {
		if hi.addresses == nil {
			qs = append(qs,
				question{QName: hi.name, QType: typeA, QClass: classInternet},
				question{QName: hi.name, QType: typeAAAA, QClass: classInternet})
			flag = true
		}
		return true
	})
	return buildQuery(h.lastQueryID, qs), flag
}

// announcement ports the ServiceAnnouncement initialiser in
// ServiceBrowser.OnServiceAdded/OnServiceChanged. Caller holds h.mu.
func (h *ifaceHandler) announcement(s *serviceInfo) ServiceAnnouncement {
	a := ServiceAnnouncement{
		Instance:         s.name.firstLabel(),
		Type:             s.name.subNameLen(1, 2).String(),
		Port:             uint16(s.port),
		Addresses:        slices.Clone(s.addresses),
		Txt:              slices.Clone(s.txt),
		NetworkInterface: h.iface,
	}
	if s.hostName != nil {
		a.Hostname = s.hostName.firstLabel()
		a.Domain = s.hostName.subName(1).String()
	}
	return a
}
