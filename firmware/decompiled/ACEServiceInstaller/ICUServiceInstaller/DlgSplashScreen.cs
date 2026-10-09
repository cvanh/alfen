using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class DlgSplashScreen : Dialog
{
	public DlgSplashScreen()
	{
		Title = AppProperties.AppName;
		Icon = Image.FromResource(typeof(App), AppProperties.AppIconName);
		ShowInTaskbar = true;
		Resizable = false;
		Decorated = false;
		HBox hBox = new HBox();
		ImageView widget = new ImageView(Image.FromResource(typeof(App), AppProperties.ResourcePath("SplashScreen.png")));
		hBox.PackStart(widget);
		Content = hBox;
	}
}
