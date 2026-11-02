import { Stack } from 'expo-router';
import { useTheme } from '@/theme/ThemeProvider';

export const unstable_settings = { initialRouteName: 'welcome' };

export default function AuthLayout() {
  const t = useTheme();
  return (
    <Stack
      screenOptions={{ headerShown: false, animation: 'slide_from_right', contentStyle: { backgroundColor: t.bg[0] } }}
    />
  );
}
