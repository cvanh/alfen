using System;
using System.Collections.Generic;
using System.Linq;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class UIConfigurationPanel : IDisposable
{
	protected string m_title = string.Empty;

	protected Table m_tableCategory = new Table();

	protected DataField<UIConfigCategory> m_dfCategory = new DataField<UIConfigCategory>();

	protected DataField<Image> m_dfCategorySelection = new DataField<Image>();

	protected TreeView m_treeCategory = new TreeView();

	protected TreeStore m_treeCategoryStore;

	protected TreePosition m_prevCategorySelection;

	protected ScrollView m_scrollView;

	protected UIConfigCategory m_currentCategory;

	protected List<UIConfigCategory> m_lsCategory = new List<UIConfigCategory>();

	protected PanelBase m_panelParent;

	protected Image m_imgEmpty = Image.FromResource(typeof(App), AppProperties.ResourcePath("empty16.png"));

	protected Image m_imgLeftArrow = Image.FromResource(typeof(App), AppProperties.ResourcePath("left-arrow.png")).Scale(0.5);

	protected Label m_lblCategory;

	protected Label m_lblAdvancedMode;

	protected CheckBox m_chkAdvancedMode;

	private static bool showAdvancedProperties;

	public UIConfigCategory CurrentCategory => m_currentCategory;

	public UIConfigurationPanel(PanelBase panelParent, string title)
	{
		m_panelParent = panelParent;
		m_panelParent.ConfigurationPanel = new Table();
		m_panelParent.ConfigurationPanel.HorizontalPlacement = WidgetPlacement.Fill;
		m_panelParent.ConfigurationPanel.VerticalPlacement = WidgetPlacement.Fill;
		m_panelParent.MarginLeft = 0.0;
		m_panelParent.MarginRight = 0.0;
		m_title = title;
		m_tableCategory.Margin = 6.0;
		Label label = new Label(m_title);
		label.TextColor = Colors.SteelBlue;
		label.Font = label.Font.WithScaledSize(1.5).WithWeight(FontWeight.Semibold);
		m_panelParent.Add(label, 0, 0, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.End);
		m_lblCategory = new Label();
		m_lblCategory.TextColor = Colors.SteelBlue;
		m_lblCategory.Font = m_lblCategory.Font.WithScaledSize(1.1).WithWeight(FontWeight.Semibold);
		m_panelParent.Add(m_lblCategory, 1, 0, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.End);
		m_chkAdvancedMode = new CheckBox
		{
			BackgroundColor = AppProperties.Color_Disabled
		};
		m_chkAdvancedMode.Active = showAdvancedProperties;
		m_chkAdvancedMode.Toggled += OnAdvancedCheckBoxToggled;
		m_lblAdvancedMode = new Label("Advanced Settings");
		m_lblAdvancedMode.TextColor = Colors.SteelBlue;
		m_lblAdvancedMode.Font = m_lblAdvancedMode.Font.WithScaledSize(1.1).WithWeight(FontWeight.Semibold);
		HBox hBox = new HBox();
		hBox.PackEnd(m_lblAdvancedMode, expand: false, WidgetPlacement.Fill, WidgetPlacement.Center);
		hBox.PackEnd(m_chkAdvancedMode, expand: false, WidgetPlacement.Fill, WidgetPlacement.Center);
		m_panelParent.Add(hBox, 2, 0, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.End, WidgetPlacement.End);
		m_treeCategoryStore = new TreeStore(m_dfCategory, m_dfCategorySelection);
		m_treeCategory.DataSource = m_treeCategoryStore;
		m_treeCategory.SelectionMode = SelectionMode.Single;
		m_treeCategory.VerticalPlacement = WidgetPlacement.Center;
		m_treeCategory.Font = Font.SystemSansSerifFont.WithSize(12.0);
		m_treeCategory.Columns.Add("Category", new TextCellView(m_dfCategory));
		m_treeCategory.Columns.Add("Selection", new ImageCellView(m_dfCategorySelection));
		m_treeCategory.HeadersVisible = false;
		m_treeCategory.SelectionChanged += OnCategorySelectionChanged;
		m_treeCategory.MarginRight = 4.0;
		m_treeCategory.MinWidth = 165.0;
		m_treeCategory.HorizontalScrollPolicy = ScrollPolicy.Never;
		m_treeCategory.Margin = new WidgetSpacing(0.0, 0.0, 0.0, 0.0);
		m_panelParent.Add(m_treeCategory, 0, 1, 1, 1, hexpand: false, vexpand: true);
		m_scrollView = new ScrollView
		{
			VerticalScrollPolicy = ScrollPolicy.Automatic,
			HorizontalScrollPolicy = ScrollPolicy.Never,
			HorizontalPlacement = WidgetPlacement.Fill,
			VerticalPlacement = WidgetPlacement.Fill,
			BorderVisible = false,
			Content = m_panelParent.ConfigurationPanel
		};
		FrameBox widget = new FrameBox
		{
			BorderColor = AppProperties.Color_Border,
			Padding = 8.0,
			Margin = 0.0,
			BorderWidth = 1.0,
			BackgroundColor = Colors.White,
			Content = m_scrollView,
			MarginRight = 1.0
		};
		m_panelParent.Add(widget, 1, 1, 1, 2, hexpand: true, vexpand: true);
	}

	public void Dispose()
	{
		UpdateControls();
	}

	private void OnAdvancedCheckBoxToggled(object sender, EventArgs e)
	{
		showAdvancedProperties = ((CheckBox)sender).Active;
		LoadCategories();
		m_panelParent.OnUpdateControls();
	}

	private void UpdateAdvancedCheckBox()
	{
		bool flag = false;
		foreach (UIConfigCategory item in m_lsCategory)
		{
			if (item.ContainsAdvancedProp)
			{
				flag = true;
				break;
			}
		}
		if (flag)
		{
			m_lblAdvancedMode.Show();
			m_chkAdvancedMode.Show();
		}
		else
		{
			m_lblAdvancedMode.Hide();
			m_chkAdvancedMode.Hide();
		}
		m_chkAdvancedMode.Sensitive = flag;
	}

	public void UpdateControls()
	{
		UpdateAdvancedCheckBox();
		LoadCategories();
	}

	public void OnCategorySelectionChanged(object sender, EventArgs e)
	{
		if (m_treeCategory.SelectedRow != null)
		{
			if (m_prevCategorySelection != null)
			{
				m_treeCategoryStore.GetNavigatorAt(m_prevCategorySelection).SetValue(m_dfCategorySelection, m_imgEmpty);
			}
			m_treeCategoryStore.GetNavigatorAt(m_treeCategory.SelectedRow).SetValue(m_dfCategorySelection, m_imgLeftArrow);
			m_prevCategorySelection = m_treeCategory.SelectedRow;
			UIConfigCategory value = m_treeCategoryStore.GetNavigatorAt(m_treeCategory.SelectedRow).GetValue(m_dfCategory);
			ShowContent(value);
			m_scrollView.VerticalScrollControl.Value = 0.0;
		}
	}

	public void ShowContent(UIConfigCategory category)
	{
		if (string.IsNullOrEmpty(category.Title))
		{
			m_lblCategory.Text = category.Name;
		}
		else
		{
			m_lblCategory.Text = category.Title;
		}
		if (m_currentCategory != null)
		{
			m_currentCategory.ShowProperties(show: false);
		}
		m_currentCategory = category;
		m_currentCategory.ShowProperties(show: true, showAdvancedProperties);
	}

	public void RefreshCurrentCategory()
	{
		if (m_currentCategory != null)
		{
			m_currentCategory.ShowProperties(show: true, showAdvancedProperties);
		}
	}

	public UIConfigCategory AddCategory(string name, string title = "")
	{
		UIConfigCategory uIConfigCategory = new UIConfigCategory(name, title);
		m_lsCategory.Add(uIConfigCategory);
		m_panelParent.SetTableRow(1);
		m_panelParent.SetLeftMargin(0);
		return uIConfigCategory;
	}

	private void LoadCategories()
	{
		string text = string.Empty;
		TreePosition treePosition = null;
		if (m_prevCategorySelection != null)
		{
			text = m_treeCategoryStore.GetNavigatorAt(m_prevCategorySelection).GetValue(m_dfCategory).Name;
		}
		m_treeCategoryStore.Clear();
		foreach (UIConfigCategory item in m_lsCategory)
		{
			if (item.Hide || (item.m_lstProperties.Count == 0 && item.m_lstWidget.Count == 0))
			{
				continue;
			}
			bool flag = item.m_lstProperties.Where((UIPropertyBase a) => !a.IsAdvancedProp).ToList().Any();
			if (((flag || item.m_lstWidget.Count != 0) && item.ShowCategory) || (showAdvancedProperties & !flag))
			{
				TreeNavigator treeNavigator = m_treeCategoryStore.AddNode().SetValues(m_dfCategory, item, m_dfCategorySelection, m_imgEmpty);
				if (item.Name == text)
				{
					treePosition = treeNavigator.CurrentPosition;
				}
			}
		}
		if (treePosition == null)
		{
			treePosition = m_treeCategoryStore.GetFirstNode().CurrentPosition;
			m_treeCategory.SelectRow(treePosition);
		}
		else
		{
			m_treeCategory.ScrollToRow(treePosition);
			m_treeCategory.SelectRow(treePosition);
		}
	}

	public void ShowCategory(UIConfigCategory cat, bool show = true)
	{
		if (cat != null)
		{
			cat.ShowCategory = show;
			LoadCategories();
		}
	}
}
