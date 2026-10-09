# 03 — ACE Service Installer Reverse-Engineering

Target: `ACE Service Installer v4.4.1_434.msi`.

---

## 1. Extraction

- MSI = OLE2 compound document (built with **WiX Toolset 6.0.2**, author
  "Alfen N.V."). Payload files are in an **embedded CAB** (`MSCF`).
- The CAB stream is **fragmented across OLE sectors** — a flat byte-carve from
  the `MSCF` offset yields a cabinet whose directory lists correctly but whose
  data blocks fail CRC (checksum errors → 0-byte extracts). Must read the stream
  through an OLE reader (e.g. Python `olefile`) to reassemble it, then
  `cabextract`.
- Result: **55 files**, a **.NET** application.

### Notable assemblies

| File | Role |
|------|------|
| `ACEServiceInstaller.exe` | Main app (WPF/Xwt) |
| `ACEFWUCreator.dll` | **F**irm**W**are-**U**pdate package **creator** (display/UI resources) |
| `ACENetwork.dll` | Device connection + **SCN UDP** discovery |
| `ACESettings.dll` | Settings / password storage |
| `ACEISAHConnection.dll` | ISAH device connection |
| `ACELogoConvertor.dll` | Logo/image conversion |
| `InstallerConfigV3.dat` | 3.9 MB config blob |

Tooling used: `olefile` (OLE), `cabextract` (CAB), `ilspycmd` (ILSpy
decompiler) for C#.

---

## 2. Three crypto subsystems — all hardcoded keys, none for firmware

### (a) `ACEFWUCreator.ICUFWUCreator.AESEncrypt` — display-resource packaging

```csharp
byte[] rgbKey = { 41,198,106,32,174,22,196,186,4,106,33,213,122,120,102,79 };
byte[] rgbIV  = new byte[16];                 // all zero
var t = new RijndaelManaged().CreateEncryptor(rgbKey, rgbIV);  // AES-128-CBC, PKCS7
```

- **Key (hex): `29 C6 6A 20 AE 16 C4 BA 04 6A 21 D5 7A 78 66 4F`** (AES-128-CBC, zero IV).
- **Encrypt only** (no decryptor in this class).
- Used by `WriteFWUFile` to build display/UI resource `.fwi` packages
  (logos, images, fonts like `OBJECT_ROBOTICA_REGULAR_30`, languages, video).
- `ICUTVFCreator` builds the matching **TVF** (unencrypted: manifest JSON +
  video-resource JSON + image tar).
- **Verified NOT a firmware key:** decrypting the real payloads
  (7.4.6 / 5.6.1-A / 4.12.0 `.fwi`, 2.7.0 `.tfw`) with this key + AES-128-CBC +
  zero IV yields **pure noise** (no zlib `78 9C`, no structure). It only unlocks
  the installer's own display-resource packages (different header magic).

### (b) `ACENetwork.SCNNetwork.AESDecrypt` — UDP discovery packets

```csharp
static readonly byte[] s_aesKey = { 222,11,77,232,113,88,244,239,138,139,108,36,96,130,116,112 };
// InitializeAESDecryptor: RijndaelManaged { Padding = PaddingMode.None }, zero IV
m_decryptor = rijndaelManaged.CreateDecryptor(s_aesKey, new byte[16]);
```

- **Key (hex): `DE 0B 4D E8 71 58 F4 EF 8A 8B 6C 24 60 82 74 70`** (AES-128, zero IV, no padding).
- Decrypts **SCN discovery/status UDP broadcasts** from chargers on the LAN
  (`StartUdpReceiveTask`); parses `SCNSocket` library version, socket index,
  `uniqueId`. **Not firmware.**

### (c) `ACESettings.dll` — config & password encryption

- `RijndaelManaged` + `PasswordDeriveBytes(saltValue, passwordIterations)`
  (PBKDF), with `EncryptDecrypt` and encryption types
  `encryptNone` / `encryptBase64` / `encryptRijndael` / `encryptRijndaelHashed`.
- Protects stored config values and passwords (`APNPassword`, `HTTPPassword`,
  `ftpPassword`, GPRS APN, etc.) and `ICUEncryptionType`. **Not firmware.**

---

## 3. The firmware upload path does NOT decrypt

`ACENetwork` → `UploadFirmware(bgw, fileName, …)` → `StartUpload(bgw, fileData,
isFirmwareFile, …)`:

```csharp
ExecuteWebRequest("firmware", string.Empty, fileData, 900000, 1, suppressPopups: true, …);
```

- The `.fwi` / `.tfw` bytes are read and **POSTed to the device verbatim**.
- No AES/Rijndael, no `CreateDecryptor`, no transform applied to `fileData`.
- The device receives the still-encrypted image and **decrypts it internally**.

---

## 4. The nine 256-bit constants are not keys

Nine 64-hex-char constants in `ACEFWUCreator.dll` sit inside a
`__StaticArrayInitTypeSize=500` block next to identifiers like
`OBJECT_ROBOTICA_REGULAR_30`, `font_robotica_30`, `DISP_FOR_FUTURE_USE_10`.
They are **display font/object hashes** used for image validation
(`ValidateImageFile`), not cryptographic keys.

---

## 5. Summary of installer crypto

| Subsystem | Key (hex) | Alg | Direction | Protects |
|-----------|-----------|-----|-----------|----------|
| ACEFWUCreator | `29C66A20AE16C4BA046A21D57A78664F` | AES-128-CBC, IV=0 | encrypt | Display/UI resource packages |
| ACENetwork/SCN | `DE0B4DE87158F4EF8A8B6C2460827470` | AES-128, IV=0, no pad | decrypt | UDP discovery/status packets |
| ACESettings | PBKDF-derived | Rijndael | both | Config values & stored passwords |
| **Firmware (.fwi/.tfw)** | **— none —** | — | **uploaded verbatim** | Decrypted **on device** |

> Keys above are extracted from software on the analyst's own machine and apply
> to the analyst's own equipment (building display packages, decoding discovery
> traffic). They do **not** decrypt controller or AHP firmware.
