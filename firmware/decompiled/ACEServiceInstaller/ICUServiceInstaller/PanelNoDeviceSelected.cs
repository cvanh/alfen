using Xwt;

namespace ICUServiceInstaller;

public class PanelNoDeviceSelected : PanelBase
{
	protected Button m_btnLogin;

	public PanelNoDeviceSelected(MainWindow parent, string pageID = "", bool showBorder = true)
		: base(parent, pageID, fIndent: false, showBorder)
	{
		IconName = "information.png";
		AddLabel("No charging station selected.\nPlease select a charging station in the list.");
	}
}
