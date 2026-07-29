import { json, redirect } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { env } from '$env/dynamic/private';
import { verifyUser } from '$lib/server/settingsStore';

const ADMIN_USER = env.ADMIN_USER || 'admin';
const ADMIN_PASS = env.ADMIN_PASS || 'admin';

export const POST: RequestHandler = async ({ request, cookies }) => {
  const { username, password } = await request.json();

  if (username === ADMIN_USER && password === ADMIN_PASS) {
    const user = { username, role: 'admin' };
    cookies.set('session', JSON.stringify(user), {
      path: '/',
      httpOnly: true,
      sameSite: 'lax',
      maxAge: 60 * 60 * 24
    });
    return json({ ok: true });
  }

  const dbUser = verifyUser(username, password);
  if (dbUser) {
    const user = { username: dbUser.username, role: dbUser.role };
    cookies.set('session', JSON.stringify(user), {
      path: '/',
      httpOnly: true,
      sameSite: 'lax',
      maxAge: 60 * 60 * 24
    });
    return json({ ok: true });
  }

  return json({ ok: false, error: 'Invalid credentials' }, { status: 401 });
};

export const DELETE: RequestHandler = async ({ cookies }) => {
  cookies.delete('session', { path: '/' });
  throw redirect(302, '/login');
};
