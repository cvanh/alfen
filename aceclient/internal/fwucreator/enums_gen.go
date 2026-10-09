// Code generated from the decompiled C# enums (see each type); DO NOT EDIT.

package fwucreator

// ObjectType ports ICUObjectTypes (ACEFWUCreator/ICUFWUCreator/ICUObjectTypes.cs).
// String returns the C# member name (or the decimal value when undefined, like Enum.ToString).
type ObjectType int

const (
	ObjectLogoCustomer      ObjectType = 0 // OBJECT_LOGO_CUSTOMER
	ObjectLogoAccepted      ObjectType = 1 // OBJECT_LOGO_ACCEPTED
	ObjectLogoCharging      ObjectType = 2 // OBJECT_LOGO_CHARGING
	ObjectLogoSocketerror   ObjectType = 3 // OBJECT_LOGO_SOCKETERROR
	ObjectLogoCommunicating ObjectType = 4 // OBJECT_LOGO_COMMUNICATING
	ObjectRoboticaRegular28 ObjectType = 5 // OBJECT_ROBOTICA_REGULAR_28
	ObjectRoboticaRegular29 ObjectType = 6 // OBJECT_ROBOTICA_REGULAR_29
	ObjectLanguage          ObjectType = 7 // OBJECT_LANGUAGE
	ObjectRoboticaRegular30 ObjectType = 8 // OBJECT_ROBOTICA_REGULAR_30
)

// objectTypeNames lists every ICUObjectTypes member in declaration order.
var objectTypeNames = []struct {
	Name  string
	Value ObjectType
}{
	{"OBJECT_LOGO_CUSTOMER", 0},
	{"OBJECT_LOGO_ACCEPTED", 1},
	{"OBJECT_LOGO_CHARGING", 2},
	{"OBJECT_LOGO_SOCKETERROR", 3},
	{"OBJECT_LOGO_COMMUNICATING", 4},
	{"OBJECT_ROBOTICA_REGULAR_28", 5},
	{"OBJECT_ROBOTICA_REGULAR_29", 6},
	{"OBJECT_LANGUAGE", 7},
	{"OBJECT_ROBOTICA_REGULAR_30", 8},
}

// ImageFormat ports ICUImageFormats (ACEFWUCreator/ICUFWUCreator/ICUImageFormats.cs).
// String returns the C# member name (or the decimal value when undefined, like Enum.ToString).
type ImageFormat int

const (
	ImageFormatL1     ImageFormat = 1  // FORMAT_L1
	ImageFormatL2     ImageFormat = 17 // FORMAT_L2
	ImageFormatL4     ImageFormat = 2  // FORMAT_L4
	ImageFormatL8     ImageFormat = 3  // FORMAT_L8
	ImageRgb332       ImageFormat = 4  // RGB332
	ImageRgb565       ImageFormat = 7  // RGB565
	ImagePaletted     ImageFormat = 8  // PALETTED
	ImagePaletted565  ImageFormat = 14 // PALETTED565
	ImagePaletted4444 ImageFormat = 15 // PALETTED4444
	ImagePaletted8    ImageFormat = 16 // PALETTED8
)

// imageFormatNames lists every ICUImageFormats member in declaration order.
var imageFormatNames = []struct {
	Name  string
	Value ImageFormat
}{
	{"FORMAT_L1", 1},
	{"FORMAT_L2", 17},
	{"FORMAT_L4", 2},
	{"FORMAT_L8", 3},
	{"RGB332", 4},
	{"RGB565", 7},
	{"PALETTED", 8},
	{"PALETTED565", 14},
	{"PALETTED4444", 15},
	{"PALETTED8", 16},
}

// DisplayString ports ICUDisplayStrings (ACEFWUCreator/ICUFWUCreator/ICUDisplayStrings.cs).
// String returns the C# member name (or the decimal value when undefined, like Enum.ToString).
type DisplayString int

const (
	DispEmpty                                 DisplayString = -1 // DISP_EMPTY
	DispChargePointBooting                    DisplayString = 0  // DISP_CHARGE_POINT_BOOTING
	DispPleaseHoldCardStart                   DisplayString = 1  // DISP_PLEASE_HOLD_CARD_START
	DispPleaseHoldCardStop                    DisplayString = 2  // DISP_PLEASE_HOLD_CARD_STOP
	DispPleaseRemoveCablePlugandcharge        DisplayString = 3  // DISP_PLEASE_REMOVE_CABLE_PLUGANDCHARGE
	DispCardAccepted                          DisplayString = 4  // DISP_CARD_ACCEPTED
	DispCardNotAccepted                       DisplayString = 5  // DISP_CARD_NOT_ACCEPTED
	DispPleaseHoldCardToUnlock                DisplayString = 6  // DISP_PLEASE_HOLD_CARD_TO_UNLOCK
	DispCableConnected                        DisplayString = 7  // DISP_CABLE_CONNECTED
	DispCableConnectedTimeout                 DisplayString = 8  // DISP_CABLE_CONNECTED_TIMEOUT
	DispPlugCableIntoSocket                   DisplayString = 9  // DISP_PLUG_CABLE_INTO_SOCKET
	DispEvConnected                           DisplayString = 10 // DISP_EV_CONNECTED
	DispWaitForEvconnect                      DisplayString = 11 // DISP_WAIT_FOR_EVCONNECT
	DispWaitForEvreconnect                    DisplayString = 12 // DISP_WAIT_FOR_EVRECONNECT
	DispCommunicatingWithVehicle              DisplayString = 13 // DISP_COMMUNICATING_WITH_VEHICLE
	DispCommunicatingWithVehicleSmartCharging DisplayString = 14 // DISP_COMMUNICATING_WITH_VEHICLE_SMART_CHARGING
	DispCharging                              DisplayString = 15 // DISP_CHARGING
	DispEndOfSession                          DisplayString = 16 // DISP_END_OF_SESSION
	DispChargingDuration                      DisplayString = 17 // DISP_CHARGING_DURATION
	DispChargingConsumption                   DisplayString = 18 // DISP_CHARGING_CONSUMPTION
	DispGoodbye                               DisplayString = 19 // DISP_GOODBYE
	DispError                                 DisplayString = 20 // DISP_ERROR
	DispReserved                              DisplayString = 21 // DISP_RESERVED
	DispTemporarilyOutOfOrder                 DisplayString = 22 // DISP_TEMPORARILY_OUT_OF_ORDER
	DispCableNotSupported                     DisplayString = 23 // DISP_CABLE_NOT_SUPPORTED
	DispCableNotSupportedRemoveCable          DisplayString = 24 // DISP_CABLE_NOT_SUPPORTED_REMOVE_CABLE
	DispTotal                                 DisplayString = 25 // DISP_TOTAL
	DispPlugCableIntoVehicle                  DisplayString = 26 // DISP_PLUG_CABLE_INTO_VEHICLE
	DispPleaseHoldCardStartLocked             DisplayString = 27 // DISP_PLEASE_HOLD_CARD_START_LOCKED
	DispPleaseHoldCardStartLocked2            DisplayString = 28 // DISP_PLEASE_HOLD_CARD_START_LOCKED2
	DispPlugCableIntoSocketUnlocked           DisplayString = 29 // DISP_PLUG_CABLE_INTO_SOCKET_UNLOCKED
	DispPlugCableIntoSocketUnlocked2          DisplayString = 30 // DISP_PLUG_CABLE_INTO_SOCKET_UNLOCKED2
	DispForFutureUse1                         DisplayString = 31 // DISP_FOR_FUTURE_USE_1
	DispForFutureUse2                         DisplayString = 32 // DISP_FOR_FUTURE_USE_2
	DispForFutureUse3                         DisplayString = 33 // DISP_FOR_FUTURE_USE_3
	DispForFutureUse4                         DisplayString = 34 // DISP_FOR_FUTURE_USE_4
	DispForFutureUse5                         DisplayString = 35 // DISP_FOR_FUTURE_USE_5
	DispForFutureUse6                         DisplayString = 36 // DISP_FOR_FUTURE_USE_6
	DispForFutureUse7                         DisplayString = 37 // DISP_FOR_FUTURE_USE_7
	DispForFutureUse8                         DisplayString = 38 // DISP_FOR_FUTURE_USE_8
	DispMasterTagMode                         DisplayString = 39 // DISP_MASTER_TAG_MODE
	DispMasterTagModeAdded                    DisplayString = 40 // DISP_MASTER_TAG_MODE_ADDED
	DispMasterTagModeRemoved                  DisplayString = 41 // DISP_MASTER_TAG_MODE_REMOVED
	DispStartMeterValue                       DisplayString = 42 // DISP_START_METER_VALUE
	DispActualMeterValue                      DisplayString = 43 // DISP_ACTUAL_METER_VALUE
	DispForFutureUse9                         DisplayString = 44 // DISP_FOR_FUTURE_USE_9
	DispForFutureUse10                        DisplayString = 45 // DISP_FOR_FUTURE_USE_10
	DispForFutureUse11                        DisplayString = 46 // DISP_FOR_FUTURE_USE_11
	DispPleaseWait                            DisplayString = 47 // DISP_PLEASE_WAIT
	DispDisclaimer                            DisplayString = 48 // DISP_DISCLAIMER
	DispCommunicatingWithVehicleSuspended     DisplayString = 49 // DISP_COMMUNICATING_WITH_VEHICLE_SUSPENDED
	DispIndexMax                              DisplayString = 50 // DISP_INDEX_MAX
)

// displayStringNames lists every ICUDisplayStrings member in declaration order.
var displayStringNames = []struct {
	Name  string
	Value DisplayString
}{
	{"DISP_EMPTY", -1},
	{"DISP_CHARGE_POINT_BOOTING", 0},
	{"DISP_PLEASE_HOLD_CARD_START", 1},
	{"DISP_PLEASE_HOLD_CARD_STOP", 2},
	{"DISP_PLEASE_REMOVE_CABLE_PLUGANDCHARGE", 3},
	{"DISP_CARD_ACCEPTED", 4},
	{"DISP_CARD_NOT_ACCEPTED", 5},
	{"DISP_PLEASE_HOLD_CARD_TO_UNLOCK", 6},
	{"DISP_CABLE_CONNECTED", 7},
	{"DISP_CABLE_CONNECTED_TIMEOUT", 8},
	{"DISP_PLUG_CABLE_INTO_SOCKET", 9},
	{"DISP_EV_CONNECTED", 10},
	{"DISP_WAIT_FOR_EVCONNECT", 11},
	{"DISP_WAIT_FOR_EVRECONNECT", 12},
	{"DISP_COMMUNICATING_WITH_VEHICLE", 13},
	{"DISP_COMMUNICATING_WITH_VEHICLE_SMART_CHARGING", 14},
	{"DISP_CHARGING", 15},
	{"DISP_END_OF_SESSION", 16},
	{"DISP_CHARGING_DURATION", 17},
	{"DISP_CHARGING_CONSUMPTION", 18},
	{"DISP_GOODBYE", 19},
	{"DISP_ERROR", 20},
	{"DISP_RESERVED", 21},
	{"DISP_TEMPORARILY_OUT_OF_ORDER", 22},
	{"DISP_CABLE_NOT_SUPPORTED", 23},
	{"DISP_CABLE_NOT_SUPPORTED_REMOVE_CABLE", 24},
	{"DISP_TOTAL", 25},
	{"DISP_PLUG_CABLE_INTO_VEHICLE", 26},
	{"DISP_PLEASE_HOLD_CARD_START_LOCKED", 27},
	{"DISP_PLEASE_HOLD_CARD_START_LOCKED2", 28},
	{"DISP_PLUG_CABLE_INTO_SOCKET_UNLOCKED", 29},
	{"DISP_PLUG_CABLE_INTO_SOCKET_UNLOCKED2", 30},
	{"DISP_FOR_FUTURE_USE_1", 31},
	{"DISP_FOR_FUTURE_USE_2", 32},
	{"DISP_FOR_FUTURE_USE_3", 33},
	{"DISP_FOR_FUTURE_USE_4", 34},
	{"DISP_FOR_FUTURE_USE_5", 35},
	{"DISP_FOR_FUTURE_USE_6", 36},
	{"DISP_FOR_FUTURE_USE_7", 37},
	{"DISP_FOR_FUTURE_USE_8", 38},
	{"DISP_MASTER_TAG_MODE", 39},
	{"DISP_MASTER_TAG_MODE_ADDED", 40},
	{"DISP_MASTER_TAG_MODE_REMOVED", 41},
	{"DISP_START_METER_VALUE", 42},
	{"DISP_ACTUAL_METER_VALUE", 43},
	{"DISP_FOR_FUTURE_USE_9", 44},
	{"DISP_FOR_FUTURE_USE_10", 45},
	{"DISP_FOR_FUTURE_USE_11", 46},
	{"DISP_PLEASE_WAIT", 47},
	{"DISP_DISCLAIMER", 48},
	{"DISP_COMMUNICATING_WITH_VEHICLE_SUSPENDED", 49},
	{"DISP_INDEX_MAX", 50},
}

// StatusIcon ports EStatusIcon (ACENetwork/ICUNetwork/EStatusIcon.cs).
// String returns the C# member name (or the decimal value when undefined, like Enum.ToString).
type StatusIcon int

const (
	StatusIconValid       StatusIcon = 0 // STATUS_ICON_VALID
	StatusIconInformation StatusIcon = 1 // STATUS_ICON_INFORMATION
	StatusIconWarning     StatusIcon = 2 // STATUS_ICON_WARNING
	StatusIconError       StatusIcon = 3 // STATUS_ICON_ERROR
)

// statusIconNames lists every EStatusIcon member in declaration order.
var statusIconNames = []struct {
	Name  string
	Value StatusIcon
}{
	{"STATUS_ICON_VALID", 0},
	{"STATUS_ICON_INFORMATION", 1},
	{"STATUS_ICON_WARNING", 2},
	{"STATUS_ICON_ERROR", 3},
}

// MainState ports EMainStates (ACENetwork/ICUNetwork/EMainStates.cs).
// String returns the C# member name (or the decimal value when undefined, like Enum.ToString).
type MainState int

const (
	StateIllegal                           MainState = -1 // STATE_ILLEGAL
	StateUnknown                           MainState = 0  // STATE_UNKNOWN
	StateBooting                           MainState = 1  // STATE_BOOTING
	StateAvailable                         MainState = 2  // STATE_AVAILABLE
	StateCableConnected                    MainState = 3  // STATE_CABLE_CONNECTED
	StateCableConnectedTimeout             MainState = 4  // STATE_CABLE_CONNECTED_TIMEOUT
	StateEvConnected                       MainState = 5  // STATE_EV_CONNECTED
	StateButtonActivated                   MainState = 6  // STATE_BUTTON_ACTIVATED
	StateNfcAvailable                      MainState = 7  // STATE_NFC_AVAILABLE
	StateNfcAuthorised                     MainState = 8  // STATE_NFC_AUTHORISED
	StateWaitForEvconnect                  MainState = 9  // STATE_WAIT_FOR_EVCONNECT
	StateChargingTestRelays                MainState = 10 // STATE_CHARGING_TEST_RELAYS
	StateChargingPowerOff                  MainState = 11 // STATE_CHARGING_POWER_OFF
	StateChargingPowerOffLowMaxcurrent     MainState = 12 // STATE_CHARGING_POWER_OFF_LOW_MAXCURRENT
	StateChargingPowerStarting             MainState = 13 // STATE_CHARGING_POWER_STARTING
	StateChargingPowerOn                   MainState = 14 // STATE_CHARGING_POWER_ON
	StateChargingPowerOnSimplified         MainState = 15 // STATE_CHARGING_POWER_ON_SIMPLIFIED
	StateChargingWaitForEvReconnect        MainState = 16 // STATE_CHARGING_WAIT_FOR_EV_RECONNECT
	StateChargingTerminating               MainState = 17 // STATE_CHARGING_TERMINATING
	StateChargingWakeup                    MainState = 18 // STATE_CHARGING_WAKEUP
	StateWaitForDisconnect                 MainState = 19 // STATE_WAIT_FOR_DISCONNECT
	StateWaitForReleaseAuthorisation       MainState = 20 // STATE_WAIT_FOR_RELEASE_AUTHORISATION
	StateChargingRecoverFromOutage         MainState = 21 // STATE_CHARGING_RECOVER_FROM_OUTAGE
	StateError                             MainState = 22 // STATE_ERROR
	StateErrorMessage                      MainState = 23 // STATE_ERROR_MESSAGE
	StateErrorMessageCableNotSupported     MainState = 24 // STATE_ERROR_MESSAGE_CABLE_NOT_SUPPORTED
	StateErrorIllegalMode3                 MainState = 25 // STATE_ERROR_ILLEGAL_MODE_3
	StateErrorTooManyRestarts              MainState = 26 // STATE_ERROR_TOO_MANY_RESTARTS
	StateErrorCharging                     MainState = 27 // STATE_ERROR_CHARGING
	StateErrorChargingOvercurrent          MainState = 28 // STATE_ERROR_CHARGING_OVERCURRENT
	StateErrorChargingHfContactorSwitching MainState = 29 // STATE_ERROR_CHARGING_HF_CONTACTOR_SWITCHING
	StateErrorS2NotOpened                  MainState = 30 // STATE_ERROR_S2_NOT_OPENED
	StateErrorProtectiveEarth              MainState = 31 // STATE_ERROR_PROTECTIVE_EARTH
	StateErrorRelays                       MainState = 32 // STATE_ERROR_RELAYS
	StateErrorLowSupplyVoltage             MainState = 33 // STATE_ERROR_LOW_SUPPLY_VOLTAGE
	StateErrorInternalVoltage              MainState = 34 // STATE_ERROR_INTERNAL_VOLTAGE
	StateErrorPowermeter                   MainState = 35 // STATE_ERROR_POWERMETER
	StateErrorTemperature                  MainState = 36 // STATE_ERROR_TEMPERATURE
	StateSuspended                         MainState = 37 // STATE_SUSPENDED
	StateInoperative                       MainState = 38 // STATE_INOPERATIVE
	StateReserved                          MainState = 39 // STATE_RESERVED
	StateErrorChargingRcdSignaled          MainState = 40 // STATE_ERROR_CHARGING_RCD_SIGNALED
	StateChargingPowerOffVentilating       MainState = 41 // STATE_CHARGING_POWER_OFF_VENTILATING
	StateChargingPowerOffSuspended         MainState = 42 // STATE_CHARGING_POWER_OFF_SUSPENDED
	StateChargingPowerOffPhaseChange       MainState = 43 // STATE_CHARGING_POWER_OFF_PHASE_CHANGE
	StateWaitForStartMetervalue            MainState = 44 // STATE_WAIT_FOR_START_METERVALUE
	StateWaitForStopMetervalue             MainState = 45 // STATE_WAIT_FOR_STOP_METERVALUE
	StateErrorSocketMotor                  MainState = 46 // STATE_ERROR_SOCKET_MOTOR
	StateCableConnectedTypeE               MainState = 47 // STATE_CABLE_CONNECTED_TYPE_E
	StateCableConnectedTimeoutTypeE        MainState = 48 // STATE_CABLE_CONNECTED_TIMEOUT_TYPE_E
	StateChargingTypeE                     MainState = 49 // STATE_CHARGING_TYPE_E
	StateWaitForDisconnectTypeE            MainState = 50 // STATE_WAIT_FOR_DISCONNECT_TYPE_E
	StateChargingSuspendedTypeE            MainState = 51 // STATE_CHARGING_SUSPENDED_TYPE_E
	StateChargingLowMaxcurrentTypeE        MainState = 52 // STATE_CHARGING_LOW_MAXCURRENT_TYPE_E
	StateInvalidCard                       MainState = 53 // STATE_INVALID_CARD
	StateEvConnectedUnauthorized           MainState = 54 // STATE_EV_CONNECTED_UNAUTHORIZED
	StateWaitForDisconnectPp               MainState = 55 // STATE_WAIT_FOR_DISCONNECT_PP
)

// mainStateNames lists every EMainStates member in declaration order.
var mainStateNames = []struct {
	Name  string
	Value MainState
}{
	{"STATE_ILLEGAL", -1},
	{"STATE_UNKNOWN", 0},
	{"STATE_BOOTING", 1},
	{"STATE_AVAILABLE", 2},
	{"STATE_CABLE_CONNECTED", 3},
	{"STATE_CABLE_CONNECTED_TIMEOUT", 4},
	{"STATE_EV_CONNECTED", 5},
	{"STATE_BUTTON_ACTIVATED", 6},
	{"STATE_NFC_AVAILABLE", 7},
	{"STATE_NFC_AUTHORISED", 8},
	{"STATE_WAIT_FOR_EVCONNECT", 9},
	{"STATE_CHARGING_TEST_RELAYS", 10},
	{"STATE_CHARGING_POWER_OFF", 11},
	{"STATE_CHARGING_POWER_OFF_LOW_MAXCURRENT", 12},
	{"STATE_CHARGING_POWER_STARTING", 13},
	{"STATE_CHARGING_POWER_ON", 14},
	{"STATE_CHARGING_POWER_ON_SIMPLIFIED", 15},
	{"STATE_CHARGING_WAIT_FOR_EV_RECONNECT", 16},
	{"STATE_CHARGING_TERMINATING", 17},
	{"STATE_CHARGING_WAKEUP", 18},
	{"STATE_WAIT_FOR_DISCONNECT", 19},
	{"STATE_WAIT_FOR_RELEASE_AUTHORISATION", 20},
	{"STATE_CHARGING_RECOVER_FROM_OUTAGE", 21},
	{"STATE_ERROR", 22},
	{"STATE_ERROR_MESSAGE", 23},
	{"STATE_ERROR_MESSAGE_CABLE_NOT_SUPPORTED", 24},
	{"STATE_ERROR_ILLEGAL_MODE_3", 25},
	{"STATE_ERROR_TOO_MANY_RESTARTS", 26},
	{"STATE_ERROR_CHARGING", 27},
	{"STATE_ERROR_CHARGING_OVERCURRENT", 28},
	{"STATE_ERROR_CHARGING_HF_CONTACTOR_SWITCHING", 29},
	{"STATE_ERROR_S2_NOT_OPENED", 30},
	{"STATE_ERROR_PROTECTIVE_EARTH", 31},
	{"STATE_ERROR_RELAYS", 32},
	{"STATE_ERROR_LOW_SUPPLY_VOLTAGE", 33},
	{"STATE_ERROR_INTERNAL_VOLTAGE", 34},
	{"STATE_ERROR_POWERMETER", 35},
	{"STATE_ERROR_TEMPERATURE", 36},
	{"STATE_SUSPENDED", 37},
	{"STATE_INOPERATIVE", 38},
	{"STATE_RESERVED", 39},
	{"STATE_ERROR_CHARGING_RCD_SIGNALED", 40},
	{"STATE_CHARGING_POWER_OFF_VENTILATING", 41},
	{"STATE_CHARGING_POWER_OFF_SUSPENDED", 42},
	{"STATE_CHARGING_POWER_OFF_PHASE_CHANGE", 43},
	{"STATE_WAIT_FOR_START_METERVALUE", 44},
	{"STATE_WAIT_FOR_STOP_METERVALUE", 45},
	{"STATE_ERROR_SOCKET_MOTOR", 46},
	{"STATE_CABLE_CONNECTED_TYPE_E", 47},
	{"STATE_CABLE_CONNECTED_TIMEOUT_TYPE_E", 48},
	{"STATE_CHARGING_TYPE_E", 49},
	{"STATE_WAIT_FOR_DISCONNECT_TYPE_E", 50},
	{"STATE_CHARGING_SUSPENDED_TYPE_E", 51},
	{"STATE_CHARGING_LOW_MAXCURRENT_TYPE_E", 52},
	{"STATE_INVALID_CARD", 53},
	{"STATE_EV_CONNECTED_UNAUTHORIZED", 54},
	{"STATE_WAIT_FOR_DISCONNECT_PP", 55},
}

// UserInterfaceError ports EUserInterfaceError (ACENetwork/ICUNetwork/EUserInterfaceError.cs).
// String returns the C# member name (or the decimal value when undefined, like Enum.ToString).
type UserInterfaceError int

const (
	UIErrorNone                  UserInterfaceError = 0   // UI_ERROR_NONE
	UIErrorGeneric               UserInterfaceError = 1   // UI_ERROR_GENERIC
	UIErrorChargingRcd           UserInterfaceError = 101 // UI_ERROR_CHARGING_RCD
	UIErrorRelays                UserInterfaceError = 102 // UI_ERROR_RELAYS
	UIErrorInternalVoltage       UserInterfaceError = 104 // UI_ERROR_INTERNAL_VOLTAGE
	UIErrorPowermeter            UserInterfaceError = 105 // UI_ERROR_POWERMETER
	UIErrorRcd                   UserInterfaceError = 106 // UI_ERROR_RCD
	UIErrorSocketMotorStartupOld UserInterfaceError = 107 // UI_ERROR_SOCKET_MOTOR_STARTUP_OLD
	UIErrorMissingpcid           UserInterfaceError = 108 // UI_ERROR_MISSINGPCID
	UIErrorNfcreader             UserInterfaceError = 109 // UI_ERROR_NFCREADER
	UIErrorProtectiveEarth       UserInterfaceError = 201 // UI_ERROR_PROTECTIVE_EARTH
	UIErrorLowSupplyVoltage      UserInterfaceError = 202 // UI_ERROR_LOW_SUPPLY_VOLTAGE
	UIErrorInoperative           UserInterfaceError = 206 // UI_ERROR_INOPERATIVE
	UIErrorHighSupplyVoltage     UserInterfaceError = 208 // UI_ERROR_HIGH_SUPPLY_VOLTAGE
	UIErrorP1pport               UserInterfaceError = 209 // UI_ERROR_P1PPORT
	UIErrorModbustcpip           UserInterfaceError = 210 // UI_ERROR_MODBUSTCPIP
	UIErrorSocketMotorStartup    UserInterfaceError = 211 // UI_ERROR_SOCKET_MOTOR_STARTUP
	UIErrorMissingphase          UserInterfaceError = 212 // UI_ERROR_MISSINGPHASE
	UIErrorTicport               UserInterfaceError = 213 // UI_ERROR_TICPORT
	UIErrorCharging              UserInterfaceError = 301 // UI_ERROR_CHARGING
	UIErrorChargingOvercurrent   UserInterfaceError = 302 // UI_ERROR_CHARGING_OVERCURRENT
	UIErrorChargingHfSwitching   UserInterfaceError = 303 // UI_ERROR_CHARGING_HF_SWITCHING
	UIErrorCableConnectedTimeout UserInterfaceError = 304 // UI_ERROR_CABLE_CONNECTED_TIMEOUT
	UIErrorTemperatureHigh       UserInterfaceError = 401 // UI_ERROR_TEMPERATURE_HIGH
	UIErrorTemperatureLow        UserInterfaceError = 402 // UI_ERROR_TEMPERATURE_LOW
	UIErrorMessage               UserInterfaceError = 403 // UI_ERROR_MESSAGE
	UIErrorSocketMotor           UserInterfaceError = 404 // UI_ERROR_SOCKET_MOTOR
	UIErrorIllegalMode3Pp        UserInterfaceError = 405 // UI_ERROR_ILLEGAL_MODE_3_PP
	UIErrorIllegalMode3Cp        UserInterfaceError = 406 // UI_ERROR_ILLEGAL_MODE_3_CP
	UIErrorTilt                  UserInterfaceError = 407 // UI_ERROR_TILT
)

// userInterfaceErrorNames lists every EUserInterfaceError member in declaration order.
var userInterfaceErrorNames = []struct {
	Name  string
	Value UserInterfaceError
}{
	{"UI_ERROR_NONE", 0},
	{"UI_ERROR_GENERIC", 1},
	{"UI_ERROR_CHARGING_RCD", 101},
	{"UI_ERROR_RELAYS", 102},
	{"UI_ERROR_INTERNAL_VOLTAGE", 104},
	{"UI_ERROR_POWERMETER", 105},
	{"UI_ERROR_RCD", 106},
	{"UI_ERROR_SOCKET_MOTOR_STARTUP_OLD", 107},
	{"UI_ERROR_MISSINGPCID", 108},
	{"UI_ERROR_NFCREADER", 109},
	{"UI_ERROR_PROTECTIVE_EARTH", 201},
	{"UI_ERROR_LOW_SUPPLY_VOLTAGE", 202},
	{"UI_ERROR_INOPERATIVE", 206},
	{"UI_ERROR_HIGH_SUPPLY_VOLTAGE", 208},
	{"UI_ERROR_P1PPORT", 209},
	{"UI_ERROR_MODBUSTCPIP", 210},
	{"UI_ERROR_SOCKET_MOTOR_STARTUP", 211},
	{"UI_ERROR_MISSINGPHASE", 212},
	{"UI_ERROR_TICPORT", 213},
	{"UI_ERROR_CHARGING", 301},
	{"UI_ERROR_CHARGING_OVERCURRENT", 302},
	{"UI_ERROR_CHARGING_HF_SWITCHING", 303},
	{"UI_ERROR_CABLE_CONNECTED_TIMEOUT", 304},
	{"UI_ERROR_TEMPERATURE_HIGH", 401},
	{"UI_ERROR_TEMPERATURE_LOW", 402},
	{"UI_ERROR_MESSAGE", 403},
	{"UI_ERROR_SOCKET_MOTOR", 404},
	{"UI_ERROR_ILLEGAL_MODE_3_PP", 405},
	{"UI_ERROR_ILLEGAL_MODE_3_CP", 406},
	{"UI_ERROR_TILT", 407},
}

// UserInterfaceState ports EUserInterfaceStates (ACENetwork/ICUNetwork/EUserInterfaceStates.cs).
// String returns the C# member name (or the decimal value when undefined, like Enum.ToString).
type UserInterfaceState int

const (
	UIStateUnknown               UserInterfaceState = 0  // UI_STATE_UNKNOWN
	UIStateBooting               UserInterfaceState = 1  // UI_STATE_BOOTING
	UIStateAvailable             UserInterfaceState = 2  // UI_STATE_AVAILABLE
	UIStateCableConnected        UserInterfaceState = 3  // UI_STATE_CABLE_CONNECTED
	UIStateEvConnected           UserInterfaceState = 4  // UI_STATE_EV_CONNECTED
	UIStateCableAuthorised       UserInterfaceState = 5  // UI_STATE_CABLE_AUTHORISED
	UIStateAuthorised            UserInterfaceState = 6  // UI_STATE_AUTHORISED
	UIStateCommunicating         UserInterfaceState = 7  // UI_STATE_COMMUNICATING
	UIStatePowerOffLowMaxCurrent UserInterfaceState = 8  // UI_STATE_POWER_OFF_LOW_MAX_CURRENT
	UIStatePowerOffSuspended     UserInterfaceState = 9  // UI_STATE_POWER_OFF_SUSPENDED
	UIStateCharging              UserInterfaceState = 10 // UI_STATE_CHARGING
	UIStateChargingFullLocked    UserInterfaceState = 11 // UI_STATE_CHARGING_FULL_LOCKED
	UIStateChargingFullUnlocked  UserInterfaceState = 12 // UI_STATE_CHARGING_FULL_UNLOCKED
	UIStateWaitForEvReconnect    UserInterfaceState = 13 // UI_STATE_WAIT_FOR_EV_RECONNECT
	UIStateTransactionInfo       UserInterfaceState = 14 // UI_STATE_TRANSACTION_INFO
	UIStateCardRejected          UserInterfaceState = 15 // UI_STATE_CARD_REJECTED
	UIStateError                 UserInterfaceState = 16 // UI_STATE_ERROR
	UIStatePleasewait            UserInterfaceState = 17 // UI_STATE_PLEASEWAIT
	UIStateReserved              UserInterfaceState = 18 // UI_STATE_RESERVED
	UIStateQrcode                UserInterfaceState = 19 // UI_STATE_QRCODE
	UIStateWarning               UserInterfaceState = 20 // UI_STATE_WARNING
	UIStateWaitForRelease        UserInterfaceState = 21 // UI_STATE_WAIT_FOR_RELEASE
	UIStatePleaseRemoveCable     UserInterfaceState = 22 // UI_STATE_PLEASE_REMOVE_CABLE
	UIStatePleasewaitEvComm      UserInterfaceState = 23 // UI_STATE_PLEASEWAIT_EV_COMM
	UIStateWaitForReleasePc      UserInterfaceState = 24 // UI_STATE_WAIT_FOR_RELEASE_PC
	UIStateInvalidCard           UserInterfaceState = 25 // UI_STATE_INVALID_CARD
)

// userInterfaceStateNames lists every EUserInterfaceStates member in declaration order.
var userInterfaceStateNames = []struct {
	Name  string
	Value UserInterfaceState
}{
	{"UI_STATE_UNKNOWN", 0},
	{"UI_STATE_BOOTING", 1},
	{"UI_STATE_AVAILABLE", 2},
	{"UI_STATE_CABLE_CONNECTED", 3},
	{"UI_STATE_EV_CONNECTED", 4},
	{"UI_STATE_CABLE_AUTHORISED", 5},
	{"UI_STATE_AUTHORISED", 6},
	{"UI_STATE_COMMUNICATING", 7},
	{"UI_STATE_POWER_OFF_LOW_MAX_CURRENT", 8},
	{"UI_STATE_POWER_OFF_SUSPENDED", 9},
	{"UI_STATE_CHARGING", 10},
	{"UI_STATE_CHARGING_FULL_LOCKED", 11},
	{"UI_STATE_CHARGING_FULL_UNLOCKED", 12},
	{"UI_STATE_WAIT_FOR_EV_RECONNECT", 13},
	{"UI_STATE_TRANSACTION_INFO", 14},
	{"UI_STATE_CARD_REJECTED", 15},
	{"UI_STATE_ERROR", 16},
	{"UI_STATE_PLEASEWAIT", 17},
	{"UI_STATE_RESERVED", 18},
	{"UI_STATE_QRCODE", 19},
	{"UI_STATE_WARNING", 20},
	{"UI_STATE_WAIT_FOR_RELEASE", 21},
	{"UI_STATE_PLEASE_REMOVE_CABLE", 22},
	{"UI_STATE_PLEASEWAIT_EV_COMM", 23},
	{"UI_STATE_WAIT_FOR_RELEASE_PC", 24},
	{"UI_STATE_INVALID_CARD", 25},
}

// FirmwareUpdateStatus ports EFirmwareUpdateStatus (ACENetwork/ICUNetwork/EFirmwareUpdateStatus.cs).
// String returns the C# member name (or the decimal value when undefined, like Enum.ToString).
type FirmwareUpdateStatus int

const (
	FwNoActiveUpdate      FirmwareUpdateStatus = 0  // NO_ACTIVE_UPDATE
	FwErasingBuffer       FirmwareUpdateStatus = 1  // ERASING_BUFFER
	FwBufferErased        FirmwareUpdateStatus = 2  // BUFFER_ERASED
	FwReadyForDownload    FirmwareUpdateStatus = 3  // READY_FOR_DOWNLOAD
	FwDownloadingFirmware FirmwareUpdateStatus = 4  // DOWNLOADING_FIRMWARE
	FwDownloadDone        FirmwareUpdateStatus = 5  // DOWNLOAD_DONE
	FwDownloadChecked     FirmwareUpdateStatus = 6  // DOWNLOAD_CHECKED
	FwReadyForUpdate      FirmwareUpdateStatus = 7  // READY_FOR_UPDATE
	FwUpdateInProgress    FirmwareUpdateStatus = 8  // UPDATE_IN_PROGRESS
	FwUpdateReadyToRoll   FirmwareUpdateStatus = 9  // UPDATE_READY_TO_ROLL
	FwUpdateDone          FirmwareUpdateStatus = 10 // UPDATE_DONE
	FwCrcCalculating      FirmwareUpdateStatus = 11 // CRC_CALCULATING
	FwCrcCalculated       FirmwareUpdateStatus = 12 // CRC_CALCULATED
	FwErrorDuringDownload FirmwareUpdateStatus = -1 // ERROR_DURING_DOWNLOAD
	FwErrorDuringUpdate   FirmwareUpdateStatus = -2 // ERROR_DURING_UPDATE
	FwExecutingRollback   FirmwareUpdateStatus = -3 // EXECUTING_ROLLBACK
	FwRolledBack          FirmwareUpdateStatus = -4 // ROLLED_BACK
)

// firmwareUpdateStatusNames lists every EFirmwareUpdateStatus member in declaration order.
var firmwareUpdateStatusNames = []struct {
	Name  string
	Value FirmwareUpdateStatus
}{
	{"NO_ACTIVE_UPDATE", 0},
	{"ERASING_BUFFER", 1},
	{"BUFFER_ERASED", 2},
	{"READY_FOR_DOWNLOAD", 3},
	{"DOWNLOADING_FIRMWARE", 4},
	{"DOWNLOAD_DONE", 5},
	{"DOWNLOAD_CHECKED", 6},
	{"READY_FOR_UPDATE", 7},
	{"UPDATE_IN_PROGRESS", 8},
	{"UPDATE_READY_TO_ROLL", 9},
	{"UPDATE_DONE", 10},
	{"CRC_CALCULATING", 11},
	{"CRC_CALCULATED", 12},
	{"ERROR_DURING_DOWNLOAD", -1},
	{"ERROR_DURING_UPDATE", -2},
	{"EXECUTING_ROLLBACK", -3},
	{"ROLLED_BACK", -4},
}
