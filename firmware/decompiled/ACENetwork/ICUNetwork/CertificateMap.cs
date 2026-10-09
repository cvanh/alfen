namespace ICUNetwork;

public class CertificateMap
{
	public CertificateStore.CertificateType type;

	public string certLocation;

	public CertificateMap(CertificateStore.CertificateType type, string location)
	{
		this.type = type;
		certLocation = location;
	}
}
