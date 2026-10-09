using Newtonsoft.Json;

namespace ICUFWUCreator.Model;

internal class AhpManifest
{
	[JsonProperty("manifest-version")]
	public int ManifestVersion { get; set; }

	[JsonProperty("meta")]
	public AhpMetadata Metadata { get; set; }

	[JsonProperty("options")]
	public AhpManifestOptions Options { get; set; }

	internal AhpManifest(string creator, string logoName, int version = 1)
	{
		ManifestVersion = version;
		Metadata = new AhpMetadata
		{
			Description = logoName,
			CreatedBy = creator
		};
		Options = new AhpManifestOptions();
	}
}
