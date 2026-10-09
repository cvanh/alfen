package keytest

import (
	"bufio"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"strings"
)

// KeyDeriver turns a passphrase/identifier line into a 16-byte AES-128 key.
type KeyDeriver func(line string) ([]byte, bool)

// DeriveRawHex interprets the line as hex for exactly 16 bytes.
func DeriveRawHex(line string) ([]byte, bool) {
	line = strings.TrimSpace(line)
	b, err := hex.DecodeString(line)
	if err != nil || len(b) != 16 {
		return nil, false
	}
	return b, true
}

// DeriveRawBytes uses the raw UTF-8 bytes, truncated/zero-padded to 16.
func DeriveRawBytes(line string) ([]byte, bool) {
	line = strings.TrimRight(line, "\r\n")
	k := make([]byte, 16)
	copy(k, []byte(line))
	return k, true
}

// DeriveMD5 uses MD5(line) (16 bytes) — a common "password -> AES-128" shortcut.
func DeriveMD5(line string) ([]byte, bool) {
	line = strings.TrimRight(line, "\r\n")
	sum := md5.Sum([]byte(line))
	return sum[:], true
}

// DeriveSHA256_16 uses the first 16 bytes of SHA-256(line).
func DeriveSHA256_16(line string) ([]byte, bool) {
	line = strings.TrimRight(line, "\r\n")
	sum := sha256.Sum256([]byte(line))
	return sum[:16], true
}

// Derivers maps CLI names to derivation functions.
var Derivers = map[string]KeyDeriver{
	"hex":    DeriveRawHex,
	"raw":    DeriveRawBytes,
	"md5":    DeriveMD5,
	"sha256": DeriveSHA256_16,
}

// StreamKeys reads lines from r, applies derive, and sends 16-byte keys on the
// returned channel. Blank lines and lines starting with '#' are skipped. The
// channel closes when r is exhausted. Runs in its own goroutine.
func StreamKeys(r io.Reader, derive KeyDeriver) <-chan []byte {
	out := make(chan []byte, 4096)
	go func() {
		defer close(out)
		sc := bufio.NewScanner(r)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for sc.Scan() {
			line := sc.Text()
			t := strings.TrimSpace(line)
			if t == "" || strings.HasPrefix(t, "#") {
				continue
			}
			if k, ok := derive(line); ok {
				out <- k
			}
		}
	}()
	return out
}

// KeysFromSlice streams an explicit set of keys (e.g. the known static keys).
func KeysFromSlice(keys [][]byte) <-chan []byte {
	out := make(chan []byte, len(keys)+1)
	for _, k := range keys {
		out <- k
	}
	close(out)
	return out
}
