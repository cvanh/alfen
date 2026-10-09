using System;

namespace ICUNetwork;

[Flags]
public enum TariffDisplayOptionsType
{
	NONE = 0,
	disclaimer = 1,
	perKwh = 2,
	perMinute = 4,
	perSession = 8,
	perOther = 0x10,
	adhocOnlyDisclaimer = 0x20
}
