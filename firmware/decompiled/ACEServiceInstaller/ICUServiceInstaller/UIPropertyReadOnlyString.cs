using System;
using System.Collections.ObjectModel;
using System.Globalization;
using System.Linq;
using System.Reflection;
using ICUIWSConnection;
using ICUNetwork;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class UIPropertyReadOnlyString : UIPropertyBase
{
	protected UIPropertyStringType m_eType;

	protected string m_sDevicePropertyName = "";

	protected Label m_lblTitle;

	protected TextEntry m_txtValue;

	protected int m_nMaxLen = 256;

	protected string m_sCustomValue = "";

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

	public UIPropertyReadOnlyString(PanelBase panelParent, Table table, int col, int line, string labelText, string devicePropertyName, UIPropertyStringType type = UIPropertyStringType.String, int nLeftMargin = 0, uint tooltipID = 0u)
		: base(panelParent, labelText, tooltipID)
	{
		m_sDevicePropertyName = devicePropertyName;
		m_eType = type;
		Initialize(table, col, line, nLeftMargin);
	}

	public UIPropertyReadOnlyString(PanelBase panelParent, Table table, int col, int line, ushort Id, byte subId = 0, UIPropertyStringType type = UIPropertyStringType.String, int nLeftMargin = 0, string overwriteLabelText = "", uint tooltipID = 0u)
		: base(panelParent, Id, subId, tooltipID)
	{
		m_eType = type;
		Initialize(table, col, line, nLeftMargin, overwriteLabelText);
	}

	private void Initialize(Table table, int col, int line, int nLeftMargin = 0, string overwriteLabelText = "")
	{
		m_lblTitle = new Label(LabelText);
		if (!string.IsNullOrWhiteSpace(overwriteLabelText))
		{
			m_lblTitle.Text = overwriteLabelText;
			m_fUnknownTitle = false;
		}
		m_lblTitle.Font = UIPropertyBase.s_fntBaseLabel;
		m_txtValue = new TextEntry();
		m_txtValue.Text = "";
		m_txtValue.ShowFrame = false;
		m_txtValue.MinHeight = UIPropertyBase.s_propertyMinHeight;
		m_txtValue.ReadOnly = true;
		m_txtValue.BackgroundColor = m_lblTitle.BackgroundColor;
		m_txtValue.Font = UIPropertyBase.s_fntBase.WithWeight(FontWeight.Normal);
		SetWidgetSizes(m_lblTitle, m_imvChanged, m_txtValue, m_btnTooltip);
		m_lblTitle.TooltipText = TooltipID;
		table.Add(m_lblTitle, col, line, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Start, UIPropertyBase.s_marginHor + nLeftMargin, marginRight: Math.Max(UIPropertyBase.s_marginHorMax - nLeftMargin, 0), marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
		table.Add(m_txtValue, col + 1, line, 1, 1, hexpand: true, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Start, UIPropertyBase.s_marginHor, marginRight: UIPropertyBase.s_marginHor, marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
		table.Add(m_imvChanged, col + 2, line, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Start, WidgetPlacement.Start, 1.0, UIPropertyBase.s_marginVer, 1.0, UIPropertyBase.s_marginVer);
		table.Add(m_btnTooltip, col + 3, line, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Start, WidgetPlacement.Start, 1.0, UIPropertyBase.s_marginVer, 1.0, UIPropertyBase.s_marginVer);
		m_imvChanged.Visible = false;
	}

	public override void SetTooltip(string text)
	{
		m_lblTitle.TooltipText = text;
	}

	public override void SetVisible(bool fVisible)
	{
		m_txtValue.Visible = fVisible;
		m_lblTitle.Visible = fVisible;
		m_btnTooltip.Visible = fVisible && !string.IsNullOrEmpty(m_tooltipText);
	}

	public override void SetMinWidth(int minWidth)
	{
		m_txtValue.MinWidth = minWidth;
	}

	protected override void OnDeviceChange(ICUDevice device)
	{
		if (device == null)
		{
			m_txtValue.Text = string.Empty;
			m_lblTitle.Sensitive = false;
		}
		else
		{
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
				double num = 0.0;
				switch (m_eType)
				{
				case UIPropertyStringType.DateTime:
				{
					ulong num2 = ((!ulong.TryParse(m_objCurrentValue.Value.ToString(), NumberStyles.AllowLeadingWhite | NumberStyles.AllowTrailingWhite, CultureInfo.InvariantCulture, out num2)) ? 0 : (num2 / 1000));
					if (num2 < 946684800)
					{
						m_txtValue.Text = UIPropertyBase.UnixTimeStampToTimeSpan(num2).ToString();
						break;
					}
					ReadOnlyCollection<TimeZoneInfo> systemTimeZones = TimeZoneInfo.GetSystemTimeZones();
					DateTime dateTime = UIPropertyBase.UnixTimeStampToDateTime(num2);
					int utcOffset = ((device.GetProperty(2125312u).DeviceValue != null) ? device.GetPropertyInt(8302, 0) : (device.GetPropertyInt(8282, 0) * 6));
					TimeZoneInfo timeZoneInfo = systemTimeZones.FirstOrDefault((TimeZoneInfo x) => x.BaseUtcOffset.TotalMinutes == (double)utcOffset);
					if (timeZoneInfo != null)
					{
						dateTime = TimeZoneInfo.ConvertTimeFromUtc(dateTime, TimeZoneInfo.FindSystemTimeZoneById(timeZoneInfo.Id));
						m_txtValue.Text = dateTime.ToLongDateString() + " " + dateTime.ToLongTimeString();
					}
					break;
				}
				case UIPropertyStringType.ControllerVersion:
					m_txtValue.Text = ICULanDevice.GetHWVersion(device, isControllerBoard: true);
					break;
				case UIPropertyStringType.PowerVersion:
					m_txtValue.Text = ICULanDevice.GetHWVersion(device, isControllerBoard: false);
					break;
				case UIPropertyStringType.NFC1_HW:
					m_txtValue.Text = ICULanDevice.GetNFCVersion(device, 1, software: false);
					break;
				case UIPropertyStringType.NFC1_SW:
					m_txtValue.Text = ICULanDevice.GetNFCVersion(device, 1);
					break;
				case UIPropertyStringType.NFC2_HW:
					m_txtValue.Text = ICULanDevice.GetNFCVersion(device, 2, software: false);
					break;
				case UIPropertyStringType.NFC2_SW:
					m_txtValue.Text = ICULanDevice.GetNFCVersion(device, 2);
					break;
				case UIPropertyStringType.Features:
				{
					string featureTextLong = IWSFirmwareFeatures.GetFeatureTextLong((IWSFirmwareFeatures.Features)Convert.ToUInt32(m_objCurrentValue.Value), ((ICULanDevice)device).isAHP, ((ICULanDevice)device).isDC);
					featureTextLong = (string.IsNullOrEmpty(featureTextLong) ? "<none>" : featureTextLong);
					m_txtValue.Text = featureTextLong;
					break;
				}
				case UIPropertyStringType.Float:
				case UIPropertyStringType.Float2:
					if (!string.IsNullOrEmpty(m_objCurrentValue.Value.ToString()))
					{
						num = Convert.ToDouble(m_objCurrentValue.Value);
						m_txtValue.Text = num.ToString((m_eType == UIPropertyStringType.Float) ? "F1" : "F2");
					}
					break;
				case UIPropertyStringType.Float_0_001:
				case UIPropertyStringType.Float2_0_001:
					if (!string.IsNullOrEmpty(m_objCurrentValue.Value.ToString()))
					{
						num = Convert.ToDouble(m_objCurrentValue.Value) * 0.001;
						m_txtValue.Text = num.ToString((m_eType == UIPropertyStringType.Float_0_001) ? "F1" : "F2");
					}
					break;
				case UIPropertyStringType.Float_0_01:
				case UIPropertyStringType.Float2_0_01:
					if (!string.IsNullOrEmpty(m_objCurrentValue.Value.ToString()))
					{
						num = Convert.ToDouble(m_objCurrentValue.Value) * 0.01;
						m_txtValue.Text = num.ToString((m_eType == UIPropertyStringType.Float_0_01) ? "F1" : "F2");
					}
					break;
				case UIPropertyStringType.PWM:
					if (!string.IsNullOrEmpty(m_objCurrentValue.Value.ToString()))
					{
						num = Convert.ToDouble(m_objCurrentValue.Value) * 0.01;
						m_txtValue.Text = string.Format("{0} ({1}A)", num.ToString("F1"), (num * 0.6).ToString("F1"));
					}
					break;
				case UIPropertyStringType.PublicKey:
				{
					string text = m_objCurrentValue.Value.ToString().ToLower();
					string text2 = string.Empty;
					for (int num3 = 0; num3 < text.Length; num3++)
					{
						if (num3 % 4 == 0 && num3 != 0)
						{
							text2 += " ";
						}
						text2 += text[num3];
					}
					m_txtValue.Font = UIPropertyBase.s_fntBaseLabel.WithScaledSize(1.1).WithWeight(FontWeight.Bold);
					m_txtValue.Text = text2;
					break;
				}
				case UIPropertyStringType.Float_0_1:
				case UIPropertyStringType.Float2_0_1:
					if (!string.IsNullOrEmpty(m_objCurrentValue.Value.ToString()))
					{
						num = Convert.ToDouble(m_objCurrentValue.Value) * 0.1;
						m_txtValue.Text = num.ToString((m_eType == UIPropertyStringType.Float_0_1) ? "F1" : "F2");
					}
					break;
				case UIPropertyStringType.Float_10:
				case UIPropertyStringType.Float2_10:
					if (!string.IsNullOrEmpty(m_objCurrentValue.Value.ToString()))
					{
						num = Convert.ToDouble(m_objCurrentValue.Value) * 10.0;
						m_txtValue.Text = num.ToString((m_eType == UIPropertyStringType.Float_10) ? "F1" : "F2");
					}
					break;
				default:
					m_txtValue.Text = m_objCurrentValue.Value.ToString();
					break;
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
		m_lblTitle.Sensitive = m_fEnabled;
	}

	protected override void OnValueChanged(ICUProperty prop)
	{
		RefreshDisplay();
	}

	public override void SetValue(object newValue)
	{
		m_txtValue.Text = newValue.ToString();
	}
}
