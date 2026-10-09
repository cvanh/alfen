using System;
using System.Collections.Generic;
using System.ComponentModel;
using System.Configuration;
using System.Diagnostics;
using System.IO;
using System.Linq;
using System.Text.RegularExpressions;
using ICUNetwork;
using ICUServiceInstaller.Properties;
using ICUServiceInstaller.Utils;
using ICUSettings;
using Serilog;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class DlgUpload : Dialog
{
	private readonly ILogger Logger = Log.ForContext<DlgUpload>();

	protected TextEntry m_txtFilename = new TextEntry
	{
		PlaceholderText = "Enter a valid firmware file"
	};

	protected TextEntry m_txtInformation = new TextEntry();

	protected ListView m_lstFilenames = new ListView();

	protected Button m_btnBrowse = new Button("...");

	protected ProgressBar m_prbUpload = new ProgressBar();

	protected Button m_btnUpload = new Button("Start upload");

	protected Spinner m_spnSpinner = new Spinner();

	protected BackgroundWorker m_bgwUpload = new BackgroundWorker();

	protected Label m_lblDeviceHeader = new Label();

	protected Label m_lblVersion = new Label();

	protected Label m_lblProgression = new Label();

	protected Label m_lblUpgradeWarning = new Label();

	private static readonly int s_nDefaultMargin;

	protected ICULanDevice m_currentDevice;

	protected Version m_initialVersion;

	protected string m_newPassword = string.Empty;

	protected ListStore m_lsFirmwareFileStore;

	protected DataField<string> m_dfFirmwareName = new DataField<string>();

	protected DataField<string> m_dfFirmwareVersion = new DataField<string>();

	protected DataField<string> m_dfFirmwareDate = new DataField<string>();

	protected DataField<string> m_dfFirmwareInformation = new DataField<string>();

	protected DataField<FileInfo> m_dfFirmwareFileInfo = new DataField<FileInfo>();

	protected Stopwatch m_stopWatch;

	protected MainWindow m_parent;

	public void Initialize(MainWindow parent)
	{
		m_parent = parent;
		Title = "Upload new firmware";
		Icon = Image.FromResource(typeof(App), AppProperties.AppIconName);
		ShowInTaskbar = false;
		Resizable = true;
		Table table = new Table
		{
			MinHeight = 400.0,
			MinWidth = 850.0,
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
		Content = content;
		table.Margin = 8.0;
		int num = 0;
		m_lblDeviceHeader.Font = m_lblDeviceHeader.Font.WithScaledSize(1.2).WithWeight(FontWeight.Semibold);
		m_lblDeviceHeader.TextColor = Colors.SteelBlue;
		table.Add(m_lblDeviceHeader, 0, num++, 1, 2, hexpand: false, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, UIPropertyBase.s_marginHor + s_nDefaultMargin, marginRight: UIPropertyBase.s_marginHorMax, marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
		m_lblVersion.Font = m_lblVersion.Font.WithWeight(FontWeight.Semibold);
		m_lblProgression.Font = m_lblVersion.Font.WithWeight(FontWeight.Semibold);
		m_lblProgression.Text = "Uploading firmware to the charger...";
		m_lblUpgradeWarning = new Label("Please update to NG9xx 6.6.2 before updating to NG9xx 7.x.x to ensure a smooth update")
		{
			TextColor = Colors.Red,
			Visible = false
		};
		table.Add(m_lblUpgradeWarning, 0, num, 1, 2, hexpand: false, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, UIPropertyBase.s_marginHor + s_nDefaultMargin);
		num++;
		table.Add(new Label("Current firmware version:"), 0, num, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, UIPropertyBase.s_marginHor + s_nDefaultMargin, marginRight: UIPropertyBase.s_marginHorMax, marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
		table.Add(m_lblVersion, 1, num, 1, 1, hexpand: true, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, UIPropertyBase.s_marginHor, marginRight: UIPropertyBase.s_marginHor, marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
		num++;
		m_txtFilename.Changed += OnFilenameChanged;
		m_btnBrowse.Clicked += OnBtnBrowse;
		m_btnUpload.Clicked += OnBtnUploadClicked;
		m_btnBrowse.MinWidth = 20.0;
		m_prbUpload.Visible = false;
		m_spnSpinner.Visible = false;
		m_lblProgression.Visible = false;
		m_spnSpinner.MinHeight = 48.0;
		m_prbUpload.MinHeight = 16.0;
		m_btnUpload.MinWidth = AppProperties.ButtonWidth;
		m_btnUpload.MinHeight = AppProperties.ButtonHeight;
		m_btnUpload.Style = ButtonStyle.Normal;
		m_lsFirmwareFileStore = new ListStore(m_dfFirmwareName, m_dfFirmwareVersion, m_dfFirmwareDate, m_dfFirmwareInformation, m_dfFirmwareFileInfo);
		m_lstFilenames.DataSource = m_lsFirmwareFileStore;
		m_lstFilenames.Columns.Add(new ListViewColumn("", new TextCellView(m_dfFirmwareName)));
		m_lstFilenames.Columns.Add(new ListViewColumn("", new TextCellView(m_dfFirmwareVersion)));
		m_lstFilenames.Columns.Add(new ListViewColumn("", new TextCellView(m_dfFirmwareDate)));
		m_lstFilenames.Columns.Add(new ListViewColumn("", new TextCellView(m_dfFirmwareInformation)));
		m_lstFilenames.GridLinesVisible = GridLines.Horizontal;
		m_lstFilenames.HeadersVisible = false;
		m_lstFilenames.HeightRequest = 125.0;
		m_lstFilenames.WidthRequest = 100.0;
		m_lstFilenames.MinWidth = 300.0;
		m_lstFilenames.SelectionChanged += OnFirmwareSelectionChanged;
		table.Add(new Label("Select firmware:"), 0, num, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Start, UIPropertyBase.s_marginHor + s_nDefaultMargin, marginRight: UIPropertyBase.s_marginHorMax, marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
		table.Add(m_lstFilenames, 1, num, 1, 1, hexpand: true, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, UIPropertyBase.s_marginHor, UIPropertyBase.s_marginVer, 0.0, UIPropertyBase.s_marginVer);
		num++;
		m_txtInformation.MinHeight = 64.0;
		m_txtInformation.ReadOnly = true;
		m_txtInformation.MultiLine = true;
		m_txtInformation.TextAlignment = Alignment.Start;
		table.Add(m_txtInformation, 1, num, 1, 1, hexpand: true, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, UIPropertyBase.s_marginHor, UIPropertyBase.s_marginVer, 0.0, UIPropertyBase.s_marginVer);
		num++;
		table.Add(new Label("Firmware file location:"), 0, num, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, UIPropertyBase.s_marginHor + s_nDefaultMargin, marginRight: UIPropertyBase.s_marginHorMax, marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
		HBox hBox = new HBox();
		hBox.PackStart(m_txtFilename, expand: true);
		hBox.PackStart(m_btnBrowse, expand: false);
		table.Add(hBox, 1, ++num, 1, 1, hexpand: true, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, UIPropertyBase.s_marginHor, UIPropertyBase.s_marginVer, 0.0, UIPropertyBase.s_marginVer);
		table.Add(m_prbUpload, 0, ++num, 1, 2, hexpand: false, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, UIPropertyBase.s_marginHor + s_nDefaultMargin);
		table.Add(m_lblProgression, 0, ++num, 1, 2, hexpand: false, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, UIPropertyBase.s_marginHor, UIPropertyBase.s_marginVer, 0.0, UIPropertyBase.s_marginVer);
		table.Add(m_spnSpinner, 0, ++num, 1, 2, hexpand: false, vexpand: false, WidgetPlacement.Center);
		table.Add(m_btnUpload, 1, num, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Start, WidgetPlacement.Fill, UIPropertyBase.s_marginHor, marginRight: UIPropertyBase.s_marginHor, marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
		m_bgwUpload.WorkerReportsProgress = true;
		m_bgwUpload.DoWork += OnUploadDoWork;
		m_bgwUpload.ProgressChanged += OnUploadProgressChanged;
		m_bgwUpload.RunWorkerCompleted += OnUploadCompleted;
		m_prbUpload.Fraction = 0.0;
		CloseRequested += (object sender, CloseRequestedEventArgs args) =>
		{
			args.AllowClose = false;
			Hide();
		};
		Buttons.Add(new DialogButton(Command.Close));
	}

	public void ChangeDevice(ICULanDevice lanDev)
	{
		m_currentDevice = lanDev;
		if (m_initialVersion == null)
		{
			m_initialVersion = m_currentDevice.FirmwareVersionNumber;
		}
		if (lanDev != null)
		{
			m_lblUpgradeWarning.Visible = ShowNg9xxUpgradeWarning();
			m_lblDeviceHeader.Text = $"Upload firmware to device '{lanDev.Identification}' (serial number: {lanDev.GetPropertyString(8273, 0, 0)})";
			m_lblVersion.Text = lanDev.GetPropertyString(4106, 0, 0);
		}
		m_prbUpload.Fraction = 0.0;
	}

	protected override void OnCommandActivated(Command cmd)
	{
		Hide();
	}

	private void FillList()
	{
		try
		{
			m_lsFirmwareFileStore.Clear();
			string localFirmwareFolder = AppProperties.LocalFirmwareFolder;
			if (!Directory.Exists(localFirmwareFolder))
			{
				Directory.CreateDirectory(localFirmwareFolder);
			}
			List<FileInfo> source = (from a in Directory.GetFiles(localFirmwareFolder)
				select new FileInfo(a)).ToList();
			if (m_currentDevice == null)
			{
				source = source.OrderByDescending((FileInfo a) => a.LastWriteTime).ToList();
			}
			else
			{
				string[] updateFileTypes = m_currentDevice.getUpdateFileTypes;
				source = (from a in source
					where Enumerable.Contains(updateFileTypes, a.Extension.ToLowerInvariant())
					orderby a.LastWriteTime descending
					select a).ToList();
			}
			foreach (FileInfo item in source)
			{
				int row = m_lsFirmwareFileStore.AddRow();
				string value = "";
				string value2 = "";
				ICUFirmware iCUFirmware = AppProperties.ICUConfig.FindFirmware(item.Name);
				if (iCUFirmware != null)
				{
					value2 = iCUFirmware.Version;
					value = iCUFirmware.Comments.Replace("\n", " ");
				}
				m_lsFirmwareFileStore.SetValues(row, m_dfFirmwareName, item.Name, m_dfFirmwareVersion, value2, m_dfFirmwareDate, item.LastWriteTime.ToString(), m_dfFirmwareInformation, value, m_dfFirmwareFileInfo, item);
			}
			m_lstFilenames.Sensitive = m_lsFirmwareFileStore.RowCount > 0;
			if (m_lsFirmwareFileStore.RowCount > 0)
			{
				m_lstFilenames.SelectRow(0);
			}
		}
		catch (Exception ex)
		{
			Logger.Error(ex, ex.Message);
		}
	}

	protected override void OnShown()
	{
		FillList();
	}

	private void OnFirmwareSelectionChanged(object sender, EventArgs e)
	{
		if (sender is ListView { SelectedRow: >=0 } listView && listView.SelectedRow < m_lsFirmwareFileStore.RowCount)
		{
			FileInfo value = m_lsFirmwareFileStore.GetValue(listView.SelectedRow, m_dfFirmwareFileInfo);
			m_txtFilename.Text = value.FullName;
			ICUFirmware iCUFirmware = AppProperties.ICUConfig.FindFirmware(value.Name);
			if (iCUFirmware != null)
			{
				m_txtInformation.Text = iCUFirmware.Comments;
			}
			else
			{
				m_txtInformation.Text = "";
			}
		}
	}

	private void OnFilenameChanged(object sender, EventArgs e)
	{
		m_btnUpload.Sensitive = File.Exists(m_txtFilename.Text);
	}

	private void OnBtnUploadClicked(object sender, EventArgs e)
	{
		Logger.Debug("Button upload firmware clicked");
		m_newPassword = string.Empty;
		m_stopWatch = null;
		if (m_currentDevice == null || m_bgwUpload == null || MessageDialog.AskQuestion("Are you sure you want to update the firmware of '" + m_currentDevice.Identification + "' to '" + Path.GetFileName(m_txtFilename.Text) + "'", Command.Yes, Command.No, Command.Cancel) != Command.Yes)
		{
			return;
		}
		bool flag = true;
		Match match = Regex.Match(Path.GetFileName(m_txtFilename.Text.ToLowerInvariant()), "[_ ](\\d+.\\d+.\\d+)[_.-]");
		if (match.Groups.Count > 1)
		{
			Version version = Version.Parse(match.Groups[1].ToString());
			if (!m_currentDevice.isAHP && m_currentDevice.FirmwareVersionNumber.Major < 5 && version.Major >= 5)
			{
				DlgDeviceNewPassword dlgDeviceNewPassword = new DlgDeviceNewPassword(m_currentDevice);
				if (dlgDeviceNewPassword.Run(this) == Command.Ok)
				{
					m_newPassword = dlgDeviceNewPassword.NewPassword;
				}
				else
				{
					flag = false;
				}
			}
		}
		if (flag)
		{
			m_btnUpload.Sensitive = false;
			m_btnBrowse.Sensitive = false;
			m_lstFilenames.Sensitive = false;
			m_txtFilename.Sensitive = false;
			m_btnUpload.Visible = false;
			m_prbUpload.Visible = true;
			m_lblProgression.Visible = true;
			m_spnSpinner.Visible = true;
			m_spnSpinner.Animate = true;
			UploadWorkerData argument = new UploadWorkerData
			{
				CurrentDevice = m_currentDevice,
				Filename = m_txtFilename.Text,
				Worker = (match.Success ? m_bgwUpload : null)
			};
			m_bgwUpload.RunWorkerAsync(argument);
		}
	}

	private void OnUploadDoWork(object sender, DoWorkEventArgs e)
	{
		if (e.Argument is UploadWorkerData uploadWorkerData)
		{
			uploadWorkerData.CurrentDevice.UploadFirmware(uploadWorkerData.Worker, uploadWorkerData.Filename, m_newPassword);
		}
	}

	private bool ShowNg9xxUpgradeWarning()
	{
		if (m_currentDevice != null && !m_currentDevice.isAHP)
		{
			return m_currentDevice.FirmwareVersionNumber < new Version(6, 6);
		}
		return false;
	}

	private void OnUploadCompleted(object sender, RunWorkerCompletedEventArgs e)
	{
		m_btnUpload.Sensitive = true;
		m_btnBrowse.Sensitive = true;
		m_btnUpload.Visible = true;
		m_lstFilenames.Sensitive = true;
		m_txtFilename.Sensitive = true;
		m_prbUpload.Visible = false;
		m_spnSpinner.Visible = false;
		m_spnSpinner.Animate = false;
		m_lblProgression.Visible = false;
		ICULanDevice lanDev = m_currentDevice;
		if (lanDev == null)
		{
			return;
		}
		m_lblUpgradeWarning.Visible = ShowNg9xxUpgradeWarning();
		m_lblVersion.Text = lanDev.GetPropertyString(4106, 0, 0);
		if (!string.IsNullOrEmpty(lanDev.LastUploadError))
		{
			Logger.Error("Error during firmware upload: {Error}", lanDev.LastUploadError);
			MessageDialog.ShowError(this, "Error during firmware upload!", lanDev.LastUploadError);
			return;
		}
		try
		{
			Application.Invoke(() =>
			{
				lanDev.UpdateCategories("generic");
				if (!string.IsNullOrEmpty(m_newPassword))
				{
					Settings.Default.LastDevicePassword = m_newPassword;
					((SettingsBase)Settings.Default).Save();
					if (m_parent != null)
					{
						m_parent.AllowAutoLogonOnce();
					}
					lanDev.Logout();
					lanDev.Login();
				}
				if (m_parent != null)
				{
					m_parent.RefreshDeviceList();
				}
			});
			string fileName = Path.GetFileName(m_txtFilename.Text);
			Logger.AddChargerContext(m_currentDevice).Information("Firmware updated successfully: {OldVersion} - {NewVersion} - {FileName}", m_initialVersion, m_currentDevice.FirmwareVersionNumber, fileName);
			m_initialVersion = m_currentDevice.FirmwareVersionNumber;
			MessageDialog.ShowMessage(this, "Firmware updated successfully!");
		}
		catch (Exception ex)
		{
			Logger.Error(ex, "Firmware update failed: {LastUploadError}", lanDev.LastUploadError);
			MessageDialog.ShowError(this, "Firmware update failed!", ex.Message);
		}
	}

	private void OnUploadProgressChanged(object sender, ProgressChangedEventArgs e)
	{
		if (e.ProgressPercentage > 100)
		{
			m_prbUpload.Fraction = 1.0;
			return;
		}
		m_prbUpload.Fraction = (double)e.ProgressPercentage / 100.0;
		UpdateProgressionText(e.ProgressPercentage, m_currentDevice.isAHP);
	}

	private void UpdateProgressionText(int progression, bool isAhp = false)
	{
		if (isAhp)
		{
			UpdateProgressionTextAhp(progression);
		}
		else
		{
			UpdateProgressionTextNG9(progression);
		}
	}

	private void UpdateProgressionTextNG9(int progression)
	{
		if (progression < 50)
		{
			m_lblProgression.Text = "Uploading firmware to the charger, can take 1 to 3 minutes.";
		}
		else if (progression <= 97)
		{
			m_lblProgression.Text = "Installing firmware (charger will reboot).";
		}
		else if (progression <= 99)
		{
			m_lblProgression.Text = "Finishing up firmware update...";
		}
		else if (progression <= 100)
		{
			m_lblProgression.Text = "Almost done";
		}
	}

	private void UpdateProgressionTextAhp(int progression)
	{
		if (progression < 4)
		{
			m_lblProgression.Text = "Uploading firmware to the charger...";
		}
		else if (progression < 8)
		{
			m_lblProgression.Text = "Firmware uploaded, preparing update...";
		}
		else if (progression <= 97)
		{
			if (m_stopWatch == null)
			{
				m_stopWatch = new Stopwatch();
				m_stopWatch.Start();
			}
			m_lblProgression.Text = $"Installing firmware (can take up to 12 minutes) {(int)(m_stopWatch.ElapsedMilliseconds / 60000)} minutes passed.";
		}
		else if (progression <= 100)
		{
			m_lblProgression.Text = "Rebooting station and finishing up firmware update.";
		}
	}

	private void OnBtnBrowse(object sender, EventArgs e)
	{
		OpenFileDialog openFileDialog = new OpenFileDialog("Select a firmware file")
		{
			InitialFileName = "",
			Multiselect = false
		};
		string[] patterns = new string[5] { "*.fwi", "*.fwu", "*.tfw", "*.tcf", "*.tvf" };
		FileDialogFilter item = new FileDialogFilter("All firmware files", patterns);
		openFileDialog.Filters.Add(item);
		string[] patterns2 = new string[2] { "*.fwi", "*.fwu" };
		FileDialogFilter item2 = new FileDialogFilter("NG9xx firmware files", patterns2);
		openFileDialog.Filters.Add(item2);
		string[] patterns3 = new string[3] { "*.tfw", "*.tcf", "*.tvf" };
		FileDialogFilter item3 = new FileDialogFilter("AHWP firmware files", patterns3);
		openFileDialog.Filters.Add(item3);
		FileDialogFilter item4 = new FileDialogFilter("All files", "*.*");
		openFileDialog.Filters.Add(item4);
		if (m_currentDevice == null)
		{
			openFileDialog.ActiveFilter = openFileDialog.Filters[1];
		}
		else if (m_currentDevice.isAHP)
		{
			openFileDialog.ActiveFilter = openFileDialog.Filters[3];
		}
		else
		{
			openFileDialog.ActiveFilter = openFileDialog.Filters[2];
		}
		if (openFileDialog.Run())
		{
			m_txtFilename.Text = openFileDialog.FileName;
		}
	}
}
