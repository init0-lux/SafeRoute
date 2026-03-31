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
