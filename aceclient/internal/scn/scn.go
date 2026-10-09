// Package scn ports the Smart Charging Network UDP discovery protocol from
// ICUNetwork.SCNNetwork / ICUNetwork.SCNSocket
// (docs/decompiled/SCNNetwork.cs, docs/decompiled/SCNSocket.cs).
//
// Transport: UDP broadcast on port 36549, listen-only. Each datagram is
// AES-128-CBC (zero IV, PaddingMode.None) under a hardcoded, fleet-wide key, so
// anyone on the segment can decode it; it is telemetry, not a control channel.
//
// The packet struct and its field offsets are a direct transcription of
// SCNSocket.ParseData's sequential BinaryReader calls — the byte offsets are
// therefore ground truth, not inferred.
package scn

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"net"
	"strings"
	"time"
)

// UDPPort is SCNNetwork.UDPPORT.
const UDPPort = 36549

const (
	udpMaxBufferLength = 1024 // SCNNetwork.udpMaxBufferLength
	udpMinPacketLength = 96   // SCNNetwork.udpMinPacketLength
)

// AESKey is SCNNetwork.s_aesKey (DE0B4DE87158F4EF8A8B6C2460827470).
var AESKey = []byte{222, 11, 77, 232, 113, 88, 244, 239, 138, 139, 108, 36, 96, 130, 116, 112}

// ValidLibraryVersions mirrors ScnLibraryVersions = {1..6}. ParseData drops any
// packet whose byte[0] is not in this set.
var ValidLibraryVersions = map[int]bool{1: true, 2: true, 3: true, 4: true, 5: true, 6: true}

// ChargingState mirrors EChargingState.
type ChargingState byte

const (
	StateEmpty ChargingState = iota
	StateIdle
	StateChargingInitializing
	StateChargingProbing
	StateChargingIncreaseCurrent
	StateCharging
	StateAlternating
	StateUnconnected
)

func (s ChargingState) String() string {
	names := []string{"Empty", "Idle", "ChargingInitializing", "ChargingProbing",
		"ChargingIncreaseCurrent", "Charging", "Alternating", "Unconnected"}
	if int(s) < len(names) {
		return names[s]
	}
	return fmt.Sprintf("State(%d)", byte(s))
}

// Socket mirrors ICUNetwork.SCNSocket's parsed fields.
type Socket struct {
	ScnLibVersion            int
	Name                     string
	Id                       uint32
	Mode3State               byte
	SocketIndex              byte
	SocketCount              byte
	State                    ChargingState
	LastUpdate               time.Time
	NetworkName              string
	Timestamp                uint32
	TotalNumberOfSockets     int
	PhaseMask                uint32
	ActiveCurrentL1          float64
	ActiveCurrentL2          float64
	ActiveCurrentL3          float64
	UniqueID                 uint64
	MaximumGroupID           int
	Clock                    uint64
	WaitingSince             uint32
	AlterningSince           int32
	MinimumCurrent           float64
	MaximumCurrent           float64
	AvailableCurrentL1       float64
	AvailableCurrentL2       float64
	AvailableCurrentL3       float64
	SetPointCurrent          float64
	ActiveChargingTime       uint32
	AlternatingCountDown     uint32
	IPAddress                net.IP
	PropSocketSafeCurrent    float64
	PropMaximumStaticCurrent float64
	PropAlternatingPeriod    int
	PropChangedCounter       int
	OptionByte               byte
	ExtraCurrentL1           float64
	ExtraCurrentL2           float64
	ExtraCurrentL3           float64
	MaximumGroupCurrent      float64
	PropTotalSafeCurrent     float64
}

// reader is a little-endian sequential reader mirroring C# BinaryReader on a
// MemoryStream; ParseData only reads forward and never past the end.
type reader struct {
	b   []byte
	pos int
}

func (r *reader) u8() byte           { v := r.b[r.pos]; r.pos++; return v }
func (r *reader) u16() uint16        { v := binary.LittleEndian.Uint16(r.b[r.pos:]); r.pos += 2; return v }
func (r *reader) u32() uint32        { v := binary.LittleEndian.Uint32(r.b[r.pos:]); r.pos += 4; return v }
func (r *reader) i32() int32         { return int32(r.u32()) }
func (r *reader) u64() uint64        { v := binary.LittleEndian.Uint64(r.b[r.pos:]); r.pos += 8; return v }
func (r *reader) i16() int16         { return int16(r.u16()) }
func (r *reader) f32() float64       { return float64(math.Float32frombits(r.u32())) }
func (r *reader) bytes(n int) []byte { v := r.b[r.pos : r.pos+n]; r.pos += n; return v }

// ParseData ports SCNSocket.ParseData. It returns (nil, nil) for packets the
// installer silently drops (unknown lib version, too short, empty network name).
func ParseData(data []byte, received time.Time, ip net.IP) (*Socket, error) {
	if len(data) == 0 {
		return nil, nil
	}
	num := int(data[0])
	if !ValidLibraryVersions[num] || len(data) < 96 {
		return nil, nil
	}
	s := &Socket{
		IPAddress:     ip,
		LastUpdate:    time.Now().UTC(),
		ScnLibVersion: num,
		State:         StateIdle,
	}
	r := &reader{b: data}
	r.u8() // version byte (already captured)
	r.u8()
	r.u8()
	r.u8()
	timestamp := r.u32()
	nameBytes := r.bytes(8)
	s.NetworkName = trimCName(nameBytes)
	if len(s.NetworkName) == 0 {
		return nil, nil
	}
	s.Id = uint32(r.u16())
	r.u8()
	s.Mode3State = r.u8()
	s.State = ChargingState(r.u8())
	s.PhaseMask = uint32(r.u8())
	s.TotalNumberOfSockets = int(r.u8())
	s.SocketIndex = r.u8()
	s.Timestamp = timestamp
	s.Name = trimCName(r.bytes(21))
	s.SocketCount = r.u8()
	s.MaximumGroupID = int(r.u16())
	s.UniqueID = r.u64()
	s.Clock = r.u64()
	s.WaitingSince = r.u32()
	s.AlterningSince = r.i32()
	s.MinimumCurrent = r.f32()
	s.MaximumCurrent = r.f32()
	s.ActiveCurrentL1 = r.f32()
	s.ActiveCurrentL2 = r.f32()
	s.ActiveCurrentL3 = r.f32()
	s.AvailableCurrentL1 = r.f32()
	s.AvailableCurrentL2 = r.f32()
	s.AvailableCurrentL3 = r.f32()
	s.SetPointCurrent = r.f32()
	s.ActiveChargingTime = r.u32()
	s.AlternatingCountDown = r.u32()
	if num >= 3 {
		s.PropSocketSafeCurrent = r.f32()
		s.PropMaximumStaticCurrent = r.f32()
	} else {
		s.PropSocketSafeCurrent = float64(r.u16()) / 10.0
		s.PropMaximumStaticCurrent = float64(r.u16()) / 10.0
	}
	s.PropAlternatingPeriod = int(r.u16())
	s.PropChangedCounter = int(r.u8())
	s.OptionByte = r.u8()
	if num >= 2 {
		s.ExtraCurrentL1 = float64(r.u16())
		s.ExtraCurrentL2 = float64(r.u16())
		s.ExtraCurrentL3 = float64(r.u16())
		r.i16()
		s.MaximumGroupCurrent = r.f32()
		if num >= 3 {
			s.PropTotalSafeCurrent = r.f32()
		}
	}
	_ = received
	return s, nil
}

// trimCName mirrors Encoding.UTF8.GetString(b).Trim('\0').Trim().
func trimCName(b []byte) string {
	return strings.TrimSpace(strings.Trim(string(b), "\x00"))
}

// Decrypt performs the AES-128-CBC, zero-IV, no-padding decryption from
// SCNNetwork.AESDecrypt. The input must be a multiple of the block size.
func Decrypt(ct []byte) ([]byte, error) {
	if len(ct) == 0 || len(ct)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("ciphertext not block-aligned: %d bytes", len(ct))
	}
	block, err := aes.NewCipher(AESKey)
	if err != nil {
		return nil, err
	}
	iv := make([]byte, aes.BlockSize)
	out := make([]byte, len(ct))
	cipher.NewCBCDecrypter(block, iv).CryptBlocks(out, ct)
	return out, nil
}

// ProcessPacket ports SCNNetwork.ProcessUdpPacket: size-gate, decrypt, validate
// and parse one datagram into a Socket.
//
// Faithful detail: the installer decrypts into a fixed `new byte[1024]` buffer,
// so ParseData always sees a 1024-byte, zero-padded array and never overruns on
// a short real packet. We reproduce that padding here.
func ProcessPacket(buf []byte, from net.IP, received time.Time) (*Socket, error) {
	if len(buf) >= udpMaxBufferLength {
		return nil, nil
	}
	plain, err := Decrypt(buf)
	if err != nil {
		return nil, err
	}
	padded := make([]byte, udpMaxBufferLength) // mirror binDataDecrypted[1024]
	copy(padded, plain)
	if !ValidLibraryVersions[int(padded[0])] {
		return nil, nil
	}
	return ParseData(padded, received, from)
}

// Listener listens for SCN broadcasts on UDPPort and feeds decoded sockets to a
// callback. It mirrors the two-task receive/process split of SCNNetwork but in a
// single goroutine for simplicity.
type Listener struct {
	conn *net.UDPConn
}

// Listen binds IPAddress.Any:36549 with SO_REUSEADDR, like StartUdpReceiveTask.
func Listen() (*Listener, error) {
	addr := &net.UDPAddr{IP: net.IPv4zero, Port: UDPPort}
	conn, err := net.ListenUDP("udp4", addr)
	if err != nil {
		return nil, err
	}
	return &Listener{conn: conn}, nil
}

// Run reads datagrams until the deadline (zero = forever) or an error, invoking
// onSocket for each successfully decoded packet.
func (l *Listener) Run(onSocket func(*Socket, net.IP), until time.Time) error {
	buf := make([]byte, 2048)
	for {
		if !until.IsZero() {
			if err := l.conn.SetReadDeadline(until); err != nil {
				return err
			}
		}
		n, raddr, err := l.conn.ReadFromUDP(buf)
		if err != nil {
			var ne net.Error
			if errors.As(err, &ne) && ne.Timeout() {
				return nil
			}
			return err
		}
		s, perr := ProcessPacket(buf[:n], raddr.IP, time.Now())
		if perr != nil || s == nil {
			continue
		}
		onSocket(s, raddr.IP)
	}
}

// Close releases the socket.
func (l *Listener) Close() error { return l.conn.Close() }
