package fwucreator

import (
	"context"
	"errors"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"

	"alfen/aceclient/internal/api"
	"alfen/aceclient/internal/logo"
)

// DlgUploadResources / AppProperties / ICULanDevice constants.
const (
	MaxColors          = 128 // DlgUploadResources.s_nMaxColors
	MaxLogoHeight      = 160 // s_nMaxLogoHeight (also ICULanDevice)
	MaxLogoHeightLarge = 350 // s_nMaxLogoHeightLarge
	MaxLogoWidth       = 320 // s_nMaxLogoWidth (also ICULanDevice)
	MaxLogoWidthLarge  = 800 // s_nMaxLogoWidthLarge
	DefaultImageMargin = 8   // m_spbMargin.Value = 8.0 in Initialize

	UILanguagesFolder = "UILanguages"           // AppProperties.UILanguagesFolder
	AppName           = "ACE Service Installer" // AppProperties.AppName (TVF "created-by")
	DefaultLogoFile   = "logo_alfen.png"        // pre-filled when it exists in the working directory
	FeatureCreateFWU  = "FEATURE_CREATEFWU"     // right that shows "Create Image Update File..."
)

// Texts of DlgUploadResources.
const (
	DialogTitle            = "Upload Logo"
	TitleCreateFile        = "Create logo update file"
	SaveDialogTitle        = "FWU file location"
	BrowseDialogTitle      = "Select an image"
	FilenamePlaceholder    = "Enter a valid firmware file"
	ButtonCreateFWU        = "Create Image Update File..."
	ButtonUpload           = "Start upload"
	MsgUploadSuccess       = "Image uploaded successfully!"
	MsgAskSmallLargeScreen = "Do you want to generate an resource file for an small (Eve mini) or a large display (EVE2) display?\nSelect Yes for a small display."
)

// Property ids behind ICULanDevice.MaxLogoWidth/MaxLogoHeight.
const (
	propDisplayID     = 0x3260 // 12896; GetProperty(3301377u) = 0x3260/1 must exist
	propDisplayPresub = 1
	propDisplayWidth  = 3
	propDisplayHeight = 4
	propSerialNumber  = 0x2051 // 8273, GetPropertyString(8273, 0, 0)
)

// IsAHPModel ports ICULanDevice.isAHP (ACENetwork/ICUNetwork/ICULanDevice.cs:334-344): Model starts with "AHWP" or "AHP",
// ignoring case.
func IsAHPModel(model string) bool {
	m := strings.ToUpper(model)
	return strings.HasPrefix(m, "AHWP") || strings.HasPrefix(m, "AHP")
}

// MaxLogoSize ports ICULanDevice.MaxLogoWidth / MaxLogoHeight
// (ICULanDevice.cs:456-498): when property
// 0x3260/1 exists, 0x3260/3 x 0x3260/4 (only if both are > 0, else 0 x 0);
// otherwise 320 x 160 for the old models with a display (Eve Mini,
// NG910-60014, NG910-60034), else 0 x 0.
func MaxLogoSize(prop PropertyLookup, isOldModelWithDisplay bool) (width, height int) {
	if prop != nil {
		if _, ok := prop(propDisplayID, propDisplayPresub); ok {
			w, wok := prop(propDisplayID, propDisplayWidth)
			h, hok := prop(propDisplayID, propDisplayHeight)
			pw, ph := 0, 0
			if wok {
				pw = w.Int(0)
			}
			if hok {
				ph = h.Int(0)
			}
			if pw > 0 && ph > 0 {
				return pw, ph
			}
			return 0, 0
		}
	}
	if isOldModelWithDisplay {
		return MaxLogoWidth, MaxLogoHeight
	}
	return 0, 0
}

// DeviceSerialNumber is the dialog header's lanDev.GetPropertyString(8273, 0, 0)
// (property 0x2051/0); empty when the property is not cached. (EDS option
// titles, which GetPropertyString would prefer, do not apply to this id.)
func DeviceSerialNumber(prop PropertyLookup) string {
	if prop == nil {
		return ""
	}
	p, ok := prop(propSerialNumber, 0)
	if !ok {
		return ""
	}
	return p.ToString()
}

// HasDisplay ports ICULanDevice.HasDisplay (ICULanDevice.cs:444-454).
func HasDisplay(maxLogoWidth int, isOldModelWithDisplay, isDualPG bool) bool {
	if maxLogoWidth <= 0 && !isOldModelWithDisplay {
		return isDualPG
	}
	return true
}

// HasLargeDisplay ports ICULanDevice.HasLargeDisplay (ICULanDevice.cs:442;
// MaxLogoWidth > 320).
func HasLargeDisplay(maxLogoWidth int) bool { return maxLogoWidth > MaxLogoWidth }

// UploadDevice is what DlgUploadResources reads from its ICULanDevice.
type UploadDevice struct {
	Client         *api.Client
	Identification string // ICULanDevice.Identification
	SerialNumber   string // GetPropertyString(8273, 0, 0)
	IsAHP          bool   // ICULanDevice.isAHP (IsAHPModel)
	// FirmwareVersion is FirmwareVersionNumber as System.Version components
	// (major, minor[, build[, revision]]).
	FirmwareVersion []int
	MaxLogoWidth    int  // MaxLogoSize
	MaxLogoHeight   int  // MaxLogoSize
	HasDisplay      bool // HasDisplay
	// Property is the device's property cache (used by UploadResource/SetDate).
	Property PropertyLookup
}

// versionLess compares System.Version values; missing components are -1.
func versionLess(a, b []int) bool {
	for i := 0; i < 4; i++ {
		x, y := -1, -1
		if i < len(a) {
			x = a[i]
		}
		if i < len(b) {
			y = b[i]
		}
		if x != y {
			return x < y
		}
	}
	return false
}

// UploadResourceWorkerData ports ICUServiceInstaller.UploadResourceWorkerData
// (ACEServiceInstaller/ICUServiceInstaller/UploadResourceWorkerData.cs; the
// BackgroundWorker argument); Worker becomes the Progress callback.
type UploadResourceWorkerData struct {
	CurrentDevice   *UploadDevice
	Image           *image.Paletted
	Filename        string // set for AHP only
	SpecialLanguage string // never set by the installer
	Margin          int    // set for AHP only
	Progress        func(percent int)
}

// Answer is the Yes/No/Cancel result of MessageDialog.AskQuestion.
type Answer int

// AskQuestion answers.
const (
	AnswerYes Answer = iota
	AnswerNo
	AnswerCancel
)

// FileFilter is an Xwt FileDialogFilter.
type FileFilter struct {
	Name    string
	Pattern string
}

// ImageFileFilter ports the OnBtnBrowse filter (DlgUploadResources.cs:409-424): every GDI+ encoder's
// FilenameExtension (BMP, JPEG, GIF, TIFF, PNG) joined by ';'. Only PNG, JPEG
// and GIF decode in this port (see logo.LoadImage).
const ImageFileFilter = "*.BMP;*.DIB;*.RLE;*.JPG;*.JPEG;*.JPE;*.JFIF;*.GIF;*.TIF;*.TIFF;*.PNG"

// BrowseFilters ports OnBtnBrowse's OpenFileDialog filters.
func BrowseFilters() []FileFilter {
	return []FileFilter{{"All Image files", ImageFileFilter}, {"All files", "*.*"}}
}

// UploadDialog holds the non-UI state of DlgUploadResources
// (ACEServiceInstaller/ICUServiceInstaller/DlgUploadResources.cs). A Fyne view binds
// its widgets to these fields and calls the methods from its event handlers.
type UploadDialog struct {
	Device       *UploadDevice // m_currentDevice (nil: "Create logo update file")
	CanCreateFWU bool          // user right FEATURE_CREATEFWU == ICURights.Full

	Filename string // m_txtFilename
	Margin   int    // m_spbMargin.Value

	MaxLogoWidth     int  // m_nMaxLogoWidth
	MaxLogoHeight    int  // m_nMaxLogoHeight
	DeviceHasDisplay bool // m_fDeviceHasDisplay

	Converted   *image.Paletted // m_imgConvertedImage
	Information string          // m_txtInformation

	// Widget state the C# toggles.
	MarginMax     int  // m_spbMargin.MaximumValue
	UploadEnabled bool // m_btnUpload.Sensitive
	CreateEnabled bool // m_btnCreateFWU.Sensitive
	CreateVisible bool // m_btnCreateFWU.Visible
	Header        string
}

// NewUploadDialog ports DlgUploadResources.Initialize (DlgUploadResources.cs:73-177): 320 x 160 limits,
// margin 8 (max min(w/2, h/2)), buttons disabled, "logo_alfen.png" pre-filled
// when it exists in the working directory.
func NewUploadDialog() *UploadDialog {
	d := &UploadDialog{
		MaxLogoWidth:  MaxLogoWidth,
		MaxLogoHeight: MaxLogoHeight,
		Margin:        DefaultImageMargin,
	}
	d.MarginMax = min(d.MaxLogoWidth/2, d.MaxLogoHeight/2)
	if st, err := os.Stat(DefaultLogoFile); err == nil && !st.IsDir() {
		d.Filename = DefaultLogoFile
	}
	return d
}

// SetObjects ports DlgUploadResources.SetObjects (DlgUploadResources.cs:179-207).
func (d *UploadDialog) SetObjects(dev *UploadDevice, canCreateFWU bool, filePath string) error {
	d.Device = dev
	d.CanCreateFWU = canCreateFWU
	if filePath != "" {
		d.Filename = filePath
	}
	d.CreateVisible = canCreateFWU
	if dev != nil {
		d.Header = fmt.Sprintf("Upload logo to device '%s' (serial number: %s)", dev.Identification, dev.SerialNumber)
	} else {
		d.Header = TitleCreateFile
	}
	d.MaxLogoWidth = MaxLogoWidthLarge
	d.MaxLogoHeight = MaxLogoHeightLarge
	d.DeviceHasDisplay = false
	if dev != nil {
		d.MaxLogoWidth = dev.MaxLogoWidth
		d.MaxLogoHeight = dev.MaxLogoHeight
		d.DeviceHasDisplay = dev.HasDisplay
	}
	d.UploadEnabled = d.DeviceHasDisplay
	return d.RefreshImage()
}

// RefreshImage ports DlgUploadResources.RefreshImage (DlgUploadResources.cs:224-246;
// run on filename and
// margin changes): when the file exists, convert it to fit
// (MaxLogoWidth-2*margin) x (MaxLogoHeight-2*margin) with 128 colours and fill
// Information; otherwise disable upload/create. The margin is clamped to the
// new maximum the way the SpinButton clamps its value.
func (d *UploadDialog) RefreshImage() error {
	st, err := os.Stat(d.Filename)
	if d.Filename == "" || err != nil || st.IsDir() {
		d.UploadEnabled = false
		d.CreateEnabled = false
		return nil
	}
	d.UploadEnabled = d.Device != nil
	d.MarginMax = min(d.MaxLogoWidth/2, d.MaxLogoHeight/2)
	if d.Margin > d.MarginMax {
		d.Margin = d.MarginMax
	}
	if d.Margin < 0 {
		d.Margin = 0
	}
	d.CreateVisible = d.CanCreateFWU
	d.CreateEnabled = true
	val, _, err := logo.LoadImage(d.Filename)
	if err != nil {
		return err
	}
	num := d.Margin
	converted, err := logo.ConvertImage(val, d.MaxLogoWidth-num*2, d.MaxLogoHeight-num*2, MaxColors)
	if err != nil {
		return err
	}
	d.Converted = converted
	b := val.Bounds()
	d.Information = fmt.Sprintf("Original image size: %d x %d pixels\nMaximum image size: %d x %d\nConverted image size: %d x %d pixels (margin %d)",
		b.Dx(), b.Dy(), d.MaxLogoWidth, d.MaxLogoHeight, converted.Rect.Dx(), converted.Rect.Dy(), num)
	return nil
}

// UploadPrompt ports the checks of OnBtnUploadClicked (DlgUploadResources.cs:248-283). With no device it
// returns ("", nil) and nothing happens; a device without a display returns
// the error the C# shows; otherwise it returns the confirmation question to
// ask (Yes starts StartUpload).
func (d *UploadDialog) UploadPrompt() (question string, err error) {
	if d.Device == nil {
		return "", nil
	}
	if !d.DeviceHasDisplay {
		return "", fmt.Errorf("The current selected device does not have a display.\nYou cannot upload a logo to device '%s'", d.Device.Identification)
	}
	return fmt.Sprintf("Are you sure upload this new image to device '%s'", d.Device.Identification), nil
}

// WorkerData builds the UploadResourceWorkerData of OnBtnUploadClicked.
func (d *UploadDialog) WorkerData(progress func(int)) UploadResourceWorkerData {
	w := UploadResourceWorkerData{CurrentDevice: d.Device, Image: d.Converted, Progress: progress}
	if d.Device != nil && d.Device.IsAHP {
		w.Filename = d.Filename
		w.Margin = d.Margin
	}
	return w
}

// BuildUploadData ports the data half of OnUploadDoWork (DlgUploadResources.cs:358-365): CreateTvfData for
// AHP, else CreateFWUData(image, 128, "UILanguages", false, "",
// maxLogoWidth > 320).
func BuildUploadData(w UploadResourceWorkerData, maxLogoWidth int) ([]byte, error) {
	if w.CurrentDevice != nil && w.CurrentDevice.IsAHP {
		return CreateTvfData(w.Filename, AppName, w.Margin, DefaultManifestVersion)
	}
	return CreateFWUData(w.Image, MaxColors, UILanguagesFolder, false, "", maxLogoWidth > MaxLogoWidth)
}

// Upload ports OnUploadDoWork (DlgUploadResources.cs:358-365) and the outcome of
// OnUploadCompleted (367-395): when the worker data has an image and a
// device, build the resource and UploadResource it. A nil return means the
// C# OnUploadCompleted shows MsgUploadSuccess; otherwise it shows the error.
func (d *UploadDialog) Upload(ctx context.Context, w UploadResourceWorkerData) error {
	if w.Image == nil || w.CurrentDevice == nil || w.CurrentDevice.Client == nil {
		return nil
	}
	data, err := BuildUploadData(w, d.MaxLogoWidth)
	if err != nil {
		return err
	}
	return UploadResource(ctx, w.CurrentDevice.Client, data, UploadOptions{
		IsAHP:    w.CurrentDevice.IsAHP,
		Progress: w.Progress,
		Property: w.CurrentDevice.Property,
	})
}

// ClampProgress ports OnUploadProgressChanged (DlgUploadResources.cs:397-407): percentages above 100 show as
// a full bar.
func ClampProgress(percent int) float64 {
	if percent > 100 {
		return 1.0
	}
	return float64(percent) / 100.0
}

// changeExtension ports Path.ChangeExtension(path, ext).
func changeExtension(path, ext string) string {
	if path == "" {
		return path
	}
	s := path
	for i := len(path) - 1; i >= 0; i-- {
		c := path[i]
		if c == '.' {
			s = path[:i]
			break
		}
		if c == '/' || c == '\\' || c == ':' {
			break
		}
	}
	if ext == "" || ext[0] != '.' {
		s += "."
	}
	return s + ext
}

// SaveDialogSetup ports SetupSaveFirmwareDialog (DlgUploadResources.cs:285-309): filters and initial file
// name of the "FWU file location" dialog.
func (d *UploadDialog) SaveDialogSetup() (filters []FileFilter, initialFileName string) {
	switch {
	case d.Device == nil:
		filters = []FileFilter{{"fwu files", "*.fwu"}, {"tvf files", "*.tvf"}}
		initialFileName = changeExtension(d.Filename, "fwu")
	case d.Device.IsAHP:
		filters = []FileFilter{{"tvf files", "*.tvf"}}
		initialFileName = changeExtension(d.Filename, "tvf")
	default:
		filters = []FileFilter{{"fwu files", "*.fwu"}}
		initialFileName = changeExtension(d.Filename, "fwu")
	}
	filters = append(filters, FileFilter{"All files", "*.*"})
	return filters, initialFileName
}

// ErrCancelled is returned by CreateFile when the small/large question was
// cancelled (the C# returns without writing).
var ErrCancelled = errors.New("cancelled")

// CreateFile ports OnBtnCreateFWUClicked after the save dialog
// (DlgUploadResources.cs:311-356): a TVF when the
// device is AHP or the chosen extension is ".tvf", otherwise an FWU — asking
// small (Eve Mini, Yes) vs large (EVE2) first when the device has a display
// and firmware < 3.3.0, then re-converting the image at 320x160 or 800x350
// (this also changes MaxLogoWidth/Height for later uploads, as in the C#).
// createCFile is the Shift+Ctrl modifier state. The file is written to
// saveFileName.
func (d *UploadDialog) CreateFile(saveFileName string, ask func(question string) Answer, createCFile bool) error {
	var array []byte
	var err error
	if (d.Device != nil && d.Device.IsAHP) || strings.ToLower(filepath.Ext(saveFileName)) == ".tvf" {
		array, err = CreateTvfData(d.Filename, AppName, d.Margin, DefaultManifestVersion)
	} else {
		flag := true
		if d.Device != nil && d.DeviceHasDisplay && versionLess(d.Device.FirmwareVersion, []int{3, 3, 0}) {
			answer := AnswerYes
			if ask != nil {
				answer = ask(MsgAskSmallLargeScreen)
			}
			if answer == AnswerCancel {
				return ErrCancelled
			}
			flag = answer != AnswerYes
		}
		if flag {
			d.MaxLogoWidth, d.MaxLogoHeight = MaxLogoWidthLarge, MaxLogoHeightLarge
		} else {
			d.MaxLogoWidth, d.MaxLogoHeight = MaxLogoWidth, MaxLogoHeight
		}
		if err := d.RefreshImage(); err != nil {
			return err
		}
		directoryName := filepath.Dir(saveFileName)
		array, err = CreateFWUData(d.Converted, MaxColors, UILanguagesFolder, createCFile, directoryName, flag)
	}
	if err != nil {
		return err
	}
	return os.WriteFile(saveFileName, array, 0o644)
}
