import Constants from 'expo-constants';

// Resolve the backend host automatically:
//  - On a physical device / Expo Go: use the same IP the Metro dev server is running on
//  - On Android emulator: Metro uses 10.0.2.2 as the host alias
//  - On iOS simulator / web: use localhost
function getApiBaseUrl(): string {
  const backendPort = 8080;

  // expo-constants exposes the dev server URL when running via Expo Go / dev client
  const hostUri = Constants.expoConfig?.hostUri;
  if (hostUri) {
    // hostUri looks like "10.213.47.64:8081" — strip the port, use the backend port
    const host = hostUri.split(':')[0];
    return `http://${host}:${backendPort}/api/v1`;
  }

  // Fallback for standalone builds — update this to your production URL
  return `http://localhost:${backendPort}/api/v1`;
}

export const API_BASE_URL = getApiBaseUrl();

// Read API key from environment variable
export const GOOGLE_MAPS_API_KEY = process.env.EXPO_PUBLIC_GOOGLE_ROUTES_API_KEY || '';

// Backend-accepted report types: harassment, unsafe_area, stalking, assault, theft, suspicious_activity
// We map user-friendly display labels to API-accepted slugs.
export const INCIDENT_TYPES_MAP: Record<string, string> = {
    'Harassment': 'harassment',
    'Unsafe Area': 'unsafe_area',
    'Stalking': 'stalking',
    'Assault': 'assault',
    'Theft': 'theft',
    'Suspicious Activity': 'suspicious_activity',
} as const;

export const INCIDENT_TYPES = Object.keys(INCIDENT_TYPES_MAP) as readonly string[];

export type IncidentType = (typeof INCIDENT_TYPES)[number];
