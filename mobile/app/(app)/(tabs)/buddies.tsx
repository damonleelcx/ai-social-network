import { router } from 'expo-router';
import { useCallback, useEffect, useState } from 'react';
import { FlatList, RefreshControl, View } from 'react-native';
import { Alert } from '@/lib/dialogs';
import * as api from '@/api/endpoints';
import type { BuddyRequest, Person } from '@/api/types';
import { BuddyPostSheet, buddyKindLabel } from '@/features/BuddyPostSheet';
import { ReportSheet } from '@/features/ReportSheet';
import { useLive, useTopic } from '@/state/live';
import { useTheme } from '@/theme/ThemeProvider';
import { space } from '@/theme/tokens';
import { Backdrop, TabHeader } from '@/ui/layout';
import { PersonAvatar } from '@/ui/person';
import { Button, Glass, IconButton, Pill, T } from '@/ui/primitives';
import { confirm, Sheet, SheetAction, showError } from '@/ui/sheet';
import { SukiLoading, SukiSays } from '@/ui/suki';
import { tr } from '@/i18n';

const MEET_SAFELY = () => tr('For a first meet-up, pick somewhere public and tell a friend. Never send money to someone you met here.');

function PersonLine({ person, size = 48 }: { person: Person; size?: number }) {
  return (
    <View style={{ flexDirection: 'row', alignItems: 'center', gap: space.md, flex: 1 }}>
      <PersonAvatar id={person.photo_ids?.[0]} size={size} />
      <View style={{ flex: 1 }}>
        <T v="subheading">{person.display_name}</T>
        <T v="small" soft>
          {person.age ? `${person.age}` : ''}
          {person.age && person.occupation ? ' · ' : ''}
          {person.occupation}
        </T>
      </View>
    </View>
  );
}

/** What the plan is. Someone's note is shown as written. */
function Plan({ request }: { request: BuddyRequest }) {
  return (
    <View style={{ gap: 6 }}>
      <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 6, alignItems: 'center' }}>
        <Pill tone="accent" label={buddyKindLabel(request.kind)} />
        <T v="small" style={{ fontWeight: '700' }}>
          {request.when}
        </T>
      </View>
      {request.note ? <T>{request.note}</T> : null}
    </View>
  );
}

function BoardRow({ request, onChange, onMore }: { request: BuddyRequest; onChange: () => void; onMore: (p: Person) => void }) {
  const [busy, setBusy] = useState(false);
  const person = request.person!;
  const toggle = async () => {
    if (!request.offered && !(await confirm(tr('Join {name}?', { name: person.display_name }), MEET_SAFELY(), tr("I'm in")))) return;
    setBusy(true);
    try {
      if (request.offered) await api.buddies.withdraw(request.id);
      else await api.buddies.offer(request.id);
      onChange();
    } catch (e) {
      showError(e);
    } finally {
      setBusy(false);
    }
  };
  return (
    <Glass style={{ gap: space.md }}>
      <View style={{ flexDirection: 'row', alignItems: 'center' }}>
        <PersonLine person={person} />
        <IconButton icon="ellipsis-horizontal" label={tr('More')} onPress={() => onMore(person)} />
      </View>
      <Plan request={request} />
      <Button
        label={request.offered ? tr('Take it back') : tr("I'm in")}
        icon={request.offered ? 'close' : 'hand-right-outline'}
        kind={request.offered ? 'secondary' : 'primary'}
        small
        loading={busy}
        onPress={toggle}
      />
      {request.offered ? <T v="small" soft>{tr("{name} will see you're in, and picks who to go with.", { name: person.display_name })}</T> : null}
    </Glass>
  );
}

function MineRow({ request, onChange, onMore }: { request: BuddyRequest; onChange: () => void; onMore: (p: Person) => void }) {
  const live = useLive();
  const [busy, setBusy] = useState(false);
  const open = request.status === 'open';
  const pick = async (person: Person) => {
    if (!(await confirm(tr('Go with {name}?', { name: person.display_name }), MEET_SAFELY(), tr('Go with {name}', { name: person.display_name })))) return;
    setBusy(true);
    try {
      const matchId = await api.buddies.pick(request.id, person.user_id);
      live.refreshMatches();
      onChange();
      router.push(`/chat/${matchId}`);
    } catch (e) {
      showError(e);
    } finally {
      setBusy(false);
    }
  };
  const close = async () => {
    if (!(await confirm(tr('Take this post down?'), tr("People can't join it any more."), tr('Take it down'), true))) return;
    setBusy(true);
    try {
      await api.buddies.close(request.id);
      onChange();
    } catch (e) {
      showError(e);
    } finally {
      setBusy(false);
    }
  };
  return (
    <Glass style={{ gap: space.md }}>
      <Plan request={request} />
      {open ? (
        <>
          {request.offers?.length ? (
            request.offers.map((o) => (
              <View key={o.person.user_id} style={{ flexDirection: 'row', alignItems: 'center', gap: space.sm }}>
                <PersonLine person={o.person} size={40} />
                <IconButton icon="ellipsis-horizontal" label={tr('More')} onPress={() => onMore(o.person)} />
                <Button label={tr('Pick')} small loading={busy} onPress={() => pick(o.person)} />
              </View>
            ))
          ) : (
            <T v="small" soft>{tr("Nobody's in yet. I'll let you know when someone is.")}</T>
          )}
          <Button label={tr('Take it down')} kind="secondary" small loading={busy} onPress={close} />
        </>
      ) : request.status === 'filled' && request.match_id ? (
        <Button label={tr('Open the chat')} icon="chatbubbles" kind="secondary" small onPress={() => router.push(`/chat/${request.match_id}`)} />
      ) : (
        <Pill tone="plain" label={tr('Closed')} />
      )}
    </Glass>
  );
}

type Row = { key: string; type: 'mine' | 'board'; request: BuddyRequest } | { key: string; type: 'heading'; label: string };

export default function Buddies() {
  const t = useTheme();
  const live = useLive();
  const [board, setBoard] = useState<BuddyRequest[]>([]);
  const [mine, setMine] = useState<BuddyRequest[]>([]);
  const [missing, setMissing] = useState<string[]>([]);
  const [loaded, setLoaded] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [posting, setPosting] = useState(false);
  const [more, setMore] = useState<Person | null>(null);
  const [reporting, setReporting] = useState<Person | null>(null);

  const load = useCallback(async () => {
    try {
      const [b, m] = await Promise.all([api.buddies.board(), api.buddies.mine()]);
      setBoard(b.requests);
      setMissing(b.missing);
      setMine(m);
      setLoaded(true);
    } catch {
      // Pull to refresh tries again; what's on screen stays.
    }
  }, []);
  useEffect(() => {
    load();
  }, [load]);
  useTopic('buddies', load);

  const openMine = mine.filter((r) => r.status === 'open');
  const pastMine = mine.filter((r) => r.status !== 'open').slice(0, 5);
  const rows: Row[] = [];
  if (openMine.length || pastMine.length) {
    rows.push({ key: 'h-mine', type: 'heading', label: tr('Your posts') });
    for (const r of [...openMine, ...pastMine]) rows.push({ key: r.id, type: 'mine', request: r });
  }
  if (board.length) {
    rows.push({ key: 'h-board', type: 'heading', label: tr('Near you') });
    for (const r of board) rows.push({ key: r.id, type: 'board', request: r });
  }
  const ready = missing.length === 0;

  return (
    <Backdrop>
      <TabHeader
        title={tr('Buddies')}
        subtitle={tr('Company for one plan: a movie tonight, a trip, a business contact.')}
        right={ready && loaded ? <Button label={tr('Post')} icon="add" small onPress={() => setPosting(true)} /> : undefined}
      />
      <FlatList
        data={rows}
        keyExtractor={(r) => r.key}
        contentContainerStyle={{ paddingHorizontal: space.lg, paddingBottom: space.xxl, gap: space.md, flexGrow: 1 }}
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
        ListEmptyComponent={
          !loaded ? (
            <SukiLoading line={tr('Seeing who is around...')} />
          ) : !ready ? (
            <SukiSays face="playful" size={132} title={tr('A little about you first')} body={tr('People want to know who is coming. Add your name, birthday, city and a photo, and you can post or join.')}>
              <Button label={tr('Do it with Suki')} icon="sparkles" onPress={() => router.navigate('/suki')} />
            </SukiSays>
          ) : (
            <SukiSays face="playful" size={132} title={tr('Nothing posted nearby yet')} body={tr('Be the first. Say what you want to do and when, and people in your city can join you.')}>
              <Button label={tr('Post a plan')} icon="add" onPress={() => setPosting(true)} />
            </SukiSays>
          )
        }
        renderItem={({ item }) => {
          if (item.type === 'heading') return <T v="subheading" style={{ marginTop: space.sm }}>{item.label}</T>;
          if (item.type === 'mine') return <MineRow request={item.request} onChange={load} onMore={setMore} />;
          return <BoardRow request={item.request} onChange={load} onMore={setMore} />;
        }}
      />

      <BuddyPostSheet
        visible={posting}
        onClose={() => setPosting(false)}
        onPosted={() => {
          setPosting(false);
          load();
        }}
      />

      <Sheet visible={!!more} onClose={() => setMore(null)}>
        <SheetAction
          icon="flag-outline"
          label={tr('Report')}
          onPress={() => {
            setReporting(more);
            setMore(null);
          }}
        />
        <SheetAction
          icon="ban-outline"
          label={tr('Block')}
          danger
          onPress={async () => {
            const person = more;
            setMore(null);
            if (!person) return;
            if (!(await confirm(tr('Block this person?'), tr("You'll stop seeing each other, and any match ends at once."), tr('Block'), true))) return;
            try {
              await api.safety.block(person.user_id);
              live.refreshMatches();
              load();
            } catch (e) {
              showError(e);
            }
          }}
        />
      </Sheet>

      <ReportSheet
        visible={!!reporting}
        name={reporting?.display_name ?? ''}
        matched={false}
        onClose={() => setReporting(null)}
        onSend={async (reason, details) => {
          if (!reporting) return;
          await api.safety.report({ user_id: reporting.user_id, reason, details });
          setReporting(null);
          Alert.alert(tr('Thank you for telling us'), tr('A person on our team will look at this.'));
        }}
      />
    </Backdrop>
  );
}
