namespace ICUNetwork;

public enum ICUTransactionStopReason
{
	Other = 0,
	Local = 1,
	EmergencyStop = 2,
	EVDisconnected = 3,
	Hardreset = 4,
	Powerloss = 5,
	Reboot = 6,
	Remote = 7,
	Softreset = 8,
	UnlockCommand = 9,
	Deauthorized = 10,
	None = 255
}
