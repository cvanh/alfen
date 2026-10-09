using System;
using System.Collections.Generic;

namespace ICUIWSConnection;

public class IWSFirmwareFeatures
{
	[Flags]
	public enum Features : uint
	{
		None = 0u,
		LoadBalancing_SCN = 1u,
		LoadBalancing_Static = 2u,
		LoadBalancing_Active = 4u,
		HighPowerSockets = 0x10u,
		RFIDReader = 0x100u,
		ISO15118 = 0x200u,
		PersonalizedDisplay = 0x1000u,
		Mobile3G4G = 0x10000u,
		Payment_Options = 0x100000u,
		Expose_SmartMeterData = 0x1000000u,
		ObjectID = 0x80000000u
	}

	public static IList<string> GetFeatureTextLongList(Features requestedFeatures, bool isAhp, bool isDC)
	{
		if (requestedFeatures == Features.None)
		{
			return new List<string> { "None" };
		}
		List<string> list = new List<string>();
		if ((requestedFeatures & Features.LoadBalancing_SCN) != 0)
		{
			list.Add("Smart Charging Network");
		}
		if ((requestedFeatures & (Features.LoadBalancing_SCN | Features.LoadBalancing_Active)) != 0)
		{
			list.Add("Active load balancing");
		}
		if ((requestedFeatures & (Features.LoadBalancing_SCN | Features.LoadBalancing_Static | Features.LoadBalancing_Active)) != 0)
		{
			list.Add("Static Load balancing");
		}
		if ((requestedFeatures & Features.HighPowerSockets) != 0 && !isDC)
		{
			list.Add("32A output per socket");
		}
		if ((requestedFeatures & Features.RFIDReader) != 0)
		{
			list.Add("RFID reader");
		}
		if ((requestedFeatures & Features.ISO15118) != 0)
		{
			list.Add("ISO15118");
		}
		if ((requestedFeatures & Features.PersonalizedDisplay) != 0)
		{
			list.Add("Personalized display");
		}
		if ((requestedFeatures & Features.Mobile3G4G) != 0 && !isAhp)
		{
			list.Add("Mobile Technology 3G & 4G");
		}
		if ((requestedFeatures & Features.Payment_Options) != 0)
		{
			list.Add("Direct Payment Solutions");
		}
		return list;
	}

	public static string GetFeatureTextLong(Features requestedFeatures, bool isAhp, bool isDC, string joinText = ", ")
	{
		return string.Join(joinText, GetFeatureTextLongList(requestedFeatures, isAhp, isDC));
	}

	public static bool IsFeatureUnlocked(Version firmwareVersion, uint featureEnabledFlags, Features feature, bool isAHP)
	{
		if (!isAHP && firmwareVersion < new Version("3.4.0"))
		{
			if (feature == Features.LoadBalancing_SCN && (featureEnabledFlags & (uint)feature) == 0)
			{
				return false;
			}
			return true;
		}
		if (isAHP && firmwareVersion < new Version("1.4.0"))
		{
			return true;
		}
		return (featureEnabledFlags & (uint)feature) != 0;
	}
}
