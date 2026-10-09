package discovery

import (
	"net/netip"
	"slices"
	"strconv"
	"testing"
	"time"
)

const (
	alfen = "_alfen._tcp"
	lolo3 = "_lolo3._http._tcp"
	host1 = "ng910-60023-1234567"
)

func onlyEvent(t *testing.T, log *eventLog, kind ServiceEventKind) ServiceAnnouncement {
	t.Helper()
	evs := log.take()
	if len(evs) != 1 || evs[0].Kind != kind {
		t.Fatalf("events = %+v, want exactly one %v", evs, kind)
	}
	return evs[0].Announcement
}

func noEvents(t *testing.T, log *eventLog) {
	t.Helper()
	if evs := log.take(); len(evs) != 0 {
		t.Fatalf("unexpected events %+v", evs)
	}
}

func TestHandlerFullAnnouncement(t *testing.T) {
	tests := []struct {
		svcType  string
		wantType string
	}{
		{alfen, "_alfen._tcp"},
		{lolo3, "_lolo3._http"}, // Name.SubName(1, 2) keeps two labels
	}
	for _, tc := range tests {
		t.Run(tc.svcType, func(t *testing.T) {
			h, c, _, log := newTestHandler(t)
			h.onReceive(c, announce(tc.svcType, "ACE0012345", host1, 443, "192.168.1.10", "identity=ACE0012345", "type=2.1.1"))
			a := onlyEvent(t, log, ServiceAdded)
			want := ServiceAnnouncement{
				Instance:         "ACE0012345",
				Type:             tc.wantType,
				Domain:           "local.",
				Hostname:         host1,
				Port:             443,
				Addresses:        []netip.Addr{netip.MustParseAddr("192.168.1.10")},
				NetworkInterface: testIface,
				Txt:              []string{"identity=ACE0012345", "type=2.1.1"},
			}
			if a.Instance != want.Instance || a.Type != want.Type || a.Domain != want.Domain ||
				a.Hostname != want.Hostname || a.Port != want.Port || a.IsRemoved ||
				!slices.Equal(a.Addresses, want.Addresses) || !slices.Equal(a.Txt, want.Txt) ||
				a.NetworkInterface.Name != testIface.Name {
				t.Fatalf("announcement = %+v\nwant %+v", a, want)
			}
			if got := h.b.Services(); len(got) != 1 || got[0].Hostname != host1 {
				t.Fatalf("Services() = %+v", got)
			}
		})
	}
}

func TestHandlerIncrementalCompletion(t *testing.T) {
	h, c, _, log := newTestHandler(t)
	inst := "ACE." + alfen + ".local"

	// 1. PTR only: service known but incomplete.
	h.onReceive(c, newResponse().ptr(secAnswer, alfen+".local", 4500, inst).bytes())
	noEvents(t, log)
	if h.serviceInfos.len() != 1 {
		t.Fatalf("serviceInfos = %d", h.serviceInfos.len())
	}

	// 2. SRV + TXT, no address: still incomplete; the host is now queried.
	h.onReceive(c, newResponse().
		srv(secAnswer, inst, 120, 443, host1+".local").
		txt(secAnswer, inst, 4500, "identity=ACE").bytes())
	noEvents(t, log)
	pkt, sent := h.buildQueryPacket()
	if !sent {
		t.Fatal("expected a query")
	}
	qs := parseQuestions(t, pkt)
	wantQs := []string{"_lolo3._http._tcp.local./12", "_alfen._tcp.local./12", host1 + ".local./1", host1 + ".local./28"}
	if !slices.Equal(qs, wantQs) {
		t.Fatalf("questions = %q, want %q", qs, wantQs)
	}

	// 3. The A record completes it.
	h.onReceive(c, newResponse().a(secAnswer, host1+".local", 120, "10.1.2.3").bytes())
	a := onlyEvent(t, log, ServiceAdded)
	if !slices.Equal(a.Addresses, []netip.Addr{netip.MustParseAddr("10.1.2.3")}) || a.Port != 443 {
		t.Fatalf("announcement = %+v", a)
	}
	if _, sent := h.buildQueryPacket(); !sent {
		t.Fatal("start-up queries still pending")
	}
}

func parseQuestions(t *testing.T, pkt []byte) []string {
	t.Helper()
	r := newMsgReader(pkt)
	hdr, err := r.readHeader()
	if err != nil || !hdr.isQuery() || hdr.Flags != 0 {
		t.Fatalf("query header %+v %v", hdr, err)
	}
	var out []string
	for i := 0; i < int(hdr.QuestionCount); i++ {
		q, err := r.readQuestion()
		if err != nil {
			t.Fatal(err)
		}
		if q.QClass != classInternet {
			t.Fatalf("class %d", q.QClass)
		}
		out = append(out, q.QName.String()+"/"+strconv.Itoa(int(q.QType)))
	}
	return out
}

func TestHandlerChangedAndUnchanged(t *testing.T) {
	h, c, _, log := newTestHandler(t)
	h.onReceive(c, announce(alfen, "ACE", host1, 443, "192.168.1.10", "identity=A"))
	onlyEvent(t, log, ServiceAdded)

	// Identical re-announcement: nothing changes, no event.
	h.onReceive(c, announce(alfen, "ACE", host1, 443, "192.168.1.10", "identity=A"))
	noEvents(t, log)

	// TXT change -> ServiceChanged.
	h.onReceive(c, announce(alfen, "ACE", host1, 443, "192.168.1.10", "identity=B"))
	a := onlyEvent(t, log, ServiceChanged)
	if !slices.Equal(a.Txt, []string{"identity=B"}) {
		t.Fatalf("txt = %q", a.Txt)
	}

	// Port change -> ServiceChanged.
	h.onReceive(c, announce(alfen, "ACE", host1, 80, "192.168.1.10", "identity=B"))
	if a := onlyEvent(t, log, ServiceChanged); a.Port != 80 {
		t.Fatalf("port = %d", a.Port)
	}

	// Address change -> ServiceChanged with the new list.
	h.onReceive(c, announce(alfen, "ACE", host1, 80, "192.168.1.11", "identity=B"))
	if a := onlyEvent(t, log, ServiceChanged); !slices.Equal(a.Addresses, []netip.Addr{netip.MustParseAddr("192.168.1.11")}) {
		t.Fatalf("addresses = %v", a.Addresses)
	}

	// SRV target moves to another host: host info is swapped.
	const host2 = "eve-single-pro-line-7654321"
	h.onReceive(c, announce(alfen, "ACE", host2, 80, "192.168.1.12", "identity=B"))
	a = onlyEvent(t, log, ServiceChanged)
	if a.Hostname != host2 || !slices.Equal(a.Addresses, []netip.Addr{netip.MustParseAddr("192.168.1.12")}) {
		t.Fatalf("moved = %+v", a)
	}
	if h.hostInfos.has(parseName(host1 + ".local.").key()) {
		t.Fatal("old host info must be dropped")
	}
}

func TestHandlerGoodbyes(t *testing.T) {
	inst := "ACE." + alfen + ".local"
	tests := []struct {
		name    string
		goodbye []byte
	}{
		{"PTR ttl 0", newResponse().ptr(secAnswer, alfen+".local", 0, inst).bytes()},
		{"SRV ttl 0", newResponse().srv(secAnswer, inst, 0, 443, host1+".local").bytes()},
		{"TXT ttl 0", newResponse().txt(secAnswer, inst, 0, "x=y").bytes()},
		{"last address ttl 0", newResponse().a(secAnswer, host1+".local", 0, "192.168.1.10").bytes()},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h, c, _, log := newTestHandler(t)
			h.onReceive(c, announce(alfen, "ACE", host1, 443, "192.168.1.10", "identity=A"))
			onlyEvent(t, log, ServiceAdded)
			h.onReceive(c, tc.goodbye)
			a := onlyEvent(t, log, ServiceRemoved)
			if !a.IsRemoved || a.Hostname != host1 {
				t.Fatalf("removed = %+v", a)
			}
			if h.serviceInfos.len() != 0 || len(h.b.Services()) != 0 {
				t.Fatal("service must be gone")
			}
		})
	}
}

func TestHandlerAddressGoodbyeKeepsOtherFamily(t *testing.T) {
	h, c, _, log := newTestHandler(t)
	inst := "ACE." + alfen + ".local"
	h.onReceive(c, newResponse().
		ptr(secAnswer, alfen+".local", 4500, inst).
		srv(secAdditional, inst, 120, 443, host1+".local").
		txt(secAdditional, inst, 4500, "identity=A").
		a(secAdditional, host1+".local", 120, "192.168.1.10").
		a(secAdditional, host1+".local", 120, "fe80::10").bytes())
	a := onlyEvent(t, log, ServiceAdded)
	if len(a.Addresses) != 2 || !a.Addresses[0].Is4() || !a.Addresses[1].Is6() {
		t.Fatalf("addresses = %v (IPv4 first, then IPv6)", a.Addresses)
	}
	h.onReceive(c, newResponse().a(secAnswer, host1+".local", 0, "192.168.1.10").bytes())
	a = onlyEvent(t, log, ServiceChanged)
	if !slices.Equal(a.Addresses, []netip.Addr{netip.MustParseAddr("fe80::10")}) {
		t.Fatalf("addresses = %v", a.Addresses)
	}
}

func TestHandlerRobustnessRemoval(t *testing.T) {
	h, c, fc, log := newTestHandler(t) // Robustness 10, ResponseTime 2 s
	h.onReceive(c, announce(alfen, "ACE", host1, 443, "192.168.1.10", "identity=A"))
	onlyEvent(t, log, ServiceAdded)

	h.mu.Lock()
	h.startQuery()
	h.mu.Unlock()
	// Queries at t=0, 5 (start-up), then every 10 s: the 10th unanswered
	// query is at t=85, removal ResponseTime later at t=87.
	fc.advance(86 * time.Second)
	noEvents(t, log)
	if got := len(c.sentPackets()); got != 10 {
		t.Fatalf("queries sent = %d, want 10", got)
	}
	fc.advance(time.Second)
	onlyEvent(t, log, ServiceRemoved)
}

func TestHandlerRobustnessResetByAnswer(t *testing.T) {
	h, c, fc, log := newTestHandler(t)
	h.onReceive(c, announce(alfen, "ACE", host1, 443, "192.168.1.10", "identity=A"))
	onlyEvent(t, log, ServiceAdded)
	h.mu.Lock()
	h.startQuery()
	h.mu.Unlock()
	fc.advance(50 * time.Second)
	// Any answer for the service resets its open query count.
	h.onReceive(c, announce(alfen, "ACE", host1, 443, "192.168.1.10", "identity=A"))
	fc.advance(40 * time.Second) // past t=87
	noEvents(t, log)
	fc.advance(60 * time.Second) // 10 more unanswered queries
	onlyEvent(t, log, ServiceRemoved)
}

func TestHandlerForeignQueriesCount(t *testing.T) {
	h, c, fc, log := newTestHandler(t)
	h.onReceive(c, announce(alfen, "ACE", host1, 443, "192.168.1.10", "identity=A"))
	onlyEvent(t, log, ServiceAdded)

	h.lastQueryID = 0x4242
	own := newPkt(0x4242, 0).question(alfen+".local", typePTR).bytes()
	for i := 0; i < 20; i++ {
		h.onReceive(c, own) // our own looped-back query: ignored
	}
	fc.advance(SearchResponseTime)
	noEvents(t, log)

	foreign := newPkt(7, 0).question("_ALFEN._TCP.LOCAL", typePTR).bytes() // case-insensitive
	for i := 0; i < SearchRobustness; i++ {
		h.onReceive(c, foreign)
	}
	fc.advance(SearchResponseTime - time.Millisecond)
	noEvents(t, log)
	fc.advance(time.Millisecond)
	onlyEvent(t, log, ServiceRemoved)
}

func TestHandlerMalformedPacketDiscardedWhole(t *testing.T) {
	h, c, _, log := newTestHandler(t)
	good := announce(alfen, "ACE", host1, 443, "192.168.1.10", "identity=A")

	truncated := append([]byte(nil), good...)
	truncated[11]++ // one more additional record than present
	h.onReceive(c, truncated)
	noEvents(t, log)
	if h.serviceInfos.len() != 0 {
		t.Fatal("a packet that fails to parse must not be applied")
	}

	// A PTR of a foreign type is still decoded (before the type check), so
	// broken PTR RDATA anywhere poisons the whole packet.
	inst := "ACE." + alfen + ".local"
	p := newResponse()
	p.ptr(secAnswer, alfen+".local", 4500, inst).
		srv(secAdditional, inst, 120, 443, host1+".local").
		txt(secAdditional, inst, 4500, "identity=A").
		a(secAdditional, host1+".local", 120, "192.168.1.10").
		rr(secAdditional, "_http._tcp.local", typePTR, 10, func() { p.b = append(p.b, 0xC0) })
	h.onReceive(c, p.bytes())
	noEvents(t, log)

	// SRV/TXT RDATA of a foreign type is never decoded (ReadRecordHeader
	// skips it), so it cannot break the packet.
	p = newResponse()
	p.rr(secAnswer, "x._http._tcp.local", typeSRV, 10, func() { p.b = append(p.b, 0xC0) }).
		rr(secAnswer, "x._http._tcp.local", typeTXT, 10, func() { p.b = append(p.b, 9) }).
		ptr(secAnswer, alfen+".local", 4500, inst).
		srv(secAdditional, inst, 120, 443, host1+".local").
		txt(secAdditional, inst, 4500, "identity=A").
		a(secAdditional, host1+".local", 120, "192.168.1.10")
	h.onReceive(c, p.bytes())
	onlyEvent(t, log, ServiceAdded)
}

func TestHandlerIgnoresNonNoErrorAndStaleSockets(t *testing.T) {
	h, c, _, log := newTestHandler(t)
	good := announce(alfen, "ACE", host1, 443, "192.168.1.10", "identity=A")
	bad := append([]byte(nil), good...)
	bad[3] |= 0x03 // NXDOMAIN
	h.onReceive(c, bad)
	noEvents(t, log)
	h.onReceive(newFakeConn("other"), good)
	noEvents(t, log)
	h.onReceive(c, good)
	onlyEvent(t, log, ServiceAdded)
}

func TestHandlerUnknownHostBreaksAddressUpdate(t *testing.T) {
	// Tmds quirk (HandlePacketHostAddresses uses break, not continue): an
	// address record for an unknown host stops the address update of the
	// hosts after it in the same packet.
	h, c, _, log := newTestHandler(t)
	h.onReceive(c, announce(alfen, "ACE", host1, 443, "192.168.1.10", "identity=A"))
	onlyEvent(t, log, ServiceAdded)

	h.onReceive(c, newResponse().
		a(secAnswer, "unrelated.local", 120, "192.168.1.99").
		a(secAnswer, host1+".local", 120, "192.168.1.20").bytes())
	noEvents(t, log)

	h.onReceive(c, newResponse().a(secAnswer, host1+".local", 120, "192.168.1.20").bytes())
	a := onlyEvent(t, log, ServiceChanged)
	if !slices.Equal(a.Addresses, []netip.Addr{netip.MustParseAddr("192.168.1.20")}) {
		t.Fatalf("addresses = %v", a.Addresses)
	}
}

func TestHandlerIPv6Zone(t *testing.T) {
	h, c, _, log := newTestHandler(t)
	h.conn6 = newFakeConn("en7/v6")
	h.ipv6Zone = "en7"
	h.onReceive(c, announce(alfen, "ACE", host1, 443, "fe80::abcd", "identity=A"))
	a := onlyEvent(t, log, ServiceAdded)
	if len(a.Addresses) != 1 || a.Addresses[0].Zone() != "en7" {
		t.Fatalf("addresses = %v, want zone en7 (ScopeId = interface)", a.Addresses)
	}
}

func TestHandlerQuerySchedule(t *testing.T) {
	h, c4, fc, _ := newTestHandler(t)
	c6 := newFakeConn("en7/v6")
	h.conn6 = c6
	h.ipv6Zone = "en7"
	h.mu.Lock()
	h.startQuery()
	h.mu.Unlock()

	steps := []struct {
		advance time.Duration
		total   int
	}{
		{0, 1},                       // immediate first query
		{4999 * time.Millisecond, 1}, // StartQueryInterval
		{time.Millisecond, 2},
		{9999 * time.Millisecond, 2}, // QueryInterval after StartQueryCount
		{time.Millisecond, 3},
		{10 * time.Second, 4},
	}
	for i, s := range steps {
		fc.advance(s.advance)
		if got := len(c4.sentPackets()); got != s.total {
			t.Fatalf("step %d: %d IPv4 queries, want %d", i, got, s.total)
		}
		if got := len(c6.sentPackets()); got != s.total {
			t.Fatalf("step %d: %d IPv6 queries, want %d", i, got, s.total)
		}
	}
	p4 := c4.sentPackets()[0]
	if p4.to.String() != "224.0.0.251:5353" {
		t.Fatalf("IPv4 destination %v", p4.to)
	}
	if p6 := c6.sentPackets()[0]; p6.to.String() != "[ff02::fb%en7]:5353" {
		t.Fatalf("IPv6 destination %v", p6.to)
	}
	if qs := parseQuestions(t, p4.data); !slices.Equal(qs, []string{"_lolo3._http._tcp.local./12", "_alfen._tcp.local./12"}) {
		t.Fatalf("questions %q", qs)
	}
	// The transaction id is the (random) _lastQueryId.
	h.mu.Lock()
	last := h.lastQueryID
	h.mu.Unlock()
	if got := c4.sentPackets()[len(c4.sentPackets())-1]; uint16(got.data[0])<<8|uint16(got.data[1]) != last {
		t.Fatalf("last query id mismatch")
	}

	// stopQuery: no more packets.
	h.mu.Lock()
	h.stopQuery()
	h.mu.Unlock()
	fc.advance(time.Minute)
	if got := len(c4.sentPackets()); got != 4 {
		t.Fatalf("queries after stop = %d", got)
	}
}

func TestHandlerDisable(t *testing.T) {
	h, c, fc, log := newTestHandler(t)
	h.onReceive(c, announce(alfen, "ACE", host1, 443, "192.168.1.10", "identity=A"))
	h.onReceive(c, newResponse().ptr(secAnswer, lolo3+".local", 4500, "L."+lolo3+".local").bytes()) // incomplete
	onlyEvent(t, log, ServiceAdded)
	h.mu.Lock()
	h.startQuery()
	h.mu.Unlock()

	h.disable()
	a := onlyEvent(t, log, ServiceRemoved) // only complete services are reported
	if a.Hostname != host1 || !a.IsRemoved {
		t.Fatalf("removed %+v", a)
	}
	if !c.isClosed() || h.conn4 != nil || h.serviceInfos.len() != 0 || h.hostInfos.len() != 0 {
		t.Fatal("disable must close sockets and clear caches")
	}
	h.onReceive(c, announce(alfen, "ACE", host1, 443, "192.168.1.10", "identity=A"))
	noEvents(t, log)
	n := len(c.sentPackets())
	fc.advance(time.Minute)
	if len(c.sentPackets()) != n {
		t.Fatal("query timer must be stopped")
	}
}

func TestHandlerRefreshOpensSockets(t *testing.T) {
	b, fc, _, conns := newTestBrowser(t)
	h := newIfaceHandler(b, testIface.Index, testIface, []dnsName{parseName("_alfen._tcp.local.")})
	h.refresh(ifaceInfo{Interface: testIface, HasIPv4: true, HasIPv6: true})
	if len(*conns) != 2 || h.conn4 == nil || h.conn6 == nil || h.ipv6Zone != "en7" {
		t.Fatalf("conns = %d", len(*conns))
	}
	fc.advance(0)
	for _, c := range *conns {
		if len(c.sentPackets()) != 1 {
			t.Fatalf("%s: expected the immediate start-up query", c.name)
		}
	}
	// Already open: refresh is a no-op.
	h.refresh(ifaceInfo{Interface: testIface, HasIPv4: true, HasIPv6: true})
	if len(*conns) != 2 {
		t.Fatal("refresh must not reopen sockets")
	}
	h.disable()
}
