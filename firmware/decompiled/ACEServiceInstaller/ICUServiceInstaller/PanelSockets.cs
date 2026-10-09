using System.Collections.Generic;
using ICUNetwork;
using ICUServiceInstaller.Utils;
using Serilog;

namespace ICUServiceInstaller;

public class PanelSockets : PanelBase
{
	private readonly ILogger Logger = Log.ForContext<PanelSockets>();

	protected Dictionary<string, string> m_dicFases = new Dictionary<string, string>
	{
		{ "0", "1F" },
		{ "2", "3F" }
	};

	protected Dictionary<string, string> m_dicCable = new Dictionary<string, string>
	{
		{ "0", "Fixed cable type 1" },
		{ "16", "Fixed cable type 2" },
		{ "32", "Fixed cable type 3" },
		{ "48", "Socket" }
	};

	public PanelSockets(MainWindow parent, string pageID = "", bool showBorder = true)
		: base(parent, pageID, fIndent: false, showBorder)
	{
		Title = "Sockets";
		Tooltip = "Socket settings";
		IconName = "power-cord.png";
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
		AddCustomText("Charge point configuration power", "32A");
		AddCustomSelect(8271, 0, "Charge point configuration fases", m_dicFases, 2);
		AddCustomSelect(8271, 0, "Charge point configuration cable", m_dicCable, 48);
		if (newDevice.NumberOfSockets == 2)
		{
			using DoubleColumnMode doubleColumnMode = new DoubleColumnMode(this);
			AddSmallHeader("Socket 1");
			AddSelect(8485, 0, 0, 0);
			AddSelect(8728, 0, 0, 0);
			doubleColumnMode.NextColumn();
			AddSmallHeader("Socket 2");
			AddSelect(12581, 0, 8485, 0);
			AddSelect(12824, 0, 8728, 0);
		}
		else
		{
			AddSmallHeader("Socket 1");
			AddSelect(8485, 0, 0, 0);
			AddSelect(8728, 0, 0, 0);
		}
		return true;
	}
}
