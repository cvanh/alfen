using System.Collections.Generic;
using System.Collections.ObjectModel;

namespace ICUIWSConnection;

public class IWSObject
{
	public int Version { get; set; }

	public string ObjectId { get; set; }

	public string PartCode { get; set; }

	public string Description { get; set; }

	public string OrderHeader { get; set; }

	public string OrderNumber { get; set; }

	public string Backoffice { get; set; }

	public string BackofficeSettingsVersion { get; set; }

	public bool IsEichrechtOrder { get; set; }

	public string PublicKeyBaseURL { get; set; }

	public string Language { get; set; }

	public bool IsLanguageDefined => !string.IsNullOrEmpty(Language);

	public int LoadBalancing { get; set; }

	public bool IsPersonalizedDisplay { get; set; }

	public string LogoFileName { get; set; }

	public string Logo { get; set; }

	public bool IsPartManagementEnabled { get; set; }

	public string FeaturesUnlockedText { get; set; }

	public bool IsThreePhases { get; set; }

	public bool IsSimNeeded { get; set; }

	public bool IsSSAEnabled { get; set; }

	public CSCommunication Communication { get; set; }

	public CSInterface Interface { get; set; }

	public List<IWSPropertyValue> Properties { get; set; }

	public Collection<Article> ArticleCollection { get; set; }

	public string InternalInfo { get; set; }
}
