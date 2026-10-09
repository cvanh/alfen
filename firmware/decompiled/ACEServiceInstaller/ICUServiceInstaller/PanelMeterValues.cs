using System.Net;
using ICUNetwork;
using ICUServiceInstaller.Utils;
using Serilog;

namespace ICUServiceInstaller;

public class PanelMeterValues : PanelBase
{
	private readonly ILogger Logger = Log.ForContext<PanelMeterValues>();

	protected static int s_nUpdateInterval = 1000;

	protected static int s_nMaxNumberOfSecondsToDisplay = 30;

	public PanelMeterValues(MainWindow parent, string pageID = "", bool showBorder = true)
		: base(parent, pageID, fIndent: false, showBorder)
	{
		Title = "Meter values";
		Tooltip = "Energy meter values";
		IconName = "graphic.png";
		StartUpdateTimer(s_nUpdateInterval);
	}

	public override bool OnChangeDevice(ICUDevice newDevice, ICUDevice previousDevice)
	{
		ClearPanel();
		if (!(newDevice is ICULanDevice))
		{
			return true;
		}
		Logger.AddChargerContext(m_currentDevice).Information("PageView, {Panel}", Title);
		AddHeader(Tooltip);
		if (newDevice.NumberOfSockets == 2)
		{
			using DoubleColumnMode doubleColumnMode = new DoubleColumnMode(this);
			AddSmallHeader("Socket 1");
			AddVoltages(8737);
			AddCurrents(8737);
			doubleColumnMode.NextColumn();
			AddSmallHeader("Socket 2");
			AddVoltages(12833);
			AddCurrents(12833);
		}
		else
		{
			AddSmallHeader("Voltages");
			AddVoltages(8737);
			AddSmallHeader("Currents");
			AddCurrents(8737);
		}
		return true;
	}

	protected void AddVoltages(ushort id)
	{
		AddReadOnlyText(id, 3, "Voltage L1N (V)");
		AddReadOnlyText(id, 4, "Voltage L2N (V)");
		AddReadOnlyText(id, 5, "Voltage L3N (V)");
	}

	protected void AddCurrents(ushort id)
	{
		AddReadOnlyText(id, 10, "Current L1 (A)");
		AddReadOnlyText(id, 11, "Current L2 (A)");
		AddReadOnlyText(id, 12, "Current L3 (A)");
	}

	protected override void OnUpdateTick()
	{
		ICULanDevice currentDevice = m_currentDevice;
		if (currentDevice != null && currentDevice.IsConnected && currentDevice.LastHttpStatusCode == HttpStatusCode.OK)
		{
			if (currentDevice.NumberOfSockets > 1)
			{
				currentDevice.UpdateCategories("meter1", "meter2");
			}
			else
			{
				currentDevice.UpdateCategories("meter1");
			}
		}
	}
}
