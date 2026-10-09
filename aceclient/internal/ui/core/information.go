package core

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// InfoRow is one line of the Information panel (PanelInformation): a category
// heading, a small header (AddSmallHeader), an info text (AddInfoText) or a
// labelled value bound to a property or computed (AddCustomText).
type InfoRow struct {
	Category string // UIConfigCategory ("General", "Sub devices", …)
	Header   bool   // AddSmallHeader
	Info     bool   // AddInfoText
	Label    string
	Value    string
	ID       uint16 // 0 for computed rows
	Sub      byte
	Editable bool // AddText/AddCheckBox/AddSelect (editable in the C#)
}

// Labels taken from EDS.xml <Title> for controls PanelInformation adds
// without an override label; used when no EDS lookup is wired.
var informationEDSTitles = map[uint32]string{
	Key(8273, 0): "Object Number",          // 0x2051 sysChargePointSerialNumber
	Key(8275, 0): "Customer Ident. number", // 0x2053 sysChargeBoxIdentity
}

// ParseRevision ports PanelInformation.ParseRevision: the names of
// EBoardRevision/EBoardAssy split on '_' ("BOARDREVISION_C" → "C",
// "BOARDASSEMBLY_02" → "02") joined by '-'. Values outside both enums make the
// C# throw (IndexOutOfRange on the split); "N/A" is returned instead.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/PanelInformation.cs:552-557, firmware/decompiled/ACENetwork/ICUNetwork/EBoardRevision.cs, EBoardAssy.cs
func ParseRevision(revision, assembly int) string {
	revNames := []string{"DEFAULT", "A", "B", "C", "D", "E", "F", "G", "H", "J", "K", "L", "M", "N", "P", "Q", "R"}
	rev := ""
	switch {
	case revision == -1:
		rev = "UNKNOWN"
	case revision >= 0 && revision < len(revNames):
		rev = revNames[revision]
	default:
		return "N/A"
	}
	asm := ""
	switch {
	case assembly == -1:
		asm = "UNKNOWN"
	case assembly == 0:
		asm = "DEFAULT"
	case assembly >= 1 && assembly <= 16:
		asm = fmt.Sprintf("%02d", assembly-1)
	default:
		return "N/A"
	}
	return rev + "-" + asm
}

// NFCVersion ports ICULanDevice.GetNFCVersion(device, reader, software).
// Before firmware 4.3.0 reader 1 parses 8276 ("…#N:hw;sw"); otherwise
// 12672/12673 hold "x:hw,y:sw" pairs. An entry without ':' makes the C#
// throw; "N/A" is returned instead.
//
// Source: firmware/decompiled/ACENetwork/ICUNetwork/ICULanDevice.cs:2660-2699
func NFCVersion(pc *PropertyCache, fw Version, reader int, software bool) string {
	idx := 0
	if software {
		idx = 1
	}
	if fw.Less(MustVersion("4.3.0")) && reader == 1 {
		res := pc.String(8276, 0)
		if !strings.Contains(res, "#N:") {
			return "N/A"
		}
		num := strings.LastIndex(res, "#N:") + 3
		text := ""
		if num < len(res) {
			text = res[num:]
		}
		parts := strings.Split(text, ";")
		if len(parts) > idx {
			return parts[idx]
		}
		return "N/A"
	}
	var res string
	if reader == 1 {
		res = pc.String(12672, 0)
	} else {
		res = pc.String(12673, 0)
	}
	if res == "" {
		return "N/A"
	}
	parts := strings.Split(res, ",")
	if len(parts) <= idx {
		return "N/A"
	}
	kv := strings.Split(parts[idx], ":")
	if len(kv) < 2 {
		return "N/A"
	}
	return kv[1]
}

// DeviceUTCOffsetMinutes ports the time-zone selection of the DateTime
// renderers: 8302 (minutes) when the device has it, else 8282 * 6.
func DeviceUTCOffsetMinutes(pc *PropertyCache) int {
	if p, ok := pc.Get(8302, 0); ok && p.HasValue {
		return pc.Int(8302, 0)
	}
	return pc.Int(8282, 0) * 6
}

// FormatDeviceDateTime ports UIPropertyReadOnlyString/UIPropertyString's
// UIPropertyStringType.DateTime rendering: value is milliseconds since the
// Unix epoch; below 946684800 s it is a duration (TimeSpan.ToString()),
// otherwise a date converted to the device's UTC offset and printed with the
// en-US ToLongDateString() + " " + ToLongTimeString(). The C# looks up a
// Windows time zone with that base offset (and applies its DST rules); a
// fixed offset is used here.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/UIPropertyReadOnlyString.cs:125-143, UIPropertyBase.cs:452-461
func FormatDeviceDateTime(value string, utcOffsetMinutes int) string {
	n, ok := parseDotNetUint(value, 64)
	if !ok {
		n = 0
	}
	secs := n / 1000
	if secs < 946684800 {
		return formatTimeSpan(secs)
	}
	t := time.Unix(int64(secs), 0).In(time.FixedZone("", utcOffsetMinutes*60))
	return t.Format("Monday, January 2, 2006") + " " + t.Format("3:04:05 PM")
}

// formatTimeSpan mirrors TimeSpan.ToString() for whole seconds:
// "[d.]hh:mm:ss".
func formatTimeSpan(secs uint64) string {
	d := secs / 86400
	h := (secs / 3600) % 24
	m := (secs / 60) % 60
	s := secs % 60
	if d > 0 {
		return fmt.Sprintf("%d.%02d:%02d:%02d", d, h, m, s)
	}
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

// FormatUTCOffset renders a time-zone offset the way Windows display names
// start ("(UTC+01:00)"); the C# shows TimeZoneInfo.DisplayName, which has no
// portable equivalent.
func FormatUTCOffset(minutes int) string {
	if minutes == 0 {
		return "(UTC)"
	}
	sign := "+"
	if minutes < 0 {
		sign = "-"
		minutes = -minutes
	}
	return fmt.Sprintf("(UTC%s%02d:%02d)", sign, minutes/60, minutes%60)
}

// InformationContext carries the device facts PanelInformation branches on.
type InformationContext struct {
	IsAHP                    bool
	IsAHPV2                  bool
	IsEcogDC                 bool
	NumberOfSockets          int
	FirmwareVersion          Version
	IsUniquePasswordRequired bool
}

// InformationContextFor gathers InformationContext from a session.
func InformationContextFor(s *Session) InformationContext {
	host := s.HostName()
	ctx := InformationContext{
		IsAHP:                    IsAHPHostName(host),
		IsAHPV2:                  IsAHPV2HostName(host),
		NumberOfSockets:          s.NumberOfSockets(),
		FirmwareVersion:          s.FirmwareVersion(),
		IsUniquePasswordRequired: s.IsUniquePasswordRequired(),
	}
	// ICULanDevice.isEcogDC
	pc := s.Props
	ctx.IsEcogDC = ctx.IsAHPV2 && pc.Has(33034, 1) && (pc.UInt(33034, 1) == 1 || pc.UInt(33034, 2) == 1)
	return ctx
}

// InformationRows ports PanelInformation.OnChangeDevice as data: the rows of
// the General, Sub devices, Modem Info, License key, Location, Station
// Password and Experimental categories, with the same conditions and labels.
//
// Not ported here (need other packages): the feature list under "License
// key" (IWSFirmwareFeatures, internal/isah), "Eve Connect access"
// (EndUserAccessType from mDNS), the buttons and the 1 s clock tick.
//
// Source: firmware/decompiled/ACEServiceInstaller/ICUServiceInstaller/PanelInformation.cs:290-498
func InformationRows(pc *PropertyCache, ctx InformationContext) []InfoRow {
	var rows []InfoRow
	cat := ""
	category := func(name string) { cat = name }
	header := func(label string) { rows = append(rows, InfoRow{Category: cat, Header: true, Label: label}) }
	info := func(text string) { rows = append(rows, InfoRow{Category: cat, Info: true, Value: text}) }
	label := func(id uint16, sub byte, override string) string {
		if override != "" {
			return override
		}
		if e, ok := pc.Lookup(id, sub); ok {
			if e.Units != "" {
				return fmt.Sprintf("%s (%s)", e.Title, e.Units)
			}
			return e.Title
		}
		if t, ok := informationEDSTitles[Key(id, sub)]; ok {
			return t
		}
		if sub == 0 {
			return fmt.Sprintf("%04X", id)
		}
		return fmt.Sprintf("%04X_%d", id, sub)
	}
	prop := func(id uint16, sub byte, override string, editable bool) {
		rows = append(rows, InfoRow{Category: cat, Label: label(id, sub, override), Value: TrimDisplayString(pc.String(id, sub)), ID: id, Sub: sub, Editable: editable})
	}
	dateProp := func(id uint16, sub byte, override string) {
		v := ""
		if p, ok := pc.Get(id, sub); ok && p.HasValue {
			v = FormatDeviceDateTime(p.Value, DeviceUTCOffsetMinutes(pc))
		}
		rows = append(rows, InfoRow{Category: cat, Label: label(id, sub, override), Value: v, ID: id, Sub: sub})
	}
	custom := func(l, v string) { rows = append(rows, InfoRow{Category: cat, Label: l, Value: v}) }
	yesNo := func(b bool) string {
		if b {
			return "Yes"
		}
		return "No"
	}

	category("General")
	header("Identification")
	prop(8272, 0, "Model", false)
	prop(8273, 0, "", false)
	prop(8275, 0, "", true)
	prop(8277, 0, "Charge point vendor", false)
	header("Information")
	if pc.Has(8583, 0) {
		dateProp(8583, 0, "Last time Configuration Changed")
	}
	prop(4104, 0, "Platform type", false)
	if ctx.IsAHP {
		custom("Hardware version SCB", ParseRevision(pc.Int(8269, 1), pc.Int(8269, 2)))
		prop(4106, 0, "Software version SCB", false)
	} else {
		custom("Hardware version controller board", ParseRevision(pc.Int(8269, 1), pc.Int(8269, 2)))
		custom("Hardware version power board", ParseRevision(pc.Int(8269, 3), pc.Int(8269, 4)))
		prop(4106, 0, "Software version controller board", false)
		if pc.Has(12674, 0) {
			prop(12674, 0, "Bootloader version controller board", false)
		}
	}
	custom("WiFi supported", yesNo(pc.Has(12943, 0) && pc.Bool(12943, 0, false)))
	custom("Tamper detection supported", yesNo(pc.Bool(8785, 1, false) || pc.Int(8784, 0) != 0))

	category("Sub devices")
	if !ctx.IsAHP {
		if ctx.FirmwareVersion.AtLeast(MustVersion("4.3.0")) {
			custom("NFC-RFID reader 1 hardw. version", NFCVersion(pc, ctx.FirmwareVersion, 1, false))
			custom("NFC-RFID reader 1 softw. version", NFCVersion(pc, ctx.FirmwareVersion, 1, true))
			if ctx.NumberOfSockets > 1 {
				custom("NFC-RFID reader 2 hardw. version", NFCVersion(pc, ctx.FirmwareVersion, 2, false))
				custom("NFC-RFID reader 2 softw. version", NFCVersion(pc, ctx.FirmwareVersion, 2, true))
			}
		} else {
			custom("NFC-RFID reader hardware version", NFCVersion(pc, ctx.FirmwareVersion, 1, false))
			custom("NFC-RFID reader software version", NFCVersion(pc, ctx.FirmwareVersion, 1, true))
		}
	} else {
		for b := 1; b <= ctx.NumberOfSockets; b++ {
			sb := byte(b)
			header("SocketBoard " + strconv.Itoa(b))
			prop(33025, sb, "Device Id", false)
			prop(33026, sb, "Hardware version", false)
			prop(33027, sb, "Software version", false)
			if pc.Has(33794, sb) {
				prop(33794, sb, "Extended software info", false)
			}
			prop(33032, sb, "Energymeter info", false)
			prop(33031, sb, "ISO15118 info", false)
		}
		if !ctx.IsAHPV2 {
			header("NFC reader")
			prop(33281, 0, "Device Id", false)
			prop(33282, 0, "Hardware version", false)
			prop(33283, 0, "Software version", false)
		}
		if ctx.IsAHPV2 && pc.Has(33537, 0) && !ctx.IsEcogDC {
			header("Auxiliary board")
			prop(33537, 0, "Device Id", false)
			prop(33538, 0, "Hardware version", false)
			prop(33539, 0, "Software version", false)
		}
		if pc.Has(8785, 1) && pc.Bool(8785, 1, false) && pc.Has(33792, 1) {
			header("Tamper detection")
			prop(33792, 1, "Extension board assembly", false)
		}
	}

	if pc.Has(8472, 0) {
		category("Modem Info")
		prop(8472, 0, "Modem manufacturer", false)
		prop(8473, 0, "Modem model", false)
		prop(8480, 0, "Modem revision", false)
		prop(8481, 0, "Modem IMEI", false)
	}

	category("License key")
	prop(8273, 0, "", false)
	prop(8609, 0, "Feature license key", true)

	category("Location")
	dateProp(8281, 0, "Charger date and time")
	if p, ok := pc.Get(8302, 0); ok && p.HasValue {
		custom("Time zone", FormatUTCOffset(pc.Int(8302, 0)))
	} else {
		custom("Time zone", FormatUTCOffset(pc.Int(8282, 0)*6))
	}
	prop(8283, 0, "", true)
	prop(8284, 1, "Latitude", true)
	prop(8284, 2, "Longitude", true)

	if ctx.IsUniquePasswordRequired {
		category("Station Password")
		header("Secure Service Access")
		info("With Secure Service Access (SSA) enabled, an Alfen certified service person can\nconfigure this charging station without the need to share any password.\nThey can also regain access in case of a lost password.\n\nWe recommend enabling SSA in your charging station.")
		rows = append(rows, InfoRow{Category: cat, Label: "Enable Secure Service Access", Value: yesNo(CheckState(pc.String(8626, 0))), ID: 8626, Editable: true})
		header("Passwords")
		dateProp(8627, 0, "Temp password expiration date")
		if pc.Has(8628, 0) {
			prop(8628, 0, "Is default owner password", false)
		}
	}

	if pc.Has(9744, 0) {
		category("Experimental")
		header("Alpha release")
		info("When you enable the 'Allow alpha releases' checkbox,\nyou can install alpha release firmware version on this Charging Station.\nAlpha releases offer you earlier access to new features and improvements\nof upcoming major releases. Please be aware that alpha versions are not\nfinished products and might contain bugs!\n\nOnly enable the alpha release when you understand the risks!")
		rows = append(rows, InfoRow{Category: cat, Label: "Allow alpha releases", Value: yesNo(CheckState(pc.String(9744, 0))), ID: 9744, Editable: true})
	}
	return rows
}
