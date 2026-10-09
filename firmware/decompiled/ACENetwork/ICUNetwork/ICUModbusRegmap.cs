using System;
using System.Collections.Generic;
using System.Linq;
using Newtonsoft.Json;
using Newtonsoft.Json.Serialization;
using Serilog;

namespace ICUNetwork;

public class ICUModbusRegmap
{
	private readonly ILogger Logger = Log.ForContext<ModbusRegmapEntry>();

	private readonly ICUDevice m_device;

	public ModbusRegmapData RegmapData { get; set; }

	public List<string> LoadFromJSONErrorList { get; set; }

	public List<string> WriteToDeviceErrorList { get; set; }

	public ICUModbusRegmap(ICUDevice m_device)
	{
		this.m_device = m_device;
		LoadFromJSONErrorList = new List<string>();
		WriteToDeviceErrorList = new List<string>();
	}

	public bool LoadFromDevice(EMeterTypes energyMeterType)
	{
		if (!m_device.UpdateCategories("MbusTCP"))
		{
			Logger.Error("Failed updating category MbusTCP");
			return false;
		}
		bool flag = energyMeterType == EMeterTypes.ENERGYMETER_MODBUS_CENTRAL;
		ICUProperty property = m_device.GetProperty(flag ? 9568u : 9584u);
		ICUProperty property2 = m_device.GetProperty(flag ? 9569u : 9585u);
		ICUProperty property3 = m_device.GetProperty(flag ? 9570u : 9586u);
		ICUProperty property4 = m_device.GetProperty(flag ? 9571u : 9587u);
		if (property == null || property2 == null || property3 == null || property4 == null)
		{
			Logger.Error("Missing a modbusTCP/IP register mapping property");
			return false;
		}
		byte[] array = (byte[])property.Value;
		ushort[] array2 = (ushort[])property2.Value;
		byte[] array3 = (byte[])property3.Value;
		sbyte[] array4 = (sbyte[])property4.Value;
		RegmapData = new ModbusRegmapData
		{
			Name = "",
			Version = "",
			Address = null,
			Parity = "",
			Baudrate = null,
			WordOrder = "",
			UpdateTime = null,
			ReadTimeout = null,
			FunctionCode = "",
			SampleIntervalMs = 2000,
			Regmap = new List<ModbusRegmapEntry>()
		};
		bool flag2 = false;
		for (int i = 0; i < array.Length; i++)
		{
			if (array[i] != byte.MaxValue)
			{
				try
				{
					RegmapData.Regmap.Add(new ModbusRegmapEntry
					{
						Key = (EnergyMeterMeasurand)Enum.ToObject(typeof(EnergyMeterMeasurand), array[i]),
						RegNum = array2[i],
						DataType = (ModbusDataType)Enum.ToObject(typeof(ModbusDataType), array3[i]),
						ScaleE = array4[i]
					});
				}
				catch (Exception ex)
				{
					Logger.Error(ex, "Can't parse at key index:{Index} msg:{Message}", i, ex.Message);
					flag2 = true;
				}
			}
		}
		return !flag2;
	}

	public bool WriteToDevice(EMeterTypes energyMeterType)
	{
		WriteToDeviceErrorList.Clear();
		if (!m_device.UpdateCategories("MbusTCP"))
		{
			WriteToDeviceErrorList.Add("Failed updating category MbusTCP.");
			return false;
		}
		bool flag = energyMeterType == EMeterTypes.ENERGYMETER_TCPIP_CENTRAL;
		ICUProperty property = m_device.GetProperty(flag ? 9568u : 9584u);
		ICUProperty property2 = m_device.GetProperty(flag ? 9569u : 9585u);
		ICUProperty property3 = m_device.GetProperty(flag ? 9570u : 9586u);
		ICUProperty property4 = m_device.GetProperty(flag ? 9571u : 9587u);
		if (property == null || property2 == null || property3 == null || property4 == null)
		{
			WriteToDeviceErrorList.Add("Missing a modbusTCP/IP register mapping property.");
			return false;
		}
		if (RegmapData.Regmap.Count > ((byte[])property.Value).Length)
		{
			WriteToDeviceErrorList.Add($"Maximum number of Modbus TCP/IP register mapping entries is {((byte[])property.Value).Length}.");
			return false;
		}
		List<byte> list = new List<byte>();
		List<ushort> list2 = new List<ushort>();
		List<byte> list3 = new List<byte>();
		List<sbyte> list4 = new List<sbyte>();
		foreach (ModbusRegmapEntry item in RegmapData.Regmap)
		{
			list.Add((byte)item.Key);
			list2.Add(item.RegNum);
			list3.Add((byte)item.DataType);
			list4.Add(item.ScaleE);
		}
		while (list.Count < ((byte[])property.Value).Length)
		{
			list.Add(byte.MaxValue);
			list2.Add(0);
			list3.Add(0);
			list4.Add(0);
		}
		property.Value = list.ToArray();
		property2.Value = list2.ToArray();
		property3.Value = list3.ToArray();
		property4.Value = list4.ToArray();
		List<ICUProperty> list5 = new List<ICUProperty>();
		if (property.IsChanged)
		{
			list5.Add(property);
		}
		if (property2.IsChanged)
		{
			list5.Add(property2);
		}
		if (property3.IsChanged)
		{
			list5.Add(property3);
		}
		if (property4.IsChanged)
		{
			list5.Add(property4);
		}
		if (list5.Count != 0)
		{
			if ((m_device as ICULanDevice).HasProperty(9507, 2))
			{
				(m_device as ICULanDevice).storeProperty(9507, 2, 2);
			}
			(m_device as ICULanDevice).storeProperty(8292, 0, 0);
			if (!m_device.StoreProperties(list5.ToArray()))
			{
				WriteToDeviceErrorList.Add("Storing properties failed.");
				return false;
			}
			(m_device as ICULanDevice).storeProperty(8292, 0, 3);
		}
		return true;
	}

	public void LoadFromJSON(string json)
	{
		LoadFromJSONErrorList.Clear();
		if (string.IsNullOrWhiteSpace(json))
		{
			LoadFromJSONErrorList.Add("json is empty");
			return;
		}
		JsonSerializerSettings settings = new JsonSerializerSettings
		{
			Error = HandleDeserializationError,
			MissingMemberHandling = MissingMemberHandling.Error
		};
		RegmapData = JsonConvert.DeserializeObject<ModbusRegmapData>(json, settings);
	}

	public string WriteToJSON()
	{
		return JsonConvert.SerializeObject(RegmapData, Formatting.Indented);
	}

	private void HandleDeserializationError(object sender, ErrorEventArgs errorArgs)
	{
		string message = errorArgs.ErrorContext.Error.Message;
		Logger.Error(message);
		LoadFromJSONErrorList.Add(message);
		errorArgs.ErrorContext.Handled = true;
	}

	public ModbusRegmapEntry GetEntry(EnergyMeterMeasurand key)
	{
		return RegmapData.Regmap.FirstOrDefault((ModbusRegmapEntry a) => a.Key == key);
	}
}
