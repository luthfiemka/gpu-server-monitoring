import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { getContainersSummary } from '$lib/server/questdb';

export const GET: RequestHandler = async () => {
  try {
    const containers = await getContainersSummary();
    return json({ containers });
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    return json({ error: message }, { status: 502 });
  }
};
