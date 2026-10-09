using System;
using System.Diagnostics;
using System.IO;
using System.Linq;
using System.Xml.Linq;
using ICUNetwork;
using Serilog;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class DlgPresets : Dialog
{
	private readonly ILogger Logger = Log.ForContext<DlgPresets>();

	private readonly ListView m_lstPresets = new ListView();

	protected ListStore m_lsPresetFileStore;

	protected DataField<string> m_dfPresetName = new DataField<string>();

	protected DataField<string> m_dfPresetVersion = new DataField<string>();

	protected DataField<string> m_dfPresetModel = new DataField<string>();

	protected DataField<string> m_dfPresetSockets = new DataField<string>();

	protected DataField<string> m_dfPresetPower = new DataField<string>();

	protected DataField<string> m_dfPresetInformation = new DataField<string>();

	protected DataField<FileInfo> m_dfFirmwareFileInfo = new DataField<FileInfo>();

	protected ICULanDevice m_currentDevice;

	protected Stopwatch m_swLastClickedTime = Stopwatch.StartNew();

	protected static int s_nDoubleClickTimeMilli = 300;

	public string FileName { get; set; }

	public DlgPresets(ICULanDevice currentDevice)
	{
		m_currentDevice = currentDevice;
		Title = "Property Presets " + AppProperties.AppName;
		Icon = Image.FromResource(typeof(App), AppProperties.AppIconName);
		ShowInTaskbar = false;
		Resizable = true;
		Table table = new Table
		{
			BackgroundColor = Colors.White,
			MinHeight = 200.0,
			MinWidth = 600.0
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
		Label label = new Label("Please select a property preset from the list below.");
		label.Font = label.Font.WithWeight(FontWeight.Semibold);
		table.Add(label, 0, 0, 1, 2);
		m_lsPresetFileStore = new ListStore(m_dfPresetName, m_dfPresetVersion, m_dfPresetModel, m_dfPresetSockets, m_dfPresetPower, m_dfPresetInformation, m_dfFirmwareFileInfo);
		m_lstPresets.DataSource = m_lsPresetFileStore;
		m_lstPresets.Columns.Add(new ListViewColumn("Name", new TextCellView(m_dfPresetName)));
		m_lstPresets.Columns.Add(new ListViewColumn("Version", new TextCellView(m_dfPresetVersion)));
		m_lstPresets.Columns.Add(new ListViewColumn("Model", new TextCellView(m_dfPresetModel)));
		m_lstPresets.Columns.Add(new ListViewColumn("Sockets", new TextCellView(m_dfPresetSockets)));
		m_lstPresets.Columns.Add(new ListViewColumn("Power", new TextCellView(m_dfPresetPower)));
		m_lstPresets.Columns.Add(new ListViewColumn("Information", new TextCellView(m_dfPresetInformation)));
		m_lstPresets.GridLinesVisible = GridLines.Horizontal;
		m_lstPresets.HeadersVisible = true;
		m_lstPresets.HeightRequest = (m_lstPresets.WidthRequest = 100.0);
		m_lstPresets.ButtonReleased += OnPresetClicked;
		table.Add(m_lstPresets, 0, 1, 1, 2, hexpand: true, vexpand: true);
		VBox vBox = new VBox();
		vBox.PackStart(widget, expand: true);
		Content = vBox;
		Buttons.Add(new DialogButton(Command.Cancel));
		Buttons.Add(new DialogButton(Command.Ok));
		FillPresetList();
	}

	private void OnPresetClicked(object sender, ButtonEventArgs e)
	{
		if (m_swLastClickedTime.ElapsedMilliseconds > s_nDoubleClickTimeMilli)
		{
			m_swLastClickedTime.Restart();
		}
		else
		{
			OnCommandActivated(Command.Ok);
		}
	}

	protected override void OnCommandActivated(Command cmd)
	{
		if (cmd == Command.Ok)
		{
			if (m_currentDevice != null)
			{
				FileName = m_lsPresetFileStore.GetValue(m_lstPresets.SelectedRow, m_dfFirmwareFileInfo).FullName;
				Respond(Command.Ok);
			}
		}
		else
		{
			Respond(Command.Cancel);
		}
		Close();
	}

	protected void LoadPresetsFromFolder(string path)
	{
		try
		{
			if (!Directory.Exists(path))
			{
				Directory.CreateDirectory(path);
			}
			foreach (FileInfo item in (from a in (from a in Directory.GetFiles(path)
					select new FileInfo(a)).ToList()
				where a.Extension.ToLowerInvariant() == ".xml" || a.Extension.ToLowerInvariant() == ".iip"
				orderby a.LastWriteTime descending
				select a).ToList())
			{
				int row = m_lsPresetFileStore.AddRow();
				string value = "";
				string value2 = "";
				string value3 = "";
				string value4 = "";
				string value5 = "";
				XDocument xDocument = XDocument.Load(item.FullName);
				if (xDocument != null)
				{
					XElement xElement = xDocument.Element("Settings");
					if (xElement != null)
					{
						value2 = GetElementString(xElement, "Version");
						value = GetElementString(xElement, "Model");
						value3 = GetElementString(xElement, "NumberOfSockets");
						value5 = GetElementString(xElement, "Information");
						foreach (XElement item2 in xElement.Element("Properties").Elements("Property"))
						{
							if (item2.Attribute("Id").Value == "2129_0")
							{
								value4 = item2.Attribute("Value").Value;
							}
						}
					}
				}
				m_lsPresetFileStore.SetValues(row, m_dfPresetName, Path.GetFileNameWithoutExtension(item.Name), m_dfPresetVersion, value2, m_dfPresetModel, value, m_dfPresetSockets, value3, m_dfPresetPower, value4, m_dfPresetInformation, value5, m_dfFirmwareFileInfo, item);
			}
		}
		catch (Exception exception)
		{
			Logger.Debug(exception, "");
		}
	}

	protected void FillPresetList()
	{
		try
		{
			m_lsPresetFileStore.Clear();
			LoadPresetsFromFolder(AppProperties.ProgramFilesPresetsFolder);
			LoadPresetsFromFolder(AppProperties.LocalTCPPresetsFolder);
			LoadPresetsFromFolder(AppProperties.LocalRTUPresetsFolder);
			LoadPresetsFromFolder(AppProperties.LocalBackofficePresetsFolder);
			m_lstPresets.Sensitive = m_lsPresetFileStore.RowCount > 0;
			if (m_lsPresetFileStore.RowCount > 0)
			{
				m_lstPresets.SelectRow(0);
			}
		}
		catch (Exception exception)
		{
			Logger.Debug(exception, "");
		}
	}

	protected string GetElementString(XElement xelem, string name)
	{
		XElement xElement = xelem.Element(name);
		if (xElement != null)
		{
			return xElement.Value;
		}
		return string.Empty;
	}
}
