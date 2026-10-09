# aceclient — Alfen ACE Service tool, reimplemented in Go

A clean-room Go reimplementation of the charger-facing parts of the **Alfen ACE
Service Installer**, for firmware-format research and interoperability on
hardware you own. Every structure and constant here is a direct port of the
decompiled installer (`docs/decompiled/*.cs`, via `ilspycmd`) and is
cross-checked against the real firmware files in `../firmware/Firmware` and the analysis
notes in `../docs`.

It exists so you can drive the charger and experiment with the firmware
containers from your own fast, scriptable client instead of the Windows GUI —
including a parallel **candidate-key sweep** over `.fwi` payloads.

## Honest scope of the "key bruteforce"

Read this before expecting magic:

- Controller/AHP firmware payloads are **AES-128**. The full key space is
  2¹²⁸. **Exhaustive brute force is physically impossible** and no client makes
  it possible. The analysis (`../docs/04`) found **no firmware key in the
  installer** — the key is held on the device.
- This tool only helps when the key space is **constrained**: a wordlist, keys
  **derived** from device identifiers (serial, MAC, …), or a specific
  key-derivation hypothesis you want to confirm or reject quickly. On ~7 cores
  it sweeps **~850k keys/s** (prefix-only CBC + entropy oracle). That is hopeless
  against 2¹²⁸ but clears, say, a 2³² derived-seed space in ~1.4 h.
- It will **instantly** recover the known display-resource key on a
  display image (that is the `selftest`), and will **correctly fail** to find a
  key for controller firmware from any small/derived set — which is the expected,
  truthful outcome, not a bug.

What this is *not*: it does nothing to the firmware **signature / secure-boot**
chain, and uploads firmware byte-for-byte exactly as the official tool does.

## Build & verify

```
go build ./cmd/aceclient        # or: go build -o aceclient ./cmd/aceclient
./aceclient selftest            # round-trips every crypto path end-to-end
```

`selftest` builds a display `.fwi`, re-parses it (header CRC + layout), unwraps
it back to the original bytes, recovers the display key with the sweep, and does
an SCN encrypt→decrypt→parse round-trip. All in-process, no device needed.

## Commands

### Firmware containers
```
aceclient fwi-info  <file.fwi>        # parse header, verify CRC, classify, try display unwrap
aceclient fwi-build <in.bin> <out.fwi># wrap data as a display-resource .fwi (known key)
aceclient tvf-info  <file.tfw>        # inspect an AHP .tfw (manifest / cert / payload region)
```
`fwi-info` parses the 160-byte header generically (it does **not** assume the
display builder's constants, because real controller firmware shares type `0x13`
yet uses different header words). Validated against 5.6.1 / 6.6.2 / 7.1.6 /
7.4.6 — all header CRCs pass.

### Charger network
```
aceclient discover [seconds]          # listen for SCN UDP broadcasts on :36549
aceclient scn-decode <packet.hex|->   # decrypt+parse one raw SCN datagram
aceclient login  --ip H --port 443 --user U --pass P [--display D] [--insecure=false]
aceclient prop-get --ip H ... [--ids 8311_0 | --params "cat=generic&limit=500"]
aceclient prop-set --ip H ... --id 8317_2 --type <sdt> --value <v> --name <n>
aceclient firmware-upload --ip H ... --file <fw.fwi|.tfw>
```
- **SCN** (`internal/scn`) is a 1:1 port of `SCNSocket.ParseData` /
  `SCNNetwork`: AES-128-CBC, zero IV, `PaddingMode.None`, static fleet key
  `DE0B4DE87158F4EF8A8B6C2460827470`, port 36549. Listen-only telemetry.
- **Settings** (`internal/api`) is the HTTPS object-dictionary channel: token
  login, `GET/POST /api/prop` (CANopen `IDX_SUB` ids, 15-property write batches,
  per-type value encoding), firmware upload as `multipart/form-data`
  `firmwarefile` in 4096-byte parts, uploaded verbatim.
- `--insecure` defaults **true** (chargers present self-signed device certs; the
  official tool pins them via its own store). Pass `--insecure=false` to enforce
  verification.

### Key research
```
aceclient keytest --file <fwi> --keys <wordlist|-> [--derive hex|raw|md5|sha256]
   [--oracle entropy|display|magic] [--threshold 7.5] [--skip-first]
   [--magic 789c --magic-offset -1] [--include-known] [--workers N] [--all]
```
- `--keys -` reads candidates from stdin (one per line; `#` comments ok).
- `--derive` turns each line into a 16-byte key: `hex` (32 hex chars), `raw`
  (UTF-8 bytes, zero-padded/truncated), `md5`, or `sha256` (first 16 bytes).
- Oracles:
  - `entropy` — flags low Shannon entropy (structured plaintext). With
    `--skip-first` it examines bytes past the first block and is therefore
    **IV-independent** (`P_i = Dec_K(C_i) ⊕ C_{i-1}` for i≥1), so it works even
    though the firmware's per-image IV is unknown. This is the default for
    unknown firmware.
  - `display` — exact `GetDataInBin` wrapper check (zeros at +8/+12 and a valid
    inner CRC32). Proves a key for display-resource images; needs IV=0.
  - `magic` — looks for `--magic` bytes at `--magic-offset` (`-1` = anywhere).
- `--include-known` seeds the sweep with the installer's static keys (display +
  SCN) as a sanity check — they will not match controller firmware.
- Oracles declare how many leading bytes they need, so the sweep CBC-decrypts
  only that prefix per key (the throughput lever).

## Layout — maps 1:1 to the ACE assemblies

| Package | Ports | From |
|---------|-------|------|
| `internal/alfencrc` | `Crc32` (reflected IEEE, `ComputeChecksum(buf,start)`) | `ACEFWUCreator.cs` |
| `internal/fwi` | `.fwi` header parse + `ICUFWUCreator` build pipeline (`AESEncrypt`, `GetDataInBin`, `DeflateData`, `CreateFWIHeader`, `WriteFWUFile`) | `ICUFWUCreator.cs` |
| `internal/tvf` | `TvfHeader` builder + real-`.tfw` sniffer | `ACEFWUCreator.cs` |
| `internal/scn` | `SCNNetwork` / `SCNSocket` UDP discovery | `SCNNetwork.cs`, `SCNSocket.cs` |
| `internal/api` | `ICUDevice`/`ICULanDevice` HTTPS: login, tokens, `prop` read/write, firmware upload | `ACENetwork.cs` |
| `internal/keytest` | candidate-key sweep + oracles (not in the installer) | — |

## Keys & constants (all from the decompiled installer)

| Use | Key / value |
|-----|-------------|
| Display-resource `.fwi` AES-128-CBC, IV=0, PKCS7 | `29C66A20AE16C4BA046A21D57A78664F` |
| SCN UDP AES-128-CBC, IV=0, no padding | `DE0B4DE87158F4EF8A8B6C2460827470` |
| `.fwi` header CRC32 | reflected IEEE poly `0xEDB88320`, over **header bytes [0x04:0xA0] only** |
| TVF magic (`0x04`) | `0xB8ED7AD5` |
| SCN UDP port | 36549 |

> Note: the `.fwi` header CRC covers the **160-byte header only**, not
> `[0x04:EOF]` as `../docs/01` originally stated. Verified empirically here
> against 5.6.1 / 6.6.2 / 7.4.6 (all pass); the notes have been corrected.

## Boundary

Interop and format/key analysis on owned hardware and public firmware files.
No defeat of firmware signing / secure boot; firmware is uploaded unmodified and
verified/decrypted on-device.
