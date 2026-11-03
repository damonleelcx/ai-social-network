import { router } from 'expo-router';
import { useState } from 'react';
import { FlatList, Pressable, RefreshControl, View } from 'react-native';
import type { DatePlan } from '@/api/types';
import { longDateTime } from '@/lib/format';
import { dateNeedsMe, useLive } from '@/state/live';
import { useTheme } from '@/theme/ThemeProvider';
import { space } from '@/theme/tokens';
import { Backdrop, TabHeader } from '@/ui/layout';
import { PersonAvatar } from '@/ui/person';
import { Glass, Pill, T } from '@/ui/primitives';
import { SukiLoading, SukiSays } from '@/ui/suki';

const OPEN = ['searching', 'voting', 'scheduling', 'confirmed'];

export function dateLine(d: DatePlan) {
  switch (d.status) {
    case 'searching':
      return "I'm looking for a place you'd both enjoy.";
    case 'voting':
      return d.my_vote ? `Waiting for ${d.person.display_name} to pick.` : 'Pick your favorite place.';
    case 'scheduling':
      if (!d.proposed_at) return `${d.venue?.name ?? 'Place chosen'}. When works for you both?`;
      return d.proposed_by_me
        ? `You suggested ${longDateTime(d.proposed_at)}.`
        : `${d.person.display_name} suggested ${longDateTime(d.proposed_at)}.`;
    case 'confirmed':
      return `${d.venue?.name ?? ''}, ${longDateTime(d.scheduled_at)}`;
    case 'completed':
      return `You met at ${d.venue?.name ?? 'your spot'}.`;
    case 'cancelled':
      return 'This one was called off.';
    case 'failed':
      return "I couldn't find a good place for this one.";
    default:
      return '';
  }
}

function DateRow({ date }: { date: DatePlan }) {
  const needsMe = dateNeedsMe(date);
  const tone = date.status === 'confirmed' ? 'good' : OPEN.includes(date.status) ? 'accent' : 'plain';
  const statusLabel: Record<string, string> = {
    searching: 'Looking',
    voting: 'Choosing a place',
    scheduling: 'Finding a time',
    confirmed: 'On!',
    completed: 'Done',
    cancelled: 'Cancelled',
    failed: 'Fell through',
  };
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={`Date with ${date.person.display_name}`}
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
            {needsMe ? <Pill tone="warn" label="Your turn" /> : null}
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

  return (
    <Backdrop>
      <TabHeader title="Dates" subtitle="I handle the logistics. You bring the charm." />
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
              await refreshDates();
              setRefreshing(false);
            }}
          />
        }
        ListEmptyComponent={
          !loaded.dates ? (
            <SukiLoading line="Checking your calendar..." />
          ) : (
            <SukiSays
              face="playful"
              size={132}
              title="No dates yet"
              body="When a chat feels ready, tap Plan a date, or just ask me. I'll find a spot you'd both like."
            />
          )
        }
        renderItem={({ item }) => <DateRow date={item} />}
      />
    </Backdrop>
  );
}
