package discovery

import (
	"bytes"
	"net/netip"
	"slices"
	"testing"
)

func TestReadNameCompression(t *testing.T) {
	// header(12) | "_alfen._tcp.local" at 12 | "inst" + ptr(12) at 31 | ptr(31)
	msg := make([]byte, 12)
	msg = append(msg, 6, '_', 'a', 'l', 'f', 'e', 'n', 4, '_', 't', 'c', 'p', 5, 'l', 'o', 'c', 'a', 'l', 0)
	instAt := len(msg)
	msg = append(msg, 4, 'i', 'n', 's', 't', 0xC0, 12)
	ptrAt := len(msg)
	msg = append(msg, 0xC0, byte(instAt), 0xAA)

	tests := []struct {
		name    string
		at      int
		want    []string
		wantPos int
	}{
		{"plain", 12, []string{"_alfen", "_tcp", "local", ""}, instAt},
		{"label+pointer", instAt, []string{"inst", "_alfen", "_tcp", "local", ""}, ptrAt},
		{"pointer chain", ptrAt, []string{"inst", "_alfen", "_tcp", "local", ""}, ptrAt + 2},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := newMsgReader(msg)
			r.pos = tc.at
			n, err := r.readName()
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(n.labels, tc.want) {
				t.Fatalf("labels = %q, want %q", n.labels, tc.want)
			}
			if r.pos != tc.wantPos {
				t.Fatalf("pos = %d, want %d (resume after first pointer)", r.pos, tc.wantPos)
			}
		})
	}
	if got := (dnsName{labels: []string{"inst", "_alfen", "_tcp", "local", ""}}).String(); got != "inst._alfen._tcp.local." {
		t.Fatalf("String() = %q", got)
	}
}

func TestReadNameMalformed(t *testing.T) {
	tests := []struct {
		name string
		msg  []byte
	}{
		{"self pointer", []byte{0xC0, 0x00}},
		{"two-pointer loop", []byte{0xC0, 0x02, 0xC0, 0x00}},
		{"label then loop back", []byte{1, 'a', 0xC0, 0x00}},
		{"pointer past end", []byte{0xC0, 0x50}},
		{"truncated pointer", []byte{0xC0}},
		{"label overruns", []byte{5, 'a', 'b'}},
		{"no terminator", []byte{1, 'a'}},
		{"empty", nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := newMsgReader(tc.msg).readName(); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestReadNameForwardPointerAllowed(t *testing.T) {
	// Tmds does not require pointers to point backwards.
	msg := []byte{0xC0, 0x03, 0xFF, 1, 'x', 0}
	n, err := newMsgReader(msg).readName()
	if err != nil {
		t.Fatal(err)
	}
	if n.String() != "x." {
		t.Fatalf("got %q", n.String())
	}
}

func TestNameKeyAndSubName(t *testing.T) {
	a := parseName("_alfen._tcp.local.")
	b := dnsName{labels: []string{"_ALFEN", "_TCP", "Local", ""}}
	if a.key() != b.key() {
		t.Fatal("names must compare OrdinalIgnoreCase")
	}
	inst := parseName("ACE-1._lolo3._http._tcp.local.")
	if got := inst.subName(1).String(); got != "_lolo3._http._tcp.local." {
		t.Fatalf("subName(1) = %q", got)
	}
	if got := inst.subNameLen(1, 2).String(); got != "_lolo3._http" {
		t.Fatalf("subNameLen(1,2) = %q (Tmds quirk: Type keeps two labels)", got)
	}
	short := dnsName{labels: []string{"x"}}
	if got := short.subNameLen(1, 2).String(); got != "" {
		t.Fatalf("clamped subNameLen = %q", got)
	}
	if got := short.subName(5).String(); got != "" {
		t.Fatalf("clamped subName = %q", got)
	}
}

func TestReadRecordHeaderSkipsUnreadRdata(t *testing.T) {
	// The first record (unknown type 99) has 3 bytes of RDATA nobody reads.
	b := newResponse()
	b.name("a.local")
	b.u16(99)
	b.u16(1)
	b.u32(5)
	b.u16(3)
	b.b = append(b.b, 1, 2, 3)
	b.cnt[secAnswer]++
	b.a(secAnswer, "b.local", 7, "10.0.0.9")
	r := newMsgReader(b.bytes())
	if _, err := r.readHeader(); err != nil {
		t.Fatal(err)
	}
	rh1, err := r.readRecordHeader()
	if err != nil || rh1.Type != 99 || rh1.DataLength != 3 || rh1.TTL != 5 {
		t.Fatalf("rh1 = %+v, %v", rh1, err)
	}
	rh2, err := r.readRecordHeader()
	if err != nil {
		t.Fatal(err)
	}
	if rh2.Name.String() != "b.local." || rh2.Type != typeA || rh2.TTL != 7 {
		t.Fatalf("rh2 = %+v", rh2)
	}
	ip, err := r.readARecord()
	if err != nil || ip != netip.MustParseAddr("10.0.0.9") {
		t.Fatalf("ip = %v, %v", ip, err)
	}
}

func TestReadARecordLengths(t *testing.T) {
	tests := []struct {
		rdata []byte
		want  string
		ok    bool
	}{
		{[]byte{192, 168, 1, 10}, "192.168.1.10", true},
		{netip.MustParseAddr("fe80::1").AsSlice(), "fe80::1", true},
		{[]byte{1, 2, 3}, "", false},
		{nil, "", false},
	}
	for _, tc := range tests {
		r := newMsgReader(tc.rdata)
		r.recordLength = len(tc.rdata)
		got, err := r.readARecord()
		if (err == nil) != tc.ok {
			t.Fatalf("%v: err = %v", tc.rdata, err)
		}
		if tc.ok && got.String() != tc.want {
			t.Fatalf("got %v want %v", got, tc.want)
		}
	}
}

func TestReadTxtRecord(t *testing.T) {
	tests := []struct {
		name  string
		rdata []byte
		want  []string
		ok    bool
	}{
		{"two strings", []byte{3, 'a', '=', 'b', 2, 'c', 'd'}, []string{"a=b", "cd"}, true},
		{"empty string kept", []byte{0, 1, 'x'}, []string{"", "x"}, true},
		{"zero rdata", nil, []string{}, true},
		{"overrun", []byte{5, 'a'}, nil, false},
		{"invalid utf8 replaced", []byte{2, 0xff, 'a'}, []string{"�a"}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := newMsgReader(tc.rdata)
			r.recordLength = len(tc.rdata)
			got, err := r.readTxtRecord()
			if (err == nil) != tc.ok {
				t.Fatalf("err = %v", err)
			}
			if tc.ok {
				if got == nil {
					t.Fatal("TXT list must be non-nil (List<string>, not null)")
				}
				if !slices.Equal(got, tc.want) {
					t.Fatalf("got %q want %q", got, tc.want)
				}
			}
		})
	}
}

func TestReadSrvRecord(t *testing.T) {
	p := newResponse().srv(secAnswer, "i._alfen._tcp.local", 120, 443, "ng910-60023-1234567.local")
	r := newMsgReader(p.bytes())
	if _, err := r.readHeader(); err != nil {
		t.Fatal(err)
	}
	rh, err := r.readRecordHeader()
	if err != nil || rh.Type != typeSRV {
		t.Fatal(rh, err)
	}
	srv, err := r.readSrvRecord()
	if err != nil {
		t.Fatal(err)
	}
	if srv.Port != 443 || srv.Target.String() != "ng910-60023-1234567.local." {
		t.Fatalf("srv = %+v", srv)
	}
}

func TestBuildQueryExactBytes(t *testing.T) {
	got := buildQuery(0x1234, []question{
		{QName: parseName("_alfen._tcp.local."), QType: typePTR, QClass: classInternet},
		{QName: dnsName{labels: []string{"host", "local", ""}}, QType: typeA, QClass: classInternet},
		{QName: parseName("x.local"), QType: typeAAAA, QClass: classInternet}, // no root label: 0 appended
	})
	want := []byte{
		0x12, 0x34, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0,
		6, '_', 'a', 'l', 'f', 'e', 'n', 4, '_', 't', 'c', 'p', 5, 'l', 'o', 'c', 'a', 'l', 0, 0, 12, 0, 1,
		4, 'h', 'o', 's', 't', 5, 'l', 'o', 'c', 'a', 'l', 0, 0, 1, 0, 1,
		1, 'x', 5, 'l', 'o', 'c', 'a', 'l', 0, 0, 28, 0, 1,
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("query =\n% x\nwant\n% x", got, want)
	}
	// And it round-trips through the reader as a query.
	r := newMsgReader(got)
	h, err := r.readHeader()
	if err != nil || !h.isQuery() || h.QuestionCount != 3 || h.TransactionID != 0x1234 {
		t.Fatalf("header %+v %v", h, err)
	}
	q, err := r.readQuestion()
	if err != nil || q.QName.String() != "_alfen._tcp.local." || q.QType != typePTR || q.QClass != classInternet {
		t.Fatalf("q = %+v %v", q, err)
	}
}

func TestHeaderFlags(t *testing.T) {
	if !(header{Flags: 0x8400}).isResponse() || !(header{Flags: 0x8400}).isNoError() {
		t.Fatal("0x8400 is a NOERROR response")
	}
	if (header{Flags: 0x8403}).isNoError() {
		t.Fatal("rcode 3 is not NOERROR")
	}
	if !(header{Flags: 0}).isQuery() {
		t.Fatal("0 is a query")
	}
}

// FuzzReceive feeds arbitrary bytes through the full receive path (parse +
// cache merge + event raising). It must never panic.
func FuzzReceive(f *testing.F) {
	f.Add(announce("_alfen._tcp", "ACE", "ng910-60023-1234567", 443, "192.168.1.10", "identity=ACE", "type=2.1.1"))
	f.Add(announce("_lolo3._http._tcp", "L", "lolo3-99", 80, "fe80::1", "fwversion=4.x.0"))
	f.Add(newResponse().ptr(secAnswer, "_alfen._tcp.local", 0, "x._alfen._tcp.local").bytes())
	f.Add(newPkt(7, 0).question("_alfen._tcp.local", typePTR).bytes())
	f.Add([]byte{0, 0, 0x84, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0xC0, 12})
	f.Fuzz(func(t *testing.T, data []byte) {
		h, c, fc, _ := newTestHandler(t)
		h.onReceive(c, data)
		h.onReceive(c, data) // second pass hits the "existing service" paths
		fc.advance(SearchResponseTime)
		_, _ = h.buildQueryPacket()
	})
}

// FuzzReadName exercises the name decoder on its own (compression pointers at
// arbitrary offsets).
func FuzzReadName(f *testing.F) {
	f.Add([]byte{0xC0, 0x00}, 0)
	f.Add([]byte{1, 'a', 0xC0, 0x00}, 2)
	f.Fuzz(func(t *testing.T, data []byte, start int) {
		r := newMsgReader(data)
		if start < 0 {
			start = -start
		}
		if len(data) > 0 {
			r.pos = start % (len(data) + 1)
		}
		n, err := r.readName()
		if err == nil && len(n.labels) > len(data) {
			t.Fatalf("%d labels from %d bytes", len(n.labels), len(data))
		}
	})
}
