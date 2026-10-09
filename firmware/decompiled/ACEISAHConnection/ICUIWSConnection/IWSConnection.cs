using System;
using System.Net;
using System.Net.Http;
using System.Net.Http.Headers;
using System.Text;
using System.Threading;
using System.Threading.Tasks;
using Newtonsoft.Json;
using Serilog;

namespace ICUIWSConnection;

public class IWSConnection
{
	private static readonly ILogger Logger = Log.ForContext<IWSConnection>();

	public const int s_nRequestTimeout = 8000;

	public static HttpResponseMessage GetObjectData(ref IWSObject iwsobj, string url, string security, string objectId, int numberOfSockets = 1, string processorId = "", string additionalInfo = "", bool productionRequest = false, string objectCode = "", bool includeLogo = true)
	{
		IsahResponse result = Task.Run(async () => await GetObjectDataAsync(url, security, objectId, numberOfSockets, processorId, additionalInfo, productionRequest, objectCode, includeLogo)).Result;
		iwsobj = result.IsahObject;
		return result.HttpResponse;
	}

	public static async Task<IsahResponse> GetObjectDataAsync(string url, string security, string objectId, int numberOfSockets = 1, string processorId = "", string additionalInfo = "", bool productionRequest = false, string objectCode = "", bool includeLogo = true)
	{
		IsahResponse isahResponse = new IsahResponse
		{
			HttpResponse = null,
			IsahObject = null
		};
		using (HttpClient m_httpClient = new HttpClient
		{
			BaseAddress = new Uri(url),
			Timeout = TimeSpan.FromMinutes(5.0)
		})
		{
			using CancellationTokenSource cancellationToken = new CancellationTokenSource(8000);
			_ = 1;
			try
			{
				m_httpClient.DefaultRequestHeaders.Accept.Add(new MediaTypeWithQualityHeaderValue("application/json"));
				byte[] bytes = Encoding.ASCII.GetBytes(security);
				m_httpClient.DefaultRequestHeaders.Authorization = new AuthenticationHeaderValue("Basic", Convert.ToBase64String(bytes));
				IsahResponse isahResponse2 = isahResponse;
				isahResponse2.HttpResponse = await m_httpClient.GetAsync(string.Format("api/settings/{0}?NumberOfSockets={1}&ProcessorId={2}&AdditionalInfo={3}&ProductionRequest={4}&ObjectCode={5}&IncludeLogo={6}", new object[7] { objectId, numberOfSockets, processorId, additionalInfo, productionRequest, objectCode, includeLogo }), cancellationToken.Token);
				if (isahResponse.HttpResponse.IsSuccessStatusCode)
				{
					isahResponse.IsahObject = JsonConvert.DeserializeObject<IWSObject>(await isahResponse.HttpResponse.Content.ReadAsStringAsync());
				}
			}
			catch (Exception exception)
			{
				isahResponse.HttpResponse = new HttpResponseMessage(HttpStatusCode.RequestTimeout);
				Logger.Error(exception, "Failed to connect to IWS");
			}
		}
		return isahResponse;
	}

	public static async Task<bool> TestIWSConnection(string url)
	{
		bool result = false;
		HttpClient m_httpClient = new HttpClient
		{
			BaseAddress = new Uri(url),
			Timeout = TimeSpan.FromMinutes(5.0)
		};
		CancellationTokenSource cancellationToken = new CancellationTokenSource(8000);
		try
		{
			result = (await m_httpClient.GetAsync(url, cancellationToken.Token)).IsSuccessStatusCode;
		}
		catch (Exception exception)
		{
			Logger.Debug(exception, "Failed to connect to IWS");
			result = false;
		}
		finally
		{
			cancellationToken.Dispose();
			m_httpClient.Dispose();
		}
		return result;
	}
}
