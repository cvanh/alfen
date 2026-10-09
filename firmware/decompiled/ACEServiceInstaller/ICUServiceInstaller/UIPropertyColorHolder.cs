using System;
using System.Collections.Generic;
using System.Linq;
using ICUNetwork;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class UIPropertyColorHolder : UIPropertyBase
{
	private Label m_lblTitle = new Label();

	private readonly ListView m_lvwColor = new ListView();

	private ListStore m_lsColorStore;

	private readonly DataField<UIPropertyColor> m_dfColor = new DataField<UIPropertyColor>();

	private readonly DataField<string> m_dfColorName = new DataField<string>();

	private readonly DataField<Color> m_dfColorColor1 = new DataField<Color>();

	private readonly DataField<string> m_dfColorTime1 = new DataField<string>();

	private readonly DataField<Color> m_dfColorColor2 = new DataField<Color>();

	private readonly DataField<string> m_dfColorTime2 = new DataField<string>();

	private readonly DataField<Image> m_dfColorChanged = new DataField<Image>();

	private readonly Label m_lblColor1 = new Label("Color 1");

	private readonly Label m_lblColor2 = new Label("Color 2");

	private readonly ColorPicker m_pickerColor1 = new ColorPicker();

	private readonly SpinButton m_spnTime1 = new SpinButton();

	private readonly ColorPicker m_pickerColor2 = new ColorPicker();

	private readonly SpinButton m_spnTime2 = new SpinButton();

	private readonly Table m_hbColorStuff = new Table();

	protected List<UIPropertyColor> m_lstColors = new List<UIPropertyColor>();

	protected bool m_fAllowChangedEvents = true;

	protected bool m_fAllowSelectionChange = true;

	protected bool m_fSensitive = true;

	public bool Sensitive
	{
		get
		{
			return m_fSensitive;
		}
		set
		{
			m_fSensitive = value;
			m_lvwColor.Sensitive = m_fSensitive;
			m_lblTitle.Sensitive = m_fSensitive;
			m_imvChanged.Sensitive = m_fSensitive;
			m_hbColorStuff.Sensitive = m_fSensitive;
		}
	}

	public UIPropertyColorHolder(PanelBase panelParent, Table table, int col, int line, string labelText, int nLeftMargin = 0, uint tooltipID = 0u)
		: base(panelParent, 0, 0, tooltipID)
	{
		Initialize(table, col, line, labelText, nLeftMargin);
	}

	public void Initialize(Table table, int col, int line, string labelText, int nLeftMargin = 0)
	{
		m_lsColorStore = new ListStore(m_dfColor, m_dfColorName, m_dfColorColor1, m_dfColorTime1, m_dfColorColor2, m_dfColorTime2, m_dfColorChanged);
		m_lvwColor.Font = UIPropertyBase.s_fntBase;
		m_lvwColor.SelectionMode = SelectionMode.Single;
		m_lvwColor.DataSource = m_lsColorStore;
		m_lvwColor.WidthRequest = 200.0;
		m_lvwColor.HeightRequest = 124.0;
		m_lvwColor.GridLinesVisible = GridLines.Horizontal;
		m_lvwColor.Columns.Add(new ListViewColumn("Name                                         ", new TextCellView(m_dfColorName)));
		m_lvwColor.Columns.Add(new ListViewColumn("Color 1 ", new ColorCell(m_dfColorColor1)));
		m_lvwColor.Columns.Add(new ListViewColumn("Time 1  ", new TextCellView(m_dfColorTime1)));
		m_lvwColor.Columns.Add(new ListViewColumn("Color 2 ", new ColorCell(m_dfColorColor2)));
		m_lvwColor.Columns.Add(new ListViewColumn("Time 2  ", new TextCellView(m_dfColorTime2)));
		m_lvwColor.SelectionChanged += OnSelectionChanged;
		m_lvwColor.Columns.Add(new ListViewColumn("        ", new ImageCellView(m_dfColorChanged)));
		m_lblColor1.Font = UIPropertyBase.s_fntBaseLabel;
		m_lblColor2.Font = UIPropertyBase.s_fntBaseLabel;
		InitializeSpinButton(m_spnTime1);
		InitializeSpinButton(m_spnTime2);
		m_lblTitle = new Label(labelText)
		{
			Font = UIPropertyBase.s_fntBaseLabel.WithScaledSize(1.1).WithWeight(FontWeight.Semibold),
			TextColor = Colors.SteelBlue,
			MinHeight = 21.0
		};
		m_imvChanged.MinHeight = 21.0;
		SetWidgetSizes(m_lblTitle, m_imvChanged, m_lvwColor);
		m_lvwColor.HeightRequest = 124.0;
		table.Add(m_lblTitle, col, line, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Start, UIPropertyBase.s_marginHor + nLeftMargin, marginRight: Math.Max(UIPropertyBase.s_marginHorMax - nLeftMargin, 0), marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
		table.Add(m_lvwColor, col, line + 1, 1, 2, hexpand: true, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, UIPropertyBase.s_marginHor, marginRight: UIPropertyBase.s_marginHor, marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
		table.Add(m_imvChanged, col + 3, line, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.End, WidgetPlacement.Start, 1.0, UIPropertyBase.s_marginVer, 1.0, UIPropertyBase.s_marginVer);
		m_hbColorStuff.ExpandVertical = false;
		m_pickerColor1.HeightRequest = 32.0;
		m_pickerColor1.Style = ButtonStyle.Flat;
		m_pickerColor2.HeightRequest = 32.0;
		m_pickerColor2.Style = ButtonStyle.Flat;
		Label widget = new Label("Period (ms)");
		m_hbColorStuff.Add(m_lblColor1, 0, 0, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Start, WidgetPlacement.Center);
		m_hbColorStuff.Add(m_pickerColor1, 1, 0, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Center, WidgetPlacement.Center);
		m_hbColorStuff.Add(widget, 2, 0, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.End, WidgetPlacement.Center);
		m_hbColorStuff.Add(m_spnTime1, 3, 0, 1, 1, hexpand: true, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Center);
		Label widget2 = new Label("Period (ms)");
		m_hbColorStuff.Add(m_lblColor2, 4, 0, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Start, WidgetPlacement.Center);
		m_hbColorStuff.Add(m_pickerColor2, 5, 0, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Center, WidgetPlacement.Center);
		m_hbColorStuff.Add(widget2, 6, 0, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.End, WidgetPlacement.Center);
		m_hbColorStuff.Add(m_spnTime2, 7, 0, 1, 1, hexpand: true, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Center);
		m_pickerColor1.ColorChanged += OnColorPatternChanged;
		m_pickerColor2.ColorChanged += OnColorPatternChanged;
		m_spnTime1.ValueChanged += OnColorPatternChanged;
		m_spnTime2.ValueChanged += OnColorPatternChanged;
		m_hbColorStuff.HeightRequest = 35.0;
		line++;
		table.Add(m_hbColorStuff, col, line + 1, 1, 2, hexpand: true, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, UIPropertyBase.s_marginHor, marginRight: UIPropertyBase.s_marginHor, marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
	}

	public void Add(UIPropertyColor uiProp)
	{
		uiProp.Changed += OnColorChanged;
		m_lstColors.Add(uiProp);
		RefreshDisplay();
	}

	private void OnColorChanged(object sender, EventArgs e)
	{
		m_imvChanged.Image = ((m_lvwColor.Sensitive && m_lstColors.Where((UIPropertyColor a) => a.IsChanged).Count() > 0) ? UIPropertyBase.s_imgChanged : UIPropertyBase.s_imgEmpty);
		RefreshDisplay();
	}

	public override void OnRefreshDisplay(ICUDevice device)
	{
		if (m_lsColorStore == null)
		{
			return;
		}
		m_fAllowSelectionChange = false;
		int num = m_lvwColor.SelectedRow;
		if (num < 0)
		{
			num = 0;
		}
		m_lsColorStore.Clear();
		foreach (UIPropertyColor lstColor in m_lstColors)
		{
			int row = m_lsColorStore.AddRow();
			m_lsColorStore.SetValues(row, m_dfColor, lstColor, m_dfColorName, lstColor.Name, m_dfColorColor1, lstColor.State1Color, m_dfColorTime1, lstColor.State1Time.ToString(), m_dfColorColor2, lstColor.State2Color, m_dfColorTime2, lstColor.State2Time.ToString(), m_dfColorChanged, lstColor.IsChanged ? UIPropertyBase.s_imgChanged : UIPropertyBase.s_imgEmpty);
		}
		if (num < m_lsColorStore.RowCount)
		{
			m_lvwColor.SelectRow(num);
		}
		m_fAllowSelectionChange = true;
		OnSelectionChanged(null, null);
		FireChange();
	}

	private void InitializeSpinButton(SpinButton sb)
	{
		sb.Digits = 0;
		sb.MinimumValue = 0.0;
		sb.MaximumValue = 2550.0;
		sb.IncrementValue = 10.0;
		sb.MinWidth = 80.0;
	}

	public override void SetVisible(bool fVisible)
	{
		m_lvwColor.Visible = fVisible;
		m_lblTitle.Visible = fVisible;
		m_imvChanged.Visible = fVisible;
		m_hbColorStuff.Visible = fVisible;
	}

	private void OnSelectionChanged(object sender, EventArgs e)
	{
		if (m_fAllowSelectionChange)
		{
			UIPropertyColor selectedColor = GetSelectedColor();
			if (selectedColor != null)
			{
				m_fAllowChangedEvents = false;
				m_pickerColor1.Color = selectedColor.State1Color;
				m_spnTime1.Value = selectedColor.State1Time;
				m_pickerColor2.Color = selectedColor.State2Color;
				m_spnTime2.Value = selectedColor.State2Time;
				m_fAllowChangedEvents = true;
			}
		}
	}

	protected override void OnDeviceChange(ICUDevice device)
	{
		if (device == null)
		{
			m_lblTitle.Sensitive = false;
			m_lvwColor.Sensitive = false;
			m_hbColorStuff.Sensitive = false;
		}
		else
		{
			m_lblTitle.Sensitive = m_fSensitive && m_lstColors.Count > 0;
			m_lvwColor.Sensitive = m_fSensitive && m_lstColors.Count > 0;
			m_hbColorStuff.Sensitive = m_fSensitive && m_lstColors.Count > 0;
		}
		foreach (UIPropertyColor lstColor in m_lstColors)
		{
			lstColor.ChangeDevice(device);
		}
		RefreshDisplay();
	}

	private UIPropertyColor GetSelectedColor()
	{
		if (m_lvwColor == null)
		{
			return null;
		}
		int selectedRow = m_lvwColor.SelectedRow;
		if (selectedRow < 0)
		{
			return null;
		}
		return m_lsColorStore.GetValue(selectedRow, m_dfColor);
	}

	private void OnColorPatternChanged(object sender, EventArgs e)
	{
		if (m_fAllowChangedEvents)
		{
			UIPropertyColor selectedColor = GetSelectedColor();
			if (selectedColor != null)
			{
				Color color = m_pickerColor1.Color;
				double value = m_spnTime1.Value;
				Color col = (selectedColor.State2Color = m_pickerColor2.Color);
				selectedColor.SetValue(color, value, col, m_spnTime2.Value);
				RefreshDisplay();
			}
		}
	}
}
