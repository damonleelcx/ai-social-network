import { useCallback, useState } from 'react';
import { FlatList, RefreshControl, View } from 'react-native';
import { useFocusEffect } from 'expo-router';
import * as api from '@/api/endpoints';
import type { BuddyRequest } from '@/api/types';
import { NearbyPlan, meetSafely } from '@/features/BuddyCards';
import { missingLabel } from '@/lib/format';
import { tr } from '@/i18n';
import { useTheme } from '@/theme/ThemeProvider';
import { space } from '@/theme/tokens';
import { Backdrop, ScreenHeader } from '@/ui/layout';
import { Pill, T } from '@/ui/primitives';
import { SukiLoading, SukiSays } from '@/ui/suki';

export default function NearbyBuddies() {
  const t = useTheme();
  const [posts, setPosts] = useState<BuddyRequest[]>([]);
  const [missing, setMissing] = useState<string[]>([]);
  const [loaded, setLoaded] = useState(false);
  const [refreshing, setRefreshing] = useState(false);

  const load = useCallback(async () => {
    try {
      const board = await api.buddies.board();
      setPosts(board.requests);
      setMissing(board.missing);
      setLoaded(true);
    } catch {
      setLoaded(true);
    }
  }, []);

  useFocusEffect(
    useCallback(() => {
      load();
    }, [load]),
  );

  return (
    <Backdrop>
      <ScreenHeader title={tr('Nearby plans')} />
      <FlatList
        data={missing.length ? [] : posts}
        keyExtractor={(p) => p.id}
        contentContainerStyle={{ padding: space.lg, paddingBottom: space.xxl, gap: space.md, flexGrow: 1 }}
        refreshControl={
          <RefreshControl
            refreshing={refreshing}
            tintColor={t.accent}
            colors={[t.accent]}
            onRefresh={async () => {
              setRefreshing(true);
              await load();
              setRefreshing(false);
            }}
          />
        }
        ListHeaderComponent={
          missing.length ? undefined : (
            <T v="small" soft>
              {meetSafely()}
            </T>
          )
        }
        ListEmptyComponent={
          !loaded ? (
            <SukiLoading line={tr('Looking nearby...')} />
          ) : missing.length ? (
            <View style={{ gap: space.md }}>
              <SukiSays face="thinking" size={132} title={tr('Finish your profile first')} body={tr('Add your name, birthday, city and a photo first, so people know who\'s coming.')} />
              <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 6 }}>
                {missing.map((m) => (
                  <Pill key={m} label={missingLabel(m)} />
                ))}
              </View>
            </View>
          ) : (
            <SukiSays face="playful" size={132} title={tr('Nothing nearby yet')} body={tr('When someone nearby posts a plan, it shows up here.')} />
          )
        }
        renderItem={({ item }) => <NearbyPlan request={item} onChange={load} />}
      />
    </Backdrop>
  );
}
