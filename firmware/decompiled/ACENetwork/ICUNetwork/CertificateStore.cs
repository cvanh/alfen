using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Reflection;
using System.Security.Cryptography.X509Certificates;
using System.Text;
using Serilog;

namespace ICUNetwork;

public static class CertificateStore
{
	public enum CertificateType
	{
		ACE_CA_ROOT
	}

	private static readonly List<CertificateMap> certificateMapping = new List<CertificateMap>
	{
		new CertificateMap(CertificateType.ACE_CA_ROOT, "ICUNetwork.Certificates.webserverrootcert.pem")
	};

	public static X509Certificate2 GetCertificate(CertificateType type)
	{
		X509Certificate2 result = null;
		try
		{
			string text = certificateMapping.FirstOrDefault((CertificateMap a) => a.type == type)?.certLocation;
			if (!string.IsNullOrEmpty(text))
			{
				using StreamReader streamReader = new StreamReader(Assembly.GetExecutingAssembly().GetManifestResourceStream(text));
				result = new X509Certificate2(Encoding.ASCII.GetBytes(streamReader.ReadToEnd()));
			}
		}
		catch (Exception exception)
		{
			Log.Logger.Debug(exception, "Failed to retreive certificate");
		}
		return result;
	}
}
