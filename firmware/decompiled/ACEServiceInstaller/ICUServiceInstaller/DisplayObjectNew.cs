using ICUNetwork;

namespace ICUServiceInstaller;

public class DisplayObjectNew
{
	public EUserInterfaceError ErrorCode { get; set; }

	public EStatusIcon Icon { get; set; }

	public string Text { get; set; }

	public DisplayObjectNew(EUserInterfaceError errorcode, string text, EStatusIcon icon)
	{
		ErrorCode = errorcode;
		Icon = icon;
		Text = text;
	}
}
