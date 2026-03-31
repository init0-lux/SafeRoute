import { useEffect, useState, useCallback, useRef } from 'react';
import { Alert, AppState } from 'react-native';
import { router } from 'expo-router';
import {
  addNotificationReceivedListener,
  addNotificationResponseListener,
  NotificationData,
  Notification,
  NotificationResponse,
  NotificationSubscription,
} from '../services/notifications';
import { getActiveSOSAlerts } from '../services/sos';

interface ActiveSOSAlert {
  sessionId: string;
  reporterName: string;
  startedAt: string;
  viewerToken: string;
  latitude?: number;
  longitude?: number;
  recordedAt?: string;
}

export function useSOSNotifications() {
  const [activeAlerts, setActiveAlerts] = useState<ActiveSOSAlert[]>([]);
  const [hasUnseenAlerts, setHasUnseenAlerts] = useState(false);
  const notificationListener = useRef<NotificationSubscription | null>(null);
  const responseListener = useRef<NotificationSubscription | null>(null);
  const appState = useRef(AppState.currentState);
  const announcedSessions = useRef<Set<string>>(new Set());

  const navigateToAlert = useCallback((alert: ActiveSOSAlert) => {
    router.push({
      pathname: '/sos-viewer',
      params: {
        token: alert.viewerToken,
        contactName: alert.reporterName,
        lat: alert.latitude?.toString(),
        lng: alert.longitude?.toString(),
      },
    });
  }, []);

  const upsertAlert = useCallback((alert: ActiveSOSAlert) => {
    setActiveAlerts((prev) => {
      const existingIndex = prev.findIndex((item) => item.sessionId === alert.sessionId);
      if (existingIndex === -1) {
        return [alert, ...prev];
      }

      const next = [...prev];
      next[existingIndex] = {
        ...next[existingIndex],
        ...alert,
      };
      return next;
    });
  }, []);

  const announceAlert = useCallback((alert: ActiveSOSAlert) => {
    if (announcedSessions.current.has(alert.sessionId)) {
      return;
    }

    announcedSessions.current.add(alert.sessionId);
    upsertAlert(alert);
    setHasUnseenAlerts(true);

    const locationText =
      alert.latitude !== undefined && alert.longitude !== undefined
        ? `\nLast known location: ${alert.latitude.toFixed(6)}, ${alert.longitude.toFixed(6)}`
        : '';

    Alert.alert(
      'SOS Alert',
      `${alert.reporterName || 'A trusted contact'} has started an SOS session.${locationText}`,
      [
        {
          text: 'Dismiss',
          style: 'cancel',
        },
        {
          text: 'View Location',
          onPress: () => navigateToAlert(alert),
        },
      ],
      { cancelable: false }
    );
  }, [navigateToAlert, upsertAlert]);

  const normalizeAlertFromNotification = useCallback((data: NotificationData): ActiveSOSAlert | null => {
    if (data.type !== 'sos_started' || !data.sos_session_id || !data.viewer_token) {
      return null;
    }

    return {
      sessionId: data.sos_session_id,
      reporterName: data.reporter_identifier || 'Trusted Contact',
      startedAt: data.started_at || new Date().toISOString(),
      viewerToken: data.viewer_token,
      latitude: normalizeCoordinate(data.lat),
      longitude: normalizeCoordinate(data.lng),
      recordedAt: data.recorded_at,
    };
  }, []);

  // Handle incoming notification while app is in foreground
  const handleNotificationReceived = useCallback(
    async (notification: Notification) => {
      const data = notification.request.content.data as NotificationData;
      const alert = normalizeAlertFromNotification(data);
      if (!alert) {
        return;
      }

      announceAlert(alert);
    },
    [announceAlert, normalizeAlertFromNotification]
  );

  // Handle notification tap (app opened from notification)
  const handleNotificationResponse = useCallback(
    (response: NotificationResponse) => {
      const data = response.notification.request.content.data as NotificationData;
      const alert = normalizeAlertFromNotification(data);
      if (!alert) {
        return;
      }

      announcedSessions.current.add(alert.sessionId);
      upsertAlert(alert);
      navigateToAlert(alert);
    },
    [navigateToAlert, normalizeAlertFromNotification, upsertAlert]
  );

  // Poll for active SOS sessions when app comes to foreground
  const checkForActiveSOSSessions = useCallback(async () => {
    try {
      const alerts = await getActiveSOSAlerts();
      for (const alert of alerts) {
        const nextAlert: ActiveSOSAlert = {
          sessionId: alert.session_id,
          reporterName: alert.reporter_name || alert.reporter_phone || 'Trusted Contact',
          startedAt: alert.started_at,
          viewerToken: alert.viewer_token,
          latitude: normalizeCoordinate(alert.lat),
          longitude: normalizeCoordinate(alert.lng),
          recordedAt: alert.recorded_at,
        };

        upsertAlert(nextAlert);
        announceAlert(nextAlert);
      }
    } catch (error) {
      console.error('Failed to check for active SOS sessions:', error);
    }
  }, [announceAlert, upsertAlert]);

  // Handle app state changes
  useEffect(() => {
    checkForActiveSOSSessions();

    const subscription = AppState.addEventListener('change', (nextAppState) => {
      if (appState.current.match(/inactive|background/) && nextAppState === 'active') {
        // App came to foreground, check for active sessions
        checkForActiveSOSSessions();
      }
      appState.current = nextAppState;
    });

    return () => {
      subscription.remove();
    };
  }, [checkForActiveSOSSessions]);

  useEffect(() => {
    const interval = setInterval(() => {
      if (appState.current === 'active') {
        checkForActiveSOSSessions();
      }
    }, 5000);

    return () => clearInterval(interval);
  }, [checkForActiveSOSSessions]);

  // Set up notification listeners
  useEffect(() => {
    notificationListener.current = addNotificationReceivedListener(
      handleNotificationReceived
    );
    responseListener.current = addNotificationResponseListener(handleNotificationResponse);

    return () => {
      notificationListener.current?.remove();
      responseListener.current?.remove();
    };
  }, [handleNotificationReceived, handleNotificationResponse]);

  // Function to mark alerts as seen
  const markAlertsAsSeen = useCallback(() => {
    setHasUnseenAlerts(false);
  }, []);

  // Function to remove an alert
  const dismissAlert = useCallback((sessionId: string) => {
    setActiveAlerts((prev) => prev.filter((a) => a.sessionId !== sessionId));
  }, []);

  // Function to clear all alerts
  const clearAllAlerts = useCallback(() => {
    setActiveAlerts([]);
    setHasUnseenAlerts(false);
  }, []);

  return {
    activeAlerts,
    hasUnseenAlerts,
    markAlertsAsSeen,
    dismissAlert,
    clearAllAlerts,
  };
}

function normalizeCoordinate(value: unknown): number | undefined {
  if (typeof value === 'number' && Number.isFinite(value)) {
    return value;
  }
  if (typeof value === 'string') {
    const parsed = Number.parseFloat(value);
    if (Number.isFinite(parsed)) {
      return parsed;
    }
  }

  return undefined;
}
