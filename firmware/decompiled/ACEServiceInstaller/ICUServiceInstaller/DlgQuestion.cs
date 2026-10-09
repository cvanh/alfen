using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class DlgQuestion : Dialog
{
	public DlgQuestion(string text, bool showInTaskbar)
	{
		Title = AppProperties.AppName;
		Icon = Image.FromResource(typeof(App), AppProperties.AppIconName);
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
			Content = table
		};
		table.Margin = 8.0;
		table.Add(new ImageView(StockIcons.Question.WithSize(IconSize.Large)), 0, 0, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Center);
		table.Add(new Label(text), 1, 0, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Center);
		Content = content;
		CloseRequested += (object sender, CloseRequestedEventArgs args) =>
		{
			Respond(Command.Cancel);
		};
		Buttons.Add(new DialogButton(Command.Yes));
		Buttons.Add(new DialogButton(Command.No));
		Buttons.Add(new DialogButton(Command.Cancel));
	}
}
