using System;
using ICUNetwork;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class UIPropertyOkNokNa : UIPropertyBase
{
	protected static int s_nButtonWidth = AppProperties.ButtonWidth;

	protected static int s_nButtonHeight = AppProperties.ButtonHeight;

	protected Label m_lblTitle;

	protected string m_sCustomId;

	protected ToggleButton m_tbnOk;

	protected ToggleButton m_tbnNok;

	protected ToggleButton m_tbnNA;

	protected EQuestion m_eDefaultValue = EQuestion.NA;

	protected EQuestion m_eValue;

	protected bool m_fChanged;

	protected EQuestionType m_eType;

	public UIPropertyOkNokNa(PanelBase panelParent, Table table, int col, int line, string customId, string labelText, EQuestion eDefaultValue = EQuestion.NA, int nLeftMargin = 0, EQuestionType eType = EQuestionType.Normal)
		: base(panelParent, 0, 0)
	{
		m_eDefaultValue = eDefaultValue;
		m_sLabelText = labelText;
		m_eType = eType;
		m_lblTitle = new Label(LabelText);
		m_lblTitle.Font = UIPropertyBase.s_fntBaseLabel;
		ButtonStyle style = ButtonStyle.Flat;
		m_tbnOk = new ToggleButton("OK")
		{
			BackgroundColor = AppProperties.Color_Disabled,
			MinWidth = s_nButtonWidth,
			MinHeight = s_nButtonHeight,
			Style = style
		};
		m_tbnNok = new ToggleButton("NOK")
		{
			BackgroundColor = AppProperties.Color_Disabled,
			MinWidth = s_nButtonWidth,
			MinHeight = s_nButtonHeight,
			Style = style
		};
		m_tbnNA = new ToggleButton("N.V.T.")
		{
			BackgroundColor = AppProperties.Color_Disabled,
			MinWidth = s_nButtonWidth,
			MinHeight = s_nButtonHeight,
			Style = style
		};
		if (m_eType == EQuestionType.Important)
		{
			m_tbnOk.BackgroundColor = Colors.Orange;
			m_tbnNok.BackgroundColor = Colors.Orange;
			m_tbnNA.BackgroundColor = Colors.Orange;
		}
		m_tbnOk.Clicked += OnBtnClicked;
		m_tbnNok.Clicked += OnBtnClicked;
		m_tbnNA.Clicked += OnBtnClicked;
		HBox hBox = new HBox();
		hBox.MinHeight = UIPropertyBase.s_propertyMinHeight;
		hBox.PackStart(m_tbnOk, expand: false);
		hBox.PackStart(m_tbnNok, expand: false);
		hBox.PackStart(m_tbnNA, expand: false);
		table.Add(m_lblTitle, col, line, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, UIPropertyBase.s_marginHor + nLeftMargin, marginRight: Math.Max(UIPropertyBase.s_marginHorMax - nLeftMargin, 0), marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
		table.Add(hBox, col + 1, line, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.End, UIPropertyBase.s_marginHor, marginRight: UIPropertyBase.s_marginHor, marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
		table.Add(m_imvChanged, col + 2, line, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Start, WidgetPlacement.Fill, 1.0, UIPropertyBase.s_marginVer, 1.0, UIPropertyBase.s_marginVer);
	}

	public override void SetTooltip(string text)
	{
		m_lblTitle.TooltipText = text;
	}

	public override void SetVisible(bool fVisible)
	{
		m_tbnOk.Visible = fVisible;
		m_tbnNok.Visible = fVisible;
		m_tbnNA.Visible = fVisible;
		m_lblTitle.Visible = fVisible;
		m_imvChanged.Visible = fVisible;
	}

	private void OnBtnClicked(object sender, EventArgs e)
	{
		m_eValue = EQuestion.OK;
		if (sender == m_tbnNok)
		{
			m_eValue = EQuestion.NOK;
		}
		else if (sender == m_tbnNA)
		{
			m_eValue = EQuestion.NA;
		}
		m_tbnOk.Active = m_eValue == EQuestion.OK;
		m_tbnNok.Active = m_eValue == EQuestion.NOK;
		m_tbnNA.Active = m_eValue == EQuestion.NA;
	}

	protected override void OnDeviceChange(ICUDevice device)
	{
		if (device == null)
		{
			m_tbnOk.Active = false;
			m_tbnNok.Active = false;
			m_tbnNA.Active = false;
			m_fChanged = false;
			m_lblTitle.Sensitive = false;
		}
		else
		{
			m_fReadOnly = false;
			m_fForceReadOnly = false;
			m_eValue = m_eDefaultValue;
			m_lblTitle.Sensitive = true;
			RefreshDisplay();
		}
	}

	public override void OnRefreshDisplay(ICUDevice device)
	{
		if (m_tbnOk != null)
		{
			if (device != null)
			{
				m_tbnOk.Active = m_eValue == EQuestion.OK;
				m_tbnNok.Active = m_eValue == EQuestion.NOK;
				m_tbnNA.Active = m_eValue == EQuestion.NA;
			}
			bool flag = m_fReadOnly;
			if (m_fForceReadOnly)
			{
				flag = true;
			}
			if (!m_fEnabled)
			{
				flag = true;
			}
			m_lblTitle.Sensitive = m_fEnabled;
			m_tbnOk.Sensitive = !flag;
			m_tbnNok.Sensitive = !flag;
			m_tbnNA.Sensitive = !flag;
		}
	}

	public override void OnChanged()
	{
		m_imvChanged.Image = ((m_tbnOk.Sensitive && m_fChanged) ? UIPropertyBase.s_imgChanged : UIPropertyBase.s_imgEmpty);
	}

	public override object GetValue()
	{
		return m_eValue;
	}

	public override void SetValue(object newValue)
	{
		m_eValue = (EQuestion)newValue;
		RefreshDisplay();
	}
}
