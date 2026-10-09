package ui

import (
	"net"
	"strconv"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"alfen/aceclient/internal/scn"
	"alfen/aceclient/internal/ui/core"
)

// PanelIDDiscover is the discovery page's registry ID.
const PanelIDDiscover = "discover"

// refreshDebounce is MainWindow.m_timRefreshDevices (1000 ms, one-shot,
// restarted on every collection change).
const refreshDebounce = 1000 * time.Millisecond

func init() {
	RegisterPanel(PanelSpec{
		ID:    PanelIDDiscover,
		Title: "Devices",
		Order: OrderDiscover,
		Mode:  PanelModeAlways,
		Build: buildDiscover,
	})
	RegisterMenuItem(MenuItemSpec{
		Menu: "Device", Label: "Refresh", Order: 700,
		Action: func(s *State) {
			if v, ok := s.view(PanelIDDiscover).(*discoverView); ok {
				v.refreshLAN()
			}
		},
	})
}

// discoverColumns are the columns of the device table: the UIListLanDevice
// fields SCN data can fill (Identification, Address, SCN group) plus the SCN
// telemetry.
var discoverColumns = []string{"Identity", "IP address", "SCN network", "Sockets", "State", "Last seen"}

type discoverView struct {
	s       *State
	table   *widget.Table
	empty   *widget.Label
	filter  *widget.Entry
	listen  *widget.Button
	connect *widget.Button
	rows    []core.Device
	sel     int

	mu       sync.Mutex
	listener *scn.Listener
	timer    *time.Timer
}

// buildDiscover ports the device tree of MainWindow (filter box, device list,
// Refresh button) fed by internal/scn instead of mDNS/ARP. Selecting a device
// fills the connection bar; "Connect" logs in to the selection.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs:110-330, 1404-1502
func buildDiscover(s *State) fyne.CanvasObject {
	v := &discoverView{s: s, sel: -1}
	s.setView(PanelIDDiscover, v)

	v.filter = widget.NewEntry()
	v.filter.SetPlaceHolder(core.DeviceFilterTooltip)
	v.filter.OnChanged = func(string) { v.refreshList() } // OnDeviceFilterChanged

	v.listen = widget.NewButton("Start listening", v.toggleListen)
	refresh := widget.NewButton("Refresh", v.refreshLAN) // "Refresh. Search for attached devices."
	v.connect = widget.NewButton("Connect", func() {
		if v.sel >= 0 && v.sel < len(v.rows) {
			s.SetConnectionTarget(v.rows[v.sel])
			s.connectFromBar()
		}
	})
	v.connect.Disable()

	v.table = widget.NewTableWithHeaders(
		func() (int, int) { return len(v.rows), len(discoverColumns) },
		func() fyne.CanvasObject {
			l := widget.NewLabel("")
			l.Truncation = fyne.TextTruncateEllipsis
			return l
		},
		func(id widget.TableCellID, o fyne.CanvasObject) {
			if id.Row < 0 || id.Row >= len(v.rows) {
				return
			}
			o.(*widget.Label).SetText(discoverCell(v.rows[id.Row], id.Col))
		})
	v.table.ShowHeaderColumn = false
	v.table.CreateHeader = func() fyne.CanvasObject {
		return widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	}
	v.table.UpdateHeader = func(id widget.TableCellID, o fyne.CanvasObject) {
		if id.Col >= 0 && id.Col < len(discoverColumns) {
			o.(*widget.Label).SetText(discoverColumns[id.Col])
		}
	}
	for i, w := range []float32{180, 130, 140, 70, 220, 90} {
		v.table.SetColumnWidth(i, w)
	}
	v.table.OnSelected = func(id widget.TableCellID) { v.selectRow(id.Row) }

	v.empty = widget.NewLabel(core.NoDevicesText)
	v.empty.Wrapping = fyne.TextWrapWord

	s.OnClose(v.stopListening)
	if s.opts.AutoDiscover {
		v.startListening()
	}
	v.refreshList()

	top := container.NewBorder(nil, nil, widget.NewLabel("Filter:"),
		container.NewHBox(v.listen, refresh, v.connect), v.filter)
	return container.NewBorder(top, v.empty, nil, nil, v.table)
}

// discoverCell renders one table cell.
func discoverCell(d core.Device, col int) string {
	switch col {
	case 0:
		return d.Identity
	case 1:
		if d.Discovered {
			return d.Address
		}
		return d.Address + " (manual)"
	case 2:
		return d.SCNNetwork
	case 3:
		if d.TotalSockets > 0 {
			return strconv.Itoa(d.TotalSockets)
		}
		return ""
	case 4:
		return d.SocketSummary()
	case 5:
		if d.LastSeen.IsZero() {
			return ""
		}
		return d.LastSeen.Local().Format("15:04:05")
	}
	return ""
}

func (v *discoverView) selectRow(row int) {
	if row < 0 || row >= len(v.rows) {
		return
	}
	v.sel = row
	v.connect.Enable()
	v.s.SetConnectionTarget(v.rows[row])
}

// refreshList ports MainWindow.RefreshDeviceList: re-order and filter the
// collection, keeping the selected device selected.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/MainWindow.cs:1404-1502
func (v *discoverView) refreshList() {
	var selAddr string
	if v.sel >= 0 && v.sel < len(v.rows) {
		selAddr = v.rows[v.sel].Address
	}
	v.rows = v.s.Devices.Rows(v.filter.Text)
	v.sel = -1
	for i, d := range v.rows {
		if d.Address == selAddr {
			v.sel = i
		}
	}
	if v.sel < 0 {
		v.connect.Disable()
		v.table.UnselectAll()
	}
	if v.s.Devices.Len() == 0 {
		v.empty.Show()
	} else {
		v.empty.Hide()
	}
	v.table.Refresh()
}

// scheduleRefresh restarts the 1 s one-shot refresh timer
// (OnDevicesCollectionChanged). Safe from any goroutine.
func (v *discoverView) scheduleRefresh() {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.timer != nil {
		v.timer.Stop()
	}
	v.timer = time.AfterFunc(refreshDebounce, func() { v.s.uiDo(v.refreshList) })
}

// RefreshDeviceList schedules a redraw of the device table after State.Devices
// changed (MainWindow.OnDevicesCollectionChanged → 1 s debounce →
// RefreshDeviceList). Discovery sources other than SCN (mDNS, manual add)
// call it after Devices.Upsert/Remove. Safe from any goroutine.
func (s *State) RefreshDeviceList() {
	if v, ok := s.view(PanelIDDiscover).(*discoverView); ok {
		v.scheduleRefresh()
	}
}

// refreshLAN ports MainWindow.OnRefreshLAN: drop every device and redraw.
func (v *discoverView) refreshLAN() {
	v.s.Devices.Clear()
	v.refreshList()
}

func (v *discoverView) toggleListen() {
	v.mu.Lock()
	running := v.listener != nil
	v.mu.Unlock()
	if running {
		v.stopListening()
	} else {
		v.startListening()
	}
}

// startListening binds the SCN UDP port (scn.Listen, IPAddress.Any:36549) and
// feeds every decoded datagram into the device list from a background
// goroutine; the table refresh is debounced onto the UI goroutine.
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/SCNNetwork.cs (StartUdpReceiveTask)
func (v *discoverView) startListening() {
	v.mu.Lock()
	if v.listener != nil {
		v.mu.Unlock()
		return
	}
	v.mu.Unlock()
	var l *scn.Listener
	var err error
	v.s.RunAsync(func() { l, err = scn.Listen() }, func() {
		if err != nil {
			v.s.SetStatus("SCN discovery unavailable on udp/%d: %v", scn.UDPPort, err)
			return
		}
		v.mu.Lock()
		v.listener = l
		v.mu.Unlock()
		v.listen.SetText("Stop listening")
		v.s.SetStatus("Listening for SCN broadcasts on udp/%d...", scn.UDPPort)
		go func() {
			_ = l.Run(func(sock *scn.Socket, from net.IP) {
				v.s.Devices.UpsertSCN(sock, from, time.Now())
				v.scheduleRefresh()
			}, time.Time{})
			v.s.uiDo(func() {
				v.mu.Lock()
				if v.listener == l {
					v.listener = nil
				}
				v.mu.Unlock()
				v.listen.SetText("Start listening")
			})
		}()
	})
}

// stopListening closes the SCN socket; Run then returns.
func (v *discoverView) stopListening() {
	v.mu.Lock()
	l := v.listener
	v.listener = nil
	if v.timer != nil {
		v.timer.Stop()
	}
	v.mu.Unlock()
	if l != nil {
		_ = l.Close()
		v.listen.SetText("Start listening")
		v.s.SetStatus("SCN discovery stopped.")
	}
}
