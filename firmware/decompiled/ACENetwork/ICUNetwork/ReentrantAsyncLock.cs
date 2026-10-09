using System;
using System.Threading;
using System.Threading.Tasks;

namespace ICUNetwork;

internal class ReentrantAsyncLock
{
	private readonly AsyncLocal<SemaphoreSlim> _semaphore = new AsyncLocal<SemaphoreSlim>();

	public async Task<T> WithLock<T>(Func<Task<T>> func)
	{
		SemaphoreSlim currentSemaphore = _semaphore.Value ?? new SemaphoreSlim(1);
		await currentSemaphore.WaitAsync();
		SemaphoreSlim nextSemaphore = new SemaphoreSlim(1);
		_semaphore.Value = nextSemaphore;
		T result;
		try
		{
			result = await func();
		}
		finally
		{
			await nextSemaphore.WaitAsync();
			_semaphore.Value = currentSemaphore;
			currentSemaphore.Release();
		}
		return result;
	}
}
