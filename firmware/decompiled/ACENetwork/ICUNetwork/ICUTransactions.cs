using System;
using System.Collections.Generic;
using System.Collections.ObjectModel;
using System.IO;
using System.Linq;
using System.Net;
using System.Text.RegularExpressions;
using Serilog;

namespace ICUNetwork;

public class ICUTransactions
{
	private readonly ILogger Logger = Log.ForContext<ICUTransactions>();

	private ICULanDevice m_device;

	private const int s_nRequestLongTimeOut = 10000;

	public ObservableCollection<ICUTransactionItem> Transactions { get; private set; } = new ObservableCollection<ICUTransactionItem>();

	public ICUTransactions(ICULanDevice device)
	{
		m_device = device;
		Transactions.Clear();
	}

	~ICUTransactions()
	{
		m_device = null;
		Transactions.Clear();
	}

	public bool Clear()
	{
		if (m_device == null)
		{
			return false;
		}
		bool result = m_device.EraseTransactionDatabase(10000);
		Transactions.Clear();
		return result;
	}

	private string[] unwrapTransActions(string rawTransactions)
	{
		if (string.IsNullOrEmpty(rawTransactions))
		{
			return new string[0];
		}
		rawTransactions = Regex.Replace(rawTransactions, "{\"version\":\\d+,", "");
		rawTransactions = Regex.Replace(rawTransactions, "AP;.*?; ", "");
		rawTransactions = Regex.Replace(rawTransactions, "\\(\\s*null\\s*\\) ", "");
		rawTransactions = rawTransactions.Replace("}", "");
		return rawTransactions.Split(new char[1] { '\n' });
	}

	public bool Read()
	{
		if (m_device == null)
		{
			return false;
		}
		Transactions = new ObservableCollection<ICUTransactionItem>();
		bool flag = false;
		uint num = uint.MaxValue;
		int num2;
		do
		{
			num2 = 0;
			try
			{
				(EWebRequestState, HttpStatusCode, string) tuple = m_device.ExecuteWebRequest("transactions", $"offset={num}", null, 10000, 1);
				if (tuple.Item1 != EWebRequestState.VALID_RESPONSE)
				{
					Logger.Debug("Transaction API web request failed {RequestOffset}", num);
					flag = true;
					break;
				}
				string[] array = unwrapTransActions(tuple.Item3);
				if (array.Count() == 0 || array[0].ToLower().Contains("empty transaction database"))
				{
					break;
				}
				string[] array2 = array;
				foreach (string text in array2)
				{
					if (!string.IsNullOrEmpty(text.Trim()))
					{
						ICUTransactionItem newItem = new ICUTransactionItem(text);
						if (Transactions.Count((ICUTransactionItem tx) => tx.Offset == newItem.Offset) <= 0)
						{
							num2++;
							Transactions.Add(newItem);
							num = (uint)newItem.Offset;
						}
					}
				}
				continue;
			}
			catch (Exception ex)
			{
				Logger.Debug(ex, ex.Message);
				flag = true;
			}
			break;
		}
		while (num2 > 0);
		return !flag;
	}

	public bool SaveToCSV(string fileName)
	{
		if (m_device == null)
		{
			return false;
		}
		List<string> list = new List<string>
		{
			"# Device, " + m_device.Identification,
			$"# Generated, {DateTime.Now}"
		};
		foreach (ICUTransactionItem transaction in Transactions)
		{
			list.Add(transaction.FullText);
		}
		File.WriteAllLines(fileName, list.ToArray());
		return true;
	}
}
