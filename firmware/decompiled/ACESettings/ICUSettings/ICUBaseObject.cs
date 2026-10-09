using System.Collections.Generic;
using System.ComponentModel;
using System.Runtime.CompilerServices;
using System.Text.RegularExpressions;
using System.Xml.Linq;

namespace ICUSettings;

public class ICUBaseObject : INotifyPropertyChanged
{
	private bool m_fDirty;

	public bool Dirty
	{
		get
		{
			return m_fDirty;
		}
		set
		{
			m_fDirty = value;
		}
	}

	public event PropertyChangedEventHandler PropertyChanged;

	public ICUBaseObject(bool fDirty = false)
	{
		m_fDirty = fDirty;
	}

	protected virtual bool OnCheckDirty()
	{
		return false;
	}

	public void CheckDirty()
	{
		m_fDirty = OnCheckDirty();
		PropertyChanged?.Invoke(this, new PropertyChangedEventArgs("Dirty"));
	}

	protected void FireChangedEvent(string propertyName)
	{
		PropertyChanged?.Invoke(this, new PropertyChangedEventArgs(propertyName));
		CheckDirty();
	}

	protected void SetPropertyField<T>(ref T field, T newValue, [CallerMemberName] string caller = null)
	{
		if (!EqualityComparer<T>.Default.Equals(field, newValue))
		{
			field = newValue;
			FireChangedEvent(caller);
		}
	}

	protected string getAttribute(XElement xelem, string attrName)
	{
		XAttribute xAttribute = xelem.Attribute(attrName);
		if (xAttribute != null)
		{
			return xAttribute.Value.ToString();
		}
		return string.Empty;
	}

	protected string setAttribute(XElement xelem, string attrName, string newValue)
	{
		XAttribute xAttribute = xelem.Attribute(attrName);
		if (xAttribute != null)
		{
			return xAttribute.Value = newValue;
		}
		return string.Empty;
	}

	protected string ValidString(string value)
	{
		value = value ?? string.Empty;
		return string.Format("\"{0}\"", Regex.Replace(value, "\\r\\n?|\\n", "<br>"));
	}
}
