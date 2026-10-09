package discovery

import (
	"net"
	"net/netip"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---- DNS packet builder (with RFC 1035 name compression) ----

type section int

const (
	secAnswer section = iota
	secAuthority
	secAdditional
)

type pkt struct {
	b     []byte
	names map[string]int
	qd    int
	cnt   [3]int
	noCmp bool
}

func newPkt(id, flags uint16) *pkt {
	p := &pkt{b: make([]byte, 12), names: map[string]int{}}
	p.b[0], p.b[1] = byte(id>>8), byte(id)
	p.b[2], p.b[3] = byte(flags>>8), byte(flags)
	return p
}

// response: QR=1, AA=1 (0x8400), as mDNS responders send.
func newResponse() *pkt { return newPkt(0, 0x8400) }

func (p *pkt) u16(v uint16) { p.b = append(p.b, byte(v>>8), byte(v)) }
func (p *pkt) u32(v uint32) { p.b = append(p.b, byte(v>>24), byte(v>>16), byte(v>>8), byte(v)) }

// name writes a dotted name ("a.b.local"), compressing known suffixes.
func (p *pkt) name(n string) {
	n = strings.TrimSuffix(n, ".")
	labels := strings.Split(n, ".")
	if n == "" {
		labels = nil
	}
	for i := range labels {
		suffix := strings.ToLower(strings.Join(labels[i:], "."))
		if off, ok := p.names[suffix]; ok && !p.noCmp {
			p.b = append(p.b, 0xC0|byte(off>>8), byte(off))
			return
		}
		if len(p.b) < 0x3FFF {
			p.names[suffix] = len(p.b)
		}
		p.b = append(p.b, byte(len(labels[i])))
		p.b = append(p.b, labels[i]...)
	}
	p.b = append(p.b, 0)
}

func (p *pkt) question(n string, t recordType) *pkt {
	p.name(n)
	p.u16(uint16(t))
	p.u16(1)
	p.qd++
	return p
}

func (p *pkt) rr(sec section, n string, t recordType, ttl uint32, rdata func()) *pkt {
	p.name(n)
	p.u16(uint16(t))
	p.u16(1)
	p.u32(ttl)
	lenAt := len(p.b)
	p.u16(0)
	start := len(p.b)
	rdata()
	l := len(p.b) - start
	p.b[lenAt], p.b[lenAt+1] = byte(l>>8), byte(l)
	p.cnt[sec]++
	return p
}

func (p *pkt) ptr(sec section, n string, ttl uint32, target string) *pkt {
	return p.rr(sec, n, typePTR, ttl, func() { p.name(target) })
}

func (p *pkt) srv(sec section, n string, ttl uint32, port uint16, target string) *pkt {
	return p.rr(sec, n, typeSRV, ttl, func() { p.u16(0); p.u16(0); p.u16(port); p.name(target) })
}

func (p *pkt) txt(sec section, n string, ttl uint32, strs ...string) *pkt {
	return p.rr(sec, n, typeTXT, ttl, func() {
		for _, s := range strs {
			p.b = append(p.b, byte(len(s)))
			p.b = append(p.b, s...)
		}
	})
}

func (p *pkt) a(sec section, n string, ttl uint32, ip string) *pkt {
	addr := netip.MustParseAddr(ip)
	t := typeA
	if addr.Is6() {
		t = typeAAAA
	}
	return p.rr(sec, n, t, ttl, func() { p.b = append(p.b, addr.AsSlice()...) })
}

func (p *pkt) bytes() []byte {
	out := append([]byte(nil), p.b...)
	out[4], out[5] = byte(p.qd>>8), byte(p.qd)
	for i, c := range p.cnt {
		out[6+2*i], out[7+2*i] = byte(c>>8), byte(c)
	}
	return out
}

// announce builds the typical DNS-SD response of a charger: PTR answer plus
// SRV, TXT and A in the additional section.
func announce(svcType, instance, host string, port uint16, ip string, txt ...string) []byte {
	full := instance + "." + svcType + ".local"
	return newResponse().
		ptr(secAnswer, svcType+".local", 4500, full).
		srv(secAdditional, full, 120, port, host+".local").
		txt(secAdditional, full, 4500, txt...).
		a(secAdditional, host+".local", 120, ip).
		bytes()
}

// ---- fake clock ----

type fakeTimer struct {
	at      time.Time
	f       func()
	stopped bool
	fired   bool
	seq     int
}

func (t *fakeTimer) Stop() bool {
	was := !t.stopped && !t.fired
	t.stopped = true
	return was
}

type fakeClock struct {
	mu     sync.Mutex
	now    time.Time
	timers []*fakeTimer
	seq    int
}

func newFakeClock() *fakeClock {
	return &fakeClock{now: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *fakeClock) AfterFunc(d time.Duration, f func()) stopper {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.seq++
	t := &fakeTimer{at: c.now.Add(d), f: f, seq: c.seq}
	c.timers = append(c.timers, t)
	return t
}

// advance moves time forward, firing due timers (in due order) synchronously.
func (c *fakeClock) advance(d time.Duration) {
	c.mu.Lock()
	target := c.now.Add(d)
	c.mu.Unlock()
	for {
		c.mu.Lock()
		var due []*fakeTimer
		for _, t := range c.timers {
			if !t.stopped && !t.fired && !t.at.After(target) {
				due = append(due, t)
			}
		}
		if len(due) == 0 {
			c.now = target
			c.mu.Unlock()
			return
		}
		sort.Slice(due, func(i, j int) bool {
			if due[i].at.Equal(due[j].at) {
				return due[i].seq < due[j].seq
			}
			return due[i].at.Before(due[j].at)
		})
		t := due[0]
		t.fired = true
		if t.at.After(c.now) {
			c.now = t.at
		}
		c.mu.Unlock()
		t.f()
	}
}

// ---- fake packet conn ----

type sentPacket struct {
	data []byte
	to   net.Addr
}

type fakeConn struct {
	name   string
	in     chan []byte
	mu     sync.Mutex
	sent   []sentPacket
	closed chan struct{}
	once   sync.Once
}

func newFakeConn(name string) *fakeConn {
	return &fakeConn{name: name, in: make(chan []byte, 16), closed: make(chan struct{})}
}

func (c *fakeConn) ReadFrom(b []byte) (int, net.Addr, error) {
	select {
	case p := <-c.in:
		return copy(b, p), &net.UDPAddr{IP: net.IPv4(192, 168, 1, 50), Port: MDNSPort}, nil
	case <-c.closed:
		return 0, nil, net.ErrClosed
	}
}

func (c *fakeConn) WriteTo(b []byte, addr net.Addr) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sent = append(c.sent, sentPacket{data: append([]byte(nil), b...), to: addr})
	return len(b), nil
}

func (c *fakeConn) Close() error {
	c.once.Do(func() { close(c.closed) })
	return nil
}

func (c *fakeConn) isClosed() bool {
	select {
	case <-c.closed:
		return true
	default:
		return false
	}
}

func (c *fakeConn) sentPackets() []sentPacket {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]sentPacket(nil), c.sent...)
}

// ---- test browser / handler ----

type eventLog struct {
	mu  sync.Mutex
	evs []ServiceEvent
}

func (l *eventLog) add(ev ServiceEvent) {
	l.mu.Lock()
	l.evs = append(l.evs, ev)
	l.mu.Unlock()
}

func (l *eventLog) take() []ServiceEvent {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := l.evs
	l.evs = nil
	return out
}

func (l *eventLog) kinds() []ServiceEventKind {
	var ks []ServiceEventKind
	for _, e := range l.take() {
		ks = append(ks, e.Kind)
	}
	return ks
}

var testIface = net.Interface{Index: 7, MTU: 1500, Name: "en7", Flags: net.FlagUp | net.FlagRunning | net.FlagMulticast | net.FlagBroadcast}

// newTestBrowser returns a browser with a fake clock, synchronous event
// delivery and fake sockets (conns records every socket opened).
func newTestBrowser(t *testing.T) (*Browser, *fakeClock, *eventLog, *[]*fakeConn) {
	t.Helper()
	b := NewBrowser()
	fc := newFakeClock()
	b.clock = fc
	b.post = func(f func()) { f() }
	conns := &[]*fakeConn{}
	var mu sync.Mutex
	b.openConn = func(ifi net.Interface, v6 bool) (packetConn, error) {
		mu.Lock()
		defer mu.Unlock()
		name := ifi.Name + "/v4"
		if v6 {
			name = ifi.Name + "/v6"
		}
		c := newFakeConn(name)
		*conns = append(*conns, c)
		return c, nil
	}
	b.listInterfaces = func() ([]ifaceInfo, error) {
		return []ifaceInfo{{Interface: testIface, HasIPv4: true}}, nil
	}
	log := &eventLog{}
	b.Subscribe(log.add)
	return b, fc, log, conns
}

// newTestHandler builds a handler for the installer's service types with an
// (unread) fake IPv4 socket attached, so onReceive accepts packets.
func newTestHandler(t *testing.T) (*ifaceHandler, *fakeConn, *fakeClock, *eventLog) {
	t.Helper()
	b, fc, log, _ := newTestBrowser(t)
	p := b.QueryParameters()
	p.Robustness = SearchRobustness
	p.ResponseTime = SearchResponseTime
	b.SetQueryParameters(p)
	var names []dnsName
	for _, st := range ServiceTypes {
		names = append(names, parseName(strings.ToLower(st)+".local."))
	}
	h := newIfaceHandler(b, testIface.Index, testIface, names)
	c := newFakeConn("en7/v4")
	h.conn4 = c
	return h, c, fc, log
}
