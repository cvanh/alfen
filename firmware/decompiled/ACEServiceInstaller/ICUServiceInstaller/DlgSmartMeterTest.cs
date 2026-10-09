using System;
using System.Collections.Generic;
using ICUNetwork;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

internal class DlgSmartMeterTest : Dialog
{
	private readonly List<ModbusMeasurandUIList> measurandUILists;

	public DlgSmartMeterTest(ICULanDevice currentDevice, EMeterTypes m_energyMeterType)
	{
		Title = ((m_energyMeterType == EMeterTypes.ENERGYMETER_TCPIP_CENTRAL) ? "Modbus TCP/IP Test" : "Modbus RTU Test");
		Icon = Image.FromResource(typeof(App), AppProperties.AppIconName);
		ShowInTaskbar = false;
		Resizable = false;
		Table table = new Table
		{
			BackgroundColor = Colors.White,
			MinHeight = 60.0,
			MinWidth = 120.0,
			Margin = 8.0
		};
		FrameBox widget = new FrameBox
		{
			BorderColor = AppProperties.Color_Border,
			Padding = 8.0,
			BorderWidth = 1.0,
			BackgroundColor = Colors.White,
			Content = table
		};
		if (currentDevice != null && currentDevice.IsConnected)
		{
			currentDevice.UpdateCategories("meter4");
		}
		measurandUILists = new List<ModbusMeasurandUIList>
		{
			new ModbusMeasurandUIList(10, "Current L1: ", "A", 2, 1),
			new ModbusMeasurandUIList(11, "Current L2: ", "A", 2, 1),
			new ModbusMeasurandUIList(12, "Current L3: ", "A", 2, 1),
			new ModbusMeasurandUIList(19, "Active Power L1: ", "kW", 3, 1000),
			new ModbusMeasurandUIList(20, "Active Power L2: ", "kW", 3, 1000),
			new ModbusMeasurandUIList(21, "Active Power L3: ", "kW", 3, 1000)
		};
		for (int i = 0; i < measurandUILists.Count; i++)
		{
			table.Add(measurandUILists[i].Label, 0, i, 1, 1, hexpand: true);
			double num = Math.Round(currentDevice.GetPropertyDouble(21025, measurandUILists[i].SubId) / (double)measurandUILists[i].Divider, measurandUILists[i].Decimals, MidpointRounding.AwayFromZero);
			table.Add(new Label($"{num} {measurandUILists[i].Unit}"), 1, i, 1, 1, hexpand: true);
		}
		VBox vBox = new VBox();
		vBox.PackStart(widget, expand: true);
		Content = vBox;
		Buttons.Add(new DialogButton(Command.Close));
	}
}
