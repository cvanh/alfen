using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.ComponentModel;
using System.Linq;
using System.Xml.Linq;

namespace ICUSettings;

public class ICUGroup : ICUBaseObject
{
	private string m_sName;

	private string m_sComment;

	private string m_sHTTPUser = string.Empty;

	private string m_sHTTPPassword = string.Empty;

	private readonly ObservableCollection<ICUFeatureRight> m_lstFeatures = new ObservableCollection<ICUFeatureRight>();

	public ICUGroup OriginalValues { get; set; }

	public int NumberOfUsers { get; set; }

	public string Name
	{
		get
		{
			return m_sName;
		}
		set
		{
			m_sName = value.Trim();
			FireChangedEvent("Name");
		}
	}

	public string Comment
	{
		get
		{
			return m_sComment;
		}
		set
		{
			m_sComment = value.Trim();
			FireChangedEvent("Comment");
		}
	}

	public string HTTPUser
	{
		get
		{
			return m_sHTTPUser;
		}
		set
		{
			m_sHTTPUser = value.Trim();
			FireChangedEvent("HTTPUser");
		}
	}

	public string HTTPPassword
	{
		get
		{
			return m_sHTTPPassword;
		}
		set
		{
			m_sHTTPPassword = value.Trim();
			FireChangedEvent("HTTPPassword");
		}
	}

	public ObservableCollection<ICUFeatureRight> Features
	{
		get
		{
			return m_lstFeatures;
		}
		set
		{
			FireChangedEvent("Features");
		}
	}

	public XElement Element
	{
		get
		{
			XElement xElement = new XElement("Group", new XAttribute("Name", Name), new XAttribute("Comment", Comment), new XAttribute("HTTPUser", HTTPUser), new XAttribute("HTTPPassword", HTTPPassword));
			foreach (ICUFeatureRight lstFeature in m_lstFeatures)
			{
				if (lstFeature.Rights != lstFeature.Feature.Default)
				{
					xElement.Add(new XElement("FeatureRight", new XAttribute("ID", lstFeature.Feature.ID), new XAttribute("Rights", lstFeature.Rights.ToString())));
				}
			}
			return xElement;
		}
	}

	public ICUGroup(IList<ICUFeature> lstFeatures)
		: base(fDirty: true)
	{
		m_sName = "";
		m_sComment = "";
		m_sHTTPUser = "";
		m_sHTTPPassword = "";
		InitializeFeatures(lstFeatures);
	}

	public ICUGroup(IList<ICUFeature> lstFeatures, string name, string comment = "", string httpuser = "", string httppassword = "")
		: base(fDirty: true)
	{
		m_sName = name;
		m_sComment = comment;
		m_sHTTPUser = httpuser;
		m_sHTTPPassword = httppassword;
		InitializeFeatures(lstFeatures);
	}

	public ICUGroup(IList<ICUFeature> lstFeatures, XElement xelem)
	{
		InitializeFeatures(lstFeatures);
		m_sName = getAttribute(xelem, "Name").Trim();
		m_sComment = getAttribute(xelem, "Comment").Trim();
		m_sHTTPUser = getAttribute(xelem, "HTTPUser").Trim();
		m_sHTTPPassword = getAttribute(xelem, "HTTPPassword").Trim();
		OriginalValues = new ICUGroup(Name, Comment, HTTPUser, HTTPPassword);
	}

	public ICUGroup(IList<ICUFeature> lstFeatures, dynamic obj)
	{
		InitializeFeatures(lstFeatures);
		m_sName = obj["Name"].Trim();
		m_sComment = obj["Comment"].Trim().Replace("<br>", "\n");
		if (((IDictionary<string, object>)obj).ContainsKey("HTTPUser"))
		{
			m_sHTTPUser = obj["HTTPUser"].Trim();
		}
		if (((IDictionary<string, object>)obj).ContainsKey("HTTPPassword"))
		{
			m_sHTTPPassword = obj["HTTPPassword"].Trim();
		}
		if (((IDictionary<string, object>)obj).ContainsKey("FeatureRights"))
		{
			foreach (dynamic fr in obj["FeatureRights"])
			{
				ICUFeatureRight iCUFeatureRight = m_lstFeatures.FirstOrDefault((ICUFeatureRight a) => a.Feature.ID == fr["ID"]);
				if (iCUFeatureRight != null)
				{
					iCUFeatureRight.Rights = (ICURights)Enum.Parse(typeof(ICURights), fr["Rights"].Trim());
					iCUFeatureRight.Commit();
				}
			}
		}
		OriginalValues = new ICUGroup(Name, Comment, HTTPUser, HTTPPassword);
		CheckDirty();
	}

	private ICUGroup(string name, string comment, string httpuser, string httppassword)
	{
		m_sName = name.Trim();
		m_sComment = comment.Trim();
		m_sHTTPUser = httpuser.Trim();
		m_sHTTPPassword = httppassword.Trim();
		OriginalValues = null;
	}

	private void InitializeFeatures(IList<ICUFeature> lstFeatures)
	{
		m_lstFeatures.Clear();
		foreach (ICUFeature lstFeature in lstFeatures)
		{
			ICUFeatureRight iCUFeatureRight = null;
			iCUFeatureRight = ((Name == null || !(Name.ToLowerInvariant() == "admin")) ? new ICUFeatureRight(lstFeature, lstFeature.Default) : new ICUFeatureRight(lstFeature, ICURights.Full));
			if (iCUFeatureRight != null)
			{
				iCUFeatureRight.PropertyChanged += OnFeatureRightChanged;
				m_lstFeatures.Add(iCUFeatureRight);
			}
		}
	}

	private void OnFeatureRightChanged(object sender, PropertyChangedEventArgs e)
	{
		FireChangedEvent("Features");
	}

	protected override bool OnCheckDirty()
	{
		if (OriginalValues == null)
		{
			return true;
		}
		bool flag = m_lstFeatures.Any((ICUFeatureRight a) => a.Dirty);
		return (m_sName != OriginalValues.Name || m_sComment != OriginalValues.Comment || m_sHTTPUser != OriginalValues.HTTPUser || m_sHTTPPassword != OriginalValues.HTTPPassword) | flag;
	}

	public void CopyFrom(ICUGroup other)
	{
		if (other != null)
		{
			Name = other.Name;
			Comment = other.Comment;
			HTTPUser = other.HTTPUser;
			HTTPPassword = other.HTTPPassword;
			CheckDirty();
		}
	}

	public string Json(bool IncludeCredentials)
	{
		List<string> list = new List<string>();
		foreach (ICUFeatureRight lstFeature in m_lstFeatures)
		{
			if (lstFeature.Rights != lstFeature.Feature.Default)
			{
				list.Add($"\n\t\t{{\"ID\":{ValidString(lstFeature.Feature.ID)},\"Rights\":{ValidString(lstFeature.Rights.ToString())}}}");
			}
		}
		string text = string.Join(",", list);
		return string.Format("\n\t{{\"Name\":{0},\"Comment\":{1},\"HTTPUser\":{2},\"HTTPPassword\":{3},\"FeatureRights\":[{4}]}}", new object[5]
		{
			ValidString(Name),
			ValidString(Comment),
			ValidString(IncludeCredentials ? HTTPUser : string.Empty),
			ValidString(IncludeCredentials ? HTTPPassword : string.Empty),
			text
		});
	}

	public void Commit()
	{
		if (OriginalValues != null)
		{
			OriginalValues.CopyFrom(this);
		}
		foreach (ICUFeatureRight lstFeature in m_lstFeatures)
		{
			lstFeature.Commit();
		}
		CheckDirty();
	}

	public void Rollback()
	{
		CopyFrom(OriginalValues);
		foreach (ICUFeatureRight lstFeature in m_lstFeatures)
		{
			lstFeature.Rollback();
		}
		CheckDirty();
	}

	public ICURights GetRights(string id)
	{
		return m_lstFeatures.FirstOrDefault((ICUFeatureRight a) => a.Feature.ID == id)?.Rights ?? ICURights.Full;
	}
}
