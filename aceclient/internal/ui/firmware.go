package ui

import (
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"

	"alfen/aceclient/internal/ui/core"
)

// PanelIDFirmware is the firmware upload page's registry ID.
const PanelIDFirmware = "firmware"

func init() {
	RegisterPanel(PanelSpec{
		ID:                 PanelIDFirmware,
		Title:              "Firmware",
		Order:              OrderFirmware,
		RequiresConnection: true,
		Build:              buildFirmware,
	})
	RegisterMenuItem(MenuItemSpec{
		Menu: "Device", Label: "Upload new firmware...", Order: 500, RequiresConnection: true,
		Action: func(s *State) { s.SelectPanel(PanelIDFirmware) },
	})
}

type firmwareView struct {
	s *State

	header   *widget.Label
	warning  *widget.Label
	version  *widget.Label
	file     *widget.Entry
	browse   *widget.Button
	report   *widget.Label
	progress *widget.ProgressBarInfinite
	progText *widget.Label
	upload   *widget.Button

	data      []byte
	dataPath  string
	uploading bool
}

// buildFirmware ports DlgUpload: device header, NG9xx upgrade warning,
// current version, file location with "..." browse (DlgUpload's filters), a
// pre-flight classification (internal/fwi, internal/tvf) and "Start upload"
// with a busy indicator. The local firmware library list (FTP-synchronised
// LocalFirmwareFolder + ICUConfig.FindFirmware comments) is not ported here.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/DlgUpload.cs:74-167
func buildFirmware(s *State) fyne.CanvasObject {
	v := &firmwareView{s: s}
	s.setView(PanelIDFirmware, v)

	v.header = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	v.warning = widget.NewLabel(core.UploadNg9xxUpgradeNotice)
	v.warning.Importance = widget.DangerImportance
	v.warning.Hide()
	v.version = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	v.file = widget.NewEntry()
	v.file.SetPlaceHolder(core.UploadFilePlaceholder)
	v.file.OnChanged = func(string) { v.fileChanged() } // OnFilenameChanged
	v.browse = widget.NewButton("...", v.showBrowse)
	v.report = widget.NewLabel("")
	v.report.Wrapping = fyne.TextWrapWord
	v.report.TextStyle = fyne.TextStyle{Monospace: true}
	v.progress = widget.NewProgressBarInfinite()
	v.progress.Hide()
	v.progText = widget.NewLabelWithStyle(core.UploadProgressInitial, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	v.progText.Hide()
	v.upload = widget.NewButton(core.UploadButton, v.startUpload)
	v.upload.Disable()

	s.OnConnect(func(*State) { v.changeDevice() })
	s.OnPropertiesChanged(func(*State) { v.changeDevice() })

	form := container.New(layout.NewFormLayout(),
		widget.NewLabel(core.UploadCurrentVersion), v.version,
		widget.NewLabel(core.UploadFileLocation), container.NewBorder(nil, nil, nil, v.browse, v.file))
	return container.NewBorder(
		container.NewVBox(v.header, v.warning, form),
		container.NewVBox(v.progress, v.progText, container.NewHBox(v.upload, layout.NewSpacer())),
		nil, nil,
		container.NewVScroll(v.report))
}

// changeDevice ports DlgUpload.ChangeDevice: header, upgrade warning and the
// current version (0x100A_0) of the connected device.
func (v *firmwareView) changeDevice() {
	sess := v.s.Session()
	if sess == nil {
		return
	}
	v.header.SetText(core.UploadHeader(sess.Identification(), sess.Props.String(8273, 0)))
	if core.ShowNg9xxUpgradeWarning(sess.IsAHP(), sess.FirmwareVersion()) {
		v.warning.Show()
	} else {
		v.warning.Hide()
	}
	v.version.SetText(sess.Props.String(4106, 0))
	v.updateUploadButton()
}

// showBrowse opens the file picker with DlgUpload's "All firmware files"
// patterns (Fyne dialogs support a single filter).
func (v *firmwareView) showBrowse() {
	d := dialog.NewFileOpen(func(rc fyne.URIReadCloser, err error) {
		if err != nil || rc == nil {
			return
		}
		path := rc.URI().Path()
		_ = rc.Close()
		v.file.SetText(path)
	}, v.s.Window)
	d.SetTitleText(core.UploadBrowseTitle)
	d.SetFilter(storage.NewExtensionFileFilter(core.FirmwareFileFilters[0].Extensions()))
	if dir := filepath.Dir(v.file.Text); v.file.Text != "" {
		if l, err := storage.ListerForURI(storage.NewFileURI(dir)); err == nil {
			d.SetLocation(l)
		}
	}
	d.Resize(fyne.NewSize(800, 560))
	d.Show()
}

// fileChanged reads and classifies the chosen file off the UI goroutine.
func (v *firmwareView) fileChanged() {
	path := strings.TrimSpace(v.file.Text)
	v.data, v.dataPath = nil, ""
	v.updateUploadButton()
	if path == "" {
		v.report.SetText("")
		return
	}
	st, err := os.Stat(path)
	if err != nil || st.IsDir() {
		v.report.SetText("")
		return
	}
	sess := v.s.Session()
	known, ahp := sess != nil, sess != nil && sess.IsAHP()
	var data []byte
	var text string
	v.s.RunAsync(func() {
		data, err = os.ReadFile(path)
		if err == nil {
			text = core.ClassifyFirmware(path, data, known, ahp).Text()
		}
	}, func() {
		if strings.TrimSpace(v.file.Text) != path {
			return // superseded
		}
		if err != nil {
			v.report.SetText(err.Error())
			return
		}
		v.data, v.dataPath = data, path
		v.report.SetText(text)
		v.updateUploadButton()
	})
}

// updateUploadButton ports `m_btnUpload.Sensitive = File.Exists(...)`, plus
// a logged-in device and no upload in progress.
func (v *firmwareView) updateUploadButton() {
	if v.data != nil && v.s.Connected() && !v.uploading {
		v.upload.Enable()
	} else {
		v.upload.Disable()
	}
}

// startUpload ports OnBtnUploadClicked → OnUploadDoWork → OnUploadCompleted
// for the HTTP upload (core.Session.UploadFirmware).
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/DlgUpload.cs:267-321, 332-398
func (v *firmwareView) startUpload() {
	sess := v.s.Session()
	if sess == nil || v.data == nil {
		return
	}
	path := v.dataPath
	v.s.AskQuestion(core.UploadConfirmQuestion(sess.Identification(), path), func(yes bool) {
		if !yes {
			return
		}
		if core.NeedsNewPassword(sess.IsAHP(), sess.FirmwareVersion(), path) {
			// DlgDeviceNewPassword + the post-reboot ChangePassword belong to
			// the firmware state machine, which is not ported yet.
			v.s.ShowError("This upgrade (NG9xx < 5.x to >= 5.x) requires setting a new device password after the update.",
				"That step is not supported by this version; use the original installer for this upgrade.")
			return
		}
		v.runUpload(sess, path)
	})
}

func (v *firmwareView) runUpload(sess *core.Session, path string) {
	data := v.data
	v.uploading = true
	v.updateUploadButton()
	v.browse.Disable()
	v.file.Disable()
	v.progress.Show()
	v.progress.Start()
	v.progText.SetText(core.UploadProgressInitial)
	v.progText.Show()
	v.s.SetStatus("Uploading %s to %s...", filepath.Base(path), sess.Identification())
	ld := sess.LoginData()
	var lastErr string
	v.s.RunAsync(func() {
		lastErr = sess.UploadFirmware(data, func() int { return sess.LoginRequest(ld).StatusCode })
		if lastErr == "" {
			_ = sess.UpdateCategories("generic") // OnUploadCompleted: lanDev.UpdateCategories("generic")
		}
	}, func() {
		v.uploading = false
		v.progress.Stop()
		v.progress.Hide()
		v.progText.Hide()
		v.browse.Enable()
		v.file.Enable()
		v.updateUploadButton()
		v.changeDevice()
		if lastErr != "" {
			v.s.SetStatus("%s %s", core.UploadErrorPrimary, lastErr)
			v.s.ShowError(core.UploadErrorPrimary, lastErr)
			return
		}
		v.s.SetStatus("Firmware file accepted by %s; the charging station installs it and reboots on its own.", sess.Identification())
		v.s.ShowMessage(core.UploadSuccess)
		v.s.NotifyPropertiesChanged()
	})
}
