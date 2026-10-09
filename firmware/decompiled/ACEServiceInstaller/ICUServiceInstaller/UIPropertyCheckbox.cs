using System;
using ICUNetwork;
using Serilog;
using Xwt;

namespace ICUServiceInstaller;

public class UIPropertyCheckbox : UIPropertyBase
{
	private readonly ILogger Logger = Log.ForContext<UIPropertyCheckbox>();

	protected Label m_lblTitle;

	protected CheckBox m_chkValue;

	protected int m_nValueMask;

	public bool IsChecked => m_chkValue.Active;

	public UIPropertyCheckbox(PanelBase panelParent, Table table, int col, int line, ushort Id, byte subId = 0, int nLeftMargin = 0, uint tooltipID = 0u)
		: base(panelParent, Id, subId, tooltipID)
	{
		Initialize(table, col, line, Id, subId, nLeftMargin);
	}

	public UIPropertyCheckbox(PanelBase panelParent, Table table, int col, int line, ushort Id, byte subId = 0, int valueMask = 0, string overwriteLabelText = "", int nLeftMargin = 0, uint tooltipID = 0u)
		: base(panelParent, Id, subId, tooltipID)
	{
		m_nValueMask = valueMask;
		Initialize(table, col, line, Id, subId, nLeftMargin, overwriteLabelText);
	}

	protected void Initialize(Table table, int col, int line, ushort Id, byte subId = 0, int nLeftMargin = 0, string overwriteLabelText = "")
	{
		m_lblTitle = new Label(LabelText);
		if (!string.IsNullOrWhiteSpace(overwriteLabelText))
		{
			m_lblTitle.Text = overwriteLabelText;
			m_fUnknownTitle = false;
		}
		m_lblTitle.Font = UIPropertyBase.s_fntBaseLabel;
		m_chkValue = new CheckBox
		{
			BackgroundColor = AppProperties.Color_Disabled
		};
		m_chkValue.Clicked += OnChkClicked;
		m_chkValue.MinHeight = UIPropertyBase.s_propertyMinHeight;
		SetWidgetSizes(m_lblTitle, m_imvChanged, m_chkValue, m_btnTooltip);
		m_lblTitle.TooltipText = TooltipID;
		table.Add(m_lblTitle, col, line, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, UIPropertyBase.s_marginHor + nLeftMargin, marginRight: Math.Max(UIPropertyBase.s_marginHorMax - nLeftMargin, 0), marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
		table.Add(m_chkValue, col + 1, line, 1, 1, hexpand: true, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.End, UIPropertyBase.s_marginHor, marginRight: UIPropertyBase.s_marginHor, marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
		table.Add(m_imvChanged, col + 2, line, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Start, WidgetPlacement.Fill, 1.0, UIPropertyBase.s_marginVer, 1.0, UIPropertyBase.s_marginVer);
		table.Add(m_btnTooltip, col + 3, line, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Start, WidgetPlacement.Fill, 1.0, UIPropertyBase.s_marginVer, 1.0, UIPropertyBase.s_marginVer);
	}

	public override void SetTooltip(string text)
	{
		m_lblTitle.TooltipText = text;
	}

	public override void SetVisible(bool fVisible)
	{
		m_chkValue.Visible = fVisible;
		m_lblTitle.Visible = fVisible;
		m_imvChanged.Visible = fVisible;
		m_btnTooltip.Visible = fVisible && !string.IsNullOrEmpty(m_tooltipText);
	}

	public override void SetMinWidth(int minWidth)
	{
		m_chkValue.MinWidth = minWidth;
	}

	private void OnChkClicked(object sender, EventArgs e)
	{
		if (m_objCurrentValue != null && !m_fReadOnly && !m_fForceReadOnly)
		{
			m_objCurrentValue.Value = GetValue();
		}
	}

	protected override void OnDeviceChange(ICUDevice device)
	{
		if (device == null)
		{
			m_chkValue.Active = false;
			m_lblTitle.Sensitive = false;
			m_chkValue.Sensitive = false;
		}
		else
		{
			m_lblTitle.Sensitive = true;
			RefreshDisplay();
		}
	}

	public override void OnRefreshDisplay(ICUDevice device)
	{
		if (m_chkValue == null)
		{
			return;
		}
		if (device != null && m_objCurrentValue != null && m_objCurrentValue.Value != null)
		{
			if (m_fUnknownTitle)
			{
				m_lblTitle.Text = m_objCurrentValue.Title;
			}
			bool flag = false;
			if (m_nValueMask != 0)
			{
				flag = (Convert.ToInt32(m_objCurrentValue.Value) & m_nValueMask) == m_nValueMask;
			}
			else
			{
				bool result = false;
				flag = bool.TryParse(m_objCurrentValue.Value.ToString(), out result);
				if (!result)
				{
					flag = Convert.ToInt32(m_objCurrentValue.Value) != 0;
				}
			}
			m_chkValue.Active = flag;
		}
		if (m_objCurrentValue != null)
		{
			m_fReadOnly = m_objCurrentValue.ReadOnly;
		}
		bool flag2 = m_fReadOnly;
		if (m_fForceReadOnly)
		{
			flag2 = true;
		}
		if (!m_fEnabled)
		{
			flag2 = true;
		}
		m_lblTitle.Sensitive = m_fEnabled;
		m_chkValue.Sensitive = !flag2;
	}

	protected override void OnValueChanged(ICUProperty prop)
	{
		RefreshDisplay();
		if (m_nValueMask == 0)
		{
			m_imvChanged.Image = ((m_chkValue.Sensitive && prop.IsChanged) ? UIPropertyBase.s_imgChanged : UIPropertyBase.s_imgEmpty);
		}
		else
		{
			m_imvChanged.Image = ((BitFlagged(prop.Value) != BitFlagged(prop.DeviceValue)) ? UIPropertyBase.s_imgChanged : UIPropertyBase.s_imgEmpty);
		}
	}

	private bool BitFlagged(object value)
	{
		return ((BitFlags)Convert.ToByte(value)).HasFlag((BitFlags)m_nValueMask);
	}

	public override object GetValue()
	{
		if (m_objCurrentValue != null)
		{
			switch (m_objCurrentValue.DataType)
			{
			case SDT.INTEGER8:
				if (m_nValueMask != 0)
				{
					sbyte b2 = Convert.ToSByte(m_objCurrentValue.Value);
					if (m_chkValue.Active)
					{
						return b2 | (sbyte)m_nValueMask;
					}
					return (sbyte)(b2 & ~m_nValueMask);
				}
				return m_chkValue.Active ? ((sbyte)1) : ((sbyte)0);
			case SDT.UNSIGNED8:
				if (m_nValueMask != 0)
				{
					byte b = Convert.ToByte(m_objCurrentValue.Value);
					if (m_chkValue.Active)
					{
						return b | (byte)m_nValueMask;
					}
					return (byte)(b & ~m_nValueMask);
				}
				return m_chkValue.Active ? ((byte)1) : ((byte)0);
			case SDT.BOOLEAN:
				return m_chkValue.Active;
			}
			Logger.Debug("Datatype {DataType} is not supported by GetValue", m_objCurrentValue.DataType);
		}
		return m_chkValue.Active;
	}

	public override void SetValue(object newValue)
	{
		if (m_objCurrentValue == null)
		{
			return;
		}
		switch (m_objCurrentValue.DataType)
		{
		case SDT.INTEGER8:
			if (m_nValueMask != 0)
			{
				m_chkValue.Active = (Convert.ToSByte(newValue) & m_nValueMask) == m_nValueMask;
			}
			else
			{
				m_chkValue.Active = Convert.ToSByte(newValue) == 1;
			}
			return;
		case SDT.UNSIGNED8:
			if (m_nValueMask != 0)
			{
				m_chkValue.Active = (Convert.ToByte(newValue) & m_nValueMask) == m_nValueMask;
			}
			else
			{
				m_chkValue.Active = Convert.ToByte(newValue) == 1;
			}
			return;
		}
		if (newValue.ToString().Trim() == "1")
		{
			m_chkValue.Active = true;
		}
		else if (newValue.ToString().Trim() == "0")
		{
			m_chkValue.Active = false;
		}
		else
		{
			m_chkValue.Active = Convert.ToBoolean(newValue);
		}
	}
}
