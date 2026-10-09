using ICUNetwork;

namespace ICUServiceInstaller;

public class OccpMeasurand
{
	public EOcppMeasurand Measurand { get; private set; }

	public EOccpVersion Version { get; private set; }

	public string Key { get; private set; }

	public OccpMeasurand(EOcppMeasurand measurand, EOccpVersion version, string key)
	{
		Measurand = measurand;
		Version = version;
		Key = key;
	}
}
