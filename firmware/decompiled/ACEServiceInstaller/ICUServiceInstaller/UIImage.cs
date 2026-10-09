using System;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class UIImage : UIPropertyBase
{
	protected Label m_lblTitle;

	protected ImageView m_imageView;

	protected FrameBox m_fbImage;

	protected int m_nMaxLen = 256;

	protected string m_sCustomValue = "";

	protected bool m_fChanged;

	public UIImage(PanelBase panelParent, Table table, int col, int line, string labelText, Image image, int borderWidth = 1, int nLeftMargin = 0, uint tooltipID = 0u)
		: base(panelParent, labelText, tooltipID)
	{
		Initialize(table, image, col, line, borderWidth, nLeftMargin);
	}

	private void Initialize(Table table, Image image, int col, int line, int borderWidth, int nLeftMargin = 0)
	{
		m_lblTitle = new Label(LabelText)
		{
			Font = UIPropertyBase.s_fntBaseLabel,
			TooltipText = MakeToolTip(m_lblTitle.Text)
		};
		m_imageView = new ImageView(image);
		m_fbImage = new FrameBox
		{
			Content = m_imageView,
			BorderWidth = borderWidth,
			BorderColor = AppProperties.Color_Border
		};
		table.Add(m_lblTitle, col, line, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Start, UIPropertyBase.s_marginHor + nLeftMargin, marginRight: Math.Max(UIPropertyBase.s_marginHorMax - nLeftMargin, 0), marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
		table.Add(m_fbImage, col + 1, line, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Start, WidgetPlacement.Fill, UIPropertyBase.s_marginHor, marginRight: UIPropertyBase.s_marginHor, marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
	}

	public override void SetTooltip(string text)
	{
		m_lblTitle.TooltipText = text;
	}

	public override void SetVisible(bool fVisible)
	{
		m_lblTitle.Visible = fVisible;
		m_fbImage.Visible = fVisible;
		m_imageView.Visible = fVisible;
	}
}
