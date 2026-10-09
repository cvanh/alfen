package fwucreator

import "fmt"

// DisplayObjectNew ports ICUServiceInstaller.DisplayObjectNew
// (ACEServiceInstaller/ICUServiceInstaller/DisplayObjectNew.cs): the text and
// icon PanelMonitoring shows for a user-interface error code.
type DisplayObjectNew struct {
	ErrorCode UserInterfaceError
	Icon      StatusIcon
	Text      string
}

// DisplayObjectOld ports ICUServiceInstaller.DisplayObjectOld
// (ACEServiceInstaller/ICUServiceInstaller/DisplayObjectOld.cs): the text and
// icon PanelMonitoring shows for a socket main state on older firmware.
type DisplayObjectOld struct {
	MainState MainState
	Icon      StatusIcon
	Text      string
}

// DisplayStatesMessagesOld ports PanelMonitoring.TDisplayStatesMessagesOld
// (ACEServiceInstaller/ICUServiceInstaller/PanelMonitoring.cs:177-195).
var DisplayStatesMessagesOld = []DisplayObjectOld{
	{StateAvailable, StatusIconValid, "Installation OK"},
	{StateError, StatusIconWarning, "001: Not able to charge."},
	{StateErrorMessage, StatusIconWarning, "002: Charging not started yet,\nto continue please reconnect cable."},
	{StateErrorIllegalMode3, StatusIconWarning, "201: No communication with vehicle.\nPlease check your charging cable"},
	{StateErrorTooManyRestarts, StatusIconError, "003: Too many retries.\nPlease check your charging cable"},
	{StateErrorCharging, StatusIconWarning, "004: One moment please...\nYour charging session will resume shortly."},
	{StateErrorChargingOvercurrent, StatusIconWarning, "005: One moment please...\nYour charging session will resume shortly."},
	{StateErrorChargingHfContactorSwitching, StatusIconWarning, "006: One moment please...\nYour charging session will resume shortly."},
	{StateErrorPowermeter, StatusIconError, "202: Not able to charge."},
	{StateErrorTemperature, StatusIconWarning, "203: Inside temperature high.\nCharging will resume shortly."},
	{StateInoperative, StatusIconWarning, "204: Temporary set to unavailable."},
	{StateErrorS2NotOpened, StatusIconError, "007: S2 not opened.\nPlease reconnect cable."},
	{StateErrorProtectiveEarth, StatusIconError, "101: Error in installation.\nPlease Check installation"},
	{StateErrorRelays, StatusIconError, "102: Not able to Charge"},
	{StateErrorLowSupplyVoltage, StatusIconError, "103: Input Voltage too low,\nnot able to charge."},
	{StateErrorInternalVoltage, StatusIconError, "104: Not able to charge."},
}

// DisplayStatesMessagesNew ports PanelMonitoring.TDisplayStatesMessagesNew
// (PanelMonitoring.cs:196-227).
var DisplayStatesMessagesNew = []DisplayObjectNew{
	{UIErrorNone, StatusIconValid, "Installation OK"},
	{UIErrorGeneric, StatusIconWarning, "Not able to charge.\n Please call for support"},
	{UIErrorChargingRcd, StatusIconWarning, "One moment please...\nYour charging session will resume shortly"},
	{UIErrorRelays, StatusIconError, "Not able to charge.\n Please call for support"},
	{UIErrorInternalVoltage, StatusIconError, "Not able to charge.\n Please call for support"},
	{UIErrorPowermeter, StatusIconError, "Not able to charge.\n Please call for support"},
	{UIErrorRcd, StatusIconError, "Not able to charge.\n Please call for support"},
	{UIErrorSocketMotorStartupOld, StatusIconError, "Not able to lock cable\nPlease call for support"},
	{UIErrorMissingpcid, StatusIconValid, ""},
	{UIErrorNfcreader, StatusIconValid, ""},
	{UIErrorProtectiveEarth, StatusIconError, "Error in installation.\nPlease Check installation or call for support"},
	{UIErrorLowSupplyVoltage, StatusIconError, "Input Voltage too low, not able to charge.\nPlease call your installer"},
	{UIErrorInoperative, StatusIconWarning, "Temporary set to unavailable.\nContact CPO or try again later"},
	{UIErrorHighSupplyVoltage, StatusIconValid, "Error high voltage"},
	{UIErrorP1pport, StatusIconValid, "Error P1 port"},
	{UIErrorModbustcpip, StatusIconValid, "Error ModbusTCPIP port"},
	{UIErrorSocketMotorStartup, StatusIconError, "Not able to lock cable\nPlease call for support"},
	{UIErrorMissingphase, StatusIconError, "Error in installation.\nPlease Check installation or call for support"},
	{UIErrorTicport, StatusIconError, "Error TIC port"},
	{UIErrorCharging, StatusIconWarning, "One moment please...\nYour charging session will resume shortly."},
	{UIErrorChargingOvercurrent, StatusIconWarning, "One moment please...\nYour charging session will resume shortly."},
	{UIErrorChargingHfSwitching, StatusIconWarning, "Charging not started yet,\nto continue please reconnect cable."},
	{UIErrorCableConnectedTimeout, StatusIconWarning, "Charging not started yet\nto continue please reconnect cable."},
	{UIErrorTemperatureHigh, StatusIconWarning, "Inside temperature high.\nCharging will resume shortly."},
	{UIErrorTemperatureLow, StatusIconWarning, "Inside temperature low. Charging will resume shortly."},
	{UIErrorMessage, StatusIconWarning, "Charging not started yet,\nto continue please reconnect cable."},
	{UIErrorSocketMotor, StatusIconWarning, "Not able to lock cable.\nPlease reconnect cable."},
	{UIErrorIllegalMode3Pp, StatusIconWarning, "Cable not supported\nPlease try connecting your cable again "},
	{UIErrorIllegalMode3Cp, StatusIconWarning, "No communication with vehicle.\n Please check your charging cable"},
	{UIErrorTilt, StatusIconValid, ""},
}

// StatusMessageNew ports the user-interface-state branch of
// PanelMonitoring.SetStatusMessages (PanelMonitoring.cs:640-663): uiState is property 0x3190/0x3191 sub 1
// (12688/12689), uiError sub 2. Only UI_STATE_ERROR uses the error code; an
// unknown code falls back to a warning. The returned text is
// $"{(int)ErrorCode:000}: {Text}".
func StatusMessageNew(uiState UserInterfaceState, uiError UserInterfaceError) (string, StatusIcon) {
	errorNumber := UIErrorNone
	if uiState == UIStateError {
		errorNumber = uiError
	}
	obj := DisplayObjectNew{errorNumber, StatusIconWarning, "Unknown Error/Warning state \nPlease call for support"}
	for _, s := range DisplayStatesMessagesNew {
		if s.ErrorCode == errorNumber {
			obj = s
			break
		}
	}
	return fmt.Sprintf("%03d: %s", int(obj.ErrorCode), obj.Text), obj.Icon
}

// StatusMessageOld ports the legacy branch of PanelMonitoring.SetStatusMessages
// (PanelMonitoring.cs:664-675)
// (socket main state, property 0x2501/0x2502 sub 1, i.e. 9473/9474): the table
// text, or $"{socketState} \nUnknown Error/Warning state" with a warning icon.
func StatusMessageOld(socketState MainState) (string, StatusIcon) {
	for _, s := range DisplayStatesMessagesOld {
		if s.MainState == socketState {
			return s.Text, s.Icon
		}
	}
	return fmt.Sprintf("%v \nUnknown Error/Warning state", socketState), StatusIconWarning
}
