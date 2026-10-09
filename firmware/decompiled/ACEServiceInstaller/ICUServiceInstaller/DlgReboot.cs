using System;
using ICUNetwork;
using Serilog;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class DlgReboot : Dialog
{
	private readonly ILogger Logger = Log.ForContext<DlgReboot>();

	protected ICULanDevice m_currentDevice;

	protected Label m_lblInfo;

	protected ProgressBar m_prbResetting = new ProgressBar();

	protected bool m_hardReboot = true;

	public DlgReboot(ICULanDevice currentDevice, bool fAutoStartReboot = false, bool hardReboot = true, DateTime uploadEnded = default(DateTime))
	{
		m_currentDevice = currentDevice;
		Title = "Rebooting";
		Icon = Image.FromResource(typeof(App), AppProperties.AppIconName);
		ShowInTaskbar = false;
		Resizable = true;
		Table table = new Table
		{
			BackgroundColor = Colors.White,
			MinHeight = 60.0,
			MinWidth = 400.0
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
		m_lblInfo = new Label($"Are you sure you want to reboot device '{m_currentDevice.Identification}'?");
		m_lblInfo.Font = m_lblInfo.Font.WithWeight(FontWeight.Semibold);
		table.Add(m_lblInfo, 0, 0, 1, 2, hexpand: true, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, UIPropertyBase.s_marginHor, marginRight: UIPropertyBase.s_marginHor, marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
		m_prbResetting.MinHeight = 16.0;
		table.Add(m_prbResetting, 0, 1, 1, 2, hexpand: true, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, UIPropertyBase.s_marginHor, marginRight: UIPropertyBase.s_marginHor, marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
		m_prbResetting.Fraction = 0.0;
		VBox vBox = new VBox();
		vBox.PackStart(widget, expand: true);
		Content = vBox;
		CloseRequested += (object sender, CloseRequestedEventArgs args) =>
		{
			Respond(Command.Cancel);
		};
		Buttons.Add(new DialogButton(Command.Cancel));
		Buttons.Add(new DialogButton(Command.No));
		m_hardReboot = hardReboot;
		if (fAutoStartReboot)
		{
			StartReboot(m_hardReboot);
		}
		else
		{
			Buttons.Add(new DialogButton(Command.Yes));
		}
	}

	protected override void OnCommandActivated(Command cmd)
	{
		if (cmd == Command.Yes)
		{
			StartReboot(m_hardReboot);
			return;
		}
		Respond(Command.Cancel);
		Close();
	}

	protected void StartReboot(bool hardReboot)
	{
		if (m_currentDevice.Reboot(RebootFinished, RebootProgress, hardReboot))
		{
			m_lblInfo.Text = "Rebooting, please wait...";
			DisableCommand(Command.Yes);
			DisableCommand(Command.No);
			DisableCommand(Command.Cancel);
		}
	}

	protected void RebootFinished(ICULanDevice device, Exception ex)
	{
		Application.Invoke(() =>
		{
			m_prbResetting.Visible = false;
			EnableCommand(Command.Yes);
			EnableCommand(Command.No);
			EnableCommand(Command.Cancel);
			if (ex != null)
			{
				m_lblInfo.Text = "Timeout while waiting for the reset to finish.";
				Buttons.Clear();
				Buttons.Add(new DialogButton(Command.Cancel));
			}
			else
			{
				Close();
			}
		});
	}

	protected void RebootProgress(ICULanDevice device, double progress)
	{
		Application.Invoke(() =>
		{
			m_prbResetting.Fraction = progress;
		});
	}
}
