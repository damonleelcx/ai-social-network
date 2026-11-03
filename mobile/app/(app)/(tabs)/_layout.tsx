import { Ionicons } from '@expo/vector-icons';
import { LinearGradient } from 'expo-linear-gradient';
import { Tabs } from 'expo-router';
import { View, type ColorValue } from 'react-native';
import { useLive } from '@/state/live';
import { useTheme } from '@/theme/ThemeProvider';
import { IrisRule, type IconName } from '@/ui/primitives';
import { SukiAvatar } from '@/ui/suki';

function TabIcon({ name, focused, color }: { name: IconName; focused: boolean; color: ColorValue }) {
  const t = useTheme();
  if (!focused) return <Ionicons name={name} size={22} color={color as string} />;
  return (
    <LinearGradient
      colors={t.brand}
      start={{ x: 0, y: 0 }}
      end={{ x: 1, y: 1 }}
      style={{ width: 46, height: 30, borderRadius: 15, alignItems: 'center', justifyContent: 'center' }}
    >
      <Ionicons name={name} size={18} color={t.onBrand} />
    </LinearGradient>
  );
}

export const unstable_settings = { initialRouteName: 'suki' };

export default function TabsLayout() {
  const t = useTheme();
  const { badges } = useLive();
  const badgeStyle = { backgroundColor: t.brand[2], color: t.onBrand, fontSize: 10, fontWeight: '800' as const };
  return (
    <Tabs
      screenOptions={{
        headerShown: false,
        tabBarHideOnKeyboard: true,
        tabBarActiveTintColor: t.accent,
        tabBarInactiveTintColor: t.inkFaint,
        tabBarBadgeStyle: badgeStyle,
        tabBarLabelStyle: { fontSize: 11, fontWeight: '700' },
        tabBarStyle: { backgroundColor: t.glassStrong, borderTopWidth: 0, elevation: 0 },
        tabBarBackground: () => (
          <View style={{ flex: 1, backgroundColor: t.glassStrong }}>
            <IrisRule style={{ opacity: 0.6 }} />
          </View>
        ),
        sceneStyle: { backgroundColor: t.bg[0] },
      }}
    >
      <Tabs.Screen
        name="suki"
        options={{
          title: 'Suki',
          tabBarAccessibilityLabel: 'Suki',
          tabBarBadge: badges.suki > 0 ? badges.suki : undefined,
          tabBarIcon: ({ focused }) => <SukiAvatar size={focused ? 32 : 28} mood="portrait" />,
        }}
      />
      <Tabs.Screen
        name="matches"
        options={{
          title: 'Matches',
          tabBarBadge: badges.matches > 0 ? badges.matches : undefined,
          tabBarIcon: ({ focused, color }) => <TabIcon name={focused ? 'heart' : 'heart-outline'} focused={focused} color={color} />,
        }}
      />
      <Tabs.Screen
        name="dates"
        options={{
          title: 'Dates',
          tabBarBadge: badges.dates > 0 ? badges.dates : undefined,
          tabBarIcon: ({ focused, color }) => <TabIcon name={focused ? 'calendar' : 'calendar-outline'} focused={focused} color={color} />,
        }}
      />
      <Tabs.Screen
        name="me"
        options={{
          title: 'Me',
          tabBarBadge: badges.me > 0 ? badges.me : undefined,
          tabBarIcon: ({ focused, color }) => <TabIcon name={focused ? 'person' : 'person-outline'} focused={focused} color={color} />,
        }}
      />
    </Tabs>
  );
}
