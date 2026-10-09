# 04 — Conclusions & Next Steps

## Verdict

| Question | Answer |
|----------|--------|
| Are the `.fwi` / `.tfw` encrypted? | **Yes** — full-entropy payloads, every version incl. the oldest (4.12.0, 2020). |
| Was encryption "added" in a release? | **No.** Encryption predates the public range. What changed at release boundaries is the **key-wrapping / header scheme** (wrapped-key block ≈6.0.0; header type + section map ≈7.0.0). |
| Can the files be decrypted from what's on disk? | **No.** No content key ships in the packages; keys are device-held. |
| Are two images encrypted with the same key? | **Not determinable from ciphertext** — per-image IV randomisation hides key equality. By design there are ≥2 content keys (A, B) + per-image wrapped keys. |
| Does the ACE Service Installer decrypt firmware? | **No.** It uploads `.fwi`/`.tfw` verbatim; the device decrypts. Its three hardcoded keys are for display packaging, UDP discovery, and settings — not firmware. |

**Bottom line:** the host-side avenue is closed. Reading the controller / AHP
firmware requires a **plaintext image**, which only exists on the device or at
Alfen.

---

## Routes to a plaintext image (if pursued, on owned hardware)

1. **Device-side dump** — recover the running, already-decrypted firmware:
   - Identify the NG9xx SoC/MCU and check whether flash readout protection is
     fused.
   - UART/console access, eMMC/SPI-flash dump, or JTAG/SWD.
   - Subject to secure-boot / readout-protection; may yield the decrypted image
     and/or the static content key ("A"/"B"), which would then decrypt same-era
     images offline.
2. **Vendor channel** — request a plaintext image / SDK / key material from
   Alfen (R&D / PSIRT) under NDA, e.g. for interoperability or coordinated
   security research.

Both are **device/vendor** actions — the files and installer alone cannot get
there.

---

## Legitimate spin-offs already unlocked (host-side)

From the installer decompile (`03`), without any device work:

- **Build / read custom display-resource `.fwi` packages** — the full build
  recipe is known: `GetDataInBin` wrapper (CRC32 `0xEDB88320` + length) →
  DEFLATE (+ zlib `78 9C`) → **AES-128-CBC (key `29C66A20AE16C4BA046A21D57A78664F`,
  IV=0)** → 160-byte `CreateFWIHeader`. This is enough to produce or inspect the
  installer's own logo/image/language packages.
- **Decode SCN discovery/status UDP packets** on the LAN using the SCN key
  `DE0B4DE87158F4EF8A8B6C2460827470` (AES-128, IV=0, no padding) — useful for
  device discovery / telemetry interop.

---

## Boundary note

This analysis covers formats, mechanisms, and keys found in software/firmware on
owned hardware, for understanding and interoperability. It stops short of
anything that would defeat the firmware **signature / secure-boot** chain to run
modified firmware on chargers — that is out of scope and not pursued here.

---

## Tooling reference

| Step | Tool |
|------|------|
| Entropy / XOR / block tests | Python (`math`, `collections`) |
| Signature carving / X.509 | `openssl x509`, `binwalk` |
| MSI → CAB (fragmented OLE stream) | Python `olefile` |
| CAB extraction | `cabextract` |
| .NET decompile → C# | `ilspycmd` (ILSpy 11.x) |
| .NET metadata (fallback) | `dnfile` / `dncil` |
