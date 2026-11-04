import { router } from 'expo-router';
import { useCallback, useEffect, useState } from 'react';
import { FlatList, Pressable, RefreshControl, View } from 'react-native';
import * as api from '@/api/endpoints';
import type { BuddyRequest, DatePlan } from '@/api/types';
import { MyBuddyPost } from '@/features/BuddyCards';
import { longDateTime } from '@/lib/format';
import { dateNeedsMe, useLive, useTopic } from '@/state/live';
import { useTheme } from '@/theme/ThemeProvider';
import { space } from '@/theme/tokens';
import { Backdrop, TabHeader } from '@/ui/layout';
import { PersonAvatar } from '@/ui/person';
import { Glass, Pill, T } from '@/ui/primitives';
import { SukiLoading, SukiSays } from '@/ui/suki';
import { tr } from '@/i18n';

const OPEN = ['searching', 'voting', 'scheduling', 'confirmed'];

export function dateLine(d: DatePlan) {
  switch (d.status) {
    case 'searching':
      return tr("I'm looking for a place you'd both enjoy.");
    case 'voting':
      return d.my_vote ? tr('Waiting for {name} to pick.', { name: d.person.display_name }) : tr('Pick your favorite place.');
    case 'scheduling':
      if (!d.proposed_at) return tr('{name}. When works for you both?', { name: d.venue?.name ?? tr('Place chosen') });
      return d.proposed_by_me
        ? tr('You suggested {time}.', { time: longDateTime(d.proposed_at) })
        : tr('{name} suggested {time}.', { name: d.person.display_name, time: longDateTime(d.proposed_at) });
    case 'confirmed':
      return `${d.venue?.name ?? ''}, ${longDateTime(d.scheduled_at)}`;
    case 'completed':
      return tr('You met at {name}.', { name: d.venue?.name ?? tr('your spot') });
    case 'cancelled':
      return tr('This one was called off.');
    case 'failed':
      return tr("I couldn't find a good place for this one.");
    default:
      return '';
  }
}

function DateRow({ date }: { date: DatePlan }) {
  const needsMe = dateNeedsMe(date);
  const tone = date.status === 'confirmed' ? 'good' : OPEN.includes(date.status) ? 'accent' : 'plain';
  const statusLabel: Record<string, string> = {
    searching: tr('Looking'),
    voting: tr('Choosing a place'),
    scheduling: tr('Finding a time'),
    confirmed: tr('On!'),
    completed: tr('Done'),
    cancelled: tr('Cancelled'),
    failed: tr('Fell through'),
  };
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={tr('Date with {name}', { name: date.person.display_name })}
      onPress={() => router.push(`/date/${date.id}`)}
      style={({ pressed }) => ({ opacity: pressed ? 0.7 : 1 })}
    >
      <Glass style={{ flexDirection: 'row', gap: space.md, alignItems: 'center' }}>
        <PersonAvatar id={date.person.photo_ids?.[0]} size={60} />
        <View style={{ flex: 1, gap: 4 }}>
          <T v="subheading">{date.person.display_name}</T>
          <T v="small" soft>
            {dateLine(date)}
          </T>
          <View style={{ flexDirection: 'row', gap: 6, marginTop: 2 }}>
            <Pill tone={tone} label={statusLabel[date.status] ?? date.status} />
            {needsMe ? <Pill tone="warn" label={tr('Your turn')} /> : null}
          </View>
        </View>
      </Glass>
    </Pressable>
  );
}

export default function Dates() {
  const t = useTheme();
  const { dates, loaded, refreshDates } = useLive();
  const [refreshing, setRefreshing] = useState(false);
  const [posts, setPosts] = useState<BuddyRequest[]>([]);

  // Their own buddy posts: open ones, and filled ones until the plan has passed.
  const loadPosts = useCallback(async () => {
    try {
      const mine = await api.buddies.mine();
      setPosts(mine.filter((p) => p.status === 'open' || (p.status === 'filled' && new Date(p.expires_at).getTime() > Date.now())));
    } catch {
      // Pull to refresh tries again; what's on screen stays.
    }
  }, []);
  useEffect(() => {
    loadPosts();
  }, [loadPosts]);
  useTopic('buddies', loadPosts);

  return (
    <Backdrop>
      <TabHeader title={tr('Dates')} subtitle={tr('I handle the logistics. You bring the charm.')} />
      <FlatList
        data={dates}
        keyExtractor={(d) => d.id}
        contentContainerStyle={{ paddingHorizontal: space.lg, paddingBottom: space.xxl, gap: space.md, flexGrow: 1 }}
        refreshControl={
          <RefreshControl
            refreshing={refreshing}
            tintColor={t.accent}
            colors={[t.accent]}
            onRefresh={async () => {
              setRefreshing(true);
              await Promise.all([refreshDates(), loadPosts()]);
              setRefreshing(false);
            }}
          />
        }
        ListHeaderComponent={
          posts.length ? (
            <View style={{ gap: space.md }}>
              <T v="subheading">{tr('Your buddy posts')}</T>
              {posts.map((p) => (
                <MyBuddyPost key={p.id} request={p} onChange={loadPosts} />
              ))}
            </View>
          ) : undefined
        }
        ListEmptyComponent={
          posts.length ? undefined : !loaded.dates ? (
            <SukiLoading line={tr('Checking your calendar...')} />
          ) : (
            <SukiSays
              face="playful"
              size={132}
              title={tr('No dates yet')}
              body={tr("When a chat feels ready, tap Plan a date, or just ask me. I'll find a spot you'd both like.")}
            />
          )
        }
        renderItem={({ item }) => <DateRow date={item} />}
      />
    </Backdrop>
  );
}
