package main

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"alfen/aceclient/internal/fwi"
	"alfen/aceclient/internal/keytest"
	"alfen/aceclient/internal/scn"
)

func cmdKeytest(args []string) error {
	_, vals, _, _ := parseConn(args, nil)
	file := vals["file"]
	if file == "" {
		return fmt.Errorf("--file <fwi> required")
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return err
	}

	// payload slice
	payloadOffset := fwi.HeaderLen
	if v := vals["payload-offset"]; v != "" {
		payloadOffset, _ = strconv.Atoi(v)
	}
	if payloadOffset > len(data) {
		return fmt.Errorf("payload-offset %d past EOF %d", payloadOffset, len(data))
	}
	ct := data[payloadOffset:]
	if len(ct)%aes.BlockSize != 0 {
		ct = ct[:len(ct)-len(ct)%aes.BlockSize] // trim to block boundary
	}

	// IV
	iv := make([]byte, aes.BlockSize)
	if v := vals["iv"]; v != "" {
		b, err := hex.DecodeString(v)
		if err != nil || len(b) != aes.BlockSize {
			return fmt.Errorf("--iv must be 32 hex chars")
		}
		iv = b
	}

	// oracle
	oracleName := vals["oracle"]
	if oracleName == "" {
		oracleName = "entropy"
	}
	var oracle keytest.Oracle
	switch oracleName {
	case "entropy":
		th := 7.5
		if v := vals["threshold"]; v != "" {
			th, _ = strconv.ParseFloat(v, 64)
		}
		oracle = keytest.EntropyOracle{Threshold: th, SkipFirstBlock: vals["skip-first"] == "true", Sample: 4096}
	case "display":
		oracle = keytest.DisplayWrapperOracle{}
	case "magic":
		mg, err := hex.DecodeString(vals["magic"])
		if err != nil || len(mg) == 0 {
			return fmt.Errorf("--magic <hex> required for magic oracle")
		}
		off := -1
		if v := vals["magic-offset"]; v != "" {
			off, _ = strconv.Atoi(v)
		}
		oracle = keytest.MagicOracle{Magic: mg, Offset: off}
	default:
		return fmt.Errorf("unknown --oracle %q", oracleName)
	}

	// key source
	deriveName := vals["derive"]
	if deriveName == "" {
		deriveName = "hex"
	}
	derive, ok := keytest.Derivers[deriveName]
	if !ok {
		return fmt.Errorf("unknown --derive %q (hex|raw|md5|sha256)", deriveName)
	}

	var keyReader *os.File = os.Stdin
	if kf := vals["keys"]; kf != "" && kf != "-" {
		keyReader, err = os.Open(kf)
		if err != nil {
			return err
		}
		defer keyReader.Close()
	}

	// optional: prepend the known static keys to the stream
	known := [][]byte{}
	if vals["include-known"] == "true" {
		known = append(known, fwi.DisplayKey, scn.AESKey)
	}

	workers := 0
	if v := vals["workers"]; v != "" {
		workers, _ = strconv.Atoi(v)
	}
	stopOnFirst := vals["all"] != "true"

	fmt.Printf("sweeping %d payload bytes (offset 0x%X), oracle=%s, derive=%s, iv=%s\n",
		len(ct), payloadOffset, oracleName, deriveName, hex.EncodeToString(iv))

	keyCh := mergeKeys(known, keytest.StreamKeys(keyReader, derive))
	hits, tried := keytest.Sweep(ct, iv, keyCh, oracle, workers, stopOnFirst)

	fmt.Printf("tried %d keys; %d hit(s)\n", tried, len(hits))
	for _, h := range hits {
		fmt.Printf("  HIT key=%s score=%.3f firstplain=%s\n",
			hex.EncodeToString(h.Key), h.Score, hex.EncodeToString(h.Plain[:min(16, len(h.Plain))]))
	}
	if len(hits) == 0 {
		fmt.Println("no key in the supplied set decrypts this payload under this oracle.")
		fmt.Println("(reminder: AES-128 cannot be brute-forced exhaustively; narrow the key space.)")
	}
	return nil
}

// mergeKeys streams `first` keys then everything from ch.
func mergeKeys(first [][]byte, ch <-chan []byte) <-chan []byte {
	out := make(chan []byte, 1024)
	go func() {
		defer close(out)
		for _, k := range first {
			out <- k
		}
		for k := range ch {
			out <- k
		}
	}()
	return out
}

func cmdSelftest(_ []string) error {
	fmt.Println("== fwi display pipeline round-trip ==")
	orig := bytes.Repeat([]byte("ACECLIENT selftest payload \x00\x01\x02\x03"), 40)
	img, err := fwi.WrapDisplayPayload(orig)
	if err != nil {
		return err
	}
	fmt.Printf("built display .fwi: %d bytes\n", len(img))

	h, err := fwi.Parse(img)
	if err != nil {
		return err
	}
	ok, stored, computed := fwi.VerifyCRC(img)
	fmt.Printf("header CRC ok=%v (stored=%08X computed=%08X)\n", ok, stored, computed)
	fmt.Printf("looksLikeDisplayResource=%v  %s\n", h.LooksLikeDisplayResource(), h)
	if !ok || !h.LooksLikeDisplayResource() {
		return fmt.Errorf("selftest: header checks failed")
	}

	recovered, err := fwi.UnwrapDisplayPayload(img)
	if err != nil {
		return fmt.Errorf("unwrap: %w", err)
	}
	if !bytes.Equal(recovered, orig) {
		return fmt.Errorf("selftest: round-trip mismatch (%d vs %d bytes)", len(recovered), len(orig))
	}
	fmt.Printf("unwrap round-trip OK (%d bytes identical)\n", len(recovered))

	fmt.Println("\n== keytest finds the known display key ==")
	ct := h.Payload(img)
	keyCh := keytest.KeysFromSlice([][]byte{
		scn.AESKey, // decoy (wrong key)
		fwi.DisplayKey,
	})
	hits, tried := keytest.Sweep(ct, make([]byte, 16), keyCh, keytest.DisplayWrapperOracle{}, 2, false)
	fmt.Printf("tried %d keys, %d hit(s)\n", tried, len(hits))
	foundDisplay := false
	for _, hit := range hits {
		if bytes.Equal(hit.Key, fwi.DisplayKey) {
			foundDisplay = true
		}
	}
	if !foundDisplay {
		return fmt.Errorf("selftest: display key not found by sweep")
	}
	fmt.Printf("sweep recovered the display key %s via display-wrapper oracle\n", hex.EncodeToString(fwi.DisplayKey))

	fmt.Println("\n== SCN encrypt/decrypt + parse round-trip ==")
	pkt := buildSCNPacket("TESTNET", "Charger-1", 1, 16.0)
	enc, err := scnEncrypt(pkt)
	if err != nil {
		return err
	}
	s, err := scn.ProcessPacket(enc, nil, time.Now())
	if err != nil {
		return err
	}
	if s == nil {
		return fmt.Errorf("selftest: SCN packet rejected")
	}
	fmt.Printf("decoded SCN: net=%q name=%q lib=%d setpoint=%.1fA state=%s\n",
		s.NetworkName, s.Name, s.ScnLibVersion, s.SetPointCurrent, s.State)
	if s.NetworkName != "TESTNET" || strings.TrimSpace(s.Name) != "Charger-1" {
		return fmt.Errorf("selftest: SCN field mismatch")
	}
	fmt.Println("\nALL SELFTESTS PASSED")
	return nil
}

// scnEncrypt is the inverse of scn.Decrypt (AES-128-CBC, zero IV, no padding),
// used only to generate selftest fixtures.
func scnEncrypt(pt []byte) ([]byte, error) {
	if len(pt)%aes.BlockSize != 0 {
		return nil, fmt.Errorf("plaintext not block-aligned: %d", len(pt))
	}
	block, err := aes.NewCipher(scn.AESKey)
	if err != nil {
		return nil, err
	}
	iv := make([]byte, aes.BlockSize)
	out := make([]byte, len(pt))
	cipher.NewCBCEncrypter(block, iv).CryptBlocks(out, pt)
	return out, nil
}

// buildSCNPacket crafts a minimal 96-byte v1 SCN packet matching the field
// offsets read by scn.ParseData.
func buildSCNPacket(network, name string, libVersion byte, setpoint float32) []byte {
	// 128 bytes (16-aligned for CBC); enough to hold all v1 fields through
	// OptionByte (offset 123) and SetPointCurrent (offset 104).
	b := make([]byte, 128)
	b[0] = libVersion
	// 1..3 reserved; 4..7 timestamp
	copy(b[8:16], []byte(network)) // NetworkName char[8]
	// 0x10 u16 Id; 0x12 reserved; 0x13 Mode3State; 0x14 State; 0x15 PhaseMask
	// 0x16 TotalNumberOfSockets; 0x17 SocketIndex
	b[0x17] = 0
	copy(b[0x18:0x18+21], []byte(name)) // Name char[21]
	// 0x68 SetPointCurrent f32
	putF32(b, 0x68, setpoint)
	return b
}

func putF32(b []byte, off int, v float32) {
	bits := math.Float32bits(v)
	b[off] = byte(bits)
	b[off+1] = byte(bits >> 8)
	b[off+2] = byte(bits >> 16)
	b[off+3] = byte(bits >> 24)
}
