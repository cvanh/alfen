using System.ComponentModel;

namespace ICUServiceInstaller.Enums;

public enum SupportedWifiSecurityType
{
	[Description("WPA PSK security with TKIP")]
	WpaTkipPsk = 2097154,
	[Description("WPA PSK security with AES")]
	WpaAesPsk = 2097156,
	[Description("WPA PSK security with AES & TKIP")]
	WpaMixedPsk = 2097158,
	[Description("WPA2 PSK security with AES")]
	Wpa2AesPsk = 4194308,
	[Description("WPA3 PSK security with AES")]
	Wpa3AesPsk = 16777220,
	[Description("WPA2 PSK security with TKIP")]
	Wpa2TkipPsk = 4194306,
	[Description("WPA2 PSK security with AES & TKIP")]
	Wpa2MixedPsk = 4194310,
	[Description("WPA2 WPA PSK Security with AES")]
	Wpa2WpaAesPsk = 6291460,
	[Description("WPA2 WPA PSK Security with AES & TKIP")]
	Wpa2WpaMixedPsk = 6291462,
	[Description("WPA3 WPA2 PSK Security with AES")]
	Wpa3Wpa2MixedPskAes = 20971524
}
