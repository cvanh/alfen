using System;
using System.Globalization;
using System.Linq;
using Serilog;

namespace ICUNetwork;

public class ICUTransactionItem
{
	private readonly ILogger Logger = Log.ForContext<ICUTransactionItem>();

	public long Offset { get; set; }

	public string FullText { get; set; }

	public uint Socket { get; set; }

	public ulong TransactionId { get; set; }

	public DateTime? StartTime { get; set; }

	public DateTime? StopTime { get; set; } = DateTime.MinValue;

	public string StartTag { get; set; }

	public string StopTag { get; set; }

	public ICUTransactionStopReason StopReason { get; set; } = ICUTransactionStopReason.None;

	public ICUTransactionTriggerReason TriggerReason { get; set; } = ICUTransactionTriggerReason.None;

	public ICUTransactionChargingState ChargingState { get; set; }

	public double? StartMeterValue { get; set; }

	public double? StopMeterValue { get; set; }

	public bool StartToSend { get; set; }

	public bool StopToSend { get; set; }

	public ICUTransactionType Type { get; set; }

	public string ExtraInfo { get; set; } = string.Empty;

	public ICUTransactionSecurityEvent SecurityEvent { get; set; }

	public ICUTransactionReservationStatus ReservationStatus { get; set; } = ICUTransactionReservationStatus.Unknown;

	public ICUTransactionItem()
	{
	}

	public ICUTransactionItem(string tx)
	{
		try
		{
			string[] array = tx.Split(new char[1] { '_' });
			if (array.Count() > 1)
			{
				if (long.TryParse(array[0], out var result))
				{
					Offset = result;
				}
				FullText = array[1];
				string text = array[1];
				if (text.StartsWith("tx:"))
				{
					Type = ICUTransactionType.Transaction;
					string[] array2 = text.Substring(text.IndexOf(":") + 1).Split(new char[1] { ',' });
					if (array2.Count() < 3)
					{
						return;
					}
					string text2 = array2[0].Trim();
					if (text2.StartsWith("id"))
					{
						TransactionId = ulong.Parse(text2.Substring(5), NumberStyles.HexNumber);
					}
					string text3 = array2[1].Trim();
					if (text3.StartsWith("socket"))
					{
						Socket = uint.Parse(text3.Substring(6));
					}
					string text4 = array2[2].Trim();
					string s = text4.Substring(0, 19);
					string[] array3 = text4.Split(new char[1] { ' ' });
					if (array3.Count() == 5)
					{
						StartTime = DateTime.ParseExact(s, "yyyy-MM-dd HH:mm:ss", CultureInfo.InvariantCulture);
						StartMeterValue = double.Parse(array3[2].Trim(new char[4] { ' ', 'k', 'W', 'h' }), CultureInfo.InvariantCulture);
						StartTag = array3[3].Trim();
						StartToSend = array3[4].Trim() == "Y";
					}
					if (array2.Count() >= 4)
					{
						string text5 = array2[3].Trim();
						string s2 = text5.Substring(0, 19);
						string[] array4 = text5.Split(new char[1] { ' ' });
						if (array4.Count() == 5)
						{
							StopTime = DateTime.ParseExact(s2, "yyyy-MM-dd HH:mm:ss", CultureInfo.InvariantCulture);
							StopMeterValue = double.Parse(array4[2].Trim(new char[4] { ' ', 'k', 'W', 'h' }), CultureInfo.InvariantCulture);
							StopTag = array4[3].Trim();
							StopToSend = array4[4].Trim() == "Y";
						}
						else if (array4.Count() == 6)
						{
							StopTime = DateTime.ParseExact(s2, "yyyy-MM-dd HH:mm:ss", CultureInfo.InvariantCulture);
							StopMeterValue = double.Parse(array4[2].Trim(new char[4] { ' ', 'k', 'W', 'h' }), CultureInfo.InvariantCulture);
							StopTag = array4[3].Trim();
							StopReason = (ICUTransactionStopReason)int.Parse(array4[4].Trim(), CultureInfo.InvariantCulture);
							StopToSend = array4[5].Trim() == "Y";
						}
					}
				}
				else if (text.StartsWith("rs:"))
				{
					Type = ICUTransactionType.Reservation;
					string[] array5 = text.Substring(3).Trim().Split(new char[1] { ' ' });
					if (array5.Count() >= 8)
					{
						if (array5[0].StartsWith("#"))
						{
							TransactionId = ulong.Parse(array5[0].Substring(1));
						}
						StartTag = array5[2].Trim();
						Socket = uint.Parse(array5[5].Trim(new char[1] { ',' }));
						StartTime = DateTime.ParseExact(array5[7] + " " + array5[8], "yyyy-MM-dd HH:mm:ss", CultureInfo.InvariantCulture);
					}
				}
				else if (text.StartsWith("sn:") || text.StartsWith("sn3:"))
				{
					Type = ICUTransactionType.StatusNotification;
					string[] array6 = text.Substring(text.IndexOf(":") + 1).Trim().Split(new char[1] { ' ' });
					if (string.Compare(array6[0], ":") == 0)
					{
						array6 = array6.Skip(1).ToArray();
					}
					if (array6.Count() >= 8)
					{
						Socket = uint.Parse(array6[1].Trim(new char[1] { ',' }));
						StartTime = DateTime.ParseExact(array6[2] + " " + array6[3], "yyyy-MM-dd HH:mm:ss", CultureInfo.InvariantCulture);
						string text6 = "";
						for (int i = 6; i < array6.Count() - 3; i++)
						{
							text6 = text6 + array6[i].Trim(new char[1] { ',' }) + " ";
						}
						ExtraInfo = (array6[4].Trim(new char[1] { ',' }) + " " + array6[5].Trim(new char[1] { ',' }) + " " + text6.Trim()).Trim();
						if (array6.Count() >= 10)
						{
							TriggerReason = (ICUTransactionTriggerReason)int.Parse(array6[array6.Count() - 3].Trim(new char[1] { ',' }).Trim(), CultureInfo.InvariantCulture);
							ChargingState = (ICUTransactionChargingState)int.Parse(array6[array6.Count() - 2].Trim(new char[1] { ',' }).Trim(), CultureInfo.InvariantCulture);
						}
					}
				}
				else if (text.StartsWith("mv:"))
				{
					Type = ICUTransactionType.MeterValue;
					string[] array7 = text.Substring(3).Trim().Split(new char[1] { ' ' });
					if (array7.Count() >= 6)
					{
						Socket = uint.Parse(array7[1].Trim(new char[1] { ',' }));
						StartTime = DateTime.ParseExact(array7[2] + " " + array7[3], "yyyy-MM-dd HH:mm:ss", CultureInfo.InvariantCulture);
						StartMeterValue = double.Parse(array7[4].Trim(new char[4] { ' ', 'k', 'W', 'h' }), CultureInfo.InvariantCulture);
						StartToSend = array7[array7.Count() - 1].Trim() == "Y";
						if (array7[array7.Count() - 2].ToLower() == "start" || array7[array7.Count() - 2].ToLower() == "stop" || array7[array7.Count() - 2].ToLower() == "regular")
						{
							ExtraInfo = array7[array7.Count() - 2];
						}
					}
				}
				else if (text.StartsWith("txstart:") || text.StartsWith("txstart2:"))
				{
					Type = ICUTransactionType.StartTransaction;
					string[] array8 = text.Substring(text.IndexOf(":") + 1).Split(new char[1] { ',' });
					if (array8.Count() < 3)
					{
						return;
					}
					string text7 = array8[0].Trim();
					if (text7.StartsWith("id"))
					{
						TransactionId = ulong.Parse(text7.Substring(5), NumberStyles.HexNumber);
					}
					string text8 = array8[1].Trim();
					if (text8.StartsWith("socket"))
					{
						Socket = uint.Parse(text8.Substring(6));
					}
					string text9 = array8[2].Trim();
					string s3 = text9.Substring(0, 19);
					string[] array9 = text9.Split(new char[1] { ' ' });
					int num = array9.Count();
					if (num == 6 || num == 7)
					{
						StartTime = DateTime.ParseExact(s3, "yyyy-MM-dd HH:mm:ss", CultureInfo.InvariantCulture);
						StartMeterValue = double.Parse(array9[2].Trim(new char[4] { ' ', 'k', 'W', 'h' }), CultureInfo.InvariantCulture);
						StartTag = array9[3].Trim();
						if (num == 6)
						{
							StopReason = (ICUTransactionStopReason)int.Parse(array9[4].Trim(), CultureInfo.InvariantCulture);
							StartToSend = array9[5].Trim() == "Y";
						}
						if (num == 7)
						{
							TriggerReason = (ICUTransactionTriggerReason)int.Parse(array9[4].Trim(), CultureInfo.InvariantCulture);
							ChargingState = (ICUTransactionChargingState)int.Parse(array9[5].Trim(), CultureInfo.InvariantCulture);
							StartToSend = array9[6].Trim() == "Y";
						}
					}
				}
				else if (text.StartsWith("txstop:") || text.StartsWith("txstop2:"))
				{
					Type = ICUTransactionType.StopTransaction;
					string[] array10 = text.Substring(text.IndexOf(":") + 1).Split(new char[1] { ',' });
					if (array10.Count() >= 3)
					{
						string text10 = array10[0].Trim();
						if (text10.StartsWith("id"))
						{
							TransactionId = ulong.Parse(text10.Substring(5), NumberStyles.HexNumber);
						}
						string text11 = array10[1].Trim();
						if (text11.StartsWith("socket"))
						{
							Socket = uint.Parse(text11.Substring(6));
						}
						string text12 = array10[2].Trim();
						string s4 = text12.Substring(0, 19);
						string[] array11 = text12.Split(new char[1] { ' ' });
						if (array11.Count() == 5)
						{
							StopTime = DateTime.ParseExact(s4, "yyyy-MM-dd HH:mm:ss", CultureInfo.InvariantCulture);
							StopMeterValue = double.Parse(array11[2].Trim(new char[4] { ' ', 'k', 'W', 'h' }), CultureInfo.InvariantCulture);
							StopTag = ((array11[3].Trim() != "(null)" && array11[3].Trim() != "") ? array11[3].Trim() : null);
							StopToSend = array11[4].Trim() == "Y";
						}
						if (array11.Count() == 7)
						{
							StopTime = DateTime.ParseExact(s4, "yyyy-MM-dd HH:mm:ss", CultureInfo.InvariantCulture);
							StopMeterValue = double.Parse(array11[2].Trim(new char[4] { ' ', 'k', 'W', 'h' }), CultureInfo.InvariantCulture);
							StopTag = ((array11[3].Trim() != "(null)" && array11[3].Trim() != "") ? array11[3].Trim() : null);
							TriggerReason = (ICUTransactionTriggerReason)int.Parse(array11[4].Trim(), CultureInfo.InvariantCulture);
							ChargingState = (ICUTransactionChargingState)int.Parse(array11[5].Trim(), CultureInfo.InvariantCulture);
							StopToSend = array11[6].Trim() == "Y";
						}
					}
				}
				else if (text.StartsWith("dto:"))
				{
					Type = ICUTransactionType.DateTimeOffset;
					if (int.TryParse(text.Replace("dto:", ""), out var result2))
					{
						TimeSpan timeSpan = TimeSpan.FromSeconds((double)result2);
						ExtraInfo = string.Format("Days: {0} Hours: {1} Minutes: {2} Seconds: {3}", new object[4] { timeSpan.Days, timeSpan.Hours, timeSpan.Minutes, timeSpan.Seconds });
					}
				}
				else if (text.StartsWith("se:"))
				{
					Type = ICUTransactionType.SecurityEvent;
					string[] array12 = text.Substring(4).Split(new char[1] { ' ' });
					if (array12.Count() >= 4)
					{
						StartTime = DateTime.ParseExact(array12[0] + " " + array12[1].Trim(new char[2] { ' ', ':' }), "yyyy-MM-dd HH:mm:ss", CultureInfo.InvariantCulture);
						if (int.TryParse(array12[2], out var result3))
						{
							SecurityEvent = (ICUTransactionSecurityEvent)result3;
						}
						StartToSend = array12[3].Trim(new char[3] { ' ', '(', ')' }) == "Y";
					}
				}
				else if (text.StartsWith("rss:"))
				{
					Type = ICUTransactionType.ReservationStatus;
					string[] array13 = text.Substring(5).Split(new char[1] { ' ' });
					if (array13.Count() >= 3)
					{
						if (ulong.TryParse(array13[0].Trim(new char[2] { ' ', ':' }), out var result4))
						{
							TransactionId = result4;
						}
						if (int.TryParse(array13[1], out var result5))
						{
							ReservationStatus = (ICUTransactionReservationStatus)result5;
						}
						StartToSend = array13[2].Trim(new char[3] { ' ', '(', ')' }) == "Y";
					}
				}
				else
				{
					Logger.Debug("Transaction part \"{Transaction}\" is not recognized", text);
				}
			}
			else
			{
				Logger.Debug("Line: \"{Tx}\" cannot be parsed", tx);
			}
		}
		catch (Exception ex)
		{
			Logger.Error(ex, "Error parsing line \"{Tx}\" with message \"{Message}\"", tx, ex.Message);
		}
	}

	public override string ToString()
	{
		return FullText;
	}
}
