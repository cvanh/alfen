using System;
using System.Threading;
using ICUNetwork;
using ICUSettings;
using Serilog;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class DlgSCNPhaseMapping : Dialog
{
	private readonly ILogger Logger = Log.ForContext<DlgSCNPhaseMapping>();

	protected ComboBox m_cmbFeederCables = new ComboBox();

	protected ComboBox m_cmbPhaseMapping1 = new ComboBox();

	protected ComboBox m_cmbPhaseMapping2 = new ComboBox();

	protected EDSParameter m_edsParameter;

	protected int phaseMapping1;

	protected int phaseMapping2;

	protected Label lblInfo;

	protected ICULanDevice m_lanDevice;

	protected SCNNetwork SCNNetwork { get; set; }

	public DlgSCNPhaseMapping(ICULanDevice lanDevice, SCNNetwork scnNetwork, LANConnection lanCon)
	{
		SCNNetwork = scnNetwork;
		m_lanDevice = lanDevice;
		m_cmbFeederCables = new ComboBox
		{
			BackgroundColor = AppProperties.Color_Disabled,
			Font = AppProperties.Font_Base,
			Items = 
			{
				(object)"1",
				(object)"2"
			}
		};
		m_cmbFeederCables.SelectionChanged += OnFeederCablesChanged;
		m_cmbPhaseMapping1 = new ComboBox
		{
			BackgroundColor = AppProperties.Color_Disabled,
			Font = AppProperties.Font_Base
		};
		m_cmbPhaseMapping1.SelectionChanged += OnPhaseMapping1Changed;
		m_cmbPhaseMapping2 = new ComboBox
		{
			BackgroundColor = AppProperties.Color_Disabled,
			Font = AppProperties.Font_Base
		};
		Title = $"Change Phasemapping of '{lanDevice.Identity}'";
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
		if (lanCon != null)
		{
			m_cmbPhaseMapping1.Items.Clear();
			m_cmbPhaseMapping2.Items.Clear();
			try
			{
				if (lanDevice.IsLoggedIn || lanDevice.Login().IsLoggedIn)
				{
					if (lanDevice.UpdateProperties(2195463u, 2195465u))
					{
						phaseMapping1 = lanDevice.GetPropertyInt(8576, 7);
						phaseMapping2 = lanDevice.GetPropertyInt(8576, 9);
					}
					lanDevice.Logout();
				}
			}
			catch (Exception ex)
			{
				Logger.Error(ex, ex.Message);
			}
			m_edsParameter = DataSheet.FindParameter(8576, 7);
			if (m_edsParameter.Options != null)
			{
				foreach (EDSParameterOption option in m_edsParameter.Options)
				{
					m_cmbPhaseMapping1.Items.Add(option.Value, option.Title);
				}
			}
			m_cmbPhaseMapping1.SelectedIndex = ((phaseMapping1 >= 0 && phaseMapping1 < m_cmbPhaseMapping1.Items.Count) ? phaseMapping1 : 0);
			m_edsParameter = DataSheet.FindParameter(8576, 9);
			if (m_edsParameter.Options != null)
			{
				foreach (EDSParameterOption option2 in m_edsParameter.Options)
				{
					m_cmbPhaseMapping2.Items.Add(option2.Value, option2.Title);
				}
			}
			m_cmbPhaseMapping2.SelectedIndex = ((phaseMapping2 >= 0 && phaseMapping2 < m_cmbPhaseMapping2.Items.Count) ? phaseMapping2 : 0);
			lblInfo = new Label("With 1 Feeder Cable both Phase Mappings are the same!");
			lblInfo.Font = lblInfo.Font.WithWeight(FontWeight.Semibold);
			lblInfo.Font = lblInfo.Font.WithStyle(FontStyle.Italic);
			lblInfo.TextColor = Colors.White;
			if (lanDevice.NumberOfFeederCables == 1)
			{
				m_cmbFeederCables.SelectedIndex = 0;
				m_cmbFeederCables.Sensitive = false;
				m_cmbPhaseMapping2.SelectedIndex = m_cmbPhaseMapping1.SelectedIndex;
				m_cmbPhaseMapping2.Visible = false;
				if (lanDevice.NumberOfSockets > 1)
				{
					lblInfo.TextColor = Colors.Black;
				}
			}
		}
		int num = 0;
		Label label = new Label($"Number of Sockets: {lanDevice.NumberOfSockets}");
		label.Font = label.Font.WithWeight(FontWeight.Semibold);
		table.Add(label, 0, num, 1, 2, hexpand: false, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, -1.0, -1.0, -1.0, -1.0, 5.0);
		num += 2;
		table.Add(new Label("Number of Feeder Cables:"), 0, num);
		table.Add(m_cmbFeederCables, 1, num, 1, 1, hexpand: true);
		num++;
		Label label2 = new Label("Phasemapping");
		label2.Font = label2.Font.WithWeight(FontWeight.Semibold);
		table.Add(label2, 0, num, 1, 2, hexpand: false, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, -1.0, -1.0, -1.0, -1.0, 5.0);
		num += 2;
		table.Add(new Label("PhaseMapping Socket 1:"), 0, num);
		table.Add(m_cmbPhaseMapping1, 1, num, 1, 1, hexpand: true);
		num++;
		if (lanDevice.NumberOfSockets > 1)
		{
			table.Add(new Label("PhaseMapping Socket 2:"), 0, num);
			table.Add(m_cmbPhaseMapping2, 1, num, 1, 1, hexpand: true);
			num++;
			table.Add(lblInfo, 0, num, 1, 2, hexpand: false, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, -1.0, -1.0, -1.0, -1.0, 3.0);
		}
		VBox vBox = new VBox();
		vBox.PackStart(widget, expand: true);
		Content = vBox;
		Buttons.Add(new DialogButton(Command.Cancel));
		Buttons.Add(new DialogButton(Command.Ok));
	}

	private void OnFeederCablesChanged(object sender, EventArgs e)
	{
		if (m_cmbFeederCables.SelectedIndex == 0)
		{
			m_cmbPhaseMapping2.SelectedIndex = m_cmbPhaseMapping1.SelectedIndex;
			m_cmbPhaseMapping2.Sensitive = false;
			lblInfo.TextColor = Colors.Black;
		}
		else
		{
			m_cmbPhaseMapping2.Sensitive = true;
			lblInfo.TextColor = Colors.Transparent;
		}
	}

	private void OnPhaseMapping1Changed(object sender, EventArgs e)
	{
		if (m_cmbFeederCables.SelectedIndex == 0 && m_lanDevice.NumberOfSockets == 2)
		{
			m_cmbPhaseMapping2.SelectedIndex = m_cmbPhaseMapping1.SelectedIndex;
		}
	}

	protected override void OnCommandActivated(Command cmd)
	{
		if (cmd == Command.Ok)
		{
			Sensitive = false;
			if (!m_lanDevice.Login().IsLoggedIn)
			{
				Respond(Command.Cancel);
				Close();
			}
			m_lanDevice.UpdateProperties(2195463u, 2195465u);
			ICUProperty property = m_lanDevice.GetProperty(8576, 7);
			property.Value = m_cmbPhaseMapping1.SelectedItem;
			ICUProperty property2 = m_lanDevice.GetProperty(8576, 9);
			property2.Value = m_cmbPhaseMapping2.SelectedItem;
			m_lanDevice.StoreProperties(property, property2);
			Thread.Sleep(1500);
			m_lanDevice.Logout();
			Sensitive = true;
			Respond(Command.Ok);
			Close();
		}
		else if (cmd == Command.Cancel)
		{
			Respond(Command.Cancel);
			Close();
		}
	}
}
