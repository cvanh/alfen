using System;
using Serilog;
using Xwt;

namespace ICUServiceInstaller;

public static class App
{
	public static void Run(ToolkitType type, AppOptions options)
	{
		Application.Initialize(type);
		DlgSplashScreen dlgSplashScreen = new DlgSplashScreen();
		dlgSplashScreen.Show();
		MainWindow mainWindow = new MainWindow(dlgSplashScreen, options);
		mainWindow.Show();
		Application.UnhandledException += (object sender, ExceptionEventArgs args) =>
		{
			HandleException(args.ErrorException);
		};
		try
		{
			Application.Run();
		}
		catch (Exception ex)
		{
			HandleException(ex);
		}
		finally
		{
			mainWindow.Dispose();
			Application.Dispose();
		}
	}

	private static void HandleException(Exception ex)
	{
		if (ex != null)
		{
			Log.Logger.Error(ex, "UnhandledException");
		}
	}
}
