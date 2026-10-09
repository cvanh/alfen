using System;
using System.Collections.Generic;
using System.Linq;
using System.Timers;
using ICUNetwork;
using ICUSettings;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class PanelBase : Table
{
	protected class DoubleColumnMode : IDisposable
	{
		protected int m_startRow;

		protected PanelBase m_panelBase;

		public DoubleColumnMode(PanelBase pb)
		{
			m_panelBase = pb;
			if (m_panelBase != null)
			{
				m_startRow = m_panelBase.m_tableRowCounter;
				m_panelBase.DoubleColumnTable = new Table();
			}
			m_panelBase.m_tableRowCounter = 0;
		}

		public void NextColumn()
		{
			if (m_panelBase != null)
			{
				m_panelBase.m_tableRowCounter = 0;
				m_panelBase.m_tableColCounter = 3;
			}
		}

		public void Dispose()
		{
			Dispose(disposing: true);
			GC.SuppressFinalize(this);
		}

		protected virtual void Dispose(bool disposing)
		{
			if (disposing && m_panelBase != null)
			{
				m_panelBase.m_tableColCounter = 0;
				HBox hBox = new HBox
				{
					MarginRight = 2.0
				};
				hBox.PackStart(m_panelBase.DoubleColumnTable, expand: true);
				m_panelBase.m_tableRowCounter = m_startRow;
				m_panelBase.Add(hBox, 0, m_panelBase.m_tableRowCounter, 1, 3);
				m_panelBase.m_tableRowCounter++;
				m_panelBase.DoubleColumnTable = null;
			}
		}
	}

	protected static int s_nUpdateControlsTimerMs = 66;

	protected static int s_nDefaultMargin = UIPropertyBase.s_marginHorMax;

	protected static Color s_colDisabled = AppProperties.Color_Disabled;

	protected MainWindow m_parent;

	protected HBox m_hboxButtons;

	protected HBox m_hboxTools;

	protected int m_tableRowCounter;

	protected int m_tableColCounter;

	private string _IconName = "";

	protected List<UIPropertyBase> m_lstProperties = new List<UIPropertyBase>();

	protected int m_nLeftMargin;

	protected Timer m_timUpdate;

	protected bool m_fSuspendUpdateTimer;

	protected Timer m_timOneShotUpdate;

	protected ICULanDevice m_currentDevice;

	protected ICULanDevice m_newDevice;

	protected ICUUser m_currentUser;

	protected bool m_fIsVisible;

	protected Dictionary<string, string> m_dicMobileNetworkTechnology = new Dictionary<string, string>
	{
		{ "0", "2G" },
		{ "1", "3G" },
		{ "2", "4G" }
	};

	protected List<ICUProperty> m_lstHiddenProperties = new List<ICUProperty>();

	public Table DoubleColumnTable { get; set; }

	public Table ConfigurationPanel { get; set; }

	public string Title { get; set; }

	public string Tooltip { get; set; }

	public string IconName
	{
		get
		{
			return _IconName;
		}
		set
		{
			_IconName = value;
			LoadIcon();
		}
	}

	public Image IconImage { get; private set; }

	public string PageID { get; set; }

	public bool BusyChangingDevice { get; set; }

	public bool ShowFrameBorder { get; set; }

	public bool IsPanelVisible
	{
		get
		{
			return m_fIsVisible;
		}
		set
		{
			if (m_fIsVisible != value)
			{
				m_fIsVisible = value;
				if (value)
				{
					OnShowPanel(show: true, forceRefresh: true);
				}
			}
		}
	}

	public List<UIPropertyBase> AllProperties => m_lstProperties;

	public virtual bool IsChanged => AllProperties.Where((UIPropertyBase a) => a.IsChanged).ToList().Count > 0;

	public event EventHandler Changed;

	public PanelBase(MainWindow parent, string pageID, bool fIndent = false, bool showBorder = true)
	{
		m_parent = parent;
		PageID = pageID;
		Title = "<todo>";
		BackgroundColor = Colors.White;
		BusyChangingDevice = false;
		ShowFrameBorder = showBorder;
		m_timOneShotUpdate = new Timer();
		m_timOneShotUpdate.Elapsed += OnShouldUpdate;
		Margin = 6.0;
		if (fIndent)
		{
			m_nLeftMargin = s_nDefaultMargin;
		}
	}

	protected void LoadIcon()
	{
		IconImage = Image.FromResource(typeof(App), AppProperties.ResourcePath(IconName));
	}

	protected void StartUpdateTimer(int intervalMs)
	{
		m_timUpdate = new Timer();
		m_timUpdate.Elapsed += OnUpdateTimerElapsed;
		m_timUpdate.AutoReset = false;
		m_timUpdate.Interval = intervalMs;
		m_timUpdate.Start();
	}

	protected void StopUpdateTimer()
	{
		m_timUpdate.Stop();
	}

	public void SuspendUpdateTimer(bool fOn)
	{
		m_fSuspendUpdateTimer = fOn;
	}

	protected bool DidPropertiesChangeInCategories(List<UIConfigCategory> categories)
	{
		return categories.Where((UIConfigCategory category) => category != null).SelectMany((UIConfigCategory category) => category.m_lstProperties).Any((UIPropertyBase property) => property.IsChanged);
	}

	private void OnUpdateTimerElapsed(object sender, ElapsedEventArgs e)
	{
		m_timUpdate.Stop();
		if (IsPanelVisible && !m_fSuspendUpdateTimer && (m_currentDevice == null || !m_currentDevice.IsRebooting))
		{
			OnUpdateTick();
		}
		m_timUpdate.Start();
	}

	public void SetUser(ICUUser user)
	{
		if (user == null)
		{
			return;
		}
		m_currentUser = user;
		if (m_currentUser.Group == null)
		{
			return;
		}
		ICURights rights = m_currentUser.GetRights(PageID);
		foreach (UIPropertyBase lstProperty in m_lstProperties)
		{
			lstProperty.CheckAccessRights(m_currentUser.Group, rights);
		}
	}

	public bool IsVisible(ICUUser user)
	{
		if (user == null)
		{
			return false;
		}
		if (user.GetRights(PageID) == ICURights.None)
		{
			return false;
		}
		return true;
	}

	protected virtual void OnUpdateTick()
	{
	}

	public void ClearPanel()
	{
		Clear();
		foreach (UIPropertyBase lstProperty in m_lstProperties)
		{
			lstProperty.ClearChangedHandlers();
		}
		m_lstProperties.Clear();
		m_hboxButtons = null;
		m_hboxTools = null;
		ConfigurationPanel = null;
		MarginTop = 0.0;
		MarginBottom = 0.0;
	}

	public void SetTableColumn(int column)
	{
		m_tableColCounter = column;
	}

	public void SetTableRow(int row)
	{
		m_tableRowCounter = row;
	}

	public void SetLeftMargin(int margin)
	{
		m_nLeftMargin = margin;
	}

	public Table GetTable(Table table = null)
	{
		return table ?? ConfigurationPanel ?? DoubleColumnTable ?? this;
	}

	public UIPropertyBase AddCheckBox(ushort Id, byte subId = 0, int valueMask = 0, string overwriteLabelText = "", Table table = null, uint tooltipID = 0u)
	{
		return AddPropertyBase(Id, new UIPropertyCheckbox(this, GetTable(table), m_tableColCounter, m_tableRowCounter, Id, subId, valueMask, overwriteLabelText, m_nLeftMargin, tooltipID));
	}

	public UIPropertyBase AddText(ushort Id, byte subId = 0, Table table = null, uint tooltipID = 0u)
	{
		return AddPropertyBase(Id, new UIPropertyString(this, GetTable(table), m_tableColCounter, m_tableRowCounter, Id, subId, UIPropertyStringType.String, m_nLeftMargin, "", 1, tooltipID));
	}

	public UIPropertyBase AddText(ushort Id, byte subId, string overwriteLabelText, Table table = null, uint tooltipID = 0u)
	{
		return AddPropertyBase(Id, new UIPropertyString(this, GetTable(table), m_tableColCounter, m_tableRowCounter, Id, subId, UIPropertyStringType.String, m_nLeftMargin, overwriteLabelText, 1, tooltipID));
	}

	public UIPropertyBase AddWriteOnlyText(ushort Id, byte subId, string overwriteLabelText, Table table = null, uint tooltipID = 0u)
	{
		return AddPropertyBase(Id, new UIPropertyString(this, GetTable(table), m_tableColCounter, m_tableRowCounter, writeOnly: true, Id, subId, UIPropertyStringType.String, m_nLeftMargin, overwriteLabelText, 1, tooltipID));
	}

	public UIPropertyBase AddReadOnlyText(ushort Id, byte subId = 0, string overwriteLabelText = "", UIPropertyStringType type = UIPropertyStringType.String, Table table = null, uint tooltipID = 0u)
	{
		return AddPropertyBase(Id, new UIPropertyReadOnlyString(this, GetTable(table), m_tableColCounter, m_tableRowCounter, Id, subId, type, m_nLeftMargin, overwriteLabelText, tooltipID), forceReadonly: true);
	}

	public UIPropertyBase AddReadOnlyDeviceText(string labelText, string devicePropertyName, string featureRightId = "", Table table = null, uint tooltipID = 0u)
	{
		return AddPropertyBase(featureRightId, new UIPropertyReadOnlyString(this, GetTable(table), m_tableColCounter, m_tableRowCounter, labelText, devicePropertyName, UIPropertyStringType.String, m_nLeftMargin, tooltipID), forceReadonly: true);
	}

	public UIPropertyBase AddNumber(ushort Id, byte subId = 0, int digits = 0, Table table = null, uint tooltipID = 0u)
	{
		return AddPropertyBase(Id, new UIPropertyNumber(this, GetTable(table), m_tableColCounter, m_tableRowCounter, Id, subId, digits, m_nLeftMargin, forceReadOnly: false, tooltipID));
	}

	public UIPropertyBase AddNumber(ushort Id, byte subId, int digits, ushort baseId, byte baseSubId, Table table = null, uint tooltipID = 0u)
	{
		return AddPropertyBase(Id, new UIPropertyNumber(this, GetTable(table), m_tableColCounter, m_tableRowCounter, Id, subId, digits, baseId, baseSubId, m_nLeftMargin, forceReadOnly: false, tooltipID));
	}

	public UIPropertyBase AddCustomNumber(ushort Id, byte subId, int digits, string customLabel, double factor = 1.0, Table table = null, uint tooltipID = 0u)
	{
		return AddPropertyBase(Id, new UIPropertyNumber(this, GetTable(table), m_tableColCounter, m_tableRowCounter, Id, subId, digits, customLabel, factor, m_nLeftMargin, forceReadOnly: false, tooltipID));
	}

	public UIPropertyBase AddReadOnlyCustomNumber(ushort Id, byte subId, string customLabel = "", double factor = 1.0, int digits = 0, Table table = null, uint tooltipID = 0u)
	{
		return AddPropertyBase(Id, new UIPropertyNumber(this, GetTable(table), m_tableColCounter, m_tableRowCounter, Id, subId, digits, customLabel, factor, m_nLeftMargin, forceReadOnly: false, tooltipID), forceReadonly: true);
	}

	public UIPropertyBase AddCustomNumber(int digits, string customLabel, string devicePropName = "", string featureRightId = "", Table table = null, uint tooltipID = 0u)
	{
		return AddPropertyBase(featureRightId, new UIPropertyNumber(this, GetTable(table), m_tableColCounter, m_tableRowCounter, customLabel, devicePropName, m_nLeftMargin, forceReadOnly: false, digits, tooltipID));
	}

	public UIPropertyBase AddReadOnlyDateTime(ushort Id, byte subId = 0, string overwriteLabelText = "", Table table = null, uint tooltipID = 0u)
	{
		return AddPropertyBase(Id, new UIPropertyString(this, GetTable(table), m_tableColCounter, m_tableRowCounter, Id, subId, UIPropertyStringType.DateTime, m_nLeftMargin, overwriteLabelText, 1, tooltipID), forceReadonly: true);
	}

	public UIPropertyBase AddIPAddress(ushort Id, byte subId = 0, string overwriteLabelText = "", Table table = null, uint tooltipID = 0u)
	{
		return AddPropertyBase(Id, new UIPropertyString(this, GetTable(table), m_tableColCounter, m_tableRowCounter, Id, subId, UIPropertyStringType.IP, m_nLeftMargin, overwriteLabelText, 1, tooltipID));
	}

	public UIPropertyBase AddDeviceText(string labelText, string devicePropertyName, string featureRightId = "", Table table = null, uint tooltipID = 0u)
	{
		return AddPropertyBase(featureRightId, new UIPropertyString(this, GetTable(table), m_tableColCounter, m_tableRowCounter, labelText, devicePropertyName, UIPropertyStringType.String, m_nLeftMargin, 1, tooltipID));
	}

	public UIPropertyBase AddCustomText(string labelText, string customValue = "", string featureRightId = "", Table table = null, int reserveLines = 1, uint tooltipID = 0u)
	{
		UIPropertyString propBase = new UIPropertyString(this, GetTable(table), m_tableColCounter, m_tableRowCounter, labelText, string.Empty, UIPropertyStringType.String, m_nLeftMargin, reserveLines, tooltipID)
		{
			CustomValue = customValue
		};
		return AddPropertyBase(featureRightId, propBase);
	}

	public UIPropertyBase AddPlainString(string labelText, string customValue = "", string featureRightId = "", Table table = null, uint tooltipID = 0u)
	{
		UIString propBase = new UIString(this, GetTable(table), m_tableColCounter, m_tableRowCounter, labelText, m_nLeftMargin, tooltipID)
		{
			CustomValue = customValue
		};
		return AddPropertyBase(featureRightId, propBase);
	}

	public UIPropertyBase AddPlainBool(string labelText, bool customValue = false, string featureRightId = "", Table table = null, uint tooltipID = 0u)
	{
		UICheckbox propBase = new UICheckbox(this, GetTable(table), m_tableColCounter, m_tableRowCounter, labelText, m_nLeftMargin, tooltipID)
		{
			CustomValue = customValue
		};
		return AddPropertyBase(featureRightId, propBase);
	}

	public UIPropertyBase AddPlainSelect(string labelText, Dictionary<string, string> lstOptions, string featureRightId = "", Table table = null, uint tooltipID = 0u)
	{
		return AddPropertyBase(featureRightId, new UISelect(this, GetTable(table), m_tableColCounter, m_tableRowCounter, labelText, lstOptions, m_nLeftMargin, tooltipID));
	}

	public UIPropertyBase AddPlainPassword(string labelText, string featureRightId = "", Table table = null, uint tooltipID = 0u)
	{
		UIPassword propBase = new UIPassword(this, GetTable(table), m_tableColCounter, m_tableRowCounter, labelText, m_nLeftMargin, tooltipID);
		return AddPropertyBase(featureRightId, propBase);
	}

	public UIPropertyBase AddSelect(ushort Id, byte subId = 0, ushort baseId = 0, byte baseSubId = 0, string customLabel = "", bool caseSensitive = true, Table table = null, uint tooltipID = 0u)
	{
		UIPropertySelect uIPropertySelect = new UIPropertySelect(this, GetTable(table), m_tableColCounter, m_tableRowCounter, Id, subId, baseId, customLabel, baseSubId, m_nLeftMargin, tooltipID);
		uIPropertySelect.SetCaseSensitive(caseSensitive);
		return AddPropertyBase(Id, uIPropertySelect);
	}

	public UIPropertyBase AddReadOnlySelect(ushort Id, byte subId = 0, ushort baseId = 0, byte baseSubId = 0, string customLabel = "", Table table = null, uint tooltipID = 0u)
	{
		return AddPropertyBase(Id, new UIPropertySelect(this, GetTable(table), m_tableColCounter, m_tableRowCounter, Id, subId, baseId, customLabel, baseSubId, m_nLeftMargin, tooltipID), forceReadonly: true);
	}

	public UIPropertyBase AddCustomSelect(ushort Id, byte subId, string customLabel, Dictionary<string, string> lstOptions, int valueMask = 0, Table table = null, uint tooltipID = 0u)
	{
		return AddPropertyBase(Id, new UIPropertySelect(this, GetTable(table), m_tableColCounter, m_tableRowCounter, Id, subId, customLabel, lstOptions, valueMask, m_nLeftMargin, tooltipID));
	}

	public UIPropertyBase AddCustomSelect(string customLabel, Dictionary<string, string> lstOptions, int valueMask = 0, string featureRightId = "", Table table = null, uint tooltipID = 0u)
	{
		return AddPropertyBase(featureRightId, new UIPropertySelect(this, GetTable(table), m_tableColCounter, m_tableRowCounter, customLabel, lstOptions, valueMask, m_nLeftMargin, forceReadOnly: false, tooltipID));
	}

	public UIPropertyBase AddReadOnlyCustomSelect(ushort Id, byte subId, string customLabel, Dictionary<string, string> lstOptions, Table table = null, uint tooltipID = 0u)
	{
		return AddPropertyBase(Id, new UIPropertySelect(this, GetTable(table), m_tableColCounter, m_tableRowCounter, Id, subId, customLabel, lstOptions, 0, m_nLeftMargin, tooltipID), forceReadonly: true);
	}

	public UIPropertyBase AddManualSelect(string customLabel, Dictionary<string, string> lstOptions, string featureRightId = "", Table table = null, uint tooltipID = 0u)
	{
		return AddPropertyBase(featureRightId, new UIPropertyManualSelect(this, GetTable(table), m_tableColCounter, m_tableRowCounter, customLabel, lstOptions, m_nLeftMargin, forceReadOnly: false, tooltipID));
	}

	public UIPropertyBase AddLabel(string labelText, string featureRightId = "", Table table = null)
	{
		return AddPropertyBase(featureRightId, new UIPropertyLabel(this, GetTable(table), m_tableColCounter, m_tableRowCounter, labelText, EUILabelType.Normal, m_nLeftMargin));
	}

	public UIPropertyBase AddHeader(string labelText, string featureRightId = "", int marginLeft = 0, Table table = null)
	{
		m_nLeftMargin = marginLeft;
		UIPropertyBase result = AddPropertyBase(featureRightId, new UIPropertyLabel(this, GetTable(table), m_tableColCounter, m_tableRowCounter, labelText, EUILabelType.Header, m_nLeftMargin));
		m_nLeftMargin = s_nDefaultMargin;
		return result;
	}

	public UIPropertyBase AddSmallHeader(string labelText, string featureRightId = "", int marginLeft = 0, Table table = null)
	{
		m_nLeftMargin = marginLeft;
		UIPropertyBase result = AddPropertyBase(featureRightId, new UIPropertyLabel(this, GetTable(table), m_tableColCounter, m_tableRowCounter, labelText, EUILabelType.SmallHeader, m_nLeftMargin));
		m_nLeftMargin = s_nDefaultMargin;
		return result;
	}

	public UIPropertyBase AddWarningText(string labelText, string featureRightId = "", int marginLeft = 0, Table table = null, int reserveLines = 1)
	{
		m_nLeftMargin = marginLeft;
		UIPropertyBase result = AddPropertyBase(featureRightId, new UIPropertyLabel(this, GetTable(table), m_tableColCounter, m_tableRowCounter, labelText, EUILabelType.Warning, m_nLeftMargin, reserveLines));
		m_nLeftMargin = s_nDefaultMargin;
		return result;
	}

	public UIPropertyBase AddLargeWarningText(string labelText, string featureRightId = "", int marginLeft = 0, Table table = null)
	{
		m_nLeftMargin = marginLeft;
		UIPropertyBase result = AddPropertyBase(featureRightId, new UIPropertyLabel(this, GetTable(table), m_tableColCounter, m_tableRowCounter, labelText, EUILabelType.LargeWarning, m_nLeftMargin));
		m_nLeftMargin = s_nDefaultMargin;
		return result;
	}

	public UIPropertyBase AddInfoText(string labelText, string featureRightId = "", int marginLeft = 0, Table table = null)
	{
		m_nLeftMargin = marginLeft;
		UIPropertyBase result = AddPropertyBase(featureRightId, new UIPropertyLabel(this, GetTable(table), m_tableColCounter, m_tableRowCounter, labelText, EUILabelType.Info, m_nLeftMargin));
		m_nLeftMargin = s_nDefaultMargin;
		return result;
	}

	public UIPropertyBase AddLabelText(string labelText, EUILabelType eType, string featureRightId = "", int marginLeft = 0, Table table = null)
	{
		m_nLeftMargin = marginLeft;
		return AddPropertyBase(featureRightId, new UIPropertyLabel(this, GetTable(table), m_tableColCounter, m_tableRowCounter, labelText, eType, m_nLeftMargin));
	}

	public UIPropertyBase AddOkNokNaQuestion(string customId, string labelText, EQuestion eDefaultValue, string featureRightId = "", Table table = null, uint tooltipID = 0u)
	{
		return AddPropertyBase(featureRightId, new UIPropertyOkNokNa(this, GetTable(table), m_tableColCounter, m_tableRowCounter, customId, labelText, eDefaultValue, m_nLeftMargin));
	}

	public UIPropertyExpander AddExpander(string title, string featureRightId = "", Table table = null)
	{
		UIPropertyExpander uIPropertyExpander = new UIPropertyExpander(this, GetTable(table), m_tableColCounter, m_tableRowCounter, title);
		AddPropertyBase(featureRightId, uIPropertyExpander);
		return uIPropertyExpander;
	}

	public UIPropertyColor AddColor(ushort Id, byte subId, string labelText)
	{
		UIPropertyColor result = (UIPropertyColor)AddPropertyBase(Id, new UIPropertyColor(this, labelText, Id, subId));
		m_tableRowCounter += 2;
		return result;
	}

	public UIPropertyColorHolder AddColorHolder(string labelText, string featureRightId = "", Table table = null, uint tooltipID = 0u)
	{
		UIPropertyColorHolder result = (UIPropertyColorHolder)AddPropertyBase(featureRightId, new UIPropertyColorHolder(this, GetTable(table), m_tableColCounter, m_tableRowCounter, labelText, m_nLeftMargin, tooltipID));
		m_tableRowCounter += 2;
		return result;
	}

	protected UIConfigurationPanel AddConfigurationPanel(string title)
	{
		m_tableRowCounter = 1;
		m_tableColCounter = 2;
		return new UIConfigurationPanel(this, title);
	}

	protected UIPropertyBase AddPropertyBase(string featureRightId, UIPropertyBase propBase, bool forceReadonly = false)
	{
		propBase.Changed += OnPropertyChanged;
		propBase.FeatureRightID = featureRightId;
		m_lstProperties.Add(propBase);
		propBase.ForceReadonly(forceReadonly);
		m_tableRowCounter++;
		return propBase;
	}

	protected UIPropertyBase AddPropertyBase(ushort Id, UIPropertyBase propBase, bool forceReadonly = false)
	{
		propBase.Changed += OnPropertyChanged;
		propBase.FeatureRightID = $"ID_{Id:X4}";
		m_lstProperties.Add(propBase);
		propBase.ForceReadonly(forceReadonly);
		m_tableRowCounter++;
		return propBase;
	}

	protected HBox AddButtonBox(bool fExpandVert = true, bool fExpandHor = false, bool forceNewInstace = false)
	{
		if ((m_hboxButtons == null) | forceNewInstace)
		{
			m_hboxButtons = new HBox();
			Table table = GetTable();
			HBox hboxButtons = m_hboxButtons;
			int tableRowCounter = m_tableRowCounter;
			bool vexpand = fExpandVert;
			table.Add(hboxButtons, 0, tableRowCounter, 1, 5, fExpandHor, vexpand, WidgetPlacement.Fill, WidgetPlacement.End, -1.0, -1.0, 0.0);
			m_tableRowCounter++;
		}
		return m_hboxButtons;
	}

	protected HBox AddToolBox(bool fExpandVert = false, bool fExpandHor = false, bool forceNewInstace = false)
	{
		if ((m_hboxTools == null) | forceNewInstace)
		{
			m_hboxTools = new HBox();
			Table table = GetTable();
			HBox hboxTools = m_hboxTools;
			int tableRowCounter = m_tableRowCounter;
			bool vexpand = fExpandVert;
			table.Add(hboxTools, 0, tableRowCounter, 1, 3, fExpandHor, vexpand, WidgetPlacement.Fill, WidgetPlacement.End, -1.0, -1.0, 0.0);
			m_tableRowCounter++;
		}
		return m_hboxTools;
	}

	protected void AddCustomButton(Button btnNew, bool rightSide = true)
	{
		AddButtonBox();
		if (rightSide)
		{
			m_hboxButtons.PackEnd(btnNew, expand: false);
		}
		else
		{
			m_hboxButtons.PackStart(btnNew, expand: false);
		}
	}

	protected Button AddImageButton(string iconName, string toolTipText, EventHandler clickEvent, bool enabled = false)
	{
		Button button = new Button(Image.FromResource(typeof(App), AppProperties.ResourcePath(iconName)))
		{
			Sensitive = enabled,
			Style = ButtonStyle.Borderless,
			TooltipText = toolTipText,
			BackgroundColor = BackgroundColor
		};
		button.Clicked += clickEvent;
		AddCustomButton(button, rightSide: false);
		return button;
	}

	protected ToggleButton AddToolImageToggleButton(string iconName, string toolTipText, EventHandler clickEvent, string id = "", bool enabled = false)
	{
		ToggleButton toggleButton = new ToggleButton(Image.FromResource(typeof(App), AppProperties.ResourcePath(iconName)))
		{
			Name = id,
			Sensitive = enabled,
			Style = ButtonStyle.Flat,
			TooltipText = toolTipText,
			BackgroundColor = BackgroundColor,
			MinWidth = 32.0,
			MinHeight = 32.0
		};
		toggleButton.Clicked += clickEvent;
		AddToolBox();
		m_hboxTools.PackStart(toggleButton, expand: false);
		return toggleButton;
	}

	protected Button AddToolImageButton(string iconName, string toolTipText, EventHandler clickEvent, bool rightSide = true, bool enabled = false)
	{
		Button button = new Button(Image.FromResource(typeof(App), AppProperties.ResourcePath(iconName)))
		{
			Sensitive = enabled,
			Style = ButtonStyle.Flat,
			TooltipText = toolTipText,
			BackgroundColor = BackgroundColor,
			MinWidth = 32.0,
			MinHeight = 32.0
		};
		button.Clicked += clickEvent;
		AddToolBox();
		if (rightSide)
		{
			m_hboxTools.PackEnd(button, expand: false);
		}
		else
		{
			m_hboxTools.PackStart(button, expand: false);
		}
		return button;
	}

	public Button AddCustomButton(string titleText, string toolTipText, EventHandler clickEvent, bool rightSide = true, bool floating = false)
	{
		Button button = new Button(titleText)
		{
			MinWidth = AppProperties.ButtonWidth,
			MinHeight = AppProperties.ButtonHeight
		};
		button.Clicked += clickEvent;
		button.Sensitive = false;
		button.TooltipText = toolTipText;
		if (floating)
		{
			button.MinHeight = 24.0;
			Add(button, 1, m_tableRowCounter, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Start, WidgetPlacement.End, UIPropertyBase.s_marginHor, marginRight: UIPropertyBase.s_marginHor, marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
			m_tableRowCounter++;
		}
		else
		{
			AddButtonBox();
			if (rightSide)
			{
				m_hboxButtons.PackEnd(button, expand: false);
			}
			else
			{
				m_hboxButtons.PackStart(button, expand: false);
			}
		}
		return button;
	}

	public Button AddCustomButton(string titleText, string toolTipText, EventHandler clickEvent, bool rightSide = true)
	{
		AddButtonBox();
		Button button = new Button(titleText)
		{
			MinWidth = AppProperties.ButtonWidth,
			MinHeight = AppProperties.ButtonHeight
		};
		button.Clicked += clickEvent;
		button.Sensitive = false;
		button.TooltipText = toolTipText;
		if (rightSide)
		{
			m_hboxButtons.PackEnd(button, expand: false);
		}
		else
		{
			m_hboxButtons.PackStart(button, expand: false);
		}
		return button;
	}

	public Button AddCustomImageButton(string iconName, string toolTipText, EventHandler clickEvent, bool rightSide = true, bool floating = false, object tag = null)
	{
		Button button = new Button(Image.FromResource(typeof(App), AppProperties.ResourcePath(iconName)));
		button.Clicked += clickEvent;
		button.Style = ButtonStyle.Borderless;
		button.Sensitive = false;
		button.BackgroundColor = Colors.Transparent;
		button.TooltipText = toolTipText;
		button.Tag = tag;
		if (floating)
		{
			button.MinHeight = 24.0;
			if (DoubleColumnTable == null)
			{
				Add(button, 1, m_tableRowCounter, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Start, WidgetPlacement.End, UIPropertyBase.s_marginHor, marginRight: UIPropertyBase.s_marginHor, marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
			}
			else
			{
				DoubleColumnTable.Add(button, m_tableColCounter, m_tableRowCounter, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Start, WidgetPlacement.End, UIPropertyBase.s_marginHor, marginRight: UIPropertyBase.s_marginHor, marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
			}
			m_tableRowCounter++;
		}
		else
		{
			AddButtonBox();
			if (rightSide)
			{
				m_hboxButtons.PackEnd(button, expand: false);
			}
			else
			{
				m_hboxButtons.PackStart(button, expand: false);
			}
		}
		return button;
	}

	public UIPropertyBase AddImage(Image image, string labelText = "", int borderWidth = 1, string featureRightId = "", uint tooltipID = 0u)
	{
		UIImage propBase = new UIImage(this, DoubleColumnTable ?? this, m_tableColCounter, m_tableRowCounter, labelText, image, borderWidth, m_nLeftMargin);
		return AddPropertyBase(featureRightId, propBase);
	}

	public int GetNumber(ushort usId, byte bSubId = 0)
	{
		if (m_currentDevice != null)
		{
			ICUProperty property = m_currentDevice.GetProperty(usId, bSubId);
			if (property != null)
			{
				return Convert.ToInt32(property.Value);
			}
		}
		return -1;
	}

	public void ChangeDevice(ICULanDevice device, bool forceRefresh = true)
	{
		m_newDevice = device;
		if (m_fIsVisible | forceRefresh)
		{
			OnShowPanel(m_fIsVisible, forceRefresh);
		}
	}

	protected virtual void OnShowPanelUser()
	{
	}

	protected virtual void OnShowPanel(bool show, bool forceRefresh = false)
	{
		if ((m_currentDevice != m_newDevice) | forceRefresh)
		{
			ICUDevice currentDevice = m_currentDevice;
			m_currentDevice = m_newDevice;
			bool flag = OnChangeDevice(m_currentDevice, currentDevice);
			foreach (UIPropertyBase lstProperty in m_lstProperties)
			{
				lstProperty.ChangeDevice(m_currentDevice);
			}
			SetUser(m_currentUser);
			if (flag)
			{
				OnChangeProperty();
			}
		}
		if (show)
		{
			OnShowPanelUser();
		}
	}

	public void AddToPropertyList(UIPropertyBase prop)
	{
		m_lstProperties.Add(prop);
	}

	public int SetProperties(List<ICUProperty> newPropertyValues)
	{
		int num = 0;
		foreach (UIPropertyBase prop in m_lstProperties)
		{
			ICUProperty iCUProperty = newPropertyValues.FirstOrDefault((ICUProperty a) => a.Id == prop.Id && a.SubId == prop.SubId);
			if (iCUProperty != null && iCUProperty.Value != null)
			{
				prop.SetValue(iCUProperty.Value);
				if (prop.IsChanged)
				{
					num++;
				}
			}
		}
		return num;
	}

	public void UpdateProperties(params ICUProperty[] propList)
	{
		if (m_currentDevice != null)
		{
			m_currentDevice.UpdateProperties(propList);
		}
	}

	public void UpdateDisplay(params ICUProperty[] properties)
	{
		foreach (ICUProperty updateProp in properties.Where((ICUProperty a) => a != null))
		{
			m_lstProperties.FirstOrDefault((UIPropertyBase a) => a.IsSameId(updateProp.Id, updateProp.SubId))?.ChangeDevice(m_currentDevice);
		}
	}

	public void UpdateDisplay(string category)
	{
		foreach (UIPropertyBase lstProperty in m_lstProperties)
		{
			ICUProperty iCUProperty = lstProperty.GetObject(m_currentDevice);
			if (iCUProperty != null && iCUProperty.Category.ToLowerInvariant() == category.ToLowerInvariant())
			{
				lstProperty.ChangeDevice(m_currentDevice);
			}
		}
	}

	public void UpdateDisplay()
	{
		foreach (UIPropertyBase lstProperty in m_lstProperties)
		{
			lstProperty.ChangeDevice(m_currentDevice);
		}
	}

	public virtual bool OnChangeDevice(ICUDevice currentDevice, ICUDevice previousDevice)
	{
		return false;
	}

	public virtual bool OnUpdateControls(string pageID = "")
	{
		return false;
	}

	public virtual bool OnChangeSCN(SCNNetwork network, string SCNName, LANConnection lanCon)
	{
		return false;
	}

	private void OnPropertyChanged(object sender, EventArgs e)
	{
		Changed?.Invoke(this, null);
		OnChangeProperty();
	}

	protected virtual void OnChangeProperty()
	{
		m_timOneShotUpdate.Interval = s_nUpdateControlsTimerMs;
		m_timOneShotUpdate.Stop();
		m_timOneShotUpdate.Start();
	}

	protected void OnShouldUpdate(object sender, ElapsedEventArgs e)
	{
		m_timOneShotUpdate.Stop();
		Application.Invoke(() =>
		{
			if (IsPanelVisible)
			{
				OnUpdateControls();
			}
		});
	}

	protected virtual bool OnValidatePropertyValue(ICUProperty prop, object newValue)
	{
		return true;
	}

	public bool AddHiddenProperty(ICUProperty newPropertyValue)
	{
		ICUProperty iCUProperty = m_lstHiddenProperties.FirstOrDefault((ICUProperty a) => a.Id == newPropertyValue.Id && a.SubId == newPropertyValue.SubId);
		if (iCUProperty != null)
		{
			iCUProperty.Value = newPropertyValue.Value;
		}
		else
		{
			m_lstHiddenProperties.Add(newPropertyValue);
		}
		return true;
	}

	public virtual bool OnSaveChanges()
	{
		return true;
	}

	public virtual void OnPostSaveChanges()
	{
	}

	public virtual void OnRevertChanges()
	{
	}

	public virtual void SaveChanges()
	{
		if (m_currentDevice == null)
		{
			return;
		}
		List<ICUProperty> list = new List<ICUProperty>();
		foreach (UIPropertyBase item in AllProperties.Where((UIPropertyBase a) => a.IsChanged).ToList())
		{
			object value = item.GetValue();
			ICUProperty iCUProperty = item.GetObject(m_currentDevice);
			if (iCUProperty != null && OnValidatePropertyValue(iCUProperty, value))
			{
				iCUProperty.Value = value;
				list.Add(iCUProperty);
			}
		}
		m_currentDevice.StoreProperties(list.ToArray());
	}
}
