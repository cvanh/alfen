# 02 — Encryption & Key Analysis

Analysis of the encryption scheme and tests for key/IV reuse across the
firmware images. All tests are **ciphertext-only** (no key required).

---

## 1. Payloads are fully encrypted

| Measure | Result |
|---------|--------|
| Payload entropy (`.fwi`, from `0xA0`) | **8.000** bits/byte |
| Payload entropy (`.tfw`, from ~`0x800`) | **~7.997** bits/byte |
| Oldest image (4.12.0, Dec 2020) | also 8.000 — already encrypted |
| binwalk / magic carving | nothing (gzip/bzip2 hits = statistical false positives in random data) |

Conclusion: AES-or-equivalent encryption on every image back to the oldest
public release. **Encryption was not "added" in any release in the public
range** — it predates all of it.

---

## 2. Cipher mode & key-reuse tests

Run pairwise across all `.fwi` images (and `.tfw`):

| Test | What a shared key would reveal | Result |
|------|-------------------------------|--------|
| **ECB duplicate 16-byte blocks** (within file) | many repeats | **0** in every file → not ECB |
| **Shared ciphertext blocks** (across files) | identical blocks if ECB+same key | **none** |
| **Global XOR entropy** (stream/CTR nonce reuse) | `< 8.0`, structured `P1⊕P2` | **7.999** every pair |
| **Windowed-min XOR entropy** (reuse in any aligned region) | an entropy dip | **~7.75, identical for every pair** |
| **Common ciphertext prefix** (CBC same key+IV) | identical leading blocks | **0 bytes** |

Decisive point: the pair that *should* share a key (**4.12.0 vs 5.6.1-A**, both
key set "A") is statistically indistinguishable from the known-different-key
control (**5.6.1-A vs 5.6.1-B**, A vs B). ~7.75 in a 1 KiB window is the normal
finite-sample dip for random bytes, not a reuse signal.

### Interpretation

- Images use a **fresh per-image IV/nonce** (CBC or CTR family, **not ECB**).
- This is the correct construction and makes **key equality undetectable from
  ciphertext**. Even two images that *do* share the underlying key look
  independent.
- Therefore "are the other files encrypted with the same key?" is **not
  answerable from the ciphertext** — by design.

---

## 3. Key segmentation (from metadata / Alfen KB / header analysis)

The images are **not** all one key:

| Scheme | Versions | Key location |
|--------|----------|--------------|
| Key set **"A"** | ≤ 4.12 | static, device-held |
| Key set **"B"** | 4.14 – 6.x | static, device-held |
| A/B unified | 7.0.0+ | — |
| **Per-image wrapped key** | 6.6.2+ (header `0x20`) | wrapped to a device key, carried in image |

- `-A` / `-B` filename suffixes = **encryption key sets**, not partition slots.
- Alfen KB (KA-01186 / KA-01244): "Between 4.14 and 7.0.0 two encryption keys
  exist"; 4.12 = "last using encryption A"; 6.6.2 = "last providing both A and
  B"; 7.0.0 removed the choice.
- Pre-6.0 images (empty key block) imply a **static content key** per scheme:
  one key recovered from a device would decrypt every image of that era. Post-6.0
  images carry a per-image wrapped key. **Both keys live on the device, not in
  the files.**

---

## 4. Signing

| Package | Signature | Chain |
|---------|-----------|-------|
| `.fwi` (NG9xx) | detached **RSA-2048 / SHA-256** (`*-sig.txt`) | leaf → `ACE firmware signing CA` (Alfen N.V. R&D) |
| `.tfw` (AHP) | **ECDSA P-384 / SHA-256** (ASN.1 in header) | `AHP firmware signing for Jenkins Pipeline` → `AHWP firmware signing CA` |

Signing identity drifted from a **named engineer** (4.12.0, 2020) to an
**automated pipeline** (7.4.6). The two subsystems (NG9xx vs AHP) use **different
signing algorithms and CAs**.

---

## 5. Net

- Encryption is well-formed (per-image IV, no ECB, no detectable reuse).
- Key-equality cannot be proven or disproven from ciphertext.
- By design there are ≥2 content keys (A, B) plus per-image wrapped keys, all
  **device-held**. None are present in the firmware files.
