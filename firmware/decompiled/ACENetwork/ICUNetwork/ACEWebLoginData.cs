using System.Net;

namespace ICUNetwork;

public class ACEWebLoginData
{
	public const string AdminUser = "admin";

	public const string ServiceUser = "service";

	public const string TempUser = "temp";

	public string Username { get; set; }

	public string Password { get; set; }

	public string DisplayName { get; set; }

	public HttpStatusCode LastHttpStatusCode { get; set; }

	public bool IsLoggedIn { get; set; }

	public bool IsLoginCancelled { get; set; }

	public string LoginError { get; set; }

	public bool IsTempUser => Username == "temp";

	public bool IsAdminUser => Username == "admin";

	public bool IsServiceUser => Username == "service";
}
