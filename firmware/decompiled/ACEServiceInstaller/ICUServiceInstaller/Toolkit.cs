using Xwt;

namespace ICUServiceInstaller;

public class Toolkit : Widget
{
	public ToolkitType GetToolKit()
	{
		return BackendHost.ToolkitEngine.Type;
	}
}
