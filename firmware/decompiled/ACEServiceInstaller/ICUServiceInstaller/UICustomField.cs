using Xwt.Drawing;

namespace ICUServiceInstaller;

public struct UICustomField(string text, Color txColor, Color bgColor)
{
	public string text = text;

	public Color txColor = txColor;

	public Color bgColor = bgColor;
}
