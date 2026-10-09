# 05 — Charger Communication: UDP Discovery & Settings Protocol

How the ACE Service Installer talks to a charger. There are **two independent
channels**, decompiled from `ACENetwork.dll` (see
[`decompiled/ACENetwork.cs`](decompiled/ACENetwork.cs),
[`decompiled/SCNNetwork.cs`](decompiled/SCNNetwork.cs),
[`decompiled/SCNSocket.cs`](decompiled/SCNSocket.cs)):

| Channel | Transport | Direction | Purpose |
|---------|-----------|-----------|---------|
| **SCN** | **UDP broadcast, port 36549** | charger → tool (listen only) | Live discovery + load-balancing status |
| **Control/Settings** | **HTTPS REST `/api/…`** | tool ↔ charger | Login, read/write settings, profiles, logs, firmware upload |

**Settings are NOT sent over UDP.** UDP is read-only telemetry; all
configuration goes over the authenticated HTTPS channel.

---

## 1. SCN UDP protocol (discovery + load balancing)

Class `ICUNetwork.SCNNetwork`. "SCN" = Smart Charging Network — chargers in a
load-balancing group broadcast their live state so peers (and the installer) can
coordinate current sharing.

### Transport & flow

- **Port:** `UDPPORT = 36549` (UDP). Binds `IPAddress.Any:36549`,
  `SO_REUSEADDR`, listens for broadcasts.
- Two tasks: a **receive** task (`UdpClient.ReceiveAsync` → queue) and a
  **process** task (`ProcessUdpPacket`).
- **Packet size gate:** ignored if `>= 1024` bytes or `< 96` bytes after
  decrypt. Because AES-CBC with `PaddingMode.None` is used, the ciphertext is a
  multiple of 16 → valid packets are **96 … 1008 bytes, 16-byte aligned**.
- Each packet is decrypted, then validated: byte `[0]` must be a known
  `ScnLibraryVersions` value; otherwise dropped.
- Device key = `uniqueId (@48) + socketIndex (@23) + source IP`.

### Encryption

```
Algorithm : AES-128-CBC  (RijndaelManaged, PaddingMode.None)
Key       : DE 0B 4D E8 71 58 F4 EF 8A 8B 6C 24 60 82 74 70   (hardcoded, static)
IV        : 00 × 16  (all zero)
```

The key is **hardcoded and shared** across all chargers — so these status
broadcasts are effectively obfuscated, not secured. Anyone on the L2 segment can
decrypt them. (This is a monitoring/telemetry channel, not a control channel.)

### Decrypted packet layout (little-endian)

Parsed by `SCNSocket.ParseData`. Offsets verified against the reader order:

| Offset | Type | Field | Meaning |
|-------:|------|-------|---------|
| `0x00` | u8 | ScnLibVersion | protocol/library version (gates v2/v3 fields) |
| `0x01`–`0x03` | u8×3 | reserved | |
| `0x04` | u32 | Timestamp | |
| `0x08` | char[8] | NetworkName | UTF-8, SCN group name (empty ⇒ packet dropped) |
| `0x10` | u16 | Id | |
| `0x12` | u8 | reserved | |
| `0x13` | u8 | Mode3State | IEC 61851 Mode-3 pilot state |
| `0x14` | u8 | State | `EChargingState` (Idle/charging/…) |
| `0x15` | u8 | PhaseMask | connected phases bitmask |
| `0x16` | u8 | TotalNumberOfSockets | |
| `0x17` | u8 | SocketIndex | which socket this packet is for |
| `0x18` | char[21] | Name | UTF-8 socket/station name |
| `0x2D` | u8 | SocketCount | |
| `0x2E` | u16 | MaximumGroupID | |
| `0x30` | u64 | UniqueID | station unique id (also used as socket key) |
| `0x38` | u64 | Clock | |
| `0x40` | u32 | WaitingSince | |
| `0x44` | i32 | AlterningSince | |
| `0x48` | f32 | MinimumCurrent | A |
| `0x4C` | f32 | MaximumCurrent | A |
| `0x50` | f32 | ActiveCurrentL1 | measured, A |
| `0x54` | f32 | ActiveCurrentL2 | A |
| `0x58` | f32 | ActiveCurrentL3 | A |
| `0x5C` | f32 | AvailableCurrentL1 | A |
| — | — | — | **← 96-byte minimum boundary** |
| `0x60` | f32 | AvailableCurrentL2 | A |
| `0x64` | f32 | AvailableCurrentL3 | A |
| `0x68` | f32 | SetPointCurrent | commanded, A |
| `0x6C` | u32 | ActiveChargingTime | |
| `0x70` | u32 | AlternatingCountDown | |
| `0x74` | f32¹ | PropSocketSafeCurrent | ¹ v<3: u16/10 |
| `0x78` | f32¹ | PropMaximumStaticCurrent | ¹ v<3: u16/10 |
| `0x7C` | u16 | PropAlternatingPeriod | |
| `0x7E` | u8 | PropChangedCounter | |
| `0x7F` | u8 | OptionByte | |
| `0x80` | u16 | ExtraCurrentL1 | v≥2 |
| `0x82` | u16 | ExtraCurrentL2 | v≥2 |
| `0x84` | u16 | ExtraCurrentL3 | v≥2 |
| `0x86` | i16 | (pad) | v≥2 |
| `0x88` | f32 | MaximumGroupCurrent | v≥2, A |
| `0x8C` | f32 | PropTotalSafeCurrent | v≥3, A |

Trailing fields depend on `ScnLibVersion` (v2 adds the Extra*/group fields; v3
promotes the safe/static currents to float and adds `PropTotalSafeCurrent`).

---

## 2. Control / settings protocol (HTTPS REST)

Classes `ICUNetwork.ICUDevice` / `ICULanDevice` / `LANConnection`. This is the
real configuration channel.

### URI scheme

```
{Protocol}://{IPAddress}:{Port}/api/{command}?{parameters}
```

`Protocol` is `https` (preferred; `http` legacy). Content type for bodies is
`application/json; charset=utf-8` (`SendStandardPost`).

### Authentication (token-based)

- `POST /api/login` with body
  `{"username":"<u>","password":"<p>","displayname":"<d>"}`.
- Returns an access token + refresh token; `POST /api/token/refresh` with
  `{"refresh":"<t>"}` renews it. `POST /api/logout` ends the session.
- Password security was introduced in FW 5.3.0 (per Alfen KB); the installer
  needs a password for the device from that version on.

### The settings model — a CANopen-style object dictionary

Every setting is an **`ICUProperty`** addressed by:

- **`propId`** — 16-bit index (hex), e.g. `0x8317`
- **`subId`** — 8-bit subindex (hex), e.g. `0x02`

Identified on the wire as the string **`"<propId:X>_<subId:X>"`** (e.g.
`"8317_2"`). This is a CANopen object dictionary (index/subindex) exposed over
REST.

**Read:** `GET /api/prop?...` (`UpdatePropertiesInternal`). JSON response is
paged:

```json
{ "version": …, "total": N, "offset": O, "count": C,
  "properties": [ { "id":"8311_0", "value":…, "name":…, "dt":…, "cat":…,
                    "len":…, "ro":0|1 }, … ] }
```

Each entry becomes a typed property (data type, category, max length, read-only
flag) in the `ICUPropertyDictionary`.

**Write:** `POST /api/prop` with a JSON object batching **up to 15** properties
(`StoreProperties`):

```json
{ "<name>": { "id":"<IDX>_<SUB>", "value": <v> }, … }
```

Value encoding by `SDT` data type:

| SDT type | Encoding in `value` |
|----------|---------------------|
| `BYTEARRAY` | hex CSV string, e.g. `"0A,1F,FF"` (each byte `X2`) |
| `ARRAY_16` | hex CSV of `X4` words |
| `UNICODE_STRING` / `VISIBLE_STRING` / `DOMAIN` / `BOOLEAN` | quoted string |
| `REAL32` / `REAL64` | number, en-US formatting |
| integer types | raw number |

Helper accessors: `GetPropertyInt/UInt/Bool/String/Double/UInt64(propId, subId)`
read the cached dictionary; `storeProperty(propId, subId, value)` marks a
property changed and pushes it via `StoreProperties`.

### Other endpoints (observed)

| Command | Use |
|---------|-----|
| `login` / `logout` / `token/refresh` | auth/session |
| `prop` | read/write settings (object dictionary) |
| `prc` | (command channel; excluded from the 429 back-off) |
| `chargingprofiles` | smart-charging profiles: `id_list`, `cpid=<id>`, `add=`, `clear=all`/`clear=<id>` |
| `domain` | bulk settings domain push (POST JSON) |
| `log` | device log lines (`offset=`, `lines=`) |
| `firmware` | **firmware upload** (see below) |

### Firmware upload (for completeness)

`UploadFirmware → StartUpload → ExecuteWebRequest("firmware", …, fileData)`.
Body is **`multipart/form-data`**, field name `firmwarefile`, over HTTPS the file
is split into **4096-byte parts**. The `.fwi`/`.tfw` is sent **as-is** — the
installer does not decrypt it (see [`03`](03-ace-installer-analysis.md)); the
device decrypts internally.

---

## 3. Connection types

- **`LANConnection` / `ICULanDevice`** — the HTTPS path above (Eve/NG9xx over LAN/Wi-Fi).
- **`COMPortInterface`** — serial/`SendKey`/`SendCommand` path (local console).
- **`ICUDevice`** — common device/property model shared by both.

---

## 4. Security notes

- The **UDP SCN key is static and shared** across the fleet → status broadcasts
  are decryptable by any LAN peer. Treat SCN data as non-confidential telemetry.
- The **control channel is authenticated HTTPS** with per-device credentials and
  tokens — this is where real security sits. Certificate handling is in
  `CertificateStore` / `CertificateMap`.
- Nothing here weakens the **firmware** encryption/signing: firmware is uploaded
  opaque and verified/decrypted on-device.

---

## Source map

| Topic | File · class |
|-------|--------------|
| UDP mechanics | `decompiled/SCNNetwork.cs` · `SCNNetwork` |
| UDP packet struct | `decompiled/SCNSocket.cs` · `SCNSocket.ParseData` |
| HTTP request/URI/auth | `decompiled/ACENetwork.cs` · `ICUDevice.ExecuteWebRequest`, `TryGetUri`, `SendStandardPost` |
| Property read | `ACENetwork.cs` · `UpdatePropertiesInternal`, `ParseProperty` |
| Property write | `ACENetwork.cs` · `StoreProperties`, `storeProperty` |
| Charging profiles | `ACENetwork.cs` · `ICUChargingProfiles` |
