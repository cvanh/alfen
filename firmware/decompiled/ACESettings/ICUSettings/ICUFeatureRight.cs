namespace ICUSettings;

public class ICUFeatureRight : ICUBaseObject
{
	private ICURights m_eCurrentRights;

	private ICURights m_eOriginalRights;

	public ICUFeature Feature { get; }

	public ICURights Rights
	{
		get
		{
			return m_eCurrentRights;
		}
		set
		{
			m_eCurrentRights = value;
			FireChangedEvent("Rights");
		}
	}

	public ICUFeatureRight(ICUFeature feature, ICURights rights)
	{
		Feature = feature;
		m_eCurrentRights = rights;
		m_eOriginalRights = rights;
	}

	protected override bool OnCheckDirty()
	{
		return m_eOriginalRights != Rights;
	}

	public void Commit()
	{
		m_eOriginalRights = m_eCurrentRights;
		CheckDirty();
		FireChangedEvent("Rights");
	}

	public void Rollback()
	{
		m_eCurrentRights = m_eOriginalRights;
		FireChangedEvent("Rights");
	}
}
