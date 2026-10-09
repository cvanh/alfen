using System.Net.Http;

namespace ICUIWSConnection;

public class IsahResponse
{
	public HttpResponseMessage HttpResponse { get; set; }

	public IWSObject IsahObject { get; set; }
}
