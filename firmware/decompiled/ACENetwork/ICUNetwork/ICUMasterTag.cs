namespace ICUNetwork;

public class ICUMasterTag
{
	protected ICUDevice m_device;

	private bool m_masterTagEnabled;

	private string m_masterTagId = string.Empty;

	public string Tag
	{
		get
		{
			m_masterTagId = "";
			if (m_device == null)
			{
				return m_masterTagId;
			}
			if (IsFeatureSupported())
			{
				ICUProperty property = m_device.GetProperty(9216, 2);
				if (property != null)
				{
					m_device.UpdateProperties(null, property);
					m_masterTagId = m_device.GetPropertyString(9216, 2, 0);
				}
			}
			return m_masterTagId;
		}
	}

	public bool Enabled
	{
		get
		{
			if (m_device == null)
			{
				return false;
			}
			m_masterTagEnabled = false;
			if (IsFeatureSupported())
			{
				ICUProperty property = m_device.GetProperty(9216, 1);
				if (property != null)
				{
					m_device.UpdateProperties(null, property);
					m_masterTagEnabled = m_device.GetPropertyBool(9216, 1);
				}
			}
			return m_masterTagEnabled;
		}
	}

	public ICUMasterTag(ICUDevice device)
	{
		ReInitialize(device);
	}

	public void ReInitialize(ICUDevice device)
	{
		m_device = device;
	}

	public bool Set(string tag)
	{
		if (m_device == null)
		{
			return false;
		}
		ICUProperty property = m_device.GetProperty(9216, 2);
		if (property != null)
		{
			property.Value = tag;
			return m_device.StoreProperties(property);
		}
		return false;
	}

	public bool Clear()
	{
		return Set("");
	}

	public bool IsFeatureSupported()
	{
		if (m_device == null)
		{
			return false;
		}
		if (m_device.GetProperty(9216, 1) != null)
		{
			return m_device.GetProperty(9216, 2) != null;
		}
		return false;
	}
}
