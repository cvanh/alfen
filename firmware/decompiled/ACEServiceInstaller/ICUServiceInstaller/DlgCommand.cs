using ICUNetwork;
using ICUServiceInstaller.Properties;
using ICUServiceInstaller.Utils;
using Serilog;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class DlgCommand : Dialog
{
	protected TextEntry m_txtCommand = new TextEntry();

	protected ICULanDevice m_currentDevice;

	private readonly ILogger Logger = Log.ForContext<DlgCommand>();

	public DlgCommand(ICULanDevice device)
	{
		m_currentDevice = device;
		Title = "Command Window " + AppProperties.AppName;
		Icon = Image.FromResource(typeof(App), AppProperties.AppIconName);
		ShowInTaskbar = false;
		Resizable = true;
		Table table = new Table
		{
			BackgroundColor = Colors.White,
			MinWidth = 300.0
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
		Label label = new Label("Please enter the command (press enter to execute).");
		label.Font = label.Font.WithWeight(FontWeight.Semibold);
		table.Add(label, 0, 0, 1, 2);
		table.Add(new Label("Command:"), 0, 1);
		table.Add(m_txtCommand, 1, 1, 1, 1, hexpand: true);
		m_txtCommand.KeyPressed += OnTxtKeyPressed;
		VBox vBox = new VBox();
		vBox.PackStart(widget, expand: true);
		Content = vBox;
		if (!string.IsNullOrEmpty(Settings.Default.LastCommand))
		{
			m_txtCommand.Text = Settings.Default.LastCommand;
		}
		m_txtCommand.SetFocus();
		Buttons.Add(new DialogButton(Command.Close));
	}

	private void OnTxtKeyPressed(object sender, KeyEventArgs e)
	{
		if (e.Key == Key.Return && m_currentDevice != null)
		{
			m_currentDevice.SendCommand(m_txtCommand.Text);
			Logger.AddChargerContext(m_currentDevice).Information("Command executed: {Command}", m_txtCommand.Text);
		}
	}

	protected override void OnCommandActivated(Command cmd)
	{
		Respond(cmd);
		Close();
	}
}
