import { router, Stack } from 'expo-router';
import * as SplashScreen from 'expo-splash-screen';
import { type ReactNode, useEffect, useRef } from 'react';
import { Platform, View } from 'react-native';
import { SafeAreaProvider } from 'react-native-safe-area-context';
import { AppUpdate } from '@/features/AppUpdate';
import '@/lib/referral';
import { SessionProvider, useSession } from '@/state/session';
import { ThemeProvider, useTheme } from '@/theme/ThemeProvider';
import { space } from '@/theme/tokens';
import { Backdrop } from '@/ui/layout';
import { Button } from '@/ui/primitives';
import { WEB_WIDTH } from '@/ui/screenSize';
import { SukiSays } from '@/ui/suki';
import { tr } from '@/i18n';

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
            title={tr("I can't reach the server")}
            body={tr('It might be your connection, or it might be us. Your account is safe. Try again in a moment.')}
          >
            <Button label={tr('Try again')} icon="refresh" onPress={retry} />
          </SukiSays>
        </View>
      </Backdrop>
    );
  }

  return (
    <>
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
      <AppUpdate />
    </>
  );
}

function WebColumn({ children }: { children: ReactNode }) {
  const t = useTheme();
  if (Platform.OS !== 'web') return children;
  return (
    <View style={{ flex: 1, backgroundColor: t.bg[1] }}>
      <View style={{ flex: 1, width: '100%', maxWidth: WEB_WIDTH, alignSelf: 'center', overflow: 'hidden' }}>{children}</View>
    </View>
  );
}

export default function RootLayout() {
  return (
    <SafeAreaProvider>
      <SessionProvider>
        <ThemeProvider>
          <WebColumn>
            <Routes />
          </WebColumn>
        </ThemeProvider>
      </SessionProvider>
    </SafeAreaProvider>
  );
}
