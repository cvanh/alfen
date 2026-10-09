using System;
using ICUNetwork;
using Xwt;

namespace ICUServiceInstaller;

public class UICheckbox : UIPropertyBase
{
	protected Label m_lblTitle;

	protected CheckBox m_chkValue;

	protected int m_nValueMask;

	protected bool m_fCustomValue;

	protected bool m_fChanged;

	public bool CustomValue
	{
		get
		{
			return m_fCustomValue;
		}
		set
		{
			m_fCustomValue = value;
			m_chkValue.State = (m_fCustomValue ? CheckBoxState.On : CheckBoxState.Off);
			FireChange();
			RefreshDisplay();
		}
	}

	public override bool IsChanged => m_chkValue.State == CheckBoxState.On != m_fCustomValue;

	public bool IsChecked => m_chkValue.Active;

	public UICheckbox(PanelBase panelParent, Table table, int col, int line, string labelText, int nLeftMargin = 0, uint tooltipID = 0u)
		: base(panelParent, labelText, tooltipID)
	{
		Initialize(table, col, line, nLeftMargin);
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
		m_chkValue = new CheckBox
		{
			BackgroundColor = AppProperties.Color_Disabled
		};
		m_chkValue.Clicked += OnChkClicked;
		m_chkValue.MinHeight = UIPropertyBase.s_propertyMinHeight;
		m_chkValue.State = (m_fCustomValue ? CheckBoxState.On : CheckBoxState.Off);
		SetWidgetSizes(m_lblTitle, m_imvChanged, m_chkValue, m_btnTooltip);
		m_lblTitle.TooltipText = TooltipID;
		table.Add(m_lblTitle, col, line, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, UIPropertyBase.s_marginHor + nLeftMargin, marginRight: Math.Max(UIPropertyBase.s_marginHorMax - nLeftMargin, 0), marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
		table.Add(m_chkValue, col + 1, line, 1, 1, hexpand: true, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.End, UIPropertyBase.s_marginHor, marginRight: UIPropertyBase.s_marginHor, marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
		table.Add(m_imvChanged, col + 2, line, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Start, WidgetPlacement.Fill, 1.0, UIPropertyBase.s_marginVer, 1.0, UIPropertyBase.s_marginVer);
		table.Add(m_btnTooltip, col + 3, line, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Start, WidgetPlacement.Fill, 1.0, UIPropertyBase.s_marginVer, 1.0, UIPropertyBase.s_marginVer);
	}

	public override void SetVisible(bool fVisible)
	{
		m_chkValue.Visible = fVisible;
		m_lblTitle.Visible = fVisible;
		m_imvChanged.Visible = fVisible;
		m_btnTooltip.Visible = fVisible && !string.IsNullOrEmpty(m_tooltipText);
	}

	public override void SetEnable(bool enable)
	{
		m_chkValue.Sensitive = enable;
		m_lblTitle.Sensitive = enable;
		m_imvChanged.Sensitive = enable;
		m_btnTooltip.Sensitive = enable && !string.IsNullOrEmpty(m_tooltipText);
	}

	public override void SetTooltip(string text)
	{
		m_lblTitle.TooltipText = text;
	}

	private void OnChkClicked(object sender, EventArgs e)
	{
		m_imvChanged.Image = (IsChanged ? UIPropertyBase.s_imgChanged : UIPropertyBase.s_imgEmpty);
		FireChange();
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
		m_imvChanged.Image = (IsChanged ? UIPropertyBase.s_imgChanged : UIPropertyBase.s_imgEmpty);
		m_chkValue.BackgroundColor = (m_fReadOnly ? m_lblTitle.BackgroundColor : AppProperties.Color_Disabled);
		bool flag = m_fReadOnly;
		if (m_fForceReadOnly)
		{
			flag = true;
		}
		if (!m_fEnabled)
		{
			flag = true;
		}
		m_chkValue.Sensitive = !flag;
	}

	protected override void OnClearValue()
	{
		m_chkValue.State = CheckBoxState.Off;
		m_fCustomValue = false;
		RefreshDisplay();
	}

	public override object GetValue()
	{
		return m_chkValue.State == CheckBoxState.On;
	}

	public override void SetValue(object newValue)
	{
		bool flag = Convert.ToBoolean(newValue);
		m_chkValue.State = (flag ? CheckBoxState.On : CheckBoxState.Off);
		FireChange();
	}
}
