using System;
using System.Collections.Generic;
using System.IO.Ports;
using System.Linq;
using System.Text;
using System.Threading;
using Serilog;

namespace ICUNetwork;

public class COMPortInterface
{
	public delegate void updateVirtualListSizeDelegate();

	public delegate void ensureVisibleDelegate(int index);

	private readonly ILogger Logger = Log.ForContext<COMPortInterface>();

	private readonly SerialPort m_port;

	private readonly byte[] m_buffer = new byte[16384];

	private readonly byte[] m_LineBuffer = new byte[16384];

	private int m_lineBufferPos;

	private readonly List<ICULogLine> m_logLines = new List<ICULogLine>();

	private Timer m_timer;

	public bool EnsureNewLineVisible { get; set; }

	~COMPortInterface()
	{
		if (m_timer != null)
		{
			m_timer.Dispose();
			m_timer = null;
		}
		if (m_port != null && m_port.IsOpen)
		{
			m_port.Close();
		}
	}

	public bool IsValid()
	{
		if (m_port != null)
		{
			return m_port.IsOpen;
		}
		return false;
	}

	private void onDataReceived(byte[] received)
	{
		Buffer.BlockCopy(received, 0, m_LineBuffer, m_lineBufferPos, received.Length);
		m_lineBufferPos += received.Length;
		int num = 0;
		bool flag = false;
		do
		{
			flag = false;
			for (int i = 0; i < m_lineBufferPos; i++)
			{
				if (m_LineBuffer[i] == 10)
				{
					byte[] array = new byte[i];
					Buffer.BlockCopy(m_LineBuffer, 0, array, 0, i);
					string text = Encoding.Default.GetString(array).Trim(new char[3] { '\r', '\n', ' ' });
					if (!string.IsNullOrEmpty(text))
					{
						ICULogLine item = new ICULogLine(text);
						m_logLines.Add(item);
					}
					m_lineBufferPos -= i + 1;
					if (m_lineBufferPos > 0)
					{
						Buffer.BlockCopy(m_LineBuffer, i + 1, m_LineBuffer, 0, m_lineBufferPos);
					}
					flag = true;
					num++;
					break;
				}
			}
		}
		while (flag);
		_ = 0;
	}

	private void startRead()
	{
		Action kickoffRead = null;
		kickoffRead = () =>
		{
			m_port.BaseStream.BeginRead(m_buffer, 0, m_buffer.Length, (IAsyncResult ar) =>
			{
				try
				{
					int num = m_port.BaseStream.EndRead(ar);
					byte[] array = new byte[num];
					Buffer.BlockCopy(m_buffer, 0, array, 0, num);
					onDataReceived(array);
				}
				catch (Exception ex)
				{
					Logger.Error(ex, "Error during reading: {Message}", ex.Message);
				}
				if (m_port.IsOpen)
				{
					kickoffRead();
				}
			}, null);
		};
		kickoffRead();
	}

	private void ensureVisible(int index)
	{
	}

	public static bool IsPortAvailable()
	{
		return SerialPort.GetPortNames().Count() > 0;
	}

	public void Reset()
	{
		if (m_port.IsOpen)
		{
			m_port.WriteLine("\u0012");
		}
	}

	public void SendCommand(string command)
	{
		if (m_port.IsOpen)
		{
			m_port.WriteLine(command);
		}
	}

	public void SendKey(string key)
	{
		if (m_port.IsOpen)
		{
			m_port.Write(key);
		}
	}

	public void Clear()
	{
		m_logLines.Clear();
	}
}
