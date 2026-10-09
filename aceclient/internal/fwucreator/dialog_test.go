package fwucreator

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"alfen/aceclient/internal/api"
	"alfen/aceclient/internal/fwi"
)

func props(m map[[2]int]string) PropertyLookup {
	return func(id uint16, sub byte) (api.Property, bool) {
		v, ok := m[[2]int{int(id), int(sub)}]
		return api.Property{ID: id, Sub: sub, DataType: api.SDTUnsigned16, Value: v}, ok
	}
}

func TestMaxLogoSizeAndDisplay(t *testing.T) {
	tests := []struct {
		name         string
		p            PropertyLookup
		old, dualPG  bool
		w, h         int
		display      bool
		largeDisplay bool
	}{
		{"display props", props(map[[2]int]string{{0x3260, 1}: "1", {0x3260, 3}: "480", {0x3260, 4}: "272"}), false, false, 480, 272, true, true},
		{"zero height", props(map[[2]int]string{{0x3260, 1}: "1", {0x3260, 3}: "480", {0x3260, 4}: "0"}), true, false, 0, 0, true, false},
		{"old model", props(nil), true, false, 320, 160, true, false},
		{"no display", props(nil), false, false, 0, 0, false, false},
		{"dual PG", nil, false, true, 0, 0, true, false},
	}
	for _, tc := range tests {
		w, h := MaxLogoSize(tc.p, tc.old)
		if w != tc.w || h != tc.h || HasDisplay(w, tc.old, tc.dualPG) != tc.display || HasLargeDisplay(w) != tc.largeDisplay {
			t.Errorf("%s: %dx%d display=%v", tc.name, w, h, HasDisplay(w, tc.old, tc.dualPG))
		}
	}
}

func TestDeviceSerialNumber(t *testing.T) {
	p := func(id uint16, sub byte) (api.Property, bool) {
		return api.Property{ID: id, Sub: sub, DataType: api.SDTVisibleString, Value: "ACE0123456"}, id == 0x2051 && sub == 0
	}
	if got := DeviceSerialNumber(p); got != "ACE0123456" || DeviceSerialNumber(nil) != "" {
		t.Fatalf("serial %q", got)
	}
}

func TestIsAHPModel(t *testing.T) {
	for m, want := range map[string]bool{"AHP02": true, "ahwp-x": true, "AHPDC": true, "NG910-60023": false, "": false, "AH": false} {
		if IsAHPModel(m) != want {
			t.Errorf("%q", m)
		}
	}
}

func TestVersionLess(t *testing.T) {
	tests := []struct {
		a    []int
		want bool
	}{
		{[]int{3, 2, 9}, true}, {[]int{3, 3}, true}, {[]int{3, 3, 0}, false}, {[]int{3, 3, 0, 0}, false}, {[]int{4, 0}, false},
	}
	for _, tc := range tests {
		if versionLess(tc.a, []int{3, 3, 0}) != tc.want {
			t.Errorf("%v < 3.3.0 != %v", tc.a, tc.want)
		}
	}
}

func TestChangeExtension(t *testing.T) {
	for in, want := range map[string]string{
		"logo.png": "logo.fwu", "dir.v1/logo": "dir.v1/logo.fwu", `C:\a.b\logo.jpeg`: `C:\a.b\logo.fwu`, "": "", "x.": "x.fwu",
	} {
		if got := changeExtension(in, "fwu"); got != want {
			t.Errorf("%q -> %q, want %q", in, got, want)
		}
	}
}

func TestUploadDialogFlow(t *testing.T) {
	src, _ := filepath.Abs(filepath.Join(installerFiles, DefaultLogoFile))
	if _, err := os.Stat(src); err != nil {
		t.Skipf("fixture missing: %v", err)
	}
	d := NewUploadDialog()
	if d.Margin != 8 || d.MarginMax != 80 {
		t.Fatalf("initial margin %d/%d", d.Margin, d.MarginMax)
	}
	// No device: "Create logo update file", large limits, upload disabled.
	if err := d.SetObjects(nil, true, src); err != nil {
		t.Fatal(err)
	}
	if d.Header != TitleCreateFile || d.MaxLogoWidth != 800 || d.MaxLogoHeight != 350 || d.UploadEnabled || !d.CreateEnabled || !d.CreateVisible {
		t.Fatalf("no-device state %+v", d)
	}
	if q, err := d.UploadPrompt(); q != "" || err != nil {
		t.Fatal("upload without device must be a no-op")
	}
	if !strings.HasPrefix(d.Information, "Original image size: 1048 x 342 pixels\nMaximum image size: 800 x 350\nConverted image size: ") ||
		!strings.HasSuffix(d.Information, " pixels (margin 8)") {
		t.Fatalf("information %q", d.Information)
	}
	if d.Converted.Rect.Dx() > 800-16 || d.Converted.Rect.Dy() > 350-16 {
		t.Fatalf("converted %v", d.Converted.Rect)
	}
	filters, name := d.SaveDialogSetup()
	if len(filters) != 3 || filters[1].Pattern != "*.tvf" || name != strings.TrimSuffix(src, ".png")+".fwu" {
		t.Fatalf("save dialog %v %q", filters, name)
	}

	// Device without display: header with serial, upload refused with the C# text.
	dev := &UploadDevice{Identification: "ACE0001", SerialNumber: "SN1", FirmwareVersion: []int{3, 2, 0}}
	if err := d.SetObjects(dev, false, ""); err != nil {
		t.Fatal(err)
	}
	if d.Header != "Upload logo to device 'ACE0001' (serial number: SN1)" || d.CreateVisible {
		t.Fatalf("device header %q", d.Header)
	}
	if _, err := d.UploadPrompt(); err == nil || err.Error() != "The current selected device does not have a display.\nYou cannot upload a logo to device 'ACE0001'" {
		t.Fatalf("no-display error %v", err)
	}

	// Device with a 320x160 display and fw < 3.3.0: the create path asks small/large.
	dev.HasDisplay, dev.MaxLogoWidth, dev.MaxLogoHeight = true, 320, 160
	if err := d.SetObjects(dev, true, ""); err != nil {
		t.Fatal(err)
	}
	if q, err := d.UploadPrompt(); err != nil || q != "Are you sure upload this new image to device 'ACE0001'" {
		t.Fatalf("prompt %q %v", q, err)
	}
	wd := d.WorkerData(nil)
	if wd.Image != d.Converted || wd.Filename != "" || wd.Margin != 0 {
		t.Fatal("non-AHP worker data must not carry filename/margin")
	}
	out := filepath.Join(t.TempDir(), "logo.fwu")
	asked := ""
	if err := d.CreateFile(out, func(q string) Answer { asked = q; return AnswerCancel }, false); err != ErrCancelled {
		t.Fatalf("cancel: %v", err)
	}
	if asked != MsgAskSmallLargeScreen {
		t.Fatalf("question %q", asked)
	}
	langCwd := t.TempDir()
	wdir, _ := os.Getwd()
	os.Chdir(langCwd) // CreateFWUData uses the relative "UILanguages" folder
	defer os.Chdir(wdir)
	if err := d.CreateFile(out, func(string) Answer { return AnswerYes }, false); err != nil {
		t.Fatal(err)
	}
	if d.MaxLogoWidth != 320 || d.MaxLogoHeight != 160 {
		t.Fatalf("small screen limits %dx%d", d.MaxLogoWidth, d.MaxLogoHeight)
	}
	file, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	stream, err := fwi.UnwrapDisplayPayload(file)
	if err != nil {
		t.Fatal(err)
	}
	objs, err := ParseObjects(stream)
	if err != nil || len(objs) != 7 || int(objs[0].Width) != d.Converted.Rect.Dx() {
		t.Fatalf("small FWU: %d objects %v", len(objs), err)
	}
	if st, err := os.Stat(filepath.Join(langCwd, UILanguagesFolder)); err != nil || !st.IsDir() {
		t.Fatal("UILanguages not created in the working directory")
	}

	// AHP: TVF regardless of the chosen extension; worker data carries file/margin.
	dev.IsAHP = true
	tvfOut := filepath.Join(t.TempDir(), "logo.fwu")
	if err := d.CreateFile(tvfOut, nil, false); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(tvfOut)
	if len(b) < 8 || b[0] != 1 || b[1] != 0 {
		t.Fatal("AHP create did not write a TVF")
	}
	if wd := d.WorkerData(nil); wd.Filename != src || wd.Margin != d.Margin {
		t.Fatalf("AHP worker data %+v", wd)
	}
	if f, n := d.SaveDialogSetup(); len(f) != 2 || f[0].Pattern != "*.tvf" || !strings.HasSuffix(n, ".tvf") {
		t.Fatalf("AHP save dialog %v %q", f, n)
	}
}

func TestRefreshImageMissingFile(t *testing.T) {
	d := NewUploadDialog()
	d.Filename = filepath.Join(t.TempDir(), "nope.png")
	d.UploadEnabled, d.CreateEnabled = true, true
	if err := d.RefreshImage(); err != nil || d.UploadEnabled || d.CreateEnabled {
		t.Fatalf("missing file: %v %+v", err, d)
	}
}

func TestClampProgress(t *testing.T) {
	if ClampProgress(150) != 1 || ClampProgress(50) != 0.5 {
		t.Fatal("clamp")
	}
}

func TestStatusMessages(t *testing.T) {
	text, icon := StatusMessageNew(UIStateError, UIErrorRelays)
	if text != "102: Not able to charge.\n Please call for support" || icon != StatusIconError {
		t.Errorf("new error %q %v", text, icon)
	}
	text, icon = StatusMessageNew(UIStateCharging, UIErrorRelays) // not in error state -> NONE
	if text != "000: Installation OK" || icon != StatusIconValid {
		t.Errorf("non-error %q", text)
	}
	text, icon = StatusMessageNew(UIStateError, 999)
	if text != "999: Unknown Error/Warning state \nPlease call for support" || icon != StatusIconWarning {
		t.Errorf("unknown %q", text)
	}
	text, icon = StatusMessageOld(StateErrorRelays)
	if text != "102: Not able to Charge" || icon != StatusIconError {
		t.Errorf("old %q", text)
	}
	text, _ = StatusMessageOld(MainState(115))
	if text != "115 \nUnknown Error/Warning state" {
		t.Errorf("old unknown %q", text)
	}
	text, _ = StatusMessageOld(StateBooting)
	if text != "STATE_BOOTING \nUnknown Error/Warning state" {
		t.Errorf("old unlisted %q", text)
	}
	if len(DisplayStatesMessagesOld) != 16 || len(DisplayStatesMessagesNew) != 30 {
		t.Errorf("table sizes %d %d", len(DisplayStatesMessagesOld), len(DisplayStatesMessagesNew))
	}
}
