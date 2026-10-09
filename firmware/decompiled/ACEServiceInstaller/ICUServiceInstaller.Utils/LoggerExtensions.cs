using ICUNetwork;
using Serilog;

namespace ICUServiceInstaller.Utils;

internal static class LoggerExtensions
{
	internal static ILogger AddChargerContext(this ILogger logger, ICULanDevice lanDevice)
	{
		if (lanDevice == null)
		{
			return logger;
		}
		if (lanDevice.HasSCNNetwork)
		{
			string.Format("{0:X}", (lanDevice.SCNNetwork + "_" + lanDevice.LoginData.Password).GetHashCode());
			logger = logger.ForContext("SCN", $"{lanDevice.SCNNetwork.GetHashCode():X}").ForContext("ScnPasskey", string.Format("{0:X}", (lanDevice.SCNNetwork + "_" + lanDevice.LoginData.Password).GetHashCode()));
		}
		logger = logger.ForContext("SolarChargingMode", lanDevice.GetSolarChargingMode().GetEnumDescription() ?? "");
		if (lanDevice.GetSolarChargingMode() != ESolarChargingModes.SOLAR_CHARGING_OFF)
		{
			logger = logger.ForContext("ComfortLevel", $"{lanDevice.GetPropertyDouble(12928, 2):F0}").ForContext("GreenShare", $"{lanDevice.GetPropertyInt(12928, 3)}");
		}
		return logger.ForContext("ChargerID", $"{lanDevice.SerialNumber?.GetHashCode():X}").ForContext("Manual", $"{lanDevice.IsManuallyAdded}").ForContext("Model", $"{lanDevice.ModelType}")
			.ForContext("Firmware", $"{lanDevice.FirmwareVersionNumber}")
			.ForContext("FixedLanIp", $"{lanDevice.HasFixedLanIpAddress}")
			.ForContext("BootCount", $"{lanDevice.GetPropertyInt(8278, 0)}")
			.ForContext("BackOffice", lanDevice.HasBackOfficeConfigured)
			.ForContext("MaxSmartMeterCurrent", $"{lanDevice.GetPropertyDouble(8295, 0):F2}")
			.ForContext("AlbSafeCurrent", $"{lanDevice.GetPropertyDouble(8296, 0):F2}")
			.ForContext("AlbConfig", lanDevice.GetAlbConfiguration().ToString());
	}
}
