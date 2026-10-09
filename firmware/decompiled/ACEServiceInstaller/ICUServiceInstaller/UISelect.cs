using System;
using System.Collections.Generic;
using System.Linq;
using ICUNetwork;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class UISelect : UIPropertyBase
{
	private Label m_lblTitle;

	private ComboBox m_cmbValue;

	private string m_sCustomValue = string.Empty;

	private Dictionary<string, string> m_lstCustomOptions;

	private bool m_fFireChangedEvent = true;

	public string CustomValue
	{
		get
		{
			return m_sCustomValue;
		}
		set
		{
			m_sCustomValue = value;
			m_cmbValue.SelectedItem = m_sCustomValue;
			FireChange();
			RefreshDisplay();
		}
	}

	public override bool IsChanged
	{
		get
		{
			if (m_cmbValue.SelectedItem != null)
			{
				return m_cmbValue.SelectedItem.ToString() != m_sCustomValue;
			}
			return false;
		}
	}

	public UISelect(PanelBase panelParent, Table table, int col, int line, string labelText, Dictionary<string, string> customOptions, int nLeftMargin = 0, uint tooltipID = 0u)
		: base(panelParent, labelText, tooltipID)
	{
		m_lstCustomOptions = customOptions;
		if (customOptions != null && customOptions.Count > 0)
		{
			m_sCustomValue = customOptions.First().Key;
		}
		Initialize(table, col, line, nLeftMargin);
	}

	public void Initialize(Table table, int col, int line, int nLeftMargin = 0)
	{
		m_cmbValue = new ComboBox
		{
			BackgroundColor = AppProperties.Color_Disabled
		};
		m_cmbValue.Font = UIPropertyBase.s_fntBase;
		m_lblTitle = new Label(m_sLabelText);
		if (m_lstCustomOptions != null)
		{
			foreach (KeyValuePair<string, string> lstCustomOption in m_lstCustomOptions)
			{
				m_cmbValue.Items.Add(lstCustomOption.Key, lstCustomOption.Value);
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

	public override void SetMinWidth(int minWidth)
	{
		m_cmbValue.MinWidth = minWidth;
	}

	private void OnSelectionChanged(object sender, EventArgs e)
	{
		if (m_fFireChangedEvent && m_cmbValue.SelectedItem != null)
		{
			m_imvChanged.Image = (IsChanged ? UIPropertyBase.s_imgChanged : UIPropertyBase.s_imgEmpty);
			FireChange();
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
		if (m_cmbValue != null)
		{
			m_imvChanged.Image = (IsChanged ? UIPropertyBase.s_imgChanged : UIPropertyBase.s_imgEmpty);
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
			return "0";
		}
		return m_cmbValue.SelectedItem.ToString();
	}

	protected override void OnClearValue()
	{
		if (m_lstCustomOptions.Count() > 0 && m_lstCustomOptions.Count() < 5)
		{
			m_sCustomValue = m_lstCustomOptions.First().Key;
			m_cmbValue.SelectedItem = m_sCustomValue;
			RefreshDisplay();
		}
	}

	public override void SetValue(object newValue)
	{
		if (m_cmbValue != null)
		{
			m_cmbValue.SelectedItem = newValue.ToString();
			if (m_cmbValue.SelectedItem == null)
			{
				m_cmbValue.SelectedIndex = 0;
			}
		}
	}

	public void SetCustomList(Dictionary<string, string> customOptions)
	{
		m_fFireChangedEvent = false;
		object value = GetValue();
		m_lstCustomOptions = customOptions;
		if (m_lstCustomOptions != null)
		{
			m_cmbValue.Items.Clear();
			foreach (KeyValuePair<string, string> lstCustomOption in m_lstCustomOptions)
			{
				m_cmbValue.Items.Add(lstCustomOption.Key, lstCustomOption.Value);
			}
		}
		SetValue(value);
		RefreshDisplay();
		m_fFireChangedEvent = true;
	}
}
