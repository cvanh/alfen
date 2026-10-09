package device

// Key identifies one object-dictionary entry (ICUProperty.Id / SubId).
type Key struct {
	ID  uint16
	Sub byte
}

// Object-dictionary indexes the ported ICULanDevice members and
// PanelInformation read. Names follow the EDS ParameterName where EDS.xml has
// one; the others are named after how the C# uses them. Hex in the comments.
const (
	PropManufacturerDeviceName uint16 = 4104  // 0x1008 "Platform type"; SupportsModem/SupportsNFC
	PropHardwareVersion        uint16 = 4105  // 0x1009 GetHWVersion fallback
	PropSoftwareVersion        uint16 = 4106  // 0x100A FirmwareVersionNumber (isah.PropSoftwareVersion)
	PropBoardRevision          uint16 = 8269  // 0x204D sub 1/2 controller, 3/4 power board revision/assembly
	PropChargePointModel       uint16 = 8272  // 0x2050 sysChargePointModel
	PropObjectID               uint16 = 8273  // 0x2051 sysChargePointSerialNumber (isah.PropObjectID)
	PropChargeBoxIdentity      uint16 = 8275  // 0x2053 sysChargeBoxIdentity
	PropNFCInfo                uint16 = 8276  // 0x2054 NFC "#N:" version text (firmware < 4.3.0)
	PropChargePointVendor      uint16 = 8277  // 0x2055 sysChargePointVendor
	PropDateTime               uint16 = 8281  // 0x2059 sysDateTime (ms since the Unix epoch)
	PropTimeZone               uint16 = 8282  // 0x205A sysTimeZone (minutes / 6)
	PropDaylightSavings        uint16 = 8283  // 0x205B sysDaylightSavings
	PropPosition               uint16 = 8284  // 0x205C sysPosition: sub 1 latitude, sub 2 longitude
	PropPlugAndChargeID        uint16 = 8291  // 0x2063 sysPlugAndChargeIdentifier (PanelInformation sync-time check)
	PropLoadBalancingMode      uint16 = 8292  // 0x2064 sysLoadBalancingMode (GetAlbConfiguration, mask 2)
	PropTimeZoneMinutes        uint16 = 8302  // 0x206E sysTimeZoneMinutes
	PropConnectMethod          uint16 = 8311  // 0x2077 commConnectMethod (HasBackOfficeConfigured)
	PropIPAddress2             uint16 = 8317  // 0x207D commIPaddress2, sub 2 isFixed
	PropSupportedRATs          uint16 = 8350  // 0x209E SupportsRats
	PropNetworkProfile1        uint16 = 8432  // 0x20F0 network profiles 8432..8435: sub 1 OCPP version, 6 backoffice, 14 priority
	PropNetworkProfile4        uint16 = 8435  // 0x20F3
	PropModemManufacturer      uint16 = 8472  // 0x2118
	PropModemModel             uint16 = 8473  // 0x2119
	PropModemRevision          uint16 = 8480  // 0x2120
	PropModemIMEI              uint16 = 8481  // 0x2121
	PropSocketType1            uint16 = 8485  // 0x2125 mainSocketType (socket 1)
	PropEichrechtEnabled       uint16 = 8554  // 0x216A mainEichrechtEnabled
	PropLastConfigChange       uint16 = 8583  // 0x2187 ms since the Unix epoch (LastUpdate)
	PropP1Source               uint16 = 8593  // 0x2191 sub 1: P1 serial/telnet/HomeWizard
	PropLicenseKey             uint16 = 8609  // 0x21A1 (isah.PropLicenseKey)
	PropFeatureFlags           uint16 = 8610  // 0x21A2 (isah.PropFeatureFlags)
	PropTargetRDP              uint16 = 8625  // 0x21B1 systargetRDP
	PropServiceEnabled         uint16 = 8626  // 0x21B2 sysServiceEnabled (Secure Service Access)
	PropTempAccessExpiration   uint16 = 8627  // 0x21B3 sysTempAccessExpiration
	PropIsAdminPWDefault       uint16 = 8628  // 0x21B4 sysIsAdminPWDefault
	PropTamperSupport          uint16 = 8784  // 0x2250 tamper detection supported when != 0
	PropTamperDetection        uint16 = 8785  // 0x2251 sub 1/2 tamperDetection
	PropModbusSlaveType        uint16 = 9507  // 0x2523 sub 2 SlaveType (1 = Socomec)
	PropModbusTCPIPSlave       uint16 = 9520  // 0x2530 sub 1 options (3 = EMS)
	PropAllowAlphaReleases     uint16 = 9744  // 0x2610
	PropSocketType2            uint16 = 12581 // 0x3125 mainSocketType (socket 2)
	PropNFCReader1             uint16 = 12672 // 0x3180 "HW:..,SW:.." (firmware >= 4.3.0)
	PropNFCReader2             uint16 = 12673 // 0x3181
	PropBootloaderVersion      uint16 = 12674 // 0x3182
	PropDisplayPresent         uint16 = 12895 // 0x325F sub 1 (MaxLogoWidth/Height guard)
	PropDisplaySize            uint16 = 12896 // 0x3260 sub 3 width, sub 4 height
	PropTariffDisplay          uint16 = 12898 // 0x3262 sub 5 TariffDisplayOptionsType
	PropSolarCharging          uint16 = 12928 // 0x3280 sub 1 ESolarChargingModes
	PropWifiEnabled            uint16 = 12932 // 0x3284 sysWifiEnabled
	PropIPAddress3             uint16 = 12933 // 0x3285 commIPaddress3, sub 2 isFixed
	PropWifiHwAvailable        uint16 = 12943 // 0x328F wifiHwAvailable
	PropAlbSource              uint16 = 21015 // 0x5217 active load balancing meter source
	PropSocketBoardDeviceID    uint16 = 33025 // 0x8101 sub n = socket board n (AHP)
	PropSocketBoardHWVersion   uint16 = 33026 // 0x8102
	PropSocketBoardSWVersion   uint16 = 33027 // 0x8103
	PropSocketBoardISO15118    uint16 = 33031 // 0x8107
	PropSocketBoardMeterInfo   uint16 = 33032 // 0x8108
	PropSocketBoardDC          uint16 = 33034 // 0x810A sub 1/2 == 1: DC (isEcogDC)
	PropNFCBoardDeviceID       uint16 = 33281 // 0x8201
	PropNFCBoardHWVersion      uint16 = 33282 // 0x8202
	PropNFCBoardSWVersion      uint16 = 33283 // 0x8203
	PropAuxBoardDeviceID       uint16 = 33537 // 0x8301
	PropAuxBoardHWVersion      uint16 = 33538 // 0x8302
	PropAuxBoardSWVersion      uint16 = 33539 // 0x8303
	PropExtensionBoardAssy     uint16 = 33792 // 0x8400 sub 1 extensionBoardAssy
	PropSocketBoardExtSWInfo   uint16 = 33794 // 0x8402 sub n
)

// AnalyticsProperties are the combined (index<<8 | sub) ids
// ICULanDevice.UpdateAnalyticsProperties requests (ICULanDevice.cs, line 588),
// in the C# order.
var AnalyticsProperties = []uint32{
	2122752, 2123520, 2123776, 5379840, 2433794, 2199809, 2204160, 2117888, 2118400,
	2119168, 3309569, 3309570, 3309571, 2129154, 3310592, 3310850, 2158598, 2158854,
	2159110, 2159366, 2158606, 2158862, 2159118, 2159374, 2127616,
}
