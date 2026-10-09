using System.Collections.Generic;
using System.Linq;

namespace ICUServiceInstaller;

public class TooltipCollection : List<Tip>
{
	public ELanguage Language { get; set; }

	public TooltipCollection(ELanguage language)
	{
		Language = language;
	}

	public Tip Get(string tooltipID)
	{
		return this.FirstOrDefault((Tip a) => a.Id == tooltipID);
	}
}
