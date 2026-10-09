# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repo is

Clean-room Go reimplementation (`aceclient/`) of the charger-facing surface of the
**Alfen ACE Service Installer**, for firmware-format research and interoperability on
owned Alfen NG-platform EV chargers. Alongside it live the reverse-engineering notes
(`docs/`) and reference material (`firmware/`) it derives from.

Keep the three source layers straight:

- `firmware/decompiled/*.cs` — C# from the real installer; the **ground-truth reference** the Go code ports 1:1.
- `docs/01`–`docs/06` — analysis notes. Context, **not** ground truth (the port has corrected errors in them).
- `firmware/` — real files / MSI extractions / presets, used as **test fixtures** (tests skip when absent).

**Scope boundary (do not cross):** no defeat of firmware signing / secure boot;
firmware is uploaded byte-for-byte. The `keytest` sweep only helps against a
*constrained* key space — AES-128 firmware payloads are not brute-forceable.

## Commands

All Go work runs inside `aceclient/` (the module root; CI sets `working-directory:
aceclient`). CGO is disabled everywhere — `fyne` is an unused indirect dep, so builds
are pure-Go and cross-compile cleanly.

```
cd aceclient
go build ./cmd/aceclient            # build the CLI (the only main package)
./aceclient selftest                # end-to-end crypto round-trip, no device needed
go test ./...                       # full suite (fixture-backed tests t.Skip when firmware/ is absent)
go test ./internal/fwi -run TestX   # single package / single test
gofmt -l .                          # CI fails on any unformatted file; fix with gofmt -w .
golangci-lint run                   # CI lint gate
```

CI (`.github/workflows/ci.yml`) runs format → lint → test → cross-compile build on
changes under `aceclient/**`, and releases binaries on push to `main`.

## Where the rest lives

- **Architecture, package map, test-fixture behavior:** [`aceclient/ARCHITECTURE.md`](aceclient/ARCHITECTURE.md)
- **Porting decompiled C# → Go (conventions):** the `porting-from-decompiled` skill, which loads on demand when you touch `firmware/decompiled/*.cs` or extend the `aceclient` packages.
