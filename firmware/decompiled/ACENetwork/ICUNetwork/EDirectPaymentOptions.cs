using System;

namespace ICUNetwork;

[Flags]
public enum EDirectPaymentOptions
{
	None = 0,
	OTS = 1,
	QRCode = 2,
	GiroE = 4
}
