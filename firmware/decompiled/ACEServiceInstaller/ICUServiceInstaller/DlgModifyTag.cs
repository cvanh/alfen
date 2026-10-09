using System;
using ICUNetwork;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class DlgModifyTag : Dialog
{
	protected TextEntry m_txtTag = new TextEntry();

	protected TextEntry m_txtParent = new TextEntry();

	protected ComboBox m_cmbStatus = new ComboBox();

	protected CheckBox m_chkHasExpire = new CheckBox();

	protected Label m_lblExpiry = new Label("Expiry date:");

	protected DatePicker m_datExpiry = new DatePicker(DateTime.Now.AddYears(1));

	public ICUWhitelistItem WhitelistItem { get; set; }

	public DlgModifyTag(ICUWhitelistItem whiteListItem = null, bool masterTagEnabled = false)
	{
		WhitelistItem = whiteListItem;
		Title = ((whiteListItem == null) ? "Add a new tag" : "Modify a tag");
		Icon = Image.FromResource(typeof(App), AppProperties.AppIconName);
		ShowInTaskbar = false;
		Resizable = true;
		Table table = new Table
		{
			BackgroundColor = Colors.White,
			MinWidth = 300.0
		};
		FrameBox widget = new FrameBox
		{
			BorderColor = AppProperties.Color_Border,
			Padding = 8.0,
			BorderWidth = 1.0,
			BackgroundColor = Colors.White,
			Content = table
		};
		table.Margin = 8.0;
		m_cmbStatus.Items.Add(ICUTagStatus.Active, "Active");
		m_cmbStatus.Items.Add(ICUTagStatus.Blocked, "Blocked");
		m_cmbStatus.Items.Add(ICUTagStatus.Deleted, "Deleted");
		if (masterTagEnabled)
		{
			m_cmbStatus.Items.Add(ICUTagStatus.MasterCard, "Master Key");
		}
		m_cmbStatus.SelectedItem = ICUTagStatus.Active;
		m_cmbStatus.SelectionChanged += StatusCmbBoxSelectionChanged;
		if (whiteListItem == null)
		{
			Label label = new Label("Add a new tag to the whitelist");
			label.Font = label.Font.WithWeight(FontWeight.Semibold);
			table.Add(label, 0, 0, 1, 2);
		}
		table.Add(new Label("Tag ID:"), 0, 1);
		table.Add(m_txtTag, 1, 1, 1, 1, hexpand: true);
		m_txtTag.Sensitive = whiteListItem == null;
		table.Add(new Label("Parent ID:"), 0, 2);
		table.Add(m_txtParent, 1, 2, 1, 1, hexpand: true);
		table.Add(new Label("Status:"), 0, 3);
		table.Add(m_cmbStatus, 1, 3, 1, 1, hexpand: true);
		table.Add(new Label("Has expiry date?"), 0, 4);
		table.Add(m_chkHasExpire, 1, 4, 1, 1, hexpand: true);
		m_chkHasExpire.Toggled += OnHasExpireChanged;
		table.Add(m_lblExpiry, 0, 5, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, 20.0);
		table.Add(m_datExpiry, 1, 5, 1, 1, hexpand: true);
		VBox vBox = new VBox();
		vBox.PackStart(widget, expand: true);
		Content = vBox;
		m_txtTag.SetFocus();
		if (WhitelistItem != null)
		{
			if (WhitelistItem.Status != ICUTagStatus.Unknown)
			{
				m_cmbStatus.SelectedItem = WhitelistItem.Status;
			}
			m_txtTag.Text = WhitelistItem.Tag;
			m_txtParent.Text = WhitelistItem.ParentTag;
			if (WhitelistItem.HasExpiryDate)
			{
				m_chkHasExpire.State = CheckBoxState.On;
				m_datExpiry.DateTime = WhitelistItem.ExpiryDate;
			}
			else
			{
				m_chkHasExpire.State = CheckBoxState.Off;
			}
		}
		Buttons.Add(new DialogButton(Command.Cancel));
		Buttons.Add(new DialogButton(Command.Ok));
		OnHasExpireChanged(null, null);
	}

	private void StatusCmbBoxSelectionChanged(object sender, EventArgs e)
	{
		if ((ICUTagStatus)(sender as ComboBox).SelectedItem == ICUTagStatus.MasterCard)
		{
			m_txtParent.Text = "";
			m_chkHasExpire.Active = false;
			m_chkHasExpire.Sensitive = false;
			m_txtParent.Sensitive = false;
		}
		else
		{
			m_chkHasExpire.Sensitive = true;
			m_txtParent.Sensitive = true;
		}
	}

	private void OnHasExpireChanged(object sender, EventArgs e)
	{
		bool sensitive = m_chkHasExpire.State == CheckBoxState.On;
		m_datExpiry.Sensitive = sensitive;
		m_lblExpiry.Sensitive = sensitive;
	}

	protected override void OnCommandActivated(Command cmd)
	{
		if (cmd == Command.Ok)
		{
			if (WhitelistItem == null)
			{
				WhitelistItem = new ICUWhitelistItem();
			}
			WhitelistItem.Tag = m_txtTag.Text;
			WhitelistItem.ParentTag = m_txtParent.Text;
			WhitelistItem.Status = (ICUTagStatus)m_cmbStatus.SelectedItem;
			WhitelistItem.HasExpiryDate = m_chkHasExpire.State == CheckBoxState.On;
			WhitelistItem.ExpiryDate = m_datExpiry.DateTime;
			Respond(Command.Ok);
			Close();
		}
		else
		{
			Respond(Command.Cancel);
			Close();
		}
	}
}
