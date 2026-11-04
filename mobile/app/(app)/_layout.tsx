import * as Notifications from 'expo-notifications';
import { Platform } from 'react-native';
import { Stack } from 'expo-router';
import { useEffect } from 'react';
import type { NotificationData } from '@/api/types';
import { openNotificationTarget } from '@/lib/links';
import { registerForPush } from '@/lib/push';
import { LiveProvider } from '@/state/live';
import { useTheme } from '@/theme/ThemeProvider';

const asData = (r: Notifications.NotificationResponse) =>
  r.notification.request.content.data as unknown as NotificationData | undefined;

function useNotificationLinks() {
  useEffect(() => {
    if (Platform.OS === 'web') return;
    registerForPush();

    // A tap that launched the app: wait a moment for the tabs to be in place.
    let launched: ReturnType<typeof setTimeout> | undefined;
    try {
      const last = Notifications.getLastNotificationResponse();
      if (last) {
        launched = setTimeout(() => {
          openNotificationTarget(asData(last));
          Notifications.clearLastNotificationResponse();
        }, 350);
      }
    } catch {
      // Notifications aren't available on this build.
    }

    const sub = Notifications.addNotificationResponseReceivedListener((r) => openNotificationTarget(asData(r)));
    return () => {
      if (launched) clearTimeout(launched);
      sub.remove();
    };
  }, []);
}

export const unstable_settings = { initialRouteName: '(tabs)' };

export default function AppLayout() {
  const t = useTheme();
  useNotificationLinks();
  return (
    <LiveProvider>
      <Stack
        screenOptions={{ headerShown: false, animation: 'slide_from_right', contentStyle: { backgroundColor: t.bg[0] } }}
      />
    </LiveProvider>
  );
}
