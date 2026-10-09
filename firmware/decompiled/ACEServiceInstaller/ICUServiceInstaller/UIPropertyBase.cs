using System;
using ICUNetwork;
using ICUSettings;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class UIPropertyBase
{
	public static int s_marginHor = 4;

	public static int s_marginVer = 1;

	public static int s_marginHorMax = 20;

	protected static int s_propertyMinHeight = 20;

	protected static int s_propertyHeaderHeight = 30;

	public static Image s_imgEmpty = Image.FromResource(typeof(App), AppProperties.ResourcePath("empty16.png"));

	protected static Image s_imgChanged = Image.FromResource(typeof(App), AppProperties.ResourcePath("pencil.png"));

	protected static Image s_imgInformation = Image.FromResource(typeof(App), AppProperties.ResourcePath("information.png")).Scale(0.5);

	protected static Image s_imgValid = Image.FromResource(typeof(App), AppProperties.ResourcePath("Valid.png"));

	protected static Image s_imgError = Image.FromResource(typeof(App), AppProperties.ResourcePath("Error.png"));

	protected static Image s_imgWarning = Image.FromResource(typeof(App), AppProperties.ResourcePath("Warning.png"));

	public static Font s_fntBaseLabel = AppProperties.Font_BaseLabel;

	public static Font s_fntBase = AppProperties.Font_Base;

	protected EDSParameter m_edsParameter;

	protected bool m_fUnknownTitle;

	protected ushort m_usId;

	protected byte m_bSubId;

	protected ImageView m_imvChanged;

	protected ICUProperty m_objCurrentValue;

	protected string m_sLabelText = "";

	protected PanelBase m_panelParent;

	protected bool m_fReadOnly;

	protected bool m_fForceReadOnly;

	protected bool m_fAlwaysReadOnly;

	protected bool m_fHidden;

	protected bool m_fEnabled = true;

	protected ICUDevice m_currentDevice;

	protected string m_tooltipText = "";

	protected Button m_btnTooltip;

	public ushort Id => m_usId;

	public byte SubId => m_bSubId;

	public ICURights AccessRights { get; set; }

	public string FeatureRightID { get; set; }

	public bool IsAdvancedProp { get; set; }

	public bool Hide { get; set; }

	public bool IsConfidential { get; set; }

	public string TooltipID { get; set; }

	protected string LabelText
	{
		get
		{
			string empty = string.Empty;
			if (m_edsParameter != null)
			{
				empty = m_edsParameter.Title;
				if (m_edsParameter.Units != null)
				{
					empty = $"{empty} ({m_edsParameter.Units})";
				}
				m_fUnknownTitle = false;
			}
			else if (m_usId == 0 && m_bSubId == 0)
			{
				empty = m_sLabelText;
			}
			else
			{
				empty = ((m_bSubId == 0) ? $"{m_usId:X4}" : $"{m_usId:X4}_{m_bSubId}");
				m_fUnknownTitle = true;
			}
			return empty;
		}
	}

	public virtual bool IsChanged
	{
		get
		{
			if (m_objCurrentValue != null)
			{
				return m_objCurrentValue.IsChanged;
			}
			return false;
		}
	}

	public event EventHandler Changed;

	public void SetHide(bool hide)
	{
		Hide = hide;
	}

	public UIPropertyBase(PanelBase panelParent, string labelText = "", uint tooltipID = 0u)
	{
		m_panelParent = panelParent;
		m_usId = 0;
		m_bSubId = 0;
		m_imvChanged = new ImageView(s_imgEmpty);
		m_imvChanged.MinWidth = 24.0;
		m_imvChanged.MinHeight = 24.0;
		m_imvChanged.HorizontalPlacement = WidgetPlacement.Start;
		m_imvChanged.TooltipText = "This property is changed";
		m_edsParameter = null;
		m_sLabelText = labelText;
		TooltipID = MakeToolTipID(tooltipID);
		m_btnTooltip = new Button(s_imgInformation)
		{
			BackgroundColor = Colors.Transparent,
			Style = ButtonStyle.Flat
		};
		m_btnTooltip.Clicked += OnBtnTooltipClicked;
		m_tooltipText = Tooltips.GetTooltip(TooltipID);
		m_btnTooltip.Visible = !string.IsNullOrEmpty(m_tooltipText);
	}

	~UIPropertyBase()
	{
		ClearChangedHandlers();
	}

	public virtual void OnBtnTooltipClicked(object sender, EventArgs e)
	{
		new DlgInfo(Tooltips.GetTooltip(TooltipID), showInTaskbar: true).Run();
	}

	public UIPropertyBase(PanelBase panelParent, ushort Id, byte subId = 0, uint tooltipID = 0u)
	{
		m_panelParent = panelParent;
		m_usId = Id;
		m_bSubId = subId;
		m_imvChanged = new ImageView(s_imgEmpty);
		m_imvChanged.TooltipText = "This property is changed";
		m_imvChanged.HorizontalPlacement = WidgetPlacement.Start;
		m_edsParameter = DataSheet.FindParameter(Id, subId);
		TooltipID = MakeToolTipID(tooltipID);
		m_btnTooltip = new Button(s_imgInformation)
		{
			BackgroundColor = Colors.Transparent,
			Style = ButtonStyle.Flat
		};
		m_btnTooltip.Clicked += OnBtnTooltipClicked;
		m_tooltipText = Tooltips.GetTooltip(TooltipID);
		m_btnTooltip.Visible = !string.IsNullOrEmpty(m_tooltipText);
	}

	protected void SetWidgetSizes(Widget widLabel, Widget widChanged = null, Widget widControl = null, Widget widTooltip = null, int extraLines = 0)
	{
		if (widChanged == null && widControl == null)
		{
			SetWidgetSize(widLabel, 0, 3);
		}
		else if (widChanged != null && widControl == null)
		{
			SetWidgetSize(widLabel, 0, 2);
			SetWidgetSize(widChanged, 2, 1);
		}
		else if (widControl != null && widChanged != null && widTooltip == null)
		{
			SetWidgetSize(widLabel, 0, 1);
			SetWidgetSize(widControl, 1, 1, extraLines);
			SetWidgetSize(widChanged, 2, 1);
		}
		else if (widControl != null && widChanged != null && widTooltip != null)
		{
			SetWidgetSize(widLabel, 0, 1);
			SetWidgetSize(widControl, 1, 1, extraLines);
			SetWidgetSize(widChanged, 2, 1);
			SetWidgetSize(widTooltip, 3, 1);
		}
	}

	protected void SetWidgetSize(Widget wid, int colStart, int colSpan, int extraLines = 0)
	{
		if (m_panelParent != null && m_panelParent.DoubleColumnTable == null)
		{
			SetWidgetSizeSingleColumn(wid, colStart, colSpan, extraLines);
		}
		else
		{
			SetWidgetSizeDoubleColumn(wid, colStart, colSpan, extraLines);
		}
	}

	public static void SetWidgetSizeSingleColumn(Widget wid, int colStart, int colSpan, int extraLines = 0)
	{
		int[] array = new int[4] { 190, 255, 24, 24 };
		int num = 0;
		for (int i = colStart; i < colStart + colSpan; i++)
		{
			num += array[i];
		}
		wid.HeightRequest = 24 + extraLines * 16;
		wid.WidthRequest = num;
	}

	public static void SetWidgetSizeDoubleColumn(Widget wid, int colStart, int colSpan, int extraLines = 0)
	{
		int[] array = new int[4] { 120, 160, 24, 24 };
		int num = 0;
		for (int i = colStart; i < colStart + colSpan; i++)
		{
			num += array[i];
		}
		wid.HeightRequest = 24 + extraLines * 16;
		wid.WidthRequest = num;
	}

	public bool IsSameId(ushort usId, byte bSubId)
	{
		if (m_usId == usId)
		{
			return m_bSubId == bSubId;
		}
		return false;
	}

	public void ForceReadonly(bool fForce)
	{
		m_fAlwaysReadOnly = fForce;
	}

	protected string MakeToolTipID(uint id)
	{
		if (id == 0)
		{
			return $"{m_usId:X4}_{m_bSubId:X2}";
		}
		return $"{id >> 8:X4}_{id & 0xFF:X2}";
	}

	protected string MakeToolTip(string text)
	{
		Tooltips.InitToolTip();
		if (string.IsNullOrEmpty(text))
		{
			return text;
		}
		return Tooltips.GetTooltip(TooltipID);
	}

	public virtual void SetTooltip(string text)
	{
	}

	public void SetToolTipIcon(EStatusIcon icon)
	{
		switch (icon)
		{
		case EStatusIcon.STATUS_ICON_VALID:
			m_btnTooltip.Image = s_imgValid;
			break;
		case EStatusIcon.STATUS_ICON_WARNING:
			m_btnTooltip.Image = s_imgWarning;
			break;
		case EStatusIcon.STATUS_ICON_ERROR:
			m_btnTooltip.Image = s_imgError;
			break;
		default:
			m_btnTooltip.Image = s_imgInformation;
			break;
		}
	}

	public ICUProperty GetObject(ICUDevice device)
	{
		return device?.GetProperty(m_usId, m_bSubId);
	}

	public void ChangeDevice(ICUDevice device)
	{
		if (m_objCurrentValue != null)
		{
			m_objCurrentValue.ValueChanged -= OnPropertyValueChanged;
			m_objCurrentValue.ExceptionOccured -= OnPropertyValueException;
		}
		m_currentDevice = device;
		if (m_currentDevice != null)
		{
			m_objCurrentValue = device.GetProperty(m_usId, m_bSubId);
			if (m_objCurrentValue != null)
			{
				m_objCurrentValue.ValueChanged += OnPropertyValueChanged;
				m_objCurrentValue.ExceptionOccured += OnPropertyValueException;
				if (m_objCurrentValue.IsChanged)
				{
					m_objCurrentValue.FireChanged();
				}
			}
		}
		else
		{
			m_objCurrentValue = null;
		}
		OnDeviceChange(m_currentDevice);
	}

	private void OnPropertyValueChanged(object sender, EventArgs e)
	{
		ICUProperty prop = sender as ICUProperty;
		if (prop != null)
		{
			Application.Invoke(() =>
			{
				OnValueChanged(prop);
				FireChange();
			});
		}
	}

	private void OnPropertyValueException(object sender, ValueExceptionEventArgs exc)
	{
		if (sender is ICUProperty iCUProperty)
		{
			using DlgInfo dlgInfo = new DlgInfo("Configuration item: " + iCUProperty.Title + " has an invalid value: " + exc.InvalidValue + ". Exception: " + exc.Exception.Message + " ", showInTaskbar: false);
			dlgInfo.Run();
		}
	}

	protected void FireChange()
	{
		Changed?.Invoke(this, EventArgs.Empty);
	}

	public void ClearValue()
	{
		OnClearValue();
	}

	protected virtual void OnDeviceChange(ICUDevice device)
	{
	}

	protected virtual void OnValueChanged(ICUProperty prop)
	{
	}

	protected virtual void OnClearValue()
	{
	}

	public virtual object GetValue()
	{
		if (m_objCurrentValue != null)
		{
			return m_objCurrentValue.Value;
		}
		return null;
	}

	public virtual void SetValue(object newValue)
	{
		if (m_objCurrentValue != null)
		{
			m_objCurrentValue.Value = newValue;
		}
	}

	public void SetProperty(ICUProperty property)
	{
		m_objCurrentValue = property;
		if (m_objCurrentValue != null)
		{
			m_objCurrentValue.ValueChanged -= OnPropertyValueChanged;
			SetValue(m_objCurrentValue.Value);
			m_objCurrentValue.ValueChanged += OnPropertyValueChanged;
			RefreshDisplay();
		}
	}

	public ICUProperty GetProperty()
	{
		return m_objCurrentValue;
	}

	public virtual void OnChanged()
	{
	}

	public virtual void SetVisible(bool fVisible)
	{
	}

	public virtual void SetMinWidth(int minWidth)
	{
	}

	public virtual void SetEnable(bool fEnable)
	{
		m_fEnabled = fEnable;
		RefreshDisplay();
	}

	public virtual bool IsEnabled()
	{
		return m_fEnabled;
	}

	public virtual void CheckAccessRights(ICUGroup newGroup, ICURights parentRights)
	{
		m_fForceReadOnly = m_fAlwaysReadOnly || parentRights == ICURights.ReadOnly || newGroup.GetRights(FeatureRightID) == ICURights.ReadOnly;
		m_fHidden = parentRights == ICURights.None || newGroup.GetRights(FeatureRightID) == ICURights.None;
		RefreshDisplay();
	}

	public void RefreshDisplay()
	{
		OnRefreshDisplay(m_currentDevice);
	}

	public virtual void OnRefreshDisplay(ICUDevice device = null)
	{
	}

	protected static DateTime UnixTimeStampToDateTime(ulong unixTimeStamp)
	{
		return new DateTime(1970, 1, 1, 0, 0, 0, 0, DateTimeKind.Utc).AddSeconds(unixTimeStamp);
	}

	protected static TimeSpan UnixTimeStampToTimeSpan(ulong unixTimeStamp)
	{
		DateTime dateTime = new DateTime(1970, 1, 1, 0, 0, 0, 0, DateTimeKind.Utc);
		return dateTime.AddSeconds(unixTimeStamp).ToUniversalTime() - dateTime;
	}

	public void ClearChangedHandlers()
	{
		if (Changed != null)
		{
			Delegate[] invocationList = Changed.GetInvocationList();
			foreach (Delegate obj in invocationList)
			{
				Changed -= (EventHandler)obj;
			}
		}
	}
}
