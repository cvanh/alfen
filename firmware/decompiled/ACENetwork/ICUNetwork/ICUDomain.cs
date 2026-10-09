using System;
using System.Text;
using Serilog;

namespace ICUNetwork;

public class ICUDomain
{
	private readonly ILogger Logger = Log.ForContext<ICUDomain>();

	private readonly ICULanDevice m_device;

	public ICUDomain(ICULanDevice device)
	{
		m_device = device;
	}

	public bool AddOrUpdateItem(EDomainItemType type, string data)
	{
		if (m_device == null)
		{
			return false;
		}
		string arg = BitConverter.ToString(Encoding.Default.GetBytes(data)).Replace("-", "");
		Logger.Debug("Adding/updating domain item '{Type}'", type);
		string content = $"{{\"cmd\":\"add\",\"type\":{(int)type},\"data\":\"{arg}\"}}";
		return m_device.ExecuteWebRequest("domain", "", content).RequestState == EWebRequestState.VALID_RESPONSE;
	}
}
