using Newtonsoft.Json;

namespace ICUFWUCreator.Model;

internal class AhpManifestOptions
{
	[JsonProperty("reboot-scb-after-stage-install")]
	public bool Reboot { get; set; }

	[JsonProperty("wait-for-charging-sessions-to-finish")]
	public bool WaitForFinish { get; set; }
}
