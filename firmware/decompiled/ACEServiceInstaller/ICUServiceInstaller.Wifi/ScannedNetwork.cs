using ICUServiceInstaller.Enums;

namespace ICUServiceInstaller.Wifi;

public class ScannedNetwork
{
	public string Ssid { get; set; }

	public int SignalStrength { get; set; }

	public SupportedWifiSecurityType Security { get; set; }

	public int Band { get; set; }
}
