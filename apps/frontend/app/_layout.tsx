import { Stack } from 'expo-router';
import { StatusBar } from 'expo-status-bar';
import { View, ActivityIndicator, StyleSheet } from 'react-native';
import { useEffect } from 'react';
import { AuthProvider, useAuth } from '@/context/AuthContext';
import {
  registerForPushNotifications,
  updatePushToken,
} from '@/services/notifications';
import { useSOSNotifications } from '@/hooks/useSOSNotifications';

function RootNavigator() {
  const { isLoading, isAuthenticated } = useAuth();
  const { activeAlerts, hasUnseenAlerts } = useSOSNotifications();

  // Register for push notifications when authenticated
  useEffect(() => {
    if (isAuthenticated) {
      registerForPushNotifications().then((token) => {
        if (token) {
          updatePushToken(token).catch((err) => {
            console.error('Failed to register push token:', err);
          });
        }
      });
    }
  }, [isAuthenticated]);

  if (isLoading) {
    return (
      <View style={styles.loading}>
        <ActivityIndicator size="large" color="#C9A0DC" />
      </View>
    );
  }

  return (
    <Stack
      screenOptions={{
        headerShown: false,
        contentStyle: { backgroundColor: '#000000' },
        animation: 'fade',
      }}
    >
      <Stack.Screen name="index" />
      <Stack.Screen name="login" />
      <Stack.Screen
        name="register"
        options={{
          animation: 'slide_from_right',
        }}
      />
      <Stack.Screen name="home" />
      <Stack.Screen
        name="history"
        options={{
          animation: 'slide_from_right',
        }}
      />
    </Stack>
  );
}

export default function RootLayout() {
  return (
    <AuthProvider>
      <StatusBar style="light" />
      <RootNavigator />
    </AuthProvider>
  );
}

const styles = StyleSheet.create({
  loading: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    backgroundColor: '#000000',
  },
});
