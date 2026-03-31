import { useEffect, useState, useCallback, useRef } from 'react';
import { Alert, AppState } from 'react-native';
import { router } from 'expo-router';
import * as Notifications from 'expo-notifications';
import {
  addNotificationReceivedListener,
  addNotificationResponseListener,
  NotificationData,
  scheduleLocalNotification,
} from '../services/notifications';
import { getSOSSession } from '../services/sos';

interface ActiveSOSAlert {
  sessionId: string;
  reporterName: string;
  startedAt: string;
  viewerToken?: string;
}

export function useSOSNotifications() {
  const [activeAlerts, setActiveAlerts] = useState<ActiveSOSAlert[]>([]);
  const [hasUnseenAlerts, setHasUnseenAlerts] = useState(false);
  const notificationListener = useRef<Notifications.Subscription>();
  const responseListener = useRef<Notifications.Subscription>();
  const appState = useRef(AppState.currentState);

  // Handle incoming notification while app is in foreground
  const handleNotificationReceived = useCallback(
    async (notification: Notifications.Notification) => {
      const data = notification.request.content.data as NotificationData;

      if (data.type === 'sos_started' && data.sos_session_id) {
        // Add to active alerts
        const newAlert: ActiveSOSAlert = {
          sessionId: data.sos_session_id,
          reporterName: data.reporter_identifier || 'Unknown',
          startedAt: data.started_at || new Date().toISOString(),
          viewerToken: data.viewer_token,
        };

        setActiveAlerts((prev) => {
          // Don't add duplicates
          if (prev.some((a) => a.sessionId === newAlert.sessionId)) {
            return prev;
          }
          return [...prev, newAlert];
        });
        setHasUnseenAlerts(true);

        // Show in-app alert
        Alert.alert(
          '🚨 SOS Alert',
          `${data.reporter_identifier || 'A trusted contact'} has started an SOS session!`,
          [
            {
              text: 'Dismiss',
              style: 'cancel',
            },
            {
              text: 'View Location',
              onPress: () => {
                router.push({
                  pathname: '/sos-viewer',
                  params: {
                    sessionId: data.sos_session_id,
                    viewerToken: data.viewer_token || '',
                  },
                });
              },
            },
          ],
          { cancelable: false }
        );
      }
    },
    []
  );

  // Handle notification tap (app opened from notification)
  const handleNotificationResponse = useCallback(
    (response: Notifications.NotificationResponse) => {
      const data = response.notification.request.content.data as NotificationData;

      if (data.type === 'sos_started' && data.sos_session_id) {
        // Navigate to SOS viewer
        router.push({
          pathname: '/sos-viewer',
          params: {
            sessionId: data.sos_session_id,
            viewerToken: data.viewer_token || '',
          },
        });
      }
    },
    []
  );

  // Poll for active SOS sessions when app comes to foreground
  const checkForActiveSOSSessions = useCallback(async () => {
    try {
      // This would require a new backend endpoint to list active SOS sessions for viewer
      // For now, we rely on push notifications
    } catch (error) {
      console.error('Failed to check for active SOS sessions:', error);
    }
  }, []);

  // Handle app state changes
  useEffect(() => {
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
