package core

import (
	"encoding/binary"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"alfen/aceclient/internal/api"
	"alfen/aceclient/internal/fwi"
	"alfen/aceclient/internal/tvf"
)

// DlgUpload texts.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/DlgUpload.cs:74-167
const (
	UploadTitle              = "Upload new firmware"
	UploadFilePlaceholder    = "Enter a valid firmware file"
	UploadCurrentVersion     = "Current firmware version:"
	UploadSelectFirmware     = "Select firmware:"
	UploadFileLocation       = "Firmware file location:"
	UploadButton             = "Start upload"
	UploadProgressInitial    = "Uploading firmware to the charger..."
	UploadNg9xxUpgradeNotice = "Please update to NG9xx 6.6.2 before updating to NG9xx 7.x.x to ensure a smooth update"
	UploadSuccess            = "Firmware updated successfully!"
	UploadErrorPrimary       = "Error during firmware upload!"
	UploadBrowseTitle        = "Select a firmware file"
	// UploadCommunicationError is StartUpload's result for a non-200 reply.
	UploadCommunicationError = "Couldn't communicate with the device, please reboot the device."
)

// FileFilter is one OpenFileDialog filter of DlgUpload.OnBtnBrowse.
type FileFilter struct {
	Name     string
	Patterns []string // "*.fwi", …
}

// Extensions returns the patterns as ".ext" (no wildcard handling needed).
func (f FileFilter) Extensions() []string {
	var out []string
	for _, p := range f.Patterns {
		if strings.HasPrefix(p, "*.") && p != "*.*" {
			out = append(out, p[1:])
		}
	}
	return out
}

// FirmwareFileFilters ports DlgUpload.OnBtnBrowse's filters, in order.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/DlgUpload.cs:457-492
var FirmwareFileFilters = []FileFilter{
	{Name: "All firmware files", Patterns: []string{"*.fwi", "*.fwu", "*.tfw", "*.tcf", "*.tvf"}},
	{Name: "NG9xx firmware files", Patterns: []string{"*.fwi", "*.fwu"}},
	{Name: "AHWP firmware files", Patterns: []string{"*.tfw", "*.tcf", "*.tvf"}},
	{Name: "All files", Patterns: []string{"*.*"}},
}

// ActiveFirmwareFilter ports OnBtnBrowse's ActiveFilter choice. The C#
// indexes Filters[1] (no device), [3] (AHP) and [2] (NG9xx), i.e. off by one
// from the intent; that index choice is mirrored.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/DlgUpload.cs:457-492
func ActiveFirmwareFilter(haveDevice, isAHP bool) FileFilter {
	switch {
	case !haveDevice:
		return FirmwareFileFilters[1]
	case isAHP:
		return FirmwareFileFilters[3]
	default:
		return FirmwareFileFilters[2]
	}
}

// UpdateFileTypes ports ICULanDevice.getUpdateFileTypes.
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICULanDevice.cs:372-382
func UpdateFileTypes(isAHP bool) []string {
	if isAHP {
		return []string{".tfw", ".tcf"}
	}
	return []string{".fwi", ".fwu"}
}

// UploadHeader ports DlgUpload.ChangeDevice's header label.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/DlgUpload.cs:169-181
func UploadHeader(identification, serial8273 string) string {
	return fmt.Sprintf("Upload firmware to device '%s' (serial number: %s)", identification, serial8273)
}

// UploadConfirmQuestion ports OnBtnUploadClicked's confirmation.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/DlgUpload.cs:267-321
func UploadConfirmQuestion(identification, fileName string) string {
	return "Are you sure you want to update the firmware of '" + identification + "' to '" + filepath.Base(fileName) + "'"
}

// ShowNg9xxUpgradeWarning ports DlgUpload.ShowNg9xxUpgradeWarning.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/DlgUpload.cs:323-330
func ShowNg9xxUpgradeWarning(isAHP bool, fw Version) bool {
	return !isAHP && fw.Less(NewVersion(6, 6))
}

var fileVersionRe = regexp.MustCompile(`[_ ](\d+.\d+.\d+)[_.-]`)

// VersionFromFileName ports OnBtnUploadClicked's
// Regex.Match(Path.GetFileName(name.ToLowerInvariant()), "[_ ](\d+.\d+.\d+)[_.-]")
// + Version.Parse. matched mirrors match.Success (it also decides whether the
// C# tracks progress); ok is false when Version.Parse would throw.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/DlgUpload.cs:267-321
func VersionFromFileName(name string) (v Version, matched, ok bool) {
	m := fileVersionRe.FindStringSubmatch(filepath.Base(strings.ToLower(name)))
	if m == nil {
		return Version{}, false, false
	}
	v, ok = ParseVersion(m[1])
	return v, true, ok
}

// NeedsNewPassword ports the DlgDeviceNewPassword trigger: an NG9xx device on
// firmware < 5 receiving a file whose name says >= 5.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/DlgUpload.cs:267-321
func NeedsNewPassword(isAHP bool, current Version, fileName string) bool {
	v, _, ok := VersionFromFileName(fileName)
	return ok && !isAHP && current.Major < 5 && v.Major >= 5
}

// ProgressionText ports DlgUpload.UpdateProgressionText (NG9 and AHP
// variants); elapsed is the AHP install stopwatch.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/DlgUpload.cs:400-455
func ProgressionText(progress int, isAHP bool, elapsed time.Duration) string {
	if isAHP {
		switch {
		case progress < 4:
			return "Uploading firmware to the charger..."
		case progress < 8:
			return "Firmware uploaded, preparing update..."
		case progress <= 97:
			return fmt.Sprintf("Installing firmware (can take up to 12 minutes) %d minutes passed.", int(elapsed/time.Minute))
		case progress <= 100:
			return "Rebooting station and finishing up firmware update."
		}
		return ""
	}
	switch {
	case progress < 50:
		return "Uploading firmware to the charger, can take 1 to 3 minutes."
	case progress <= 97:
		return "Installing firmware (charger will reboot)."
	case progress <= 99:
		return "Finishing up firmware update..."
	case progress <= 100:
		return "Almost done"
	}
	return ""
}

// FirmwareFamily is the device family a file's extension targets.
type FirmwareFamily int

const (
	FamilyUnknown FirmwareFamily = iota
	FamilyNG9xx                  // *.fwi, *.fwu
	FamilyAHWP                   // *.tfw, *.tcf, *.tvf
)

func (f FirmwareFamily) String() string {
	switch f {
	case FamilyNG9xx:
		return "NG9xx"
	case FamilyAHWP:
		return "AHWP"
	}
	return "unknown"
}

// FamilyForExtension maps an extension to DlgUpload's filter groups.
func FamilyForExtension(ext string) FirmwareFamily {
	switch strings.ToLower(ext) {
	case ".fwi", ".fwu":
		return FamilyNG9xx
	case ".tfw", ".tcf", ".tvf":
		return FamilyAHWP
	}
	return FamilyUnknown
}

// FirmwareClassification is the pre-flight report for a firmware file. The
// installer itself uploads files unexamined; this report only reads what
// internal/fwi and internal/tvf can confirm and never blocks an upload.
type FirmwareClassification struct {
	FileName    string
	Size        int
	Extension   string
	Family      FirmwareFamily
	NameVersion string // from VersionFromFileName ("" when none)

	// .fwi container (fwi.Parse) — set when the content has the 0xA1FE magic
	// and a 160-byte header length, whatever the extension.
	FWI             *fwi.Header
	FWICRCOK        bool
	FWIStoredCRC    uint32
	FWIComputedCRC  uint32
	DisplayResource bool  // LooksLikeDisplayResource
	DisplayUnwrapOK bool  // UnwrapDisplayPayload succeeded (known display key)
	DisplayUnwrapN  int   // bytes recovered by the unwrap
	DisplayUnwrapEr error // unwrap error, if attempted and failed

	// TVF-style container (u16 version, u16 header length, u32 magic) — the
	// TvfHeader layout; LocalTVF when the magic is tvf.MagicTVF.
	TVFHeaderLength int
	TVFMagic        uint32
	LocalTVF        bool
	TVF             *tvf.SniffResult

	Warnings []string
}

// ClassifyFirmware inspects a firmware file for the upload pre-flight. deviceKnown /
// isAHP add the getUpdateFileTypes mismatch warning (DlgUpload.FillList only
// lists those extensions for the current device).
func ClassifyFirmware(name string, data []byte, deviceKnown, isAHP bool) FirmwareClassification {
	c := FirmwareClassification{
		FileName:  filepath.Base(name),
		Size:      len(data),
		Extension: strings.ToLower(filepath.Ext(name)),
	}
	c.Family = FamilyForExtension(c.Extension)
	if v, matched, ok := VersionFromFileName(name); matched && ok {
		c.NameVersion = v.String()
	}
	if len(data) == 0 {
		c.Warnings = append(c.Warnings, "file is empty")
		return c
	}
	looksFWI := len(data) >= fwi.HeaderLen &&
		binary.LittleEndian.Uint32(data[0x04:]) == fwi.HeaderLen &&
		binary.LittleEndian.Uint16(data[0x0A:]) == fwi.MagicA1FE
	looksTVF := len(data) >= 8 && binary.LittleEndian.Uint16(data[0:]) == 1 &&
		int(binary.LittleEndian.Uint16(data[2:])) > 8 && int(binary.LittleEndian.Uint16(data[2:])) <= len(data)
	if looksFWI {
		if h, err := fwi.Parse(data); err == nil {
			c.FWI = h
			c.FWICRCOK, c.FWIStoredCRC, c.FWIComputedCRC = fwi.VerifyCRC(data)
			if !c.FWICRCOK {
				c.Warnings = append(c.Warnings, "header CRC mismatch")
			}
			if h.LooksLikeDisplayResource() {
				c.DisplayResource = true
				obj, err := fwi.UnwrapDisplayPayload(data)
				if err != nil {
					c.DisplayUnwrapEr = err
				} else {
					c.DisplayUnwrapOK = true
					c.DisplayUnwrapN = len(obj)
				}
			}
		}
	} else if looksTVF {
		c.TVFHeaderLength = int(binary.LittleEndian.Uint16(data[2:]))
		c.TVFMagic = binary.LittleEndian.Uint32(data[4:])
		c.LocalTVF = c.TVFMagic == tvf.MagicTVF
		r := tvf.Sniff(data)
		c.TVF = &r
	}
	switch {
	case c.Family == FamilyUnknown:
		c.Warnings = append(c.Warnings, "extension is not one of *.fwi, *.fwu, *.tfw, *.tcf, *.tvf")
	case c.Family == FamilyNG9xx && c.FWI == nil:
		c.Warnings = append(c.Warnings, "extension says NG9xx (.fwi/.fwu) but the content has no .fwi header")
	case c.Family == FamilyAHWP && c.FWI != nil:
		c.Warnings = append(c.Warnings, "extension says AHWP (.tfw/.tcf/.tvf) but the content is a .fwi container")
	case c.Family == FamilyAHWP && c.TVFHeaderLength == 0:
		c.Warnings = append(c.Warnings, "extension says AHWP but the content has no TVF-style header")
	}
	if deviceKnown {
		allowed := UpdateFileTypes(isAHP)
		ok := false
		for _, e := range allowed {
			if e == c.Extension {
				ok = true
			}
		}
		if !ok {
			c.Warnings = append(c.Warnings, fmt.Sprintf("the connected device takes %s files", strings.Join(allowed, "/")))
		}
	}
	return c
}

// Text renders the classification for the pre-flight panel.
func (c FirmwareClassification) Text() string {
	var b strings.Builder
	fmt.Fprintf(&b, "File: %s (%d bytes)\n", c.FileName, c.Size)
	fmt.Fprintf(&b, "Type: %s firmware file (%s)\n", c.Family, c.Extension)
	if c.NameVersion != "" {
		fmt.Fprintf(&b, "Version in file name: %s\n", c.NameVersion)
	}
	switch {
	case c.FWI != nil:
		h := c.FWI
		fmt.Fprintf(&b, "Container: .fwi, header type 0x%02X, payload %d bytes\n", h.Type, h.PayloadLength)
		fmt.Fprintf(&b, "Header CRC: stored %08X, computed %08X (%s)\n", c.FWIStoredCRC, c.FWIComputedCRC, okText(c.FWICRCOK))
		if c.DisplayResource {
			if c.DisplayUnwrapOK {
				fmt.Fprintf(&b, "Variant: display resource (installer-built, known key); unwrap OK, %d bytes\n", c.DisplayUnwrapN)
			} else {
				fmt.Fprintf(&b, "Variant: display resource (installer-built, known key); unwrap failed: %v\n", c.DisplayUnwrapEr)
			}
		} else {
			fmt.Fprintf(&b, "Variant: controller firmware (device-held key), wrapped key block: %s, section map: %s\n",
				yesNoText(h.HasWrappedKey()), yesNoText(h.HasSectionMap()))
		}
	case c.TVFHeaderLength > 0:
		if c.LocalTVF {
			fmt.Fprintf(&b, "Container: TVF, header %d bytes, magic 0x%08X (installer-built resource TVF)\n", c.TVFHeaderLength, c.TVFMagic)
		} else {
			fmt.Fprintf(&b, "Container: TVF, header %d bytes, magic 0x%08X (Alfen release)\n", c.TVFHeaderLength, c.TVFMagic)
		}
		if c.TVF != nil {
			if c.TVF.Manifest != "" {
				fmt.Fprintf(&b, "Manifest @%d: %q\n", c.TVF.ManifestOffset, c.TVF.Manifest)
			}
			fmt.Fprintf(&b, "Signing certificate: %s\n", yesNoText(c.TVF.CertPresent))
		}
	default:
		b.WriteString("Container: not recognised\n")
	}
	b.WriteString("The file is uploaded unchanged; the charger decrypts and verifies it.\n")
	for _, w := range c.Warnings {
		fmt.Fprintf(&b, "Warning: %s\n", w)
	}
	return strings.TrimRight(b.String(), "\n")
}

func okText(ok bool) string {
	if ok {
		return "OK"
	}
	return "MISMATCH"
}

func yesNoText(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// UploadFirmware ports the HTTP part of ICULanDevice.StartUpload: up to three
// POST /api/firmware attempts (multipart, 15 min budget, one try each). On 401
// or 403 the device is logged in again (relogin returns the login status); a
// renewed 401 retries after 1 s, any other login result ends the loop as if
// the upload went through (as the C#). A non-200 reply returns
// UploadCommunicationError. The returned string is LastUploadError ("" on
// success).
//
// Not ported (needs the firmware state machine and other endpoints): the
// pre-upload SetDate/GetFirmwareUploadStatus checks, progress tracking, the
// reboot/re-login polling, "forcefirmwarepermanent" and the post-upgrade
// password change.
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICULanDevice.cs:2090-2156 (StartUpload), 2378-2400
func (s *Session) UploadFirmware(data []byte, relogin func() int) string {
	s.MarkUploading(true)
	defer s.MarkUploading(false)
	for num := 3; num > 0; num-- {
		var resp api.Response
		var err error
		s.io.Lock()
		_ = s.withTimeout(FirmwareUploadTimeout, func() error { resp, err = s.Client.UploadFirmware(data); return nil })
		s.io.Unlock()
		status := resp.StatusCode
		if err != nil {
			status, _ = StatusFromError(err)
		}
		if status == StatusUnauthorized || status == StatusForbidden {
			if relogin != nil && relogin() == StatusUnauthorized {
				time.Sleep(uploadRetryDelay)
				continue
			}
		} else if status != StatusOK {
			return UploadCommunicationError
		}
		break
	}
	return ""
}

// uploadRetryDelay is StartUpload's Thread.Sleep(1000); a var for tests.
var uploadRetryDelay = time.Second
