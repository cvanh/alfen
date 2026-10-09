using System;
using System.Linq;
using ICUNetwork;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class UIPropertyColor : UIPropertyBase
{
	public string Name { get; set; }

	public ICUProperty Property { get; set; }

	public Color State1Color { get; set; }

	public int State1Time { get; set; }

	public Color State2Color { get; set; }

	public int State2Time { get; set; }

	public byte[] OriginalState { get; set; }

	public UIPropertyColor(PanelBase panelParent, string colorName, ushort Id, byte subId = 0)
		: base(panelParent, Id, subId)
	{
		Name = colorName;
	}

	protected override void OnDeviceChange(ICUDevice device)
	{
		RefreshDisplay();
	}

	public override void OnRefreshDisplay(ICUDevice device)
	{
		if (device != null)
		{
			m_objCurrentValue = device.GetProperty(m_usId, m_bSubId);
			if (m_objCurrentValue != null)
			{
				byte[] array = (byte[])m_objCurrentValue.Value;
				if (InitializeFromArray(array))
				{
					OriginalState = array;
				}
			}
		}
		if (m_objCurrentValue != null)
		{
			m_fReadOnly = m_objCurrentValue.ReadOnly;
		}
	}

	protected bool InitializeFromArray(byte[] arData)
	{
		if (arData != null && arData.Length > 9)
		{
			State1Color = Color.FromBytes(arData[0], arData[1], arData[2]);
			State1Time = arData[4] * 10;
			State2Color = Color.FromBytes(arData[5], arData[6], arData[7]);
			State2Time = arData[9] * 10;
			return true;
		}
		return false;
	}

	protected override void OnValueChanged(ICUProperty prop)
	{
		RefreshDisplay();
		m_imvChanged.Image = (prop.IsChanged ? UIPropertyBase.s_imgChanged : UIPropertyBase.s_imgEmpty);
	}

	public override object GetValue()
	{
		if (m_objCurrentValue != null)
		{
			return string.Join(",", new byte[10]
			{
				(byte)(State1Color.Red * 255.0),
				(byte)(State1Color.Green * 255.0),
				(byte)(State1Color.Blue * 255.0),
				OriginalState[3],
				(byte)(State1Time / 10),
				(byte)(State2Color.Red * 255.0),
				(byte)(State2Color.Green * 255.0),
				(byte)(State2Color.Blue * 255.0),
				OriginalState[8],
				(byte)(State2Time / 10)
			}.Select((byte a) => a.ToString("X2")));
		}
		return null;
	}

	public override void SetValue(object newValue)
	{
		byte[] array = (byte[])newValue;
		if (InitializeFromArray(array) && m_objCurrentValue != null)
		{
			m_objCurrentValue.Value = array;
		}
	}

	public void SetValue(Color col1, double time1, Color col2, double time2)
	{
		State1Color = col1;
		State2Color = col2;
		State1Time = Convert.ToInt32(time1);
		State2Time = Convert.ToInt32(time2);
		byte[] value = new byte[10]
		{
			(byte)(State1Color.Red * 255.0),
			(byte)(State1Color.Green * 255.0),
			(byte)(State1Color.Blue * 255.0),
			OriginalState[3],
			(byte)(State1Time / 10),
			(byte)(State2Color.Red * 255.0),
			(byte)(State2Color.Green * 255.0),
			(byte)(State2Color.Blue * 255.0),
			OriginalState[8],
			(byte)(State2Time / 10)
		};
		if (m_objCurrentValue != null)
		{
			m_objCurrentValue.Value = value;
		}
	}
}
