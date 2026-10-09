using System.Collections.Generic;
using System.Drawing;
using SimplePaletteQuantizer.Helpers;

namespace SimplePaletteQuantizer.Quantizers.Uniform;

public class UniformQuantizer : BaseColorQuantizer
{
	private UniformColorSlot[] redSlots;

	private UniformColorSlot[] greenSlots;

	private UniformColorSlot[] blueSlots;

	public override bool AllowParallel => true;

	protected override void OnPrepare(ImageBuffer image)
	{
		redSlots = new UniformColorSlot[8];
		greenSlots = new UniformColorSlot[8];
		blueSlots = new UniformColorSlot[4];
	}

	protected override void OnAddColor(Color color, int key, int x, int y)
	{
		base.OnAddColor(color, key, x, y);
		int num = color.R >> 5;
		int num2 = color.G >> 5;
		int num3 = color.B >> 6;
		redSlots[num].AddValue(color.R);
		greenSlots[num2].AddValue(color.G);
		blueSlots[num3].AddValue(color.B);
	}

	protected override List<Color> OnGetPalette(int colorCount)
	{
		List<Color> list = base.OnGetPalette(colorCount);
		if (list != null)
		{
			return list;
		}
		List<Color> list2 = new List<Color>();
		UniformColorSlot[] array = redSlots;
		for (int i = 0; i < array.Length; i++)
		{
			UniformColorSlot uniformColorSlot = array[i];
			UniformColorSlot[] array2 = greenSlots;
			for (int j = 0; j < array2.Length; j++)
			{
				UniformColorSlot uniformColorSlot2 = array2[j];
				UniformColorSlot[] array3 = blueSlots;
				foreach (UniformColorSlot uniformColorSlot3 in array3)
				{
					int average = uniformColorSlot.GetAverage();
					int average2 = uniformColorSlot2.GetAverage();
					int average3 = uniformColorSlot3.GetAverage();
					Color item = Color.FromArgb(255, average, average2, average3);
					list2.Add(item);
				}
			}
		}
		return list2;
	}

	protected override void OnGetPaletteIndex(Color color, int key, int x, int y, out int paletteIndex)
	{
		int num = color.R >> 5;
		int num2 = color.G >> 5;
		int num3 = color.B >> 6;
		paletteIndex = (num << 5) + (num2 << 2) + num3;
	}
}
