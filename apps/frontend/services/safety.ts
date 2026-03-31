import { get, post } from './api';
import { GOOGLE_MAPS_API_KEY } from '@/constants/config';

// ── Types ────────────────────────────────────────────────────────────────────

export interface Coordinates {
  lat: number;
  lng: number;
}

export interface SafetyScoreResponse {
  score: number;
  risk_level: string;
  factors: {
    recent_reports: number;
    historical_reports: number;
    recent_trust_weight: number;
    historical_trust_weight: number;
    time_risk: string;
    time_risk_multiplier: number;
    confidence: string;
    confidence_score: number;
    radius_meters: number;
    recent_window_hours: number;
  };
}

export interface RouteSegment {
  start: Coordinates;
  end: Coordinates;
  score: number;
  risk_level: string;
  recent_reports: number;
}

export interface RouteSafetyResponse {
  score: number;
  risk_level: string;
  route: {
    distance_meters: number;
    duration_seconds: number;
    polyline: string;
  };
  segments: RouteSegment[];
  summary: {
    recent_reports: number;
    historical_reports: number;
    high_risk_segments: number;
    moderate_risk_segments: number;
    low_risk_segments: number;
  };
}

export interface PlacePrediction {
  place_id: string;
  description: string;
  main_text: string;
  secondary_text: string;
}

export interface PlaceDetails {
  place_id: string;
  name: string;
  formatted_address: string;
  coordinates: Coordinates;
}

// ── Safety API ───────────────────────────────────────────────────────────────

export async function getSafetyScore(
  lat: number,
  lng: number,
  radius?: number
): Promise<SafetyScoreResponse> {
  const params = new URLSearchParams({
    lat: lat.toString(),
    lng: lng.toString(),
  });
  if (radius) {
    params.append('radius', radius.toString());
  }
  return get<SafetyScoreResponse>(`/safety/score?${params.toString()}`);
}

export async function getRouteSafetyScore(
  origin: Coordinates,
  destination: Coordinates,
  travelMode: 'walking' | 'transit' = 'walking'
): Promise<RouteSafetyResponse> {
  return post<RouteSafetyResponse>('/safety/route-score', {
    origin,
    destination,
    travel_mode: travelMode,
  });
}

// ── Google Places API (Geocoding) ────────────────────────────────────────────

const PLACES_AUTOCOMPLETE_URL = 'https://maps.googleapis.com/maps/api/place/autocomplete/json';
const PLACES_DETAILS_URL = 'https://maps.googleapis.com/maps/api/place/details/json';
const GEOCODING_URL = 'https://maps.googleapis.com/maps/api/geocode/json';

export async function searchPlaces(
  query: string,
  location?: Coordinates
): Promise<PlacePrediction[]> {
  if (!GOOGLE_MAPS_API_KEY || !query.trim()) {
    return [];
  }

  const params = new URLSearchParams({
    input: query,
    key: GOOGLE_MAPS_API_KEY,
    types: 'geocode|establishment',
  });

  // Bias results toward user's location if available
  if (location) {
    params.append('location', `${location.lat},${location.lng}`);
    params.append('radius', '50000'); // 50km radius bias
  }

  try {
    const response = await fetch(`${PLACES_AUTOCOMPLETE_URL}?${params.toString()}`);
    const data = await response.json();

    if (data.status !== 'OK' && data.status !== 'ZERO_RESULTS') {
      console.warn('Places Autocomplete error:', data.status, data.error_message);
      return [];
    }

    return (data.predictions || []).map((p: any) => ({
      place_id: p.place_id,
      description: p.description,
      main_text: p.structured_formatting?.main_text || p.description,
      secondary_text: p.structured_formatting?.secondary_text || '',
    }));
  } catch (err) {
    console.error('Places Autocomplete failed:', err);
    return [];
  }
}

export async function getPlaceDetails(placeId: string): Promise<PlaceDetails | null> {
  if (!GOOGLE_MAPS_API_KEY || !placeId) {
    return null;
  }

  const params = new URLSearchParams({
    place_id: placeId,
    key: GOOGLE_MAPS_API_KEY,
    fields: 'place_id,name,formatted_address,geometry',
  });

  try {
    const response = await fetch(`${PLACES_DETAILS_URL}?${params.toString()}`);
    const data = await response.json();

    if (data.status !== 'OK') {
      console.warn('Place Details error:', data.status, data.error_message);
      return null;
    }

    const result = data.result;
    return {
      place_id: result.place_id,
      name: result.name,
      formatted_address: result.formatted_address,
      coordinates: {
        lat: result.geometry.location.lat,
        lng: result.geometry.location.lng,
      },
    };
  } catch (err) {
    console.error('Place Details failed:', err);
    return null;
  }
}

export async function geocodeAddress(address: string): Promise<Coordinates | null> {
  if (!GOOGLE_MAPS_API_KEY || !address.trim()) {
    return null;
  }

  const params = new URLSearchParams({
    address: address,
    key: GOOGLE_MAPS_API_KEY,
  });

  try {
    const response = await fetch(`${GEOCODING_URL}?${params.toString()}`);
    const data = await response.json();

    if (data.status !== 'OK' || !data.results?.length) {
      console.warn('Geocoding error:', data.status, data.error_message);
      return null;
    }

    const location = data.results[0].geometry.location;
    return {
      lat: location.lat,
      lng: location.lng,
    };
  } catch (err) {
    console.error('Geocoding failed:', err);
    return null;
  }
}
