using System;
using System.Diagnostics;
using ICUNetwork;
using Serilog;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class UIPropertyLabel : UIPropertyBase
{
	protected string m_sSimpleLabelText = "";

	protected Label m_lblTitle;

	protected Color m_colText = Colors.SteelBlue;

	protected Color m_colDisabled = Colors.DarkGray;

	protected Label m_lblHyperlink;

	protected string m_sHyperlinkUrl = "";

	protected VBox m_boxContainer;

	public UIPropertyLabel(PanelBase panelParent, Table table, int col, int line, string labelText, EUILabelType header, int nLeftMargin = 0, int reserveLines = 1)
		: base(panelParent, labelText)
	{
		m_sSimpleLabelText = labelText;
		Initialize(table, col, line, header, nLeftMargin, reserveLines);
	}

	private void Initialize(Table table, int col, int line, EUILabelType header, int nLeftMargin, int reserveLines)
	{
		m_lblTitle = new Label(m_sSimpleLabelText);
		int num = m_sSimpleLabelText.Split(new char[1] { '\n' }).Length;
		num = ((num >= reserveLines) ? num : reserveLines);
		VBox vBox = new VBox();
		vBox.Spacing = 0.0;
		m_boxContainer = vBox;
		vBox.PackStart(m_lblTitle);
		switch (header)
		{
		case EUILabelType.Header:
			m_lblTitle.Font = UIPropertyBase.s_fntBaseLabel.WithScaledSize(1.5).WithWeight(FontWeight.Semibold);
			m_lblTitle.Wrap = WrapMode.Word;
			m_lblTitle.TextAlignment = Alignment.Start;
			m_colText = Colors.SteelBlue;
			SetWidgetSize(m_lblTitle, 0, 3);
			table.Add(vBox, col, line, 1, 3, hexpand: false, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, UIPropertyBase.s_marginHor + nLeftMargin, marginRight: Math.Max(UIPropertyBase.s_marginHorMax - nLeftMargin, 0), marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
			break;
		case EUILabelType.SmallHeader:
			m_lblTitle.Font = UIPropertyBase.s_fntBaseLabel.WithScaledSize(1.1).WithWeight(FontWeight.Semibold);
			m_lblTitle.Wrap = WrapMode.Word;
			m_lblTitle.TextAlignment = Alignment.Start;
			m_colText = Colors.SteelBlue;
			SetWidgetSize(m_lblTitle, 0, 3);
			table.Add(vBox, col, line, 1, 3, hexpand: false, vexpand: false, WidgetPlacement.Start, WidgetPlacement.Fill, UIPropertyBase.s_marginHor + nLeftMargin, marginRight: Math.Max(UIPropertyBase.s_marginHorMax - nLeftMargin, 0), marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
			break;
		case EUILabelType.Warning:
		case EUILabelType.Info:
		case EUILabelType.Error:
		case EUILabelType.LargeWarning:
		case EUILabelType.LargeInfo:
		case EUILabelType.LargeError:
			m_lblTitle.Wrap = WrapMode.Word;
			m_lblTitle.TextAlignment = Alignment.Start;
			m_colText = Colors.Black;
			SetType(header);
			SetWidgetSize(m_lblTitle, 0, 3, num - 1);
			table.Add(vBox, col, line, 1, 3, hexpand: false, vexpand: false, WidgetPlacement.Start, WidgetPlacement.Fill, UIPropertyBase.s_marginHor + nLeftMargin, marginRight: Math.Max(UIPropertyBase.s_marginHorMax - nLeftMargin, 0), marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
			break;
		default:
			m_lblTitle.Font = UIPropertyBase.s_fntBaseLabel;
			m_lblTitle.Wrap = WrapMode.Word;
			m_lblTitle.TextAlignment = Alignment.Center;
			m_colText = Colors.Black;
			SetWidgetSize(m_lblTitle, 0, 1, num - 1);
			table.Add(vBox, col, line, 1, 1, hexpand: true, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, UIPropertyBase.s_marginHor + nLeftMargin, marginRight: Math.Max(UIPropertyBase.s_marginHorMax - nLeftMargin, 0), marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
			break;
		}
		RefreshDisplay();
	}

	public override void SetValue(object newValue)
	{
		string text = newValue.ToString();
		if (m_sSimpleLabelText != text && !string.IsNullOrEmpty(m_sSimpleLabelText))
		{
			ClearHyperlink();
		}
		m_sSimpleLabelText = text;
		if (m_lblTitle != null)
		{
			m_lblTitle.Text = m_sSimpleLabelText;
		}
	}

	public override void SetMinWidth(int minWidth)
	{
		m_lblTitle.MinWidth = minWidth;
	}

	public void SetType(EUILabelType header)
	{
		switch (header)
		{
		case EUILabelType.Info:
		case EUILabelType.LargeInfo:
			m_colText = Colors.SteelBlue;
			m_lblTitle.Font = UIPropertyBase.s_fntBaseLabel.WithScaledSize(1.0).WithWeight(FontWeight.Normal);
			break;
		case EUILabelType.Warning:
			m_colText = Colors.Orange;
			m_lblTitle.Font = UIPropertyBase.s_fntBaseLabel.WithScaledSize(1.0).WithWeight(FontWeight.Normal);
			break;
		case EUILabelType.LargeWarning:
			m_colText = Colors.Orange;
			m_lblTitle.Font = UIPropertyBase.s_fntBaseLabel.WithScaledSize(1.1).WithWeight(FontWeight.Bold);
			break;
		case EUILabelType.Error:
		case EUILabelType.LargeError:
			m_colText = Colors.Red;
			m_lblTitle.Font = UIPropertyBase.s_fntBaseLabel.WithScaledSize(1.1).WithWeight(FontWeight.Semibold);
			break;
		}
		RefreshDisplay();
	}

	public override void SetVisible(bool fVisible)
	{
		m_lblTitle.Visible = fVisible;
		if (m_lblHyperlink != null)
		{
			m_lblHyperlink.Visible = fVisible;
		}
	}

	public override void OnRefreshDisplay(ICUDevice device)
	{
		if (!m_fEnabled)
		{
			m_lblTitle.TextColor = m_colDisabled;
		}
		else
		{
			m_lblTitle.TextColor = m_colText;
		}
	}

	public void SetHyperlink(string linkText, string url)
	{
		m_sHyperlinkUrl = url;
		if (m_lblHyperlink != null && m_boxContainer != null)
		{
			m_boxContainer.Remove(m_lblHyperlink);
			m_lblHyperlink = null;
		}
		if (string.IsNullOrEmpty(url) || m_boxContainer == null)
		{
			return;
		}
		m_lblHyperlink = new Label(linkText);
		m_lblHyperlink.Font = m_lblTitle.Font.WithWeight(FontWeight.Semibold);
		m_lblHyperlink.TextColor = Colors.Blue;
		m_lblHyperlink.Cursor = CursorType.Hand;
		m_lblHyperlink.TextAlignment = Alignment.Start;
		m_lblHyperlink.ButtonPressed += (object sender, ButtonEventArgs e) =>
		{
			if (e.Button == PointerButton.Left)
			{
				try
				{
					Process.Start(new ProcessStartInfo
					{
						FileName = m_sHyperlinkUrl,
						UseShellExecute = true
					});
				}
				catch (Exception exception)
				{
					Log.Error(exception, "Failed to open URL: {Url}", m_sHyperlinkUrl);
				}
			}
		};
		m_lblHyperlink.MouseEntered += (object sender, EventArgs e) =>
		{
			m_lblHyperlink.TextColor = Colors.DarkBlue;
		};
		m_lblHyperlink.MouseExited += (object sender, EventArgs e) =>
		{
			m_lblHyperlink.TextColor = Colors.Blue;
		};
		m_boxContainer.PackStart(m_lblHyperlink);
		m_lblHyperlink.MarginTop = -15.0;
		m_lblHyperlink.Visible = m_lblTitle.Visible;
	}

	public void ClearHyperlink()
	{
		if (m_lblHyperlink != null && m_boxContainer != null)
		{
			m_boxContainer.Remove(m_lblHyperlink);
			m_lblHyperlink = null;
		}
		m_sHyperlinkUrl = "";
	}
}
