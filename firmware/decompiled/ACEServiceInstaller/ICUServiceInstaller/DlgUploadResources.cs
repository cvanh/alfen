using System;
using System.ComponentModel;
using System.Drawing;
using System.Drawing.Imaging;
using System.IO;
using System.Linq;
using ICUFWUCreator;
using ICULogoConvertor;
using ICUNetwork;
using ICUServiceInstaller.Utils;
using ICUSettings;
using Serilog;
using Xwt;
using Xwt.Drawing;

namespace ICUServiceInstaller;

public class DlgUploadResources : Dialog
{
	private readonly ILogger Logger = Log.ForContext<DlgUploadResources>();

	public static int s_nMaxColors = 128;

	public static int s_nMaxLogoHeight = 160;

	public static int s_nMaxLogoHeightLarge = 350;

	public static int s_nMaxLogoWidth = 320;

	public static int s_nMaxLogoWidthLarge = 800;

	public int m_nMaxLogoHeight = s_nMaxLogoHeight;

	public int m_nMaxLogoWidth = s_nMaxLogoWidth;

	protected static int s_nDefaultMargin = 0;

	protected BackgroundWorker m_bgwUpload = new BackgroundWorker();

	protected Button m_btnBrowse = new Button("...");

	protected Button m_btnCreateFWU = new Button("Create Image Update File...");

	protected Button m_btnUpload = new Button("Start upload");

	protected ICULanDevice m_currentDevice;

	protected ICUUser m_currentUser;

	protected bool m_fDeviceHasDisplay;

	protected Image m_imgConvertedImage;

	protected ImageView m_imgLogo = new ImageView();

	protected Label m_lblDeviceHeader = new Label();

	protected ListView m_lstFilenames = new ListView();

	protected ProgressBar m_prbUpload = new ProgressBar();

	protected SpinButton m_spbMargin = new SpinButton();

	protected Spinner m_spnSpinner = new Spinner();

	protected TextEntry m_txtFilename = new TextEntry
	{
		PlaceholderText = "Enter a valid firmware file"
	};

	protected TextEntry m_txtInformation = new TextEntry();

	public void Initialize()
	{
		Title = "Upload Logo";
		Icon = Image.FromResource(typeof(App), AppProperties.AppIconName);
		ShowInTaskbar = false;
		Resizable = true;
		Table table = new Table
		{
			MinHeight = 300.0,
			MinWidth = 700.0,
			BackgroundColor = Colors.White
		};
		FrameBox content = new FrameBox
		{
			BorderColor = AppProperties.Color_Border,
			Padding = 8.0,
			BorderWidth = 1.0,
			BackgroundColor = Colors.White,
			Content = table
		};
		Content = content;
		table.Margin = 8.0;
		int num = 0;
		m_lblDeviceHeader = new Label();
		m_lblDeviceHeader.Font = m_lblDeviceHeader.Font.WithScaledSize(1.2).WithWeight(FontWeight.Semibold);
		m_lblDeviceHeader.TextColor = Colors.SteelBlue;
		table.Add(m_lblDeviceHeader, 0, num, 1, 2, hexpand: false, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, UIPropertyBase.s_marginHor + s_nDefaultMargin, marginRight: UIPropertyBase.s_marginHorMax, marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
		num++;
		m_txtFilename.Changed += OnFilenameChanged;
		m_btnBrowse.Clicked += OnBtnBrowse;
		m_btnUpload.Clicked += OnBtnUploadClicked;
		m_btnCreateFWU.Clicked += OnBtnCreateFWUClicked;
		m_btnBrowse.MinWidth = 20.0;
		m_prbUpload.Visible = false;
		m_spnSpinner.Visible = false;
		m_spnSpinner.MinHeight = 48.0;
		m_prbUpload.MinHeight = 16.0;
		m_btnUpload.MinWidth = AppProperties.ButtonWidth;
		m_btnUpload.MinHeight = AppProperties.ButtonHeight;
		m_btnUpload.Style = ButtonStyle.Normal;
		m_btnUpload.Sensitive = false;
		m_btnCreateFWU.MinWidth = AppProperties.ButtonWidth + 40;
		m_btnCreateFWU.MinHeight = AppProperties.ButtonHeight;
		m_btnCreateFWU.Style = ButtonStyle.Normal;
		m_btnCreateFWU.Sensitive = false;
		table.Add(new Label("Image file location:"), 0, num, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, UIPropertyBase.s_marginHor + s_nDefaultMargin, marginRight: UIPropertyBase.s_marginHorMax, marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
		HBox hBox = new HBox();
		hBox.PackStart(m_txtFilename, expand: true);
		hBox.PackStart(m_btnBrowse, expand: false);
		table.Add(hBox, 1, num, 1, 1, hexpand: true, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, UIPropertyBase.s_marginHor, UIPropertyBase.s_marginVer, 0.0, UIPropertyBase.s_marginVer);
		num++;
		m_spbMargin.Digits = 0;
		m_spbMargin.IncrementValue = 1.0;
		m_spbMargin.MinimumValue = 0.0;
		m_spbMargin.MaximumValue = Math.Min(m_nMaxLogoWidth / 2, m_nMaxLogoHeight / 2);
		m_spbMargin.Value = 8.0;
		table.Add(new Label("Image margin:"), 0, num, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, UIPropertyBase.s_marginHor + s_nDefaultMargin, marginRight: UIPropertyBase.s_marginHorMax, marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
		table.Add(m_spbMargin, 1, num, 1, 1, hexpand: true, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, UIPropertyBase.s_marginHor, UIPropertyBase.s_marginVer, 0.0, UIPropertyBase.s_marginVer);
		m_spbMargin.ValueChanged += OnMarginChanged;
		num++;
		ScrollView scrollView = new ScrollView
		{
			Content = m_imgLogo
		};
		scrollView.MinHeight = 200.0;
		FrameBox frameBox = new FrameBox
		{
			Content = scrollView
		};
		frameBox.BorderColor = AppProperties.Color_Border;
		frameBox.BorderWidth = 1.0;
		table.Add(new Label("Converted image:"), 0, num, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Start, UIPropertyBase.s_marginHor + s_nDefaultMargin, marginRight: UIPropertyBase.s_marginHorMax, marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
		table.Add(frameBox, 1, num, 1, 1, hexpand: true, vexpand: true, WidgetPlacement.Fill, WidgetPlacement.Fill, UIPropertyBase.s_marginHor, UIPropertyBase.s_marginVer, 0.0, UIPropertyBase.s_marginVer);
		num++;
		m_txtInformation.MinHeight = 64.0;
		m_txtInformation.ReadOnly = true;
		m_txtInformation.MultiLine = true;
		m_txtInformation.Sensitive = false;
		m_txtInformation.TextAlignment = Alignment.Start;
		table.Add(m_txtInformation, 1, num, 1, 1, hexpand: true, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, UIPropertyBase.s_marginHor, UIPropertyBase.s_marginVer, 0.0, UIPropertyBase.s_marginVer);
		num++;
		table.Add(m_prbUpload, 0, num, 1, 2, hexpand: false, vexpand: false, WidgetPlacement.Fill, WidgetPlacement.Fill, UIPropertyBase.s_marginHor + s_nDefaultMargin, -1.0, UIPropertyBase.s_marginHorMax);
		num++;
		table.Add(m_spnSpinner, 0, num, 1, 3, hexpand: false, vexpand: false, WidgetPlacement.Center);
		num++;
		HBox hBox2 = new HBox();
		hBox2.PackStart(m_btnCreateFWU);
		hBox2.PackStart(m_btnUpload);
		table.Add(hBox2, 1, num, 1, 1, hexpand: false, vexpand: false, WidgetPlacement.Start, WidgetPlacement.Fill, UIPropertyBase.s_marginHor, marginRight: UIPropertyBase.s_marginHor, marginTop: UIPropertyBase.s_marginVer, marginBottom: UIPropertyBase.s_marginVer);
		num++;
		m_bgwUpload.WorkerReportsProgress = true;
		m_bgwUpload.DoWork += OnUploadDoWork;
		m_bgwUpload.ProgressChanged += OnUploadProgressChanged;
		m_bgwUpload.RunWorkerCompleted += OnUploadCompleted;
		CloseRequested += (object sender, CloseRequestedEventArgs args) =>
		{
			args.AllowClose = false;
			Hide();
		};
		Buttons.Add(new DialogButton(Command.Close));
		if (File.Exists("logo_alfen.png"))
		{
			m_txtFilename.Text = "logo_alfen.png";
		}
	}

	public void SetObjects(ICULanDevice lanDev, ICUUser currentUser, string filePath)
	{
		m_currentDevice = lanDev;
		m_currentUser = currentUser;
		if (!string.IsNullOrEmpty(filePath))
		{
			m_txtFilename.Text = filePath;
		}
		m_btnCreateFWU.Visible = m_currentUser != null && m_currentUser.GetRights("FEATURE_CREATEFWU") == ICURights.Full;
		if (lanDev != null)
		{
			m_lblDeviceHeader.Text = $"Upload logo to device '{lanDev.Identification}' (serial number: {lanDev.GetPropertyString(8273, 0, 0)})";
		}
		else
		{
			m_lblDeviceHeader.Text = string.Format("Create logo update file", Array.Empty<object>());
		}
		m_nMaxLogoWidth = s_nMaxLogoWidthLarge;
		m_nMaxLogoHeight = s_nMaxLogoHeightLarge;
		m_fDeviceHasDisplay = false;
		if (lanDev != null)
		{
			m_nMaxLogoWidth = lanDev.MaxLogoWidth;
			m_nMaxLogoHeight = lanDev.MaxLogoHeight;
			m_fDeviceHasDisplay = lanDev.HasDisplay;
		}
		m_btnUpload.Sensitive = m_fDeviceHasDisplay;
		RefreshImage();
	}

	protected override void OnCommandActivated(Command cmd)
	{
		Hide();
	}

	private void OnFilenameChanged(object sender, EventArgs e)
	{
		RefreshImage();
	}

	private void OnMarginChanged(object sender, EventArgs e)
	{
		RefreshImage();
	}

	private void RefreshImage()
	{
		if (File.Exists(m_txtFilename.Text))
		{
			m_btnUpload.Sensitive = m_currentDevice != null;
			m_spbMargin.MaximumValue = Math.Min(m_nMaxLogoWidth / 2, m_nMaxLogoHeight / 2);
			m_btnCreateFWU.Visible = m_currentUser != null && m_currentUser.GetRights("FEATURE_CREATEFWU") == ICURights.Full;
			m_btnCreateFWU.Sensitive = true;
			Image val = Image.FromFile(m_txtFilename.Text);
			int num = Convert.ToInt32(m_spbMargin.Value);
			System.Drawing.Size desiredSize = new System.Drawing.Size(m_nMaxLogoWidth - num * 2, m_nMaxLogoHeight - num * 2);
			m_imgConvertedImage = global::ICULogoConvertor.ICULogoConvertor.ConvertImage(val, desiredSize, s_nMaxColors);
			using MemoryStream memoryStream = new MemoryStream();
			m_imgConvertedImage.Save((Stream)memoryStream, ImageFormat.Png);
			memoryStream.Position = 0L;
			Image image = Image.FromStream(memoryStream);
			m_imgLogo.Image = image;
			m_txtInformation.Text = string.Format("Original image size: {0} x {1} pixels\nMaximum image size: {2} x {3}\nConverted image size: {4} x {5} pixels (margin {6})", new object[7] { val.Width, val.Height, m_nMaxLogoWidth, m_nMaxLogoHeight, image.Width, image.Height, num });
			return;
		}
		m_btnUpload.Sensitive = false;
		m_btnCreateFWU.Sensitive = false;
	}

	private void OnBtnUploadClicked(object sender, EventArgs e)
	{
		if (m_currentDevice == null)
		{
			return;
		}
		if (!m_fDeviceHasDisplay)
		{
			MessageDialog.ShowError(this, $"The current selected device does not have a display.\nYou cannot upload a logo to device '{m_currentDevice.Identification}'");
		}
		else if (m_bgwUpload != null && MessageDialog.AskQuestion($"Are you sure upload this new image to device '{m_currentDevice.Identification}'", Command.Yes, Command.No, Command.Cancel) == Command.Yes)
		{
			m_btnUpload.Sensitive = false;
			m_btnCreateFWU.Sensitive = false;
			m_btnBrowse.Sensitive = false;
			m_lstFilenames.Sensitive = false;
			m_txtFilename.Sensitive = false;
			m_btnUpload.Visible = false;
			m_btnCreateFWU.Visible = false;
			m_prbUpload.Visible = true;
			m_spnSpinner.Visible = true;
			m_spnSpinner.Animate = true;
			UploadResourceWorkerData uploadResourceWorkerData = new UploadResourceWorkerData
			{
				CurrentDevice = m_currentDevice,
				Image = m_imgConvertedImage,
				Worker = m_bgwUpload
			};
			if (m_currentDevice.isAHP)
			{
				uploadResourceWorkerData.Filename = m_txtFilename.Text;
				uploadResourceWorkerData.Margin = Convert.ToInt32(m_spbMargin.Value);
			}
			m_bgwUpload.RunWorkerAsync(uploadResourceWorkerData);
		}
	}

	private SaveFileDialog SetupSaveFirmwareDialog()
	{
		SaveFileDialog saveFileDialog = new SaveFileDialog("FWU file location")
		{
			Multiselect = false
		};
		if (m_currentDevice == null)
		{
			saveFileDialog.Filters.Add(new FileDialogFilter("fwu files", "*.fwu"));
			saveFileDialog.Filters.Add(new FileDialogFilter("tvf files", "*.tvf"));
			saveFileDialog.InitialFileName = Path.ChangeExtension(m_txtFilename.Text, "fwu");
		}
		else if (m_currentDevice.isAHP)
		{
			saveFileDialog.Filters.Add(new FileDialogFilter("tvf files", "*.tvf"));
			saveFileDialog.InitialFileName = Path.ChangeExtension(m_txtFilename.Text, "tvf");
		}
		else
		{
			saveFileDialog.Filters.Add(new FileDialogFilter("fwu files", "*.fwu"));
			saveFileDialog.InitialFileName = Path.ChangeExtension(m_txtFilename.Text, "fwu");
		}
		saveFileDialog.Filters.Add(new FileDialogFilter("All files", "*.*"));
		return saveFileDialog;
	}

	private void OnBtnCreateFWUClicked(object sender, EventArgs e)
	{
		SaveFileDialog saveFileDialog = SetupSaveFirmwareDialog();
		if (!saveFileDialog.Run())
		{
			return;
		}
		try
		{
			byte[] array = null;
			ICULanDevice currentDevice = m_currentDevice;
			if ((currentDevice != null && currentDevice.isAHP) || Path.GetExtension(saveFileDialog.FileName).ToLower() == ".tvf")
			{
				array = ICUTVFCreator.CreateTvfData(m_txtFilename.Text, AppProperties.AppName, Convert.ToInt32(m_spbMargin.Value));
			}
			else
			{
				bool flag = true;
				if (m_currentDevice != null && m_fDeviceHasDisplay && m_currentDevice.FirmwareVersionNumber < new Version("3.3.0"))
				{
					Command command = MessageDialog.AskQuestion(string.Format("Do you want to generate an resource file for an small (Eve mini) or a large display (EVE2) display?\nSelect Yes for a small display.", Array.Empty<object>()), Command.Yes, Command.No, Command.Cancel);
					if (command == Command.Cancel)
					{
						return;
					}
					flag = command != Command.Yes;
				}
				m_nMaxLogoWidth = (flag ? s_nMaxLogoWidthLarge : s_nMaxLogoWidth);
				m_nMaxLogoHeight = (flag ? s_nMaxLogoHeightLarge : s_nMaxLogoHeight);
				RefreshImage();
				bool createCFile = (Keyboard.CurrentModifiers & ModifierKeys.Shift) != 0 && (Keyboard.CurrentModifiers & ModifierKeys.Control) != 0;
				string directoryName = Path.GetDirectoryName(saveFileDialog.FileName);
				array = global::ICUFWUCreator.ICUFWUCreator.CreateFWUData(m_imgConvertedImage, s_nMaxColors, AppProperties.UILanguagesFolder, createCFile, directoryName, flag);
			}
			using (BinaryWriter binaryWriter = new BinaryWriter(File.Open(saveFileDialog.FileName, FileMode.Create)))
			{
				binaryWriter.Write(array);
			}
			Logger.AddChargerContext(m_currentDevice).Information("Image packaged and saved");
		}
		catch (Exception ex)
		{
			Logger.Debug(ex, "");
			MessageDialog.ShowError(this, ex.Message);
		}
	}

	private void OnUploadDoWork(object sender, DoWorkEventArgs e)
	{
		if (e.Argument is UploadResourceWorkerData { Image: not null, CurrentDevice: ICULanDevice currentDevice } uploadResourceWorkerData)
		{
			currentDevice.UploadResource(resourceData: (!currentDevice.isAHP) ? global::ICUFWUCreator.ICUFWUCreator.CreateFWUData(uploadResourceWorkerData.Image, s_nMaxColors, AppProperties.UILanguagesFolder, createCFile: false, "", m_nMaxLogoWidth > 320) : ICUTVFCreator.CreateTvfData(uploadResourceWorkerData.Filename, AppProperties.AppName, uploadResourceWorkerData.Margin), bgw: uploadResourceWorkerData.Worker);
			Logger.AddChargerContext(m_currentDevice).Information("Image uploaded");
		}
	}

	private void OnUploadCompleted(object sender, RunWorkerCompletedEventArgs e)
	{
		m_btnUpload.Sensitive = m_currentDevice != null;
		m_btnCreateFWU.Sensitive = true;
		m_btnBrowse.Sensitive = true;
		m_btnUpload.Visible = true;
		m_btnCreateFWU.Visible = m_currentUser != null && m_currentUser.GetRights("FEATURE_CREATEFWU") == ICURights.Full;
		m_lstFilenames.Sensitive = true;
		m_txtFilename.Sensitive = true;
		m_prbUpload.Visible = false;
		m_spnSpinner.Visible = false;
		m_spnSpinner.Animate = false;
		ICULanDevice currentDevice = m_currentDevice;
		if (currentDevice != null)
		{
			if (e.Error != null)
			{
				MessageDialog.ShowError(this, e.Error.Message);
			}
			else if (!string.IsNullOrEmpty(currentDevice.LastUploadError))
			{
				MessageDialog.ShowError(this, currentDevice.LastUploadError);
			}
			else
			{
				MessageDialog.ShowMessage(this, "Image uploaded successfully!");
			}
		}
	}

	private void OnUploadProgressChanged(object sender, ProgressChangedEventArgs e)
	{
		if (e.ProgressPercentage > 100)
		{
			m_prbUpload.Fraction = 1.0;
		}
		else
		{
			m_prbUpload.Fraction = (double)e.ProgressPercentage / 100.0;
		}
	}

	private void OnBtnBrowse(object sender, EventArgs e)
	{
		ImageCodecInfo[] imageEncoders = ImageCodecInfo.GetImageEncoders();
		string text = string.Join(";", imageEncoders.Select((ImageCodecInfo codec) => codec.FilenameExtension).ToArray());
		OpenFileDialog openFileDialog = new OpenFileDialog("Select an image")
		{
			InitialFileName = text,
			Multiselect = false
		};
		openFileDialog.Filters.Add(new FileDialogFilter("All Image files", text));
		openFileDialog.Filters.Add(new FileDialogFilter("All files", "*.*"));
		if (openFileDialog.Run())
		{
			m_txtFilename.Text = openFileDialog.FileName;
		}
	}
}
