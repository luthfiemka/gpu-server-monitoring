import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { getGpuTrend } from '$lib/server/questdb';

const DEFAULT_SAMPLE_BY = '30m';

export const GET: RequestHandler = async ({ url }) => {
  const now = new Date();
  const weekAgo = new Date(now.getTime() - 7 * 24 * 60 * 60 * 1000);
  const from = url.searchParams.get('from') || weekAgo.toISOString().slice(0, 19);
  const to = url.searchParams.get('to') || now.toISOString().slice(0, 19);
  const sampleBy = url.searchParams.get('sample_by') || DEFAULT_SAMPLE_BY;

  try {
    const trend = await getGpuTrend(from, to, sampleBy);
    return json({ trend, from, to, sample_by: sampleBy });
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    return json({ error: message }, { status: 502 });
  }
};
