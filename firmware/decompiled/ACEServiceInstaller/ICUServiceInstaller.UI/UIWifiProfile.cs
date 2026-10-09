using ICUServiceInstaller.Enums;
using Xwt.Drawing;

namespace ICUServiceInstaller.UI;

public class UIWifiProfile
{
	public string Ssid { get; set; }

	public string Password { get; set; }

	public int SignalStrength { get; set; }

	public WifiSignalStrength SignalType { get; set; }

	public SupportedWifiSecurityType WifiSecurityType { get; set; }

	public bool IsConnected { get; set; }

	public Image Icon
	{
		get
		{
			string iconPath = GetIconPath(SignalType);
			return Image.FromResource(typeof(App), AppProperties.ResourcePath(iconPath)).Scale(0.8);
		}
	}

	public UIWifiProfile(int signalStrength)
	{
		SignalType = GetSignalStrengthType(signalStrength);
	}

	public override string ToString()
	{
		return $"SSID: {Ssid} \nStrength: {SignalStrength} \nSecurity: {WifiSecurityType.GetDescription()}";
	}

	private WifiSignalStrength GetSignalStrengthType(int signalStrength)
	{
		if (signalStrength < -71)
		{
			return WifiSignalStrength.Weak;
		}
		if (signalStrength < -61)
		{
			return WifiSignalStrength.Fair;
		}
		if (signalStrength < -51)
		{
			return WifiSignalStrength.Good;
		}
		return WifiSignalStrength.Excellent;
	}

	private string GetIconPath(WifiSignalStrength signalType)
	{
		return "wifi-" + signalType.ToString().ToLower() + ".png";
	}
}
