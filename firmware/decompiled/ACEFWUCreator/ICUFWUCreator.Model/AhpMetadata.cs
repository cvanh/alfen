using System;
using Newtonsoft.Json;

namespace ICUFWUCreator.Model;

internal class AhpMetadata
{
	[JsonProperty("description")]
	public string Description { get; set; } = "Company logo";

	[JsonProperty("created-by")]
	public string CreatedBy { get; set; }

	[JsonProperty("created-at")]
	public string CreatedAt { get; set; } = DateTime.UtcNow.ToString("s") + "Z";
}
