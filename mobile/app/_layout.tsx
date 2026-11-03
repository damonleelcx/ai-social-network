import { router, Stack } from 'expo-router';
import * as SplashScreen from 'expo-splash-screen';
import { useEffect, useRef } from 'react';
import { View } from 'react-native';
import { SafeAreaProvider } from 'react-native-safe-area-context';
import { SessionProvider, useSession } from '@/state/session';
import { ThemeProvider, useTheme } from '@/theme/ThemeProvider';
import { space } from '@/theme/tokens';
import { Backdrop } from '@/ui/layout';
import { Button } from '@/ui/primitives';
import { SukiSays } from '@/ui/suki';

SplashScreen.preventAutoHideAsync().catch(() => {});

function Routes() {
  const { status, retry } = useSession();
  const t = useTheme();

  useEffect(() => {
    if (status !== 'loading') SplashScreen.hideAsync().catch(() => {});
  }, [status]);

  // The auth stack remembers where it was, so signing out would reopen "Welcome back".
  const previous = useRef(status);
  useEffect(() => {
    const was = previous.current;
    previous.current = status;
    if (status === 'signedOut' && was !== 'signedOut' && was !== 'loading') {
      const id = setTimeout(() => router.dismissTo('/welcome'), 0);
      return () => clearTimeout(id);
    }
  }, [status]);

  if (status === 'loading') return null;

  if (status === 'unreachable') {
    return (
      <Backdrop>
        <View style={{ flex: 1, justifyContent: 'center', padding: space.xl }}>
          <SukiSays
            face="pouty"
            size={132}
            title="I can't reach the server"
            body="It might be your connection, or it might be us. Your account is safe. Try again in a moment."
          >
            <Button label="Try again" icon="refresh" onPress={retry} />
          </SukiSays>
        </View>
      </Backdrop>
    );
  }

  return (
    <Stack screenOptions={{ headerShown: false, contentStyle: { backgroundColor: t.bg[0] } }}>
      <Stack.Screen name="index" />
      <Stack.Protected guard={status === 'signedOut'}>
        <Stack.Screen name="(auth)" />
      </Stack.Protected>
      <Stack.Protected guard={status === 'unverified'}>
        <Stack.Screen name="verify" />
      </Stack.Protected>
      <Stack.Protected guard={status === 'ready'}>
        <Stack.Screen name="(app)" />
      </Stack.Protected>
    </Stack>
  );
}

export default function RootLayout() {
  return (
    <SafeAreaProvider>
      <SessionProvider>
        <ThemeProvider>
          <Routes />
        </ThemeProvider>
      </SessionProvider>
    </SafeAreaProvider>
  );
}
