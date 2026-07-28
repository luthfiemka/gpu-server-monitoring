import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { getHistory } from '$lib/server/questdb';

export const GET: RequestHandler = async ({ url }) => {
  const from = url.searchParams.get('from');
  const to = url.searchParams.get('to');
  const sampleBy = url.searchParams.get('sample_by') || '5m';

  if (!from || !to) {
    return json({ error: 'Missing from or to params' }, { status: 400 });
  }

  try {
    const history = await getHistory(from, to, sampleBy);
    return json({ history });
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    return json({ error: message }, { status: 502 });
  }
};
