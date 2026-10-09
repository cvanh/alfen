using System;
using Xwt;

namespace ICUServiceInstaller;

public class PanelNotLoggedIn : PanelBase
{
	protected Button m_btnLogin;

	public PanelNotLoggedIn(MainWindow parent, string pageID = "", bool showBorder = true)
		: base(parent, pageID, fIndent: false, showBorder)
	{
		IconName = "information.png";
		AddLabel("You are logged out.\nPlease retry to login.");
		m_btnLogin = AddCustomButton("Login...", "Login to the charging station", OnLoginClicked, true);
		m_btnLogin.Sensitive = true;
	}

	private void OnLoginClicked(object sender, EventArgs e)
	{
		m_parent?.ReselectCurrentItem(showLoginDialog: true);
	}
}
