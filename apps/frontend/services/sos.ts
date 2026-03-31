import { ApiError, post, get, getAccessToken } from './api';
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

export interface StartSOSInput {
  lat?: number;
  lng?: number;
  ts?: string;
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

export interface ActiveSOSAlert {
  session_id: string;
  trusted_contact_id: string;
  viewer_token: string;
  reporter_name: string;
  reporter_phone: string;
  started_at: string;
  lat?: number;
  lng?: number;
  recorded_at?: string;
}

export interface SOSViewerStatusResponse {
  session_id: string;
  trusted_contact_id: string;
  status: 'active' | 'ended';
  ended_at?: string;
  lat?: number;
  lng?: number;
  recorded_at?: string;
}

// ── SOS API ──────────────────────────────────────────────────────────────────

export async function startSOS(input?: StartSOSInput): Promise<SOSSession> {
  const data = await post<SOSStartResponse>('/sos/start', input || {});
  return data.session;
}

export async function getSOSSession(sessionId: string): Promise<SOSSession> {
  const data = await get<{ session: SOSSession }>(`/sos/${sessionId}`);
  return data.session;
}

export async function getActiveSOSSession(): Promise<SOSSession> {
  const data = await get<{ session: SOSSession }>('/sos/active');
  return data.session;
}

export async function endSOS(sessionId: string): Promise<SOSSession> {
  const data = await post<SOSStartResponse>(`/sos/${sessionId}/end`, {});
  return data.session;
}

export async function endActiveSOS(): Promise<SOSSession> {
  const data = await post<SOSStartResponse>('/sos/active/end', {});
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

export async function getActiveSOSAlerts(): Promise<ActiveSOSAlert[]> {
  try {
    const data = await get<{ alerts: ActiveSOSAlert[] }>('/sos/alerts/active');
    return data.alerts || [];
  } catch (err) {
    console.error('Failed to get active SOS alerts:', err);
    return [];
  }
}

export async function getSOSViewerStatus(
  viewerToken: string
): Promise<SOSViewerStatusResponse> {
  return get<SOSViewerStatusResponse>(
    `/sos/viewer/status?token=${encodeURIComponent(viewerToken)}`,
    { skipAuth: true }
  );
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

  let eventSource: EventSource | null = null;
  let pollTimeout: ReturnType<typeof setTimeout> | null = null;
  let isClosed = false;
  let readyEventSent = false;
  let endedEventSent = false;
  let lastLocationSignature: string | null = null;

  const isTerminalViewerError = (error: unknown) => {
    if (!(error instanceof ApiError)) {
      return false;
    }

    const message = error.message.toLowerCase();
    return (
      error.status === 400 ||
      error.status === 404 ||
      message.includes('viewer grant revoked') ||
      message.includes('viewer grant expired') ||
      message.includes('viewer token is invalid')
    );
  };

  const emitViewerStatus = (status: SOSViewerStatusResponse) => {
    if (!readyEventSent) {
      readyEventSent = true;
      onEvent({
        type: 'ready',
        session_id: status.session_id,
        contact_id: status.trusted_contact_id,
      });
    }

    if (status.lat !== undefined && status.lng !== undefined) {
      const signature = status.recorded_at || `${status.lat}:${status.lng}`;
      if (signature !== lastLocationSignature) {
        lastLocationSignature = signature;
        onEvent({
          type: 'location',
          session_id: status.session_id,
          location: {
            lat: status.lat,
            lng: status.lng,
            ts: status.recorded_at || new Date().toISOString(),
          },
        });
      }
    }

    if (status.status === 'ended' && !endedEventSent) {
      endedEventSent = true;
      onEvent({
        type: 'ended',
        message: 'SOS session has ended',
      });
    }
  };

  const pollViewerStatus = async () => {
    if (isClosed) {
      return;
    }

    try {
      const status = await getSOSViewerStatus(viewerToken);
      emitViewerStatus(status);

      if (!endedEventSent) {
        pollTimeout = setTimeout(pollViewerStatus, 3000);
      }
    } catch (error) {
      console.error('[SOS Viewer] Polling error:', error);
      onError?.(error as Error);

      if (isTerminalViewerError(error)) {
        isClosed = true;
        return;
      }

      if (!isClosed) {
        pollTimeout = setTimeout(pollViewerStatus, 5000);
      }
    }
  };

  const connect = () => {
    if (isClosed) return;

    const EventSourceCtor = globalThis.EventSource;
    if (typeof EventSourceCtor === 'undefined') {
      pollViewerStatus();
      return;
    }

    try {
      eventSource = new EventSourceCtor(sseUrl);

      const handleIncomingEvent = (event: MessageEvent, explicitType?: SOSLocationEvent['type']) => {
        try {
          const data = JSON.parse(event.data);
          const eventType = explicitType || data.type;

          if (eventType === 'ready') {
            onEvent({
              type: 'ready',
              session_id: data.session_id,
              contact_id: data.contact_id || data.trusted_contact_id,
            });
            return;
          }

          if (eventType === 'location' || data.lat !== undefined) {
            onEvent({
              type: 'location',
              session_id: data.session_id,
              location: {
                lat: data.lat,
                lng: data.lng,
                ts: data.recorded_at || data.ts || new Date().toISOString(),
              },
            });
            return;
          }

          if (eventType === 'ended') {
            onEvent({
              type: 'ended',
              message: data.message || 'SOS session has ended',
            });
          }
        } catch (parseErr) {
          console.warn('[SOS Viewer] Failed to parse SSE event:', parseErr);
        }
      };

      eventSource.onopen = () => {
        console.log('[SOS Viewer] SSE connection opened');
      };

      eventSource.onmessage = (event) => {
        handleIncomingEvent(event);
      };

      const typedEventSource = eventSource as EventSource & {
        addEventListener?: (type: string, listener: (event: MessageEvent) => void) => void;
      };
      typedEventSource.addEventListener?.('ready', (event) => handleIncomingEvent(event, 'ready'));
      typedEventSource.addEventListener?.('location', (event) => handleIncomingEvent(event, 'location'));
      typedEventSource.addEventListener?.('ended', (event) => handleIncomingEvent(event, 'ended'));

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
      if (pollTimeout) {
        clearTimeout(pollTimeout);
        pollTimeout = null;
      }
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
