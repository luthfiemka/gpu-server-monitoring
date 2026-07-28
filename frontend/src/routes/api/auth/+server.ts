import { json, redirect } from '@sveltejs/kit';
import type { RequestHandler } from './$types';
import { env } from '$env/dynamic/private';

const ADMIN_USER = env.ADMIN_USER || 'admin';
const ADMIN_PASS = env.ADMIN_PASS || 'admin';

export const POST: RequestHandler = async ({ request, cookies }) => {
  const { username, password } = await request.json();

  if (username === ADMIN_USER && password === ADMIN_PASS) {
    const user = { username };
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
