import { post, get, setTokens, clearTokens } from './api';

// ── Types ────────────────────────────────────────────────────────────────────

export interface User {
  id: string;
  username: string;
  phone: string;
  email?: string;
  trust_score: number;
  verified: boolean;
}

interface AuthResponse {
  user: User;
  tokens: {
    access_token: string;
    refresh_token: string;
    expires_in: number;
  };
}

interface LogoutResponse {
  status: string;
}

// ── Auth Service ─────────────────────────────────────────────────────────────

export async function register(username: string, phone: string, password: string, email?: string): Promise<User> {
  const data = await post<AuthResponse>('/auth/register', {
    username,
    phone,
    password,
    ...(email ? { email } : {}),
  }, { skipAuth: true });

  await setTokens(data.tokens.access_token, data.tokens.refresh_token);
  return data.user;
}

export async function login(phone: string, password: string): Promise<User> {
  const data = await post<AuthResponse>('/auth/login', {
    phone,
    password,
  }, { skipAuth: true });

  await setTokens(data.tokens.access_token, data.tokens.refresh_token);
  return data.user;
}

export async function getMe(): Promise<User> {
  const data = await get<{ user: User }>('/auth/me');
  return data.user;
}

export async function logout(): Promise<void> {
  try {
    await post<LogoutResponse>('/auth/logout');
  } catch {
    // Ignore network errors on logout — we clear tokens anyway
  }
  await clearTokens();
}

export async function updatePushToken(token: string): Promise<void> {
  await post('/auth/push-token', { token });
}
