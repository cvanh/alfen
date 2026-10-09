using System.Net.Http;
using System.Text.RegularExpressions;
using System.Threading.Tasks;
using ICUIWSConnection;
using ICUNetwork;
using ICUSettings;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class DlgObjectID : Dialog
{
	public readonly string m_regexObjectNrFormat = "^[0-9]{5,6}r[0-9]{2,3}$";

	public readonly string m_regexNewObjectNrFormat = "^ACE[0-9]{7}$";

	protected TextEntry m_txtObjectID = new TextEntry();

	private readonly ICULanDevice lanDevice;

	private readonly ICUUser currentUser;

	public string NewObjectID { get; set; }

	public DlgObjectID(ICULanDevice lanDev, ICUUser currentUser)
	{
		Title = "Device Object ID";
		Icon = Image.FromResource(typeof(App), AppProperties.AppIconName);
		ShowInTaskbar = false;
		Resizable = false;
		lanDevice = lanDev;
		this.currentUser = currentUser;
		Table table = new Table
		{
			BackgroundColor = Colors.White,
			MinWidth = 300.0,
			Margin = 8.0
		};
		FrameBox content = new FrameBox
		{
			BorderColor = AppProperties.Color_Border,
			Padding = 16.0,
			BorderWidth = 1.0,
			BackgroundColor = Colors.White,
			Content = table
		};
		table.Add(new Label("Empty or invalid Object ID found, please enter a valid one."), 0, 0, 1, 2);
		table.Add(new Label("Object ID:"), 0, 1);
		table.Add(m_txtObjectID, 1, 1, 1, 1, hexpand: true);
		Content = content;
		CloseRequested += (object sender, CloseRequestedEventArgs args) =>
		{
			args.AllowClose = false;
			Show();
		};
		Buttons.Add(new DialogButton(Command.Ok));
	}

	protected override void OnCommandActivated(Command cmd)
	{
		if (cmd == Command.Ok)
		{
			Regex regex = new Regex(m_regexObjectNrFormat, RegexOptions.IgnoreCase);
			Regex regex2 = new Regex(m_regexNewObjectNrFormat, RegexOptions.IgnoreCase);
			string text = m_txtObjectID.Text.ToLower().Trim();
			Application.Invoke(() =>
			{
				Sensitive = false;
			});
			if (!regex.Match(text).Success && !regex2.Match(text).Success)
			{
				m_txtObjectID.SetFocus();
				MessageDialog.ShowError(this, "Invalid Object ID: " + text + "!\nPlease provide a correct one.");
				Application.Invoke(() =>
				{
					Sensitive = true;
				});
				return;
			}
			if (!CheckIfObjectIDExists(text))
			{
				m_txtObjectID.SetFocus();
				Application.Invoke(() =>
				{
					Sensitive = true;
				});
				return;
			}
			NewObjectID = m_txtObjectID.Text;
			Respond(Command.Ok);
			Close();
		}
		base.OnCommandActivated(cmd);
	}

	private bool CheckIfObjectIDExists(string objectId)
	{
		if (currentUser == null || !Task.Run(async () => await IWSConnection.TestIWSConnection(AppProperties.IsahSite)).Result)
		{
			return MessageDialog.AskQuestion("Unable to verify the entered Object ID with ISAH. Are you sure to change the Object ID to " + objectId + "?", 0, Command.Yes, Command.No, Command.Cancel) == Command.Yes;
		}
		lanDevice.UpdateCategories("generic");
		ulong propertyUInt = lanDevice.GetPropertyUInt64(8608, 0, 0uL);
		IWSObject iwsobj = null;
		HttpResponseMessage objectData = IWSConnection.GetObjectData(ref iwsobj, AppProperties.IsahSite, currentUser.Company, objectId, lanDevice.NumberOfSockets, propertyUInt.ToString());
		if (iwsobj != null && objectData.IsSuccessStatusCode)
		{
			MessageDialog.ShowMessage("The Object ID of the Charge Station will be changed to " + objectId + ".");
			return true;
		}
		return MessageDialog.AskQuestion("The entered Object ID seems to be invalid. Are you sure to change the Object ID to " + objectId + "?", 0, Command.Yes, Command.No, Command.Cancel) == Command.Yes;
	}
}
