package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"alfen/aceclient/internal/ui/core"
)

// PanelIDInformation is the Information page's registry ID.
const PanelIDInformation = "information"

func init() {
	RegisterPanel(PanelSpec{
		ID:                 PanelIDInformation,
		Title:              "Information", // PanelInformation.Title
		Order:              OrderInformation,
		FeatureID:          FeatureInformation,
		RequiresConnection: true,
		Build:              buildInformation,
	})
	// PanelInformation.OnSaveChanges runs for every Save, whatever page
	// edited the property (MainWindow iterates all active panels).
	RegisterSaveCheck(func(s *State) (string, string) {
		sess := s.Session()
		if sess == nil {
			return "", ""
		}
		c := core.InformationSaveCheck(sess.Props, sess.IsAHP(), sess.Identification())
		return c.Error, c.Question
	})
}

type informationView struct {
	s    *State
	body *fyne.Container
	head *widget.Label
	busy bool
}

// buildInformation ports PanelInformation as a read-only identity view: on
// every login it reads the "comm" category (OnChangeDevice →
// UpdateCategories("comm")) and renders core.InformationRows. The editable
// rows of the C# (identity, license key, location, SSA) are edited in the
// All Properties page until a later phase deepens this view.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/PanelInformation.cs:290-498
func buildInformation(s *State) fyne.CanvasObject {
	v := &informationView{s: s}
	s.setView(PanelIDInformation, v)
	v.head = widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	v.body = container.NewVBox()

	refresh := widget.NewButton("Refresh", v.load)
	logout := widget.NewButton("Logout", func() { s.Disconnect(nil) }) // "Logout from this CS"
	upload := widget.NewButton("Upload Firmware...", func() { s.SelectPanel(PanelIDFirmware) })

	s.OnConnect(func(*State) { v.load() })
	s.OnPropertiesChanged(func(*State) { v.render() })
	s.OnDisconnect(func(*State) {
		v.body.Objects = nil
		v.body.Refresh()
		v.head.SetText("")
	})

	tools := container.NewHBox(v.head, layout.NewSpacer(), refresh, upload, logout)
	return container.NewBorder(tools, nil, nil, nil, container.NewVScroll(v.body))
}

// load reads the "comm" category off the UI goroutine, then renders.
func (v *informationView) load() {
	sess := v.s.Session()
	if sess == nil || v.busy {
		return
	}
	v.busy = true
	var err error
	v.s.RunAsync(func() { err = sess.UpdateCategories("comm") }, func() {
		v.busy = false
		if err != nil {
			v.s.SetStatus("Reading information failed: %s", requestErrorText(err))
		}
		v.s.NotifyPropertiesChanged()
	})
}

// render rebuilds the rows from the property cache (UI goroutine).
func (v *informationView) render() {
	sess := v.s.Session()
	if sess == nil || !v.s.Connected() {
		return
	}
	v.head.SetText(sess.Identification())
	rows := core.InformationRows(sess.Props, core.InformationContextFor(sess))
	var objs []fyne.CanvasObject
	var grid *fyne.Container
	cat := ""
	flush := func() {
		if grid != nil && len(grid.Objects) > 0 {
			objs = append(objs, grid)
		}
		grid = container.New(layout.NewFormLayout())
	}
	flush()
	for _, r := range rows {
		if r.Category != cat {
			flush()
			cat = r.Category
			objs = append(objs, widget.NewSeparator(),
				widget.NewLabelWithStyle(cat, fyne.TextAlignLeading, fyne.TextStyle{Bold: true}))
		}
		switch {
		case r.Header:
			flush()
			objs = append(objs, widget.NewLabelWithStyle(r.Label, fyne.TextAlignLeading, fyne.TextStyle{Italic: true}))
		case r.Info:
			flush()
			l := widget.NewLabel(r.Value)
			l.Wrapping = fyne.TextWrapWord
			l.Importance = widget.LowImportance
			objs = append(objs, l)
		default:
			val := widget.NewLabel(r.Value)
			val.Wrapping = fyne.TextWrapWord
			grid.Add(widget.NewLabel(r.Label))
			grid.Add(val)
		}
	}
	flush()
	v.body.Objects = objs
	v.body.Refresh()
}
