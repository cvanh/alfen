package discovery

import (
	"errors"
	"net/netip"
	"strings"
)

// mDNS endpoints (NetworkInterfaceHandler.IPv4EndPoint / IPv6EndPoint).
const (
	MDNSPort     = 5353
	MDNSIPv4Addr = "224.0.0.251"
	MDNSIPv6Addr = "ff02::fb"
)

// recvBufferSize is the receive buffer of NetworkInterfaceHandler.CreateEventArgs
// (and the DnsMessageWriter buffer): new byte[9000].
const recvBufferSize = 9000

// recordType ports Tmds.MDns.RecordType.
type recordType uint16

const (
	typeA    recordType = 1
	typePTR  recordType = 12
	typeTXT  recordType = 16
	typeAAAA recordType = 28
	typeSRV  recordType = 33
	typeAll  recordType = 255
)

// recordClass ports Tmds.MDns.RecordClass.
type recordClass uint16

const (
	classInternet recordClass = 1
	classAll      recordClass = 255
)

var errMalformed = errors.New("discovery: malformed DNS message")

// dnsName ports Tmds.MDns.Name: a list of labels. Names read from the wire end
// with an empty root label, so String() of "_alfen._tcp.local" read from a
// packet is "_alfen._tcp.local." — the same form the browser builds from
// "<type>.local.". Equality is OrdinalIgnoreCase on String() (see key).
type dnsName struct {
	labels []string
}

// parseName ports the Name(string) constructor: name.Split('.').
func parseName(s string) dnsName {
	return dnsName{labels: strings.Split(s, ".")}
}

// String ports Name.ToString: labels joined with ".".
func (n dnsName) String() string { return strings.Join(n.labels, ".") }

// key is the dictionary key used wherever Tmds.MDns keys a Dictionary/compares
// by Name, whose Equals/GetHashCode are StringComparer.OrdinalIgnoreCase over
// ToString(). Upper-casing approximates OrdinalIgnoreCase (exact for ASCII).
func (n dnsName) key() string { return strings.ToUpper(n.String()) }

// subName ports Name.SubName(startIndex).
func (n dnsName) subName(start int) dnsName {
	if start >= len(n.labels) {
		return dnsName{}
	}
	return dnsName{labels: append([]string(nil), n.labels[start:]...)}
}

// subNameLen ports Name.SubName(startIndex, length). The C# throws
// ArgumentOutOfRangeException when the name is shorter than requested (which
// a crafted packet can trigger); here the range is clamped instead.
func (n dnsName) subNameLen(start, length int) dnsName {
	if start >= len(n.labels) {
		return dnsName{}
	}
	end := start + length
	if end > len(n.labels) {
		end = len(n.labels)
	}
	return dnsName{labels: append([]string(nil), n.labels[start:end]...)}
}

// firstLabel returns Labels[0] ("" for an empty name).
func (n dnsName) firstLabel() string {
	if len(n.labels) == 0 {
		return ""
	}
	return n.labels[0]
}

// header ports Tmds.MDns.Header.
type header struct {
	TransactionID   uint16
	Flags           uint16
	QuestionCount   uint16
	AnswerCount     uint16
	AuthorityCount  uint16
	AdditionalCount uint16
}

func (h header) isResponse() bool { return h.Flags&0x8000 != 0 }
func (h header) isQuery() bool    { return !h.isResponse() }
func (h header) isNoError() bool  { return h.Flags&0xF == 0 }

// question ports Tmds.MDns.Question.
type question struct {
	QName  dnsName
	QType  recordType
	QClass recordClass
}

// recordHeader ports Tmds.MDns.RecordHeader.
type recordHeader struct {
	Name       dnsName
	Type       recordType
	Class      recordClass
	TTL        uint32
	DataLength uint16
}

// srvRecord ports Tmds.MDns.SrvRecord.
type srvRecord struct {
	Priority uint16
	Weight   uint16
	Port     uint16
	Target   dnsName
}

// msgReader ports Tmds.MDns.DnsMessageReader over a byte slice instead of a
// MemoryStream. Reading past the end is an error (the C# throws, and the
// caller discards the whole packet). Seeks past the end are allowed, as with
// MemoryStream; the next read then fails.
type msgReader struct {
	b            []byte
	pos          int
	recordLength int
	nextRecord   int
}

func newMsgReader(b []byte) *msgReader {
	return &msgReader{b: b, recordLength: -1}
}

func (r *msgReader) readByte() (byte, error) {
	if r.pos < 0 || r.pos >= len(r.b) {
		return 0, errMalformed
	}
	v := r.b[r.pos]
	r.pos++
	return v, nil
}

func (r *msgReader) readBytes(n int) ([]byte, error) {
	if n < 0 || r.pos < 0 || r.pos > len(r.b) || len(r.b)-r.pos < n {
		return nil, errMalformed
	}
	v := r.b[r.pos : r.pos+n]
	r.pos += n
	return v, nil
}

func (r *msgReader) readUint16() (uint16, error) {
	b, err := r.readBytes(2)
	if err != nil {
		return 0, err
	}
	return uint16(b[0])<<8 | uint16(b[1]), nil
}

func (r *msgReader) readUint32() (uint32, error) {
	b, err := r.readBytes(4)
	if err != nil {
		return 0, err
	}
	return uint32(b[0])<<24 | uint32(b[1])<<16 | uint32(b[2])<<8 | uint32(b[3]), nil
}

// readHeader ports DnsMessageReader.ReadHeader.
func (r *msgReader) readHeader() (header, error) {
	var h header
	var err error
	fields := []*uint16{&h.TransactionID, &h.Flags, &h.QuestionCount, &h.AnswerCount, &h.AuthorityCount, &h.AdditionalCount}
	for _, f := range fields {
		if *f, err = r.readUint16(); err != nil {
			return header{}, err
		}
	}
	return h, nil
}

// readQuestion ports DnsMessageReader.ReadQuestion.
func (r *msgReader) readQuestion() (question, error) {
	name, err := r.readName()
	if err != nil {
		return question{}, err
	}
	qt, err := r.readUint16()
	if err != nil {
		return question{}, err
	}
	qc, err := r.readUint16()
	if err != nil {
		return question{}, err
	}
	return question{QName: name, QType: recordType(qt), QClass: recordClass(qc)}, nil
}

// readRecordHeader ports DnsMessageReader.ReadRecordHeader: it first skips
// whatever is left of the previous record's RDATA, then remembers where the
// next record starts (position + DataLength).
func (r *msgReader) readRecordHeader() (recordHeader, error) {
	if r.nextRecord > 0 {
		r.pos = r.nextRecord // SkipRecordBytes
	}
	name, err := r.readName()
	if err != nil {
		return recordHeader{}, err
	}
	t, err := r.readUint16()
	if err != nil {
		return recordHeader{}, err
	}
	c, err := r.readUint16()
	if err != nil {
		return recordHeader{}, err
	}
	ttl, err := r.readUint32()
	if err != nil {
		return recordHeader{}, err
	}
	dl, err := r.readUint16()
	if err != nil {
		return recordHeader{}, err
	}
	r.recordLength = int(dl)
	r.nextRecord = r.pos + int(dl)
	return recordHeader{Name: name, Type: recordType(t), Class: recordClass(c), TTL: ttl, DataLength: dl}, nil
}

// readARecord ports DnsMessageReader.ReadARecord: new IPAddress(ReadBytes(len)).
// The address family follows the RDATA length (4 => IPv4, 16 => IPv6), not the
// record type; any other length throws in .NET (ArgumentException) and is an
// error here.
func (r *msgReader) readARecord() (netip.Addr, error) {
	b, err := r.readBytes(r.recordLength)
	if err != nil {
		return netip.Addr{}, err
	}
	switch len(b) {
	case 4:
		return netip.AddrFrom4([4]byte(b)), nil
	case 16:
		return netip.AddrFrom16([16]byte(b)), nil
	}
	return netip.Addr{}, errMalformed
}

// readTxtRecord ports DnsMessageReader.ReadTxtRecord: length-prefixed UTF-8
// strings until RDATA is consumed; a string overrunning RDATA is an error.
// Zero-length strings are kept. The result is never nil.
func (r *msgReader) readTxtRecord() ([]string, error) {
	list := []string{}
	num := r.recordLength
	for num > 0 {
		n, err := r.readByte()
		if err != nil {
			return nil, err
		}
		num -= int(n) + 1
		if num < 0 {
			return nil, errMalformed
		}
		b, err := r.readBytes(int(n))
		if err != nil {
			return nil, err
		}
		list = append(list, decodeUTF8(b))
	}
	return list, nil
}

// readSrvRecord ports DnsMessageReader.ReadSrvRecord.
func (r *msgReader) readSrvRecord() (srvRecord, error) {
	var s srvRecord
	var err error
	if s.Priority, err = r.readUint16(); err != nil {
		return srvRecord{}, err
	}
	if s.Weight, err = r.readUint16(); err != nil {
		return srvRecord{}, err
	}
	if s.Port, err = r.readUint16(); err != nil {
		return srvRecord{}, err
	}
	if s.Target, err = r.readName(); err != nil {
		return srvRecord{}, err
	}
	return s, nil
}

// readPtrRecord ports DnsMessageReader.ReadPtrRecord.
func (r *msgReader) readPtrRecord() (dnsName, error) { return r.readName() }

// readName ports DnsMessageReader.ReadName. Every length byte, including the
// terminating 0, adds a label (so names end with an empty label). A byte with
// both top bits set is a 14-bit compression pointer; any other value is a
// label length (0x40..0xBF are read as plain lengths, as in the C#). The C#
// follows pointers recursively with no loop check (a pointer loop overflows the
// stack); here a pointer to an offset already visited is rejected instead.
func (r *msgReader) readName() (dnsName, error) {
	var n dnsName
	var visited []int
	resume := -1 // position to restore after the first pointer jump
	for {
		b, err := r.readByte()
		if err != nil {
			return dnsName{}, err
		}
		if b&0xC0 == 0xC0 {
			lo, err := r.readByte()
			if err != nil {
				return dnsName{}, err
			}
			off := int(b&0x3F)<<8 | int(lo)
			for _, v := range visited {
				if v == off {
					return dnsName{}, errMalformed
				}
			}
			visited = append(visited, off)
			if resume < 0 {
				resume = r.pos
			}
			r.pos = off
			continue
		}
		label, err := r.readBytes(int(b))
		if err != nil {
			return dnsName{}, err
		}
		n.labels = append(n.labels, decodeUTF8(label))
		if b == 0 {
			break
		}
	}
	if resume >= 0 {
		r.pos = resume
	}
	return n, nil
}

// decodeUTF8 mirrors Encoding.UTF8.GetString, which substitutes U+FFFD for
// invalid byte sequences instead of failing.
func decodeUTF8(b []byte) string {
	return strings.ToValidUTF8(string(b), "�")
}

// buildQuery ports the DnsMessageWriter calls in
// NetworkInterfaceHandler.OnQueryTimerElapsed: WriteQueryHeader(id) (flags 0,
// i.e. a standard QM query), one WriteQuestion per entry with class IN, then
// Finish() which patches the section counts. Names are written uncompressed;
// a terminating 0 is appended only when the last label is not already empty.
//
// Deviation: Name labels are length-prefixed with their UTF-8 byte length. The
// C# writes label.Length (UTF-16 code units), which corrupts non-ASCII labels;
// for the ASCII names the browser actually queries the output is identical.
func buildQuery(id uint16, qs []question) []byte {
	b := make([]byte, 12, 512)
	b[0], b[1] = byte(id>>8), byte(id)
	for _, q := range qs {
		b = appendName(b, q.QName)
		b = append(b, byte(uint16(q.QType)>>8), byte(q.QType), byte(uint16(q.QClass)>>8), byte(q.QClass))
	}
	n := len(qs)
	b[4], b[5] = byte(n>>8), byte(n)
	return b
}

// appendName ports DnsMessageWriter.WriteName.
func appendName(b []byte, n dnsName) []byte {
	lastEmpty := false
	for _, l := range n.labels {
		lastEmpty = len(l) == 0
		b = append(b, byte(len(l)))
		b = append(b, l...)
	}
	if !lastEmpty {
		b = append(b, 0)
	}
	return b
}
