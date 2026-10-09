using System.Diagnostics;

namespace ICUIWSConnection;

[DebuggerDisplay("{DebuggerDisplay,nq}")]
public class IWSPropertyValue
{
	public string DebuggerDisplay => $"{Id:X4}_{SubId:X2}: {Value}";

	public ushort Id { get; set; }

	public byte SubId { get; set; }

	public object Value { get; set; }
}
