using System;
using Serilog.Core;
using Serilog.Events;

namespace ACEServiceInstaller;

public class AlfenDomain : ILogEventEnricher
{
	private static readonly bool _inAlfenDomain;

	static AlfenDomain()
	{
		_inAlfenDomain = Environment.UserDomainName.Equals("VANALFEN");
	}

	public void Enrich(LogEvent logEvent, ILogEventPropertyFactory propertyFactory)
	{
		LogEventProperty property = propertyFactory.CreateProperty("AlfenDomain", _inAlfenDomain);
		logEvent.AddOrUpdateProperty(property);
	}
}
