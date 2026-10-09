namespace ICUNetwork;

public enum EChargingState
{
	Empty,
	Idle,
	ChargingInitializing,
	ChargingProbing,
	ChargingIncreaseCurrent,
	Charging,
	Alternating,
	Unconnected
}
