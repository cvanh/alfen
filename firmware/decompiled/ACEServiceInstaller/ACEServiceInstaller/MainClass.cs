using System;
using System.Diagnostics;
using System.IO;
using System.Reflection;
using System.Threading;
using ICUServiceInstaller;
using ICUServiceInstaller.Properties;
using ICUSettings;
using Microsoft.ApplicationInsights.Extensibility;
using Microsoft.ApplicationInsights.WindowsServer.TelemetryChannel;
using Serilog;
using Serilog.Configuration;
using Serilog.Events;
using Serilog.Exceptions;
using Serilog.Formatting.Compact;
using Xwt;

namespace ACEServiceInstaller;

internal static class MainClass
{
	private static readonly string _appName = "ACE Service Installer 4.0";

	private static readonly Mutex _mutex = new Mutex(initiallyOwned: false, _appName);

	private static readonly EncryptDecrypt _encryptDecrypt = new EncryptDecrypt();

	[STAThread]
	public static void Main(string[] args)
	{
		string logFilePath = GetOrCreateLogFolder();
		TelemetryConfiguration telemetryConfiguration = TelemetryConfiguration.CreateDefault();
		telemetryConfiguration.TelemetryInitializers.Add(new SessionInitializer());
		telemetryConfiguration.ConnectionString = GetAppInsightsConnectionString();
		ServerTelemetryChannel telemetryChannel = new ServerTelemetryChannel
		{
			StorageFolder = logFilePath,
			MaxTelemetryBufferCapacity = 500,
			MaxTransmissionBufferCapacity = 500
		};
		telemetryConfiguration.TelemetryChannel = telemetryChannel;
		LoggerConfiguration loggerConfiguration = new LoggerConfiguration().Enrich.With(new AlfenDomain()).Enrich.WithExceptionDetails();
		loggerConfiguration.MinimumLevel.Information();
		loggerConfiguration.WriteTo.ApplicationInsights(telemetryConfiguration, TelemetryConverter.Events);
		if (!string.IsNullOrEmpty(logFilePath))
		{
			loggerConfiguration.WriteTo.Async((LoggerSinkConfiguration a) =>
			{
				a.File(new CompactJsonFormatter(), Path.Combine(logFilePath, $"service_installer_{DateTime.Now:yyyyMMdd-HHmmss}.json"), LogEventLevel.Verbose, retainedFileCountLimit: 20, fileSizeLimitBytes: 1073741824L, levelSwitch: null, buffered: false, shared: false, flushToDiskInterval: null, rollingInterval: RollingInterval.Day);
			});
		}
		Log.Logger = loggerConfiguration.CreateLogger();
		FileVersionInfo versionInfo = FileVersionInfo.GetVersionInfo(Assembly.GetExecutingAssembly().Location);
		Log.Logger.Information("Application started, Version: {SiaVersion}", versionInfo.FileVersion);
		bool flag = args.Length != 0 && args[0] == "multimode";
		if (!flag && !_mutex.WaitOne(TimeSpan.FromSeconds(2.0), exitContext: false))
		{
			Log.Logger.Error("Another instance of the app is running. Bye!");
			return;
		}
		Log.Logger.Debug("Log folder: {Path}", logFilePath);
		try
		{
			AppOptions options = AppOptions.aoNormal;
			if (args.Length != 0 && args[0].ToLowerInvariant().Trim(new char[2] { ' ', '-' }) == "large")
			{
				options = AppOptions.aoLarge;
			}
			App.Run(ToolkitType.Wpf, options);
		}
		finally
		{
			Log.Logger.Information("Application stopped");
			Log.CloseAndFlush();
			if (!flag)
			{
				_mutex.ReleaseMutex();
			}
		}
	}

	private static string GetOrCreateLogFolder()
	{
		string text = Path.Combine(Path.GetTempPath(), "SiaLogs");
		try
		{
			if (!Directory.Exists(text))
			{
				Directory.CreateDirectory(text);
			}
			return text;
		}
		catch (Exception)
		{
			MessageDialog.ShowMessage("Logging not enabled, could not find or create a folder at location: " + Path.GetTempPath());
		}
		return null;
	}

	private static string GetAppInsightsConnectionString()
	{
		string text;
		try
		{
			text = _encryptDecrypt.Decrypt(Settings.Default.AppInsightsConnStrEncrypted, _appName);
		}
		catch
		{
			text = string.Empty;
		}
		string cipherText = "E7ji0TFnGUmotxceBawiF0E+Yboq1TUOzi2G72f/p6j3c/aCpYXdjvDvaREZbNaNPkezgTVGVcNxgKo2iHGFcMxZSspYnNczxxhcswcpd6CJ53z15Lit9znAnutxWWrXwNzB7ObOano0c+4L2W3xyt8uVPlnOWwRWQVaUkE4LcWinVAr6e3xYMtsg/96bWsC4SR6XxNUGu/MsULiPY00bxkZViG76SAV3dLalwDAriLm1Ktyx0EOMDkFn33eb/iRrvN8ZHvcjBl9TkbuOuEjmH3JiNNey9dcwt17Jip8YBokzZFz5MmkG2U3Kqe4uh4VMjdQMWZdjZ0uokqDwlLTGw==";
		if (string.IsNullOrEmpty(text))
		{
			text = _encryptDecrypt.Decrypt(cipherText, _appName);
		}
		return text;
	}
}
