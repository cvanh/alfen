using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class DlgInfo : Dialog
{
	public DlgInfo(string text, bool showInTaskbar)
	{
		Title = AppProperties.AppName;
		ShowInTaskbar = showInTaskbar;
		Resizable = false;
		Table table = new Table
		{
			BackgroundColor = Colors.White
		};
		FrameBox content = new FrameBox
		{
			BorderColor = AppProperties.Color_Border,
			Padding = 16.0,
			BorderWidth = 1.0,
			BackgroundColor = Colors.White,
			Content = table,
			MinWidth = 420.0
		};
		table.Margin = 8.0;
		table.Add(new Label(text), 1, 0, 1, 1, hexpand: true, vexpand: false, WidgetPlacement.Center);
		Content = content;
		CloseRequested += (object sender, CloseRequestedEventArgs args) =>
		{
			Respond(Command.Cancel);
		};
		Buttons.Add(new DialogButton(Command.Ok));
	}
}
