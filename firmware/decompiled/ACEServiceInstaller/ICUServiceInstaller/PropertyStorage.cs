using System;
using System.Collections.Generic;
using System.IO;
using System.Text;
using System.Xml.Linq;
using ICUNetwork;
using ICUSettings;
using Serilog;
using Xwt;

namespace ICUServiceInstaller;

internal class PropertyStorage
{
	private static readonly ILogger Logger = Log.ForContext<PropertyStorage>();

	private const string s_key = "Alfen";

	protected static List<ushort> s_forbiddenProperties = new List<ushort>
	{
		8271, 8273, 8274, 8275, 8277, 8281, 8287, 8309, 8317, 8451,
		8549, 8550, 8551, 8576, 9506, 9507, 9520, 8583, 8609, 8728,
		8729, 9568, 9569, 9570, 9571, 9572, 9573, 9584, 9585, 9586,
		9587, 9588, 9589, 12824, 16919, 16920, 21015, 21016
	};

	public static bool SaveProperties(string fileName, ICULanDevice lanDevice)
	{
		try
		{
			lanDevice.UpdateCategories();
			XDocument xDocument = new XDocument();
			XElement xElement = new XElement("Settings");
			xElement.Add(new XElement("XMLVersion", "1.0"));
			xElement.Add(new XElement("Version", "1.0"));
			xElement.Add(new XElement("Date", DateTime.Now));
			xElement.Add(new XElement("Model", lanDevice.Model));
			xElement.Add(new XElement("NumberOfSockets", lanDevice.NumberOfSockets));
			xElement.Add(new XElement("Information", ""));
			XElement xElement2 = new XElement("Device");
			xElement2.Add(new XElement("Identity", lanDevice.Identification));
			xElement2.Add(new XElement("IPAddress", lanDevice.IPAddress));
			xElement2.Add(new XElement("Port", lanDevice.Port));
			xElement2.Add(new XElement("HostName", lanDevice.HostName));
			xElement.Add(xElement2);
			XElement xElement3 = new XElement("Properties");
			foreach (ICUProperty item in lanDevice.PropertyDictionary)
			{
				if (!item.ReadOnly && item.Value != null && !s_forbiddenProperties.Contains(item.Id))
				{
					xElement3.Add(item.Element);
				}
			}
			xElement.Add(xElement3);
			xDocument.Add(xElement);
			StringBuilder value = new StringBuilder(new EncryptDecrypt().Encrypt(xDocument.ToString(), "Alfen"));
			using (StreamWriter streamWriter = new StreamWriter(fileName))
			{
				streamWriter.Write((object?)value);
			}
			return true;
		}
		catch (Exception ex)
		{
			Logger.Error(ex, ex.Message);
			MessageDialog.ShowError(ex.Message);
		}
		return false;
	}

	public static int LoadProperties(string fileName, ICULanDevice lanDevice)
	{
		try
		{
			int num = 0;
			new List<ICUProperty>();
			string text = File.ReadAllText(fileName);
			if (string.Compare(Path.GetExtension(fileName), ".exml") == 0)
			{
				text = new EncryptDecrypt().Decrypt(text, "Alfen");
			}
			XDocument xDocument = XDocument.Parse(text);
			if (xDocument != null)
			{
				XElement xElement = xDocument.Element("Settings");
				if (xElement != null)
				{
					if (xElement.Element("XMLVersion").Value == "1.0")
					{
						bool flag = false;
						ICUDeviceModel iCUDeviceModel = ICUDeviceModelExtension.From(xElement.Element("Model").Value);
						if (iCUDeviceModel != lanDevice.ModelType)
						{
							if (MessageDialog.AskQuestion("Warning! The setting you are trying to load are for a '" + ICUDeviceModelExtension.ToString(iCUDeviceModel, "") + "', you currently have selected a '" + lanDevice.Model + "'.", "Are you sure you want to load this settings file?", Command.Yes, Command.No, Command.Cancel) == Command.Yes)
							{
								flag = true;
							}
						}
						else
						{
							flag = true;
						}
						if (flag)
						{
							foreach (XElement item in xElement.Element("Properties").Elements("Property"))
							{
								string value = item.Attribute("Id").Value;
								ICUProperty property = lanDevice.PropertyDictionary.GetProperty(value);
								if (property != null && !s_forbiddenProperties.Contains(property.Id))
								{
									bool isChanged = property.IsChanged;
									property.Value = item.Attribute("Value").Value;
									if (!isChanged && property.IsChanged)
									{
										num++;
									}
								}
							}
						}
					}
					else
					{
						MessageDialog.ShowError("Incorrect version of setting file");
					}
				}
			}
			return num;
		}
		catch (Exception ex)
		{
			Logger.Error(ex, ex.Message);
			MessageDialog.ShowError(ex.Message);
		}
		return 0;
	}
}
