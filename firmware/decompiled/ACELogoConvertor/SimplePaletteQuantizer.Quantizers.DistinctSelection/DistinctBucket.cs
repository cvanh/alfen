using System.Collections.Generic;
using System.Drawing;
using System.Linq;

namespace SimplePaletteQuantizer.Quantizers.DistinctSelection;

public class DistinctBucket
{
	public DistinctColorInfo ColorInfo { get; private set; }

	public DistinctBucket[] Buckets { get; private set; }

	public DistinctBucket()
	{
		Buckets = new DistinctBucket[16];
	}

	public void StoreColor(Color color)
	{
		int num = color.R >> 5;
		DistinctBucket distinctBucket = Buckets[num];
		if (distinctBucket == null)
		{
			distinctBucket = new DistinctBucket();
			Buckets[num] = distinctBucket;
		}
		int num2 = color.G >> 5;
		DistinctBucket distinctBucket2 = distinctBucket.Buckets[num2];
		if (distinctBucket2 == null)
		{
			distinctBucket2 = new DistinctBucket();
			distinctBucket.Buckets[num2] = distinctBucket2;
		}
		int num3 = color.B >> 5;
		DistinctBucket distinctBucket3 = distinctBucket2.Buckets[num3];
		if (distinctBucket3 == null)
		{
			distinctBucket3 = new DistinctBucket();
			distinctBucket2.Buckets[num3] = distinctBucket3;
		}
		DistinctColorInfo colorInfo = distinctBucket3.ColorInfo;
		if (colorInfo == null)
		{
			colorInfo = new DistinctColorInfo(color);
			distinctBucket3.ColorInfo = colorInfo;
		}
		else
		{
			colorInfo.IncreaseCount();
		}
	}

	public List<DistinctColorInfo> GetValues()
	{
		return (from redBucket in Buckets
			where redBucket != null
			from greenBucket in from green in redBucket.Buckets
				where green != null
				select green
			select greenBucket into greenBucket
			from blueBucket in from blue in greenBucket.Buckets
				where blue != null
				select blue
			select blueBucket.ColorInfo).ToList();
	}
}
