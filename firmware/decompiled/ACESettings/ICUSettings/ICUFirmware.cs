using System.Xml.Linq;

namespace ICUSettings;

public class ICUFirmware : ICUBaseObject
{
	private string m_sFilename;

	private string m_sVersion;

	private string m_sComments;

	private string m_sDate;

	private bool m_fFoundOnFtp;

	public ICUFirmware OriginalValues { get; set; }

	public string Filename
	{
		get
		{
			return m_sFilename;
		}
		set
		{
			m_sFilename = value.Trim();
			FireChangedEvent("Filename");
		}
	}

	public string Version
	{
		get
		{
			return m_sVersion;
		}
		set
		{
			m_sVersion = value.Trim();
			FireChangedEvent("Version");
		}
	}

	public string Comments
	{
		get
		{
			return m_sComments;
		}
		set
		{
			m_sComments = value;
			FireChangedEvent("Comments");
		}
	}

	public string Date
	{
		get
		{
			return m_sDate;
		}
		set
		{
			m_sDate = value;
			FireChangedEvent("Date");
		}
	}

	public bool OnFTP
	{
		get
		{
			return m_fFoundOnFtp;
		}
		set
		{
			m_fFoundOnFtp = value;
			FireChangedEvent("OnFTP");
		}
	}

	public XElement Element => new XElement("Firmware", new XAttribute("Filename", Filename), new XAttribute("Version", Version), new XAttribute("Date", Date), new XAttribute("Comments", Comments));

	public string Json => string.Format("\n\t{{\"Filename\":{0},\"Version\":{1},\"Date\":{2},\"Comments\":{3}}}", new object[4]
	{
		ValidString(Filename),
		ValidString(Version),
		ValidString(Date.ToString()),
		ValidString(Comments)
	});

	public ICUFirmware()
		: base(fDirty: true)
	{
		m_sFilename = "";
		m_sVersion = "0.0.0";
		m_sComments = "";
		m_sDate = "";
		m_fFoundOnFtp = false;
	}

	public ICUFirmware(XElement xelem)
	{
		m_sFilename = getAttribute(xelem, "Filename").Trim();
		m_sVersion = getAttribute(xelem, "Version").Trim();
		m_sComments = getAttribute(xelem, "Comments").Trim();
		m_sDate = getAttribute(xelem, "Date").Trim();
		OriginalValues = new ICUFirmware(m_sFilename, m_sVersion, m_sComments, m_sDate);
		m_fFoundOnFtp = false;
	}

	public ICUFirmware(dynamic obj)
	{
		m_sFilename = obj["Filename"].Trim();
		m_sVersion = obj["Version"].Trim();
		m_sComments = obj["Comments"].Trim().Replace("<br>", "\n");
		m_sDate = obj["Date"].Trim();
		OriginalValues = new ICUFirmware(m_sFilename, m_sVersion, m_sComments, m_sDate);
		m_fFoundOnFtp = false;
	}

	public ICUFirmware(string fileName, string version, string comments, string date)
	{
		m_sFilename = fileName.Trim();
		m_sVersion = version.Trim();
		m_sComments = comments.Trim();
		m_sDate = date.Trim();
		m_fFoundOnFtp = false;
		OriginalValues = null;
	}

	protected override bool OnCheckDirty()
	{
		if (OriginalValues == null)
		{
			return true;
		}
		if (!(m_sFilename != OriginalValues.Filename) && !(m_sVersion != OriginalValues.Version) && !(m_sComments != OriginalValues.Comments))
		{
			return m_sDate != OriginalValues.Date;
		}
		return true;
	}

	public void CopyFrom(ICUFirmware other)
	{
		if (other != null)
		{
			Filename = other.Filename;
			Version = other.Version;
			Comments = other.Comments;
			Date = other.Date;
			CheckDirty();
		}
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
}
