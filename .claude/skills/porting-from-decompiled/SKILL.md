---
name: porting-from-decompiled
description: Conventions for porting or extending the aceclient Go code from the decompiled Alfen ACE installer. Use when porting C# from firmware/decompiled/*.cs into Go, adding or extending internal/ packages (fwi, tvf, scn, api, eds, isah, fwucreator, keytest, ...), implementing a container format or wire protocol from the installer, or reconciling the docs/ analysis notes against decompiled source or real firmware.
---

# Porting decompiled C# → Go (aceclient)

The `aceclient/` module is a clean-room 1:1 port of the decompiled Alfen ACE Service
Installer. When you add to it or port a new piece, follow the house rules:

- **Port 1:1.** Preserve the C# constants, field order, and structure. Do not
  "improve" an algorithm or constant during a port — correctness is defined by
  matching the installer and the real firmware, not by taste.
- **Cite the source.** Reference the decompiled location in a Go doc comment:
  `firmware/decompiled/<File>.cs`, the class, and the line range. The existing
  packages already do this (e.g. `alfencrc`, `tvf`, `fwucreator`) — match that style.
- **Trust the existing packages.** They are validated against real firmware. Extend
  them rather than re-auditing or second-guessing their constants.
- **Code/firmware wins over notes.** When a `docs/01`–`docs/06` note disagrees with
  the decompiled source or a real firmware file, the code/firmware is authoritative —
  fix the note (the port has already corrected several, e.g. the `.fwi` header CRC
  covering the 160-byte header only).
- **Validate against fixtures.** Where real firmware exists under `firmware/`, add or
  extend a test that parses/round-trips it. See `aceclient/ARCHITECTURE.md` for the
  fixture-skip behavior (tests `t.Skip` when `firmware/` is not populated).

## Scope boundary

No defeat of firmware signing / secure boot. Firmware is uploaded byte-for-byte. The
`keytest` sweep only helps against a *constrained* key space (wordlist / device-derived
keys); AES-128 firmware payloads are not brute-forceable, and the tool stays honest
about that.
