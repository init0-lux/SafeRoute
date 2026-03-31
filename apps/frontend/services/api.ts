import * as SecureStore from 'expo-secure-store';
import { API_BASE_URL } from '@/constants/config';

const TOKEN_KEYS = {
  ACCESS: 'saferoute_access_token',
  REFRESH: 'saferoute_refresh_token',
} as const;

// ── Token Storage (SecureStore) ──────────────────────────────────────────────

export async function getAccessToken(): Promise<string | null> {
  try {
    return await SecureStore.getItemAsync(TOKEN_KEYS.ACCESS);
  } catch {
    return null;
  }
}

export async function getRefreshToken(): Promise<string | null> {
  try {
    return await SecureStore.getItemAsync(TOKEN_KEYS.REFRESH);
  } catch {
    return null;
  }
}

export async function setTokens(accessToken: string, refreshToken: string): Promise<void> {
  await SecureStore.setItemAsync(TOKEN_KEYS.ACCESS, accessToken);
  await SecureStore.setItemAsync(TOKEN_KEYS.REFRESH, refreshToken);
}

export async function clearTokens(): Promise<void> {
  await SecureStore.deleteItemAsync(TOKEN_KEYS.ACCESS);
  await SecureStore.deleteItemAsync(TOKEN_KEYS.REFRESH);
}

// ── API Error ────────────────────────────────────────────────────────────────

export class ApiError extends Error {
  status: number;
  body: Record<string, unknown>;

  constructor(status: number, body: Record<string, unknown>) {
    super((body.error as string) ?? `Request failed with status ${status}`);
    this.status = status;
    this.body = body;
  }
}

// ── Core Fetch Wrapper ───────────────────────────────────────────────────────

let isRefreshing = false;
let refreshQueue: Array<{
  resolve: (token: string) => void;
  reject: (err: Error) => void;
}> = [];

function processRefreshQueue(error: Error | null, token: string | null) {
  refreshQueue.forEach(({ resolve, reject }) => {
    if (error) reject(error);
    else resolve(token!);
  });
  refreshQueue = [];
}

async function attemptTokenRefresh(): Promise<string> {
  const refreshToken = await getRefreshToken();
  if (!refreshToken) {
    throw new ApiError(401, { error: 'No refresh token available' });
  }

  const response = await fetch(`${API_BASE_URL}/auth/refresh`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      'X-Refresh-Token': refreshToken,
    },
  });

  if (!response.ok) {
    await clearTokens();
    throw new ApiError(response.status, { error: 'Token refresh failed' });
  }

  const data = await response.json();
  const { access_token, refresh_token } = data.tokens;
  await setTokens(access_token, refresh_token);
  return access_token;
}

interface RequestOptions {
  method?: string;
  body?: unknown;
  headers?: Record<string, string>;
  skipAuth?: boolean;
}

async function request<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const { method = 'GET', body, headers = {}, skipAuth = false } = options;

  const requestHeaders: Record<string, string> = {
    'Content-Type': 'application/json',
    ...headers,
  };

  if (!skipAuth) {
    const accessToken = await getAccessToken();
    if (accessToken) {
      requestHeaders['Authorization'] = `Bearer ${accessToken}`;
    }
  }

  let response = await fetch(`${API_BASE_URL}${path}`, {
    method,
    headers: requestHeaders,
    body: body ? JSON.stringify(body) : undefined,
  });

  // Attempt token refresh on 401 (only once, not for auth routes)
  if (response.status === 401 && !skipAuth && !path.startsWith('/auth/')) {
    if (!isRefreshing) {
      isRefreshing = true;
      try {
        const newToken = await attemptTokenRefresh();
        processRefreshQueue(null, newToken);
        isRefreshing = false;

        // Retry the original request with the new token
        requestHeaders['Authorization'] = `Bearer ${newToken}`;
        response = await fetch(`${API_BASE_URL}${path}`, {
          method,
          headers: requestHeaders,
          body: body ? JSON.stringify(body) : undefined,
        });
      } catch (err) {
        processRefreshQueue(err as Error, null);
        isRefreshing = false;
        throw err;
      }
    } else {
      // Another refresh is already in progress — queue this request
      const newToken = await new Promise<string>((resolve, reject) => {
        refreshQueue.push({ resolve, reject });
      });

      requestHeaders['Authorization'] = `Bearer ${newToken}`;
      response = await fetch(`${API_BASE_URL}${path}`, {
        method,
        headers: requestHeaders,
        body: body ? JSON.stringify(body) : undefined,
      });
    }
  }

  const data = await response.json();

  if (!response.ok) {
    throw new ApiError(response.status, data);
  }

  return data as T;
}

// ── Convenience Methods ──────────────────────────────────────────────────────

export function get<T>(path: string, options?: Omit<RequestOptions, 'method' | 'body'>): Promise<T> {
  return request<T>(path, { ...options, method: 'GET' });
}

export function post<T>(path: string, body?: unknown, options?: Omit<RequestOptions, 'method' | 'body'>): Promise<T> {
  return request<T>(path, { ...options, method: 'POST', body });
}

export function put<T>(path: string, body?: unknown, options?: Omit<RequestOptions, 'method' | 'body'>): Promise<T> {
  return request<T>(path, { ...options, method: 'PUT', body });
}

export function del<T>(path: string, options?: Omit<RequestOptions, 'method' | 'body'>): Promise<T> {
  return request<T>(path, { ...options, method: 'DELETE' });
}
