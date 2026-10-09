# 01 — Firmware Container Formats

Two container families, two subsystems:

- **`.fwi`** — NG9xx **controller** firmware (versioned 4.x–7.x).
- **`.tfw`** — AHP **application** releases, internally the **TVF** format
  (versioned 2.x).

Both are **signed + encrypted**. Format details below are confirmed by
reverse-engineering the files *and* by decompiling the installer's
`ACEFWUCreator.dll` (see `03`), whose `CreateFWIHeader` / `WriteFWUFile`
reproduce the `.fwi` layout exactly.

---

## 1. `.fwi` container (NG9xx controller)

Layout: **160-byte (`0xA0`) header + AES-encrypted payload**.

### Header fields (little-endian)

| Offset | Size | Field | Notes |
|--------|------|-------|-------|
| `0x00` | 4 | CRC32 | Over the **160-byte header only** (`0x04..0xA0`), *not* the payload. Poly `0xEDB88320` (standard reversed CRC-32); installer uses `new Crc32(3988292384u)`. Verified empirically against 5.6.1 / 6.6.2 / 7.1.6 / 7.4.6 (all pass) by `aceclient fwi-info`. *(Correction: earlier revisions of this note said `0x04..EOF`, which does not match.)* |
| `0x04` | 4 | Header length | Always `0x000000A0` (160). |
| `0x08` | 2 | Type | `0x13` (19) for 4.12.0 / 5.6.1 / 6.6.2; `0x21` (33) for 7.x. |
| `0x0A` | 2 | Magic | `0xA1FE` constant. |
| `0x0C` | 4 | (constant) | Controller images: `0x080101F0`. |
| `0x10` | 4 | (constant/flags) | e.g. `0x0047B001` on 7.4.6. |
| `0x14` | 4 | Payload length | e.g. `0x001BC100` = 1,818,880 on 7.4.6. |
| `0x18` | 8 | padding | `0xFF…` |
| `0x20` | 64 | **Wrapped-key block** | Present (high-entropy) from 6.6.2 onward; all `0xFF` (empty) on 4.12.0 / 5.6.1. |
| `0x60` | 16 | **Section map** | Four `uint32` offsets/sizes on 7.x; all `0xFFFFFFFF` before. |
| `0x70` | 48 | padding | `0xFF…` to `0xA0`. |
| `0xA0` | … | **Encrypted payload** | Entropy 8.000. |

7.x section-map example (7.4.6): `0x000CAD90, 0x000E53B0, 0x000CAE30,
0x00CFBFB0`. The last value (`0x00CFBFB0` = 13,614,000) is constant across 7.x —
likely the decrypted-image / flash region size. Values are uninterpreted
(payload is encrypted).

### Per-version evolution (observed)

| Version (date) | Type | Key block `0x20` | Section map `0x60` | Interpretation |
|----------------|------|------------------|--------------------|----------------|
| 4.12.0 (2020-12) | `0x13` | empty | empty | Static device key "A", no per-image wrap |
| 5.6.1-A / -B | `0x13` | empty | empty | Static device key "A"/"B" |
| 6.6.2-BL-upgrade | `0x13` | **present** | empty | **Wrapped-key block introduced** (≈ 6.0.0 "new bootloader, no downgrade") |
| 7.1.6 / 7.3.0 / 7.4.6 | `0x21` | present | **present** | **Header type bump** (≈ 7.0.0, A/B unified) |

So "encryption added in a release" is **not** accurate — payload encryption
predates 4.12.0. What was added at release boundaries is the **key-wrapping /
header scheme** (wrapped key ≈6.0.0; type+section map ≈7.0.0).

### Build pipeline (from installer source, display-resource variant)

`WriteFWUFile(allData)`:
1. `GetDataInBin(data)` — prepend `[CRC32][uint length][0][0]` wrapper.
2. `DeflateData(data)` — raw DEFLATE, then prepend zlib header `78 9C`.
3. `AESEncrypt(...)` — AES-128-CBC, zero IV (display variant; see `03`).
4. `CreateFWIHeader(len)` — the 160-byte header above.
5. Concatenate header + ciphertext.

> The controller (5.x–7.x) images use the **same family** of layout but a
> different type/magic and a **device-held key** — not the installer's
> display-resource key. See `03` and `04`.

### Sidecar files (in the official download zips)

Each release zip ships the `.fwi` plus:

- `*-cert.pem` — signing certificate chain.
- `*-sig.txt` — base64 **RSA-2048 / SHA-256** detached signature.
- `*-info.txt` — `Valid until <timestamp>` package validity window.

Signing cert (issuer `CN=ACE firmware signing CA`, O=Alfen N.V. R&D):

- 4.12.0 (2020): leaf `CN=ACE firmware signing of Thomas Nederlof`
  (named engineer, email in cert), RSA-2048.
- 7.4.6 (2026): leaf `CN=ACE firmware signing of NG9 firmware release pipeline`
  (automated).

The detached signature does **not** verify over naive byte ranges of the file
(whole file, payload-only, header-stripped) under PKCS#1 v1.5 — it covers an
internal structure (likely the decrypted payload or a header digest). Not
pursued further.

---

## 2. `.tfw` container (AHP application — "TVF")

Layout: **plaintext manifest + certificate chain + ECDSA signatures, then an
encrypted payload** (entropy 8.000 from ~`0x800` to EOF).

### Observed structure (AHP 2.7.0 FULL)

- `~0x16`: ASCII manifest string, e.g.
  `",Devices(s)= 2.7.0-0-gbf54430e4, SCB= 2.7.0, (FULL release manifest created by Jenkins)"`.
- Header region: ASN.1 **ECDSA P-384** signatures (`30 65 02 30 … 02 31 …`).
- Embedded **X.509 certificate**:
  - Subject `CN=AHP firmware signing for Jenkins Pipeline`
  - Issuer `CN=AHWP firmware signing CA` (O=Alfen N.V. R&D, C=NL)
  - Key: **id-ecPublicKey, secp384r1 (P-384)**; sig alg **ecdsa-with-SHA256**
  - Validity example: 2025-09-09 → 2027-09-09
- `~0x800 → EOF`: encrypted payload.

### TVF builder (installer, `ICUTVFCreator`)

The installer's local TVF builder (`CreateTvfData`) writes a `TvfHeader` + an
**inner tar** containing a JSON manifest (`AhpManifest`), a JSON video-resource
descriptor (`AhpVideoResource`), and an image — and is **not encrypted**. This
is the *local service* variant (e.g. display/video resources); the official AHP
releases are built, encrypted, and ECDSA-signed by Alfen's build pipeline.

### AHP release component types (2.0.0 set)

`FULL`, `SCB`, `NFC`, `SB` — all encrypted from ~`0x800`, all entropy ~8.0,
including the small `NFC` (358 KB) and `SB`/secure-boot (407 KB) parts. No
plaintext component exists in the set.
