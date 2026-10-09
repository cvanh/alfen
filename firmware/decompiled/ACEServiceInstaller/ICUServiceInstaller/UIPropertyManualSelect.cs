using System;
using System.Collections.Generic;
using ICUNetwork;
using ICUSettings;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class UIPropertyManualSelect : UIPropertyBase
{
	private Label m_lblTitle;

	private ComboBox m_cmbValue;

	protected int m_nValueMask;

	protected bool m_fCaseSensitive = true;

	protected bool m_fForceChanged;

	private readonly string m_sCustomLabel = string.Empty;

	private Dictionary<string, string> m_lstCustomOptions;

	private string m_sInitialValue = string.Empty;

	public override bool IsChanged
	{
		get
		{
			if (!(GetValue().ToString() != m_sInitialValue))
			{
				return m_fForceChanged;
			}
			return true;
		}
	}

	public event EventHandler SelectionChanged;

	public UIPropertyManualSelect(PanelBase panelParent, Table table, int col, int line, string customLabel, Dictionary<string, string> customOptions, int nLeftMargin = 0, bool forceReadOnly = false, uint tooltipID = 0u)
		: base(panelParent, customLabel, tooltipID)
	{
		m_sCustomLabel = customLabel;
		m_edsParameter = null;
		m_lstCustomOptions = customOptions;
		Initialize(table, col, line, nLeftMargin);
	}

	public void SetCustomList(Dictionary<string, string> customOptions)
	{
		m_lstCustomOptions = customOptions;
		if (m_lstCustomOptions != null)
		{
			m_cmbValue.Items.Clear();
			foreach (KeyValuePair<string, string> lstCustomOption in m_lstCustomOptions)
			{
				m_cmbValue.Items.Add(lstCustomOption.Key, lstCustomOption.Value);
			}
		}
		RefreshDisplay();
	}

	public void Initialize(Table table, int col, int line, int nLeftMargin = 0)
	{
		m_cmbValue = new ComboBox
		{
			BackgroundColor = AppProperties.Color_Disabled
		};
		m_cmbValue.Font = UIPropertyBase.s_fntBase;
		if (m_edsParameter != null && m_edsParameter.Options != null)
		{
			m_lblTitle = new Label(m_edsParameter.Title);
			foreach (EDSParameterOption option in m_edsParameter.Options)
			{
				m_cmbValue.Items.Add(option.Value, option.Title);
			}
		}
		else
		{
			m_lblTitle = new Label(m_sCustomLabel);
			if (m_lstCustomOptions != null)
			{
				foreach (KeyValuePair<string, string> lstCustomOption in m_lstCustomOptions)
				{
					m_cmbValue.Items.Add(lstCustomOption.Key, lstCustomOption.Value);
				}
			}
		}
		m_cmbValue.SelectionChanged += OnSelectionChanged;
		SetWidgetSizes(m_lblTitle, m_imvChanged, m_cmbValue, m_btnTooltip);
		m_lblTitle.Font = UIPropertyBase.s_fntBaseLabel;
		m_lblTitle.TooltipText = TooltipID;
		table.Add(m_lblTitle, col, line, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, UIPropertyBase.s_marginHor + nLeftMargin, marginRight: Math.Max(UIPropertyBase.s_marginHorMax - nLeftMargin, 0), marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
		table.Add(m_cmbValue, col + 1, line, 1, 1, hexpand: true, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, UIPropertyBase.s_marginHor, marginRight: UIPropertyBase.s_marginHor, marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
		table.Add(m_imvChanged, col + 2, line, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Start, WidgetPlacement.Fill, 1.0, UIPropertyBase.s_marginVer, 1.0, UIPropertyBase.s_marginVer);
		table.Add(m_btnTooltip, col + 3, line, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Start, WidgetPlacement.Fill, 1.0, UIPropertyBase.s_marginVer, 1.0, UIPropertyBase.s_marginVer);
	}

	public override void SetTooltip(string text)
	{
		m_lblTitle.TooltipText = text;
	}

	public override void SetVisible(bool fVisible)
	{
		m_cmbValue.Visible = fVisible;
		m_lblTitle.Visible = fVisible;
		m_imvChanged.Visible = fVisible;
		m_btnTooltip.Visible = fVisible && !string.IsNullOrEmpty(m_tooltipText);
	}

	public void SetCaseSensitive(bool caseSensitive)
	{
		m_fCaseSensitive = caseSensitive;
	}

	public override void SetMinWidth(int minWidth)
	{
		m_cmbValue.MinWidth = minWidth;
	}

	private void OnSelectionChanged(object sender, EventArgs e)
	{
		SelectionChanged?.Invoke(this, EventArgs.Empty);
		RefreshDisplay();
	}

	protected override void OnDeviceChange(ICUDevice device)
	{
		if (device == null)
		{
			m_cmbValue.SelectedIndex = -1;
			m_lblTitle.Sensitive = false;
			m_cmbValue.Sensitive = false;
		}
		else
		{
			m_lblTitle.Sensitive = true;
		}
	}

	public override void OnRefreshDisplay(ICUDevice device)
	{
		if (m_cmbValue != null)
		{
			if (device != null)
			{
				m_imvChanged.Image = (IsChanged ? UIPropertyBase.s_imgChanged : UIPropertyBase.s_imgEmpty);
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
			m_cmbValue.Sensitive = !flag;
			if (flag)
			{
				m_cmbValue.Font = UIPropertyBase.s_fntBase.WithWeight(FontWeight.Normal);
			}
			else
			{
				m_cmbValue.Font = UIPropertyBase.s_fntBase.WithWeight(FontWeight.Semibold);
			}
			m_lblTitle.Sensitive = m_fEnabled;
		}
	}

	public override object GetValue()
	{
		if (m_cmbValue.SelectedItem == null)
		{
			return string.Empty;
		}
		return m_cmbValue.SelectedItem.ToString();
	}

	public void SetInitialValue(object newValue)
	{
		if (newValue != null)
		{
			if (string.IsNullOrEmpty(m_sInitialValue))
			{
				SelectOption(newValue.ToString());
			}
			m_sInitialValue = newValue.ToString();
			RefreshDisplay();
		}
	}

	public override void SetValue(object newValue)
	{
		if (newValue != null)
		{
			m_cmbValue.SelectedItem = newValue.ToString();
		}
	}

	public void CommitChange()
	{
		SetInitialValue(GetSelectedOption());
	}

	public void RevertChange()
	{
		SelectOption(m_sInitialValue);
	}

	public void ForceChanged(bool changed)
	{
		m_fForceChanged = changed;
	}

	public string GetSelectedOption()
	{
		if (m_cmbValue.SelectedItem == null)
		{
			return string.Empty;
		}
		return m_cmbValue.SelectedItem.ToString();
	}

	public void SelectOption(string newSelection)
	{
		m_cmbValue.SelectedItem = newSelection;
	}

	protected override void OnClearValue()
	{
		m_cmbValue.SelectedIndex = 0;
	}
}
