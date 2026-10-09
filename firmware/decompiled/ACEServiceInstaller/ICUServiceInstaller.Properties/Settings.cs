using System;
using System.CodeDom.Compiler;
using System.Collections.Specialized;
using System.Configuration;
using System.Diagnostics;
using System.Runtime.CompilerServices;

namespace ICUServiceInstaller.Properties;

[CompilerGenerated]
[GeneratedCode("Microsoft.VisualStudio.Editors.SettingsDesigner.SettingsSingleFileGenerator", "17.13.0.0")]
internal sealed class Settings : ApplicationSettingsBase
{
	private static Settings defaultInstance = (Settings)(object)SettingsBase.Synchronized((SettingsBase)(object)new Settings());

	public static Settings Default => defaultInstance;

	[UserScopedSetting]
	[DebuggerNonUserCode]
	[DefaultSettingValue("")]
	public string LastUserName
	{
		get
		{
			return (string)((SettingsBase)this)["LastUserName"];
		}
		set
		{
			((SettingsBase)this)["LastUserName"] = value;
		}
	}

	[UserScopedSetting]
	[DebuggerNonUserCode]
	public StringCollection LocalPasswords
	{
		get
		{
			return (StringCollection)((SettingsBase)this)["LocalPasswords"];
		}
		set
		{
			((SettingsBase)this)["LocalPasswords"] = value;
		}
	}

	[UserScopedSetting]
	[DebuggerNonUserCode]
	[DefaultSettingValue("")]
	public string LastManualIPAddress
	{
		get
		{
			return (string)((SettingsBase)this)["LastManualIPAddress"];
		}
		set
		{
			((SettingsBase)this)["LastManualIPAddress"] = value;
		}
	}

	[UserScopedSetting]
	[DebuggerNonUserCode]
	[DefaultSettingValue("443")]
	public int LastManualIPPort
	{
		get
		{
			return (int)((SettingsBase)this)["LastManualIPPort"];
		}
		set
		{
			((SettingsBase)this)["LastManualIPPort"] = value;
		}
	}

	[UserScopedSetting]
	[DebuggerNonUserCode]
	[DefaultSettingValue("")]
	public string LastCommand
	{
		get
		{
			return (string)((SettingsBase)this)["LastCommand"];
		}
		set
		{
			((SettingsBase)this)["LastCommand"] = value;
		}
	}

	[UserScopedSetting]
	[DebuggerNonUserCode]
	[DefaultSettingValue("True")]
	public bool AskConfirmExit
	{
		get
		{
			return (bool)((SettingsBase)this)["AskConfirmExit"];
		}
		set
		{
			((SettingsBase)this)["AskConfirmExit"] = value;
		}
	}

	[UserScopedSetting]
	[DebuggerNonUserCode]
	[DefaultSettingValue("True")]
	public bool AskSaveChanges
	{
		get
		{
			return (bool)((SettingsBase)this)["AskSaveChanges"];
		}
		set
		{
			((SettingsBase)this)["AskSaveChanges"] = value;
		}
	}

	[UserScopedSetting]
	[DebuggerNonUserCode]
	[DefaultSettingValue("2020-01-01")]
	public DateTime LastFTPActivity
	{
		get
		{
			return (DateTime)((SettingsBase)this)["LastFTPActivity"];
		}
		set
		{
			((SettingsBase)this)["LastFTPActivity"] = value;
		}
	}

	[UserScopedSetting]
	[DebuggerNonUserCode]
	[DefaultSettingValue("2020-01-01")]
	public DateTime LastFTPConnectionWarning
	{
		get
		{
			return (DateTime)((SettingsBase)this)["LastFTPConnectionWarning"];
		}
		set
		{
			((SettingsBase)this)["LastFTPConnectionWarning"] = value;
		}
	}

	[UserScopedSetting]
	[DebuggerNonUserCode]
	[DefaultSettingValue("")]
	public string LastDevicePassword
	{
		get
		{
			return (string)((SettingsBase)this)["LastDevicePassword"];
		}
		set
		{
			((SettingsBase)this)["LastDevicePassword"] = value;
		}
	}

	[UserScopedSetting]
	[DebuggerNonUserCode]
	[DefaultSettingValue("")]
	public string LastDeviceUsername
	{
		get
		{
			return (string)((SettingsBase)this)["LastDeviceUsername"];
		}
		set
		{
			((SettingsBase)this)["LastDeviceUsername"] = value;
		}
	}

	[UserScopedSetting]
	[DebuggerNonUserCode]
	[DefaultSettingValue("False")]
	public bool StorePasswords
	{
		get
		{
			return (bool)((SettingsBase)this)["StorePasswords"];
		}
		set
		{
			((SettingsBase)this)["StorePasswords"] = value;
		}
	}

	[UserScopedSetting]
	[DebuggerNonUserCode]
	[DefaultSettingValue("2020-01-01")]
	public DateTime LastPasswordStoreTime
	{
		get
		{
			return (DateTime)((SettingsBase)this)["LastPasswordStoreTime"];
		}
		set
		{
			((SettingsBase)this)["LastPasswordStoreTime"] = value;
		}
	}

	[UserScopedSetting]
	[DebuggerNonUserCode]
	[DefaultSettingValue("")]
	public string LastManualHostname
	{
		get
		{
			return (string)((SettingsBase)this)["LastManualHostname"];
		}
		set
		{
			((SettingsBase)this)["LastManualHostname"] = value;
		}
	}

	[UserScopedSetting]
	[DebuggerNonUserCode]
	[DefaultSettingValue("2")]
	public int LastManualNumberOfSockets
	{
		get
		{
			return (int)((SettingsBase)this)["LastManualNumberOfSockets"];
		}
		set
		{
			((SettingsBase)this)["LastManualNumberOfSockets"] = value;
		}
	}

	[UserScopedSetting]
	[DebuggerNonUserCode]
	[DefaultSettingValue("True")]
	public bool LastManualLoginRequired
	{
		get
		{
			return (bool)((SettingsBase)this)["LastManualLoginRequired"];
		}
		set
		{
			((SettingsBase)this)["LastManualLoginRequired"] = value;
		}
	}

	[UserScopedSetting]
	[DebuggerNonUserCode]
	[DefaultSettingValue("")]
	public string LastManualModelType
	{
		get
		{
			return (string)((SettingsBase)this)["LastManualModelType"];
		}
		set
		{
			((SettingsBase)this)["LastManualModelType"] = value;
		}
	}

	[UserScopedSetting]
	[DebuggerNonUserCode]
	[DefaultSettingValue("E7ji0TFnGUmotxceBawiF+hQsMNRKyCxnWpyUgaCFk0tuys8pdc8NZc4hIrq/9KEU9ImgSb8M8a/R/pUsPibbb3A1bF0fV0Fb3HL4CBsRogXfExg8GGiyAf3M3IxYkEgptTGLicaggQQFFVOGX6RiRKadF7p4OvnrpdIQH3CDSJx8ELDkF3zoBt6MkFtt4KKC7xVI04shWSZE9oPysbzAsAm524dXSwJXoUHsRddvq0UxbJYflFoRjia6tTetsFQG9BCFC+n2fHXDbIgnqYJlS3dwmIwWVAjrbHlbPR8lG6LIeZUgAUCmj1foQAQyUFr/E9AYh3hlcv8FVjzXrtgvA==")]
	public string AppInsightsConnStrEncrypted
	{
		get
		{
			return (string)((SettingsBase)this)["AppInsightsConnStrEncrypted"];
		}
		set
		{
			((SettingsBase)this)["AppInsightsConnStrEncrypted"] = value;
		}
	}
}
