using System;
using Microsoft.ApplicationInsights.Channel;
using Microsoft.ApplicationInsights.Extensibility;

namespace ACEServiceInstaller;

internal class SessionInitializer : ITelemetryInitializer
{
	private static readonly string _sessionId = Guid.NewGuid().ToString();

	private static readonly string _domain = $"{(Environment.UserDomainName ?? string.Empty).GetHashCode():X}";

	private static readonly string _user = $"{(Environment.UserName ?? string.Empty).GetHashCode():X}";

	public void Initialize(ITelemetry telemetry)
	{
		telemetry.Context.Cloud.RoleInstance = _domain;
		telemetry.Context.User.Id = _user;
		telemetry.Context.Session.Id = _sessionId;
	}
}
