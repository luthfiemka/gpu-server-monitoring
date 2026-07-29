import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { env } from '$env/dynamic/private';
import { getBrandSettings, updateBrandSettings } from '$lib/server/settingsStore';

const ADMIN_USER = env.ADMIN_USER || 'admin';

function isAdmin(event: { locals: App.Locals }) {
  return event.locals.user?.username === ADMIN_USER;
}

export const GET: RequestHandler = async (event) => {
  if (!isAdmin(event)) return json({ error: 'Unauthorized' }, { status: 403 });
  return json(getBrandSettings());
};

export const PUT: RequestHandler = async (event) => {
  if (!isAdmin(event)) return json({ error: 'Unauthorized' }, { status: 403 });
  const body = await event.request.json();
  const updated = updateBrandSettings({
    logo_url: body.logo_url,
    brand_name: body.brand_name
  });
  return json(updated);
};
