import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { getGpuHistory, getGpuProcessesHistory } from '$lib/server/questdb';

export const GET: RequestHandler = async ({ url }) => {
  const hostname = url.searchParams.get('hostname');
  const gpuId = url.searchParams.get('gpu_id');
  const from = url.searchParams.get('from');
  const to = url.searchParams.get('to');
  const sampleBy = url.searchParams.get('sample_by') || '1m';

  if (!hostname || !gpuId || !from || !to) {
    return json({ error: 'Missing hostname, gpu_id, from, or to params' }, { status: 400 });
  }

  try {
    const [history, processes] = await Promise.all([
      getGpuHistory(hostname, gpuId, from, to, sampleBy),
      getGpuProcessesHistory(hostname, gpuId, from, to, sampleBy)
    ]);
    return json({ history, processes });
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    return json({ error: message }, { status: 502 });
  }
};
