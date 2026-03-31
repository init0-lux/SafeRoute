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

// Placeholder API key — replace with your actual Google Maps API key
export const GOOGLE_MAPS_API_KEY = 'YOUR_API_KEY_HERE';

export const INCIDENT_TYPES = [
    'Rape / Attempt to Rape',
    'Sexual Harassment',
    'Molestation',
    'Domestic Violence',
    'Dowry Harassment / Dowry Death',
    'Kidnapping / Abduction',
    'Stalking / Cyberstalking',
    'Cyber Crime Against Women',
    'Acid Attack',
    'Trafficking',
    'Insult to Modesty',
    'Other',
] as const;

export type IncidentType = (typeof INCIDENT_TYPES)[number];
