using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class ColorCell : CanvasCellView
{
	public IDataField<Color> ValueField { get; set; }

	public ColorCell(IDataField<Color> valueField)
	{
		ValueField = valueField;
	}

	protected override Size OnGetRequiredSize(SizeConstraint widthConstraint)
	{
		return new Size(32.0, 16.0);
	}

	protected override void OnDraw(Context ctx, Rectangle cellArea)
	{
		if (Visible)
		{
			ctx.Rectangle(BackgroundBounds);
			Color value = GetValue(ValueField);
			ctx.SetColor(value);
			ctx.Fill();
		}
	}
}
