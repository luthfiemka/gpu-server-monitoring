import type { Handle } from '@sveltejs/kit';

const PUBLIC_ROUTES = ['/login'];
const isApiRoute = (path: string) => path.startsWith('/api/');

export const handle: Handle = async ({ event, resolve }) => {
  if (isApiRoute(event.url.pathname)) {
    const session = event.cookies.get('session');
    event.locals.user = session ? JSON.parse(session) : null;
    return resolve(event);
  }

  const session = event.cookies.get('session');

  if (!session && !PUBLIC_ROUTES.includes(event.url.pathname)) {
    return new Response(null, {
      status: 302,
      headers: { location: '/login' }
    });
  }

  if (session && PUBLIC_ROUTES.includes(event.url.pathname)) {
    return new Response(null, {
      status: 302,
      headers: { location: '/' }
    });
  }

  event.locals.user = session ? JSON.parse(session) : null;
  return resolve(event);
};
