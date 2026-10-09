// Package presets ports the ACE Service Installer's property settings files
// and property presets:
//
//   - PropertyStorage (ACEServiceInstaller/ICUServiceInstaller/PropertyStorage.cs):
//     "Save Settings As..." writes every writable, non-forbidden property of
//     the device to an encrypted .exml "Settings" XML document
//     (BuildSettingsXML / SaveProperties, byte-compatible with
//     XDocument.ToString() + ICUSettings.EncryptDecrypt); "Load Settings..."
//     and "Load Preset..." assign a file's properties to the device's property
//     dictionary (LoadProperties) after the XMLVersion and model checks.
//   - DlgPresets (DlgPresets.cs): the preset picker lists *.xml / *.iip
//     Settings files from the program and local preset folders
//     (LoadPresetsFromFolder / FillPresetList).
//   - AppProperties preset folder names and the legacy Presets folder removal
//     of MainWindow (Folders, RemoveOldPresetFolder).
//   - The bits of ICUNetwork the codec depends on: ICUProperty value typing,
//     DeviceValue/IsChanged and ICUPropertyDictionary order (Dictionary), and
//     ICUDeviceModel with ICUDeviceModelExtension.From/ToString (DeviceModel).
//
// Flow, as MainWindow does it: build a Dictionary from a fresh device read,
// LoadProperties(file, ...) -> n changed; when n > 0 show
// SettingsLoadedMessage/SettingsLoadedDetail(n); on "Save" write
// dict.ChangedProperties() (StoreChangedProperties does the 15-per-request
// store and commit).
//
// What the preset folders hold in practice (firmware/ fixtures): TCPPresets
// and RTUPresets contain Modbus energy-meter register maps (*.json, loaded by
// DlgModbusRegisterMap into ICUModbusRegmap, not by this package);
// BackofficePresets contains backoffice configuration update files, *.fwi
// (/.fwu) for pre-AHP chargers and *.tcf (/.tfw) for AHP chargers
// (ICULanDevice.getUpdateFileTypes), offered by PanelConnectivity; the legacy
// Presets folder held the older Modbus *.json files. None of them contain
// *.xml/*.iip, so the DlgPresets list is normally filled only from the
// program-files Presets folder.
package presets
