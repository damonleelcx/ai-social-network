import Constants from 'expo-constants';
import * as Notifications from 'expo-notifications';
import { Platform } from 'react-native';
import { notifications as api } from '@/api/endpoints';

Notifications.setNotificationHandler({
  handleNotification: async () => ({
    shouldShowBanner: true,
    shouldShowList: true,
    shouldPlaySound: false,
    shouldSetBadge: false,
  }),
});

/**
 * Registers this device for push. It needs an EAS project to get a token at all, so
 * without one, or on anything else going wrong (emulator, no Google services, the
 * person says no), it does nothing. In-app notifications still work over the socket.
 */
export async function registerForPush() {
  try {
    const projectId =
      (Constants.expoConfig?.extra as { eas?: { projectId?: string } } | undefined)?.eas?.projectId ??
      Constants.easConfig?.projectId;
    if (!projectId) return;

    if (Platform.OS === 'android') {
      await Notifications.setNotificationChannelAsync('default', {
        name: 'SynAgent',
        importance: Notifications.AndroidImportance.DEFAULT,
      });
    }
    let { status } = await Notifications.getPermissionsAsync();
    if (status !== 'granted') {
      status = (await Notifications.requestPermissionsAsync()).status;
    }
    if (status !== 'granted') return;

    const { data } = await Notifications.getExpoPushTokenAsync({ projectId });
    await api.registerPush(data, Platform.OS);
  } catch {
    // Push is a nicety. Never let it get in the way.
  }
}
