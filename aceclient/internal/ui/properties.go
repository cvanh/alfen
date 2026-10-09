package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"alfen/aceclient/internal/ui/core"
)

// PanelIDProperties is the All Properties page's registry ID.
const PanelIDProperties = "allproperties"

const (
	allCategories = "All categories"
	searchResults = "Search" // PanelAllProperties' search category name
)

func init() {
	RegisterPanel(PanelSpec{
		ID:                 PanelIDProperties,
		Title:              "All Properties", // PanelAllProperties.Title
		Order:              OrderAllProperties,
		FeatureID:          FeatureAllProperties,
		RequiresConnection: true,
		Build:              buildProperties,
	})
}

var propertyColumns = []string{"ID", "Name", "Value", "Type", "Access", "", "Device name"}

type propertiesView struct {
	s *State

	categories   []string
	groups       []core.PropertyGroup
	search       []core.PropertyRow
	searchTerm   string
	rows         []core.PropertyRow
	sel          int
	loadedFor    *core.Session
	specialWrite bool
	loading      bool

	category *widget.Select
	term     *widget.Entry
	chkName  *widget.Check
	chkValue *widget.Check
	chkID    *widget.Check
	chkRegex *widget.Check
	table    *widget.Table
	info     *widget.Label

	editTitle *widget.Label
	editMeta  *widget.Label
	editHost  *fyne.Container
	editMsg   *widget.Label
	apply     *widget.Button
	undo      *widget.Button
	entry     *widget.Entry
	check     *widget.Check
	choice    *widget.Select
}

// buildProperties ports PanelAllProperties: every category the device lists
// (GET /api/categories), the properties of each read with UpdateCategories(),
// a category filter, the Name/Value/ID/Regex search, and select-to-edit with
// the per-SDT control the C# would create. Edits mark the property changed;
// the shell's Save stores them (SaveAllChanges → StoreProperties).
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/PanelAllProperties.cs:46-355
func buildProperties(s *State) fyne.CanvasObject {
	v := &propertiesView{s: s, sel: -1}
	s.setView(PanelIDProperties, v)

	v.category = widget.NewSelect(nil, func(string) { v.showRows() })
	v.term = widget.NewEntry()
	v.term.SetPlaceHolder("Search term")
	v.term.OnSubmitted = func(string) { v.runSearch() }
	searchBtn := widget.NewButton("Search", v.runSearch)
	v.chkName = widget.NewCheck("Name", nil)
	v.chkValue = widget.NewCheck("Value", nil)
	v.chkID = widget.NewCheck("ID", nil)
	v.chkRegex = widget.NewCheck("Regex", nil)
	v.chkName.SetChecked(core.DefaultSearchOptions.Name)
	v.chkID.SetChecked(core.DefaultSearchOptions.ID)
	reload := widget.NewButton("Reload", v.load)

	v.info = widget.NewLabel(core.AdvancedSettingsWarning)
	v.info.Wrapping = fyne.TextWrapWord
	v.info.Importance = widget.WarningImportance

	v.table = widget.NewTableWithHeaders(
		func() (int, int) { return len(v.rows), len(propertyColumns) },
		func() fyne.CanvasObject {
			l := widget.NewLabel("")
			l.Truncation = fyne.TextTruncateEllipsis
			return l
		},
		func(id widget.TableCellID, o fyne.CanvasObject) {
			if id.Row >= 0 && id.Row < len(v.rows) {
				o.(*widget.Label).SetText(v.cell(v.rows[id.Row], id.Col))
			}
		})
	v.table.ShowHeaderColumn = false
	v.table.CreateHeader = func() fyne.CanvasObject {
		return widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	}
	v.table.UpdateHeader = func(id widget.TableCellID, o fyne.CanvasObject) {
		if id.Col >= 0 && id.Col < len(propertyColumns) {
			o.(*widget.Label).SetText(propertyColumns[id.Col])
		}
	}
	for i, w := range []float32{100, 260, 260, 120, 60, 24, 200} {
		v.table.SetColumnWidth(i, w)
	}
	v.table.OnSelected = func(id widget.TableCellID) { v.selectRow(id.Row) }

	v.editTitle = widget.NewLabelWithStyle("Select a property to edit it.", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	v.editMeta = widget.NewLabel("")
	v.editHost = container.NewStack()
	v.editMsg = widget.NewLabel("")
	v.editMsg.Wrapping = fyne.TextWrapWord
	v.apply = widget.NewButton("Apply", v.applyEdit)
	v.undo = widget.NewButton("Undo", v.undoEdit)
	v.apply.Disable()
	v.undo.Disable()

	s.OnPanelShown(PanelIDProperties, func(*State) {
		if s.Connected() && v.loadedFor != s.Session() {
			v.load()
		}
	})
	s.OnPropertiesChanged(func(*State) { v.rebuild() })
	s.OnDisconnect(func(*State) { v.clear() })

	tools := container.NewVBox(
		container.NewBorder(nil, nil, widget.NewLabel("Category:"), reload, v.category),
		container.NewBorder(nil, nil, widget.NewLabel("Search:"),
			container.NewHBox(widget.NewLabel("Filter:"), v.chkName, v.chkValue, v.chkID, v.chkRegex, searchBtn), v.term),
		v.info)
	editor := container.NewVBox(widget.NewSeparator(), v.editTitle, v.editMeta,
		container.NewBorder(nil, nil, nil, container.NewHBox(v.undo, v.apply), v.editHost), v.editMsg)
	return container.NewBorder(tools, editor, nil, nil, v.table)
}

func (v *propertiesView) cell(r core.PropertyRow, col int) string {
	pc := v.s.Props()
	switch col {
	case 0:
		return core.ODIndex(r.ID, r.Sub)
	case 1:
		return r.Label
	case 2:
		if pc != nil {
			return core.RowDisplay(pc, r)
		}
		return r.Display
	case 3:
		return core.SDTName(r.DataType)
	case 4:
		if r.ReadOnly {
			return "RO"
		}
		return "RW"
	case 5:
		if r.Changed {
			return "*" // UIPropertyBase m_imvChanged "This property is changed"
		}
		return ""
	case 6:
		return r.Name
	}
	return ""
}

// specialWritePermission ports PanelAllProperties.OnChangeDevice's
// Keyboard.CurrentModifiers check: Shift+Ctrl held while the page loads
// unlocks 8271/8272/8273.
func (v *propertiesView) specialWritePermission() bool {
	d, ok := v.s.App.Driver().(desktop.Driver)
	if !ok {
		return false
	}
	m := d.CurrentKeyModifiers()
	return m&fyne.KeyModifierShift != 0 && m&fyne.KeyModifierControl != 0
}

// load ports OnChangeDevice's reads: RequestCategories for the category list,
// then UpdateCategories() (which lists the categories again and reads each).
func (v *propertiesView) load() {
	sess := v.s.Session()
	if sess == nil || v.loading {
		return
	}
	v.loading = true
	v.specialWrite = v.specialWritePermission()
	var cats []string
	var err error
	v.s.SetStatus("Reading all properties of %s...", sess.Identification())
	v.s.RunAsync(func() {
		cats = sess.RequestCategories()
		err = sess.UpdateCategories()
	}, func() {
		v.loading = false
		if v.s.Session() != sess {
			return
		}
		v.categories = cats
		v.loadedFor = sess
		if err != nil {
			v.s.SetStatus("Reading properties failed: %s", requestErrorText(err))
		} else {
			v.s.SetStatus("Read %d properties from %s.", sess.Props.Len(), sess.Identification())
		}
		v.s.NotifyPropertiesChanged()
	})
}

// rebuild recomputes the groups from the cache (UI goroutine).
func (v *propertiesView) rebuild() {
	pc := v.s.Props()
	if pc == nil || v.loadedFor == nil {
		return
	}
	v.groups = core.AllPropertiesGroups(pc, v.categories, v.specialWrite)
	if v.searchTerm != "" {
		if res, err := core.Search(pc, v.searchTerm, v.searchOptions()); err == nil {
			v.search = core.PropertyRows(pc, res.Matches, v.specialWrite)
		}
	}
	opts := []string{allCategories}
	for _, g := range v.groups {
		opts = append(opts, g.Name)
	}
	if v.searchTerm != "" {
		opts = append(opts, searchResults)
	}
	cur := v.category.Selected
	v.category.Options = opts
	if cur == "" || !contains(opts, cur) {
		cur = allCategories
	}
	v.category.Selected = cur
	v.category.Refresh()
	v.showRows()
}

func contains(l []string, s string) bool {
	for _, x := range l {
		if x == s {
			return true
		}
	}
	return false
}

// showRows fills the table for the selected category, keeping the selection.
func (v *propertiesView) showRows() {
	var key uint32
	hadSel := v.sel >= 0 && v.sel < len(v.rows)
	if hadSel {
		key = v.rows[v.sel].Key()
	}
	var rows []core.PropertyRow
	switch v.category.Selected {
	case searchResults:
		rows = v.search
	case allCategories, "":
		for _, g := range v.groups {
			rows = append(rows, g.Rows...)
		}
	default:
		for _, g := range v.groups {
			if g.Name == v.category.Selected {
				rows = g.Rows
			}
		}
	}
	v.rows = rows
	v.sel = -1
	if hadSel {
		for i, r := range rows {
			if r.Key() == key {
				v.sel = i
			}
		}
	}
	v.table.Refresh()
	if v.sel >= 0 {
		v.showEditor(v.rows[v.sel], false)
	} else {
		v.clearEditor()
	}
}

func (v *propertiesView) searchOptions() core.SearchOptions {
	return core.SearchOptions{Name: v.chkName.Checked, Value: v.chkValue.Checked, ID: v.chkID.Checked, Regex: v.chkRegex.Checked}
}

// runSearch ports OnSearchClicked.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/PanelAllProperties.cs:163-211
func (v *propertiesView) runSearch() {
	pc := v.s.Props()
	if pc == nil {
		return
	}
	res, err := core.Search(pc, v.term.Text, v.searchOptions())
	if err != nil {
		v.s.ShowError(err.Error())
		return
	}
	if res.Warning != "" {
		v.s.ShowError(res.Warning)
	}
	v.searchTerm = v.term.Text
	v.search = core.PropertyRows(pc, res.Matches, v.specialWrite)
	if len(v.search) == 0 {
		v.s.SetStatus("%s", core.SearchNoResults(v.term.Text))
	} else {
		v.s.SetStatus("%d properties match %q.", len(v.search), v.term.Text)
	}
	v.rebuild()
	v.category.SetSelected(searchResults)
}

func (v *propertiesView) selectRow(row int) {
	if row < 0 || row >= len(v.rows) {
		return
	}
	v.sel = row
	v.showEditor(v.rows[row], true)
}

func (v *propertiesView) clearEditor() {
	v.editTitle.SetText("Select a property to edit it.")
	v.editMeta.SetText("")
	v.editHost.Objects = nil
	v.editHost.Refresh()
	v.editMsg.SetText("")
	v.apply.Disable()
	v.undo.Disable()
	v.entry, v.check, v.choice = nil, nil, nil
}

// editable reports whether the row may be edited: an editable kind, not
// read-only, and page rights not ReadOnly (UIPropertyBase.CheckAccessRights).
func (v *propertiesView) editable(r core.PropertyRow) bool {
	return r.Kind != core.EditorReadOnly && !r.ReadOnly && v.s.Rights(FeatureAllProperties) == RightsFull
}

// showEditor builds the control the C# would use for the row.
func (v *propertiesView) showEditor(r core.PropertyRow, reset bool) {
	pc := v.s.Props()
	v.editTitle.SetText(r.Label)
	v.editMeta.SetText(fmt.Sprintf("%s  %s  device value: %s  max length: %d",
		core.ODIndex(r.ID, r.Sub), core.SDTName(r.DataType), r.DeviceValue, r.MaxLength))
	if !reset && len(v.editHost.Objects) > 0 {
		v.undo.Enable()
		if !r.Changed {
			v.undo.Disable()
		}
		return
	}
	v.entry, v.check, v.choice = nil, nil, nil
	v.editMsg.SetText("")
	var obj fyne.CanvasObject
	switch r.Kind {
	case core.EditorText:
		e := widget.NewEntry()
		e.SetText(r.Display)
		var eds *core.EDSParam
		if p, ok := pc.Lookup(r.ID, r.Sub); ok {
			eds = &p
		}
		prop := r.Property
		e.OnChanged = func(t string) { // UIPropertyString.onTxtChanged truncation
			if c := core.TextInput(prop, eds, t); c != t {
				e.SetText(c)
			}
		}
		v.entry, obj = e, e
	case core.EditorNumber:
		e := widget.NewEntry()
		e.SetText(r.Value)
		min, max := core.NumberRange(r.DataType, r.MaxLength)
		e.SetPlaceHolder(fmt.Sprintf("%g … %g", min, max))
		v.entry, obj = e, e
	case core.EditorCheck:
		c := widget.NewCheck(r.Label, nil)
		c.SetChecked(core.CheckState(r.Value))
		v.check, obj = c, c
	case core.EditorSelect:
		var titles []string
		for _, o := range r.Options {
			titles = append(titles, o.Title)
		}
		c := widget.NewSelect(titles, nil)
		for _, o := range r.Options {
			if o.Value == r.Value {
				c.SetSelected(o.Title)
			}
		}
		v.choice, obj = c, c
	default:
		l := widget.NewLabel(core.RowDisplay(pc, r))
		l.Wrapping = fyne.TextWrapWord
		obj = l
	}
	v.editHost.Objects = []fyne.CanvasObject{obj}
	v.editHost.Refresh()
	if v.editable(r) {
		v.apply.Enable()
		if d, ok := obj.(fyne.Disableable); ok {
			d.Enable()
		}
	} else {
		v.apply.Disable()
		if d, ok := obj.(fyne.Disableable); ok {
			d.Disable()
		}
		if r.Kind != core.EditorReadOnly {
			v.editMsg.SetText("This property is read-only.")
		}
	}
	if r.Changed {
		v.undo.Enable()
	} else {
		v.undo.Disable()
	}
}

// applyEdit converts the control's value like the C# control does and
// assigns it to the property (ICUProperty.Value → IsChanged).
func (v *propertiesView) applyEdit() {
	pc := v.s.Props()
	if pc == nil || v.sel < 0 || v.sel >= len(v.rows) {
		return
	}
	r := v.rows[v.sel]
	if !v.editable(r) {
		return
	}
	var value string
	note := ""
	switch r.Kind {
	case core.EditorText:
		var eds *core.EDSParam
		if p, ok := pc.Lookup(r.ID, r.Sub); ok {
			eds = &p
		}
		value = core.TextInput(r.Property, eds, v.entry.Text)
	case core.EditorNumber:
		val, clamped, err := core.NumberInput(r.DataType, r.MaxLength, r.Digits, v.entry.Text)
		if err != nil {
			v.editMsg.SetText(err.Error())
			return
		}
		if clamped {
			note = "Value limited to the allowed range: " + val
		}
		value = val
	case core.EditorCheck:
		value = core.CheckboxValue(r.DataType, v.check.Checked)
	case core.EditorSelect:
		for _, o := range r.Options {
			if o.Title == v.choice.Selected {
				value = o.Value
			}
		}
	}
	p, err := pc.SetValue(r.ID, r.Sub, value)
	if err != nil {
		v.editMsg.SetText(err.Error())
		return
	}
	if note == "" && p.Changed {
		note = "Changed; press Save to store it on the charging station."
	}
	v.editMsg.SetText(note)
	v.s.NotifyPropertiesChanged()
}

// undoEdit rolls the selected property back (ICUProperty.Rollback).
func (v *propertiesView) undoEdit() {
	pc := v.s.Props()
	if pc == nil || v.sel < 0 || v.sel >= len(v.rows) {
		return
	}
	pc.Revert(v.rows[v.sel].Key())
	v.s.NotifyPropertiesChanged()
	if v.sel >= 0 && v.sel < len(v.rows) {
		v.showEditor(v.rows[v.sel], true)
	}
}

func (v *propertiesView) clear() {
	v.categories, v.groups, v.search, v.rows = nil, nil, nil, nil
	v.searchTerm = ""
	v.loadedFor = nil
	v.sel = -1
	v.category.Options = nil
	v.category.Selected = ""
	v.category.Refresh()
	v.table.Refresh()
	v.clearEditor()
}
