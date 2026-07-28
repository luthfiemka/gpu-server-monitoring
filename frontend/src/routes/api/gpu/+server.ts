import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { getLatestGpuMetrics, getLatestProcesses } from '$lib/server/questdb';

export const GET: RequestHandler = async () => {
  try {
    const [gpus, processes] = await Promise.all([
      getLatestGpuMetrics(),
      getLatestProcesses()
    ]);
    return json({ gpus, processes });
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    return json({ error: message }, { status: 502 });
  }
};
