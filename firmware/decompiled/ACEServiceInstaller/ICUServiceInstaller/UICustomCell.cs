using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class UICustomCell : CanvasCellView
{
	protected enum ECellOptions
	{
		None,
		Bold,
		Border
	}

	protected IDataField<string> m_dfValueField;

	protected IDataField<UICustomField> m_dfCustomField;

	protected IDataField<int> m_dfOptionField;

	protected Color m_colText = Colors.Black;

	protected Color m_colBG = Colors.Transparent;

	protected int m_nWidth = 100;

	protected TextLayout m_taLayout;

	public UICustomCell(IDataField<UICustomField> valueField, int width, IDataField<int> optionField = null)
	{
		m_dfCustomField = valueField;
		Initialize(width, optionField);
	}

	public UICustomCell(IDataField<string> valueField, int width, IDataField<int> optionField = null, Color? txCol = null, Color? bgCol = null)
	{
		m_dfValueField = valueField;
		m_colText = txCol ?? Colors.Black;
		m_colBG = bgCol ?? Colors.Transparent;
		Initialize(width, optionField);
	}

	private void Initialize(int width, IDataField<int> optionField)
	{
		m_dfOptionField = optionField;
		m_nWidth = width;
		m_taLayout = new TextLayout();
	}

	protected override Size OnGetRequiredSize(SizeConstraint sizeConstraint)
	{
		return new Size(m_nWidth, 16.0);
	}

	protected override void OnDraw(Context ctx, Rectangle cellArea)
	{
		double num = 0.0;
		if (!Visible)
		{
			return;
		}
		m_taLayout.ClearAttributes();
		int value = GetValue(m_dfOptionField, 0);
		if (value == 1)
		{
			m_taLayout.Font = Font.SystemFont.WithSize(12.0).WithWeight(FontWeight.Semibold);
		}
		else
		{
			m_taLayout.Font = Font.SystemFont.WithSize(12.0);
		}
		if (value == 2)
		{
			ctx.SetColor(Colors.Black);
			ctx.SetLineWidth(4.0);
			ctx.MoveTo(cellArea.Left, cellArea.Top + cellArea.Height / 2.0 - 1.0);
			ctx.LineTo(cellArea.Right, cellArea.Top + cellArea.Height / 2.0 - 1.0);
		}
		else if (m_dfValueField != null)
		{
			string value2 = GetValue(m_dfValueField);
			if (value2 != null)
			{
				m_taLayout.Text = value2;
				num = 2.0;
				ctx.Rectangle(Bounds);
				ctx.SetColor(m_colBG);
				ctx.FillPreserve();
				m_taLayout.SetForeground(m_colText, 0, m_taLayout.Text.Length);
				ctx.DrawTextLayout(m_taLayout, cellArea.Left + 4.0, cellArea.Top + num);
			}
		}
		else if (m_dfCustomField != null)
		{
			UICustomField value3 = GetValue(m_dfCustomField);
			if (value3.text != null)
			{
				m_taLayout.Text = value3.text;
				num = 0.0;
				ctx.Rectangle(Bounds);
				ctx.SetColor(value3.bgColor);
				ctx.FillPreserve();
				m_taLayout.SetForeground(value3.txColor, 0, m_taLayout.Text.Length);
				ctx.DrawTextLayout(m_taLayout, cellArea.Left + 4.0, cellArea.Top + num);
			}
		}
	}
}
