# Alfen NG-Platform Firmware & ACE Service Installer — Analysis Notes

Reference notes from a reverse-engineering / interoperability analysis of Alfen
NG-Platform EV-charger firmware packages (`.fwi`, `.tfw`) and the **ACE Service
Installer** (`.msi`). Produced for understanding the firmware update format on
owned hardware.

## Scope & intent

- Target: Alfen NG9xx controller firmware (`.fwi`) and AHP application releases
  (`.tfw` / TVF), plus the ACE Service Installer that deploys them.
- Purpose: understand the container formats, the encryption/signing scheme, and
  whether firmware decryption happens host-side or on the device.
- These notes document **formats and mechanisms**. The production firmware
  images remain encrypted; no key capable of decrypting controller/AHP firmware
  was found in the installer (see `03` and `04`).

## Index

| Doc | Contents |
|-----|----------|
| [`01-firmware-formats.md`](01-firmware-formats.md) | `.fwi` and `.tfw`/TVF container layouts, header fields, per-version evolution |
| [`02-encryption-key-analysis.md`](02-encryption-key-analysis.md) | Entropy, ECB/IV-reuse tests, key segmentation (A/B), signing chains |
| [`03-ace-installer-analysis.md`](03-ace-installer-analysis.md) | MSI extraction, .NET assemblies, the 3 crypto subsystems + keys, firmware-upload path |
| [`04-conclusions-next-steps.md`](04-conclusions-next-steps.md) | Summary verdict, what's achievable, device-side route |
| [`05-udp-and-settings-protocol.md`](05-udp-and-settings-protocol.md) | UDP SCN discovery protocol (port 36549, packet struct) + HTTPS settings/object-dictionary channel |
| [`06-go-port-gui-plan.md`](06-go-port-gui-plan.md) | Phased plan for porting the full installer + a cross-platform (Fyne) GUI onto the Go `aceclient` backend |
| [`../firmware/decompiled/`](../firmware/decompiled/) | C# decompiled reference sources (incl. full `ACEServiceInstaller.exe` project) |
| [`../aceclient/`](../aceclient/) | Go reimplementation of the ACE tool's charger surface (formats, SCN UDP, HTTPS settings, parallel key sweep); ports the decompiled sources 1:1 and validates against the real firmware |

## One-paragraph summary

The `.fwi` (NG9xx controller) and `.tfw` (AHP application, "TVF" format)
packages are **signed and encrypted**. Payloads are full-entropy (AES or
equivalent) with per-image IV randomisation, so key equality is not detectable
from ciphertext. Encryption predates the oldest public release (4.12.0, Dec
2020); what changed across releases was the **key-wrapping / header scheme** (a
wrapped-key block appears ~6.0.0, header type bumps at 7.0.0). The ACE Service
Installer contains three hardcoded-key crypto subsystems — display-resource
packaging, UDP discovery, and settings/passwords — but **none decrypt the
controller or AHP firmware**; it uploads those images verbatim and the device
decrypts them internally.

## Source artefacts referenced

- `firmware/Firmware/NG9xx *.fwi` — controller images (4.12.0, 5.6.1-A/B, 6.6.2-BL-A/B, 7.1.6, 7.3.0, 7.4.6)
- `firmware/Firmware/AHP_release_FW_2.*.tfw` — AHP application releases (2.0.0–2.7.0)
- `firmware/ACE Service Installer v4.4.1_434.msi` — deployment tool (WiX/.NET)
