import type { LayoutServerLoad } from './$types';
import { getBrandSettings } from '$lib/server/settingsStore';

export const load: LayoutServerLoad = async ({ locals }) => {
  const brand = getBrandSettings();
  return { user: locals.user, brand };
};
