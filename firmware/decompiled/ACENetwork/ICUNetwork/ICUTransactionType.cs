namespace ICUNetwork;

public enum ICUTransactionType
{
	Unknown,
	Transaction,
	Reservation,
	MeterValue,
	StatusNotification,
	StartTransaction,
	StopTransaction,
	DateTimeOffset,
	ReservationStatus,
	SecurityEvent
}
