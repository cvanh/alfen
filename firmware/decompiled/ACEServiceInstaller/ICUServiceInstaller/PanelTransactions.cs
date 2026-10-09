using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.ComponentModel;
using System.Linq;
using System.Net;
using ICUNetwork;
using ICUServiceInstaller.Utils;
using Serilog;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class PanelTransactions : PanelBase
{
	private readonly ILogger Logger = Log.ForContext<PanelTransactions>();

	protected static int s_tickInterval = 1000;

	protected ListView m_lstTransactions = new ListView();

	protected ListStore m_lsTransactions;

	protected DataField<string> m_dfType = new DataField<string>();

	protected DataField<string> m_dfId = new DataField<string>();

	protected DataField<DateTime?> m_dfStartTime = new DataField<DateTime?>();

	protected DataField<string> m_dfDuration = new DataField<string>();

	protected DataField<string> m_dfStartEnergy = new DataField<string>();

	protected DataField<string> m_dfStopEnergy = new DataField<string>();

	protected DataField<string> m_dfTag = new DataField<string>();

	protected DataField<int> m_dfSocket = new DataField<int>();

	protected DataField<long> m_dfOffset = new DataField<long>();

	protected DataField<string> m_dfInfo = new DataField<string>();

	private readonly Dictionary<ICUTransactionType, string> m_transactionTypeText;

	protected bool m_fTransactionsCollected;

	protected Button m_btnClearList;

	protected Button m_btnSaveList;

	protected Button m_btnRefresh;

	protected Spinner m_spnSaveSpinner = new Spinner();

	protected Label m_lblSaveInfo = new Label();

	protected ProgressBar m_prbUpload = new ProgressBar();

	protected Label m_lblLineInfo = new Label();

	protected BackgroundWorker m_bgwUpload = new BackgroundWorker();

	public PanelTransactions(MainWindow parent, string pageID = "", bool showBorder = true)
		: base(parent, pageID, fIndent: false, showBorder)
	{
		Title = "Transactions";
		Tooltip = "View all transactions";
		IconName = "cash-register.png";
		AddHeader(Tooltip);
		m_btnClearList = AddToolImageButton("closed-trash-can.png", "Clear all transactions", OnClearClicked);
		m_btnSaveList = AddToolImageButton("save.png", "Save all transactions to a file", OnSaveListClicked);
		m_btnRefresh = AddToolImageButton("refresh24.png", "Refresh the transaction list", OnRefreshClicked);
		m_prbUpload.Visible = false;
		m_prbUpload.MinHeight = 16.0;
		m_lsTransactions = new ListStore(m_dfType, m_dfId, m_dfStartTime, m_dfDuration, m_dfStartEnergy, m_dfStopEnergy, m_dfTag, m_dfSocket, m_dfOffset, m_dfInfo);
		m_lstTransactions.DataSource = m_lsTransactions;
		m_lstTransactions.SelectionMode = SelectionMode.Single;
		m_lstTransactions.Columns.Add(new ListViewColumn("Type", new TextCellView(m_dfType)));
		m_lstTransactions.Columns.Add(new ListViewColumn("Id", new TextCellView(m_dfId)));
		m_lstTransactions.Columns.Add(new ListViewColumn("Start time", new TextCellView(m_dfStartTime)));
		m_lstTransactions.Columns.Add(new ListViewColumn("Duration", new TextCellView(m_dfDuration)));
		m_lstTransactions.Columns.Add(new ListViewColumn("Start Energy (kWh)", new TextCellView(m_dfStartEnergy)));
		m_lstTransactions.Columns.Add(new ListViewColumn("Stop Energy (kWh)", new TextCellView(m_dfStopEnergy)));
		m_lstTransactions.Columns.Add(new ListViewColumn("Tag", new TextCellView(m_dfTag)));
		m_lstTransactions.Columns.Add(new ListViewColumn("Socket", new TextCellView(m_dfSocket)));
		m_lstTransactions.Columns.Add(new ListViewColumn("Info", new TextCellView(m_dfInfo)));
		m_lstTransactions.GridLinesVisible = GridLines.Horizontal;
		m_lstTransactions.HeightRequest = (m_lstTransactions.WidthRequest = 100.0);
		m_lstTransactions.MinHeight = 240.0;
		m_lstTransactions.SelectionMode = SelectionMode.Single;
		m_lstTransactions.Visible = false;
		Add(m_lstTransactions, 0, m_tableRowCounter, 1, 3, hexpand: true, vexpand: true);
		m_tableRowCounter++;
		Add(m_prbUpload, 0, m_tableRowCounter, 1, 3, hexpand: true);
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
		Add(m_lblLineInfo, 0, m_tableRowCounter, 1, 3, hexpand: true);
		m_lblLineInfo.TextColor = Colors.Gray;
		m_tableRowCounter++;
		m_transactionTypeText = new Dictionary<ICUTransactionType, string>
		{
			{
				ICUTransactionType.Unknown,
				"[unknown]"
			},
			{
				ICUTransactionType.Transaction,
				"Transaction"
			},
			{
				ICUTransactionType.Reservation,
				"Reservation"
			},
			{
				ICUTransactionType.MeterValue,
				"Meter value"
			},
			{
				ICUTransactionType.StatusNotification,
				"Status Notification"
			},
			{
				ICUTransactionType.StartTransaction,
				"Start Transaction"
			},
			{
				ICUTransactionType.StopTransaction,
				"Stop Transaction"
			},
			{
				ICUTransactionType.DateTimeOffset,
				"Time sync offset"
			},
			{
				ICUTransactionType.ReservationStatus,
				"Reservation Status"
			},
			{
				ICUTransactionType.SecurityEvent,
				"Security Event"
			}
		};
		StartUpdateTimer(s_tickInterval);
		m_bgwUpload.WorkerReportsProgress = true;
		m_bgwUpload.DoWork += OnUploadDoWork;
	}

	public override bool OnChangeDevice(ICUDevice newDevice, ICUDevice previousDevice)
	{
		if (!(newDevice is ICULanDevice))
		{
			return true;
		}
		Logger.AddChargerContext(m_currentDevice).Information("PageView, {Panel}", Title);
		if (newDevice != previousDevice)
		{
			m_fTransactionsCollected = false;
		}
		UpdateButtons();
		return false;
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
			if (!m_fTransactionsCollected && !m_bgwUpload.IsBusy)
			{
				ShowLoadingMessage(loading: true);
				TransactionWorkerData argument = new TransactionWorkerData
				{
					CurrentDevice = lanDev,
					Worker = m_bgwUpload
				};
				m_bgwUpload.RunWorkerAsync(argument);
				m_fTransactionsCollected = true;
			}
		});
	}

	private void UpdateButtons()
	{
		m_btnClearList.Sensitive = m_currentDevice != null;
		m_btnSaveList.Sensitive = m_currentDevice != null;
		m_btnRefresh.Sensitive = m_currentDevice != null;
		m_lblLineInfo.Text = $"Showing {m_lsTransactions.RowCount} transaction lines";
	}

	private void ShowLoadingMessage(bool loading)
	{
		m_lstTransactions.Visible = !loading;
		m_lblLineInfo.Visible = !loading;
		m_lblSaveInfo.Text = "Loading transactions, please wait...";
		m_lblSaveInfo.Visible = loading;
		m_spnSaveSpinner.Visible = loading;
		m_spnSaveSpinner.Animate = loading;
		UpdateButtons();
	}

	private void FillList()
	{
		try
		{
			long lOffset = 0L;
			Application.Invoke(() =>
			{
				int selectedRow = m_lstTransactions.SelectedRow;
				if (selectedRow >= 0)
				{
					lOffset = m_lsTransactions.GetValue(selectedRow, m_dfOffset);
				}
				m_lsTransactions.Clear();
			});
			int newSel = 0;
			ICULanDevice currentDevice = m_currentDevice;
			if (currentDevice != null)
			{
				currentDevice.Transactions.Read();
				ObservableCollection<ICUTransactionItem> observableCollection = new ObservableCollection<ICUTransactionItem>();
				foreach (ICUTransactionItem transaction in currentDevice.Transactions.Transactions)
				{
					if (transaction.Type != ICUTransactionType.StartTransaction && transaction.Type != ICUTransactionType.StopTransaction)
					{
						observableCollection.Add(transaction);
					}
					else
					{
						if (transaction.Type != ICUTransactionType.StartTransaction)
						{
							continue;
						}
						ICUTransactionItem iCUTransactionItem = null;
						transaction.Type = ICUTransactionType.Transaction;
						int count = currentDevice.Transactions.Transactions.IndexOf(transaction);
						foreach (ICUTransactionItem item in currentDevice.Transactions.Transactions.Skip(count))
						{
							if (item.Type == ICUTransactionType.StopTransaction && item.Socket == transaction.Socket && item.StopTime > transaction.StartTime)
							{
								iCUTransactionItem = item;
								break;
							}
						}
						if (iCUTransactionItem != null)
						{
							transaction.StopMeterValue = iCUTransactionItem.StopMeterValue;
							transaction.StopTag = iCUTransactionItem.StopTag;
							transaction.StopTime = iCUTransactionItem.StopTime;
							transaction.StopToSend = iCUTransactionItem.StopToSend;
						}
						observableCollection.Add(transaction);
					}
				}
				foreach (ICUTransactionItem ti in observableCollection)
				{
					string sType = m_transactionTypeText.FirstOrDefault((KeyValuePair<ICUTransactionType, string> x) => x.Key == ti.Type).Value;
					if (string.IsNullOrEmpty(sType))
					{
						m_transactionTypeText.TryGetValue(ICUTransactionType.Unknown, out sType);
					}
					string sDuration = "";
					if (ti.StopTime != DateTime.MinValue)
					{
						sDuration = (ti.StopTime - ti.StartTime)?.ToString();
					}
					else if (ti.Type == ICUTransactionType.Transaction)
					{
						sDuration = "[Charging]";
					}
					string sStartEnergy = "";
					if (ti.StartMeterValue.HasValue)
					{
						sStartEnergy = $"{ti.StartMeterValue:0.000}";
					}
					string sStopEnergy = "";
					if (ti.StopMeterValue.HasValue && ti.StopMeterValue != 0.0 && ti.Type == ICUTransactionType.Transaction)
					{
						sStopEnergy = $"{ti.StopMeterValue:0.000}";
					}
					string sTag = ti.StartTag;
					if (ti.StopTag != ti.StartTag && ti.StopTag != null)
					{
						sTag = ti.StartTag + " - " + ti.StopTag;
					}
					string sInfo = ti.ExtraInfo;
					if (ti.StopReason != ICUTransactionStopReason.Local && ti.StopReason != ICUTransactionStopReason.None)
					{
						sInfo += ti.StopReason;
					}
					else if (ti.TriggerReason != ICUTransactionTriggerReason.None || ti.ChargingState != ICUTransactionChargingState.None)
					{
						if (ti.TriggerReason != ICUTransactionTriggerReason.None)
						{
							sInfo += ti.TriggerReason;
						}
						if (ti.ChargingState != ICUTransactionChargingState.None)
						{
							sInfo += ti.ChargingState;
						}
					}
					else if (ti.SecurityEvent != ICUTransactionSecurityEvent.None)
					{
						sInfo += ti.SecurityEvent;
					}
					else if (ti.ReservationStatus != ICUTransactionReservationStatus.Unknown)
					{
						sInfo += ti.ReservationStatus;
					}
					Application.Invoke(() =>
					{
						int num = m_lsTransactions.AddRow();
						if (ti.Offset == lOffset)
						{
							newSel = num;
						}
						m_lsTransactions.SetValues(num, m_dfType, sType, m_dfId, ti.TransactionId.ToString(), m_dfStartTime, ti.StartTime, m_dfDuration, sDuration, m_dfStartEnergy, sStartEnergy, m_dfStopEnergy, sStopEnergy, m_dfTag, sTag, m_dfSocket, (int)ti.Socket, m_dfOffset, ti.Offset, m_dfInfo, sInfo);
					});
				}
			}
			Application.Invoke(() =>
			{
				m_lstTransactions.Sensitive = m_lsTransactions.RowCount > 0;
				if (m_lsTransactions.RowCount > newSel)
				{
					m_lstTransactions.SelectRow(newSel);
				}
			});
		}
		catch (Exception ex)
		{
			Logger.Error(ex, ex.Message);
		}
		Application.Invoke(() =>
		{
			ShowLoadingMessage(loading: false);
		});
	}

	private void OnClearClicked(object sender, EventArgs e)
	{
		ICULanDevice currentDevice = m_currentDevice;
		if (currentDevice != null && MessageDialog.AskQuestion("Are you sure you want to clear the entire transaction database?", Command.Yes, Command.No, Command.Cancel) == Command.Yes && currentDevice.Transactions.Clear())
		{
			m_lsTransactions.Clear();
			RefreshList();
		}
	}

	private void OnRefreshClicked(object sender, EventArgs e)
	{
		RefreshList();
	}

	private void RefreshList()
	{
		ICULanDevice currentDevice = m_currentDevice;
		if (currentDevice != null)
		{
			ShowLoadingMessage(loading: true);
			TransactionWorkerData argument = new TransactionWorkerData
			{
				CurrentDevice = currentDevice,
				Worker = m_bgwUpload
			};
			m_bgwUpload.RunWorkerAsync(argument);
			m_fTransactionsCollected = true;
		}
	}

	private void OnSaveListClicked(object sender, EventArgs e)
	{
		ICULanDevice currentDevice = m_currentDevice;
		if (currentDevice == null)
		{
			return;
		}
		using SaveFileDialog saveFileDialog = new SaveFileDialog("Save Transactions As")
		{
			InitialFileName = m_currentDevice.Identification + "_Transactions.csv",
			Multiselect = false
		};
		saveFileDialog.Filters.Add(new FileDialogFilter("Transactions files", "*.csv"));
		saveFileDialog.Filters.Add(new FileDialogFilter("All files", "*.*"));
		if (saveFileDialog.Run(ParentWindow))
		{
			currentDevice.Transactions.SaveToCSV(saveFileDialog.FileName);
			Logger.AddChargerContext(m_currentDevice).Information("Transactions saved");
		}
	}

	private void OnUploadDoWork(object sender, DoWorkEventArgs e)
	{
		if (e.Argument is TransactionWorkerData)
		{
			FillList();
		}
	}
}
