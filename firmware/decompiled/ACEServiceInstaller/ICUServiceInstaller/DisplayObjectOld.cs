using ICUNetwork;

namespace ICUServiceInstaller;

public class DisplayObjectOld
{
	public EMainStates MainState { get; set; }

	public EStatusIcon Icon { get; set; }

	public string Text { get; set; }

	public DisplayObjectOld(EMainStates mainstate, string text, EStatusIcon icon)
	{
		MainState = mainstate;
		Icon = icon;
		Text = text;
	}
}
