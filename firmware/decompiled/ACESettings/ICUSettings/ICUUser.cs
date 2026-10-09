using System;
using System.Collections.Generic;
using System.Linq;
using System.Security.Cryptography;
using System.Text;
using System.Xml;
using System.Xml.Linq;

namespace ICUSettings;

public class ICUUser : ICUBaseObject
{
	public static int s_passwordLength = 6;

	private string m_sUser;

	private string m_sPassword;

	private string m_sFullname;

	private ICUGroup m_oGroup;

	private string m_sCompany;

	private string m_sComment = string.Empty;

	public ICUUser OriginalValues { get; set; }

	public string User
	{
		get
		{
			return m_sUser;
		}
		set
		{
			m_sUser = value.Trim();
			FireChangedEvent("User");
		}
	}

	public string Password
	{
		get
		{
			return m_sPassword;
		}
		set
		{
			m_sPassword = value.Trim();
			FireChangedEvent("Password");
		}
	}

	public ICUGroup Group
	{
		get
		{
			return m_oGroup;
		}
		set
		{
			m_oGroup = value;
			FireChangedEvent("Group");
		}
	}

	public string Fullname
	{
		get
		{
			return m_sFullname;
		}
		set
		{
			m_sFullname = value.Trim();
			FireChangedEvent("Fullname");
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

	public string Company
	{
		get
		{
			return m_sCompany;
		}
		set
		{
			m_sCompany = value.Trim();
			FireChangedEvent("Company");
		}
	}

	public XElement Element
	{
		get
		{
			string value = ((Group != null) ? Group.Name : "");
			return new XElement("User", new XAttribute("User", User), new XAttribute("Password", Password), new XAttribute("Group", value), new XAttribute("Fullname", Fullname), new XAttribute("Comment", Comment), new XAttribute("Company", Company));
		}
	}

	public string Json
	{
		get
		{
			string value = ((Group != null) ? Group.Name : "");
			return string.Format("\n\t{{\"User\":{0},\"Password\":{1},\"Group\":{2},\"Fullname\":{3},\"Comment\":{4},\"Company\":{5}}}", new object[6]
			{
				ValidString(User),
				ValidString(Password),
				ValidString(value),
				ValidString(Fullname),
				ValidString(Comment),
				ValidString(Company)
			});
		}
	}

	public string CSVLine => $"\"{Fullname}\", \"{User}\", \"{Password}\"";

	public ICUUser()
		: base(fDirty: true)
	{
		m_oGroup = null;
		m_sPassword = CreatePassword(s_passwordLength);
		m_sComment = "";
	}

	public ICUUser(List<ICUGroup> allGroups, XElement xelem)
	{
		m_sUser = getAttribute(xelem, "User").Trim();
		m_sPassword = getAttribute(xelem, "Password").Trim();
		string rightsName = getAttribute(xelem, "Rights").Trim();
		m_sCompany = getAttribute(xelem, "Group").Trim();
		m_oGroup = allGroups.FirstOrDefault((ICUGroup a) => a.Name == rightsName);
		if (m_oGroup == null)
		{
			m_oGroup = allGroups.FirstOrDefault((ICUGroup a) => a.Name == $"Extern_{m_sCompany}");
			if (m_oGroup == null)
			{
				m_oGroup = allGroups.FirstOrDefault((ICUGroup a) => a.Name == "Extern_ICU");
			}
		}
		m_sFullname = getAttribute(xelem, "Fullname").Trim();
		if (xelem.NextNode != null && xelem.NextNode.NodeType == XmlNodeType.Comment)
		{
			m_sComment = ((XComment)xelem.NextNode).Value.Trim();
		}
		OriginalValues = new ICUUser(User, Password, Group, Fullname, Comment, Company);
	}

	public ICUUser(List<ICUGroup> allGroups, dynamic obj)
	{
		m_sUser = obj["User"].Trim();
		m_sPassword = obj["Password"].Trim();
		m_sFullname = obj["Fullname"].Trim();
		string groupName = obj["Group"].Trim();
		if (((IDictionary<string, object>)obj).ContainsKey("Company"))
		{
			m_sCompany = obj["Company"].Trim();
		}
		else
		{
			m_sCompany = groupName;
		}
		m_oGroup = allGroups.FirstOrDefault((ICUGroup a) => a.Name == groupName);
		if (m_oGroup == null)
		{
			m_oGroup = allGroups.FirstOrDefault((ICUGroup a) => a.Name == $"Extern_{m_sCompany}");
			if (m_oGroup == null)
			{
				m_oGroup = allGroups.FirstOrDefault((ICUGroup a) => a.Name == "Extern_ICU");
			}
		}
		if (((IDictionary<string, object>)obj).ContainsKey("Comment"))
		{
			m_sComment = obj["Comment"].Trim().Replace("<br>", "\n");
		}
		OriginalValues = new ICUUser(User, Password, Group, Fullname, Comment, Company);
	}

	private ICUUser(string name, string password, ICUGroup group, string fullName, string comment, string company)
	{
		m_sUser = name.Trim();
		m_sPassword = password.Trim();
		m_oGroup = group;
		m_sFullname = fullName.Trim();
		m_sComment = comment.Trim();
		m_sCompany = company.Trim();
		OriginalValues = null;
	}

	public static ICUUser CreateHashedUser(SHA256 sha256Hasher, RNGCryptoServiceProvider rng, string username, string password, ICUGroup group, string isahData, string httpData)
	{
		ICUUser iCUUser = new ICUUser();
		byte[] array = new byte[4];
		byte[] array2 = new byte[1];
		rng.GetBytes(array);
		rng.GetNonZeroBytes(array2);
		int num = array2[0];
		iCUUser.User = Convert.ToBase64String(sha256Hasher.ComputeHash(Encoding.UTF8.GetBytes(username.ToLowerInvariant())));
		string text = BitConverter.ToString(array).Replace("-", "");
		string text2 = text + password;
		byte[] array3 = Encoding.UTF8.GetBytes(text2);
		for (int i = 0; i < num; i++)
		{
			array3 = sha256Hasher.ComputeHash(array3);
		}
		string arg = Convert.ToBase64String(array3);
		iCUUser.Password = $"{num:X2}:{text}:{arg}";
		iCUUser.Group = group;
		iCUUser.Fullname = string.Empty;
		iCUUser.Comment = new EncryptDecrypt().Encrypt(httpData, text2);
		iCUUser.Company = new EncryptDecrypt().Encrypt(isahData, text2);
		return iCUUser;
	}

	public bool ValidateHashedPassword(SHA256 sha256Hash, string otherPassword)
	{
		string[] array = Password.Split(new char[1] { ':' });
		if (array.Length == 3)
		{
			ushort num = Convert.ToUInt16(array[0], 16);
			string s = array[1] + otherPassword;
			byte[] array2 = Encoding.UTF8.GetBytes(s);
			for (int i = 0; i < num; i++)
			{
				array2 = sha256Hash.ComputeHash(array2);
			}
			return array[2] == Convert.ToBase64String(array2);
		}
		return false;
	}

	public static string CreatePassword(int length)
	{
		StringBuilder stringBuilder = new StringBuilder();
		Random random = new Random();
		while (0 < length--)
		{
			stringBuilder.Append("aeiouyaeiouyaeiouybcdfghjklmnpqrstvwxz"[random.Next("aeiouyaeiouyaeiouybcdfghjklmnpqrstvwxz".Length)]);
		}
		stringBuilder[random.Next(stringBuilder.Length)] = "1234567890"[random.Next("1234567890".Length)];
		return stringBuilder.ToString();
	}

	protected override bool OnCheckDirty()
	{
		if (OriginalValues == null)
		{
			return true;
		}
		if (!(m_sUser != OriginalValues.User) && !(m_sPassword != OriginalValues.Password) && !(m_sFullname != OriginalValues.Fullname) && m_oGroup == OriginalValues.Group && !(m_sComment != OriginalValues.Comment))
		{
			return m_sCompany != OriginalValues.Company;
		}
		return true;
	}

	public void CopyFrom(ICUUser other)
	{
		if (other != null)
		{
			User = other.User;
			Password = other.Password;
			if (!string.IsNullOrEmpty(other.Fullname.Trim()))
			{
				Fullname = other.Fullname;
			}
			Group = other.Group;
			if (!string.IsNullOrEmpty(other.Comment.Trim()))
			{
				Comment = other.Comment;
			}
			if (!string.IsNullOrEmpty(other.Company.Trim()))
			{
				Company = other.Company;
			}
			CheckDirty();
		}
	}

	public void AddOldElement(XElement xuser)
	{
		string value = ((Group != null) ? Group.Name : "");
		XElement content = new XElement("Field", new XAttribute("User", User), new XAttribute("Password", Password), new XAttribute("Group", value), new XAttribute("Fullname", Fullname));
		xuser.Add(content);
		xuser.Add(new XComment(Comment));
	}

	public void Commit()
	{
		if (OriginalValues != null)
		{
			OriginalValues.CopyFrom(this);
		}
		CheckDirty();
	}

	public void Rollback()
	{
		CopyFrom(OriginalValues);
	}

	public ICURights GetRights(string id)
	{
		if (m_oGroup != null)
		{
			return m_oGroup.GetRights(id);
		}
		return ICURights.ReadOnly;
	}
}
