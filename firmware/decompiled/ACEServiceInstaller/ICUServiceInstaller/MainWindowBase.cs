using System;
using System.Collections.Generic;
using System.Threading;
using Serilog;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class MainWindowBase : Window
{
	private readonly ILogger Logger = Log.ForContext<MainWindowBase>();

	protected static int s_nInitialWidth = 768;

	protected static int s_nInitialHeight = 550;

	protected const int DEFAULT_LOCK_TIMEOUT = 5000;

	protected const int INFINITE_LOCK_TIMEOUT = -1;

	protected const int MAX_TAB_BUTTONS = 12;

	public static int s_nButtonHeight = AppProperties.ButtonHeight;

	public static int s_nButtonWidth = AppProperties.ButtonWidth;

	protected ScrollView m_tabFrame;

	protected PanelNoDevice m_panelNoDevice;

	protected PanelNotLoggedIn m_panelNotLoggedIn;

	protected PanelNoDeviceSelected m_panelNoDeviceSelected;

	protected HBox m_tabButtons;

	protected int m_nSelectedTab;

	protected FrameBox m_tabContentFrame = new FrameBox();

	protected static List<PanelBase> ActivePanels { get; private set; } = new List<PanelBase>();

	protected static List<ToggleButton> TabButtons { get; private set; } = new List<ToggleButton>();

	public void SetTabFrameBorder(bool visible, int padding = 4)
	{
		m_tabContentFrame.BorderWidth = (visible ? 1 : 0);
		m_tabContentFrame.Padding = padding;
	}

	protected VBox CreateTabControl()
	{
		VBox vBox = new VBox();
		vBox.BackgroundColor = Colors.White;
		m_tabContentFrame.BorderColor = AppProperties.Color_Border;
		m_tabContentFrame.Padding = 4.0;
		m_tabContentFrame.BorderWidth = 1.0;
		m_tabContentFrame.BackgroundColor = Colors.White;
		m_tabContentFrame.MarginLeft = 8.0;
		m_tabContentFrame.MarginBottom = 4.0;
		m_tabContentFrame.MinWidth = s_nInitialWidth;
		m_tabContentFrame.MinHeight = s_nInitialHeight;
		m_tabFrame = new ScrollView();
		m_tabFrame.VerticalScrollPolicy = ScrollPolicy.Automatic;
		m_tabFrame.HorizontalScrollPolicy = ScrollPolicy.Never;
		m_tabFrame.BorderVisible = false;
		m_tabContentFrame.Content = m_tabFrame;
		m_panelNoDevice = new PanelNoDevice((MainWindow)this);
		m_panelNoDevice.MinWidth = 200.0;
		m_panelNoDevice.MinHeight = 200.0;
		m_panelNotLoggedIn = new PanelNotLoggedIn((MainWindow)this);
		m_panelNotLoggedIn.MinWidth = 200.0;
		m_panelNotLoggedIn.MinHeight = 200.0;
		m_panelNoDeviceSelected = new PanelNoDeviceSelected((MainWindow)this);
		m_panelNoDeviceSelected.MinWidth = 200.0;
		m_panelNoDeviceSelected.MinHeight = 200.0;
		m_tabButtons = new HBox();
		m_tabButtons.HorizontalPlacement = WidgetPlacement.Start;
		m_tabButtons.MarginLeft = m_tabContentFrame.MarginLeft;
		for (int i = 0; i < 12; i++)
		{
			ToggleButton toggleButton = new ToggleButton
			{
				Style = ButtonStyle.Normal,
				BackgroundColor = Colors.Transparent,
				MinWidth = 64.0,
				MinHeight = 64.0,
				Tag = i
			};
			toggleButton.Clicked += onTabButtonClicked;
			TabButtons.Add(toggleButton);
			m_tabButtons.PackStart(toggleButton, expand: true);
		}
		ReCreateTabControl();
		vBox.PackStart(m_tabButtons);
		vBox.PackStart(m_tabContentFrame, expand: true);
		return vBox;
	}

	protected void SelectTab(int iNewIndex)
	{
		if (Monitor.TryEnter(TabButtons, 5000))
		{
			try
			{
				foreach (ToggleButton tabButton in TabButtons)
				{
					tabButton.Active = (int)tabButton.Tag == iNewIndex;
				}
			}
			catch (Exception exception)
			{
				Logger.Error(exception, "");
			}
			finally
			{
				Monitor.Exit(TabButtons);
			}
		}
		if (!Monitor.TryEnter(ActivePanels, 5000))
		{
			return;
		}
		try
		{
			switch (iNewIndex)
			{
			case -3:
				m_panelNoDevice.Visible = false;
				m_panelNotLoggedIn.Visible = false;
				m_panelNoDeviceSelected.Visible = true;
				m_tabButtons.Sensitive = false;
				ActivePanels.ForEach((PanelBase a) =>
				{
					a.IsPanelVisible = false;
				});
				m_tabFrame.Content = m_panelNoDeviceSelected;
				return;
			case -2:
				m_panelNoDevice.Visible = false;
				m_panelNotLoggedIn.Visible = true;
				m_panelNoDeviceSelected.Visible = false;
				m_tabButtons.Sensitive = false;
				ActivePanels.ForEach((PanelBase a) =>
				{
					a.IsPanelVisible = false;
				});
				m_tabFrame.Content = m_panelNotLoggedIn;
				return;
			case -1:
				m_panelNoDevice.Visible = true;
				m_panelNotLoggedIn.Visible = false;
				m_panelNoDeviceSelected.Visible = false;
				m_tabButtons.Sensitive = false;
				ActivePanels.ForEach((PanelBase a) =>
				{
					a.IsPanelVisible = false;
				});
				m_tabFrame.Content = m_panelNoDevice;
				return;
			}
			m_panelNoDevice.Visible = false;
			m_panelNotLoggedIn.Visible = false;
			m_panelNoDeviceSelected.Visible = false;
			m_tabButtons.Sensitive = true;
			if (m_tabFrame == null || iNewIndex >= ActivePanels.Count)
			{
				return;
			}
			m_nSelectedTab = iNewIndex;
			if (!ActivePanels[iNewIndex].IsPanelVisible)
			{
				ActivePanels.ForEach((PanelBase a) =>
				{
					a.IsPanelVisible = false;
				});
				m_tabFrame.Content = ActivePanels[iNewIndex];
				ActivePanels[iNewIndex].IsPanelVisible = true;
			}
			bool showFrameBorder = ActivePanels[iNewIndex].ShowFrameBorder;
			SetTabFrameBorder(showFrameBorder, showFrameBorder ? 4 : 0);
		}
		catch (Exception ex)
		{
			Logger.Error(ex, ex.Message);
		}
		finally
		{
			Monitor.Exit(ActivePanels);
		}
	}

	private void onTabButtonClicked(object sender, EventArgs e)
	{
		if (sender is Button button)
		{
			SelectTab((int)button.Tag);
		}
	}

	public static Button AddImageButton(string imageName, string toolTipText, EventHandler handler)
	{
		Button button = new Button(Image.FromResource(typeof(App), AppProperties.ResourcePath(imageName)));
		button.TooltipText = toolTipText;
		button.Style = ButtonStyle.Normal;
		button.BackgroundColor = Colors.Transparent;
		button.MinHeight = 32.0;
		button.MinWidth = 32.0;
		button.Clicked += handler;
		return button;
	}

	protected ToggleButton AddImageToggleButton(string imageName, string toolTipText, EventHandler handler)
	{
		ToggleButton toggleButton = new ToggleButton(Image.FromResource(typeof(App), AppProperties.ResourcePath(imageName)));
		toggleButton.TooltipText = toolTipText;
		toggleButton.Style = ButtonStyle.Normal;
		toggleButton.BackgroundColor = Colors.Transparent;
		toggleButton.MinHeight = 32.0;
		toggleButton.Clicked += handler;
		return toggleButton;
	}

	protected MenuItem AddMenuItemHelper(MenuItem parentMenu, string title, EventHandler handler)
	{
		MenuItem menuItem = new MenuItem(title);
		menuItem.Clicked += handler;
		parentMenu.SubMenu.Items.Add(menuItem);
		return menuItem;
	}

	protected void ReCreateTabControl(bool startTimer = true)
	{
		for (int i = 0; i < 12; i++)
		{
			if (i < ActivePanels.Count)
			{
				TabButtons[i].Image = ActivePanels[i].IconImage;
				TabButtons[i].TooltipText = ActivePanels[i].Tooltip;
				TabButtons[i].Sensitive = true;
				TabButtons[i].Opacity = 1.0;
			}
			else
			{
				TabButtons[i].Image = null;
				TabButtons[i].Sensitive = false;
				TabButtons[i].Opacity = 0.0;
			}
		}
	}
}
