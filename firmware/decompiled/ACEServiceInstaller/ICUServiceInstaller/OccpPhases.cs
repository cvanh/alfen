using ICUNetwork;

namespace ICUServiceInstaller;

public class OccpPhases
{
	public EOcppMeasurand Measurand { get; private set; }

	public EOccpPhase Phase { get; private set; }

	public string Key { get; private set; }

	public OccpPhases(EOcppMeasurand measurand, EOccpPhase phase, string key)
	{
		Measurand = measurand;
		Phase = phase;
		Key = key;
	}
}
