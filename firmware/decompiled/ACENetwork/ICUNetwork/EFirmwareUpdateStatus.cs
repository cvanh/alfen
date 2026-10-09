using System.ComponentModel;

namespace ICUNetwork;

public enum EFirmwareUpdateStatus
{
	NO_ACTIVE_UPDATE = 0,
	ERASING_BUFFER = 1,
	BUFFER_ERASED = 2,
	READY_FOR_DOWNLOAD = 3,
	DOWNLOADING_FIRMWARE = 4,
	DOWNLOAD_DONE = 5,
	DOWNLOAD_CHECKED = 6,
	READY_FOR_UPDATE = 7,
	UPDATE_IN_PROGRESS = 8,
	UPDATE_READY_TO_ROLL = 9,
	UPDATE_DONE = 10,
	CRC_CALCULATING = 11,
	CRC_CALCULATED = 12,
	[Description("Error during download")]
	ERROR_DURING_DOWNLOAD = -1,
	[Description("Error during update")]
	ERROR_DURING_UPDATE = -2,
	[Description("Executing rollback")]
	EXECUTING_ROLLBACK = -3,
	[Description("Rolled back")]
	ROLLED_BACK = -4
}
