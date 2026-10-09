using System;
using System.IO;
using ICUSettings;
using Serilog;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public static class AppProperties
{
	public static string AppName = "ACE Service Installer";

	public static string AppIconName = "ICUServiceInstaller.Resources.Icon.png";

	public static string FTPSite = "ftp://ftp.alfen.com/";

	public static string FTPUsername = "installer";

	public static string FTPPassword = "jIf978FQmk1W";

	public static string IsahSite = "https://installer.alfen.com";

	public static int FTPConnectTimeout = 2000;

	public static int FTPCommunicationTimeout = 2000;

	public static string LocalInstallerFolder = "ACE Service Installer";

	public static string LocalFirmwareFolderName = "Firmware";

	public static string LocalLogFolderName = "Log";

	public static string LocalOldPresetsFolderName = "Presets";

	public static string LocalTCPPresetsFolderName = "TCPPresets";

	public static string LocalRTUPresetsFolderName = "RTUPresets";

	public static string LocalBackofficePresetsFolderName = "BackofficePresets";

	public static string FTPFirmwareFolder = "Firmware";

	public static string FTPTCPPresetsFolder = "TCPPresets";

	public static string FTPRTUPresetsFolder = "RTUPresets";

	public static string FTPBackofficePresetsFolder = "BackofficePresets";

	public static string ConfigFileNameOld = "InstallerConfig.dat";

	public static string ConfigFileNameV2 = "InstallerConfigV2.dat";

	public static string ConfigFileName = "InstallerConfigV3.dat";

	public static string ErrorCodesFileName = "ErrorCodes.xlsx";

	public static string StatusDescription = "StatusDescription.xlsx";

	public static string StatusDescriptionPassword = "@lfen";

	public static string DefaultUserName = "User";

	public static int ButtonHeight = 24;

	public static int ButtonWidth = 120;

	public static ICUConfig ICUConfig = new ICUConfig();

	public static Font Font_BaseLabel = Font.FromName("Segoe UI");

	public static Font Font_Base = Font.FromName("Segoe UI");

	public static Color Color_Border = Color.FromBytes(171, 173, 179);

	public static Color Color_Disabled = Color.FromBytes(250, 250, 250);

	public static string FallbackSettingsFilename => "InstallerSettings.dat";

	public static string LocalSetupFolder => Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.ApplicationData), LocalInstallerFolder);

	public static string SettingsFilename => Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.ApplicationData), LocalInstallerFolder, ConfigFileName);

	public static string SettingsFilenameV2 => Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.ApplicationData), LocalInstallerFolder, ConfigFileNameV2);

	public static string SettingsFilenameOld => Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.ApplicationData), LocalInstallerFolder, ConfigFileNameOld);

	public static string LocalFirmwareFolder => Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.ApplicationData), LocalInstallerFolder, LocalFirmwareFolderName);

	public static string LocalLogFolder => Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.ApplicationData), LocalInstallerFolder, LocalFirmwareFolderName);

	public static string LocalOldPresetsFolder => Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.ApplicationData), LocalInstallerFolder, LocalOldPresetsFolderName);

	public static string LocalTCPPresetsFolder => Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.ApplicationData), LocalInstallerFolder, LocalTCPPresetsFolderName);

	public static string LocalRTUPresetsFolder => Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.ApplicationData), LocalInstallerFolder, LocalRTUPresetsFolderName);

	public static string LocalBackofficePresetsFolder => Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.ApplicationData), LocalInstallerFolder, LocalBackofficePresetsFolderName);

	public static string ProgramFilesPresetsFolder => "Presets";

	public static string UILanguagesFolder => "UILanguages";

	public static void InitializeSettings()
	{
		try
		{
			File.Delete(SettingsFilenameOld);
		}
		catch (Exception ex)
		{
			Log.Logger.Debug(ex, "Error while removing old settings file: {Message}", ex.Message);
		}
		try
		{
			File.Delete(SettingsFilenameV2);
		}
		catch (Exception ex2)
		{
			Log.Logger.Debug(ex2, "Error while removing old settings file: {Message}", ex2.Message);
		}
		if (!ICUConfig.ReadInstallerSettings(SettingsFilename, ICUEncryptionType.encryptRijndaelHashed))
		{
			ICUConfig.ReadInstallerSettings(ConfigFileName, ICUEncryptionType.encryptRijndaelHashed);
		}
	}

	public static string ResourcePath(string resourceName)
	{
		return $"ICUServiceInstaller.Resources.{resourceName}";
	}
}
