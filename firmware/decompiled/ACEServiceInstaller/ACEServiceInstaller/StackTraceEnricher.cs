using System.Diagnostics;
using System.Reflection;
using Serilog.Core;
using Serilog.Events;

namespace ACEServiceInstaller;

public class StackTraceEnricher : ILogEventEnricher
{
	public void Enrich(LogEvent logEvent, ILogEventPropertyFactory propertyFactory)
	{
		StackFrame frame = new StackTrace(fNeedFileInfo: true).GetFrame(2);
		if (frame != null)
		{
			MethodBase method = frame.GetMethod();
			string fileName = frame.GetFileName();
			int fileLineNumber = frame.GetFileLineNumber();
			logEvent.AddPropertyIfAbsent(propertyFactory.CreateProperty("MemberName", method?.Name));
			logEvent.AddPropertyIfAbsent(propertyFactory.CreateProperty("FilePath", fileName));
			logEvent.AddPropertyIfAbsent(propertyFactory.CreateProperty("LineNumber", fileLineNumber));
		}
	}
}
