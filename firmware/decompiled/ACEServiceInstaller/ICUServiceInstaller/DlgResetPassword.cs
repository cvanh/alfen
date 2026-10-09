using System;
using ICUNetwork;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class DlgResetPassword : Dialog
{
	private readonly ICULanDevice m_lanDevice;

	private readonly TextEntry m_txtPRC;

	public string PRC { get; set; }

	public DlgResetPassword(ICULanDevice lanDevice)
	{
		m_lanDevice = lanDevice;
		Title = "Forgot password";
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
			Padding = 16.0,
			BorderWidth = 1.0,
			BackgroundColor = Colors.White,
			Content = table,
			MinWidth = 420.0
		};
		table.Margin = 8.0;
		Label label = new Label("As the owner of this charging station, you can reset the current owner password to the default password.\nYou do this by entering the Password Reset Code (PRC) of this charging station.\nThe PRC and the default password can be found on the flyer that was inside the charging station box when it was shipped.\nAbove is applicable for AHP chargers (all firmware versions) and NG9xx chargers (delivered with FW 5.0 onwards, June 2021).\n");
		label.Font = label.Font.WithWeight(FontWeight.Semibold);
		table.Add(label, 0, 0, 1, 2, hexpand: true, vexpand: false, WidgetPlacement.Center);
		table.Add(new Label(""), 0, 2, 1, 2);
		table.Add(new Label("Charging Station identity: " + m_lanDevice.Identity + " (Serial number: " + m_lanDevice.SerialNumber + ")\n")
		{
			TextColor = Colors.Gray
		}, 0, 1, 1, 2);
		m_txtPRC = new TextEntry();
		table.Add(new Label("Password Reset Code:"), 0, 2);
		table.Add(m_txtPRC, 1, 2, 1, 1, hexpand: true);
		table.Add(new Label(""), 0, 3, 1, 2);
		Content = content;
		m_txtPRC.SetFocus();
		CloseRequested += (object sender, CloseRequestedEventArgs args) =>
		{
			Respond(Command.Cancel);
		};
		DialogButton dialogButton = new DialogButton("Ok");
		dialogButton.Clicked += OnOkClicked;
		Buttons.Add(new DialogButton(Command.Cancel));
		Buttons.Add(dialogButton);
	}

	private void OnOkClicked(object sender, EventArgs e)
	{
		PRC = m_txtPRC.Text.Trim().Replace("-", "").Replace("_", "");
		Respond(Command.Ok);
	}
}
