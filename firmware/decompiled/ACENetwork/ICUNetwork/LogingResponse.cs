using Newtonsoft.Json;

namespace ICUNetwork;

internal class LogingResponse
{
	[JsonProperty("access")]
	internal string AccessToken;

	[JsonProperty("refresh")]
	internal string RefreshToken;
}
