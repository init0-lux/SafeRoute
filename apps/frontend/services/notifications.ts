import Constants from 'expo-constants';
import type {
  Notification as ExpoNotification,
  NotificationResponse as ExpoNotificationResponse,
} from 'expo-notifications';
import { Platform } from 'react-native';
import { post } from './api';

type NotificationsModule = typeof import('expo-notifications');
type DeviceModule = typeof import('expo-device');

export type Notification = ExpoNotification;
export type NotificationResponse = ExpoNotificationResponse;

export interface NotificationSubscription {
  remove: () => void;
}

const noopSubscription: NotificationSubscription = {
  remove: () => {},
};

let notificationsModule: NotificationsModule | null | undefined;
let deviceModule: DeviceModule | null | undefined;
let notificationHandlerConfigured = false;
let availabilityWarningShown = false;

export interface NotificationData {
  type?: string;
  sos_session_id?: string;
  viewer_token?: string;
  viewer_url?: string;
  reporter_identifier?: string;
  started_at?: string;
  lat?: number;
  lng?: number;
  recorded_at?: string;
}

function getExpoProjectId(): string | null {
  return (
    process.env.EXPO_PUBLIC_PROJECT_ID ||
    Constants.easConfig?.projectId ||
    Constants.expoConfig?.extra?.eas?.projectId ||
    null
  );
}

function getNotificationsModule(): NotificationsModule | null {
  if (notificationsModule !== undefined) {
    return notificationsModule;
  }

  try {
    notificationsModule = require('expo-notifications') as NotificationsModule;
  } catch (error) {
    notificationsModule = null;
    logNotificationsUnavailable(error);
  }

  return notificationsModule;
}

function getDeviceModule(): DeviceModule | null {
  if (deviceModule !== undefined) {
    return deviceModule;
  }

  try {
    deviceModule = require('expo-device') as DeviceModule;
  } catch (error) {
    deviceModule = null;
    logNotificationsUnavailable(error);
  }

  return deviceModule;
}

function ensureNotificationHandlerConfigured(): NotificationsModule | null {
  const Notifications = getNotificationsModule();
  if (!Notifications) {
    return null;
  }

  if (!notificationHandlerConfigured) {
    Notifications.setNotificationHandler({
      handleNotification: async () => ({
        shouldShowAlert: true,
        shouldShowBanner: true,
        shouldShowList: true,
        shouldPlaySound: true,
        shouldSetBadge: true,
      }),
    });
    notificationHandlerConfigured = true;
  }

  return Notifications;
}

function logNotificationsUnavailable(error?: unknown) {
  if (availabilityWarningShown) {
    return;
  }

  availabilityWarningShown = true;
  console.warn(
    'expo-notifications is unavailable in this client build. Rebuild the Android dev client to enable push notifications.',
    error
  );
}

export async function registerForPushNotifications(): Promise<string | null> {
  const Notifications = ensureNotificationHandlerConfigured();
  const Device = getDeviceModule();
  if (!Notifications || !Device) {
    return null;
  }

  if (!Device.isDevice) {
    console.log('Push notifications require a physical device');
    return null;
  }

  // Check existing permissions
  const { status: existingStatus } = await Notifications.getPermissionsAsync();
  let finalStatus = existingStatus;

  // Request permissions if not granted
  if (existingStatus !== 'granted') {
    const { status } = await Notifications.requestPermissionsAsync();
    finalStatus = status;
  }

  if (finalStatus !== 'granted') {
    console.log('Permission not granted for push notifications');
    return null;
  }

  try {
    const projectId = getExpoProjectId();
    if (!projectId) {
      console.log('Expo project ID is missing; set EXPO_PUBLIC_PROJECT_ID to enable push notifications');
      return null;
    }

    const token = (
      await Notifications.getExpoPushTokenAsync({
        projectId,
      })
    ).data;

    // Configure Android notification channel
    if (Platform.OS === 'android') {
      await Notifications.setNotificationChannelAsync('default', {
        name: 'default',
        importance: Notifications.AndroidImportance.MAX,
        vibrationPattern: [0, 250, 250, 250],
        lightColor: '#FF231F7C',
      });

      // SOS-specific channel
      await Notifications.setNotificationChannelAsync('sos', {
        name: 'SOS Alerts',
        importance: Notifications.AndroidImportance.MAX,
        vibrationPattern: [0, 500, 250, 500],
        lightColor: '#FF0000',
        sound: 'default',
      });
    }

    return token;
  } catch (error) {
    console.error('Failed to get push token:', error);
    return null;
  }
}

export async function updatePushToken(token: string): Promise<void> {
  try {
    await post('/auth/push-token', { token });
  } catch (error) {
    console.error('Failed to update push token on server:', error);
  }
}

export function addNotificationReceivedListener(
  callback: (notification: Notification) => void
): NotificationSubscription {
  const Notifications = ensureNotificationHandlerConfigured();
  if (!Notifications) {
    return noopSubscription;
  }

  return Notifications.addNotificationReceivedListener(callback);
}

export function addNotificationResponseListener(
  callback: (response: NotificationResponse) => void
): NotificationSubscription {
  const Notifications = ensureNotificationHandlerConfigured();
  if (!Notifications) {
    return noopSubscription;
  }

  return Notifications.addNotificationResponseReceivedListener(callback);
}

export async function scheduleLocalNotification(
  title: string,
  body: string,
  data?: NotificationData
): Promise<string> {
  const Notifications = ensureNotificationHandlerConfigured();
  if (!Notifications) {
    return '';
  }

  return await Notifications.scheduleNotificationAsync({
    content: {
      title,
      body,
      data: (data || {}) as Record<string, unknown>,
      sound: 'default',
      priority: Notifications.AndroidNotificationPriority.HIGH,
    },
    trigger: null, // Show immediately
  });
}

export async function dismissAllNotifications(): Promise<void> {
  const Notifications = ensureNotificationHandlerConfigured();
  if (!Notifications) {
    return;
  }

  await Notifications.dismissAllNotificationsAsync();
}

export async function getBadgeCount(): Promise<number> {
  const Notifications = ensureNotificationHandlerConfigured();
  if (!Notifications) {
    return 0;
  }

  return await Notifications.getBadgeCountAsync();
}

export async function setBadgeCount(count: number): Promise<void> {
  const Notifications = ensureNotificationHandlerConfigured();
  if (!Notifications) {
    return;
  }

  await Notifications.setBadgeCountAsync(count);
}
