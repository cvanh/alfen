package settings

// DlgSettings texts (ACEServiceInstaller/ICUServiceInstaller/DlgSettings.cs).
const (
	DlgSettingsTitle = "Settings"
	// LabelAskConfirmExit is the m_chkAskConfirm caption.
	LabelAskConfirmExit = "Ask confirmation when closing the application."
	// LabelAskSaveChanges is the m_chkAskSaveChanges caption.
	LabelAskSaveChanges = "Ask save changes when switching to another device."
)

// Preferences are the two options DlgSettings edits.
//
// ports DlgSettings (ACEServiceInstaller/ICUServiceInstaller/DlgSettings.cs)
type Preferences struct {
	AskConfirmExit bool // Settings.AskConfirmExit
	AskSaveChanges bool // Settings.AskSaveChanges
}

// Preferences returns the values DlgSettings initialises its check boxes
// with (the DlgSettings constructor).
//
// ports DlgSettings..ctor (ACEServiceInstaller/ICUServiceInstaller/DlgSettings.cs)
func (st *Store) Preferences() Preferences {
	st.mu.RLock()
	defer st.mu.RUnlock()
	return Preferences{AskConfirmExit: st.s.AskConfirmExit, AskSaveChanges: st.s.AskSaveChanges}
}

// SetPreferences ports DlgSettings.OnCommandActivated(Command.Ok): both
// values are stored and saved. Note: in C# closing the dialog with the window
// button responds Ok without running OnCommandActivated, so nothing is saved
// then; only the Ok button saves.
//
// ports DlgSettings.OnCommandActivated (ACEServiceInstaller/ICUServiceInstaller/DlgSettings.cs)
func (st *Store) SetPreferences(p Preferences) error {
	return st.Update(func(s *Settings) {
		s.AskConfirmExit = p.AskConfirmExit
		s.AskSaveChanges = p.AskSaveChanges
	})
}

// Texts of the exit confirmation (MainWindow.HandleCloseRequested).
const (
	// MsgUnsavedChanges is (sic) the primary text when a device has unsaved
	// property changes.
	MsgUnsavedChanges = "You have unsaved changed!"
	// MsgConfirmClose is the close question.
	MsgConfirmClose = "Are you sure you want to close this application?"
)

// ExitPrompt describes what MainWindow.HandleCloseRequested asks.
//
// ports MainWindow.HandleCloseRequested (ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs)
type ExitPrompt struct {
	// Ask is false when the window may close without a question.
	Ask bool
	// Primary / Secondary are MessageDialog.AskQuestion's texts (Secondary
	// empty for the one-text overload). Buttons: Yes, No, Cancel; only Yes
	// closes.
	Primary   string
	Secondary string
}

// ExitPrompt ports MainWindow.HandleCloseRequested's decision:
//
//   - no installer user logged on (m_currentUser == null): close, no question;
//   - AskConfirmExit and a selected device with property changes:
//     AskQuestion("You have unsaved changed!", "Are you sure ...?");
//   - AskConfirmExit otherwise: AskQuestion("Are you sure ...?");
//   - AskConfirmExit off: close (CloseRequestedEventArgs.AllowClose defaults
//     to true).
//
// When the window closes the C# logs out of the current device first.
//
// ports MainWindow.HandleCloseRequested (ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs)
func (p Preferences) ExitPrompt(userLoggedOn, deviceSelected, propertyChanges bool) ExitPrompt {
	if !userLoggedOn || !p.AskConfirmExit {
		return ExitPrompt{}
	}
	if deviceSelected && propertyChanges {
		return ExitPrompt{Ask: true, Primary: MsgUnsavedChanges, Secondary: MsgConfirmClose}
	}
	return ExitPrompt{Ask: true, Primary: MsgConfirmClose}
}

// PendingChangesAction is what happens to unsaved property changes when the
// user selects another device (MainWindow.OnDeviceSelectionChanged).
//
// ports MainWindow.OnDeviceSelectionChanged (ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs)
type PendingChangesAction int

const (
	// PendingNone: no changes (or AllowStoreChanges false) — nothing to do.
	PendingNone PendingChangesAction = iota
	// PendingAsk: ask SaveChangesQuestion (Yes, No, Cancel); Yes saves all
	// changes, any other answer reverts them.
	PendingAsk
	// PendingRevert: AskSaveChanges is off — the changes are reverted
	// silently (the C# never saves without asking).
	PendingRevert
)

// PendingChangesOnSwitch ports the m_fPropertyChanges branch of
// MainWindow.OnDeviceSelectionChanged:
//
//	if (m_fPropertyChanges && AllowStoreChanges) {
//	    if (AskSaveChanges && AskQuestion(...) == Yes) SaveAllChanges(); else RevertAllChanges();
//	}
//
// Afterwards the C# clears the change flag and logs out of the old device.
//
// ports MainWindow.OnDeviceSelectionChanged (ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs)
func (p Preferences) PendingChangesOnSwitch(propertyChanges, allowStoreChanges bool) PendingChangesAction {
	if !propertyChanges || !allowStoreChanges {
		return PendingNone
	}
	if p.AskSaveChanges {
		return PendingAsk
	}
	return PendingRevert
}

// SaveChangesQuestion returns the OnDeviceSelectionChanged question for the
// device whose Identity has unsaved changes.
//
// ports MainWindow.OnDeviceSelectionChanged (ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs)
func SaveChangesQuestion(identity string) string {
	return "You have changed some properties for device '" + identity + "', do you want to save the changes?"
}
