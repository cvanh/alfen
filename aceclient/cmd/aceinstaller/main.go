// Command aceinstaller is the cross-platform Fyne GUI of the Go port of the
// Alfen ACE Service Installer (ACEServiceInstaller.exe). It opens the
// "ACE Service Installer" main window (internal/ui), whose pages are
// registered by the files of internal/ui. The CGO-free CLI stays in
// cmd/aceclient.
package main

import "alfen/aceclient/internal/ui"

func main() {
	a := ui.NewApp()
	s := ui.NewMainWindow(a, ui.Options{AutoDiscover: true})
	s.Window.ShowAndRun()
}
