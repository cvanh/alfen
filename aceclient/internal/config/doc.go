// Package config ports the installer's user/rights configuration from the
// ACESettings assembly (namespace ICUSettings, firmware/decompiled/ACESettings)
// plus the parts of ACEServiceInstaller.exe that consume it:
//
//   - EncryptDecrypt.cs (ICUSettings) and ICU_Diag/EncryptDecrypt.cs (exe):
//     PasswordDeriveBytes(SHA1, 2 iterations) -> 256-bit key, RijndaelManaged
//     with its default 128-bit block (i.e. AES-256-CBC, PKCS7), fixed IV,
//     Base64 text. See encrypt.go.
//   - ICUConfig.cs: ReadInstallerSettings / WriteInstallerSettings for
//     InstallerConfig*.dat in every ICUEncryptionType (none, base64, Rijndael,
//     Rijndael + hashed users), the hand-built JSON layout (byte-exact), the
//     manual/backoffice feature list, default groups and lookups. See config.go.
//   - ICUBaseObject, ICUFeature, ICUFeatureRight, ICUFeatureType, ICURights,
//     ICUGroup, ICUUser (incl. the "NN:SALTHEX:b64" hashed-password scheme),
//     ICUFirmware, ICUEncryptionType. See enums.go, model.go, user.go.
//   - DlgLogon.cs: installer-user logon against the config (hashed user name
//     lookup, password verification, per-PC "local passwords", decryption of
//     the group HTTP credentials / ISAH data) and the local password change.
//     See logon.go.
//   - AppProperties.cs: config file names and InitializeSettings' file
//     selection (including the partial state a failed read leaves in the
//     static ICUConfig). See appprops.go. The other AppProperties folders and
//     the user-scoped Properties/Settings.cs store (LastUserName,
//     LocalPasswords) belong to internal/settings; Logon and
//     ChangeLocalPassword take and return the LocalPasswords list as a plain
//     []string.
//   - MainWindow.FillPanelList / PanelBase.IsVisible / PanelBase.SetUser /
//     UIPropertyBase.CheckAccessRights: how feature rights gate pages and
//     properties. See rights.go.
//
// ICUBackOffice / ICUPMBackOffice are owned by internal/backoffice. The
// "Backoffices" and "PMBackOffices" arrays are kept verbatim as
// json.RawMessage. This package ports only the parts of their dynamic
// constructors that can throw, so the same files load or fail as in the C#,
// plus each backoffice's Title/Groups, from which ICUConfig derives the "BO_*"
// features. See backoffices.go.
//
// The C# model's property setters trim (and ICUFeature.ID upper-cases) what an
// editor assigns. The exported fields here are plain, so editors must assign
// through the Set* methods in model.go to keep that normalisation.
//
// The UI-only plumbing of the C# model (INotifyPropertyChanged, Dirty flags,
// Commit/Rollback, XML import/export used by the admin settings editor, FTP)
// is intentionally not ported. Nothing in this package imports Fyne.
package config
