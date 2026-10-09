using System;
using ICUNetwork;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class CustomLogCell : CanvasCellView
{
	protected static Color[] s_arrBackGroundColors = new Color[9]
	{
		Colors.White,
		Colors.White,
		Colors.Orange,
		Colors.OrangeRed,
		Colors.LightYellow,
		Colors.White,
		Colors.Black,
		Colors.Black,
		Colors.White
	};

	protected static Color[] s_arrForeGroundColors = new Color[9]
	{
		Colors.Gray,
		Colors.Black,
		Colors.Black,
		Colors.Black,
		Colors.Black,
		Colors.Green,
		Colors.White,
		Colors.Green,
		Colors.Purple
	};

	protected static Color[] s_stateColors = new Color[8]
	{
		Colors.White,
		Colors.White,
		Color.FromBytes(252, 228, 214),
		Color.FromBytes(155, 194, 230),
		Color.FromBytes(237, 125, 49),
		Color.FromBytes(0, 176, 80),
		Color.FromBytes(208, 228, 206),
		Color.FromBytes(byte.MaxValue, 0, 0)
	};

	protected IDataField<ICULanLogLine> m_dfValueField;

	protected LogField m_eLogField;

	protected TextLayout m_taLayout;

	public CustomLogCell(IDataField<ICULanLogLine> valueField, LogField logField)
	{
		m_dfValueField = valueField;
		m_eLogField = logField;
		m_taLayout = new TextLayout();
		m_taLayout.Font = Font.SystemMonospaceFont.WithSize(9.0);
	}

	protected override Size OnGetRequiredSize(SizeConstraint sizeConstraint)
	{
		if (m_eLogField == LogField.State1 || m_eLogField == LogField.State2)
		{
			return new Size(100.0, 16.0);
		}
		return new Size(400.0, 16.0);
	}

	protected override void OnDraw(Context ctx, Rectangle cellArea)
	{
		if (!Visible)
		{
			return;
		}
		ICULanLogLine value = GetValue(m_dfValueField);
		if (value != null)
		{
			int num = Math.Min((int)value.Type, s_arrBackGroundColors.Length);
			m_taLayout.ClearAttributes();
			string text = "";
			Color color = s_arrBackGroundColors[num];
			if (m_eLogField == LogField.Information || m_eLogField == LogField.InformationGrayed)
			{
				text = value.Text;
			}
			else if (m_eLogField == LogField.Socket1)
			{
				text = value.Socket1;
			}
			else if (m_eLogField == LogField.Socket2)
			{
				text = value.Socket2;
			}
			else if (m_eLogField == LogField.State1)
			{
				text = ICULanLogLine.GetStateName(value.State1);
				color = s_stateColors[(int)value.State1];
			}
			else if (m_eLogField == LogField.State2)
			{
				text = ICULanLogLine.GetStateName(value.State2);
				color = s_stateColors[(int)value.State2];
			}
			m_taLayout.Text = text;
			double num2 = 4.0;
			if (!Selected && color != Colors.White)
			{
				ctx.Rectangle(Bounds);
				ctx.SetColor(color);
				ctx.FillPreserve();
			}
			if (value.IsSocketMsg && m_eLogField == LogField.InformationGrayed)
			{
				m_taLayout.SetForeground(Colors.LightGray, 0, m_taLayout.Text.Length);
			}
			else
			{
				m_taLayout.SetForeground(s_arrForeGroundColors[num], 0, m_taLayout.Text.Length);
			}
			ctx.DrawTextLayout(m_taLayout, cellArea.Left + 4.0, cellArea.Top + num2);
		}
	}
}
