import { router } from 'expo-router';
import { useCallback, useEffect, useState } from 'react';
import { View } from 'react-native';
import { Alert } from '@/lib/dialogs';
import * as api from '@/api/endpoints';
import type { BuddyRequest, Card, Person } from '@/api/types';
import { useLive } from '@/state/live';
import { space } from '@/theme/tokens';
import { PersonAvatar } from '@/ui/person';
import { Button, ErrorText, Glass, IconButton, Pill, T } from '@/ui/primitives';
import { confirm, Sheet, SheetAction, showError } from '@/ui/sheet';
import { tr } from '@/i18n';
import { ReportSheet } from './ReportSheet';

const KINDS: Record<string, string> = {
  movie: 'Movie',
  meal: 'Meal',
  show: 'Show',
  travel: 'Travel',
  sports: 'Sports',
  games: 'Games',
  business: 'Business',
  study: 'Study',
  other: 'Something else',
};

const meetSafely = () => tr('For a first meet-up, pick somewhere public and tell a friend. Never send money to someone you met here.');

function Plan({ kind, when, note }: { kind: string; when: string; note?: string }) {
  return (
    <View style={{ gap: 6 }}>
      <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 6, alignItems: 'center' }}>
        <Pill tone="accent" label={tr(KINDS[kind] ?? 'Something else')} />
        <T v="small" style={{ fontWeight: '700', flexShrink: 1 }}>
          {when}
        </T>
      </View>
      {note ? <T>{note}</T> : null}
    </View>
  );
}

/** Who they are, with the report and block everyone in the app can always reach. */
function PersonRow({ person, onBlocked }: { person: Person; onBlocked: () => void }) {
  const live = useLive();
  const [menu, setMenu] = useState(false);
  const [reporting, setReporting] = useState(false);
  return (
    <View style={{ flexDirection: 'row', alignItems: 'center', gap: space.md }}>
      <PersonAvatar id={person.photo_ids?.[0]} size={44} />
      <View style={{ flex: 1 }}>
        <T v="subheading">{person.display_name}</T>
        <T v="small" soft>
          {[person.age ? String(person.age) : '', person.occupation].filter(Boolean).join(' · ')}
        </T>
      </View>
      <IconButton icon="ellipsis-horizontal" label={tr('More')} onPress={() => setMenu(true)} />
      <Sheet visible={menu} onClose={() => setMenu(false)}>
        <SheetAction
          icon="flag-outline"
          label={tr('Report')}
          onPress={() => {
            setMenu(false);
            setReporting(true);
          }}
        />
        <SheetAction
          icon="ban-outline"
          label={tr('Block')}
          danger
          onPress={async () => {
            setMenu(false);
            if (!(await confirm(tr('Block this person?'), tr("You'll stop seeing each other, and any match ends at once."), tr('Block'), true))) return;
            try {
              await api.safety.block(person.user_id);
              live.refreshMatches();
              onBlocked();
            } catch (e) {
              showError(e);
            }
          }}
        />
      </Sheet>
      <ReportSheet
        visible={reporting}
        name={person.display_name}
        matched={false}
        onClose={() => setReporting(false)}
        onSend={async (reason, details) => {
          await api.safety.report({ user_id: person.user_id, reason, details });
          setReporting(false);
          Alert.alert(tr('Thank you for telling us'), tr('A person on our team will look at this.'));
        }}
      />
    </View>
  );
}

function useBuddyRequest(id: string) {
  const [request, setRequest] = useState<BuddyRequest | null>(null);
  const [gone, setGone] = useState(false);
  const load = useCallback(async () => {
    try {
      setRequest(await api.buddies.get(id));
    } catch (e: any) {
      // A post that closed, or someone blocked, reads the same: it isn't there any more.
      if (e?.status === 404) setGone(true);
    }
  }, [id]);
  useEffect(() => {
    load();
  }, [load]);
  return { request, gone, load, setGone };
}

const Closed = ({ label }: { label: string }) => <Pill tone="plain" icon="time-outline" label={label} />;

/** One of this person's own plans, on the Dates tab. */
export function MyBuddyPost({ request, onChange }: { request: BuddyRequest; onChange: () => void }) {
  const live = useLive();
  const [busy, setBusy] = useState(false);
  const pick = async (person: Person) => {
    if (!(await confirm(tr('Go with {name}?', { name: person.display_name }), meetSafely(), tr('Pick them')))) return;
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
      <Plan kind={request.kind} when={request.when} note={request.note} />
      {request.status === 'filled' && request.match_id ? (
        <Button label={tr('Open the chat')} icon="chatbubbles" kind="secondary" small onPress={() => router.push(`/chat/${request.match_id}`)} />
      ) : (
        <>
          {request.offers?.length ? (
            request.offers.map((o) => (
              <View key={o.person.user_id} style={{ gap: space.sm }}>
                <PersonRow person={o.person} onBlocked={onChange} />
                <Button label={tr('Pick them')} icon="checkmark" small loading={busy} onPress={() => pick(o.person)} />
              </View>
            ))
          ) : (
            <T v="small" soft>{tr("Nobody's in yet. I'll let you know when someone is.")}</T>
          )}
          <Button label={tr('Take it down')} kind="secondary" small loading={busy} onPress={close} />
        </>
      )}
    </Glass>
  );
}

/** Suki drafted this from what they told her. Nothing is up until the tap. */
export function BuddyPostCard({ card }: { card: Extract<Card, { kind: 'buddy_post' }> }) {
  const [state, setState] = useState<'idle' | 'busy' | 'done'>('idle');
  const [error, setError] = useState<string | null>(null);
  if (state === 'done') return <Pill tone="good" icon="checkmark-circle" label={tr('Posted.')} />;
  return (
    <Glass style={{ gap: space.sm }}>
      <Plan kind={card.activity} when={card.when} note={card.note} />
      {card.same_gender ? <T v="small" soft>{tr('Only people of my gender')}</T> : null}
      <ErrorText message={error} />
      <Button
        label={tr('Post it')}
        icon="paper-plane"
        small
        loading={state === 'busy'}
        onPress={async () => {
          setState('busy');
          setError(null);
          try {
            await api.buddies.post({
              kind: card.activity,
              when: card.when,
              note: card.note ?? '',
              expires_in_hours: card.expires_in_hours,
              same_gender: !!card.same_gender,
            });
            setState('done');
          } catch (e: any) {
            setError(e.message);
            setState('idle');
          }
        }}
      />
    </Glass>
  );
}

/** Someone else's plan. */
export function BuddyRequestCard({ id }: { id: string }) {
  const { request, gone, load, setGone } = useBuddyRequest(id);
  const [busy, setBusy] = useState(false);
  if (gone) return <Closed label={tr('This plan has closed.')} />;
  if (!request?.person) return null;
  const person = request.person;
  const toggle = async () => {
    if (!request.offered && !(await confirm(tr('Join {name}?', { name: person.display_name }), meetSafely(), tr("I'm in")))) return;
    setBusy(true);
    try {
      if (request.offered) await api.buddies.withdraw(id);
      else await api.buddies.offer(id);
      await load();
    } catch (e) {
      showError(e);
    } finally {
      setBusy(false);
    }
  };
  return (
    <Glass style={{ gap: space.md }}>
      <PersonRow person={person} onBlocked={() => setGone(true)} />
      <Plan kind={request.kind} when={request.when} note={request.note} />
      {request.offered ? <T v="small" soft>{tr("{name} will see you're in, and picks who to go with.", { name: person.display_name })}</T> : null}
      <Button
        label={request.offered ? tr('Take back') : tr("I'm in")}
        icon={request.offered ? 'close' : 'hand-right-outline'}
        kind={request.offered ? 'secondary' : 'primary'}
        small
        loading={busy}
        onPress={toggle}
      />
    </Glass>
  );
}

/** Someone who said they're in on this person's own plan. */
export function BuddyOfferCard({ id, userId }: { id: string; userId: string }) {
  const live = useLive();
  const { request, gone, load, setGone } = useBuddyRequest(id);
  const [busy, setBusy] = useState(false);
  if (gone) return <Closed label={tr('This plan has closed.')} />;
  if (!request) return null;
  if (request.status === 'filled') return <Closed label={tr('You picked someone for this one.')} />;
  if (request.status !== 'open') return <Closed label={tr('This plan has closed.')} />;
  const person = request.offers?.find((o) => o.person.user_id === userId)?.person;
  if (!person) return null;
  return (
    <Glass style={{ gap: space.md }}>
      <PersonRow person={person} onBlocked={() => setGone(true)} />
      <Plan kind={request.kind} when={request.when} note={request.note} />
      <Button
        label={tr('Pick them')}
        icon="checkmark"
        small
        loading={busy}
        onPress={async () => {
          if (!(await confirm(tr('Go with {name}?', { name: person.display_name }), meetSafely(), tr('Pick them')))) return;
          setBusy(true);
          try {
            const matchId = await api.buddies.pick(id, userId);
            live.refreshMatches();
            await load();
            router.push(`/chat/${matchId}`);
          } catch (e) {
            showError(e);
          } finally {
            setBusy(false);
          }
        }}
      />
    </Glass>
  );
}
