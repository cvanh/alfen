using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.Linq;
using System.Net.Http;
using System.Threading;
using System.Threading.Tasks;
using Newtonsoft.Json;
using Serilog;
using Serilog.Context;
using Serilog.Events;

namespace ICUNetwork;

public class LogHttpClientHandler(HttpMessageHandler innerHandler) : DelegatingHandler(innerHandler)
{
	private readonly ILogger Logger = Log.ForContext<LogHttpClientHandler>();

	protected override async Task<HttpResponseMessage> SendAsync(HttpRequestMessage request, CancellationToken cancellationToken)
	{
		string value = Guid.NewGuid().ToString().Substring(0, 5);
		using (LogContext.PushProperty("CorrelationId", value))
		{
			using (LogContext.PushProperty("IpAddress", request.RequestUri.Host))
			{
				if (request.Content is MultipartFormDataContent)
				{
					return await base.SendAsync(request, cancellationToken);
				}
				if ((!Logger.IsEnabled(LogEventLevel.Verbose) && request.RequestUri.ToString().Contains("/prop?ids=2059_0,2060_0")) || request.RequestUri.ToString().Contains("/prop?ids=2191_2,2191_3") || request.RequestUri.ToString().Contains("/prop?ids=2210_0,2211_0,2212_0"))
				{
					return await base.SendAsync(request, cancellationToken);
				}
				Stopwatch stopwatch = Stopwatch.StartNew();
				string requestHeaders = string.Join(", ", request.Headers.Select((KeyValuePair<string, IEnumerable<string>> h) => "\"" + h.Key + "\": \"" + string.Join(", ", h.Value) + "\""));
				string text = ((request.Content == null) ? "{}" : (await request.Content.ReadAsStringAsync()));
				string requestContent = text;
				string command = string.Empty;
				if (request.RequestUri.ToString().Contains("/cmd"))
				{
					command = JsonConvert.DeserializeAnonymousType(requestContent, new
					{
						command = string.Empty
					}).command;
				}
				HttpResponseMessage response = null;
				try
				{
					requestContent = ((!request.RequestUri.ToString().Contains("/login")) ? requestContent : "<masked>");
					if (!Logger.IsEnabled(LogEventLevel.Verbose))
					{
						requestHeaders = $"Length={requestHeaders.Length}";
						requestContent = $"Length={requestContent.Length}";
					}
					Logger.Debug("-> {Uri} {Command} {Request}", request.RequestUri, (!string.IsNullOrEmpty(command)) ? (command ?? "") : "", new
					{
						Method = request.Method,
						RequestUri = request.RequestUri,
						Headers = requestHeaders,
						Content = requestContent
					});
					_ = string.Empty;
					_ = string.Empty;
					response = await base.SendAsync(request, cancellationToken);
					stopwatch.Stop();
					text = ((response.Content == null) ? string.Empty : (await response.Content.ReadAsStringAsync()));
					string text2 = text;
					int length = text2.Length;
					string text3 = string.Join(", ", response.Headers.Select((KeyValuePair<string, IEnumerable<string>> h) => "\"" + h.Key + "\": \"" + string.Join(", ", h.Value) + "\""));
					if (!Logger.IsEnabled(LogEventLevel.Verbose) || request.RequestUri.ToString().Contains("/api/log"))
					{
						text3 = $"Length={text3.Length}";
						text2 = "...";
					}
					Logger.Debug("<- {StatusCode} {Uri} {Length:0.0}kB {ElapsedMilliseconds}ms {Command} {Response}", response.StatusCode, request.RequestUri, (double)length / 1024.0, stopwatch.ElapsedMilliseconds, (!string.IsNullOrEmpty(command)) ? (command + " ") : "", new
					{
						HttpStatus = (int)response.StatusCode + ": " + response.ReasonPhrase.Trim(),
						Headers = text3,
						Content = text2
					});
					return response;
				}
				catch (Exception ex)
				{
					stopwatch.Stop();
					Logger.Debug("<- {StatusCode} {Uri} {ElapsedMilliseconds}ms {Command} {Request} {ExceptionDetails}", response?.StatusCode.ToString() ?? "NOK", request.RequestUri, stopwatch.ElapsedMilliseconds, (!string.IsNullOrEmpty(command)) ? (command + " ") : "", new
					{
						Method = request.Method,
						RequestUri = request.RequestUri,
						Headers = requestHeaders,
						Content = requestContent
					}, new { ex.Message, ex.StackTrace });
					throw;
				}
			}
		}
	}
}
