using System.Configuration;
using ICUServiceInstaller.Properties;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class DlgSettings : Dialog
{
	protected CheckBox m_chkAskConfirm = new CheckBox("Ask confirmation when closing the application.");

	protected CheckBox m_chkAskSaveChanges = new CheckBox("Ask save changes when switching to another device.");

	public DlgSettings()
	{
		Title = "Settings";
		Icon = Image.FromResource(typeof(App), AppProperties.AppIconName);
		ShowInTaskbar = false;
		Resizable = false;
		Table table = new Table
		{
			BackgroundColor = Colors.White
		};
		FrameBox content = new FrameBox
		{
			BorderColor = AppProperties.Color_Border,
			Padding = 8.0,
			BorderWidth = 1.0,
			BackgroundColor = Colors.White,
			Content = table
		};
		table.Margin = 8.0;
		Content = content;
		table.Add(m_chkAskConfirm, 0, 0);
		table.Add(m_chkAskSaveChanges, 0, 1);
		m_chkAskConfirm.State = (Settings.Default.AskConfirmExit ? CheckBoxState.On : CheckBoxState.Off);
		m_chkAskSaveChanges.State = (Settings.Default.AskSaveChanges ? CheckBoxState.On : CheckBoxState.Off);
		CloseRequested += (object sender, CloseRequestedEventArgs args) =>
		{
			Respond(Command.Ok);
		};
		Buttons.Add(new DialogButton(Command.Ok));
	}

	protected override void OnCommandActivated(Command cmd)
	{
		if (cmd == Command.Ok)
		{
			Settings.Default.AskConfirmExit = m_chkAskConfirm.State == CheckBoxState.On;
			Settings.Default.AskSaveChanges = m_chkAskSaveChanges.State == CheckBoxState.On;
			((SettingsBase)Settings.Default).Save();
		}
		base.OnCommandActivated(cmd);
	}
}
