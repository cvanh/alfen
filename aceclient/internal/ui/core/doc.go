// Package core holds the UI-independent logic behind the Fyne views in
// internal/ui. It is the "pure" half of the installer's panels and dialogs,
// extracted from the Xwt-coupled C# so it can be unit-tested and reused
// without a GUI. It never imports Fyne.
//
// What it ports (all from the decompiled ACE Service Installer, ilspycmd 11.1):
//
//   - ICUNetwork.ICULanDevice (ACENetwork/ICUNetwork/ICULanDevice.cs): the
//     per-device session — Protocol/IsHTTPS, LoginRequest/ClassicLogin,
//     HandleUnsuccessfulLoginRequest, Logout, ExecuteWebRequest's retry and
//     HandleUnsuccessfulRequest policy, UpdateCategories/RequestCategories,
//     UpdateProperties, StoreProperties, FirmwareVersionNumber, isAHP,
//     IsUniquePasswordRequired, GetNFCVersion and the HTTP part of
//     StartUpload.
//   - ICUNetwork.ICUProperty / ICUPropertyDictionary / ICUDevice
//     (ACENetwork/ICUNetwork/ICUProperty.cs, ICUPropertyDictionary.cs,
//     ICUDevice.cs): the property cache, SetValue's per-SDT conversion,
//     SetInitialValue/CommitChange/Rollback and the GetProperty* accessors.
//   - MainWindow (ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs): the
//     device list ordering/filter (RefreshDeviceList, DeviceMatchesFilter),
//     SaveAllChanges, ValidateCSConfiguration and the tamper acknowledge.
//   - PanelAllProperties, PanelInformation, DlgDeviceLogin, DlgManualIP and
//     DlgUpload (ACEServiceInstaller/ICUServiceInstaller/*.cs): grouping,
//     editor selection and search; the identity rows, ParseRevision and the
//     identity/SSA save checks; the login flow texts; IP validation; upload
//     pre-flight texts, filters and progression texts.
//   - UIPropertyNumber/UIPropertyString/UIPropertyCheckbox/
//     UIPropertyReadOnlyString: the per-SDT input rules (spin ranges, max
//     length, checkbox value mapping, DateTime rendering).
//
// Device I/O goes exclusively through internal/api; firmware inspection uses
// internal/fwi and internal/tvf; discovery data comes from internal/scn.
package core
