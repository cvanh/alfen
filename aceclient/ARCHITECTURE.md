# aceclient architecture

Contributor orientation for the Go module. For scope, build, and the CLI surface see
[`README.md`](README.md); for the C#→Go porting conventions see the
`porting-from-decompiled` skill.

## Entry point

`cmd/aceclient/main.go` is a hand-rolled subcommand dispatcher (no cobra) with a
minimal `--k v` / `--k=v` long-opt parser (`parseConn`). It is the **only `main`
package** — the CLI is the sole entry point today. Subcommands group into:

- **Firmware containers** — `fwi-info`, `fwi-build`, `tvf-info`
- **Charger network** — `discover`, `scn-decode`, `login`, `prop-get`, `prop-set`, `firmware-upload`
- **Key research** — `keytest`, `selftest`

## Packages (`internal/`), by concern

- **Containers & crypto primitives** — `fwi` (`.fwi` header / CRC / display unwrap),
  `tvf` (`.tfw`/TVF manifest / cert / payload), `fwucreator` (the installer's FWU
  builder port, incl. the large generated `objects_gen.go` / `enums_gen.go`),
  `alfencrc` (the reflected CRC used throughout — one implementation, many call sites),
  `keytest` (parallel candidate-key sweep with entropy / display / magic oracles).
- **Device channels** — `scn` (AES-128-CBC UDP telemetry decode, static fleet key,
  port 36549, listen-only), `api` (HTTPS object dictionary: token login,
  `GET/POST /api/prop`, multipart firmware upload), `discovery` (device / LAN
  discovery + DNS-SD).
- **Object dictionary & config** — `eds` (EDS.xml parse, CANopen value types, UI type
  mapping), `isah` (ISAH object model + a Newtonsoft-JSON-compatible decoder, since the
  installer persists .NET-serialized data), `config`, `settings`, `backoffice`,
  `presets`, `i18n`, `logo`.
- **GUI (not yet wired)** — `internal/ui` + `internal/ui/core`. `core` is the pure,
  testable half (device / protocol logic, no fyne imports); `ui` is the thin fyne view
  layer with `NewApp()`. No `main` launches the GUI yet. The build stays CGO-free
  because this code is never linked into `cmd/aceclient`.

## Test fixtures

Many tests read real reference files from `../../../firmware/...` (decompiled sources,
`msi_work/files3`, presets) and **`t.Skip` when the fixture is missing**. A green
`go test ./...` with skips is expected on a checkout where the `firmware/` tree is not
populated. When validating format / CRC / parse logic, make sure the relevant
`firmware/` fixtures are present — otherwise that coverage is silently skipped.
