using System.Collections.Generic;
using System.Linq;
using Xwt;

namespace ICUServiceInstaller;

public class UIConfigCategory
{
	public string Name { get; }

	public string Title { get; }

	public List<UIPropertyBase> m_lstProperties { get; private set; } = new List<UIPropertyBase>();

	public List<Widget> m_lstWidget { get; private set; } = new List<Widget>();

	public bool ContainsAdvancedProp => m_lstProperties.Where((UIPropertyBase a) => a.IsAdvancedProp).ToList().Count() > 0;

	public bool ShowCategory { get; set; } = true;

	public bool Hide { get; set; }

	public UIConfigCategory(string name, string title = "")
	{
		Name = name;
		Title = title;
	}

	public UIPropertyBase Add(UIPropertyBase propBase, bool advancedProperty = false)
	{
		propBase.IsAdvancedProp = advancedProperty;
		m_lstProperties.Add(propBase);
		propBase.SetVisible(fVisible: false);
		return propBase;
	}

	public Widget AddWidget(Widget widget)
	{
		m_lstWidget.Add(widget);
		widget.Visible = false;
		return widget;
	}

	public void ShowProperties(bool show, bool showAdvanced = false)
	{
		m_lstProperties.ForEach((UIPropertyBase a) =>
		{
			a.SetVisible(show && ((!a.IsAdvancedProp & !a.Hide) || (a.IsAdvancedProp & showAdvanced)) && !a.IsConfidential);
		});
		m_lstWidget.ForEach((Widget a) =>
		{
			a.Visible = show;
		});
	}

	public override string ToString()
	{
		return Name;
	}
}
