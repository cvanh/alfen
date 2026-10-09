using System;
using Serilog;

namespace ICUNetwork;

public class ICUChargingProfile
{
	private readonly ILogger Logger = Log.ForContext<ICUChargingProfile>();

	public int ConnectorId { get; set; }

	public int ChargingProfileId { get; set; }

	public ICUChargingProfileKind ChargingProfileKind { get; set; }

	public ICURecurrencyKind RecurrencyKind { get; set; }

	public ICUChargingProfilePurposeType ChargingProfilePurpose { get; set; }

	public DateTime? StartSchedule { get; set; }

	public DateTime? ValidFrom { get; set; }

	public DateTime? ValidTo { get; set; } = DateTime.MinValue;

	public int StackLevel { get; set; }

	public int TransactionId { get; set; }

	public bool UseLocalTime { get; set; }

	public bool UseRandomisedDelay { get; set; }

	public ICUChargingRateUnit ChargingRateUnit { get; set; }

	public ICUChargingProfile()
	{
	}

	public ICUChargingProfile(int connectorId, dynamic chargingProfileJson)
	{
		ConnectorId = connectorId;
		try
		{
			ChargingProfileId = Convert.ToInt32(chargingProfileJson["chargingProfileId"]);
			ChargingProfileKind = Enum.Parse(typeof(ICUChargingProfileKind), chargingProfileJson["chargingProfileKind"], true);
			RecurrencyKind = Enum.Parse(typeof(ICURecurrencyKind), chargingProfileJson["recurrencyKind"], true);
			ChargingProfilePurpose = Enum.Parse(typeof(ICUChargingProfilePurposeType), chargingProfileJson["chargingProfilePurpose"], true);
			StartSchedule = Convert.ToDateTime(chargingProfileJson["startSchedule"]);
			ValidFrom = ((chargingProfileJson["validFrom"] != null) ? Convert.ToDateTime(chargingProfileJson["validFrom"]) : null);
			ValidTo = ((chargingProfileJson["validTo"] != null) ? Convert.ToDateTime(chargingProfileJson["validTo"]) : null);
			StackLevel = Convert.ToInt32(chargingProfileJson["stackLevel"]);
			TransactionId = Convert.ToInt32(chargingProfileJson["transactionId"]);
			UseLocalTime = Convert.ToBoolean(chargingProfileJson["useLocalTime"]);
			UseRandomisedDelay = Convert.ToBoolean(chargingProfileJson["useRandomisedDelay"]);
			ChargingRateUnit = Enum.Parse(typeof(ICUChargingRateUnit), chargingProfileJson["chargingRateUnit"], true);
		}
		catch (Exception exception)
		{
			Logger.Error(exception, "");
		}
	}

	public override string ToString()
	{
		return $"Profile {ChargingProfileId}";
	}
}
