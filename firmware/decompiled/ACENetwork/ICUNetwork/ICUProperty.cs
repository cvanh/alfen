using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.Globalization;
using System.Linq;
using System.Xml.Linq;
using ICUSettings;
using Serilog;

namespace ICUNetwork;

[DebuggerDisplay("{DebuggerDisplay,nq}")]
public class ICUProperty
{
	private readonly ILogger Logger = Log.ForContext<ICUProperty>();

	private object m_objValue;

	public ushort Id { get; set; }

	public byte SubId { get; set; }

	public string ICUName { get; set; }

	public string Title { get; set; }

	public string Units { get; set; }

	public SDT DataType { get; set; }

	public bool ReadOnly { get; set; }

	public bool IsChanged { get; private set; }

	public object DeviceValue { get; private set; }

	public string Category { get; set; }

	public ulong MaxLength { get; set; }

	private string DebuggerDisplay => $"{ID_SUB} ({DataType}) = {Value}";

	public object Value
	{
		get
		{
			return m_objValue;
		}
		set
		{
			SetValue(value);
		}
	}

	public EDSParameter Parameter { get; set; }

	public string ID_SUB => $"{Id:X4}_{SubId:X2}";

	public string ODIndex => $"{Id:X4}_{SubId:X}";

	public XElement Element
	{
		get
		{
			if (DataType == SDT.BYTEARRAY)
			{
				string value = "";
				byte[] array = (byte[])Value;
				if (array != null)
				{
					value = string.Join(",", array.Select((byte a) => a.ToString("X2")));
				}
				return new XElement("Property", new XAttribute("Id", ID_SUB), new XAttribute("Value", value));
			}
			if (DataType == SDT.ARRAY_16)
			{
				string value2 = "";
				ushort[] array2 = (ushort[])Value;
				if (array2 != null)
				{
					value2 = string.Join(",", array2.Select((ushort a) => a.ToString("X4")));
				}
				return new XElement("Property", new XAttribute("Id", ID_SUB), new XAttribute("Value", value2));
			}
			return new XElement("Property", new XAttribute("Id", ID_SUB), new XAttribute("Value", Value));
		}
	}

	public event EventHandler ValueChanged;

	public event EventHandler<ValueExceptionEventArgs> ExceptionOccured;

	internal ICUProperty()
	{
	}

	protected void SetValue(object newValue, bool invokeChange = true)
	{
		if (newValue == null)
		{
			Logger.Error("Setting a null value for {Name} Index={Index} Category={Category}", ICUName, ID_SUB, Category);
			return;
		}
		string text = (newValue.GetType().IsArray ? string.Empty : newValue.ToString());
		object objValue = m_objValue;
		try
		{
			switch (DataType)
			{
			case SDT.VISIBLE_STRING:
				m_objValue = newValue;
				break;
			case SDT.UNSIGNED8:
			{
				byte result12;
				if (text.Equals("false", StringComparison.OrdinalIgnoreCase))
				{
					m_objValue = (byte)0;
				}
				else if (text.Equals("true", StringComparison.OrdinalIgnoreCase))
				{
					m_objValue = (byte)1;
				}
				else if (byte.TryParse(text, NumberStyles.AllowLeadingWhite | NumberStyles.AllowTrailingWhite, CultureInfo.InvariantCulture, out result12))
				{
					m_objValue = result12;
				}
				else
				{
					m_objValue = (byte)0;
				}
				break;
			}
			case SDT.INTEGER8:
			{
				sbyte result6;
				if (text.Equals("false", StringComparison.OrdinalIgnoreCase))
				{
					m_objValue = (sbyte)0;
				}
				else if (text.Equals("true", StringComparison.OrdinalIgnoreCase))
				{
					m_objValue = (sbyte)1;
				}
				else if (sbyte.TryParse(text, NumberStyles.Integer, CultureInfo.InvariantCulture, out result6))
				{
					m_objValue = result6;
				}
				else
				{
					m_objValue = (sbyte)0;
				}
				break;
			}
			case SDT.UNSIGNED16:
			{
				if (ushort.TryParse(text, NumberStyles.AllowLeadingWhite | NumberStyles.AllowTrailingWhite, CultureInfo.InvariantCulture, out var result9))
				{
					m_objValue = result9;
				}
				else
				{
					m_objValue = (ushort)0;
				}
				break;
			}
			case SDT.UNSIGNED32:
			{
				if (uint.TryParse(text, NumberStyles.AllowLeadingWhite | NumberStyles.AllowTrailingWhite, CultureInfo.InvariantCulture, out var result2))
				{
					m_objValue = result2;
				}
				else
				{
					m_objValue = 0u;
				}
				break;
			}
			case SDT.UNSIGNED64:
			{
				if (ulong.TryParse(text, NumberStyles.AllowLeadingWhite | NumberStyles.AllowTrailingWhite, CultureInfo.InvariantCulture, out var result8))
				{
					m_objValue = result8;
				}
				else
				{
					m_objValue = 0uL;
				}
				break;
			}
			case SDT.INTEGER16:
			{
				if (short.TryParse(text, NumberStyles.Integer, CultureInfo.InvariantCulture, out var result3))
				{
					m_objValue = result3;
				}
				else
				{
					m_objValue = (short)0;
				}
				break;
			}
			case SDT.INTEGER32:
			{
				if (int.TryParse(text, NumberStyles.Integer, CultureInfo.InvariantCulture, out var result11))
				{
					m_objValue = result11;
				}
				else
				{
					m_objValue = 0;
				}
				break;
			}
			case SDT.INTEGER64:
			{
				if (long.TryParse(text, NumberStyles.Integer, CultureInfo.InvariantCulture, out var result7))
				{
					m_objValue = result7;
				}
				else
				{
					m_objValue = 0L;
				}
				break;
			}
			case SDT.REAL32:
			{
				if (float.TryParse(text.Replace(',', '.'), NumberStyles.Float, CultureInfo.InvariantCulture, out var result4))
				{
					m_objValue = result4;
				}
				else
				{
					m_objValue = 0f;
				}
				break;
			}
			case SDT.REAL64:
			{
				if (double.TryParse(text.Replace(',', '.'), NumberStyles.Float, CultureInfo.InvariantCulture, out var result13))
				{
					m_objValue = result13;
				}
				else
				{
					m_objValue = 0.0;
				}
				break;
			}
			case SDT.BOOLEAN:
			{
				if (bool.TryParse(text, out var result10))
				{
					m_objValue = result10;
				}
				else
				{
					m_objValue = (byte)0;
				}
				break;
			}
			case SDT.BYTEARRAY:
			{
				if (newValue.GetType().IsArray)
				{
					m_objValue = newValue;
					break;
				}
				List<byte> list2 = new List<byte>();
				string[] array = text.Split(new char[1] { ',' });
				for (int i = 0; i < array.Length; i++)
				{
					if (byte.TryParse(array[i], NumberStyles.HexNumber, CultureInfo.InvariantCulture, out var result5))
					{
						list2.Add(result5);
					}
					else
					{
						list2.Add(0);
					}
				}
				m_objValue = list2.ToArray();
				break;
			}
			case SDT.ARRAY_16:
			{
				if (newValue.GetType().IsArray)
				{
					m_objValue = newValue;
					break;
				}
				List<ushort> list = new List<ushort>();
				string[] array = text.Split(new char[1] { ',' });
				for (int i = 0; i < array.Length; i++)
				{
					if (ushort.TryParse(array[i], NumberStyles.HexNumber, CultureInfo.InvariantCulture, out var result))
					{
						list.Add(result);
					}
					else
					{
						list.Add(0);
					}
				}
				m_objValue = list.ToArray();
				break;
			}
			case SDT.LOW_LIMIT:
				m_objValue = string.Empty;
				ReadOnly = true;
				break;
			default:
				Logger.Debug("SDT {Name} has unknown datatype {DataType:X}. Id={Id:X} SubId={SubId} Category={Category}", ICUName, DataType, Id, SubId, Category);
				break;
			}
		}
		catch (Exception ex)
		{
			Logger.Verbose(ex, ex.Message);
			ValueExceptionEventArgs e = new ValueExceptionEventArgs
			{
				Exception = ex,
				InvalidValue = text
			};
			ExceptionOccured?.Invoke(this, e);
			return;
		}
		if (m_objValue != null)
		{
			IsChanged = !m_objValue.Equals(DeviceValue);
			if (!m_objValue.Equals(objValue) & invokeChange)
			{
				ValueChanged?.Invoke(this, EventArgs.Empty);
			}
		}
		else
		{
			IsChanged = false;
		}
	}

	public void SetDirty()
	{
		IsChanged = true;
	}

	public ICUProperty(EDSParameter param)
	{
		Initialize(param);
	}

	public ICUProperty(SDT dataType, ushort id, byte subId = 0)
	{
		DataType = dataType;
		Id = id;
		SubId = subId;
		IsChanged = false;
	}

	private void Initialize(EDSParameter param)
	{
		Parameter = param;
		DataType = (SDT)param.DataType;
		Id = (ushort)param.Id;
		SubId = (byte)param.SubId;
		ICUName = "OD_" + param.Name;
		Title = param.Title;
		Units = param.Units;
		IsChanged = false;
	}

	public override string ToString()
	{
		if (Value != null)
		{
			return Value.ToString();
		}
		return string.Empty;
	}

	public void SetInitialValue(object objInitialValue, bool notifyChanges = true)
	{
		SetValue(objInitialValue, invokeChange: false);
		DeviceValue = m_objValue;
		IsChanged = false;
		if (notifyChanges)
		{
			ValueChanged?.Invoke(this, EventArgs.Empty);
		}
	}

	public void CommitChange()
	{
		DeviceValue = m_objValue;
		IsChanged = false;
		ValueChanged?.Invoke(this, EventArgs.Empty);
	}

	public void Rollback()
	{
		SetValue(DeviceValue);
	}

	public void FireChanged()
	{
		ValueChanged?.Invoke(this, EventArgs.Empty);
	}
}
