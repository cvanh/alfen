using ICUNetwork;

namespace ICUServiceInstaller;

public class OccpVersions
{
	public string Key { get; private set; }

	public string Name { get; private set; }

	public EOccpVersion Version { get; private set; }

	public OccpVersions(string key, string name, EOccpVersion version)
	{
		Key = key;
		Name = name;
		Version = version;
	}
}
