using System;
using System.Collections.ObjectModel;
using System.Globalization;
using System.Linq;
using System.Net;
using System.Reflection;
using System.Text.RegularExpressions;
using ICUNetwork;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class UIPropertyString : UIPropertyBase
{
	protected UIPropertyStringType m_eType;

	protected string m_sDevicePropertyName = "";

	protected Label m_lblTitle;

	protected TextEntry m_txtValue;

	protected ulong m_nMaxLen = 256uL;

	protected string m_sCustomValue = "";

	protected bool m_writeOnly;

	public string CustomValue
	{
		get
		{
			return m_sCustomValue;
		}
		set
		{
			m_sCustomValue = value;
			RefreshDisplay();
		}
	}

	public UIPropertyString(PanelBase panelParent, Table table, int col, int line, string labelText, string devicePropertyName, UIPropertyStringType type = UIPropertyStringType.String, int nLeftMargin = 0, int reserveLines = 1, uint tooltipID = 0u)
		: base(panelParent, labelText, tooltipID)
	{
		m_sDevicePropertyName = devicePropertyName;
		m_eType = type;
		Initialize(table, col, line, nLeftMargin, "", reserveLines);
	}

	public UIPropertyString(PanelBase panelParent, Table table, int col, int line, ushort Id, byte subId = 0, UIPropertyStringType type = UIPropertyStringType.String, int nLeftMargin = 0, string overwriteLabelText = "", int reserveLines = 1, uint tooltipID = 0u)
		: base(panelParent, Id, subId, tooltipID)
	{
		m_eType = type;
		Initialize(table, col, line, nLeftMargin, overwriteLabelText, reserveLines);
	}

	public UIPropertyString(PanelBase panelParent, Table table, int col, int line, bool writeOnly, ushort Id, byte subId = 0, UIPropertyStringType type = UIPropertyStringType.String, int nLeftMargin = 0, string overwriteLabelText = "", int reserveLines = 1, uint tooltipID = 0u)
		: base(panelParent, Id, subId, tooltipID)
	{
		m_writeOnly = writeOnly;
		m_eType = type;
		Initialize(table, col, line, nLeftMargin, overwriteLabelText, reserveLines);
	}

	private void Initialize(Table table, int col, int line, int nLeftMargin = 0, string overwriteLabelText = "", int reserveLines = 1)
	{
		m_lblTitle = new Label(LabelText);
		if (!string.IsNullOrWhiteSpace(overwriteLabelText))
		{
			m_lblTitle.Text = overwriteLabelText;
			m_fUnknownTitle = false;
		}
		m_lblTitle.Font = UIPropertyBase.s_fntBaseLabel;
		m_txtValue = new TextEntry
		{
			Text = "",
			BackgroundColor = m_lblTitle.BackgroundColor
		};
		m_txtValue.Font = UIPropertyBase.s_fntBase;
		m_txtValue.ShowFrame = false;
		m_txtValue.Changed += onTxtChanged;
		m_txtValue.GotFocus += OnTxtGotFocus;
		m_txtValue.LostFocus += OnTxtLostFocus;
		m_txtValue.MinHeight = UIPropertyBase.s_propertyMinHeight;
		SetWidgetSizes(m_lblTitle, m_imvChanged, m_txtValue, m_btnTooltip, reserveLines - 1);
		m_lblTitle.TooltipText = TooltipID;
		table.Add(m_lblTitle, col, line, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, UIPropertyBase.s_marginHor + nLeftMargin, marginRight: Math.Max(UIPropertyBase.s_marginHorMax - nLeftMargin, 0), marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
		table.Add(m_txtValue, col + 1, line, 1, 1, hexpand: true, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, UIPropertyBase.s_marginHor, marginRight: UIPropertyBase.s_marginHor, marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
		table.Add(m_imvChanged, col + 2, line, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Start, WidgetPlacement.Fill, 1.0, UIPropertyBase.s_marginVer, 1.0, UIPropertyBase.s_marginVer);
		table.Add(m_btnTooltip, col + 3, line, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Start, WidgetPlacement.Fill, 1.0, UIPropertyBase.s_marginVer, 1.0, UIPropertyBase.s_marginVer);
		if (m_edsParameter != null)
		{
			m_nMaxLen = m_edsParameter.MaxLength;
		}
		if (m_eType == UIPropertyStringType.IP)
		{
			m_nMaxLen = 15uL;
		}
	}

	public override void SetTooltip(string text)
	{
		m_lblTitle.TooltipText = text;
	}

	public override void SetVisible(bool fVisible)
	{
		m_txtValue.Visible = fVisible;
		m_lblTitle.Visible = fVisible;
		m_imvChanged.Visible = fVisible;
		m_btnTooltip.Visible = fVisible && !string.IsNullOrEmpty(m_tooltipText);
	}

	public override void SetMinWidth(int minWidth)
	{
		m_txtValue.MinWidth = minWidth;
	}

	private void OnTxtGotFocus(object sender, EventArgs e)
	{
		m_txtValue.ShowFrame = true;
	}

	private void OnTxtLostFocus(object sender, EventArgs e)
	{
		if (!string.IsNullOrEmpty(m_txtValue.Text) && m_eType == UIPropertyStringType.IP)
		{
			if (IPAddress.TryParse(Regex.Replace(m_txtValue.Text, "0*([0-9]+)", "${1}"), out IPAddress address))
			{
				m_txtValue.Text = address.ToString();
			}
			else
			{
				m_txtValue.Text = "0.0.0.0";
			}
		}
		m_txtValue.ShowFrame = false;
	}

	private void onTxtChanged(object sender, EventArgs e)
	{
		if (m_objCurrentValue != null)
		{
			int cursorPosition = m_txtValue.CursorPosition;
			switch (m_eType)
			{
			case UIPropertyStringType.IP:
				m_txtValue.Text = Regex.Replace(m_txtValue.Text, "[^0-9.]", "");
				break;
			}
			if ((ulong)m_txtValue.Text.Length > m_nMaxLen)
			{
				m_txtValue.Text = m_txtValue.Text.Substring(0, (int)m_nMaxLen);
			}
			m_txtValue.CursorPosition = cursorPosition;
			if (!m_fReadOnly && !m_fForceReadOnly)
			{
				m_objCurrentValue.Value = m_txtValue.Text;
			}
		}
	}

	protected override void OnDeviceChange(ICUDevice device)
	{
		if (device == null)
		{
			m_txtValue.Text = string.Empty;
			m_txtValue.Sensitive = false;
			m_lblTitle.Sensitive = false;
		}
		else
		{
			m_txtValue.Sensitive = true;
			m_lblTitle.Sensitive = true;
			RefreshDisplay();
		}
	}

	public override void OnRefreshDisplay(ICUDevice device)
	{
		m_fReadOnly = true;
		if (device != null)
		{
			if (m_objCurrentValue != null && m_objCurrentValue.Value != null)
			{
				if (m_fUnknownTitle)
				{
					m_lblTitle.Text = m_objCurrentValue.Title;
				}
				UIPropertyStringType eType = m_eType;
				if (eType != UIPropertyStringType.String && eType == UIPropertyStringType.DateTime)
				{
					ulong num = ((!ulong.TryParse(m_objCurrentValue.Value.ToString(), NumberStyles.AllowLeadingWhite | NumberStyles.AllowTrailingWhite, CultureInfo.InvariantCulture, out num)) ? 0 : (num / 1000));
					if (num < 946684800)
					{
						m_txtValue.Text = UIPropertyBase.UnixTimeStampToTimeSpan(num).ToString();
					}
					else
					{
						ReadOnlyCollection<TimeZoneInfo> systemTimeZones = TimeZoneInfo.GetSystemTimeZones();
						DateTime dateTime = UIPropertyBase.UnixTimeStampToDateTime(num);
						int utcOffset = ((device.GetProperty(2125312u).DeviceValue != null) ? device.GetPropertyInt(8302, 0) : (device.GetPropertyInt(8282, 0) * 6));
						dateTime = TimeZoneInfo.ConvertTimeFromUtc(dateTime, TimeZoneInfo.FindSystemTimeZoneById(systemTimeZones.FirstOrDefault((TimeZoneInfo x) => x.BaseUtcOffset.TotalMinutes == (double)utcOffset).Id));
						m_txtValue.Text = dateTime.ToLongDateString() + " " + dateTime.ToLongTimeString();
					}
				}
				else if (!m_writeOnly)
				{
					m_txtValue.Text = m_objCurrentValue.Value.ToString().Trim(new char[2] { '\u0003', '\u001e' });
				}
			}
			else if (!string.IsNullOrEmpty(m_sDevicePropertyName))
			{
				PropertyInfo property = device.GetType().GetProperty(m_sDevicePropertyName);
				if (property != null)
				{
					m_txtValue.Text = property.GetValue(device, null).ToString();
				}
			}
			else
			{
				m_txtValue.Text = CustomValue;
			}
		}
		else
		{
			m_txtValue.Text = CustomValue;
		}
		if (m_objCurrentValue != null)
		{
			m_fReadOnly = m_objCurrentValue.ReadOnly;
			if (m_objCurrentValue.MaxLength > 1)
			{
				m_nMaxLen = m_objCurrentValue.MaxLength - 1;
			}
		}
		if (m_fForceReadOnly)
		{
			m_fReadOnly = true;
		}
		bool flag = m_fReadOnly;
		if (!m_fEnabled)
		{
			flag = true;
		}
		m_txtValue.BackgroundColor = (flag ? m_lblTitle.BackgroundColor : AppProperties.Color_Disabled);
		m_txtValue.ReadOnly = flag;
		m_lblTitle.Sensitive = m_fEnabled;
		if (flag)
		{
			m_txtValue.Font = UIPropertyBase.s_fntBase.WithWeight(FontWeight.Normal);
		}
		else
		{
			m_txtValue.Font = UIPropertyBase.s_fntBase.WithWeight(FontWeight.Semibold);
		}
	}

	protected override void OnValueChanged(ICUProperty prop)
	{
		RefreshDisplay();
		m_imvChanged.Image = (prop.IsChanged ? UIPropertyBase.s_imgChanged : UIPropertyBase.s_imgEmpty);
	}

	public override void SetValue(object newValue)
	{
		m_txtValue.Text = newValue.ToString();
	}
}
