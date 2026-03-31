import { post, get, getAccessToken } from './api';
import { API_BASE_URL } from '@/constants/config';

// ── Types ────────────────────────────────────────────────────────────────────

export interface SOSSession {
  id: string;
  user_id: string;
  status: 'active' | 'ended';
  started_at: string;
  ended_at?: string;
}

export interface SOSStartResponse {
  session: SOSSession;
  notifications_sent?: number;
}

export interface ViewerGrant {
  id: string;
  session_id: string;
  user_id: string;
  trusted_contact_id: string;
  expires_at: string;
  created_at: string;
}

export interface ViewerGrantResponse {
  grant: ViewerGrant;
  viewer_token: string;
  sse_url: string;
}

export interface LocationPing {
  lat: number;
  lng: number;
  ts: string;
}

export interface SOSLocationEvent {
  type: 'ready' | 'location' | 'ended';
  session_id?: string;
  contact_id?: string;
  location?: LocationPing;
  message?: string;
}

// ── SOS API ──────────────────────────────────────────────────────────────────

export async function startSOS(): Promise<SOSSession> {
  const data = await post<SOSStartResponse>('/sos/start', {});
  return data.session;
}

export async function getSOSSession(sessionId: string): Promise<SOSSession> {
  const data = await get<{ session: SOSSession }>(`/sos/${sessionId}`);
  return data.session;
}

export async function endSOS(sessionId: string): Promise<SOSSession> {
  const data = await post<SOSStartResponse>(`/sos/${sessionId}/end`, {});
  return data.session;
}

export async function pingSOSLocation(
  sessionId: string,
  lat: number,
  lng: number
): Promise<void> {
  await post(`/sos/${sessionId}/ping`, {
    lat,
    lng,
    ts: new Date().toISOString(),
  });
}

export async function createViewerGrant(
  sessionId: string,
  trustedContactId: string
): Promise<ViewerGrantResponse> {
  return post<ViewerGrantResponse>(`/sos/${sessionId}/viewers`, {
    trusted_contact_id: trustedContactId,
  });
}

// ── SOS Viewer Stream (SSE) ──────────────────────────────────────────────────

export type SOSEventCallback = (event: SOSLocationEvent) => void;
export type SOSErrorCallback = (error: Error) => void;

export interface SOSViewerStream {
  close: () => void;
}

export function connectToSOSViewerStream(
  viewerToken: string,
  onEvent: SOSEventCallback,
  onError?: SOSErrorCallback
): SOSViewerStream {
  const sseUrl = `${API_BASE_URL}/sos/viewer/stream?token=${encodeURIComponent(viewerToken)}`;
  
  // Use EventSource for SSE - native support in React Native via polyfill
  let eventSource: EventSource | null = null;
  let isClosed = false;

  const connect = () => {
    if (isClosed) return;

    try {
      eventSource = new EventSource(sseUrl);

      eventSource.onopen = () => {
        console.log('[SOS Viewer] SSE connection opened');
      };

      eventSource.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          
          if (data.type === 'ready') {
            onEvent({
              type: 'ready',
              session_id: data.session_id,
              contact_id: data.contact_id,
            });
          } else if (data.type === 'location' || data.lat !== undefined) {
            onEvent({
              type: 'location',
              location: {
                lat: data.lat,
                lng: data.lng,
                ts: data.ts || new Date().toISOString(),
              },
            });
          } else if (data.type === 'ended') {
            onEvent({
              type: 'ended',
              message: data.message || 'SOS session has ended',
            });
          }
        } catch (parseErr) {
          console.warn('[SOS Viewer] Failed to parse SSE event:', parseErr);
        }
      };

      eventSource.onerror = (err) => {
        console.error('[SOS Viewer] SSE error:', err);
        if (!isClosed) {
          onError?.(new Error('SSE connection error'));
          // Attempt reconnect after delay
          eventSource?.close();
          setTimeout(connect, 3000);
        }
      };
    } catch (err) {
      console.error('[SOS Viewer] Failed to create EventSource:', err);
      onError?.(err as Error);
    }
  };

  connect();

  return {
    close: () => {
      isClosed = true;
      eventSource?.close();
      eventSource = null;
    },
  };
}

// ── SOS WebSocket Stream (for Reporter) ──────────────────────────────────────

export interface SOSReporterStream {
  sendLocation: (lat: number, lng: number) => void;
  close: () => void;
}

export async function connectToSOSReporterStream(
  sessionId: string,
  onAck?: (recorded_at: string) => void,
  onError?: SOSErrorCallback
): Promise<SOSReporterStream> {
  const token = await getAccessToken();
  if (!token) {
    throw new Error('Not authenticated');
  }

  // Convert http(s) to ws(s) and add token as query param (since WS doesn't support headers easily)
  const wsUrl = API_BASE_URL.replace(/^http/, 'ws') + `/sos/${sessionId}/stream?token=${encodeURIComponent(token)}`;
  
  let ws: WebSocket | null = null;
  let isClosed = false;

  const connect = () => {
    if (isClosed) return;

    ws = new WebSocket(wsUrl);

    ws.onopen = () => {
      console.log('[SOS Reporter] WebSocket connected');
    };

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        if (data.status === 'accepted' && data.recorded_at) {
          onAck?.(data.recorded_at);
        }
      } catch (err) {
        console.warn('[SOS Reporter] Failed to parse WS message:', err);
      }
    };

    ws.onerror = (err) => {
      console.error('[SOS Reporter] WebSocket error:', err);
      onError?.(new Error('WebSocket connection error'));
    };

    ws.onclose = () => {
      if (!isClosed) {
        console.log('[SOS Reporter] WebSocket closed, reconnecting...');
        setTimeout(connect, 2000);
      }
    };
  };

  connect();

  return {
    sendLocation: (lat: number, lng: number) => {
      if (ws?.readyState === WebSocket.OPEN) {
        ws.send(JSON.stringify({
          lat,
          lng,
          ts: new Date().toISOString(),
        }));
      }
    },
    close: () => {
      isClosed = true;
      ws?.close();
      ws = null;
    },
  };
}
