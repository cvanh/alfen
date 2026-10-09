using System;
using System.Collections.Generic;
using System.Linq;
using System.Text.RegularExpressions;
using ICUNetwork;
using ICUServiceInstaller.Utils;
using Serilog;
using Xwt;

namespace ICUServiceInstaller;

public class PanelAllProperties : PanelBase
{
	private readonly ILogger Logger = Log.ForContext<PanelAllProperties>();

	protected UIConfigurationPanel m_configPanel;

	protected List<UIPropertyBase> m_lstControls = new List<UIPropertyBase>();

	protected bool specialWritePermission;

	private CheckBox m_chkName;

	private CheckBox m_chkValue;

	private CheckBox m_chkId;

	private CheckBox m_chkRegex;

	private TextEntry m_txtSearch;

	private Button m_btnSearch;

	private UIConfigCategory m_catSearch;

	private Table m_tbSearch;

	public PanelAllProperties(MainWindow parent, string pageID = "", bool showBorder = true)
		: base(parent, pageID, fIndent: false, showBorder)
	{
		Title = "All Properties";
		Tooltip = "All settings";
		IconName = "settings-gears.png";
	}

	public override bool OnChangeDevice(ICUDevice newDevice, ICUDevice previousDevice)
	{
		ClearPanel();
		if (!(newDevice is ICULanDevice))
		{
			return true;
		}
		Logger.AddChargerContext(m_currentDevice).Information("PageView, {Panel}", Title);
		if (!IsPanelVisible)
		{
			return true;
		}
		specialWritePermission = (Keyboard.CurrentModifiers & ModifierKeys.Shift) != 0 && (Keyboard.CurrentModifiers & ModifierKeys.Control) != 0;
		List<string> list = newDevice.RequestCategories(Application.MainLoop.DispatchPendingEvents).Distinct().ToList();
		newDevice.UpdateCategories();
		UIConfigCategory uIConfigCategory = null;
		using (m_configPanel = AddConfigurationPanel(Tooltip))
		{
			foreach (string category in list)
			{
				List<ICUProperty> list2 = (from b in newDevice.PropertyDictionary.Where((ICUProperty a) => a.Category == category).ToList()
					orderby b.ID_SUB
					select b).ToList();
				if (list2.Count > 0)
				{
					if (!string.IsNullOrEmpty(category) && category.Length > 1)
					{
						string name = char.ToUpper(category[0]) + category.Substring(1);
						uIConfigCategory = m_configPanel.AddCategory(name);
						uIConfigCategory.Add(AddWarningText("These settings are advanced settings, only change them when you know what you are doing. Incorrect values might damage your charging station!", "", 0, null, 2));
					}
					AddProperties(uIConfigCategory, list2);
					uIConfigCategory.Hide = uIConfigCategory.m_lstProperties.Count == 1;
				}
			}
			AddSearchFunction(m_configPanel);
		}
		Visible = false;
		Visible = true;
		return true;
	}

	private void AddSearchFunction(UIConfigurationPanel configPanel)
	{
		m_chkName = new CheckBox("Name");
		m_chkValue = new CheckBox("Value");
		m_chkId = new CheckBox("ID");
		m_chkRegex = new CheckBox("Regex");
		m_txtSearch = new TextEntry();
		m_btnSearch = new Button("Search");
		m_tbSearch = new Table();
		m_catSearch = configPanel.AddCategory("Search");
		m_btnSearch.MinWidth = 80.0;
		m_btnSearch.Clicked += OnSearchClicked;
		m_txtSearch.KeyPressed += OnTxtKeyPressed;
		HBox hBox;
		m_catSearch.AddWidget(hBox = AddToolBox(fExpandVert: false, fExpandHor: true, forceNewInstace: true));
		hBox.PackStart(m_txtSearch, expand: true);
		hBox.PackEnd(m_btnSearch);
		HBox hBox2;
		m_catSearch.AddWidget(hBox2 = AddToolBox(fExpandVert: false, fExpandHor: true, forceNewInstace: true));
		hBox2.PackStart(new Label("Filter: "));
		hBox2.PackStart(m_chkName);
		hBox2.PackStart(m_chkValue);
		hBox2.PackStart(m_chkId);
		hBox2.PackStart(m_chkRegex);
		m_chkName.Active = true;
		m_chkId.Active = true;
		GetTable().Add(m_tbSearch, 0, m_tableRowCounter, 1, 3, hexpand: true, vexpand: true);
		m_catSearch.AddWidget(m_tbSearch);
	}

	public override bool OnSaveChanges()
	{
		if (IsChanged)
		{
			Logger.AddChargerContext(m_currentDevice).Information("Save changes from: {Panel}", Title);
		}
		return true;
	}

	private static bool IsTextMatch(string text, string searchTerm, Regex regex)
	{
		if (string.IsNullOrWhiteSpace(text))
		{
			return false;
		}
		return regex?.IsMatch(text) ?? (text.IndexOf(searchTerm, StringComparison.OrdinalIgnoreCase) >= 0);
	}

	private static bool TryMatchPropertyName(ICUProperty prop, string searchTerm, Regex regex)
	{
		if (IsTextMatch(prop.Title, searchTerm, regex))
		{
			return true;
		}
		if (IsTextMatch(prop.Parameter?.Name, searchTerm, regex))
		{
			return true;
		}
		return false;
	}

	private static bool TryMatchPropertyValue(ICUProperty prop, string searchTerm, Regex regex)
	{
		if (prop.Value == null)
		{
			return false;
		}
		return IsTextMatch(prop.Value.ToString(), searchTerm, regex);
	}

	private static bool TryMatchPropertyId(ICUProperty prop, string searchTerm, Regex regex)
	{
		return IsTextMatch($"0x{prop.Id:X4}_{prop.SubId:X}", searchTerm, regex);
	}

	private void OnSearchClicked(object sender, EventArgs e)
	{
		List<ICUProperty> list = new List<ICUProperty>();
		if (m_currentDevice == null)
		{
			return;
		}
		if (string.IsNullOrWhiteSpace(m_txtSearch.Text))
		{
			MessageDialog.ShowError("Please enter a search term.");
			return;
		}
		Regex regex = CreateSearchRegex(m_chkRegex.Active, m_txtSearch.Text);
		foreach (ICUProperty item in m_currentDevice.PropertyDictionary)
		{
			if (item.Value != null)
			{
				bool flag = false;
				if (m_chkName.Active && TryMatchPropertyName(item, m_txtSearch.Text, regex))
				{
					flag = true;
				}
				if (!flag && m_chkValue.Active && TryMatchPropertyValue(item, m_txtSearch.Text, regex))
				{
					flag = true;
				}
				if (!flag && m_chkId.Active && TryMatchPropertyId(item, m_txtSearch.Text, regex))
				{
					flag = true;
				}
				if (flag)
				{
					list.Add(item);
				}
			}
		}
		m_tableRowCounter = 0;
		m_catSearch.m_lstProperties.Clear();
		m_tbSearch.Clear();
		AddProperties(m_catSearch, list, m_tbSearch);
		if (list.Count == 0)
		{
			UIPropertyLabel uIPropertyLabel;
			m_catSearch.Add(uIPropertyLabel = (UIPropertyLabel)AddSmallHeader($"No results found for: {m_txtSearch.Text}", "", 0, m_tbSearch));
			uIPropertyLabel.RefreshDisplay();
		}
		UpdateDisplay();
		m_configPanel.UpdateControls();
	}

	private static Regex CreateSearchRegex(bool useRegex, string regexString)
	{
		if (useRegex)
		{
			try
			{
				return new Regex(regexString, RegexOptions.IgnoreCase, TimeSpan.FromSeconds(1.0));
			}
			catch (Exception ex)
			{
				MessageDialog.ShowError($"Invalid regular expression: {ex.Message}\nNormal text search will be used instead.");
				return null;
			}
		}
		return null;
	}

	private void OnTxtKeyPressed(object sender, KeyEventArgs e)
	{
		if (e.Key == Key.Return)
		{
			OnSearchClicked(null, null);
		}
	}

	private void AddProperties(UIConfigCategory category, List<ICUProperty> lsProp, Table table = null)
	{
		foreach (ICUProperty item in lsProp)
		{
			if (item.Id == 8273 || item.Id == 8272 || item.Id == 8271)
			{
				if (specialWritePermission)
				{
					item.ReadOnly = false;
				}
				else
				{
					item.ReadOnly = true;
				}
			}
			if (item.Value == null || item.Category == null || category == null)
			{
				continue;
			}
			switch (item.DataType)
			{
			case SDT.VISIBLE_STRING:
			case SDT.OCTET_STRING:
			case SDT.UNICODE_STRING:
				category.Add(AddText(item.Id, item.SubId, table));
				break;
			case SDT.INTEGER8:
			case SDT.INTEGER16:
			case SDT.INTEGER32:
			case SDT.UNSIGNED8:
			case SDT.UNSIGNED16:
			case SDT.UNSIGNED32:
			case SDT.REAL32:
			case SDT.REAL64:
			case SDT.INTEGER64:
			{
				int digits = 0;
				if (item.DataType == SDT.REAL32 || item.DataType == SDT.REAL64)
				{
					digits = 3;
				}
				if (item.Parameter == null)
				{
					if (item.MaxLength == 2)
					{
						category.Add(AddCheckBox(item.Id, item.SubId, 0, item.ICUName, table));
					}
					else
					{
						category.Add(AddCustomNumber(item.Id, item.SubId, digits, item.ICUName, 1.0, table));
					}
				}
				else if (item.Parameter.Options != null && item.Parameter.Options.Count > 0)
				{
					if (item.Parameter.Options.Count == 1)
					{
						category.Add(AddCheckBox(item.Id, item.SubId, 0, "", table));
					}
					else
					{
						category.Add(AddSelect(item.Id, item.SubId, 0, 0, "", caseSensitive: true, table));
					}
				}
				else if (item.Value != null)
				{
					category.Add(AddNumber(item.Id, item.SubId, digits, table));
				}
				break;
			}
			case SDT.UNSIGNED64:
				if (item.ReadOnly)
				{
					category.Add(AddCustomText(item.Title, item.Value.ToString(), "", table));
				}
				else
				{
					category.Add(AddReadOnlyText(item.Id, item.SubId, "", UIPropertyStringType.DateTime, table));
				}
				break;
			case SDT.BOOLEAN:
				category.Add(AddCheckBox(item.Id, item.SubId, 0, "", table));
				break;
			case SDT.BYTEARRAY:
			{
				if (!(item.Category.ToLowerInvariant() != "leds"))
				{
					break;
				}
				string customValue = "";
				byte[] array = (byte[])item.Value;
				if (array != null)
				{
					customValue = string.Join(",", array.Select((byte a) => a.ToString("X2")));
				}
				category.Add(AddCustomText(item.ICUName, customValue, "", table));
				break;
			}
			case SDT.ARRAY_16:
			{
				string customValue2 = "";
				ushort[] array2 = (ushort[])item.Value;
				if (array2 != null)
				{
					customValue2 = string.Join(",", array2.Select((ushort a) => a.ToString("X4")));
				}
				category.Add(AddCustomText(item.ICUName, customValue2, "", table));
				break;
			}
			default:
				if (item.Value != null)
				{
					category.Add(AddCustomText(item.ICUName, item.Value.ToString(), "", table));
				}
				break;
			}
		}
	}
}
