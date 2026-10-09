namespace ICUServiceInstaller;

public class PanelNoDevice : PanelBase
{
	public PanelNoDevice(MainWindow parent, string pageID = "", bool showBorder = true)
		: base(parent, pageID, fIndent: false, showBorder)
	{
		IconName = "information.png";
		AddLabel("No devices found on the current network, please make sure the Charging Station and this PC are connected to each other over ethernet.");
	}
}
