import { Ionicons } from '@expo/vector-icons';
import { useEffect, useState } from 'react';
import { FlatList, Pressable, RefreshControl, View } from 'react-native';
import * as api from '@/api/endpoints';
import type { AppNotification } from '@/api/types';
import { shortTime } from '@/lib/format';
import { openNotificationTarget } from '@/lib/links';
import { useLive } from '@/state/live';
import { useTheme } from '@/theme/ThemeProvider';
import { space } from '@/theme/tokens';
import { Backdrop, ScreenHeader } from '@/ui/layout';
import { Glass, T, type IconName } from '@/ui/primitives';
import { SukiLoading, SukiSays } from '@/ui/suki';

function iconFor(n: AppNotification): IconName {
  switch (n.data?.screen) {
    case 'proposal':
      return 'sparkles';
    case 'chat':
      return 'chatbubble-ellipses';
    case 'date':
      return 'calendar';
    default:
      return 'heart';
  }
}

export default function Notifications() {
  const t = useTheme();
  const { notifications, loaded, refreshNotifications } = useLive();
  const [refreshing, setRefreshing] = useState(false);
  // Opening the list reads it, but the ones that were new stay marked for this visit.
  const [freshIds, setFreshIds] = useState<Set<number> | null>(null);

  useEffect(() => {
    if (!loaded.notifications || freshIds) return;
    setFreshIds(new Set(notifications.filter((n) => !n.read).map((n) => n.id)));
    if (notifications.some((n) => !n.read)) {
      api.notifications
        .markRead()
        .then(refreshNotifications)
        .catch(() => {});
    }
  }, [freshIds, loaded.notifications, notifications, refreshNotifications]);

  return (
    <Backdrop>
      <ScreenHeader title="Notifications" />
      <FlatList
        data={notifications}
        keyExtractor={(n) => String(n.id)}
        contentContainerStyle={{ padding: space.lg, gap: space.sm, flexGrow: 1 }}
        refreshControl={
          <RefreshControl
            refreshing={refreshing}
            tintColor={t.accent}
            colors={[t.accent]}
            onRefresh={async () => {
              setRefreshing(true);
              await refreshNotifications();
              setRefreshing(false);
            }}
          />
        }
        ListEmptyComponent={
          !loaded.notifications ? (
            <SukiLoading line="Checking..." />
          ) : (
            <SukiSays face="happy" title="All quiet" body="When something happens, you'll hear it from me first." />
          )
        }
        renderItem={({ item }) => {
          const fresh = !item.read || freshIds?.has(item.id);
          return (
            <Pressable
              accessibilityRole="button"
              accessibilityLabel={`${item.title}. ${item.body}`}
              onPress={() => openNotificationTarget(item.data)}
              style={({ pressed }) => ({ opacity: pressed ? 0.7 : 1 })}
            >
              <Glass style={[{ flexDirection: 'row', gap: space.md, padding: space.md }, fresh && { backgroundColor: t.cardTint }]}>
                <View style={{ width: 38, height: 38, borderRadius: 19, alignItems: 'center', justifyContent: 'center', backgroundColor: t.accentSoft }}>
                  <Ionicons name={iconFor(item)} size={18} color={t.accent} />
                </View>
                <View style={{ flex: 1, gap: 2 }}>
                  <View style={{ flexDirection: 'row', gap: space.sm }}>
                    <T v="small" style={{ flex: 1, fontWeight: '700' }}>
                      {item.title}
                    </T>
                    <T v="caption" faint>
                      {shortTime(item.created_at)}
                    </T>
                  </View>
                  <T v="small" soft>
                    {item.body}
                  </T>
                </View>
              </Glass>
            </Pressable>
          );
        }}
      />
    </Backdrop>
  );
}
