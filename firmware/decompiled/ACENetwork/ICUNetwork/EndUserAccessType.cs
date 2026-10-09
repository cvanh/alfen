using System.ComponentModel;

namespace ICUNetwork;

public enum EndUserAccessType
{
	NotAvailable,
	[Description("Disabled")]
	Disabled,
	[Description("Enabled (without PIN)")]
	Enabled,
	[Description("Enabled (with PIN)")]
	Configured
}
