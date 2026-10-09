# Decompiled reference sources

C# decompiled from the ACE Service Installer v4.4.1_434 assemblies (via
`ilspycmd` / ILSpy 11.x). Kept for reference alongside the analysis docs.
Decompiler output — identifiers and structure are faithful, but it is not the
original source and won't compile verbatim.

## Files

| File | Source assembly | Key contents |
|------|-----------------|--------------|
| `ACEFWUCreator.cs` | `ACEFWUCreator.dll` (full) | `ICUFWUCreator` (display-resource `.fwi` builder + AES-128 `AESEncrypt`), `ICUTVFCreator` (TVF/`.tfw` builder), `Crc32`, `TvfHeader` |
| `ACENetwork.cs` | `ACENetwork.dll` (full) | `SCNNetwork`/`SCNSocket` (UDP), `ICUDevice`/`ICULanDevice`/`LANConnection` (HTTPS control), `ICUProperty*` (settings model), `ICUChargingProfiles`, `UploadFirmware` |
| `ACESettings.cs` | `ACESettings.dll` (full) | `EncryptDecrypt` (Rijndael + `PasswordDeriveBytes`) for config values & passwords |
| `SCNNetwork.cs` | (subset) | UDP listener, AES decrypt, packet dispatch |
| `SCNSocket.cs` | (subset) | UDP packet field parser (`ParseData`) |
| `ICUFWUCreator.cs` | (subset) | `.fwi` build pipeline + `AESEncrypt` key |
| `ICUTVFCreator.cs` | (subset) | TVF inner-tar builder |

## Quick landmarks

- **UDP key / mechanics:** `SCNNetwork.cs` → `s_aesKey`, `UDPPORT = 36549`,
  `InitializeAESDecryptor`, `ProcessUdpPacket`.
- **UDP packet layout:** `SCNSocket.cs` → `ParseData`.
- **Display `.fwi` AES key:** `ICUFWUCreator.cs` → `AESEncrypt` (`rgbKey`).
- **Settings read/write:** `ACENetwork.cs` → `UpdatePropertiesInternal`,
  `ParseProperty`, `StoreProperties`, `storeProperty`, `ExecuteWebRequest`,
  `TryGetUri`.
- **Firmware upload (verbatim):** `ACENetwork.cs` → `StartUpload`,
  `SendMultipartFormDataContent`.

See [`../03-ace-installer-analysis.md`](../03-ace-installer-analysis.md) and
[`../05-udp-and-settings-protocol.md`](../05-udp-and-settings-protocol.md) for
the written analysis.

## Provenance

- `ACE Service Installer v4.4.1_434.msi` → OLE2 → embedded CAB → 55 files.
- Full extracted file set (assemblies, configs, resources) lives under
  `firmware/msi_work/files3/` in this workspace; only the decompiled sources relevant to
  the analysis are committed here.
