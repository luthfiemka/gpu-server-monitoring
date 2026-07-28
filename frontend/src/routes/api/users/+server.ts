import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { getUsersSummary } from '$lib/server/questdb';

export const GET: RequestHandler = async () => {
  try {
    const users = await getUsersSummary();
    return json({ users });
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    return json({ error: message }, { status: 502 });
  }
};
