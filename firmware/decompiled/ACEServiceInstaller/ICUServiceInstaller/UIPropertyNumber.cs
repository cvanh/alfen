using System;
using System.Reflection;
using ICUNetwork;
using ICUSettings;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class UIPropertyNumber : UIPropertyBase
{
	protected string m_sDevicePropertyName = "";

	protected Label m_lblTitle;

	protected SpinButton m_spnValue;

	protected ulong m_nMaxLen = 256uL;

	protected string m_sCustomLabel = string.Empty;

	protected double m_dFactor = 1.0;

	protected int m_nForceDigits = -1;

	protected bool m_bForceMinMax;

	public bool m_isFocused { get; private set; }

	public UIPropertyNumber(PanelBase panelParent, Table table, int col, int line, string labelText, string devicePropertyName, int nLeftMargin = 0, bool forceReadOnly = false, int digits = 0, uint tooltipID = 0u)
		: base(panelParent, labelText, tooltipID)
	{
		m_nForceDigits = digits;
		m_sDevicePropertyName = devicePropertyName;
		m_fForceReadOnly = forceReadOnly;
		Initialize(table, col, line, nLeftMargin);
	}

	public UIPropertyNumber(PanelBase panelParent, Table table, int col, int line, ushort Id, byte subId = 0, int digits = 0, int nLeftMargin = 0, bool forceReadOnly = false, uint tooltipID = 0u)
		: base(panelParent, Id, subId, tooltipID)
	{
		m_nForceDigits = digits;
		m_fForceReadOnly = forceReadOnly;
		Initialize(table, col, line, nLeftMargin);
	}

	public UIPropertyNumber(PanelBase panelParent, Table table, int col, int line, ushort Id, byte subId = 0, int digits = 0, ushort baseId = 0, byte baseSubId = 0, int nLeftMargin = 0, bool forceReadOnly = false, uint tooltipID = 0u)
		: base(panelParent, Id, subId, tooltipID)
	{
		m_nForceDigits = digits;
		if (baseId != 0 || baseSubId != 0)
		{
			m_edsParameter = DataSheet.FindParameter(baseId, baseSubId);
		}
		m_fForceReadOnly = forceReadOnly;
		Initialize(table, col, line, nLeftMargin);
	}

	public UIPropertyNumber(PanelBase panelParent, Table table, int col, int line, ushort Id, byte subId = 0, int digits = 0, string customLabel = "", double factor = 1.0, int nLeftMargin = 0, bool forceReadOnly = false, uint tooltipID = 0u)
		: base(panelParent, Id, subId, tooltipID)
	{
		m_fForceReadOnly = forceReadOnly;
		m_sCustomLabel = customLabel;
		m_dFactor = factor;
		m_nForceDigits = digits;
		Initialize(table, col, line, nLeftMargin);
	}

	private void Initialize(Table table, int col, int line, int nLeftMargin = 0)
	{
		if (!string.IsNullOrEmpty(m_sCustomLabel))
		{
			m_lblTitle = new Label(m_sCustomLabel);
		}
		else
		{
			m_lblTitle = new Label(LabelText);
		}
		m_lblTitle.Font = UIPropertyBase.s_fntBaseLabel;
		m_spnValue = new SpinButton
		{
			BackgroundColor = m_lblTitle.BackgroundColor
		};
		m_spnValue.Font = UIPropertyBase.s_fntBase;
		m_spnValue.Style = ButtonStyle.Normal;
		m_spnValue.ValueChanged += OnSpinValueChanged;
		m_spnValue.GotFocus += OnValueGotFocus;
		m_spnValue.LostFocus += OnValueLostFocus;
		m_spnValue.MinHeight = UIPropertyBase.s_propertyMinHeight;
		SetWidgetSizes(m_lblTitle, m_imvChanged, m_spnValue, m_btnTooltip);
		m_lblTitle.TooltipText = TooltipID;
		table.Add(m_lblTitle, col, line, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, UIPropertyBase.s_marginHor + nLeftMargin, marginRight: Math.Max(UIPropertyBase.s_marginHorMax - nLeftMargin, 0), marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
		table.Add(m_spnValue, col + 1, line, 1, 1, hexpand: true, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, UIPropertyBase.s_marginHor, marginRight: UIPropertyBase.s_marginHor, marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
		table.Add(m_imvChanged, col + 2, line, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Start, WidgetPlacement.Fill, 1.0, UIPropertyBase.s_marginVer, 1.0, UIPropertyBase.s_marginVer);
		table.Add(m_btnTooltip, col + 3, line, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Start, WidgetPlacement.Fill, 1.0, UIPropertyBase.s_marginVer, 1.0, UIPropertyBase.s_marginVer);
		if (m_edsParameter != null)
		{
			m_nMaxLen = m_edsParameter.MaxLength;
		}
	}

	public override void SetTooltip(string text)
	{
		m_lblTitle.TooltipText = text;
	}

	public void SetLabelText(string text)
	{
		m_lblTitle.Text = text;
	}

	public override void SetVisible(bool fVisible)
	{
		m_spnValue.Visible = fVisible;
		m_lblTitle.Visible = fVisible;
		m_imvChanged.Visible = fVisible;
		m_btnTooltip.Visible = fVisible && !string.IsNullOrEmpty(m_tooltipText);
	}

	public override void SetMinWidth(int minWidth)
	{
	}

	private void OnValueGotFocus(object sender, EventArgs e)
	{
		m_isFocused = true;
	}

	private void OnValueLostFocus(object sender, EventArgs e)
	{
		m_isFocused = false;
	}

	private void OnSpinValueChanged(object sender, EventArgs e)
	{
		if (m_objCurrentValue != null && !m_fReadOnly && !m_fForceReadOnly)
		{
			m_objCurrentValue.Value = m_spnValue.Value / m_dFactor;
		}
	}

	protected override void OnDeviceChange(ICUDevice device)
	{
		if (device == null)
		{
			m_spnValue.Value = 0.0;
			m_spnValue.Sensitive = false;
			m_lblTitle.Sensitive = false;
		}
		else
		{
			m_spnValue.Sensitive = true;
			m_lblTitle.Sensitive = true;
			RefreshDisplay();
		}
	}

	public override void OnRefreshDisplay(ICUDevice device)
	{
		m_fReadOnly = true;
		if (m_nForceDigits != -1)
		{
			m_spnValue.Digits = m_nForceDigits;
		}
		if (m_objCurrentValue != null && m_objCurrentValue.Value != null)
		{
			if (m_fUnknownTitle)
			{
				m_lblTitle.Text = m_objCurrentValue.Title;
			}
			if (!m_bForceMinMax)
			{
				switch (m_objCurrentValue.DataType)
				{
				case SDT.UNSIGNED8:
					SetValueMinMax(0.0, 255.0, 1.0, 0, force: false);
					break;
				case SDT.UNSIGNED16:
					SetValueMinMax(0.0, 65535.0, 1.0, 0, force: false);
					break;
				case SDT.UNSIGNED32:
					SetValueMinMax(0.0, 4294967295.0, 1.0, 0, force: false);
					break;
				case SDT.UNSIGNED64:
					SetValueMinMax(0.0, 1.8446744073709552E+19, 1.0, 0, force: false);
					break;
				case SDT.INTEGER8:
					SetValueMinMax(-128.0, 127.0, 1.0, 0, force: false);
					break;
				case SDT.INTEGER16:
					SetValueMinMax(-32768.0, 32767.0, 1.0, 0, force: false);
					break;
				case SDT.INTEGER32:
					SetValueMinMax(-2147483648.0, 2147483647.0, 1.0, 0, force: false);
					break;
				case SDT.INTEGER64:
					SetValueMinMax(-9.223372036854776E+18, 9.223372036854776E+18, 1.0, 0, force: false);
					break;
				case SDT.REAL32:
					SetValueMinMax(-3.4028234663852886E+38, 3.4028234663852886E+38, 0.001, 3, force: false);
					break;
				}
			}
			m_spnValue.Value = Convert.ToDouble(m_objCurrentValue.Value) * m_dFactor;
		}
		else if (!string.IsNullOrEmpty(m_sDevicePropertyName) && device != null)
		{
			PropertyInfo property = device.GetType().GetProperty(m_sDevicePropertyName);
			if (property != null)
			{
				m_spnValue.Value = Convert.ToDouble(property.GetValue(device, null)) * m_dFactor;
			}
		}
		if (m_objCurrentValue != null)
		{
			m_fReadOnly = m_objCurrentValue.ReadOnly;
			if (m_objCurrentValue.MaxLength != 0)
			{
				m_spnValue.MaximumValue = m_objCurrentValue.MaxLength;
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
		m_spnValue.BackgroundColor = (flag ? m_lblTitle.BackgroundColor : AppProperties.Color_Disabled);
		m_spnValue.Sensitive = !flag;
		if (flag)
		{
			m_spnValue.Font = UIPropertyBase.s_fntBase.WithWeight(FontWeight.Normal);
		}
		else
		{
			m_spnValue.Font = UIPropertyBase.s_fntBase.WithWeight(FontWeight.Semibold);
		}
		m_lblTitle.Sensitive = m_fEnabled;
	}

	protected override void OnValueChanged(ICUProperty prop)
	{
		RefreshDisplay();
		m_imvChanged.Image = ((!m_fReadOnly && prop.IsChanged) ? UIPropertyBase.s_imgChanged : UIPropertyBase.s_imgEmpty);
	}

	public void SetValueMinMax(double minValue, double maxValue, double dStep = 1.0, int digits = 0, bool force = true)
	{
		m_bForceMinMax = force;
		m_spnValue.MinimumValue = minValue;
		m_spnValue.MaximumValue = maxValue;
		if (m_nForceDigits >= 0)
		{
			m_spnValue.IncrementValue = 1.0 / Math.Pow(10.0, m_nForceDigits);
			m_spnValue.Digits = m_nForceDigits;
		}
		else
		{
			m_spnValue.Digits = digits;
			m_spnValue.IncrementValue = dStep;
		}
	}

	public override object GetValue()
	{
		return m_spnValue.Value;
	}

	public override void SetValue(object newValue)
	{
		m_spnValue.Value = Convert.ToDouble(newValue);
	}

	protected override void OnClearValue()
	{
	}

	public override void SetEnable(bool fEnable)
	{
		base.SetEnable(fEnable);
		m_spnValue.Sensitive = fEnable;
	}
}
