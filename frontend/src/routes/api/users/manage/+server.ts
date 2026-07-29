import { json } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { env } from '$env/dynamic/private';
import { getUsers, createUser, updateUser, deleteUser } from '$lib/server/settingsStore';

const ADMIN_USER = env.ADMIN_USER || 'admin';

function isAdmin(event: { locals: App.Locals }) {
  return event.locals.user?.username === ADMIN_USER;
}

export const GET: RequestHandler = async (event) => {
  if (!isAdmin(event)) return json({ error: 'Unauthorized' }, { status: 403 });
  const users = getUsers().map(({ password_hash, ...u }) => ({ ...u, password_hash: '' }));
  return json(users);
};

export const POST: RequestHandler = async (event) => {
  if (!isAdmin(event)) return json({ error: 'Unauthorized' }, { status: 403 });
  const { username, password, display_name } = await event.request.json();
  if (!username || !password) {
    return json({ error: 'Username and password required' }, { status: 400 });
  }
  try {
    const user = createUser(username, password, display_name || username);
    return json(user, { status: 201 });
  } catch (e) {
    return json({ error: (e as Error).message }, { status: 409 });
  }
};

export const PUT: RequestHandler = async (event) => {
  if (!isAdmin(event)) return json({ error: 'Unauthorized' }, { status: 403 });
  const { username, display_name, password } = await event.request.json();
  if (!username) return json({ error: 'Username required' }, { status: 400 });
  const updated = updateUser(username, { display_name, password });
  if (!updated) return json({ error: 'User not found' }, { status: 404 });
  return json(updated);
};

export const DELETE: RequestHandler = async (event) => {
  if (!isAdmin(event)) return json({ error: 'Unauthorized' }, { status: 403 });
  const { username } = await event.request.json();
  if (!username) return json({ error: 'Username required' }, { status: 400 });
  const deleted = deleteUser(username);
  if (!deleted) return json({ error: 'User not found' }, { status: 404 });
  return json({ ok: true });
};
