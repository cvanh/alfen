using System;

namespace ICUServiceInstaller.Utils;

internal class DisplayLanguage
{
	internal string Locale { get; set; }

	internal string Name { get; set; }

	internal Version FromNG9Version { get; set; }

	internal Version ToNG9Version { get; set; } = new Version(999, 0);

	internal Version FromAhpVersion { get; set; }

	internal Version FromDcVersion { get; set; }

	internal static Version MaxVersion { get; } = new Version(999, 0);

	internal DisplayLanguage(string locale, string name, string ng9VersionRange, string ahpVersion, string dcVersion)
	{
		Locale = locale;
		Name = name;
		try
		{
			SetNg9VersionRange(ng9VersionRange);
			FromAhpVersion = (string.IsNullOrEmpty(ahpVersion) ? MaxVersion : new Version(ahpVersion));
			FromDcVersion = (string.IsNullOrEmpty(dcVersion) ? MaxVersion : new Version(dcVersion));
		}
		catch (Exception)
		{
		}
	}

	private void SetNg9VersionRange(string versionRange)
	{
		if (!string.IsNullOrEmpty(versionRange))
		{
			string[] array = versionRange.Split(new char[1] { '-' });
			FromNG9Version = (string.IsNullOrEmpty(array[0]) ? MaxVersion : new Version(array[0]));
			ToNG9Version = ((array.Length == 2 && !string.IsNullOrEmpty(array[1])) ? new Version(array[1]) : MaxVersion);
		}
	}
}
