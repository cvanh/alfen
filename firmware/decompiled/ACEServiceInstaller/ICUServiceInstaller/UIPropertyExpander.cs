using System;
using System.Collections.Generic;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class UIPropertyExpander : UIPropertyBase
{
	protected Expander m_expExpander;

	protected List<UIPropertyBase> m_lstProperties = new List<UIPropertyBase>();

	protected bool m_fChildIsChanged;

	public string Name => m_expExpander.Label;

	public bool IsExpanded { get; private set; }

	public int SubControlCount => m_lstProperties.Count;

	public UIPropertyExpander(PanelBase panelParent, Table table, int col, int line, string title)
		: base(panelParent)
	{
		m_expExpander = new Expander();
		m_expExpander.Label = title;
		m_expExpander.MinHeight = UIPropertyBase.s_propertyMinHeight;
		m_expExpander.Font = UIPropertyBase.s_fntBaseLabel.WithStyle(FontStyle.Oblique);
		m_expExpander.ExpandChanged += OnExpandedChanged;
		SetWidgetSizes(m_expExpander, m_imvChanged);
		table.Add(m_expExpander, col, line, 1, 2, hexpand: false, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, 0.0, 0.0, 0.0, 0.0);
		table.Add(m_imvChanged, col + 2, line, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Start, WidgetPlacement.Start, 5.0, UIPropertyBase.s_marginVer, -2.0, UIPropertyBase.s_marginVer);
	}

	private void OnExpandedChanged(object sender, EventArgs e)
	{
		IsExpanded = m_expExpander.Expanded;
		m_lstProperties.ForEach((UIPropertyBase a) =>
		{
			a.SetVisible(IsExpanded);
		});
		if (!IsExpanded && m_panelParent.Parent is ScrollView scrollView)
		{
			scrollView.VerticalScrollControl.Value = 0.0;
		}
	}

	public UIPropertyBase Add(UIPropertyBase propBase)
	{
		propBase.Changed += OnPropertyChanged;
		m_lstProperties.Add(propBase);
		propBase.SetVisible(fVisible: false);
		return propBase;
	}

	private void OnPropertyChanged(object sender, EventArgs e)
	{
		m_fChildIsChanged = m_lstProperties.Exists((UIPropertyBase a) => a.IsChanged);
		m_imvChanged.Image = (m_fChildIsChanged ? UIPropertyBase.s_imgChanged : UIPropertyBase.s_imgEmpty);
	}

	public override void SetVisible(bool fVisible)
	{
		m_expExpander.Visible = fVisible;
		m_imvChanged.Visible = fVisible;
		if (!fVisible)
		{
			m_lstProperties.ForEach((UIPropertyBase a) =>
			{
				a.SetVisible(fVisible: false);
			});
			return;
		}
		bool fSubsVisible = m_expExpander.Expanded;
		m_lstProperties.ForEach((UIPropertyBase a) =>
		{
			a.SetVisible(fSubsVisible);
		});
	}

	public override void SetEnable(bool fEnable)
	{
		m_fEnabled = fEnable;
		m_lstProperties.ForEach((UIPropertyBase a) =>
		{
			a.SetEnable(fEnable);
		});
	}

	public List<UIPropertyBase> Childs()
	{
		return m_lstProperties;
	}
}
