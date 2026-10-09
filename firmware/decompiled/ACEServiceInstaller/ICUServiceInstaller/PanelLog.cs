using System;
using System.Collections.Generic;
using System.Collections.Specialized;
using System.ComponentModel;
using System.IO;
using System.Linq;
using System.Net;
using ICUNetwork;
using ICUServiceInstaller.Utils;
using ICUSettings;
using Serilog;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class PanelLog : PanelBase
{
	private class DeviceWorkerData
	{
		public ICULanDevice CurrentDevice { get; set; }

		public BackgroundWorker Worker { get; set; }

		public int NumberOfDays { get; set; }

		public string Filename { get; set; }

		public bool JumpToAddedLines { get; set; }
	}

	private readonly ILogger Logger = Log.ForContext<PanelLog>();

	private static readonly string[] s_IconNames = new string[9] { "empty16.png", "Info.png", "Warning.png", "Error.png", "comm.png", "user.png", "reboot.png", "console.png", "security.png" };

	protected static List<Image> s_lstImages = new List<Image>();

	protected static List<ToggleButton> s_lstFilterButtons = new List<ToggleButton>();

	protected static int s_tickInterval = 2000;

	protected static int s_maxLogLineBatchSize = 512;

	private int linesAdded;

	protected int m_nDesiredNumberOfLines = s_maxLogLineBatchSize;

	protected ListView m_lvwLog;

	protected ListStore m_lsStore;

	protected DataField<string> m_dfDate = new DataField<string>();

	protected DataField<Image> m_dfType = new DataField<Image>();

	protected DataField<string> m_dfLine = new DataField<string>();

	protected DataField<string> m_dfFile = new DataField<string>();

	protected DataField<ICULanLogLine> m_dfSocket1 = new DataField<ICULanLogLine>();

	protected DataField<ICULanLogLine> m_dfSocket2 = new DataField<ICULanLogLine>();

	protected DataField<ICULanLogLine> m_dfInformation = new DataField<ICULanLogLine>();

	protected DataField<ICULanLogLine> m_dfState1 = new DataField<ICULanLogLine>();

	protected DataField<ICULanLogLine> m_dfState2 = new DataField<ICULanLogLine>();

	protected ToggleButton m_btnFollow;

	protected ToggleButton m_btnFileAndLine;

	protected ToggleButton m_btnSocket;

	protected Button m_btnCommand;

	protected Button m_btnSaveToFile;

	protected Button m_btnLoadNext;

	protected Button m_btnRefresh;

	protected Spinner m_spnSaveSpinner = new Spinner();

	protected Label m_lblSaveInfo = new Label();

	protected Label m_lblLineInfo = new Label();

	protected BackgroundWorker m_bgwUpdate = new BackgroundWorker();

	protected BackgroundWorker m_bgwSave = new BackgroundWorker();

	protected string m_strDownloadMessage;

	public PanelLog(MainWindow parent, string pageID = "", bool showBorder = true)
		: base(parent, pageID, fIndent: false, showBorder)
	{
		Title = "Log";
		Tooltip = "Logging";
		IconName = "interface.png";
		AddHeader(Tooltip);
		InitializeIcons();
		m_btnFollow = AddToolImageToggleButton("arrow-to-last-track.png", "Tail the log (auto-jump to latest entry)", delegate
		{
			if (m_btnFollow.Active)
			{
				SelectLastLogItem();
			}
		});
		m_btnFollow.Active = true;
		m_btnFileAndLine = AddToolImageToggleButton("linenumber.png", "Show/hide file and line numbers", delegate
		{
			AddListColumns();
		});
		m_btnSocket = AddToolImageToggleButton("socket.png", "Show/hide socket columns", delegate
		{
			AddListColumns();
		});
		s_lstFilterButtons = new List<ToggleButton>
		{
			AddToolImageToggleButton(s_IconNames[1], "Show general information", delegate
			{
				ToggleFilterButton(Enum.GetName(typeof(ICULanLogType), ICULanLogType.INFO));
			}, Enum.GetName(typeof(ICULanLogType), ICULanLogType.INFO)),
			AddToolImageToggleButton(s_IconNames[2], "Show warnings", delegate
			{
				ToggleFilterButton(Enum.GetName(typeof(ICULanLogType), ICULanLogType.WARNING));
			}, Enum.GetName(typeof(ICULanLogType), ICULanLogType.WARNING)),
			AddToolImageToggleButton(s_IconNames[3], "Show errors", delegate
			{
				ToggleFilterButton(Enum.GetName(typeof(ICULanLogType), ICULanLogType.ERROR));
			}, Enum.GetName(typeof(ICULanLogType), ICULanLogType.ERROR)),
			AddToolImageToggleButton(s_IconNames[4], "Show backoffice communication", delegate
			{
				ToggleFilterButton(Enum.GetName(typeof(ICULanLogType), ICULanLogType.COM));
			}, Enum.GetName(typeof(ICULanLogType), ICULanLogType.COM)),
			AddToolImageToggleButton(s_IconNames[5], "Show user information", delegate
			{
				ToggleFilterButton(Enum.GetName(typeof(ICULanLogType), ICULanLogType.USER));
			}, Enum.GetName(typeof(ICULanLogType), ICULanLogType.USER)),
			AddToolImageToggleButton(s_IconNames[6], "Show all reboots", delegate
			{
				ToggleFilterButton(Enum.GetName(typeof(ICULanLogType), ICULanLogType.RESET));
			}, Enum.GetName(typeof(ICULanLogType), ICULanLogType.RESET)),
			AddToolImageToggleButton(s_IconNames[7], "Show console commands", delegate
			{
				ToggleFilterButton(Enum.GetName(typeof(ICULanLogType), ICULanLogType.CONSOLE));
			}, Enum.GetName(typeof(ICULanLogType), ICULanLogType.CONSOLE)),
			AddToolImageToggleButton(s_IconNames[8], "Show security information", delegate
			{
				ToggleFilterButton(Enum.GetName(typeof(ICULanLogType), ICULanLogType.SECURITY));
			}, Enum.GetName(typeof(ICULanLogType), ICULanLogType.SECURITY))
		};
		m_btnCommand = AddToolImageButton("command.png", "Execute a command", delegate
		{
			ExecuteCommand();
		});
		m_btnSaveToFile = AddToolImageButton("save.png", "Save the log to a file", delegate
		{
			SaveLogFile();
		});
		m_btnLoadNext = AddToolImageButton("double-up-arrow.png", "Load previous log lines", delegate
		{
			LoadPreviousLines();
		});
		m_btnRefresh = AddToolImageButton("refresh24.png", "Refresh the log", delegate
		{
			ReloadLog();
		});
		m_lsStore = new ListStore(m_dfDate, m_dfType, m_dfFile, m_dfLine, m_dfInformation, m_dfSocket1, m_dfSocket2, m_dfState1, m_dfState2);
		m_lvwLog = new ListView
		{
			SelectionMode = SelectionMode.Single,
			DataSource = m_lsStore,
			WidthRequest = 200.0,
			HeightRequest = 200.0,
			GridLinesVisible = GridLines.Horizontal,
			Visible = false
		};
		AddListColumns();
		Add(m_lvwLog, 0, m_tableRowCounter, 1, 3, hexpand: true, vexpand: true);
		m_tableRowCounter++;
		Add(m_lblLineInfo, 0, m_tableRowCounter, 1, 3, hexpand: true);
		m_lblLineInfo.TextColor = Colors.Gray;
		m_tableRowCounter++;
		m_spnSaveSpinner.MinHeight = 48.0;
		Add(m_spnSaveSpinner, 0, m_tableRowCounter, 1, 3, hexpand: true, vexpand: true, WidgetPlacement.Fill, WidgetPlacement.Center);
		m_spnSaveSpinner.Visible = false;
		m_tableRowCounter++;
		m_lblSaveInfo.MinHeight = 100.0;
		m_lblSaveInfo.TextColor = Colors.Gray;
		Add(m_lblSaveInfo, 0, m_tableRowCounter, 1, 3, hexpand: true, vexpand: false, WidgetPlacement.Center);
		m_lblSaveInfo.Visible = false;
		m_tableRowCounter++;
		StartUpdateTimer(s_tickInterval);
		m_bgwUpdate.DoWork += OnUpdateDoWork;
		m_bgwUpdate.RunWorkerCompleted += OnUpdateCompleted;
		m_bgwUpdate.WorkerSupportsCancellation = true;
		m_bgwSave.WorkerSupportsCancellation = true;
		m_bgwSave.DoWork += OnSaveDoWork;
		m_bgwSave.ProgressChanged += OnSaveProgressChanged;
		m_bgwSave.RunWorkerCompleted += OnSaveCompleted;
		m_bgwSave.WorkerReportsProgress = true;
	}

	private void ToggleFilterButton(string buttonActive)
	{
		foreach (ToggleButton s_lstFilterButton in s_lstFilterButtons)
		{
			if (string.Compare(s_lstFilterButton.Name, buttonActive, ignoreCase: true) != 0)
			{
				s_lstFilterButton.Active = false;
			}
		}
		RefreshLog(m_currentDevice, updateSelectedRow: false);
	}

	public static void InitializeIcons()
	{
		string[] array = s_IconNames;
		foreach (string resourceName in array)
		{
			s_lstImages.Add(Image.FromResource(typeof(App), AppProperties.ResourcePath(resourceName)));
		}
	}

	private void AddListColumns()
	{
		if (m_lvwLog == null)
		{
			return;
		}
		m_lvwLog.Columns.Clear();
		m_lvwLog.Columns.Add(new ListViewColumn("Date", new TextCellView(m_dfDate)));
		m_lvwLog.Columns.Add(new ListViewColumn("", new ImageCellView(m_dfType)));
		if (m_btnFileAndLine.Active)
		{
			m_lvwLog.Columns.Add(new ListViewColumn("File", new TextCellView(m_dfFile)));
			m_lvwLog.Columns.Add(new ListViewColumn("Line", new TextCellView(m_dfLine)));
		}
		if (m_btnSocket.Active)
		{
			int num = 1;
			if (m_currentDevice != null)
			{
				num = m_currentDevice.NumberOfSockets;
			}
			m_lvwLog.Columns.Add(new ListViewColumn("Information", new CustomLogCell(m_dfInformation, LogField.InformationGrayed)));
			m_lvwLog.Columns.Add(new ListViewColumn("Socket #1", new CustomLogCell(m_dfInformation, LogField.Socket1)));
			m_lvwLog.Columns.Add(new ListViewColumn("State #1", new CustomLogCell(m_dfInformation, LogField.State1)));
			if (num > 1)
			{
				m_lvwLog.Columns.Add(new ListViewColumn("State #2", new CustomLogCell(m_dfInformation, LogField.State2)));
				m_lvwLog.Columns.Add(new ListViewColumn("Socket #2", new CustomLogCell(m_dfInformation, LogField.Socket2)));
			}
		}
		else
		{
			m_lvwLog.Columns.Add(new ListViewColumn("Information", new CustomLogCell(m_dfInformation, LogField.Information)));
		}
		foreach (ListViewColumn column in m_lvwLog.Columns)
		{
			column.CanResize = true;
		}
	}

	public override bool OnChangeDevice(ICUDevice newDevice, ICUDevice previousDevice)
	{
		if (!(newDevice is ICULanDevice iCULanDevice))
		{
			return true;
		}
		Logger.AddChargerContext(m_currentDevice).Information("PageView, {Panel}", Title);
		if (previousDevice is ICULanDevice iCULanDevice2)
		{
			iCULanDevice2.Log.LogLines.CollectionChanged -= OnLogLinesChanged;
		}
		m_lvwLog.Sensitive = newDevice != null;
		iCULanDevice.Log.LogLines.CollectionChanged -= OnLogLinesChanged;
		iCULanDevice.Log.LogLines.CollectionChanged += OnLogLinesChanged;
		if (iCULanDevice.IsLoggedIn)
		{
			m_bgwUpdate.DoWork -= OnUpdateDoWork;
			m_bgwUpdate.DoWork += OnUpdateDoWork;
		}
		RefreshLog(iCULanDevice, updateSelectedRow: true);
		m_btnFollow.Sensitive = newDevice != null;
		m_btnFileAndLine.Sensitive = newDevice != null;
		m_btnSocket.Sensitive = newDevice != null;
		m_btnSaveToFile.Sensitive = newDevice != null;
		bool sensitive = newDevice != null && (newDevice as ICULanDevice).Log.LogLines.Any();
		foreach (ToggleButton s_lstFilterButton in s_lstFilterButtons)
		{
			s_lstFilterButton.Sensitive = sensitive;
		}
		m_btnLoadNext.Sensitive = sensitive;
		m_btnRefresh.Sensitive = sensitive;
		if (m_currentUser != null && m_currentUser.GetRights("PAGE_LOG") == ICURights.Full)
		{
			s_lstFilterButtons[7].Visible = true;
			m_btnCommand.Visible = true;
			m_btnCommand.Sensitive = newDevice != null;
		}
		else
		{
			s_lstFilterButtons[7].Visible = false;
			m_btnCommand.Visible = false;
		}
		return false;
	}

	private void RefreshLog(ICUDevice newDevice, bool updateSelectedRow)
	{
		ICULanLogLine iCULanLogLine = null;
		if (m_lvwLog.SelectedRow >= 0)
		{
			iCULanLogLine = m_lsStore.GetValue(m_lvwLog.SelectedRow, m_dfInformation);
		}
		m_lsStore.Clear();
		if (!(newDevice is ICULanDevice iCULanDevice))
		{
			return;
		}
		int num = -1;
		int num2 = 0;
		ToggleButton toggleButton = s_lstFilterButtons.FirstOrDefault((ToggleButton a) => a.Active);
		foreach (ICULanLogLine logLine in iCULanDevice.Log.LogLines)
		{
			if (toggleButton == null || string.Compare(Enum.GetName(typeof(ICULanLogType), logLine.Type), toggleButton.Name, ignoreCase: true) == 0)
			{
				num2 = AddICULogLineToListStore(m_lsStore.AddRow(), logLine);
			}
			if (logLine == iCULanLogLine)
			{
				num = num2;
			}
		}
		if ((updateSelectedRow || linesAdded > 0) && num >= 0 && num < m_lsStore.RowCount)
		{
			m_lvwLog.SelectRow(num);
			m_lvwLog.ScrollToRow(num);
		}
	}

	private void OnLogLinesChanged(object sender, NotifyCollectionChangedEventArgs e)
	{
		Application.Invoke(() =>
		{
			switch (e.Action)
			{
			case NotifyCollectionChangedAction.Add:
			{
				ICULanLogLine ill = (ICULanLogLine)e.NewItems[0];
				AddICULogLineToListStore(m_lsStore.InsertRowAfter(e.NewStartingIndex - 1), ill);
				return;
			}
			case NotifyCollectionChangedAction.Remove:
				m_lsStore.RemoveRow(e.OldStartingIndex);
				return;
			case NotifyCollectionChangedAction.Replace:
			case NotifyCollectionChangedAction.Move:
				Logger.Error("Error, I do not support this event :)");
				break;
			}
			m_lsStore.Clear();
		});
	}

	private int AddICULogLineToListStore(int index, ICULanLogLine ill)
	{
		Image value = s_lstImages[0];
		if (ill.Type >= ICULanLogType.UNKNOWN && (int)ill.Type < s_lstImages.Count)
		{
			value = s_lstImages[(int)ill.Type];
		}
		m_lsStore.SetValues(index, m_dfDate, ill.Time.ToString(), m_dfType, value, m_dfFile, ill.SourceFileName, m_dfLine, ill.SourceLineNumber.ToString(), m_dfInformation, ill, m_dfSocket1, null, m_dfSocket2, null, m_dfState1, null, m_dfState2, null);
		return index;
	}

	private void SelectLastLogItem()
	{
		if (m_lsStore.RowCount > 0 && m_lvwLog != null)
		{
			int row = m_lsStore.RowCount - 1;
			m_lvwLog.SelectRow(row);
			m_lvwLog.ScrollToRow(row);
		}
	}

	protected override void OnUpdateTick()
	{
		ICULanDevice lanDev = m_currentDevice;
		if (lanDev == null || !lanDev.IsConnected || lanDev.LastHttpStatusCode != HttpStatusCode.OK)
		{
			return;
		}
		Application.Invoke(() =>
		{
			if (lanDev.Log.LogLines.Count == 0)
			{
				ShowLoadingMessage(loading: true);
			}
			UpdateLog();
		});
	}

	protected void UpdateLog(bool loadPreviousLines = false)
	{
		if (m_currentDevice != null && !m_bgwUpdate.IsBusy && !m_bgwSave.IsBusy && !m_bgwUpdate.CancellationPending)
		{
			Logger.Debug("Updating log lines");
			DeviceWorkerData argument = new DeviceWorkerData
			{
				CurrentDevice = m_currentDevice,
				Worker = m_bgwUpdate,
				JumpToAddedLines = loadPreviousLines
			};
			m_bgwUpdate.RunWorkerAsync(argument);
		}
	}

	private int GetPreferredNumberOfLines(bool loadPreviousLines)
	{
		if ((m_lsStore.RowCount == 0) | loadPreviousLines)
		{
			return m_nDesiredNumberOfLines;
		}
		return 16;
	}

	private void OnUpdateDoWork(object sender, DoWorkEventArgs e)
	{
		object argument = e.Argument;
		DeviceWorkerData wd = argument as DeviceWorkerData;
		if (wd == null)
		{
			return;
		}
		linesAdded = wd.CurrentDevice.Log.UpdateLog(GetPreferredNumberOfLines(wd.JumpToAddedLines));
		if (linesAdded > 0)
		{
			Application.Invoke(() =>
			{
				if (wd.JumpToAddedLines && m_lsStore.RowCount > linesAdded && m_lvwLog != null)
				{
					m_lvwLog.SelectRow(linesAdded);
					m_lvwLog.ScrollToRow(linesAdded);
				}
				else if (m_btnFollow.Active)
				{
					SelectLastLogItem();
				}
			});
		}
		else if (linesAdded == -1)
		{
			m_bgwUpdate.DoWork -= OnUpdateDoWork;
			m_bgwUpdate.CancelAsync();
		}
	}

	private void OnUpdateCompleted(object sender, RunWorkerCompletedEventArgs e)
	{
		ICULanDevice currentDevice = m_currentDevice;
		if (currentDevice != null)
		{
			if (m_bgwUpdate.CancellationPending)
			{
				MessageDialog.ShowWarning("Unable to retrieve Log Lines.");
				return;
			}
			ShowInfo(currentDevice);
			ShowLoadingMessage(loading: false);
			RefreshLog(currentDevice, updateSelectedRow: false);
		}
	}

	protected void ShowInfo(ICULanDevice lanDev)
	{
		int count = lanDev.Log.LogLines.Count;
		if (count > 0)
		{
			int num = lanDev.Log.LogLines.Count((ICULanLogLine a) => a.Type == ICULanLogType.ERROR);
			int num2 = lanDev.Log.LogLines.Count((ICULanLogLine a) => a.Type == ICULanLogType.WARNING);
			List<string> list = new List<string>();
			if (num > 0)
			{
				list.Add($"{num} error(s)");
			}
			if (num2 > 0)
			{
				list.Add($"{num2} warnings(s)");
			}
			m_lblLineInfo.Text = string.Format("Log from {0} to {1} ({2} lines). {3}", new object[4]
			{
				lanDev.Log.LogLines[count - 1].Time,
				lanDev.Log.LogLines[0].Time,
				count,
				string.Join(", ", list)
			});
		}
		else
		{
			m_lblLineInfo.Text = string.Format("No log lines available.", Array.Empty<object>());
		}
	}

	protected void LoadPreviousLines()
	{
		m_nDesiredNumberOfLines += s_maxLogLineBatchSize;
		ShowLoadingMessage(loading: true);
		UpdateLog(loadPreviousLines: true);
	}

	protected void ReloadLog()
	{
		ICULanDevice currentDevice = m_currentDevice;
		if (currentDevice != null)
		{
			ShowLoadingMessage(loading: true);
			currentDevice.Log.Clear();
			m_bgwUpdate.DoWork -= OnUpdateDoWork;
			m_bgwUpdate.DoWork += OnUpdateDoWork;
			RefreshLog(currentDevice, updateSelectedRow: true);
		}
	}

	private void ShowLoadingMessage(bool loading, bool saving = false)
	{
		m_lvwLog.Visible = !loading;
		m_lblLineInfo.Visible = !loading;
		m_lblSaveInfo.Text = "Loading log lines, please wait...";
		m_lblSaveInfo.Visible = loading;
		m_spnSaveSpinner.Visible = loading;
		m_spnSaveSpinner.Animate = loading;
		foreach (ToggleButton s_lstFilterButton in s_lstFilterButtons)
		{
			s_lstFilterButton.Sensitive = !loading;
		}
		m_btnLoadNext.Sensitive = !loading;
		m_btnRefresh.Sensitive = !loading;
		m_btnFollow.Sensitive = !loading;
		m_btnFileAndLine.Sensitive = !loading;
		m_btnSocket.Sensitive = !loading;
		m_btnCommand.Sensitive = !saving;
		m_btnSaveToFile.Sensitive = !saving;
	}

	protected void ExecuteCommand()
	{
		using DlgCommand dlgCommand = new DlgCommand(m_currentDevice);
		dlgCommand.Run(ParentWindow);
	}

	protected void SaveLogFile()
	{
		ICULanDevice currentDevice = m_currentDevice;
		if (currentDevice == null)
		{
			return;
		}
		MainWindow mainWindow = (MainWindow)ParentWindow;
		mainWindow.SuspendUpdateTimers(fSuspend: true);
		Dialog dialog = new Dialog
		{
			Title = "Save log file"
		};
		Table table = new Table();
		table.Add(new Label("How many days would you like to store to the log file?"), 0, 0, 1, 1, hexpand: true, vexpand: true, WidgetPlacement.Fill, WidgetPlacement.Fill, -1.0, -1.0, -1.0, -1.0, 8.0);
		dialog.Content = table;
		dialog.Buttons.Add(new DialogButton(new Command("1", "1 day")));
		dialog.Buttons.Add(new DialogButton(new Command("3", "3 days")));
		dialog.Buttons.Add(new DialogButton(new Command("7", "1 week")));
		dialog.Buttons.Add(new DialogButton(new Command("21", "3 weeks")));
		dialog.Buttons.Add(new DialogButton(new Command("0", "All")));
		dialog.Buttons.Add(new DialogButton(Command.Cancel));
		Command command = dialog.Run(ParentWindow);
		dialog.Dispose();
		if (command != null && command != Command.Cancel)
		{
			TimeSpan timeSpan = new TimeSpan(Convert.ToInt32(command.Id), 0, 0, 0);
			DateTime dateTime = DateTime.Now - timeSpan;
			string text = currentDevice.Identity + "-" + currentDevice.GetPropertyString(8273, 0, 0) + "-" + dateTime.ToUniversalTime().ToString("dd.MM.yyyy_HH.mm") + "-" + DateTime.Now.ToUniversalTime().ToString("dd.MM.yyyy_HH.mm");
			string text2 = new string(Path.GetInvalidFileNameChars()) + new string(Path.GetInvalidPathChars());
			for (int i = 0; i < text2.Length; i++)
			{
				text = text.Replace(text2[i].ToString(), "");
			}
			SaveFileDialog saveFileDialog = new SaveFileDialog("Log file location")
			{
				InitialFileName = text,
				Multiselect = false
			};
			saveFileDialog.Filters.Add(new FileDialogFilter("txt files", "*.txt"));
			saveFileDialog.Filters.Add(new FileDialogFilter("All files", "*.*"));
			if (saveFileDialog.Run(ParentWindow))
			{
				ShowLoadingMessage(loading: true, saving: true);
				if (!m_bgwSave.IsBusy)
				{
					DeviceWorkerData argument = new DeviceWorkerData
					{
						CurrentDevice = currentDevice,
						NumberOfDays = Convert.ToInt32(command.Id),
						Worker = m_bgwSave,
						Filename = saveFileDialog.FileName
					};
					m_bgwSave.RunWorkerAsync(argument);
				}
			}
			Logger.AddChargerContext(m_currentDevice).Information("Log saved, days: {Days}", command.Id);
		}
		mainWindow.SuspendUpdateTimers(fSuspend: false);
	}

	private void OnSaveDoWork(object sender, DoWorkEventArgs e)
	{
		if (e.Argument is DeviceWorkerData deviceWorkerData)
		{
			m_strDownloadMessage = deviceWorkerData.CurrentDevice.Log.SaveToFile(new TimeSpan(deviceWorkerData.NumberOfDays, 0, 0, 0), deviceWorkerData.Filename, deviceWorkerData.Worker);
		}
	}

	private void OnSaveCompleted(object sender, RunWorkerCompletedEventArgs e)
	{
		ShowLoadingMessage(loading: false);
		if (m_strDownloadMessage.StartsWith("Error:", StringComparison.OrdinalIgnoreCase))
		{
			MessageDialog.ShowWarning(m_strDownloadMessage);
		}
		else
		{
			MessageDialog.ShowMessage(m_strDownloadMessage);
		}
	}

	private void OnSaveProgressChanged(object sender, ProgressChangedEventArgs e)
	{
		if (e.UserState != null)
		{
			m_lblSaveInfo.Text = e.UserState.ToString();
		}
	}
}
