using System;
using System.Diagnostics;
using System.Linq;
using System.Reflection;
using ICUSettings;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class DlgAbout : Dialog
{
	protected Label m_lblName;

	public DlgAbout(ICUConfig icuConfig)
	{
		Title = "About " + AppProperties.AppName;
		Icon = Image.FromResource(typeof(App), AppProperties.AppIconName);
		ShowInTaskbar = false;
		Resizable = false;
		Table table = new Table
		{
			BackgroundColor = Colors.White
		};
		FrameBox widget = new FrameBox
		{
			BorderColor = AppProperties.Color_Border,
			Padding = 8.0,
			BorderWidth = 1.0,
			BackgroundColor = Colors.White,
			Content = table
		};
		table.Margin = 8.0;
		int num = 0;
		Label label = new Label("ACE Service Installer");
		label.Font = label.Font.WithWeight(FontWeight.Semibold);
		table.Add(label, 0, num++, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Center);
		table.Add(new Label(""), 0, num++, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Center);
		m_lblName = new Label("Programmed by the Alfen Software Team");
		m_lblName.Font = m_lblName.Font.WithStyle(FontStyle.Italic);
		table.Add(m_lblName, 0, num, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Center);
		m_lblName.MouseEntered += (object sender, EventArgs e) =>
		{
			m_lblName.Opacity = 1.0;
		};
		m_lblName.MouseExited += (object sender, EventArgs e) =>
		{
			m_lblName.Opacity = 0.0;
		};
		m_lblName.Opacity = 0.0;
		num++;
		num = AddItalicLabel(table, num, "Alfen Charging Stations");
		num = AddItalicLabel(table, num, "www.alfen.com");
		table.Add(new Label(""), 0, num++, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Center);
		if (icuConfig != null)
		{
			table.Add(new Label($"Settings version: {icuConfig.Version}"), 0, num++, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Center);
			table.Add(new Label($"Settings date: {icuConfig.Date}"), 0, num++, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Center);
		}
		table.Add(new Label(""), 0, num++, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Center);
		string[] array = FileVersionInfo.GetVersionInfo(Assembly.GetExecutingAssembly().Location).FileVersion.Split(new char[1] { '.' });
		if (array.Count() >= 4)
		{
			table.Add(new Label(string.Format("Application version: {0}.{1}.{2}-{3}", new object[4]
			{
				array[0],
				array[1],
				array[2],
				array[3]
			})), 0, num++, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Center);
		}
		table.Add(new Label(""), 0, num++, 1, 1, hexpand: false, vexpand: true, WidgetPlacement.Center, WidgetPlacement.End);
		num = AddItalicLabel(table, num, "                                                                   ");
		num = AddSmallLabel(table, num, "  Icons made by Freepik, Gregor Cresnar & Dave Gandy from www.flaticon.com  ");
		num = AddSmallLabel(table, num, "  Palette Quantizer by SmartK8 (codeproject)  ");
		num = AddSmallLabel(table, num, "  mDNS by Tom Deseyn  ");
		HBox hBox = new HBox();
		ImageView widget2 = new ImageView(Image.FromResource(typeof(App), AppProperties.ResourcePath("Banner.png")));
		hBox.PackStart(widget2);
		hBox.PackEnd(widget, expand: true);
		Content = hBox;
		CloseRequested += (object sender, CloseRequestedEventArgs args) =>
		{
			Respond(Command.Ok);
		};
		Buttons.Add(new DialogButton(Command.Ok));
	}

	private int AddSmallLabel(Table tblMain, int lineIx, string text)
	{
		Label label = new Label(text);
		label.Font = label.Font.WithScaledSize(0.8).WithWeight(FontWeight.Thin);
		tblMain.Add(label, 0, lineIx, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Center, WidgetPlacement.End);
		return ++lineIx;
	}

	private int AddItalicLabel(Table tblMain, int lineIx, string text)
	{
		Label label = new Label(text);
		label.Font = label.Font.WithStyle(FontStyle.Italic);
		tblMain.Add(label, 0, lineIx, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Center);
		return ++lineIx;
	}
}
