using System;
using System.Collections.Generic;
using System.Linq;
using ICUNetwork;
using ICUSettings;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class UIPropertySelect : UIPropertyBase
{
	private Label m_lblTitle;

	private ComboBox m_cmbValue;

	protected int m_nValueMask;

	protected bool m_fCaseSensitive = true;

	private readonly string m_sCustomLabel = string.Empty;

	private Dictionary<string, string> m_lstCustomOptions;

	private Dictionary<string, string> m_lstHideOptions;

	public string Name => m_lblTitle.Text;

	public UIPropertySelect(PanelBase panelParent, Table table, int col, int line, ushort Id, byte subId = 0, int nValueMask = 0, int nLeftMargin = 0, uint tooltipID = 0u)
		: base(panelParent, Id, subId, tooltipID)
	{
		m_nValueMask = nValueMask;
		Initialize(table, col, line, nLeftMargin);
	}

	public UIPropertySelect(PanelBase panelParent, Table table, int col, int line, string customLabel, Dictionary<string, string> customOptions, int nValueMask, int nLeftMargin = 0, bool forceReadOnly = false, uint tooltipID = 0u)
		: base(panelParent, customLabel, tooltipID)
	{
		m_nValueMask = nValueMask;
		if (!string.IsNullOrEmpty(customLabel))
		{
			m_sCustomLabel = customLabel;
		}
		else if (m_edsParameter != null)
		{
			m_sCustomLabel = m_edsParameter.Title;
		}
		m_edsParameter = null;
		m_lstCustomOptions = customOptions;
		Initialize(table, col, line, nLeftMargin);
	}

	public UIPropertySelect(PanelBase panelParent, Table table, int col, int line, ushort Id, byte subId, ushort baseId, string customLabel, byte baseSubId = 0, int nLeftMargin = 0, uint tooltipID = 0u)
		: base(panelParent, Id, subId, tooltipID)
	{
		if (baseId != 0 || baseSubId != 0)
		{
			m_edsParameter = DataSheet.FindParameter(baseId, baseSubId);
		}
		else
		{
			m_edsParameter = DataSheet.FindParameter(Id, SubId);
		}
		if (!string.IsNullOrEmpty(customLabel))
		{
			m_sCustomLabel = customLabel;
		}
		else if (m_edsParameter != null)
		{
			m_sCustomLabel = m_edsParameter.Title;
		}
		Initialize(table, col, line, nLeftMargin);
	}

	public UIPropertySelect(PanelBase panelParent, Table table, int col, int line, ushort Id, byte subId, string customLabel, Dictionary<string, string> customOptions, int nValueMask = 0, int nLeftMargin = 0, uint tooltipID = 0u, bool hideItems = false)
		: base(panelParent, Id, subId, tooltipID)
	{
		m_nValueMask = nValueMask;
		if (!string.IsNullOrEmpty(customLabel))
		{
			m_sCustomLabel = customLabel;
		}
		else if (m_edsParameter != null)
		{
			m_sCustomLabel = m_edsParameter.Title;
		}
		m_edsParameter = null;
		if (hideItems)
		{
			m_lstHideOptions = customOptions;
			m_lstCustomOptions = null;
			m_edsParameter = DataSheet.FindParameter(Id, SubId);
		}
		else
		{
			m_lstCustomOptions = customOptions;
		}
		Initialize(table, col, line, nLeftMargin);
	}

	public UIPropertySelect(PanelBase panelParent, Table table, int col, int line, ushort Id, byte subId, int nLeftMargin = 0, uint tooltipID = 0u)
		: base(panelParent, Id, subId, tooltipID)
	{
		m_edsParameter = DataSheet.FindParameter(Id, subId);
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
		if (string.IsNullOrEmpty(m_sCustomLabel) && !string.IsNullOrEmpty(m_edsParameter.Title))
		{
			m_lblTitle = new Label(m_edsParameter.Title);
		}
		else
		{
			m_lblTitle = new Label(m_sCustomLabel);
		}
		if (m_lstCustomOptions != null)
		{
			foreach (KeyValuePair<string, string> lstCustomOption in m_lstCustomOptions)
			{
				m_cmbValue.Items.Add(lstCustomOption.Key, lstCustomOption.Value);
			}
		}
		else if (m_edsParameter?.Options != null)
		{
			foreach (EDSParameterOption option in m_edsParameter.Options)
			{
				if (m_lstHideOptions == null)
				{
					m_cmbValue.Items.Add(option.Value, option.Title);
				}
				else if (!m_lstHideOptions.ContainsKey(option.Value))
				{
					m_cmbValue.Items.Add(option.Value, option.Title);
				}
			}
			m_lstHideOptions = null;
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
		if (m_cmbValue.SelectedItem == null || m_objCurrentValue == null || m_fReadOnly || m_fForceReadOnly)
		{
			return;
		}
		string text = GetValue().ToString();
		if (!m_fCaseSensitive)
		{
			if (m_objCurrentValue.Value.ToString().ToLowerInvariant().Trim() != text.ToString().ToLowerInvariant().Trim())
			{
				m_objCurrentValue.Value = text;
			}
		}
		else
		{
			m_objCurrentValue.Value = text;
		}
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
			RefreshDisplay();
		}
	}

	public override void OnRefreshDisplay(ICUDevice device)
	{
		if (m_cmbValue == null)
		{
			return;
		}
		if (device != null && m_objCurrentValue != null && m_objCurrentValue.Value != null)
		{
			string value = m_objCurrentValue.Value.ToString();
			if (m_nValueMask != 0)
			{
				int num = Convert.ToInt32(m_objCurrentValue.Value);
				value = (num & m_nValueMask).ToString();
			}
			m_cmbValue.SelectedItem = value;
			if (!m_fCaseSensitive)
			{
				object obj = m_cmbValue.Items.FirstOrDefault((object a) => a.ToString().ToLowerInvariant().Trim() == value.ToLowerInvariant().Trim());
				if (obj != null)
				{
					m_cmbValue.SelectedItem = obj.ToString();
				}
			}
		}
		if (m_objCurrentValue != null)
		{
			m_fReadOnly = m_objCurrentValue.ReadOnly;
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

	protected override void OnValueChanged(ICUProperty prop)
	{
		RefreshDisplay();
		m_imvChanged.Image = ((m_cmbValue.Sensitive && prop.IsChanged) ? UIPropertyBase.s_imgChanged : UIPropertyBase.s_imgEmpty);
	}

	public override object GetValue()
	{
		if (m_nValueMask != 0)
		{
			int num = Convert.ToInt32(m_objCurrentValue.Value);
			int num2 = Convert.ToInt32(m_cmbValue.SelectedItem);
			num &= ~m_nValueMask;
			return (num | num2).ToString();
		}
		if (m_cmbValue.SelectedItem == null)
		{
			return "0";
		}
		return m_cmbValue.SelectedItem.ToString();
	}

	public override void SetValue(object newValue)
	{
		if (m_nValueMask != 0)
		{
			int num = Convert.ToInt32(newValue) & m_nValueMask;
			m_cmbValue.SelectedItem = num.ToString();
		}
		else
		{
			m_cmbValue.SelectedItem = newValue.ToString();
		}
	}

	public string GetSelectedOption()
	{
		if (m_cmbValue.SelectedItem == null)
		{
			return string.Empty;
		}
		return m_cmbValue.SelectedItem.ToString();
	}

	public string GetSelectedText()
	{
		if (m_cmbValue.SelectedItem == null)
		{
			return string.Empty;
		}
		return m_cmbValue.SelectedText;
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
