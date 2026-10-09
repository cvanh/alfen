namespace ICUNetwork;

public enum ICUTransactionSecurityEvent
{
	None,
	FirmwareUpdated,
	AuthenticationFailedAtCSMS,
	CSMSFailedToAuthenticate,
	SetSystemTime,
	StartUpDevice,
	ResetOrReboot,
	SecurityLogCleared,
	ReconfigOfParameter,
	MemoryExhaustion,
	InvalidMessages,
	ReplayAttack,
	TamperDetection,
	FirmwareSignature,
	FirmwareSignCertificate,
	CSMSCertificate,
	CPCertificate,
	TLSVersion,
	TLSCipherSuite
}
