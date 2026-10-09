using System;
using ICUNetwork;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class UIPassword : UIPropertyBase
{
	protected Label m_lblTitle;

	protected PasswordEntry m_txtValue;

	protected int m_nMaxLen = 256;

	private static readonly string s_sDefaultValue = "                ";

	protected string m_sCustomValue = s_sDefaultValue;

	protected bool m_fChanged;

	protected bool m_fIsUsingDefaultPassword = true;

	public string CustomValue
	{
		get
		{
			return m_sCustomValue;
		}
		set
		{
			m_sCustomValue = value;
			m_txtValue.Password = m_sCustomValue;
			FireChange();
			RefreshDisplay();
		}
	}

	public override bool IsChanged => m_txtValue.Password != m_sCustomValue;

	public UIPassword(PanelBase panelParent, Table table, int col, int line, string labelText, int nLeftMargin = 0, uint tooltipID = 0u)
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
		m_txtValue = new PasswordEntry
		{
			Password = s_sDefaultValue,
			BackgroundColor = m_lblTitle.BackgroundColor
		};
		m_txtValue.Font = UIPropertyBase.s_fntBase;
		m_txtValue.Changed += OnTxtChanged;
		m_txtValue.KeyPressed += OnKeyPressed;
		m_txtValue.MinHeight = UIPropertyBase.s_propertyMinHeight;
		SetWidgetSizes(m_lblTitle, m_imvChanged, m_txtValue, m_btnTooltip);
		m_lblTitle.TooltipText = TooltipID;
		table.Add(m_lblTitle, col, line, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, UIPropertyBase.s_marginHor + nLeftMargin, marginRight: Math.Max(UIPropertyBase.s_marginHorMax - nLeftMargin, 0), marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
		table.Add(m_txtValue, col + 1, line, 1, 1, hexpand: true, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, UIPropertyBase.s_marginHor, marginRight: UIPropertyBase.s_marginHor, marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
		table.Add(m_imvChanged, col + 2, line, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Start, WidgetPlacement.Fill, 1.0, UIPropertyBase.s_marginVer, 1.0, UIPropertyBase.s_marginVer);
		table.Add(m_btnTooltip, col + 3, line, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Start, WidgetPlacement.Fill, 1.0, UIPropertyBase.s_marginVer, 1.0, UIPropertyBase.s_marginVer);
	}

	private void OnKeyPressed(object sender, KeyEventArgs e)
	{
		if (m_fIsUsingDefaultPassword)
		{
			m_fIsUsingDefaultPassword = false;
			m_txtValue.Password = "";
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

	public void ChangesAreSaved()
	{
		CustomValue = m_txtValue.Password;
	}

	public override void SetMinWidth(int minWidth)
	{
		m_txtValue.MinWidth = minWidth;
	}

	private void OnTxtChanged(object sender, EventArgs e)
	{
		m_imvChanged.Image = (IsChanged ? UIPropertyBase.s_imgChanged : UIPropertyBase.s_imgEmpty);
		FireChange();
	}

	protected override void OnDeviceChange(ICUDevice device)
	{
		if (device == null)
		{
			m_txtValue.Password = s_sDefaultValue;
			m_txtValue.Sensitive = false;
			m_lblTitle.Sensitive = false;
		}
		else
		{
			m_txtValue.Sensitive = true;
			m_lblTitle.Sensitive = true;
			RefreshDisplay();
		}
		FireChange();
	}

	public override void OnRefreshDisplay(ICUDevice device)
	{
		m_imvChanged.Image = (IsChanged ? UIPropertyBase.s_imgChanged : UIPropertyBase.s_imgEmpty);
		m_txtValue.BackgroundColor = (m_fReadOnly ? m_lblTitle.BackgroundColor : AppProperties.Color_Disabled);
		m_txtValue.Sensitive = m_fEnabled && !m_fReadOnly;
		m_lblTitle.Sensitive = m_fEnabled;
		if (m_fReadOnly)
		{
			m_txtValue.Font = UIPropertyBase.s_fntBase.WithWeight(FontWeight.Normal);
		}
		else
		{
			m_txtValue.Font = UIPropertyBase.s_fntBase.WithWeight(FontWeight.Semibold);
		}
	}

	protected override void OnClearValue()
	{
		m_txtValue.Password = s_sDefaultValue;
		m_sCustomValue = s_sDefaultValue;
		m_fIsUsingDefaultPassword = true;
		RefreshDisplay();
	}

	public override object GetValue()
	{
		if (!m_fEnabled)
		{
			return string.Empty;
		}
		return m_txtValue.Password;
	}

	public override void SetValue(object newValue)
	{
		m_txtValue.Password = newValue.ToString();
		FireChange();
	}
}
