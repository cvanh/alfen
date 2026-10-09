using System;
using Serilog;

namespace ICUNetwork;

public class ICUWhitelistItem
{
	private readonly ILogger Logger = Log.ForContext<ICUWhitelistItem>();

	public string Tag { get; set; }

	public string ParentTag { get; set; }

	public ICUTagStatus Status { get; set; }

	public bool HasExpiryDate { get; set; }

	public DateTime ExpiryDate { get; set; }

	public string Expiry
	{
		get
		{
			if (HasExpiryDate)
			{
				return ExpiryDate.ToString();
			}
			return "<no expiry date>";
		}
	}

	public string ExpireDate
	{
		get
		{
			if (HasExpiryDate)
			{
				return ExpiryDate.ToShortDateString();
			}
			return "<no expiry date>";
		}
	}

	public ICUWhitelistItem()
	{
		Tag = "";
		ParentTag = "";
		Status = ICUTagStatus.Unknown;
	}

	public ICUWhitelistItem(dynamic item)
	{
		Tag = "";
		ParentTag = "";
		Status = ICUTagStatus.Unknown;
		try
		{
			Tag = Convert.ToString(item["tag"]);
			ParentTag = Convert.ToString(item["parent"]);
			int num = Convert.ToInt32(item["status"]);
			if (num >= 0 && num <= 3)
			{
				Status = (ICUTagStatus)num;
			}
			string s = Convert.ToString(item["expiryDate"]);
			HasExpiryDate = false;
			if (long.TryParse(s, out var result) && result != 0L)
			{
				DateTime unixEpoch = ICUDevice.UnixEpoch;
				ExpiryDate = unixEpoch.AddSeconds(result).ToLocalTime();
				HasExpiryDate = true;
			}
		}
		catch (Exception ex)
		{
			Logger.Error(ex, ex.Message);
		}
	}

	public override string ToString()
	{
		return Tag;
	}
}
