using Newtonsoft.Json;

namespace ICUFWUCreator.Model;

internal class AhpVideoResource
{
	[JsonProperty("logo")]
	public string LogoFileName { get; set; }

	[JsonProperty("logoMargins")]
	public int[] LogoMargins { get; set; }

	public AhpVideoResource(string logoFileName, int margin)
	{
		LogoFileName = logoFileName;
		LogoMargins = new int[4] { margin, margin, margin, margin };
	}
}
