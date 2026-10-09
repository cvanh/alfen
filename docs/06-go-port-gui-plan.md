# 06 — Porting the full ACE Service Installer to Go + a cross-platform GUI

Plan for growing the existing `aceclient` Go port from a CLI covering the
charger-facing protocol into a full cross-platform desktop replacement for the
Windows-only **ACE Service Installer** (`ACEServiceInstaller.exe`, .NET 4.8.1 /
WPF / Xwt).

## Decisions (locked)

| Decision | Choice | Why |
|----------|--------|-----|
| GUI toolkit | **Fyne v2** (`fyne.io/fyne/v2`, pinned `v2.8.1`) | Pure-Go, one toolchain for Win/Mac/Linux, form/table/widget oriented — natural fit for a panel-heavy device-config tool. Single binary; `fyne package` builds `.app`/`.exe`. Needs CGO + a C compiler (present on this machine). |
| Rollout | **Core flow first, then expand** | A runnable, useful app in days; panels added one phase at a time against the existing backend. |
| Binary layout | Separate GUI binary `cmd/aceinstaller` | Keeps the existing `cmd/aceclient` CLI **CGO-free**; GUI isolates the Fyne/CGO dependency. |
| UI logic home | `internal/ui` (Fyne) calling `internal/*` backend | UI stays thin; all device/protocol logic lives in reusable, testable `internal/` packages. |

## Current state (already ported — the backbone)

From README + source. These are clean 1:1 ports and are **reused as-is** by the GUI:

- `internal/scn` — SCN UDP discovery (`Listen`/`Run`, `ParseData`, AES-128).
- `internal/api` — HTTPS control: `Login`/`RefreshSession`/`Logout`, `ReadProperties`
  (paged), `StoreProperties` (15-batch, per-SDT encoding), `GetProperty`,
  `UploadFirmware` (multipart).
- `internal/fwi` — `.fwi` parse/build, display-resource wrap/unwrap, header CRC.
- `internal/tvf` — `.tfw` build header + real-file sniffer.
- `internal/alfencrc` — reflected-IEEE `Crc32`.
- `internal/keytest` — candidate-key sweep + oracles (research tool, not in installer).

## Backend gaps (domain logic still trapped in the installer's panels/dialogs)

The protocol layer is done; what remains is the **domain logic** that in the
`.exe` lives inside the ~20 panels and ~28 dialogs, plus two satellite
assemblies not yet ported. Each becomes an `internal/` package:

| Area | New package (planned) | Source in decompile |
|------|----------------------|---------------------|
| Object-dictionary metadata (names, types, categories, enums, ranges) | `internal/eds` | `EDS.xml` (110 KB) |
| Settings/config persistence (encrypted `InstallerConfig*.dat`) + connection profiles | `internal/config` | `ACESettings.cs` `EncryptDecrypt`, `AppProperties`, `PropertyStorage` |
| Presets (TCP/RTU/Backoffice) save/apply | `internal/presets` | `DlgPresets`, `AppProperties.Local*PresetsFolder` |
| Device commands (reboot, reset, password lifecycle, end-user PIN, unlock) | `internal/command` | `DlgCommand`, `DlgReboot`, `DlgDevice*Password`, `DlgEndUserPin`, `DlgUnlockFeature` |
| Load balancing + SCN management | `internal/scnmgmt` (extends `scn`) | `PanelLoadbalancing`, `PanelSCN*`, `DlgAddToSCN`, `DlgSCNPhaseMapping` |
| Authorization (RFID whitelist/tags) | `internal/authz` | `PanelAuthorization`, `DlgModifyTag` |
| Modbus register map | `internal/modbus` | `DlgModbusRegisterMap`, `ModbusRegmapUI*` |
| Monitoring / meter values / transactions / power / log | reads via `api` + view models | `PanelMonitoring/MeterValues/Transactions/Power/Log` |
| Wi-Fi scan + profiles | `internal/wifi` | `DlgScanWifiNetworks`, `UI.UIWifiProfile`, `ACENetwork` wifi |
| Display-resource / logo upload | `internal/logo` (+ existing `fwi`) | `ACELogoConvertor.dll`, `DlgUploadResources` |
| ISAH connection (if needed) | `internal/isah` | `ACEISAHConnection.dll` |
| i18n strings + tooltips | `internal/i18n` | `Lang_Eve_Mini_*.csv`, `tooltip_en_GB.csv` |

> Note: `System.Drawing` and the WPF/P-Invoke bits (`ArpList` → `IpHlpApi.dll`)
> from the `.exe` are **not** ported verbatim — ARP discovery is replaced by the
> existing SCN/mDNS discovery, and image work by Go-native libraries.

## Architecture

```
cmd/aceclient        existing CLI (CGO-free, unchanged)
cmd/aceinstaller     new GUI entrypoint (Fyne, CGO)
internal/ui          Fyne views; one file per installer panel/dialog
  app.go               window shell, connection bar, tab host, shared state
  discover.go          SCN discovery list  (→ PanelNoDevice / device picker)
  connect.go           login form          (→ DlgDeviceLogin)
  information.go       identity view       (→ PanelInformation)
  properties.go       property grid R/W    (→ PanelAllProperties)
  firmware.go         firmware upload      (→ DlgUpload)
  ...                  (added per phase)
internal/<domain>    backend packages from the gap table above
```

UI rule: a view never talks HTTP/UDP directly — it calls an `internal/` package
and marshals results onto widgets inside `fyne.Do(...)`.

## Phases

Effort estimates are rough engineering-days for one developer; they are
sequential but each phase ends in something runnable.

### Phase 0 — Plan + scaffold  *(this doc)*
- This plan committed; Fyne dependency added (`v2.8.1`); `cmd/aceinstaller`
  skeleton that builds and opens an empty window.
- **Done when:** `go build ./cmd/aceinstaller` succeeds and the window opens.

### Phase 1 — Core flow  *(next; ~3–5 days)*
- App shell: connection bar (IP/port/user/pass/insecure), tab host, status.
- **Discover** tab: SCN scan → device table → select fills connection fields.
- **Connect/Login** against `api`, token stored, connected/disconnected state.
- **Information** tab: identity view derived from a property read.
- **Properties** tab: `ReadProperties` into a table; select-to-edit + `StoreProperties` write-back.
- **Firmware** tab: file picker + `UploadFirmware`, with `fwi`/`tvf` pre-flight classification.
- **Done when:** discover → connect → read/edit a property → upload firmware works end-to-end against a device (or the `selftest` fixtures where no device).

### Phase 2 — Metadata + persistence  *(~4–6 days)*
- `internal/eds`: parse `EDS.xml` for property name/type/category/enum/range;
  property grid groups by category, validates input, renders enums as dropdowns.
- `internal/config`: port `ACESettings` `EncryptDecrypt`; read/write the
  encrypted `InstallerConfig*.dat`; persist connection profiles + app settings.
- **Done when:** grid shows friendly categories/enums with validation; app
  remembers connections and settings across restarts.

### Phase 3 — Device operations  *(~3–5 days)*
- `internal/command`: reboot, factory/settings reset, password lifecycle
  (temp/new/change/reset), end-user PIN, feature unlock.
- UI: a Commands/menu area wiring these with confirmations.
- **Done when:** each command round-trips against a device with proper guards.

### Phase 4 — Load balancing + SCN management  *(~4–6 days)*
- `internal/scnmgmt`: add-to-SCN, SCN overview/settings, phase mapping, sockets.
- UI: `PanelLoadbalancing`, `PanelSCNOverview/Settings`, `DlgAddToSCN`, `DlgSCNPhaseMapping`.
- **Done when:** a charger can be added to an SCN and load-balancing params set.

### Phase 5 — Authorization + Modbus  *(~3–5 days)*
- `internal/authz` (RFID whitelist/tags), `internal/modbus` (register map editor).
- **Done when:** whitelist CRUD and register-map view/edit work.

### Phase 6 — Dashboards  *(~3–4 days)*
- Read-mostly panels: Monitoring, Meter values, Transactions, Power, Log.
- **Done when:** live values render and refresh for a connected device.

### Phase 7 — Connectivity + Wi-Fi  *(~2–4 days)*
- `internal/wifi` (scan + profiles), `PanelConnectivity`/`PanelInterface`.
- **Done when:** scan networks, set a Wi-Fi profile, view interface status.

### Phase 8 — Firmware/logo authoring + presets  *(~3–5 days)*
- `internal/logo` (port `ACELogoConvertor`), display-resource upload, firmware
  builder dialogs, `internal/presets` apply/save.
- **Done when:** build/upload a display resource and apply a preset.

### Phase 9 — Polish + packaging  *(~3–5 days)*
- `internal/i18n` (CSV strings/tooltips), About/splash, consistent theming.
- Packaging: `fyne package` for macOS (`.app`, signing/notarization note),
  Windows (`.exe`), Linux; CI matrix build.
- **Done when:** signed/zipped artifacts build for all three OSes from CI.

## Cross-platform & packaging notes

- **macOS:** CGO + Cocoa/OpenGL via Xcode CLT. Distribution needs Developer ID
  codesign + notarization (or `xattr -d com.apple.quarantine` for local use).
  Apple Silicon native (`arm64`); universal binary optional.
- **Windows:** CGO via MinGW/MSVC; `fyne package -os windows` embeds icon/manifest.
- **Linux:** needs X11/GL dev headers at build; ships a single binary.
- The `cmd/aceclient` CLI stays CGO-free so it cross-compiles trivially and
  covers headless/scripted use on all three OSes with no GUI deps.

## Risks / open questions

- **EDS.xml fidelity** — the grid's usefulness hinges on correct metadata
  parsing (Phase 2); if `EDS.xml` is incomplete, fall back to device-reported
  `type`/`cat` from `/api/prop`.
- **Device-only verification** — most phases need a real charger (or captured
  fixtures) to confirm wire behaviour; `selftest` only covers crypto/format.
- **Satellite DLL behaviour** — `ACELogoConvertor`/`ACEISAHConnection` are only
  available decompiled; their exact algorithms must be re-derived (Phases 8, and
  ISAH only if required).
- **Fidelity vs. pragmatism** — this reorders/streamlines the installer's UI
  rather than cloning it pixel-for-pixel; flag if a 1:1 clone is required.

## Status

- [x] Phase 0 — plan + Fyne dependency added
- [ ] Phase 1 — core flow *(in progress)*
- [ ] Phases 2–9
