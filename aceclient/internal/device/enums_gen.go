// Code generated from the decompiled C# enum definitions; DO NOT EDIT by hand
// without re-checking the sources listed on each type.

package device

// MainState ports the enum EMainStates (ACENetwork/ICUNetwork/EMainStates.cs).
// String returns the C# member name (Enum.ToString).
type MainState int

// MainState values, 1:1 with EMainStates.
const (
	MainStateIllegal MainState = -1 // STATE_ILLEGAL
	MainStateUnknown MainState = 0 // STATE_UNKNOWN
	MainStateBooting MainState = 1 // STATE_BOOTING
	MainStateAvailable MainState = 2 // STATE_AVAILABLE
	MainStateCableConnected MainState = 3 // STATE_CABLE_CONNECTED
	MainStateCableConnectedTimeout MainState = 4 // STATE_CABLE_CONNECTED_TIMEOUT
	MainStateEVConnected MainState = 5 // STATE_EV_CONNECTED
	MainStateButtonActivated MainState = 6 // STATE_BUTTON_ACTIVATED
	MainStateNFCAvailable MainState = 7 // STATE_NFC_AVAILABLE
	MainStateNFCAuthorised MainState = 8 // STATE_NFC_AUTHORISED
	MainStateWaitForEvconnect MainState = 9 // STATE_WAIT_FOR_EVCONNECT
	MainStateChargingTestRelays MainState = 10 // STATE_CHARGING_TEST_RELAYS
	MainStateChargingPowerOff MainState = 11 // STATE_CHARGING_POWER_OFF
	MainStateChargingPowerOffLowMaxcurrent MainState = 12 // STATE_CHARGING_POWER_OFF_LOW_MAXCURRENT
	MainStateChargingPowerStarting MainState = 13 // STATE_CHARGING_POWER_STARTING
	MainStateChargingPowerOn MainState = 14 // STATE_CHARGING_POWER_ON
	MainStateChargingPowerOnSimplified MainState = 15 // STATE_CHARGING_POWER_ON_SIMPLIFIED
	MainStateChargingWaitForEVReconnect MainState = 16 // STATE_CHARGING_WAIT_FOR_EV_RECONNECT
	MainStateChargingTerminating MainState = 17 // STATE_CHARGING_TERMINATING
	MainStateChargingWakeup MainState = 18 // STATE_CHARGING_WAKEUP
	MainStateWaitForDisconnect MainState = 19 // STATE_WAIT_FOR_DISCONNECT
	MainStateWaitForReleaseAuthorisation MainState = 20 // STATE_WAIT_FOR_RELEASE_AUTHORISATION
	MainStateChargingRecoverFromOutage MainState = 21 // STATE_CHARGING_RECOVER_FROM_OUTAGE
	MainStateError MainState = 22 // STATE_ERROR
	MainStateErrorMessage MainState = 23 // STATE_ERROR_MESSAGE
	MainStateErrorMessageCableNotSupported MainState = 24 // STATE_ERROR_MESSAGE_CABLE_NOT_SUPPORTED
	MainStateErrorIllegalMode3 MainState = 25 // STATE_ERROR_ILLEGAL_MODE_3
	MainStateErrorTooManyRestarts MainState = 26 // STATE_ERROR_TOO_MANY_RESTARTS
	MainStateErrorCharging MainState = 27 // STATE_ERROR_CHARGING
	MainStateErrorChargingOvercurrent MainState = 28 // STATE_ERROR_CHARGING_OVERCURRENT
	MainStateErrorChargingHFContactorSwitching MainState = 29 // STATE_ERROR_CHARGING_HF_CONTACTOR_SWITCHING
	MainStateErrorS2NotOpened MainState = 30 // STATE_ERROR_S2_NOT_OPENED
	MainStateErrorProtectiveEarth MainState = 31 // STATE_ERROR_PROTECTIVE_EARTH
	MainStateErrorRelays MainState = 32 // STATE_ERROR_RELAYS
	MainStateErrorLowSupplyVoltage MainState = 33 // STATE_ERROR_LOW_SUPPLY_VOLTAGE
	MainStateErrorInternalVoltage MainState = 34 // STATE_ERROR_INTERNAL_VOLTAGE
	MainStateErrorPowermeter MainState = 35 // STATE_ERROR_POWERMETER
	MainStateErrorTemperature MainState = 36 // STATE_ERROR_TEMPERATURE
	MainStateSuspended MainState = 37 // STATE_SUSPENDED
	MainStateInoperative MainState = 38 // STATE_INOPERATIVE
	MainStateReserved MainState = 39 // STATE_RESERVED
	MainStateErrorChargingRCDSignaled MainState = 40 // STATE_ERROR_CHARGING_RCD_SIGNALED
	MainStateChargingPowerOffVentilating MainState = 41 // STATE_CHARGING_POWER_OFF_VENTILATING
	MainStateChargingPowerOffSuspended MainState = 42 // STATE_CHARGING_POWER_OFF_SUSPENDED
	MainStateChargingPowerOffPhaseChange MainState = 43 // STATE_CHARGING_POWER_OFF_PHASE_CHANGE
	MainStateWaitForStartMetervalue MainState = 44 // STATE_WAIT_FOR_START_METERVALUE
	MainStateWaitForStopMetervalue MainState = 45 // STATE_WAIT_FOR_STOP_METERVALUE
	MainStateErrorSocketMotor MainState = 46 // STATE_ERROR_SOCKET_MOTOR
	MainStateCableConnectedTypeE MainState = 47 // STATE_CABLE_CONNECTED_TYPE_E
	MainStateCableConnectedTimeoutTypeE MainState = 48 // STATE_CABLE_CONNECTED_TIMEOUT_TYPE_E
	MainStateChargingTypeE MainState = 49 // STATE_CHARGING_TYPE_E
	MainStateWaitForDisconnectTypeE MainState = 50 // STATE_WAIT_FOR_DISCONNECT_TYPE_E
	MainStateChargingSuspendedTypeE MainState = 51 // STATE_CHARGING_SUSPENDED_TYPE_E
	MainStateChargingLowMaxcurrentTypeE MainState = 52 // STATE_CHARGING_LOW_MAXCURRENT_TYPE_E
	MainStateInvalidCard MainState = 53 // STATE_INVALID_CARD
	MainStateEVConnectedUnauthorized MainState = 54 // STATE_EV_CONNECTED_UNAUTHORIZED
	MainStateWaitForDisconnectPP MainState = 55 // STATE_WAIT_FOR_DISCONNECT_PP
)

var mainStateEnum = newEnumInfo(false, []enumMember{
	{"STATE_ILLEGAL", -1, ""},
	{"STATE_UNKNOWN", 0, ""},
	{"STATE_BOOTING", 1, ""},
	{"STATE_AVAILABLE", 2, ""},
	{"STATE_CABLE_CONNECTED", 3, ""},
	{"STATE_CABLE_CONNECTED_TIMEOUT", 4, ""},
	{"STATE_EV_CONNECTED", 5, ""},
	{"STATE_BUTTON_ACTIVATED", 6, ""},
	{"STATE_NFC_AVAILABLE", 7, ""},
	{"STATE_NFC_AUTHORISED", 8, ""},
	{"STATE_WAIT_FOR_EVCONNECT", 9, ""},
	{"STATE_CHARGING_TEST_RELAYS", 10, ""},
	{"STATE_CHARGING_POWER_OFF", 11, ""},
	{"STATE_CHARGING_POWER_OFF_LOW_MAXCURRENT", 12, ""},
	{"STATE_CHARGING_POWER_STARTING", 13, ""},
	{"STATE_CHARGING_POWER_ON", 14, ""},
	{"STATE_CHARGING_POWER_ON_SIMPLIFIED", 15, ""},
	{"STATE_CHARGING_WAIT_FOR_EV_RECONNECT", 16, ""},
	{"STATE_CHARGING_TERMINATING", 17, ""},
	{"STATE_CHARGING_WAKEUP", 18, ""},
	{"STATE_WAIT_FOR_DISCONNECT", 19, ""},
	{"STATE_WAIT_FOR_RELEASE_AUTHORISATION", 20, ""},
	{"STATE_CHARGING_RECOVER_FROM_OUTAGE", 21, ""},
	{"STATE_ERROR", 22, ""},
	{"STATE_ERROR_MESSAGE", 23, ""},
	{"STATE_ERROR_MESSAGE_CABLE_NOT_SUPPORTED", 24, ""},
	{"STATE_ERROR_ILLEGAL_MODE_3", 25, ""},
	{"STATE_ERROR_TOO_MANY_RESTARTS", 26, ""},
	{"STATE_ERROR_CHARGING", 27, ""},
	{"STATE_ERROR_CHARGING_OVERCURRENT", 28, ""},
	{"STATE_ERROR_CHARGING_HF_CONTACTOR_SWITCHING", 29, ""},
	{"STATE_ERROR_S2_NOT_OPENED", 30, ""},
	{"STATE_ERROR_PROTECTIVE_EARTH", 31, ""},
	{"STATE_ERROR_RELAYS", 32, ""},
	{"STATE_ERROR_LOW_SUPPLY_VOLTAGE", 33, ""},
	{"STATE_ERROR_INTERNAL_VOLTAGE", 34, ""},
	{"STATE_ERROR_POWERMETER", 35, ""},
	{"STATE_ERROR_TEMPERATURE", 36, ""},
	{"STATE_SUSPENDED", 37, ""},
	{"STATE_INOPERATIVE", 38, ""},
	{"STATE_RESERVED", 39, ""},
	{"STATE_ERROR_CHARGING_RCD_SIGNALED", 40, ""},
	{"STATE_CHARGING_POWER_OFF_VENTILATING", 41, ""},
	{"STATE_CHARGING_POWER_OFF_SUSPENDED", 42, ""},
	{"STATE_CHARGING_POWER_OFF_PHASE_CHANGE", 43, ""},
	{"STATE_WAIT_FOR_START_METERVALUE", 44, ""},
	{"STATE_WAIT_FOR_STOP_METERVALUE", 45, ""},
	{"STATE_ERROR_SOCKET_MOTOR", 46, ""},
	{"STATE_CABLE_CONNECTED_TYPE_E", 47, ""},
	{"STATE_CABLE_CONNECTED_TIMEOUT_TYPE_E", 48, ""},
	{"STATE_CHARGING_TYPE_E", 49, ""},
	{"STATE_WAIT_FOR_DISCONNECT_TYPE_E", 50, ""},
	{"STATE_CHARGING_SUSPENDED_TYPE_E", 51, ""},
	{"STATE_CHARGING_LOW_MAXCURRENT_TYPE_E", 52, ""},
	{"STATE_INVALID_CARD", 53, ""},
	{"STATE_EV_CONNECTED_UNAUTHORIZED", 54, ""},
	{"STATE_WAIT_FOR_DISCONNECT_PP", 55, ""},
})

// String ports EMainStates.ToString() (.NET Framework Enum.ToString).
func (v MainState) String() string { return mainStateEnum.format(int64(v)) }

// LEDState ports the enum ELEDStates (ACENetwork/ICUNetwork/ELEDStates.cs).
// String returns the C# member name (Enum.ToString).
type LEDState int

// LEDState values, 1:1 with ELEDStates.
const (
	LEDUnknown LEDState = 0 // LED_UNKNOWN
	LEDOff LEDState = 1 // LED_OFF
	LEDBooting LEDState = 2 // LED_BOOTING
	LEDBootingCheckMains LEDState = 3 // LED_BOOTING_CHECK_MAINS
	LEDAvailable LEDState = 4 // LED_AVAILABLE
	LEDPrepAuthorizing LEDState = 5 // LED_PREP_AUTHORIZING
	LEDPrepAuthorized LEDState = 6 // LED_PREP_AUTHORIZED
	LEDPrepCableConnected LEDState = 7 // LED_PREP_CABLE_CONNECTED
	LEDPrepEVConnected LEDState = 8 // LED_PREP_EV_CONNECTED
	LEDChargingPreparing LEDState = 9 // LED_CHARGING_PREPARING
	LEDChargingWaitVehicle LEDState = 10 // LED_CHARGING_WAIT_VEHICLE
	LEDChargingActiveNormal LEDState = 11 // LED_CHARGING_ACTIVE_NORMAL
	LEDChargingActiveSimplified LEDState = 12 // LED_CHARGING_ACTIVE_SIMPLIFIED
	LEDChargingSuspendedOvercurrent LEDState = 13 // LED_CHARGING_SUSPENDED_OVERCURRENT
	LEDChargingSuspendedHFSwitching LEDState = 14 // LED_CHARGING_SUSPENDED_HF_SWITCHING
	LEDChargingSuspendedEVDisconnected LEDState = 15 // LED_CHARGING_SUSPENDED_EV_DISCONNECTED
	LEDFinishWaitVehicle LEDState = 16 // LED_FINISH_WAIT_VEHICLE
	LEDFinishWaitForDisconnect LEDState = 17 // LED_FINISH_WAIT_FOR_DISCONNECT
	LEDErrorProtectiveEarth LEDState = 18 // LED_ERROR_PROTECTIVE_EARTH
	LEDErrorPowerlineFault LEDState = 19 // LED_ERROR_POWERLINE_FAULT
	LEDErrorContactorFault LEDState = 20 // LED_ERROR_CONTACTOR_FAULT
	LEDErrorCharging LEDState = 21 // LED_ERROR_CHARGING
	LEDErrorPowerfailure LEDState = 22 // LED_ERROR_POWERFAILURE
	LEDErrorTemperature LEDState = 23 // LED_ERROR_TEMPERATURE
	LEDErrorIllegalCPValue LEDState = 24 // LED_ERROR_ILLEGAL_CP_VALUE
	LEDErrorIllegalPPValue LEDState = 25 // LED_ERROR_ILLEGAL_PP_VALUE
	LEDError LEDState = 26 // LED_ERROR
	LEDErrorTooManyRestarts LEDState = 27 // LED_ERROR_TOO_MANY_RESTARTS
	LEDErrormessage LEDState = 28 // LED_ERRORMESSAGE
	LEDErrormessageNotAuthorized LEDState = 29 // LED_ERRORMESSAGE_NOT_AUTHORIZED
	LEDErrormessageCableNotSupported LEDState = 30 // LED_ERRORMESSAGE_CABLE_NOT_SUPPORTED
	LEDErrormessageS2NotOpened LEDState = 31 // LED_ERRORMESSAGE_S2_NOT_OPENED
	LEDErrormessageTimeout LEDState = 32 // LED_ERRORMESSAGE_TIMEOUT
	LEDReserved LEDState = 33 // LED_RESERVED
	LEDInoperative LEDState = 34 // LED_INOPERATIVE
	LEDLoadbalancingLimited LEDState = 35 // LED_LOADBALANCING_LIMITED
	LEDLoadbalancingForcedOff LEDState = 36 // LED_LOADBALANCING_FORCED_OFF
	LEDTagMode LEDState = 37 // LED_TAG_MODE
	LEDTagModeAdded LEDState = 38 // LED_TAG_MODE_ADDED
	LEDTagModeRemoved LEDState = 39 // LED_TAG_MODE_REMOVED
	LEDChargingNonCharging LEDState = 40 // LED_CHARGING_NON_CHARGING
)

var lEDStateEnum = newEnumInfo(false, []enumMember{
	{"LED_UNKNOWN", 0, ""},
	{"LED_OFF", 1, ""},
	{"LED_BOOTING", 2, ""},
	{"LED_BOOTING_CHECK_MAINS", 3, ""},
	{"LED_AVAILABLE", 4, ""},
	{"LED_PREP_AUTHORIZING", 5, ""},
	{"LED_PREP_AUTHORIZED", 6, ""},
	{"LED_PREP_CABLE_CONNECTED", 7, ""},
	{"LED_PREP_EV_CONNECTED", 8, ""},
	{"LED_CHARGING_PREPARING", 9, ""},
	{"LED_CHARGING_WAIT_VEHICLE", 10, ""},
	{"LED_CHARGING_ACTIVE_NORMAL", 11, ""},
	{"LED_CHARGING_ACTIVE_SIMPLIFIED", 12, ""},
	{"LED_CHARGING_SUSPENDED_OVERCURRENT", 13, ""},
	{"LED_CHARGING_SUSPENDED_HF_SWITCHING", 14, ""},
	{"LED_CHARGING_SUSPENDED_EV_DISCONNECTED", 15, ""},
	{"LED_FINISH_WAIT_VEHICLE", 16, ""},
	{"LED_FINISH_WAIT_FOR_DISCONNECT", 17, ""},
	{"LED_ERROR_PROTECTIVE_EARTH", 18, ""},
	{"LED_ERROR_POWERLINE_FAULT", 19, ""},
	{"LED_ERROR_CONTACTOR_FAULT", 20, ""},
	{"LED_ERROR_CHARGING", 21, ""},
	{"LED_ERROR_POWERFAILURE", 22, ""},
	{"LED_ERROR_TEMPERATURE", 23, ""},
	{"LED_ERROR_ILLEGAL_CP_VALUE", 24, ""},
	{"LED_ERROR_ILLEGAL_PP_VALUE", 25, ""},
	{"LED_ERROR", 26, ""},
	{"LED_ERROR_TOO_MANY_RESTARTS", 27, ""},
	{"LED_ERRORMESSAGE", 28, ""},
	{"LED_ERRORMESSAGE_NOT_AUTHORIZED", 29, ""},
	{"LED_ERRORMESSAGE_CABLE_NOT_SUPPORTED", 30, ""},
	{"LED_ERRORMESSAGE_S2_NOT_OPENED", 31, ""},
	{"LED_ERRORMESSAGE_TIMEOUT", 32, ""},
	{"LED_RESERVED", 33, ""},
	{"LED_INOPERATIVE", 34, ""},
	{"LED_LOADBALANCING_LIMITED", 35, ""},
	{"LED_LOADBALANCING_FORCED_OFF", 36, ""},
	{"LED_TAG_MODE", 37, ""},
	{"LED_TAG_MODE_ADDED", 38, ""},
	{"LED_TAG_MODE_REMOVED", 39, ""},
	{"LED_CHARGING_NON_CHARGING", 40, ""},
})

// String ports ELEDStates.ToString() (.NET Framework Enum.ToString).
func (v LEDState) String() string { return lEDStateEnum.format(int64(v)) }

// AHWPCCState ports the enum EAHWPCCStates (ACENetwork/ICUNetwork/EAHWPCCStates.cs).
// String returns the C# member name (Enum.ToString).
type AHWPCCState int

// AHWPCCState values, 1:1 with EAHWPCCStates.
const (
	AHWPCCUnknown AHWPCCState = 0 // unknown
	AHWPCCInit AHWPCCState = 1 // init
	AHWPCCIdle AHWPCCState = 2 // idle
	AHWPCCWaitForStartMV AHWPCCState = 3 // waitForStartMV
	AHWPCCPreparing AHWPCCState = 4 // preparing
	AHWPCCWaitForEVConnect AHWPCCState = 5 // waitForEVConnect
	AHWPCCChargingPowerOn AHWPCCState = 6 // chargingPowerOn
	AHWPCCChargingPowerOff AHWPCCState = 7 // chargingPowerOff
	AHWPCCChargingWaitForEVReconnect AHWPCCState = 8 // chargingWaitForEVReconnect
	AHWPCCSwitchPhases AHWPCCState = 9 // switchPhases
	AHWPCCWaitForStopMV AHWPCCState = 10 // waitForStopMV
	AHWPCCFinished AHWPCCState = 11 // finished
	AHWPCCLowSupplyVoltageError AHWPCCState = 12 // lowSupplyVoltageError
	AHWPCCRecoverFromOutage AHWPCCState = 13 // recoverFromOutage
	AHWPCCRelayError AHWPCCState = 14 // relayError
	AHWPCCChargingSuspendedEVSE AHWPCCState = 15 // chargingSuspendedEVSE
	AHWPCCTemperatureError AHWPCCState = 16 // temperatureError
	AHWPCCOvercurrentError AHWPCCState = 17 // overcurrentError
	AHWPCCSocketMotorError AHWPCCState = 18 // socketMotorError
	AHWPCCIllegalMode3Error AHWPCCState = 19 // illegalMode3Error
	AHWPCCEnergyMeterError AHWPCCState = 20 // energyMeterError
	AHWPCCPhaseError AHWPCCState = 21 // phaseError
	AHWPCCInternalRCDError AHWPCCState = 22 // internalRCDError
	AHWPCCHfSwitchingError AHWPCCState = 23 // hfSwitchingError
	AHWPCCStopRequested AHWPCCState = 24 // stopRequested
	AHWPCCStationError AHWPCCState = 25 // stationError
	AHWPCCChargeParameterDiscovery AHWPCCState = 26 // chargeParameterDiscovery
	AHWPCCCableCheck AHWPCCState = 27 // cableCheck
	AHWPCCPreCharge AHWPCCState = 28 // preCharge
	AHWPCCGenericError AHWPCCState = 29 // genericError
	AHWPCCTamperError AHWPCCState = 30 // tamperError
	AHWPCCComponentError AHWPCCState = 31 // componentError
	AHWPCCInternalVoltageError AHWPCCState = 32 // InternalVoltageError
	AHWPCCIllegalMode3PPError AHWPCCState = 33 // IllegalMode3PPError
	AHWPCCSocketMotorBootError AHWPCCState = 34 // SocketMotorBootError
	AHWPCCTemperatureLowError AHWPCCState = 35 // TemperatureLowError
	AHWPCCInternalRCDTrippedError AHWPCCState = 36 // InternalRCDTrippedError
	AHWPCCStateCount AHWPCCState = 37 // stateCount
)

var aHWPCCStateEnum = newEnumInfo(false, []enumMember{
	{"unknown", 0, ""},
	{"init", 1, ""},
	{"idle", 2, ""},
	{"waitForStartMV", 3, ""},
	{"preparing", 4, ""},
	{"waitForEVConnect", 5, ""},
	{"chargingPowerOn", 6, ""},
	{"chargingPowerOff", 7, ""},
	{"chargingWaitForEVReconnect", 8, ""},
	{"switchPhases", 9, ""},
	{"waitForStopMV", 10, ""},
	{"finished", 11, ""},
	{"lowSupplyVoltageError", 12, ""},
	{"recoverFromOutage", 13, ""},
	{"relayError", 14, ""},
	{"chargingSuspendedEVSE", 15, ""},
	{"temperatureError", 16, ""},
	{"overcurrentError", 17, ""},
	{"socketMotorError", 18, ""},
	{"illegalMode3Error", 19, ""},
	{"energyMeterError", 20, ""},
	{"phaseError", 21, ""},
	{"internalRCDError", 22, ""},
	{"hfSwitchingError", 23, ""},
	{"stopRequested", 24, ""},
	{"stationError", 25, ""},
	{"chargeParameterDiscovery", 26, ""},
	{"cableCheck", 27, ""},
	{"preCharge", 28, ""},
	{"genericError", 29, ""},
	{"tamperError", 30, ""},
	{"componentError", 31, ""},
	{"InternalVoltageError", 32, ""},
	{"IllegalMode3PPError", 33, ""},
	{"SocketMotorBootError", 34, ""},
	{"TemperatureLowError", 35, ""},
	{"InternalRCDTrippedError", 36, ""},
	{"stateCount", 37, ""},
})

// String ports EAHWPCCStates.ToString() (.NET Framework Enum.ToString).
func (v AHWPCCState) String() string { return aHWPCCStateEnum.format(int64(v)) }

// AHWPCSMMainState ports the enum EAHWPCSMMainStates (ACENetwork/ICUNetwork/EAHWPCSMMainStates.cs).
// String returns the C# member name (Enum.ToString).
type AHWPCSMMainState int

// AHWPCSMMainState values, 1:1 with EAHWPCSMMainStates.
const (
	AHWPCSMUnknown AHWPCSMMainState = 0 // Unknown
	AHWPCSMBooting AHWPCSMMainState = 15 // Booting
	AHWPCSMAvailable AHWPCSMMainState = 1 // Available
	AHWPCSMAuthorising AHWPCSMMainState = 2 // Authorising
	AHWPCSMAuthorised AHWPCSMMainState = 4 // Authorised
	AHWPCSMRejected AHWPCSMMainState = 8 // Rejected
	AHWPCSMCableConnected AHWPCSMMainState = 16 // CableConnected
	AHWPCSMCableConnectedAuthorising AHWPCSMMainState = 18 // CableConnectedAuthorising
	AHWPCSMCableConnectedAuthorised AHWPCSMMainState = 20 // CableConnectedAuthorised
	AHWPCSMCableConnectedRejected AHWPCSMMainState = 24 // CableConnectedRejected
	AHWPCSMEVConnected AHWPCSMMainState = 48 // EVConnected
	AHWPCSMEVConnectedAuthorising AHWPCSMMainState = 50 // EVConnectedAuthorising
	AHWPCSMEVConnectedAuthorised AHWPCSMMainState = 52 // EVConnectedAuthorised
	AHWPCSMEVConnectedRejected AHWPCSMMainState = 56 // EVConnectedRejected
	AHWPCSMCableLocked AHWPCSMMainState = 65 // CableLocked
	AHWPCSMChargingStarting AHWPCSMMainState = 66 // ChargingStarting
	AHWPCSMCharging AHWPCSMMainState = 67 // Charging
	AHWPCSMChargingFinishing AHWPCSMMainState = 68 // ChargingFinishing
	AHWPCSMChargingFinished AHWPCSMMainState = 69 // ChargingFinished
	AHWPCSMCableUnlock AHWPCSMMainState = 70 // CableUnlock
	AHWPCSMSuspendedEV AHWPCSMMainState = 71 // SuspendedEV
	AHWPCSMSuspendedEVSE AHWPCSMMainState = 72 // SuspendedEVSE
	AHWPCSMChargingEVFull AHWPCSMMainState = 73 // ChargingEVFull
	AHWPCSMChargeParameterDiscovery AHWPCSMMainState = 74 // ChargeParameterDiscovery
	AHWPCSMCableCheck AHWPCSMMainState = 75 // CableCheck
	AHWPCSMPreCharge AHWPCSMMainState = 76 // PreCharge
	AHWPCSMWaitForCableDisconnect AHWPCSMMainState = 79 // WaitForCableDisconnect
	AHWPCSMTimeoutWaitingForCable AHWPCSMMainState = 128 // TimeoutWaitingForCable
	AHWPCSMTimeoutWaitingForEVConnect AHWPCSMMainState = 129 // TimeoutWaitingForEVConnect
	AHWPCSMTimeoutWaitingForAuthorisation AHWPCSMMainState = 130 // TimeoutWaitingForAuthorisation
	AHWPCSMTimeoutWaitingForS2 AHWPCSMMainState = 131 // TimeoutWaitingForS2
	AHWPCSMTimeoutWaitingForCableRemoval AHWPCSMMainState = 132 // TimeoutWaitingForCableRemoval
	AHWPCSMOffline AHWPCSMMainState = 159 // Offline
	AHWPCSMInoperative AHWPCSMMainState = 160 // Inoperative
	AHWPCSMReserved AHWPCSMMainState = 161 // Reserved
	AHWPCSMTariffAndOrTimeChanged AHWPCSMMainState = 162 // TariffAndOrTimeChanged
	AHWPCSMErrorMask AHWPCSMMainState = 192 // ErrorMask
	AHWPCSMErrorRelay AHWPCSMMainState = 193 // ErrorRelay
	AHWPCSMErrorTemperatureHigh AHWPCSMMainState = 194 // ErrorTemperatureHigh
	AHWPCSMErrorOvercurrent AHWPCSMMainState = 195 // ErrorOvercurrent
	AHWPCSMErrorSocketMotor AHWPCSMMainState = 196 // ErrorSocketMotor
	AHWPCSMErrorIllegalMode3CP AHWPCSMMainState = 197 // ErrorIllegalMode3CP
	AHWPCSMErrorEnergyMeter AHWPCSMMainState = 198 // ErrorEnergyMeter
	AHWPCSMErrorPhase AHWPCSMMainState = 199 // ErrorPhase
	AHWPCSMErrorInternalRCDTripped AHWPCSMMainState = 200 // ErrorInternalRCDTripped
	AHWPCSMErrorHFSwitching AHWPCSMMainState = 201 // ErrorHFSwitching
	AHWPCSMErrorLowSupplyVoltage AHWPCSMMainState = 202 // ErrorLowSupplyVoltage
	AHWPCSMErrorExternalRCD AHWPCSMMainState = 203 // ErrorExternalRCD
	AHWPCSMErrorGeneric AHWPCSMMainState = 204 // ErrorGeneric
	AHWPCSMErrorTamper AHWPCSMMainState = 205 // ErrorTamper
	AHWPCSMErrorComponent AHWPCSMMainState = 206 // ErrorComponent
	AHWPCSMErrorInternalVoltage AHWPCSMMainState = 207 // ErrorInternalVoltage
	AHWPCSMErrorIllegalMode3PP AHWPCSMMainState = 208 // ErrorIllegalMode3PP
	AHWPCSMErrorRelayOrRCD AHWPCSMMainState = 209 // ErrorRelayOrRCD
	AHWPCSMErrorSocketMotorBoot AHWPCSMMainState = 210 // ErrorSocketMotorBoot
	AHWPCSMErrorTemperatureLow AHWPCSMMainState = 211 // ErrorTemperatureLow
	AHWPCSMErrorInternalRCDFailure AHWPCSMMainState = 212 // ErrorInternalRCDFailure
	AHWPCSMErrorSCBMask AHWPCSMMainState = 224 // ErrorSCBMask
	AHWPCSMErrorStationError AHWPCSMMainState = 225 // ErrorStationError
	AHWPCSMErrorMissingRFID AHWPCSMMainState = 226 // ErrorMissingRFID
	AHWPCSMErrorMissingPnCID AHWPCSMMainState = 227 // ErrorMissingPnCID
	AHWPCSMErrorMissingSignedMeter AHWPCSMMainState = 228 // ErrorMissingSignedMeter
	AHWPCSMErrorMissingTariffs AHWPCSMMainState = 229 // ErrorMissingTariffs
)

var aHWPCSMMainStateEnum = newEnumInfo(false, []enumMember{
	{"Unknown", 0, ""},
	{"Booting", 15, ""},
	{"Available", 1, ""},
	{"Authorising", 2, ""},
	{"Authorised", 4, ""},
	{"Rejected", 8, ""},
	{"CableConnected", 16, ""},
	{"CableConnectedAuthorising", 18, ""},
	{"CableConnectedAuthorised", 20, ""},
	{"CableConnectedRejected", 24, ""},
	{"EVConnected", 48, ""},
	{"EVConnectedAuthorising", 50, ""},
	{"EVConnectedAuthorised", 52, ""},
	{"EVConnectedRejected", 56, ""},
	{"CableLocked", 65, ""},
	{"ChargingStarting", 66, ""},
	{"Charging", 67, ""},
	{"ChargingFinishing", 68, ""},
	{"ChargingFinished", 69, ""},
	{"CableUnlock", 70, ""},
	{"SuspendedEV", 71, ""},
	{"SuspendedEVSE", 72, ""},
	{"ChargingEVFull", 73, ""},
	{"ChargeParameterDiscovery", 74, ""},
	{"CableCheck", 75, ""},
	{"PreCharge", 76, ""},
	{"WaitForCableDisconnect", 79, ""},
	{"TimeoutWaitingForCable", 128, ""},
	{"TimeoutWaitingForEVConnect", 129, ""},
	{"TimeoutWaitingForAuthorisation", 130, ""},
	{"TimeoutWaitingForS2", 131, ""},
	{"TimeoutWaitingForCableRemoval", 132, ""},
	{"Offline", 159, ""},
	{"Inoperative", 160, ""},
	{"Reserved", 161, ""},
	{"TariffAndOrTimeChanged", 162, ""},
	{"ErrorMask", 192, ""},
	{"ErrorRelay", 193, ""},
	{"ErrorTemperatureHigh", 194, ""},
	{"ErrorOvercurrent", 195, ""},
	{"ErrorSocketMotor", 196, ""},
	{"ErrorIllegalMode3CP", 197, ""},
	{"ErrorEnergyMeter", 198, ""},
	{"ErrorPhase", 199, ""},
	{"ErrorInternalRCDTripped", 200, ""},
	{"ErrorHFSwitching", 201, ""},
	{"ErrorLowSupplyVoltage", 202, ""},
	{"ErrorExternalRCD", 203, ""},
	{"ErrorGeneric", 204, ""},
	{"ErrorTamper", 205, ""},
	{"ErrorComponent", 206, ""},
	{"ErrorInternalVoltage", 207, ""},
	{"ErrorIllegalMode3PP", 208, ""},
	{"ErrorRelayOrRCD", 209, ""},
	{"ErrorSocketMotorBoot", 210, ""},
	{"ErrorTemperatureLow", 211, ""},
	{"ErrorInternalRCDFailure", 212, ""},
	{"ErrorSCBMask", 224, ""},
	{"ErrorStationError", 225, ""},
	{"ErrorMissingRFID", 226, ""},
	{"ErrorMissingPnCID", 227, ""},
	{"ErrorMissingSignedMeter", 228, ""},
	{"ErrorMissingTariffs", 229, ""},
})

// String ports EAHWPCSMMainStates.ToString() (.NET Framework Enum.ToString).
func (v AHWPCSMMainState) String() string { return aHWPCSMMainStateEnum.format(int64(v)) }

// UserInterfaceState ports the enum EUserInterfaceStates (ACENetwork/ICUNetwork/EUserInterfaceStates.cs).
// String returns the C# member name (Enum.ToString).
type UserInterfaceState int

// UserInterfaceState values, 1:1 with EUserInterfaceStates.
const (
	UIStateUnknown UserInterfaceState = 0 // UI_STATE_UNKNOWN
	UIStateBooting UserInterfaceState = 1 // UI_STATE_BOOTING
	UIStateAvailable UserInterfaceState = 2 // UI_STATE_AVAILABLE
	UIStateCableConnected UserInterfaceState = 3 // UI_STATE_CABLE_CONNECTED
	UIStateEVConnected UserInterfaceState = 4 // UI_STATE_EV_CONNECTED
	UIStateCableAuthorised UserInterfaceState = 5 // UI_STATE_CABLE_AUTHORISED
	UIStateAuthorised UserInterfaceState = 6 // UI_STATE_AUTHORISED
	UIStateCommunicating UserInterfaceState = 7 // UI_STATE_COMMUNICATING
	UIStatePowerOffLowMaxCurrent UserInterfaceState = 8 // UI_STATE_POWER_OFF_LOW_MAX_CURRENT
	UIStatePowerOffSuspended UserInterfaceState = 9 // UI_STATE_POWER_OFF_SUSPENDED
	UIStateCharging UserInterfaceState = 10 // UI_STATE_CHARGING
	UIStateChargingFullLocked UserInterfaceState = 11 // UI_STATE_CHARGING_FULL_LOCKED
	UIStateChargingFullUnlocked UserInterfaceState = 12 // UI_STATE_CHARGING_FULL_UNLOCKED
	UIStateWaitForEVReconnect UserInterfaceState = 13 // UI_STATE_WAIT_FOR_EV_RECONNECT
	UIStateTransactionInfo UserInterfaceState = 14 // UI_STATE_TRANSACTION_INFO
	UIStateCardRejected UserInterfaceState = 15 // UI_STATE_CARD_REJECTED
	UIStateError UserInterfaceState = 16 // UI_STATE_ERROR
	UIStatePleasewait UserInterfaceState = 17 // UI_STATE_PLEASEWAIT
	UIStateReserved UserInterfaceState = 18 // UI_STATE_RESERVED
	UIStateQrcode UserInterfaceState = 19 // UI_STATE_QRCODE
	UIStateWarning UserInterfaceState = 20 // UI_STATE_WARNING
	UIStateWaitForRelease UserInterfaceState = 21 // UI_STATE_WAIT_FOR_RELEASE
	UIStatePleaseRemoveCable UserInterfaceState = 22 // UI_STATE_PLEASE_REMOVE_CABLE
	UIStatePleasewaitEVComm UserInterfaceState = 23 // UI_STATE_PLEASEWAIT_EV_COMM
	UIStateWaitForReleasePC UserInterfaceState = 24 // UI_STATE_WAIT_FOR_RELEASE_PC
	UIStateInvalidCard UserInterfaceState = 25 // UI_STATE_INVALID_CARD
)

var userInterfaceStateEnum = newEnumInfo(false, []enumMember{
	{"UI_STATE_UNKNOWN", 0, ""},
	{"UI_STATE_BOOTING", 1, ""},
	{"UI_STATE_AVAILABLE", 2, ""},
	{"UI_STATE_CABLE_CONNECTED", 3, ""},
	{"UI_STATE_EV_CONNECTED", 4, ""},
	{"UI_STATE_CABLE_AUTHORISED", 5, ""},
	{"UI_STATE_AUTHORISED", 6, ""},
	{"UI_STATE_COMMUNICATING", 7, ""},
	{"UI_STATE_POWER_OFF_LOW_MAX_CURRENT", 8, ""},
	{"UI_STATE_POWER_OFF_SUSPENDED", 9, ""},
	{"UI_STATE_CHARGING", 10, ""},
	{"UI_STATE_CHARGING_FULL_LOCKED", 11, ""},
	{"UI_STATE_CHARGING_FULL_UNLOCKED", 12, ""},
	{"UI_STATE_WAIT_FOR_EV_RECONNECT", 13, ""},
	{"UI_STATE_TRANSACTION_INFO", 14, ""},
	{"UI_STATE_CARD_REJECTED", 15, ""},
	{"UI_STATE_ERROR", 16, ""},
	{"UI_STATE_PLEASEWAIT", 17, ""},
	{"UI_STATE_RESERVED", 18, ""},
	{"UI_STATE_QRCODE", 19, ""},
	{"UI_STATE_WARNING", 20, ""},
	{"UI_STATE_WAIT_FOR_RELEASE", 21, ""},
	{"UI_STATE_PLEASE_REMOVE_CABLE", 22, ""},
	{"UI_STATE_PLEASEWAIT_EV_COMM", 23, ""},
	{"UI_STATE_WAIT_FOR_RELEASE_PC", 24, ""},
	{"UI_STATE_INVALID_CARD", 25, ""},
})

// String ports EUserInterfaceStates.ToString() (.NET Framework Enum.ToString).
func (v UserInterfaceState) String() string { return userInterfaceStateEnum.format(int64(v)) }

// UserInterfaceError ports the enum EUserInterfaceError (ACENetwork/ICUNetwork/EUserInterfaceError.cs).
// String returns the C# member name (Enum.ToString).
type UserInterfaceError int

// UserInterfaceError values, 1:1 with EUserInterfaceError.
const (
	UIErrorNone UserInterfaceError = 0 // UI_ERROR_NONE
	UIErrorGeneric UserInterfaceError = 1 // UI_ERROR_GENERIC
	UIErrorChargingRCD UserInterfaceError = 101 // UI_ERROR_CHARGING_RCD
	UIErrorRelays UserInterfaceError = 102 // UI_ERROR_RELAYS
	UIErrorInternalVoltage UserInterfaceError = 104 // UI_ERROR_INTERNAL_VOLTAGE
	UIErrorPowermeter UserInterfaceError = 105 // UI_ERROR_POWERMETER
	UIErrorRCD UserInterfaceError = 106 // UI_ERROR_RCD
	UIErrorSocketMotorStartupOld UserInterfaceError = 107 // UI_ERROR_SOCKET_MOTOR_STARTUP_OLD
	UIErrorMissingpcid UserInterfaceError = 108 // UI_ERROR_MISSINGPCID
	UIErrorNfcreader UserInterfaceError = 109 // UI_ERROR_NFCREADER
	UIErrorProtectiveEarth UserInterfaceError = 201 // UI_ERROR_PROTECTIVE_EARTH
	UIErrorLowSupplyVoltage UserInterfaceError = 202 // UI_ERROR_LOW_SUPPLY_VOLTAGE
	UIErrorInoperative UserInterfaceError = 206 // UI_ERROR_INOPERATIVE
	UIErrorHighSupplyVoltage UserInterfaceError = 208 // UI_ERROR_HIGH_SUPPLY_VOLTAGE
	UIErrorP1pport UserInterfaceError = 209 // UI_ERROR_P1PPORT
	UIErrorModbustcpip UserInterfaceError = 210 // UI_ERROR_MODBUSTCPIP
	UIErrorSocketMotorStartup UserInterfaceError = 211 // UI_ERROR_SOCKET_MOTOR_STARTUP
	UIErrorMissingphase UserInterfaceError = 212 // UI_ERROR_MISSINGPHASE
	UIErrorTicport UserInterfaceError = 213 // UI_ERROR_TICPORT
	UIErrorCharging UserInterfaceError = 301 // UI_ERROR_CHARGING
	UIErrorChargingOvercurrent UserInterfaceError = 302 // UI_ERROR_CHARGING_OVERCURRENT
	UIErrorChargingHFSwitching UserInterfaceError = 303 // UI_ERROR_CHARGING_HF_SWITCHING
	UIErrorCableConnectedTimeout UserInterfaceError = 304 // UI_ERROR_CABLE_CONNECTED_TIMEOUT
	UIErrorTemperatureHigh UserInterfaceError = 401 // UI_ERROR_TEMPERATURE_HIGH
	UIErrorTemperatureLow UserInterfaceError = 402 // UI_ERROR_TEMPERATURE_LOW
	UIErrorMessage UserInterfaceError = 403 // UI_ERROR_MESSAGE
	UIErrorSocketMotor UserInterfaceError = 404 // UI_ERROR_SOCKET_MOTOR
	UIErrorIllegalMode3PP UserInterfaceError = 405 // UI_ERROR_ILLEGAL_MODE_3_PP
	UIErrorIllegalMode3CP UserInterfaceError = 406 // UI_ERROR_ILLEGAL_MODE_3_CP
	UIErrorTilt UserInterfaceError = 407 // UI_ERROR_TILT
)

var userInterfaceErrorEnum = newEnumInfo(false, []enumMember{
	{"UI_ERROR_NONE", 0, ""},
	{"UI_ERROR_GENERIC", 1, ""},
	{"UI_ERROR_CHARGING_RCD", 101, ""},
	{"UI_ERROR_RELAYS", 102, ""},
	{"UI_ERROR_INTERNAL_VOLTAGE", 104, ""},
	{"UI_ERROR_POWERMETER", 105, ""},
	{"UI_ERROR_RCD", 106, ""},
	{"UI_ERROR_SOCKET_MOTOR_STARTUP_OLD", 107, ""},
	{"UI_ERROR_MISSINGPCID", 108, ""},
	{"UI_ERROR_NFCREADER", 109, ""},
	{"UI_ERROR_PROTECTIVE_EARTH", 201, ""},
	{"UI_ERROR_LOW_SUPPLY_VOLTAGE", 202, ""},
	{"UI_ERROR_INOPERATIVE", 206, ""},
	{"UI_ERROR_HIGH_SUPPLY_VOLTAGE", 208, ""},
	{"UI_ERROR_P1PPORT", 209, ""},
	{"UI_ERROR_MODBUSTCPIP", 210, ""},
	{"UI_ERROR_SOCKET_MOTOR_STARTUP", 211, ""},
	{"UI_ERROR_MISSINGPHASE", 212, ""},
	{"UI_ERROR_TICPORT", 213, ""},
	{"UI_ERROR_CHARGING", 301, ""},
	{"UI_ERROR_CHARGING_OVERCURRENT", 302, ""},
	{"UI_ERROR_CHARGING_HF_SWITCHING", 303, ""},
	{"UI_ERROR_CABLE_CONNECTED_TIMEOUT", 304, ""},
	{"UI_ERROR_TEMPERATURE_HIGH", 401, ""},
	{"UI_ERROR_TEMPERATURE_LOW", 402, ""},
	{"UI_ERROR_MESSAGE", 403, ""},
	{"UI_ERROR_SOCKET_MOTOR", 404, ""},
	{"UI_ERROR_ILLEGAL_MODE_3_PP", 405, ""},
	{"UI_ERROR_ILLEGAL_MODE_3_CP", 406, ""},
	{"UI_ERROR_TILT", 407, ""},
})

// String ports EUserInterfaceError.ToString() (.NET Framework Enum.ToString).
func (v UserInterfaceError) String() string { return userInterfaceErrorEnum.format(int64(v)) }

// OcppMeasurand ports the enum EOcppMeasurand (ACENetwork/ICUNetwork/EOcppMeasurand.cs).
// String returns the C# member name (Enum.ToString).
type OcppMeasurand int

// OcppMeasurand values, 1:1 with EOcppMeasurand.
const (
	MeasNone OcppMeasurand = 0 // MEAS_NONE
	MeasEnergyActiveExport OcppMeasurand = 1 // MEAS_ENERGY_ACTIVE_EXPORT
	MeasEnergyActiveImport OcppMeasurand = 2 // MEAS_ENERGY_ACTIVE_IMPORT
	MeasEnergyReactiveExport OcppMeasurand = 3 // MEAS_ENERGY_REACTIVE_EXPORT
	MeasEnergyReactiveImport OcppMeasurand = 4 // MEAS_ENERGY_REACTIVE_IMPORT
	MeasEnergyActiveExportInterval OcppMeasurand = 5 // MEAS_ENERGY_ACTIVE_EXPORT_INTERVAL
	MeasEnergyActiveImportInterval OcppMeasurand = 6 // MEAS_ENERGY_ACTIVE_IMPORT_INTERVAL
	MeasEnergyReactiveExportInterval OcppMeasurand = 7 // MEAS_ENERGY_REACTIVE_EXPORT_INTERVAL
	MeasEnergyReactiveImportInterval OcppMeasurand = 8 // MEAS_ENERGY_REACTIVE_IMPORT_INTERVAL
	MeasPowerActiveExport OcppMeasurand = 9 // MEAS_POWER_ACTIVE_EXPORT
	MeasPowerActiveImport OcppMeasurand = 10 // MEAS_POWER_ACTIVE_IMPORT
	MeasPowerReactiveExport OcppMeasurand = 11 // MEAS_POWER_REACTIVE_EXPORT
	MeasPowerReactiveImport OcppMeasurand = 12 // MEAS_POWER_REACTIVE_IMPORT
	MeasCurrentExport OcppMeasurand = 13 // MEAS_CURRENT_EXPORT
	MeasCurrentImport OcppMeasurand = 14 // MEAS_CURRENT_IMPORT
	MeasVoltage OcppMeasurand = 15 // MEAS_VOLTAGE
	MeasTemp OcppMeasurand = 16 // MEAS_TEMP
	MeasAlfenCurrentL1 OcppMeasurand = 17 // MEAS_ALFEN_CURRENT_L1
	MeasAlfenCurrentL2 OcppMeasurand = 18 // MEAS_ALFEN_CURRENT_L2
	MeasAlfenCurrentL3 OcppMeasurand = 19 // MEAS_ALFEN_CURRENT_L3
	MeasAlfenCurrentMax OcppMeasurand = 20 // MEAS_ALFEN_CURRENT_MAX
	MeasPowerFactor OcppMeasurand = 21 // MEAS_POWER_FACTOR
	MeasCurrentOffered OcppMeasurand = 22 // MEAS_CURRENT_OFFERED
	MeasPowerOffered OcppMeasurand = 23 // MEAS_POWER_OFFERED
	MeasFrequency OcppMeasurand = 24 // MEAS_FREQUENCY
	MeasRPM OcppMeasurand = 25 // MEAS_RPM
	MeasSOC OcppMeasurand = 26 // MEAS_SOC
	MeasEnergyActiveNet OcppMeasurand = 29 // MEAS_ENERGY_ACTIVE_NET
	MeasEnergyReactiveNet OcppMeasurand = 30 // MEAS_ENERGY_REACTIVE_NET
	MeasEnergyApparentNet OcppMeasurand = 31 // MEAS_ENERGY_APPARENT_NET
	MeasEnergyApparentImport OcppMeasurand = 32 // MEAS_ENERGY_APPARENT_IMPORT
	MeasEnergyApparentExport OcppMeasurand = 33 // MEAS_ENERGY_APPARENT_EXPORT
)

var ocppMeasurandEnum = newEnumInfo(false, []enumMember{
	{"MEAS_NONE", 0, ""},
	{"MEAS_ENERGY_ACTIVE_EXPORT", 1, ""},
	{"MEAS_ENERGY_ACTIVE_IMPORT", 2, ""},
	{"MEAS_ENERGY_REACTIVE_EXPORT", 3, ""},
	{"MEAS_ENERGY_REACTIVE_IMPORT", 4, ""},
	{"MEAS_ENERGY_ACTIVE_EXPORT_INTERVAL", 5, ""},
	{"MEAS_ENERGY_ACTIVE_IMPORT_INTERVAL", 6, ""},
	{"MEAS_ENERGY_REACTIVE_EXPORT_INTERVAL", 7, ""},
	{"MEAS_ENERGY_REACTIVE_IMPORT_INTERVAL", 8, ""},
	{"MEAS_POWER_ACTIVE_EXPORT", 9, ""},
	{"MEAS_POWER_ACTIVE_IMPORT", 10, ""},
	{"MEAS_POWER_REACTIVE_EXPORT", 11, ""},
	{"MEAS_POWER_REACTIVE_IMPORT", 12, ""},
	{"MEAS_CURRENT_EXPORT", 13, ""},
	{"MEAS_CURRENT_IMPORT", 14, ""},
	{"MEAS_VOLTAGE", 15, ""},
	{"MEAS_TEMP", 16, ""},
	{"MEAS_ALFEN_CURRENT_L1", 17, ""},
	{"MEAS_ALFEN_CURRENT_L2", 18, ""},
	{"MEAS_ALFEN_CURRENT_L3", 19, ""},
	{"MEAS_ALFEN_CURRENT_MAX", 20, ""},
	{"MEAS_POWER_FACTOR", 21, ""},
	{"MEAS_CURRENT_OFFERED", 22, ""},
	{"MEAS_POWER_OFFERED", 23, ""},
	{"MEAS_FREQUENCY", 24, ""},
	{"MEAS_RPM", 25, ""},
	{"MEAS_SOC", 26, ""},
	{"MEAS_ENERGY_ACTIVE_NET", 29, ""},
	{"MEAS_ENERGY_REACTIVE_NET", 30, ""},
	{"MEAS_ENERGY_APPARENT_NET", 31, ""},
	{"MEAS_ENERGY_APPARENT_IMPORT", 32, ""},
	{"MEAS_ENERGY_APPARENT_EXPORT", 33, ""},
})

// String ports EOcppMeasurand.ToString() (.NET Framework Enum.ToString).
func (v OcppMeasurand) String() string { return ocppMeasurandEnum.format(int64(v)) }

// EnergyMeterMeasurand ports the enum EnergyMeterMeasurand (ACENetwork/ICUNetwork/EnergyMeterMeasurand.cs).
// String returns the C# member name (Enum.ToString).
type EnergyMeterMeasurand int

// EnergyMeterMeasurand values, 1:1 with EnergyMeterMeasurand.
const (
	EnergyMeterVoltageL1N EnergyMeterMeasurand = 0 // VOLTAGE_L1N
	EnergyMeterVoltageL2N EnergyMeterMeasurand = 1 // VOLTAGE_L2N
	EnergyMeterVoltageL3N EnergyMeterMeasurand = 2 // VOLTAGE_L3N
	EnergyMeterVoltageL1L2 EnergyMeterMeasurand = 3 // VOLTAGE_L1L2
	EnergyMeterVoltageL2L3 EnergyMeterMeasurand = 4 // VOLTAGE_L2L3
	EnergyMeterVoltageL3L1 EnergyMeterMeasurand = 5 // VOLTAGE_L3L1
	EnergyMeterCurrentN EnergyMeterMeasurand = 6 // CURRENT_N
	EnergyMeterCurrentL1 EnergyMeterMeasurand = 7 // CURRENT_L1
	EnergyMeterCurrentL2 EnergyMeterMeasurand = 8 // CURRENT_L2
	EnergyMeterCurrentL3 EnergyMeterMeasurand = 9 // CURRENT_L3
	EnergyMeterCurrentSum EnergyMeterMeasurand = 10 // CURRENT_SUM
	EnergyMeterCosphiL1 EnergyMeterMeasurand = 11 // COSPHI_L1
	EnergyMeterCosphiL2 EnergyMeterMeasurand = 12 // COSPHI_L2
	EnergyMeterCosphiL3 EnergyMeterMeasurand = 13 // COSPHI_L3
	EnergyMeterCosphiSum EnergyMeterMeasurand = 14 // COSPHI_SUM
	EnergyMeterFrequency EnergyMeterMeasurand = 15 // FREQUENCY
	EnergyMeterPowerRealL1 EnergyMeterMeasurand = 16 // POWER_REAL_L1
	EnergyMeterPowerRealL2 EnergyMeterMeasurand = 17 // POWER_REAL_L2
	EnergyMeterPowerRealL3 EnergyMeterMeasurand = 18 // POWER_REAL_L3
	EnergyMeterPowerRealSum EnergyMeterMeasurand = 19 // POWER_REAL_SUM
	EnergyMeterPowerApparentL1 EnergyMeterMeasurand = 20 // POWER_APPARENT_L1
	EnergyMeterPowerApparentL2 EnergyMeterMeasurand = 21 // POWER_APPARENT_L2
	EnergyMeterPowerApparentL3 EnergyMeterMeasurand = 22 // POWER_APPARENT_L3
	EnergyMeterPowerApparentSum EnergyMeterMeasurand = 23 // POWER_APPARENT_SUM
	EnergyMeterPowerReactiveL1 EnergyMeterMeasurand = 24 // POWER_REACTIVE_L1
	EnergyMeterPowerReactiveL2 EnergyMeterMeasurand = 25 // POWER_REACTIVE_L2
	EnergyMeterPowerReactiveL3 EnergyMeterMeasurand = 26 // POWER_REACTIVE_L3
	EnergyMeterPowerReactiveSum EnergyMeterMeasurand = 27 // POWER_REACTIVE_SUM
	EnergyMeterEnergyRealDeliveredL1 EnergyMeterMeasurand = 28 // ENERGY_REAL_DELIVERED_L1
	EnergyMeterEnergyRealDeliveredL2 EnergyMeterMeasurand = 29 // ENERGY_REAL_DELIVERED_L2
	EnergyMeterEnergyRealDeliveredL3 EnergyMeterMeasurand = 30 // ENERGY_REAL_DELIVERED_L3
	EnergyMeterEnergyRealDeliveredSum EnergyMeterMeasurand = 31 // ENERGY_REAL_DELIVERED_SUM
	EnergyMeterEnergyRealConsumedL1 EnergyMeterMeasurand = 32 // ENERGY_REAL_CONSUMED_L1
	EnergyMeterEnergyRealConsumedL2 EnergyMeterMeasurand = 33 // ENERGY_REAL_CONSUMED_L2
	EnergyMeterEnergyRealConsumedL3 EnergyMeterMeasurand = 34 // ENERGY_REAL_CONSUMED_L3
	EnergyMeterEnergyRealConsumedSum EnergyMeterMeasurand = 35 // ENERGY_REAL_CONSUMED_SUM
	EnergyMeterEnergyApparentL1 EnergyMeterMeasurand = 36 // ENERGY_APPARENT_L1
	EnergyMeterEnergyApparentL2 EnergyMeterMeasurand = 37 // ENERGY_APPARENT_L2
	EnergyMeterEnergyApparentL3 EnergyMeterMeasurand = 38 // ENERGY_APPARENT_L3
	EnergyMeterEnergyApparentSum EnergyMeterMeasurand = 39 // ENERGY_APPARENT_SUM
	EnergyMeterEnergyReactiveL1 EnergyMeterMeasurand = 40 // ENERGY_REACTIVE_L1
	EnergyMeterEnergyReactiveL2 EnergyMeterMeasurand = 41 // ENERGY_REACTIVE_L2
	EnergyMeterEnergyReactiveL3 EnergyMeterMeasurand = 42 // ENERGY_REACTIVE_L3
	EnergyMeterEnergyReactiveSum EnergyMeterMeasurand = 43 // ENERGY_REACTIVE_SUM
	EnergyMeterMaxMeasurandCount EnergyMeterMeasurand = 44 // MAX_MEASURAND_COUNT
)

var energyMeterMeasurandEnum = newEnumInfo(false, []enumMember{
	{"VOLTAGE_L1N", 0, ""},
	{"VOLTAGE_L2N", 1, ""},
	{"VOLTAGE_L3N", 2, ""},
	{"VOLTAGE_L1L2", 3, ""},
	{"VOLTAGE_L2L3", 4, ""},
	{"VOLTAGE_L3L1", 5, ""},
	{"CURRENT_N", 6, ""},
	{"CURRENT_L1", 7, ""},
	{"CURRENT_L2", 8, ""},
	{"CURRENT_L3", 9, ""},
	{"CURRENT_SUM", 10, ""},
	{"COSPHI_L1", 11, ""},
	{"COSPHI_L2", 12, ""},
	{"COSPHI_L3", 13, ""},
	{"COSPHI_SUM", 14, ""},
	{"FREQUENCY", 15, ""},
	{"POWER_REAL_L1", 16, ""},
	{"POWER_REAL_L2", 17, ""},
	{"POWER_REAL_L3", 18, ""},
	{"POWER_REAL_SUM", 19, ""},
	{"POWER_APPARENT_L1", 20, ""},
	{"POWER_APPARENT_L2", 21, ""},
	{"POWER_APPARENT_L3", 22, ""},
	{"POWER_APPARENT_SUM", 23, ""},
	{"POWER_REACTIVE_L1", 24, ""},
	{"POWER_REACTIVE_L2", 25, ""},
	{"POWER_REACTIVE_L3", 26, ""},
	{"POWER_REACTIVE_SUM", 27, ""},
	{"ENERGY_REAL_DELIVERED_L1", 28, ""},
	{"ENERGY_REAL_DELIVERED_L2", 29, ""},
	{"ENERGY_REAL_DELIVERED_L3", 30, ""},
	{"ENERGY_REAL_DELIVERED_SUM", 31, ""},
	{"ENERGY_REAL_CONSUMED_L1", 32, ""},
	{"ENERGY_REAL_CONSUMED_L2", 33, ""},
	{"ENERGY_REAL_CONSUMED_L3", 34, ""},
	{"ENERGY_REAL_CONSUMED_SUM", 35, ""},
	{"ENERGY_APPARENT_L1", 36, ""},
	{"ENERGY_APPARENT_L2", 37, ""},
	{"ENERGY_APPARENT_L3", 38, ""},
	{"ENERGY_APPARENT_SUM", 39, ""},
	{"ENERGY_REACTIVE_L1", 40, ""},
	{"ENERGY_REACTIVE_L2", 41, ""},
	{"ENERGY_REACTIVE_L3", 42, ""},
	{"ENERGY_REACTIVE_SUM", 43, ""},
	{"MAX_MEASURAND_COUNT", 44, ""},
})

// String ports EnergyMeterMeasurand.ToString() (.NET Framework Enum.ToString).
func (v EnergyMeterMeasurand) String() string { return energyMeterMeasurandEnum.format(int64(v)) }

// RadioAccessTechnology ports the [Flags] enum ERadioAccessTechnology (ACENetwork/ICUNetwork/ERadioAccessTechnology.cs).
// String returns the C# member name (Enum.ToString).
type RadioAccessTechnology int

// RadioAccessTechnology values, 1:1 with ERadioAccessTechnology.
const (
	RatGPRS RadioAccessTechnology = 1 // GPRS
	RatUMTS RadioAccessTechnology = 2 // UMTS
	RatLTE RadioAccessTechnology = 4 // LTE
)

var radioAccessTechnologyEnum = newEnumInfo(true, []enumMember{
	{"GPRS", 1, ""},
	{"UMTS", 2, ""},
	{"LTE", 4, ""},
})

// String ports ERadioAccessTechnology.ToString() (.NET Framework Enum.ToString).
func (v RadioAccessTechnology) String() string { return radioAccessTechnologyEnum.format(int64(v)) }

// AlbConfiguration ports the enum EAlbConfiguration (ACENetwork/ICUNetwork/EAlbConfiguration.cs).
// String returns the C# member name (Enum.ToString).
type AlbConfiguration int

// AlbConfiguration values, 1:1 with EAlbConfiguration.
const (
	AlbNoLicense AlbConfiguration = 0 // NoLicense
	AlbNotConfigered AlbConfiguration = 1 // NotConfigered
	AlbEMS AlbConfiguration = 2 // EMS
	AlbSMTcpCustom AlbConfiguration = 3 // SM_Tcp_Custom
	AlbSMTcpSocomec AlbConfiguration = 4 // SM_Tcp_Socomec
	AlbSMTicLinky AlbConfiguration = 5 // SM_TicLinky
	AlbSMRtu AlbConfiguration = 6 // SM_Rtu
	AlbSMP1Serial AlbConfiguration = 7 // SM_P1_Serial
	AlbSMP1Telnet AlbConfiguration = 8 // SM_P1_Telnet
	AlbSMP1HomeWizard AlbConfiguration = 9 // SM_P1_HomeWizard
	AlbSMP1Unknown AlbConfiguration = 10 // SM_P1_Unknown
)

var albConfigurationEnum = newEnumInfo(false, []enumMember{
	{"NoLicense", 0, ""},
	{"NotConfigered", 1, ""},
	{"EMS", 2, ""},
	{"SM_Tcp_Custom", 3, ""},
	{"SM_Tcp_Socomec", 4, ""},
	{"SM_TicLinky", 5, ""},
	{"SM_Rtu", 6, ""},
	{"SM_P1_Serial", 7, ""},
	{"SM_P1_Telnet", 8, ""},
	{"SM_P1_HomeWizard", 9, ""},
	{"SM_P1_Unknown", 10, ""},
})

// String ports EAlbConfiguration.ToString() (.NET Framework Enum.ToString).
func (v AlbConfiguration) String() string { return albConfigurationEnum.format(int64(v)) }

// SolarChargingMode ports the enum ESolarChargingModes (ACENetwork/ICUNetwork/ESolarChargingModes.cs).
// String returns the C# member name (Enum.ToString).
type SolarChargingMode int

// SolarChargingMode values, 1:1 with ESolarChargingModes.
const (
	SolarChargingOff SolarChargingMode = 0 // SOLAR_CHARGING_OFF
	SolarChargingComfort SolarChargingMode = 1 // SOLAR_CHARGING_COMFORT
	SolarChargingGreen SolarChargingMode = 2 // SOLAR_CHARGING_GREEN
)

var solarChargingModeEnum = newEnumInfo(false, []enumMember{
	{"SOLAR_CHARGING_OFF", 0, "Off"},
	{"SOLAR_CHARGING_COMFORT", 1, "Comfort"},
	{"SOLAR_CHARGING_GREEN", 2, "Green"},
})

// String ports ESolarChargingModes.ToString() (.NET Framework Enum.ToString).
func (v SolarChargingMode) String() string { return solarChargingModeEnum.format(int64(v)) }

// Description ports EnumExtensions.GetEnumDescription: the [Description]
// attribute of the member, else String().
func (v SolarChargingMode) Description() string { return solarChargingModeEnum.description(int64(v)) }

// TariffDisplayOptionsType ports the [Flags] enum TariffDisplayOptionsType (ACENetwork/ICUNetwork/TariffDisplayOptionsType.cs).
// String returns the C# member name (Enum.ToString).
type TariffDisplayOptionsType int

// TariffDisplayOptionsType values, 1:1 with TariffDisplayOptionsType.
const (
	TariffDisplayNone TariffDisplayOptionsType = 0 // NONE
	TariffDisplayDisclaimer TariffDisplayOptionsType = 1 // disclaimer
	TariffDisplayPerKwh TariffDisplayOptionsType = 2 // perKwh
	TariffDisplayPerMinute TariffDisplayOptionsType = 4 // perMinute
	TariffDisplayPerSession TariffDisplayOptionsType = 8 // perSession
	TariffDisplayPerOther TariffDisplayOptionsType = 16 // perOther
	TariffDisplayAdhocOnlyDisclaimer TariffDisplayOptionsType = 32 // adhocOnlyDisclaimer
)

var tariffDisplayOptionsTypeEnum = newEnumInfo(true, []enumMember{
	{"NONE", 0, ""},
	{"disclaimer", 1, ""},
	{"perKwh", 2, ""},
	{"perMinute", 4, ""},
	{"perSession", 8, ""},
	{"perOther", 16, ""},
	{"adhocOnlyDisclaimer", 32, ""},
})

// String ports TariffDisplayOptionsType.ToString() (.NET Framework Enum.ToString).
func (v TariffDisplayOptionsType) String() string { return tariffDisplayOptionsTypeEnum.format(int64(v)) }

// OccpVersion ports the [Flags] enum EOccpVersion (ACENetwork/ICUNetwork/EOccpVersion.cs).
// String returns the C# member name (Enum.ToString).
type OccpVersion int

// OccpVersion values, 1:1 with EOccpVersion.
const (
	OccpVersionNone OccpVersion = 0 // VERSION_NONE
	OccpVersion15 OccpVersion = 1 // VERSION_15
	OccpVersion16 OccpVersion = 2 // VERSION_16
	OccpVersion20 OccpVersion = 4 // VERSION_20
	OccpVersion15_16 OccpVersion = 3 // VERSION_15_16
	OccpVersion16_20 OccpVersion = 6 // VERSION_16_20
	OccpVersion15_16_20 OccpVersion = 7 // VERSION_15_16_20
)

var occpVersionEnum = newEnumInfo(true, []enumMember{
	{"VERSION_NONE", 0, ""},
	{"VERSION_15", 1, ""},
	{"VERSION_16", 2, ""},
	{"VERSION_20", 4, ""},
	{"VERSION_15_16", 3, ""},
	{"VERSION_16_20", 6, ""},
	{"VERSION_15_16_20", 7, ""},
})

// String ports EOccpVersion.ToString() (.NET Framework Enum.ToString).
func (v OccpVersion) String() string { return occpVersionEnum.format(int64(v)) }

// OccpPhase ports the enum EOccpPhase (ACENetwork/ICUNetwork/EOccpPhase.cs).
// String returns the C# member name (Enum.ToString).
type OccpPhase int

// OccpPhase values, 1:1 with EOccpPhase.
const (
	OccpPhaseNone OccpPhase = 0 // PHASE_NONE
	OccpPhaseL1 OccpPhase = 1 // PHASE_L1
	OccpPhaseL2 OccpPhase = 2 // PHASE_L2
	OccpPhaseL3 OccpPhase = 3 // PHASE_L3
	OccpPhaseN OccpPhase = 4 // PHASE_N
	OccpPhaseL1N OccpPhase = 5 // PHASE_L1N
	OccpPhaseL2N OccpPhase = 6 // PHASE_L2N
	OccpPhaseL3N OccpPhase = 7 // PHASE_L3N
	OccpPhaseL1L2 OccpPhase = 8 // PHASE_L1L2
	OccpPhaseL2L3 OccpPhase = 9 // PHASE_L2L3
	OccpPhaseL3L1 OccpPhase = 10 // PHASE_L3L1
)

var occpPhaseEnum = newEnumInfo(false, []enumMember{
	{"PHASE_NONE", 0, ""},
	{"PHASE_L1", 1, ""},
	{"PHASE_L2", 2, ""},
	{"PHASE_L3", 3, ""},
	{"PHASE_N", 4, ""},
	{"PHASE_L1N", 5, ""},
	{"PHASE_L2N", 6, ""},
	{"PHASE_L3N", 7, ""},
	{"PHASE_L1L2", 8, ""},
	{"PHASE_L2L3", 9, ""},
	{"PHASE_L3L1", 10, ""},
})

// String ports EOccpPhase.ToString() (.NET Framework Enum.ToString).
func (v OccpPhase) String() string { return occpPhaseEnum.format(int64(v)) }

// BoardRevision ports the enum EBoardRevision (ACENetwork/ICUNetwork/EBoardRevision.cs).
// String returns the C# member name (Enum.ToString).
type BoardRevision int

// BoardRevision values, 1:1 with EBoardRevision.
const (
	BoardRevisionUnknown BoardRevision = -1 // BOARDREVISION_UNKNOWN
	BoardRevisionDefault BoardRevision = 0 // BOARDREVISION_DEFAULT
	BoardRevisionA BoardRevision = 1 // BOARDREVISION_A
	BoardRevisionB BoardRevision = 2 // BOARDREVISION_B
	BoardRevisionC BoardRevision = 3 // BOARDREVISION_C
	BoardRevisionD BoardRevision = 4 // BOARDREVISION_D
	BoardRevisionE BoardRevision = 5 // BOARDREVISION_E
	BoardRevisionF BoardRevision = 6 // BOARDREVISION_F
	BoardRevisionG BoardRevision = 7 // BOARDREVISION_G
	BoardRevisionH BoardRevision = 8 // BOARDREVISION_H
	BoardRevisionJ BoardRevision = 9 // BOARDREVISION_J
	BoardRevisionK BoardRevision = 10 // BOARDREVISION_K
	BoardRevisionL BoardRevision = 11 // BOARDREVISION_L
	BoardRevisionM BoardRevision = 12 // BOARDREVISION_M
	BoardRevisionN BoardRevision = 13 // BOARDREVISION_N
	BoardRevisionP BoardRevision = 14 // BOARDREVISION_P
	BoardRevisionQ BoardRevision = 15 // BOARDREVISION_Q
	BoardRevisionR BoardRevision = 16 // BOARDREVISION_R
)

var boardRevisionEnum = newEnumInfo(false, []enumMember{
	{"BOARDREVISION_UNKNOWN", -1, ""},
	{"BOARDREVISION_DEFAULT", 0, ""},
	{"BOARDREVISION_A", 1, ""},
	{"BOARDREVISION_B", 2, ""},
	{"BOARDREVISION_C", 3, ""},
	{"BOARDREVISION_D", 4, ""},
	{"BOARDREVISION_E", 5, ""},
	{"BOARDREVISION_F", 6, ""},
	{"BOARDREVISION_G", 7, ""},
	{"BOARDREVISION_H", 8, ""},
	{"BOARDREVISION_J", 9, ""},
	{"BOARDREVISION_K", 10, ""},
	{"BOARDREVISION_L", 11, ""},
	{"BOARDREVISION_M", 12, ""},
	{"BOARDREVISION_N", 13, ""},
	{"BOARDREVISION_P", 14, ""},
	{"BOARDREVISION_Q", 15, ""},
	{"BOARDREVISION_R", 16, ""},
})

// String ports EBoardRevision.ToString() (.NET Framework Enum.ToString).
func (v BoardRevision) String() string { return boardRevisionEnum.format(int64(v)) }

// BoardAssy ports the enum EBoardAssy (ACENetwork/ICUNetwork/EBoardAssy.cs).
// String returns the C# member name (Enum.ToString).
type BoardAssy int

// BoardAssy values, 1:1 with EBoardAssy.
const (
	BoardAssyUnknown BoardAssy = -1 // BOARDASSEMBLY_UNKNOWN
	BoardAssyDefault BoardAssy = 0 // BOARDASSEMBLY_DEFAULT
	BoardAssy00 BoardAssy = 1 // BOARDASSEMBLY_00
	BoardAssy01 BoardAssy = 2 // BOARDASSEMBLY_01
	BoardAssy02 BoardAssy = 3 // BOARDASSEMBLY_02
	BoardAssy03 BoardAssy = 4 // BOARDASSEMBLY_03
	BoardAssy04 BoardAssy = 5 // BOARDASSEMBLY_04
	BoardAssy05 BoardAssy = 6 // BOARDASSEMBLY_05
	BoardAssy06 BoardAssy = 7 // BOARDASSEMBLY_06
	BoardAssy07 BoardAssy = 8 // BOARDASSEMBLY_07
	BoardAssy08 BoardAssy = 9 // BOARDASSEMBLY_08
	BoardAssy09 BoardAssy = 10 // BOARDASSEMBLY_09
	BoardAssy10 BoardAssy = 11 // BOARDASSEMBLY_10
	BoardAssy11 BoardAssy = 12 // BOARDASSEMBLY_11
	BoardAssy12 BoardAssy = 13 // BOARDASSEMBLY_12
	BoardAssy13 BoardAssy = 14 // BOARDASSEMBLY_13
	BoardAssy14 BoardAssy = 15 // BOARDASSEMBLY_14
	BoardAssy15 BoardAssy = 16 // BOARDASSEMBLY_15
)

var boardAssyEnum = newEnumInfo(false, []enumMember{
	{"BOARDASSEMBLY_UNKNOWN", -1, ""},
	{"BOARDASSEMBLY_DEFAULT", 0, ""},
	{"BOARDASSEMBLY_00", 1, ""},
	{"BOARDASSEMBLY_01", 2, ""},
	{"BOARDASSEMBLY_02", 3, ""},
	{"BOARDASSEMBLY_03", 4, ""},
	{"BOARDASSEMBLY_04", 5, ""},
	{"BOARDASSEMBLY_05", 6, ""},
	{"BOARDASSEMBLY_06", 7, ""},
	{"BOARDASSEMBLY_07", 8, ""},
	{"BOARDASSEMBLY_08", 9, ""},
	{"BOARDASSEMBLY_09", 10, ""},
	{"BOARDASSEMBLY_10", 11, ""},
	{"BOARDASSEMBLY_11", 12, ""},
	{"BOARDASSEMBLY_12", 13, ""},
	{"BOARDASSEMBLY_13", 14, ""},
	{"BOARDASSEMBLY_14", 15, ""},
	{"BOARDASSEMBLY_15", 16, ""},
})

// String ports EBoardAssy.ToString() (.NET Framework Enum.ToString).
func (v BoardAssy) String() string { return boardAssyEnum.format(int64(v)) }

// TamperState ports the enum ETamperState (ACEServiceInstaller/ICUServiceInstaller/ETamperState.cs).
// String returns the C# member name (Enum.ToString).
type TamperState int

// TamperState values, 1:1 with ETamperState.
const (
	TamperStateUnknown TamperState = 0 // Unknown
	TamperStateNotTampered TamperState = 1 // Not_Tampered
	TamperStateTamperActive TamperState = 2 // TamperActive
	TamperStateTampered TamperState = 3 // Tampered
	TamperStateTamperedAck TamperState = 4 // Tampered_Ack
)

var tamperStateEnum = newEnumInfo(false, []enumMember{
	{"Unknown", 0, ""},
	{"Not_Tampered", 1, ""},
	{"TamperActive", 2, ""},
	{"Tampered", 3, ""},
	{"Tampered_Ack", 4, ""},
})

// String ports ETamperState.ToString() (.NET Framework Enum.ToString).
func (v TamperState) String() string { return tamperStateEnum.format(int64(v)) }

// BitFlags ports the [Flags] enum BitFlags (ACEServiceInstaller/ICUServiceInstaller/BitFlags.cs).
// String returns the C# member name (Enum.ToString).
type BitFlags int

// BitFlags values, 1:1 with BitFlags.
const (
	BitFlagsFirst BitFlags = 1 // First
	BitFlagsSecond BitFlags = 2 // Second
	BitFlagsThird BitFlags = 3 // Third
	BitFlagsFourth BitFlags = 4 // Fourth
	BitFlagsFifth BitFlags = 5 // Fifth
	BitFlagsSixth BitFlags = 6 // Sixth
	BitFlagsSeventh BitFlags = 7 // Seventh
	BitFlagsEighth BitFlags = 8 // Eighth
)

var bitFlagsEnum = newEnumInfo(true, []enumMember{
	{"First", 1, ""},
	{"Second", 2, ""},
	{"Third", 3, ""},
	{"Fourth", 4, ""},
	{"Fifth", 5, ""},
	{"Sixth", 6, ""},
	{"Seventh", 7, ""},
	{"Eighth", 8, ""},
})

// String ports BitFlags.ToString() (.NET Framework Enum.ToString).
func (v BitFlags) String() string { return bitFlagsEnum.format(int64(v)) }

