// Package backoffice ports the backoffice ("back end" / OCPP central system)
// model and the backoffice-preset device logic of the ACE Service Installer
// v4.4.1.
//
// # Config model (ACESettings, namespace ICUSettings)
//
//   - ICUBackOffice.cs            -> BackOffice
//   - ICUBackOfficeProperty.cs    -> Property
//   - ICUBackOfficePropetyTypes.cs -> PropertyType (Hidden/Normal/Extended)
//   - ICUPMBackOffice.cs          -> PMBackOffice
//   - ICUPMProperty.cs            -> PMProperty
//   - ICUConfig.cs (backoffice parts only): the "Backoffices"/"PMBackOffices"
//     arrays of the InstallerConfig JSON (ParseBackoffices,
//     ParsePMBackoffices, MarshalBackoffices, MarshalPMBackoffices),
//     AddBackoffice/AddPMBackoffice (CopyBackOffice/CopyPMBackOffice),
//     UpdatePMBackOffices (DerivePMBackOffices), WritePMBackOfficeSettingsXML
//     and the backoffice feature IDs (FeatureID, Features, CompanyFeatureIDs).
//
// The JSON is hand-built by the C# (string.Format, no escaping); the writers
// here reproduce it byte-for-byte. Keys the v4.4.1 model does not know (the
// newer "OD_commProfileN_*" entries present in firmware/InstallerConfig*.json)
// are dropped on load exactly as ICUBackOffice.SetProperty drops them.
//
// IMPORTANT: in v4.4.1 nothing in ACEServiceInstaller.exe applies an
// ICUBackOffice/ICUPMBackOffice to a charger. The exe only uses the config
// model for user-rights features (BO_<TITLE>). What the installer actually
// applies to a device is a backoffice PRESET FILE.
//
// # Backoffice presets (ACEServiceInstaller PanelConnectivity, ACENetwork)
//
// %AppData%\ACE Service Installer\BackofficePresets\ holds one file per
// backoffice, synced from the FTP folder "BackofficePresets" (MainWindow ->
// UpdateManager.CheckAdditionalFTPFiles; new files are stored LOWERCASED).
// PanelConnectivity lists the files whose extension matches
// ICULanDevice.getUpdateFileTypes (".fwi"/".fwu" for NG9xx, ".tfw"/".tcf" for
// AHP), strips the "-a"/"-b"/"-c" encryption-key suffix for display, and on
// Save:
//
//  1. clears every backoffice property (ClearAllBackOfficeSettings) and
//     stores that list immediately,
//  2. uploads the preset file VERBATIM through /api/firmware (the same path as
//     a firmware update: ICULanDevice.UploadFirmware),
//  3. on success writes 8310 ("BackOffice short name") = "<preset>[,<meter>]".
//
// "_MAN" (manual) and "_SA" (standalone) are pseudo-presets that only write
// 8310/8321 (and, for "_SA", clear everything). PlanSave/Execute port that
// sequence; the rest of PanelConnectivity's backoffice section is ported as
// pure helpers (SelectionFromDevice, SelectionChanged, Controls, ...).
//
// # .tcf files
//
// A .tcf ("AHWP firmware files" filter in DlgUpload) is the AHP backoffice
// configuration preset. The C# never parses it: it is classified purely by
// extension (UpdateFileTypes) and uploaded as an opaque blob. For reference
// only (asserted by tcf_test.go against firmware/BackofficePresets, NOT
// implemented as a parser): the files use the same header layout as
// ACEFWUCreator.TvfHeader (see internal/tvf.BuildHeader) — u16 version 1,
// u16 header length (= 1350 + len(manifestVersion) + len(description)),
// u32 magic 0x84FC8EB7 (TvfHeader uses 0xB8ED7AD5), u32 data length, ...,
// length-prefixed empty manifest version, length-prefixed description (the
// preset name), a u16-length-prefixed 256-byte signature area (DER ECDSA),
// a u16-length-prefixed 1024-byte certificate area and a CRC-32 (IEEE) over
// the header; total file size = header length + data length.
package backoffice
