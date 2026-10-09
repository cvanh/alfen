using System.Collections.Generic;
using System.ComponentModel;
using System.IO;
using ICUNetwork;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class DlgDownloadFiles : Dialog
{
	private readonly Label m_lblText = new Label();

	private readonly List<string> m_downloadFiles;

	private readonly string m_sFTPSite;

	private readonly string m_sLocalFolder;

	private int m_nSuccessCounter;

	private readonly BackgroundWorker m_bgwUpload = new BackgroundWorker();

	public int DownloadCount => m_nSuccessCounter;

	public DlgDownloadFiles(List<string> downloadFiles, string ftpSite, string localFolder, bool showInTaskbar = false)
	{
		m_downloadFiles = downloadFiles;
		m_sFTPSite = ftpSite;
		m_sLocalFolder = localFolder;
		Title = AppProperties.AppName;
		ShowInTaskbar = showInTaskbar;
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
		m_lblText.Text = "";
		table.Add(m_lblText, 1, 0, 1, 1, hexpand: true, vexpand: false, WidgetPlacement.Center);
		Content = content;
		m_bgwUpload.WorkerReportsProgress = true;
		m_bgwUpload.DoWork += OnDownloadDoWork;
		m_bgwUpload.ProgressChanged += OnDownloadProgressChanged;
		m_bgwUpload.RunWorkerCompleted += OnDownloadCompleted;
		m_bgwUpload.RunWorkerAsync();
	}

	private void OnDownloadDoWork(object sender, DoWorkEventArgs e)
	{
		int num = 0;
		m_nSuccessCounter = 0;
		foreach (string downloadFile in m_downloadFiles)
		{
			string userState = $"Downloading file {num} of {m_downloadFiles.Count}";
			m_bgwUpload.ReportProgress(0, userState);
			if (UpdateManager.DownloadFile(Path.GetFileName(downloadFile), m_sFTPSite, m_sLocalFolder, OnShowError))
			{
				m_nSuccessCounter++;
			}
			num++;
		}
	}

	private void OnDownloadProgressChanged(object sender, ProgressChangedEventArgs e)
	{
		m_lblText.Text = e.UserState.ToString();
	}

	private void OnDownloadCompleted(object sender, RunWorkerCompletedEventArgs e)
	{
		Close();
	}

	private void OnShowError(string primaryText, string secondaryText)
	{
		MessageDialog.ShowError(primaryText, secondaryText);
	}
}
