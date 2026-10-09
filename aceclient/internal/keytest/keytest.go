// Package keytest is a candidate-key sweep for `.fwi` payloads.
//
// IMPORTANT, up front: the payload is AES-128. The full key space is 2^128;
// exhaustive brute force is physically impossible. This tool only helps when
// the key space is CONSTRAINED — a wordlist, keys derived from device
// identifiers, or a specific derivation hypothesis you want to confirm or
// reject fast. It will instantly confirm the known display-resource key on a
// display image (end-to-end self-test) and will (correctly) fail to find any
// key for controller firmware, whose key is device-held.
//
// Key CBC fact exploited here: only the FIRST plaintext block depends on the
// IV. For every block i>=1, P_i = Dec_K(C_i) XOR C_{i-1}, independent of the
// IV. So structure/entropy oracles that look at bytes past offset 16 are valid
// even though the firmware's per-image IV is unknown. Oracles default to
// skipping the first block for exactly this reason.
package keytest

import (
	"crypto/aes"
	"crypto/cipher"
	"math"
	"runtime"
	"sync"
	"sync/atomic"

	"alfen/aceclient/internal/alfencrc"
)

// Oracle scores a decrypted payload and decides whether it is a plausible hit.
type Oracle interface {
	Name() string
	// Eval returns a score (higher = more plausible) and whether it is a hit.
	Eval(plain []byte) (score float64, hit bool)
}

// prefixNeeder lets an oracle declare how many leading plaintext bytes it needs.
// Sweep then CBC-decrypts only that prefix (valid because each CBC block depends
// only on the previous ciphertext block), which is the main speed lever for big
// payloads. A return <= 0 means "the whole payload".
type prefixNeeder interface{ NeedBytes() int }

// EntropyOracle flags plaintext whose Shannon entropy (bits/byte) over the
// examined region is at or below Threshold — i.e. structured, not random.
// SkipFirstBlock makes it IV-independent.
type EntropyOracle struct {
	Threshold      float64 // e.g. 7.5; real AES ciphertext is ~8.0
	SkipFirstBlock bool
	Sample         int // bytes to examine (0 = all)
}

func (o EntropyOracle) Name() string { return "entropy" }

func (o EntropyOracle) NeedBytes() int {
	n := 0
	if o.SkipFirstBlock {
		n += aes.BlockSize
	}
	if o.Sample > 0 {
		return n + o.Sample
	}
	return -1 // whole payload
}

func (o EntropyOracle) Eval(plain []byte) (float64, bool) {
	b := plain
	if o.SkipFirstBlock && len(b) > aes.BlockSize {
		b = b[aes.BlockSize:]
	}
	if o.Sample > 0 && o.Sample < len(b) {
		b = b[:o.Sample]
	}
	if len(b) == 0 {
		return 8, false
	}
	e := shannon(b)
	// score: how far below max entropy (so lower entropy = higher score)
	return 8 - e, e <= o.Threshold
}

// MagicOracle flags plaintext that contains Magic at Offset (-1 = anywhere).
type MagicOracle struct {
	Magic  []byte
	Offset int
}

func (o MagicOracle) Name() string { return "magic" }

func (o MagicOracle) NeedBytes() int {
	if o.Offset >= 0 {
		return o.Offset + len(o.Magic)
	}
	return -1 // search anywhere -> whole payload
}

func (o MagicOracle) Eval(plain []byte) (float64, bool) {
	if o.Offset >= 0 {
		if o.Offset+len(o.Magic) <= len(plain) && equal(plain[o.Offset:o.Offset+len(o.Magic)], o.Magic) {
			return 1, true
		}
		return 0, false
	}
	if indexOf(plain, o.Magic) >= 0 {
		return 1, true
	}
	return 0, false
}

// DisplayWrapperOracle is the exact oracle for display-resource images: after an
// IV=0 CBC decrypt the plaintext is the GetDataInBin wrapper
// [crc32][16+len][0][0][data], so words at offset 8 and 12 are zero and the
// stored CRC32 over [4:total] validates. This both proves a key and verifies
// the whole fwi pipeline. (Requires IV=0, so do not skip the first block.)
type DisplayWrapperOracle struct{}

func (o DisplayWrapperOracle) Name() string { return "display-wrapper" }

func (o DisplayWrapperOracle) Eval(plain []byte) (float64, bool) {
	if len(plain) < 16 {
		return 0, false
	}
	le := func(i int) uint32 {
		return uint32(plain[i]) | uint32(plain[i+1])<<8 | uint32(plain[i+2])<<16 | uint32(plain[i+3])<<24
	}
	if le(8) != 0 || le(12) != 0 {
		return 0, false
	}
	total := le(4)
	if int(total) < 16 || int(total) > len(plain) {
		return 0, false
	}
	if alfencrc.ComputeChecksum(plain[:total], 4) != le(0) {
		return 0, false
	}
	return 1, true
}

// Result is a successful hit.
type Result struct {
	Key   []byte
	Score float64
	Plain []byte // the decrypted payload for the winning key
}

// Sweep tries every key from `keys` against ciphertext `ct` using CBC with `iv`
// (len 16; typically all-zero) and the given oracle, across `workers`
// goroutines (0 = NumCPU). It returns every hit and the number of keys tried.
//
// The scan stops early once a hit is found if stopOnFirst is true.
func Sweep(ct, iv []byte, keys <-chan []byte, oracle Oracle, workers int, stopOnFirst bool) ([]Result, uint64) {
	if workers <= 0 {
		workers = runtime.NumCPU()
	}
	if len(iv) != aes.BlockSize {
		iv = make([]byte, aes.BlockSize)
	}
	// Decrypt only the prefix the oracle needs (rounded up to a block), which
	// is valid for CBC and is the primary throughput win on large payloads.
	decLen := len(ct)
	if pn, ok := oracle.(prefixNeeder); ok {
		if n := pn.NeedBytes(); n > 0 {
			n = ((n + aes.BlockSize - 1) / aes.BlockSize) * aes.BlockSize
			if n < decLen {
				decLen = n
			}
		}
	}
	ct = ct[:decLen]
	var tried uint64
	var mu sync.Mutex
	var hits []Result
	var done atomic.Bool

	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			buf := make([]byte, len(ct))
			localIV := make([]byte, aes.BlockSize)
			for key := range keys {
				if done.Load() {
					return
				}
				atomic.AddUint64(&tried, 1)
				block, err := aes.NewCipher(key)
				if err != nil {
					continue
				}
				copy(localIV, iv)
				cipher.NewCBCDecrypter(block, localIV).CryptBlocks(buf, ct)
				score, hit := oracle.Eval(buf)
				if hit {
					plain := make([]byte, len(buf))
					copy(plain, buf)
					kcopy := make([]byte, len(key))
					copy(kcopy, key)
					mu.Lock()
					hits = append(hits, Result{Key: kcopy, Score: score, Plain: plain})
					mu.Unlock()
					if stopOnFirst {
						done.Store(true)
						return
					}
				}
			}
		}()
	}
	wg.Wait()
	return hits, atomic.LoadUint64(&tried)
}

// shannon returns the Shannon entropy of b in bits per byte (0..8).
func shannon(b []byte) float64 {
	var counts [256]int
	for _, c := range b {
		counts[c]++
	}
	n := float64(len(b))
	e := 0.0
	for _, c := range counts {
		if c == 0 {
			continue
		}
		p := float64(c) / n
		e -= p * math.Log2(p)
	}
	return e
}

func equal(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func indexOf(h, n []byte) int {
	if len(n) == 0 || len(n) > len(h) {
		return -1
	}
	for i := 0; i+len(n) <= len(h); i++ {
		if equal(h[i:i+len(n)], n) {
			return i
		}
	}
	return -1
}
