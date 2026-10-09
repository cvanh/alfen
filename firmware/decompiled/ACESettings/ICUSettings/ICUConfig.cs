using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.ComponentModel;
using System.IO;
using System.Linq;
using System.Net;
using System.Security.Cryptography;
using System.Text;
using System.Web.Script.Serialization;
using System.Xml.Linq;
using Serilog;

namespace ICUSettings;

public class ICUConfig : INotifyPropertyChanged
{
	private readonly ILogger Logger = Log.ForContext<ICUConfig>();

	protected static string s_fileID = "ICUConfigFile";

	protected string m_sVersion = string.Empty;

	protected string m_sDateTime = string.Empty;

	protected string m_sOriginalVersion = string.Empty;

	protected ObservableCollection<ICUFeature> m_lstFeatures = new ObservableCollection<ICUFeature>();

	protected ObservableCollection<ICUGroup> m_lstGroups = new ObservableCollection<ICUGroup>();

	protected ObservableCollection<ICUUser> m_lstUsers = new ObservableCollection<ICUUser>();

	protected ObservableCollection<ICUBackOffice> m_lstBackOffices = new ObservableCollection<ICUBackOffice>();

	protected ObservableCollection<ICUPMBackOffice> m_lstPMBackOffices = new ObservableCollection<ICUPMBackOffice>();

	protected ObservableCollection<ICUFirmware> m_lstFirmwares = new ObservableCollection<ICUFirmware>();

	private bool m_fThisDirty;

	private bool m_fGroupCollectionDirty;

	private bool m_fUserCollectionDirty;

	private bool m_fBackofficeCollectionDirty;

	private bool m_fPMBackofficeCollectionDirty;

	private bool m_fFirmwareCollectionDirty;

	public ObservableCollection<ICUFeature> Features => m_lstFeatures;

	public ObservableCollection<ICUGroup> Groups => m_lstGroups;

	public ObservableCollection<ICUUser> Users => m_lstUsers;

	public ObservableCollection<ICUBackOffice> BackOffices => m_lstBackOffices;

	public ObservableCollection<ICUPMBackOffice> PMBackOffices => m_lstPMBackOffices;

	public ObservableCollection<ICUFirmware> Firmwares => m_lstFirmwares;

	public string Version
	{
		get
		{
			return m_sVersion;
		}
		set
		{
			m_sVersion = value;
			m_fThisDirty = m_sVersion != m_sOriginalVersion;
			PropertyChanged(this, new PropertyChangedEventArgs("Version"));
		}
	}

	public string Date => m_sDateTime;

	public bool IsGroupsDirty
	{
		get
		{
			if (m_lstGroups == null)
			{
				return false;
			}
			if (!m_fGroupCollectionDirty)
			{
				return m_lstGroups.FirstOrDefault((ICUGroup a) => a.Dirty) != null;
			}
			return true;
		}
	}

	public bool IsUsersDirty
	{
		get
		{
			if (m_lstUsers == null)
			{
				return false;
			}
			if (!m_fUserCollectionDirty)
			{
				return m_lstUsers.FirstOrDefault((ICUUser a) => a.Dirty) != null;
			}
			return true;
		}
	}

	public bool IsBackofficesDirty
	{
		get
		{
			if (m_lstBackOffices == null)
			{
				return false;
			}
			if (!m_fBackofficeCollectionDirty)
			{
				return m_lstBackOffices.FirstOrDefault((ICUBackOffice a) => a.Dirty) != null;
			}
			return true;
		}
	}

	public bool IsPMBackofficesDirty
	{
		get
		{
			if (m_lstPMBackOffices == null)
			{
				return false;
			}
			if (!m_fPMBackofficeCollectionDirty)
			{
				return m_lstPMBackOffices.FirstOrDefault((ICUPMBackOffice a) => a.Dirty) != null;
			}
			return true;
		}
	}

	public bool IsFirmwareDirty
	{
		get
		{
			if (m_lstFirmwares == null)
			{
				return false;
			}
			if (!m_fFirmwareCollectionDirty)
			{
				return m_lstFirmwares.FirstOrDefault((ICUFirmware a) => a.Dirty) != null;
			}
			return true;
		}
	}

	public bool IsDirty
	{
		get
		{
			if (!m_fThisDirty && !IsGroupsDirty && !IsUsersDirty && !IsBackofficesDirty && !IsPMBackofficesDirty)
			{
				return IsFirmwareDirty;
			}
			return true;
		}
	}

	public event PropertyChangedEventHandler PropertyChanged;

	public bool UpdateUsersFromConfig(string configFilename)
	{
		try
		{
			XDocument xDocument = XDocument.Parse(File.ReadAllText(configFilename));
			List<XElement> list = xDocument.Descendants("Field").ToList();
			if (list.Count == 0)
			{
				list = xDocument.Descendants("User").ToList();
			}
			List<ICUUser> list2 = new List<ICUUser>();
			foreach (XElement item in list)
			{
				ICUUser iCUUser = new ICUUser(m_lstGroups.ToList(), item);
				iCUUser.PropertyChanged += OnUserChanged;
				list2.Add(iCUUser);
			}
			foreach (ICUUser newUser in list2)
			{
				ICUUser iCUUser2 = m_lstUsers.FirstOrDefault((ICUUser a) => a.User.ToLowerInvariant() == newUser.User.ToLowerInvariant());
				if (iCUUser2 == null)
				{
					newUser.Dirty = true;
					m_lstUsers.Add(newUser);
				}
				else
				{
					iCUUser2.CopyFrom(newUser);
				}
			}
			return true;
		}
		catch (Exception ex)
		{
			Logger.Error(ex, ex.Message);
			return false;
		}
	}

	private void OnFeatureChanged(object sender, PropertyChangedEventArgs e)
	{
		PropertyChanged(this, new PropertyChangedEventArgs("Features"));
	}

	private void OnGroupChanged(object sender, PropertyChangedEventArgs e)
	{
		PropertyChanged(this, new PropertyChangedEventArgs("Groups"));
	}

	private void OnUserChanged(object sender, PropertyChangedEventArgs e)
	{
		PropertyChanged(this, new PropertyChangedEventArgs("Users"));
	}

	private void OnBackOfficeChanged(object sender, PropertyChangedEventArgs e)
	{
		PropertyChanged(this, new PropertyChangedEventArgs("BackOffices"));
		AddBackofficesToFeatures();
	}

	private void OnFirmwareChanged(object sender, PropertyChangedEventArgs e)
	{
		PropertyChanged(this, new PropertyChangedEventArgs("Firmwares"));
	}

	public bool WriteUsersXML(string filename = "Users2.xml", bool saveInBase64 = true)
	{
		try
		{
			XDocument xDocument = new XDocument(new XElement("Config", new XAttribute("Version", Version), new XElement("Users", m_lstUsers.Select((ICUUser x) => x.Element))));
			StringBuilder stringBuilder = new StringBuilder();
			using (TextWriter textWriter = new StringWriter(stringBuilder))
			{
				xDocument.Save(textWriter);
			}
			if (saveInBase64)
			{
				stringBuilder = new StringBuilder(Convert.ToBase64String(Encoding.UTF8.GetBytes(stringBuilder.ToString())));
			}
			using (StreamWriter streamWriter = new StreamWriter(filename))
			{
				streamWriter.Write((object?)stringBuilder);
			}
			return true;
		}
		catch (Exception ex)
		{
			Logger.Error(ex, ex.Message);
		}
		return false;
	}

	public bool WritePMBackOfficeSettingsXML(string directory)
	{
		string empty = string.Empty;
		try
		{
			foreach (ICUPMBackOffice lstPMBackOffice in m_lstPMBackOffices)
			{
				XDocument xDocument = new XDocument(new XElement("Config", new XElement("Setting", new XElement("Product", new XAttribute("Model", "NG9xx"), new XAttribute("Device", "NG9xx"), from x in lstPMBackOffice.m_lstProperties
					where x.Element != null
					select x.Element))));
				empty = Path.Combine(directory, lstPMBackOffice.Title.Trim().Replace(' ', '-') + ".xml");
				xDocument.Save(empty);
				string[] source = File.ReadAllLines(empty);
				File.WriteAllLines(empty, source.Skip(1).ToArray());
			}
			return true;
		}
		catch (Exception ex)
		{
			Logger.Error(ex, ex.Message);
		}
		return false;
	}

	public bool ReadInstallerSettings(string filename = "InstallerSettings.dat", ICUEncryptionType encType = ICUEncryptionType.encryptBase64)
	{
		//IL_003d: Unknown result type (might be due to invalid IL or missing references)
		//IL_0042: Unknown result type (might be due to invalid IL or missing references)
		try
		{
			string text = File.ReadAllText(filename);
			switch (encType)
			{
			case ICUEncryptionType.encryptBase64:
			{
				byte[] array = Convert.FromBase64String(text);
				text = Encoding.UTF8.GetString(array, 0, array.Length);
				break;
			}
			case ICUEncryptionType.encryptRijndael:
			case ICUEncryptionType.encryptRijndaelHashed:
				text = new EncryptDecrypt().Decrypt(text, "Pas5pR@sE");
				break;
			}
			dynamic val = new JavaScriptSerializer
			{
				MaxJsonLength = 52428800
			}.DeserializeObject(text);
			if (val["Type"].ToString() != s_fileID)
			{
				throw new ArgumentException("Incorrect file type!");
			}
			m_sVersion = val["Version"].ToString();
			m_sOriginalVersion = m_sVersion;
			m_sDateTime = val["Date"].ToString();
			m_lstFeatures = new ObservableCollection<ICUFeature>();
			m_lstGroups = new ObservableCollection<ICUGroup>();
			m_lstUsers = new ObservableCollection<ICUUser>();
			m_lstBackOffices = new ObservableCollection<ICUBackOffice>();
			m_lstPMBackOffices = new ObservableCollection<ICUPMBackOffice>();
			m_lstFirmwares = new ObservableCollection<ICUFirmware>();
			foreach (dynamic item in val["Backoffices"])
			{
				ICUBackOffice iCUBackOffice = new ICUBackOffice(item);
				iCUBackOffice.PropertyChanged += OnBackOfficeChanged;
				m_lstBackOffices.Add(iCUBackOffice);
			}
			if (((IDictionary<string, object>)val).ContainsKey("PMBackOffices"))
			{
				foreach (dynamic item2 in val["PMBackOffices"])
				{
					ICUPMBackOffice iCUPMBackOffice = new ICUPMBackOffice(item2);
					iCUPMBackOffice.PropertyChanged += OnBackOfficeChanged;
					m_lstPMBackOffices.Add(iCUPMBackOffice);
				}
			}
			if (((IDictionary<string, object>)val).ContainsKey("Features"))
			{
				foreach (dynamic item3 in val["Features"])
				{
					ICUFeature iCUFeature = new ICUFeature(item3);
					iCUFeature.PropertyChanged += OnFeatureChanged;
					m_lstFeatures.Add(iCUFeature);
				}
			}
			AddManualFeatures();
			AddBackofficesToFeatures();
			if (((IDictionary<string, object>)val).ContainsKey("Groups"))
			{
				foreach (dynamic item4 in val["Groups"])
				{
					ICUGroup iCUGroup = new ICUGroup(m_lstFeatures, item4);
					iCUGroup.PropertyChanged += OnGroupChanged;
					m_lstGroups.Add(iCUGroup);
				}
			}
			if (m_lstGroups.Count == 0)
			{
				List<string> list = new List<string>();
				foreach (dynamic item5 in val["Users"])
				{
					if (((IDictionary<string, object>)item5).ContainsKey("Company"))
					{
						list.Add(item5["Company"].Trim());
					}
				}
				AddDefaultGroups(list);
			}
			foreach (dynamic item6 in val["Users"])
			{
				ICUUser iCUUser = new ICUUser(m_lstGroups.ToList(), item6);
				iCUUser.PropertyChanged += OnUserChanged;
				m_lstUsers.Add(iCUUser);
			}
			foreach (dynamic item7 in val["Firmwares"])
			{
				ICUFirmware iCUFirmware = new ICUFirmware(item7);
				iCUFirmware.PropertyChanged += OnFirmwareChanged;
				m_lstFirmwares.Add(iCUFirmware);
			}
			foreach (ICUUser lstUser in m_lstUsers)
			{
				if (lstUser.Group != null)
				{
					lstUser.Group.NumberOfUsers++;
				}
			}
			return true;
		}
		catch (Exception ex)
		{
			Logger.Error(ex, ex.Message);
		}
		return false;
	}

	public bool WriteInstallerSettings(string filename = "InstallerSettings.dat", ICUEncryptionType encType = ICUEncryptionType.encryptBase64, bool addBackoffices = true)
	{
		try
		{
			if (File.Exists(filename))
			{
				File.Copy(filename, Path.ChangeExtension(filename, "bak"), overwrite: true);
			}
			List<ICUFeature> source = Features.Where((ICUFeature a) => a.Type != ICUFeatureType.Backoffice).ToList();
			StringBuilder stringBuilder = new StringBuilder();
			string text = string.Join(",", source.Select((ICUFeature a) => a.Json).ToArray());
			string text2 = string.Join(",", Groups.Select((ICUGroup a) => a.Json(encType != ICUEncryptionType.encryptRijndaelHashed)).ToArray());
			string text3 = string.Join(",", Users.Select((ICUUser a) => a.Json).ToArray());
			if (encType == ICUEncryptionType.encryptRijndaelHashed)
			{
				ICUUser iCUUser = Users.FirstOrDefault((ICUUser a) => a.User.ToLowerInvariant() == "isah");
				List<ICUUser> list = new List<ICUUser>();
				using (SHA256 sha256Hasher = SHA256.Create())
				{
					RNGCryptoServiceProvider rng = new RNGCryptoServiceProvider();
					foreach (ICUUser user in Users)
					{
						if (user.User.ToLowerInvariant() != "isah")
						{
							string httpData = user.Group.HTTPUser + ":" + user.Group.HTTPPassword;
							string password = iCUUser.Password;
							list.Add(ICUUser.CreateHashedUser(sha256Hasher, rng, user.User, user.Password, user.Group, password, httpData));
						}
					}
				}
				list = list.OrderBy((ICUUser a) => a.Password).ToList();
				text3 = string.Join(",", list.Select((ICUUser a) => a.Json).ToArray());
			}
			string text4 = string.Join(",", Firmwares.Select((ICUFirmware a) => a.Json).ToArray());
			string text5 = string.Empty;
			string text6 = string.Empty;
			if (addBackoffices)
			{
				text5 = string.Join(",", BackOffices.Select((ICUBackOffice a) => a.Json).ToArray());
				text6 = string.Join(",", PMBackOffices.Select((ICUPMBackOffice a) => a.Json).ToArray());
			}
			stringBuilder.AppendFormat("{{\n\"Type\":\"{0}\",\n\"Version\":\"{1}\",\n\"Date\":\"{2}\",\n\"Features\":[{3}\n],\n\"Groups\":[{4}\n],\n\"Users\":[{5}\n],\n\"Backoffices\":[{6}\n],\n\"PMBackOffices\":[{7}\n],\n\"Firmwares\":[{8}\n]}}", new object[9]
			{
				s_fileID,
				Version,
				DateTime.Now.ToString(),
				text,
				text2,
				text3,
				text5,
				text6,
				text4
			});
			if (encType == ICUEncryptionType.encryptBase64)
			{
				stringBuilder = new StringBuilder(Convert.ToBase64String(Encoding.UTF8.GetBytes(stringBuilder.ToString())));
			}
			else if (encType == ICUEncryptionType.encryptRijndael || encType == ICUEncryptionType.encryptRijndaelHashed)
			{
				stringBuilder = new StringBuilder(new EncryptDecrypt().Encrypt(stringBuilder.ToString(), "Pas5pR@sE"));
			}
			using (StreamWriter streamWriter = new StreamWriter(filename))
			{
				streamWriter.Write((object?)stringBuilder);
			}
			return true;
		}
		catch (Exception ex)
		{
			Logger.Error(ex, ex.Message);
		}
		return false;
	}

	public bool UploadFileToFTP(string ftpSite, string fileName, string fullFilePath, string userName, string password)
	{
		try
		{
			Uri uri = new Uri(new Uri(ftpSite), fileName);
			FtpWebRequest ftpWebRequest = (FtpWebRequest)WebRequest.Create(uri);
			ftpWebRequest.Method = "STOR";
			ftpWebRequest.Credentials = new NetworkCredential(userName, password);
			StreamReader streamReader = new StreamReader(fullFilePath);
			byte[] bytes = Encoding.UTF8.GetBytes(streamReader.ReadToEnd());
			streamReader.Close();
			ftpWebRequest.ContentLength = bytes.Length;
			Stream requestStream = ftpWebRequest.GetRequestStream();
			requestStream.Write(bytes, 0, bytes.Length);
			requestStream.Close();
			FtpWebResponse ftpWebResponse = (FtpWebResponse)ftpWebRequest.GetResponse();
			Logger.Debug("Upload file complete to file {Uri}, status {Description}", uri.ToString(), ftpWebResponse.StatusDescription);
			ftpWebResponse.Close();
		}
		catch (Exception exception)
		{
			Logger.Error(exception, "Upload failure");
			return false;
		}
		return true;
	}

	public bool UpdateFTPFirmwareList(string ftpSite, string ftpUserName, string ftpPassword, int timeout = 1000)
	{
		try
		{
			Uri uri = new Uri(new Uri(ftpSite), "Firmware");
			FtpWebRequest ftpWebRequest = (FtpWebRequest)WebRequest.Create(uri);
			ftpWebRequest.Timeout = timeout;
			ftpWebRequest.Method = "NLST";
			ftpWebRequest.Credentials = new NetworkCredential(ftpUserName, ftpPassword);
			FtpWebResponse ftpWebResponse = (FtpWebResponse)ftpWebRequest.GetResponse();
			string text = new StreamReader(ftpWebResponse.GetResponseStream()).ReadToEnd();
			ftpWebResponse.Close();
			string[] array = (from a in text.Split(new char[1] { '\n' })
				select a.Trim(new char[3] { ' ', '\r', '\n' }) into a
				where !string.IsNullOrEmpty(a)
				select a).ToArray();
			foreach (string fiFile in array)
			{
				ICUFirmware iCUFirmware = m_lstFirmwares.FirstOrDefault((ICUFirmware a) => a.Filename.ToLowerInvariant() == fiFile.ToLowerInvariant());
				if (iCUFirmware == null)
				{
					DateTime dateTime = DateTime.Now;
					try
					{
						FtpWebRequest ftpWebRequest2 = (FtpWebRequest)WebRequest.Create(new Uri(uri, fiFile));
						ftpWebRequest2.Timeout = timeout;
						ftpWebRequest2.Method = "MDTM";
						ftpWebRequest2.Credentials = new NetworkCredential(ftpUserName, ftpPassword);
						FtpWebResponse ftpWebResponse2 = (FtpWebResponse)ftpWebRequest2.GetResponse();
						dateTime = ftpWebResponse.LastModified;
						ftpWebResponse2.Close();
					}
					catch (Exception ex)
					{
						Logger.Error(ex, "Error while requesting the filedate for {File}: {Message}", fiFile, ex.Message);
					}
					ICUFirmware iCUFirmware2 = new ICUFirmware(fiFile, "0.0.0", "", dateTime.ToString());
					iCUFirmware2.OnFTP = true;
					m_lstFirmwares.Add(iCUFirmware2);
				}
				else
				{
					iCUFirmware.OnFTP = true;
				}
			}
		}
		catch (Exception exception)
		{
			Logger.Error(exception, "Upload failure!");
			return false;
		}
		return true;
	}

	public ICUGroup AddGroup()
	{
		ICUGroup iCUGroup = new ICUGroup(m_lstFeatures);
		iCUGroup.PropertyChanged += OnGroupChanged;
		m_lstGroups.Add(iCUGroup);
		m_fGroupCollectionDirty = true;
		OnGroupChanged(this, new PropertyChangedEventArgs("Group"));
		return iCUGroup;
	}

	public bool RemoveGroup(ICUGroup oldGroup)
	{
		m_lstGroups.Remove(oldGroup);
		m_fGroupCollectionDirty = true;
		return true;
	}

	public void ResetGroup(ICUGroup group)
	{
		group?.Rollback();
	}

	public ICUUser AddUser()
	{
		ICUUser iCUUser = new ICUUser();
		iCUUser.PropertyChanged += OnUserChanged;
		m_lstUsers.Add(iCUUser);
		m_fUserCollectionDirty = true;
		OnUserChanged(this, new PropertyChangedEventArgs("User"));
		return iCUUser;
	}

	public bool RemoveUser(ICUUser oldUser)
	{
		m_lstUsers.Remove(oldUser);
		m_fUserCollectionDirty = true;
		return true;
	}

	public void ResetUser(ICUUser user)
	{
		user?.Rollback();
	}

	public ICUBackOffice AddBackoffice(ICUBackOffice copyFrom)
	{
		ICUBackOffice iCUBackOffice = new ICUBackOffice();
		iCUBackOffice.PropertyChanged += OnBackOfficeChanged;
		iCUBackOffice.CopyFrom(copyFrom, allProperties: true);
		iCUBackOffice.Title += "(copy)";
		iCUBackOffice.TitleNL += "(copy)";
		iCUBackOffice.TitleDE += "(copy)";
		iCUBackOffice.TitleFR += "(copy)";
		m_lstBackOffices.Add(iCUBackOffice);
		m_fBackofficeCollectionDirty = true;
		OnBackOfficeChanged(this, new PropertyChangedEventArgs("Title"));
		return iCUBackOffice;
	}

	public bool RemoveBackoffice(ICUBackOffice oldBO)
	{
		m_lstBackOffices.Remove(oldBO);
		m_fBackofficeCollectionDirty = true;
		return true;
	}

	public void ResetBackoffice(ICUBackOffice backoffice)
	{
		backoffice?.Rollback();
	}

	public ICUPMBackOffice AddPMBackoffice(ICUPMBackOffice copyFrom)
	{
		ICUPMBackOffice iCUPMBackOffice = new ICUPMBackOffice();
		iCUPMBackOffice.PropertyChanged += OnBackOfficeChanged;
		iCUPMBackOffice.CopyFrom(copyFrom, allProperties: true);
		iCUPMBackOffice.Title += "(copy)";
		m_lstPMBackOffices.Add(iCUPMBackOffice);
		m_fPMBackofficeCollectionDirty = true;
		OnBackOfficeChanged(this, new PropertyChangedEventArgs("Title"));
		return iCUPMBackOffice;
	}

	public bool RemovePMBackoffice(ICUPMBackOffice oldBO)
	{
		m_lstPMBackOffices.Remove(oldBO);
		m_fPMBackofficeCollectionDirty = true;
		return true;
	}

	public void ResetPMBackoffice(ICUPMBackOffice backoffice)
	{
		backoffice?.Rollback();
	}

	public ICUFirmware AddFirmware()
	{
		ICUFirmware iCUFirmware = new ICUFirmware();
		iCUFirmware.PropertyChanged += OnFirmwareChanged;
		m_lstFirmwares.Add(iCUFirmware);
		m_fFirmwareCollectionDirty = true;
		OnFirmwareChanged(this, new PropertyChangedEventArgs("Firmware"));
		return iCUFirmware;
	}

	public ICUFirmware FindFirmware(string fileName)
	{
		string filenameNoPath = Path.GetFileName(fileName).ToLowerInvariant();
		return m_lstFirmwares.FirstOrDefault((ICUFirmware a) => a.Filename.ToLowerInvariant() == filenameNoPath);
	}

	public bool RemoveFirmware(ICUFirmware oldFirmware)
	{
		m_lstFirmwares.Remove(oldFirmware);
		m_fFirmwareCollectionDirty = true;
		return true;
	}

	public void ResetFirmware(ICUFirmware firmware)
	{
		firmware?.Rollback();
	}

	public void CommitChanges()
	{
		m_fThisDirty = false;
		m_fGroupCollectionDirty = false;
		m_fUserCollectionDirty = false;
		m_fBackofficeCollectionDirty = false;
		m_fFirmwareCollectionDirty = false;
		foreach (ICUFeature lstFeature in m_lstFeatures)
		{
			lstFeature.Commit();
		}
		foreach (ICUGroup lstGroup in m_lstGroups)
		{
			lstGroup.Commit();
		}
		foreach (ICUUser lstUser in m_lstUsers)
		{
			lstUser.Commit();
		}
		foreach (ICUBackOffice lstBackOffice in m_lstBackOffices)
		{
			lstBackOffice.Commit();
		}
		foreach (ICUPMBackOffice lstPMBackOffice in m_lstPMBackOffices)
		{
			lstPMBackOffice.Commit();
		}
		foreach (ICUFirmware lstFirmware in m_lstFirmwares)
		{
			lstFirmware.Commit();
		}
	}

	protected void AddBackofficesToFeatures()
	{
		m_lstFeatures = new ObservableCollection<ICUFeature>(m_lstFeatures.Where((ICUFeature a) => a.Type != ICUFeatureType.Backoffice));
		foreach (ICUBackOffice lstBackOffice in m_lstBackOffices)
		{
			if (!string.IsNullOrEmpty(lstBackOffice.Title))
			{
				AddFeature(ICUFeature.CreateBackOfficeFeature(lstBackOffice.Title, $"BackOffice '{lstBackOffice.Title}'"));
			}
		}
	}

	protected void AddDefaultGroups(List<string> allCompanies)
	{
		List<string> list = allCompanies.Distinct().ToList();
		m_lstGroups.Add(new ICUGroup(m_lstFeatures, "Admin", "Administrators"));
		m_lstGroups.Add(new ICUGroup(m_lstFeatures, "Production", "Production"));
		m_lstGroups.Add(new ICUGroup(m_lstFeatures, "Service", "Our own service engineers"));
		m_lstGroups.Add(new ICUGroup(m_lstFeatures, "Customer", "Customers"));
		foreach (string company in list)
		{
			if (!(company != ""))
			{
				continue;
			}
			ICUGroup iCUGroup = new ICUGroup(m_lstFeatures, $"Extern_{company}", company);
			if (iCUGroup == null)
			{
				continue;
			}
			List<string> list2 = (from a in m_lstBackOffices
				where a.Groups.Contains(company.ToUpperInvariant())
				select $"BO_{a.Title.Replace(' ', '_').Trim().ToUpperInvariant()}").ToList();
			foreach (ICUFeatureRight feature in iCUGroup.Features)
			{
				if (feature.Feature.Type == ICUFeatureType.Backoffice)
				{
					if (list2.Contains(feature.Feature.ID))
					{
						feature.Rights = ICURights.ReadOnly;
					}
					else
					{
						feature.Rights = ICURights.None;
					}
				}
			}
			m_lstGroups.Add(iCUGroup);
		}
	}

	protected void AddFeature(ICUFeature newFeature)
	{
		if (newFeature != null && !m_lstFeatures.Any((ICUFeature a) => a.ID == newFeature.ID))
		{
			m_lstFeatures.Add(newFeature);
		}
	}

	protected void AddManualFeatures()
	{
		AddFeature(ICUFeature.CreatePageFeature("INFORMATION", "Page Information", ICURights.ReadOnly));
		AddFeature(ICUFeature.CreatePageFeature("BACKOFFICE", "Page Backoffice"));
		AddFeature(ICUFeature.CreatePageFeature("POWER", "Page Power", ICURights.ReadOnly));
		AddFeature(ICUFeature.CreatePageFeature("NETWORK", "Page Network", ICURights.ReadOnly));
		AddFeature(ICUFeature.CreatePageFeature("STATES", "Page States", ICURights.ReadOnly));
		AddFeature(ICUFeature.CreatePageFeature("SOCKET", "Page Socket"));
		AddFeature(ICUFeature.CreatePageFeature("LOG", "Page Log"));
		AddFeature(ICUFeature.CreatePageFeature("METERVALUES", "Page MeterValue", ICURights.ReadOnly));
		AddFeature(ICUFeature.CreatePageFeature("WHITELIST", "Page Whitelist", ICURights.ReadOnly));
		AddFeature(ICUFeature.CreatePageFeature("UI", "Page UserInterface", ICURights.ReadOnly));
		AddFeature(ICUFeature.CreatePageFeature("UPLOAD", "Page Upload", ICURights.ReadOnly));
		AddFeature(ICUFeature.CreatePageFeature("PRODUCTION", "Page Production", ICURights.None));
		AddFeature(ICUFeature.CreatePageFeature("ALLPROPERTIES", "All settings page"));
		AddFeature(ICUFeature.CreatePageFeature("TRANSACTIONS", "Page Transactions"));
		AddFeature(ICUFeature.CreatePageFeature("FAT", "Factory Acceptance Test", ICURights.None));
		AddFeature(ICUFeature.CreatePageFeature("SAT", "Service Acceptance Test"));
		AddFeature(ICUFeature.CreateFeature("CREATEFWU", "Create an FWU file"));
		AddFeature(ICUFeature.CreateIDFeature(8272, "Charge Box Model", ICURights.ReadOnly));
		AddFeature(ICUFeature.CreateIDFeature(8273, "Charge Box Serial Number", ICURights.ReadOnly));
		AddFeature(ICUFeature.CreateIDFeature(8275, "Charge Box Identity", ICURights.ReadOnly));
		AddFeature(ICUFeature.CreateIDFeature(8271, "Charge Box Configuration", ICURights.ReadOnly));
		AddFeature(ICUFeature.CreateIDFeature(8290, "Max Station Current"));
		AddFeature(ICUFeature.CreateIDFeature(8488, "Start Max Current"));
		AddFeature(ICUFeature.CreateIDFeature(8489, "Normal Max Current"));
		AddFeature(ICUFeature.CreateIDFeature(8496, "Simplified Max Current"));
		AddFeature(ICUFeature.CreateIDFeature(8292, "Load Balancing Mode", ICURights.ReadOnly));
		AddFeature(ICUFeature.CreateIDFeature(8295, "P1 Max Installation Current"));
		AddFeature(ICUFeature.CreateIDFeature(8296, "P1 Balancing Safe Current"));
		AddFeature(ICUFeature.CreateIDFeature(8310, "BackOffice short name"));
		AddFeature(ICUFeature.CreateIDFeature(8311, "Connection method"));
		AddFeature(ICUFeature.CreateIDFeature(8312, "BackOffice Server Domain and Port"));
		AddFeature(ICUFeature.CreateIDFeature(8305, "BackOffice Server Domain and Port Wired"));
		AddFeature(ICUFeature.CreateIDFeature(8487, "Main Offline NFC Authorization"));
		AddFeature(ICUFeature.CreateIDFeature(8503, "Main EV Disconnect Action"));
		AddFeature(ICUFeature.CreateIDFeature(8448, "GPRS APN Name"));
		AddFeature(ICUFeature.CreateIDFeature(8449, "GPRS APN User"));
		AddFeature(ICUFeature.CreateIDFeature(8450, "GPRS APN Password"));
		AddFeature(ICUFeature.CreateIDFeature(8313, "Communication DNS 1"));
		AddFeature(ICUFeature.CreateIDFeature(8320, "Communication DNS 2"));
		AddFeature(ICUFeature.CreateIDFeature(8339, "Send Station Status"));
		AddFeature(ICUFeature.CreateIDFeature(8321, "Protocol Name"));
		AddFeature(ICUFeature.CreateIDFeature(8322, "Protocol Version"));
		AddFeature(ICUFeature.CreateIDFeature(8502, "EV Disconnect Timeout"));
		AddFeature(ICUFeature.CreateIDFeature(8289, "LED/Display AutoDim & Intensity"));
		AddFeature(ICUFeature.CreateIDFeature(8485, "Main Socket Type (Socket 1)"));
		AddFeature(ICUFeature.CreateIDFeature(12581, "Main Socket Type (Socket 2)"));
		AddFeature(ICUFeature.CreateIDFeature(8728, "Energy Meter Type (Socket 1)"));
		AddFeature(ICUFeature.CreateIDFeature(12824, "Energy Meter Type (Socket 2)"));
		AddFeature(ICUFeature.CreateIDFeature(8309, "IP1 Address"));
		AddFeature(ICUFeature.CreateIDFeature(8307, "IP1 Netmask"));
		AddFeature(ICUFeature.CreateIDFeature(8308, "IP1 Gateway Address"));
		AddFeature(ICUFeature.CreateIDFeature(8313, "IP1 DNS 1"));
		AddFeature(ICUFeature.CreateIDFeature(8320, "IP1 DNS 2"));
		AddFeature(ICUFeature.CreateIDFeature(8317, "IP2 Address"));
		AddFeature(ICUFeature.CreateIDFeature(8316, "IP2 Netmask"));
		AddFeature(ICUFeature.CreateIDFeature(8315, "IP2 Gateway Address"));
		AddFeature(ICUFeature.CreateIDFeature(8318, "IP2 DNS 1"));
		AddFeature(ICUFeature.CreateIDFeature(8319, "IP2 DNS 2"));
		AddFeature(ICUFeature.CreateIDFeature(8485, "Main Socket Type (Socket 1)"));
		AddFeature(ICUFeature.CreateIDFeature(12581, "Main Socket Type (Socket 2)"));
		AddFeature(ICUFeature.CreateIDFeature(8728, "Energy Meter Type (Socket 1)"));
		AddFeature(ICUFeature.CreateIDFeature(12824, "Energy Meter Type (Socket 2)"));
	}

	public bool UpdatePMBackOffices()
	{
		m_lstPMBackOffices.Clear();
		foreach (ICUBackOffice lstBackOffice in m_lstBackOffices)
		{
			string text = lstBackOffice.Title.ToLowerInvariant().Trim();
			string text2 = lstBackOffice.Title.Substring(0, lstBackOffice.Title.LastIndexOf('-')).Trim();
			if (text.EndsWith("production auto") || text.EndsWith("productionauto"))
			{
				AddPMBackOffice(text2, lstBackOffice, lstBackOffice);
			}
			else if (text.EndsWith("production gprs") || text.EndsWith("productiongprs"))
			{
				string autoName = lstBackOffice.Title.Replace("gprs", "auto");
				if (m_lstBackOffices.FirstOrDefault((ICUBackOffice a) => a.Title == autoName) == null)
				{
					string wiredName = lstBackOffice.Title.Replace("gprs", "wired");
					ICUBackOffice boWired = m_lstBackOffices.FirstOrDefault((ICUBackOffice a) => a.Title == wiredName);
					AddPMBackOffice(text2, lstBackOffice, boWired);
				}
			}
			else if (text.EndsWith("production wired") || text.EndsWith("productionwired"))
			{
				string autoName2 = lstBackOffice.Title.Replace("wired", "auto");
				string gprsName = lstBackOffice.Title.Replace("wired", "gprs");
				if (m_lstBackOffices.FirstOrDefault((ICUBackOffice a) => a.Title == autoName2) == null && m_lstBackOffices.FirstOrDefault((ICUBackOffice a) => a.Title == gprsName) == null)
				{
					AddPMBackOffice(text2, null, lstBackOffice);
				}
			}
			else if (text.EndsWith("sandbox auto"))
			{
				text2 = $"{text2} sandbox";
				AddPMBackOffice(text2, lstBackOffice, lstBackOffice);
			}
			else if (text.EndsWith("sandbox gprs"))
			{
				string autoName3 = lstBackOffice.Title.Replace("gprs", "auto");
				if (m_lstBackOffices.FirstOrDefault((ICUBackOffice a) => a.Title == autoName3) == null)
				{
					string wiredName2 = lstBackOffice.Title.Replace("gprs", "wired");
					ICUBackOffice boWired2 = m_lstBackOffices.FirstOrDefault((ICUBackOffice a) => a.Title == wiredName2);
					text2 = $"{text2} sandbox";
					AddPMBackOffice(text2, lstBackOffice, boWired2);
				}
			}
			else if (text.EndsWith("sandbox wired"))
			{
				string autoName4 = lstBackOffice.Title.Replace("wired", "auto");
				string gprsName2 = lstBackOffice.Title.Replace("wired", "gprs");
				if (m_lstBackOffices.FirstOrDefault((ICUBackOffice a) => a.Title == autoName4) == null && m_lstBackOffices.FirstOrDefault((ICUBackOffice a) => a.Title == gprsName2) == null)
				{
					text2 = $"{text2} sandbox";
					AddPMBackOffice(text2, null, lstBackOffice);
				}
			}
		}
		return true;
	}

	private bool AddPMBackOffice(string newtitle, ICUBackOffice boGPRS, ICUBackOffice boWired)
	{
		bool isLANEnabled = true;
		bool isGPRSEnabled = true;
		if (boWired == null)
		{
			isLANEnabled = false;
			boWired = boGPRS;
		}
		if (boGPRS == null)
		{
			isGPRSEnabled = false;
			boGPRS = boWired;
		}
		if (boWired == null && boGPRS == null)
		{
			return false;
		}
		ICUPMBackOffice item = new ICUPMBackOffice
		{
			Title = newtitle,
			APNName = boGPRS.APNName,
			APNUser = boGPRS.APNUser,
			APNPassword = boGPRS.APNPassword,
			BackOfficeURLwired_Domain = boWired.BackOfficeURLwired_Domain,
			BackOfficeURLwired_Path = boWired.BackOfficeURLwired_Path,
			BackOfficeURL_Domain = boGPRS.BackOfficeURL_Domain,
			BackOfficeURL_Path = boGPRS.BackOfficeURL_Path,
			CentralMeterValueAlignment = boGPRS.CentralMeterValueAlignment,
			DNS1_1 = boGPRS.DNS1_1,
			DNS1_2 = boGPRS.DNS1_2,
			DNS2_1 = boWired.DNS2_1,
			DNS2_2 = boWired.DNS2_2,
			EVDisconnectAction = boGPRS.EVDisconnectAction,
			EVDisconnectTimeout = boGPRS.EVDisconnectTimeout,
			IntensityAuto = boGPRS.IntensityAuto,
			IntensityIntensity = boGPRS.IntensityIntensity,
			Language = boGPRS.Language,
			OCPP15SmartChargingType = boGPRS.OCPP15SmartChargingType,
			OfflineNFCAuthorization = boGPRS.OfflineNFCAuthorization,
			OnlineNFCAuthorization = boGPRS.OnlineNFCAuthorization,
			PingPongInterval = boGPRS.PingPongInterval,
			ProtocolName = boGPRS.ProtocolName,
			ProtocolVersion = boGPRS.ProtocolVersion,
			SendStationStatus = boGPRS.SendStationStatus,
			SimPin = boGPRS.SimPin,
			TimezoneMinutes = boGPRS.TimezoneMinutes,
			TransactionMessageAttempts = boGPRS.TransactionMessageAttempts,
			TransactionMessageRetryInterval = boGPRS.TransactionMessageRetryInterval,
			IsGPRSEnabled = isGPRSEnabled,
			IsLANEnabled = isLANEnabled
		};
		m_lstPMBackOffices.Add(item);
		return true;
	}

	public ICUUser FindUser(string username)
	{
		string loweruser = username.ToLowerInvariant();
		return Users.FirstOrDefault((ICUUser a) => a.User.ToLowerInvariant() == loweruser);
	}
}
