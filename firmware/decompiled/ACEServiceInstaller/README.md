# ACEServiceInstaller.exe — full decompiled project

Full `ilspycmd -p` (ILSpy 11.1) decompile of the main GUI executable
`ACEServiceInstaller.exe` from ACE Service Installer v4.4.1_434
(`firmware/msi_work/files3/`). This is the application itself; the satellite
assemblies (`ACEFWUCreator`, `ACENetwork`, `ACESettings`) are decompiled in the
parent folder.

Decompiler output — identifiers and structure are faithful, but it is not the
original source and won't compile verbatim. 122 `.cs` files. Extracted artwork
PNG resources were dropped; the `.resx` string table is kept.

## Layout

| Path | Contents |
|------|----------|
| `ACEServiceInstaller/` | `MainClass` (entrypoint), `AlfenDomain`/`SessionInitializer`/`StackTraceEnricher` (Serilog + App Insights telemetry) |
| `ICU_Diag/EncryptDecrypt.cs` | Rijndael-256 / `PasswordDeriveBytes` helper, hardcoded passphrase/salt/IV |
| `ICUServiceInstaller/` | The whole Xwt/WPF UI: `App`, `MainWindow(Base)`, all `Dlg*` dialogs, all `Panel*` tabs, `UIProperty*` widgets, `Tooltips`, `PropertyStorage` |
| `ICUServiceInstaller.Dialogs/` | `DlgScanWifiNetworks` |
| `ICUServiceInstaller.Enums/` | Wifi security/signal enums + `EnumExtensions` |
| `ICUServiceInstaller.Utils/` | `DisplayLanguage(Helper)`, `Flags`, `LoggerExtensions` |
| `ICUServiceInstaller.UI/`, `.Wifi/` | `UIWifiProfile`, `ScannedNetwork`, `WifiResult` |
| `ICUServiceInstaller.Properties/` | generated `Resources` / `Settings` |

## Landmarks

- **Entrypoint:** `ACEServiceInstaller/MainClass.cs` → `Main` — single-instance
  mutex (`multimode` arg bypasses it), Serilog + Application Insights wiring,
  then `App.Run(ToolkitType.Wpf, ...)`. `-large` arg selects `aoLarge`.
- **Telemetry conn string:** `MainClass.GetAppInsightsConnectionString` — reads
  `Settings.Default.AppInsightsConnStrEncrypted`, else a hardcoded fallback
  ciphertext, decrypted via the `ICUSettings.EncryptDecrypt` keyed on the app
  name `"ACE Service Installer 4.0"`.
- **Config crypto:** `ICU_Diag/EncryptDecrypt.cs` — passphrase `Pas5pR@sE`,
  salt `s@1tVaLue`, IV `@1B2c3D4e5F6g7H8`, SHA1, 2 iterations, 256-bit
  (same Rijndael family as the documented `ACESettings` helper).
- **Main UI flow:** `ICUServiceInstaller/App.cs` → `MainWindow.cs` (58 K) and
  the `Panel*` tabs; device I/O goes through the `ACENetwork` assembly.

## Provenance

`ilspycmd -p -o <dir> ACEServiceInstaller.exe` (ILSpy 11.1.0.9782), from the
MSI-extracted payload in `firmware/msi_work/files3/`.
