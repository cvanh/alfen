using System;
using System.Collections.Generic;
using System.ComponentModel;
using System.Globalization;
using System.Linq;
using ICUSettings;
using Serilog;

namespace ICUNetwork;

public class ICUDevice
{
	public static readonly DateTime UnixEpoch = new DateTime(1970, 1, 1, 0, 0, 0, DateTimeKind.Utc);

	private readonly ILogger Logger = Log.ForContext<ICUDevice>();

	private const string PropertyConvertErrorFormat = "{0} type: {1} {2} message: {3}";

	public BaseConnection Connection { get; set; }

	public string Name { get; set; }

	public string SerialNumber { get; set; }

	public ICUPropertyDictionary PropertyDictionary { get; set; }

	public int NumberOfSockets { get; set; }

	public int NumberOfFeederCables { get; set; }

	public int[] SocketTypes { get; set; }

	public EndUserAccessType EndUserAccessType { get; set; }

	public string DisplayUserName { get; set; }

	public bool IsConnected
	{
		get
		{
			if (Connection != null)
			{
				return Connection.Devices.Any((ICUDevice a) => a == this);
			}
			return false;
		}
	}

	public virtual string Identification => "<unknown>";

	public virtual string DisplayNameLine2 => "";

	public virtual string Address => "";

	public virtual ICUDeviceModel ModelType => ICUDeviceModel.Unknown;

	public virtual string Model => "Unknown";

	public ICUDevice(BaseConnection connection, string connectionAddress = "")
	{
		Connection = connection;
		PropertyDictionary = new ICUPropertyDictionary();
		Name = string.Empty;
		SerialNumber = string.Empty;
	}

	public ICUProperty GetProperty(uint combinedPropId)
	{
		ushort num = (ushort)(combinedPropId >> 8);
		byte b = (byte)(combinedPropId & 0xFF);
		if (combinedPropId <= 65535)
		{
			num = (ushort)combinedPropId;
			b = 0;
		}
		if (num == 0 && b == 0)
		{
			return null;
		}
		return PropertyDictionary.GetProperty(num, b);
	}

	public ICUProperty GetProperty(ushort propId, byte subId = 0)
	{
		if (propId == 0 && subId == 0)
		{
			return null;
		}
		return PropertyDictionary.GetProperty(propId, subId);
	}

	public string GetPropertyString(ushort propId, byte subId = 0, ushort parentPropId = 0)
	{
		ICUProperty prop = GetProperty(propId, subId);
		ICUProperty iCUProperty = prop;
		if (prop != null && prop.Value != null)
		{
			if (parentPropId != 0)
			{
				iCUProperty = GetProperty(parentPropId, subId);
			}
			if (iCUProperty != null && iCUProperty.Parameter != null && iCUProperty.Parameter.Options != null)
			{
				EDSParameterOption eDSParameterOption = iCUProperty.Parameter.Options.FirstOrDefault((EDSParameterOption a) => a.Value == prop.Value.ToString());
				if (eDSParameterOption != null)
				{
					return eDSParameterOption.Title;
				}
			}
			return prop.Value.ToString();
		}
		return string.Empty;
	}

	public string GetDeviceValueString(ushort propId, byte subId = 0, ushort parentPropId = 0)
	{
		ICUProperty prop = GetProperty(propId, subId);
		ICUProperty iCUProperty = prop;
		if (prop != null && prop.DeviceValue != null)
		{
			if (parentPropId != 0)
			{
				iCUProperty = GetProperty(parentPropId, subId);
			}
			if (iCUProperty != null && iCUProperty.Parameter != null && iCUProperty.Parameter.Options != null)
			{
				EDSParameterOption eDSParameterOption = iCUProperty.Parameter.Options.FirstOrDefault((EDSParameterOption a) => a.Value == prop.DeviceValue.ToString());
				if (eDSParameterOption != null)
				{
					return eDSParameterOption.Title;
				}
			}
			if (prop.DataType == SDT.REAL32 || prop.DataType == SDT.REAL64)
			{
				return Convert.ToDouble(prop.DeviceValue).ToString("0.000", CultureInfo.InvariantCulture);
			}
			return prop.DeviceValue.ToString();
		}
		return string.Empty;
	}

	public int GetPropertyInt(ushort propId, byte subId = 0, int nValueMask = 0)
	{
		ICUProperty property = GetProperty(propId, subId);
		int num = 0;
		if (property != null && property.Value != null)
		{
			try
			{
				num = Convert.ToInt32(property.Value);
				if (nValueMask != 0)
				{
					num &= nValueMask;
				}
			}
			catch (Exception ex)
			{
				Logger.Error(ex, "{0} type: {1} {2} message: {3}", property.ID_SUB, property.DataType, property, ex.Message);
			}
		}
		return num;
	}

	public uint GetPropertyUInt(ushort propId, byte subId = 0, uint nValueMask = 0u)
	{
		ICUProperty property = GetProperty(propId, subId);
		uint num = 0u;
		if (property != null && property.Value != null)
		{
			try
			{
				num = Convert.ToUInt32(property.Value);
				if (nValueMask != 0)
				{
					num &= nValueMask;
				}
			}
			catch (Exception ex)
			{
				Logger.Error(ex, "{0} type: {1} {2} message: {3}", property.ID_SUB, property.DataType, property, ex.Message);
			}
		}
		return num;
	}

	public bool GetPropertyBool(ushort propId, byte subId = 0, bool defaultValue = false)
	{
		ICUProperty property = GetProperty(propId, subId);
		if (property != null && property.Value != null)
		{
			try
			{
				return Convert.ToBoolean(property.Value);
			}
			catch (Exception ex)
			{
				Logger.Error(ex, "{0} type: {1} {2} message: {3}", property.ID_SUB, property.DataType, property, ex.Message);
			}
		}
		return defaultValue;
	}

	public int GetDeviceValueInt(ushort propId, byte subId = 0, int nValueMask = 0)
	{
		ICUProperty property = GetProperty(propId, subId);
		int num = 0;
		if (property != null && property.Value != null)
		{
			try
			{
				num = Convert.ToInt32(property.DeviceValue);
				if (nValueMask != 0)
				{
					num &= nValueMask;
				}
			}
			catch (Exception ex)
			{
				Logger.Error(ex, "{0} type: {1} {2} message: {3}", property.ID_SUB, property.DataType, property, ex.Message);
			}
		}
		return num;
	}

	public ulong GetPropertyUInt64(ushort propId, byte subId = 0, ulong defaultValue = 0uL)
	{
		ICUProperty property = GetProperty(propId, subId);
		if (property != null && property.Value != null)
		{
			try
			{
				return Convert.ToUInt64(property.Value);
			}
			catch (Exception ex)
			{
				Logger.Error(ex, "{0} type: {1} {2} message: {3}", property.ID_SUB, property.DataType, property, ex.Message);
			}
		}
		return defaultValue;
	}

	public double GetPropertyDouble(ushort propId, byte subId = 0)
	{
		ICUProperty property = GetProperty(propId, subId);
		double result = 0.0;
		if (property != null && property.Value != null)
		{
			try
			{
				result = Convert.ToDouble(property.Value);
			}
			catch (Exception ex)
			{
				Logger.Error(ex, "{0} type: {1} {2} message: {3}", property.ID_SUB, property.DataType, property, ex.Message);
			}
		}
		return result;
	}

	public virtual bool OnDeviceRemoved()
	{
		return true;
	}

	public virtual bool UpdateProperties(params ICUProperty[] properties)
	{
		return false;
	}

	public virtual bool UpdateProperties(string sIds, bool notifyChanges = true)
	{
		return false;
	}

	public virtual bool UpdateProperties(params uint[] properties)
	{
		string[] value = properties.Select((uint a) => $"{a >> 8:X}_{a & 0xFF:X}").ToArray();
		return UpdateProperties(string.Join(",", value));
	}

	public virtual bool UpdateCategories(params string[] categories)
	{
		return false;
	}

	public virtual List<string> RequestCategories(Action dispatchUiEvents = null)
	{
		return null;
	}

	public virtual bool StoreChangedProperties()
	{
		List<ICUProperty> list = PropertyDictionary.Where((ICUProperty a) => a.IsChanged).ToList();
		if (list != null)
		{
			return StoreProperties(list.ToArray());
		}
		return false;
	}

	public virtual void RevertChanges()
	{
		foreach (ICUProperty item in PropertyDictionary.Where((ICUProperty a) => a.IsChanged).ToList())
		{
			item.Rollback();
		}
	}

	public virtual bool StoreProperties(params ICUProperty[] properties)
	{
		return false;
	}

	public virtual bool UploadFirmware(BackgroundWorker bgw, string fileName, string newPassword = "")
	{
		return false;
	}

	public virtual bool SaveProperties(string fileName)
	{
		return false;
	}
}
