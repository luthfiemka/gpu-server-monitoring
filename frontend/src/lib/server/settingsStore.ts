import { env } from '$env/dynamic/private';
import { existsSync, mkdirSync, readFileSync, writeFileSync } from 'fs';
import { join } from 'path';
import { createHash, randomBytes } from 'crypto';

const SETTINGS_DIR = env.SETTINGS_DIR || join(process.cwd(), 'data', 'settings');

export interface BrandSettings {
  logo_url: string;
  brand_name: string;
}

export interface AppUser {
  username: string;
  password_hash: string;
  display_name: string;
  role: 'user' | 'admin';
  created_at: string;
}

function ensureDir() {
  if (!existsSync(SETTINGS_DIR)) {
    mkdirSync(SETTINGS_DIR, { recursive: true });
  }
}

function readJson<T>(file: string, fallback: T): T {
  ensureDir();
  const path = join(SETTINGS_DIR, file);
  if (!existsSync(path)) return fallback;
  try {
    return JSON.parse(readFileSync(path, 'utf-8'));
  } catch {
    return fallback;
  }
}

function writeJson<T>(file: string, data: T): void {
  ensureDir();
  writeFileSync(join(SETTINGS_DIR, file), JSON.stringify(data, null, 2), 'utf-8');
}

export function getBrandSettings(): BrandSettings {
  return readJson<BrandSettings>('brand.json', { logo_url: '', brand_name: 'GPU Dashboard' });
}

export function updateBrandSettings(data: Partial<BrandSettings>): BrandSettings {
  const current = getBrandSettings();
  const updated = { ...current, ...data };
  writeJson('brand.json', updated);
  return updated;
}

function hashPassword(password: string): string {
  const salt = randomBytes(16).toString('hex');
  const hash = createHash('sha256').update(salt + password).digest('hex');
  return salt + ':' + hash;
}

function verifyPassword(password: string, stored: string): boolean {
  const [salt, hash] = stored.split(':');
  const verify = createHash('sha256').update(salt + password).digest('hex');
  return hash === verify;
}

export function getUsers(): AppUser[] {
  return readJson<AppUser[]>('users.json', []);
}

export function createUser(username: string, password: string, displayName: string): AppUser {
  const users = getUsers();
  if (users.find(u => u.username === username)) {
    throw new Error('Username already exists');
  }
  const user: AppUser = {
    username,
    password_hash: hashPassword(password),
    display_name: displayName,
    role: 'user',
    created_at: new Date().toISOString()
  };
  users.push(user);
  writeJson('users.json', users);
  return { ...user, password_hash: '' };
}

export function updateUser(username: string, data: { display_name?: string; password?: string }): AppUser | null {
  const users = getUsers();
  const idx = users.findIndex(u => u.username === username);
  if (idx === -1) return null;
  if (data.display_name !== undefined) users[idx].display_name = data.display_name;
  if (data.password) users[idx].password_hash = hashPassword(data.password);
  writeJson('users.json', users);
  return { ...users[idx], password_hash: '' };
}

export function deleteUser(username: string): boolean {
  const users = getUsers();
  const filtered = users.filter(u => u.username !== username);
  if (filtered.length === users.length) return false;
  writeJson('users.json', filtered);
  return true;
}

export function verifyUser(username: string, password: string): AppUser | null {
  const users = getUsers();
  const user = users.find(u => u.username === username);
  if (!user) return null;
  if (!verifyPassword(password, user.password_hash)) return null;
  return { ...user, password_hash: '' };
}
