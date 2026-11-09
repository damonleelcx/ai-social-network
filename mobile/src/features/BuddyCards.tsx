import { router } from 'expo-router';
import { useCallback, useEffect, useState } from 'react';
import { View } from 'react-native';
import { Alert } from '@/lib/dialogs';
import * as api from '@/api/endpoints';
import type { BuddyRequest, Card, Person, Trip } from '@/api/types';
import { useLive } from '@/state/live';
import { space } from '@/theme/tokens';
import { PersonAvatar } from '@/ui/person';
import { Button, ErrorText, Glass, IconButton, Pill, T } from '@/ui/primitives';
import { confirm, Sheet, SheetAction, showError } from '@/ui/sheet';
import { tr } from '@/i18n';
import { ReportSheet } from './ReportSheet';
import { monthDay } from '@/lib/format';
import { HeroCard, PhotoBadge } from '@/ui/HeroCard';

export const KINDS: Record<string, string> = {
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

export const meetSafely = () => tr('For a first meet-up, pick somewhere public and tell a friend. Never send money to someone you met here.');

/** How full a group is: everyone going counts the poster, spots count only who can join. */
export function groupLine(going: number, spots: number) {
  const left = Math.max(0, spots - (going - 1));
  return left > 0 ? tr('{n} going · {left} spots left', { n: String(going), left: String(left) }) : tr('{n} going · full', { n: String(going) });
}

export function Plan({
  kind,
  when,
  note,
  place,
  city,
  group,
}: {
  kind: string;
  when: string;
  note?: string;
  place?: string;
  city?: string;
  group?: { going: number; spots: number };
}) {
  return (
    <View style={{ gap: 6 }}>
      <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 6, alignItems: 'center' }}>
        <Pill tone="accent" label={tr(KINDS[kind] ?? 'Something else')} />
        {group ? <Pill tone="plain" icon="people-outline" label={groupLine(group.going, group.spots)} /> : null}
        <T v="small" style={{ fontWeight: '700', flexShrink: 1 }}>
          {when}
        </T>
      </View>
      {place || city ? (
        <T v="small" soft>
          {[place, city].filter(Boolean).join(' · ')}
        </T>
      ) : null}
      {note ? <T>{note}</T> : null}
    </View>
  );
}

/** The plan's details, from a post. */
export const planOf = (r: BuddyRequest, city?: boolean) => ({
  kind: r.kind,
  when: r.when,
  note: r.note,
  place: r.place,
  city: city ? r.city_name : undefined,
  group: r.spots > 1 ? { going: r.going, spots: r.spots } : undefined,
});

/** The report and block everyone in the app can always reach, behind a ⋯ button. */
export function PersonMenu({ person, onBlocked, onPhoto }: { person: Person; onBlocked: () => void; onPhoto?: boolean }) {
  const live = useLive();
  const [menu, setMenu] = useState(false);
  const [reporting, setReporting] = useState(false);
  const button = <IconButton icon="ellipsis-horizontal" label={tr('More')} color={onPhoto ? '#fff' : undefined} onPress={() => setMenu(true)} />;
  return (
    <>
      {onPhoto ? <View style={{ backgroundColor: 'rgba(8,4,24,0.45)', borderRadius: 999 }}>{button}</View> : button}
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
    </>
  );
}

/** Who they are, small: for lists such as who's in a group. */
export function PersonRow({ person, onBlocked }: { person: Person; onBlocked: () => void }) {
  return (
    <View style={{ flexDirection: 'row', alignItems: 'center', gap: space.md }}>
      <PersonAvatar id={person.photo_ids?.[0]} size={44} />
      <View style={{ flex: 1 }}>
        <T v="subheading">{person.display_name}</T>
        <T v="small" soft>
          {[person.age ? String(person.age) : '', person.occupation].filter(Boolean).join(' · ')}
        </T>
      </View>
      <PersonMenu person={person} onBlocked={onBlocked} />
    </View>
  );
}

/** Their name and age, and their work and city, for over a photo. */
const heroTitle = (p: Person) => [p.display_name, p.age ? String(p.age) : ''].filter(Boolean).join(', ');
const heroLine = (p: Person) => [p.occupation, p.city_name].filter(Boolean).join(' · ');

/** What kind of plan, and how full a group is, as labels over the photo. */
function PlanBadges({ r }: { r: BuddyRequest }) {
  return (
    <>
      {r.person?.verified ? <PhotoBadge label={tr('Verified')} /> : null}
      <PhotoBadge label={tr(KINDS[r.kind] ?? 'Something else')} />
      {r.spots > 1 ? <PhotoBadge label={groupLine(r.going, r.spots)} /> : null}
    </>
  );
}

/** When, where and what, under the photo. */
function PlanBody({ r, city }: { r: BuddyRequest; city?: boolean }) {
  return (
    <View style={{ gap: 6 }}>
      <T v="subheading">{r.when}</T>
      {r.place || (city && r.city_name) ? (
        <T v="small" soft>
          {[r.place, city ? r.city_name : ''].filter(Boolean).join(' · ')}
        </T>
      ) : null}
      {r.note ? <T>{r.note}</T> : null}
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
    const group = request.spots > 1;
    const ok = group
      ? await confirm(tr('Call off this plan?'), tr("Everyone who joined is told, and the group chat closes."), tr('Call it off'), true)
      : await confirm(tr('Take this post down?'), tr("People can't join it any more."), tr('Take it down'), true);
    if (!ok) return;
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
  if (request.spots > 1) {
    const others = (request.members ?? []).slice(1);
    return (
      <Glass style={{ gap: space.md }}>
        <Plan {...planOf(request, true)} />
        {others.length ? (
          others.map((m) => <PersonRow key={m.user_id} person={m} onBlocked={onChange} />)
        ) : (
          <T v="small" soft>{tr("Nobody's joined yet. I'll let you know when someone does.")}</T>
        )}
        {request.status !== 'closed' ? (
          <>
            <Button label={tr('Open the group chat')} icon="chatbubbles" kind="secondary" small onPress={() => router.push(`/group/${request.id}`)} />
            <Button label={tr('Call it off')} kind="secondary" small loading={busy} onPress={close} />
          </>
        ) : null}
      </Glass>
    );
  }
  return (
    <Glass style={{ gap: space.md }}>
      <Plan {...planOf(request, true)} />
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
  const group = !!card.spots && card.spots > 1;
  const footer = (
    <>
      {group ? <T v="small" soft>{tr('A group: people who join are in at once, and you all share a group chat.')}</T> : null}
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
              spots: card.spots,
              place: card.place,
              trip_id: card.trip_id,
            });
            setState('done');
          } catch (e: any) {
            setError(e.message);
            setState('idle');
          }
        }}
      />
    </>
  );
  if (card.scene) {
    return (
      <HeroCard
        scene={card.scene}
        title={card.when}
        subtitle={[card.place, card.city_name].filter(Boolean).join(' · ')}
        badges={
          <>
            <PhotoBadge label={tr(KINDS[card.activity] ?? 'Something else')} />
            {group ? <PhotoBadge label={groupLine(1, card.spots!)} /> : null}
          </>
        }
        height={220}
      >
        {card.note ? <T>{card.note}</T> : null}
        {footer}
      </HeroCard>
    );
  }
  return (
    <Glass style={{ gap: space.sm }}>
      <Plan
        kind={card.activity}
        when={card.when}
        note={card.note}
        place={card.place}
        city={card.city_name}
        group={group ? { going: 1, spots: card.spots! } : undefined}
      />
      {footer}
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
  const group = request.spots > 1;
  // Their own group: in it, but never offered to join it.
  if (group && request.joined && !request.offered) return <GroupCard id={id} />;
  const toggle = async () => {
    if (!request.offered && !(await confirm(group ? tr("Join {name}'s group?", { name: person.display_name }) : tr('Join {name}?', { name: person.display_name }), meetSafely(), tr("I'm in")))) return;
    if (request.offered && group && !(await confirm(tr('Leave the group?'), tr("You'll leave its group chat too."), tr('Leave'), true))) return;
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
    <HeroCard
      photo={person.photo_ids?.[0]}
      scene={request.scene}
      title={heroTitle(person)}
      subtitle={heroLine(person)}
      badges={<PlanBadges r={request} />}
      corner={<PersonMenu person={person} onBlocked={() => setGone(true)} onPhoto />}
    >
      <PlanBody r={request} city />
      {request.offered && !group ? <T v="small" soft>{tr("{name} will see you're in, and picks who to go with.", { name: person.display_name })}</T> : null}
      {request.offered && group ? (
        <Button label={tr('Open the group chat')} icon="chatbubbles" small onPress={() => router.push(`/group/${id}`)} />
      ) : null}
      <Button
        label={request.offered ? (group ? tr('Leave the group') : tr('Take back')) : group ? tr('Join the group') : tr("I'm in")}
        icon={request.offered ? 'close' : 'hand-right-outline'}
        kind={request.offered ? 'secondary' : 'primary'}
        small
        loading={busy}
        onPress={toggle}
      />
    </HeroCard>
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
    <HeroCard
      photo={person.photo_ids?.[0]}
      title={heroTitle(person)}
      subtitle={heroLine(person)}
      badges={<PhotoBadge label={tr('Wants to join you')} />}
      corner={<PersonMenu person={person} onBlocked={() => setGone(true)} onPhoto />}
    >
      <PlanBody r={request} />
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
    </HeroCard>
  );
}

/** Suki's button to a group chat this person is in. */
export function GroupCard({ id }: { id: string }) {
  const { request, gone } = useBuddyRequest(id);
  if (gone) return <Closed label={tr('This plan has closed.')} />;
  if (!request) return null;
  if (request.scene) {
    return (
      <HeroCard scene={request.scene} title={request.when} subtitle={[request.place, request.city_name].filter(Boolean).join(' · ')} badges={<PlanBadges r={request} />} height={220}>
        {request.note ? <T>{request.note}</T> : null}
        <Button label={tr('Open the group chat')} icon="chatbubbles" small onPress={() => router.push(`/group/${id}`)} />
      </HeroCard>
    );
  }
  return (
    <Glass style={{ gap: space.md }}>
      <Plan {...planOf(request, true)} />
      <Button label={tr('Open the group chat')} icon="chatbubbles" small onPress={() => router.push(`/group/${id}`)} />
    </Glass>
  );
}

const dayLabel = (d: string) => {
  const [y, m, day] = d.split('-').map(Number);
  return monthDay(new Date(y, m - 1, day));
};
export const tripDates = (from: string, to: string) => (from === to ? dayLabel(from) : `${dayLabel(from)} – ${dayLabel(to)}`);

/** Suki found the city; the tap adds the trip. */
export function TripPostCard({ card }: { card: Extract<Card, { kind: 'trip_post' }> }) {
  const [trip, setTrip] = useState<Trip | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // A new trip comes back before its picture is ready, and the card's is that same picture.
  if (trip) return <TripView trip={{ ...trip, scene: trip.scene || card.scene }} />;
  const footer = (
    <>
      {card.note ? <T>{card.note}</T> : null}
      <T v="small" soft>{tr("While you're there, plans in this city reach you, and you can post your own.")}</T>
      <ErrorText message={error} />
      <Button
        label={tr('Add the trip')}
        icon="checkmark"
        small
        loading={busy}
        onPress={async () => {
          setBusy(true);
          setError(null);
          try {
            setTrip(await api.trips.add({ city: card.city, starts_on: card.starts_on, ends_on: card.ends_on, note: card.note }));
          } catch (e: any) {
            setError(e.message);
          } finally {
            setBusy(false);
          }
        }}
      />
    </>
  );
  if (card.scene) {
    return (
      <HeroCard scene={card.scene} title={card.city} subtitle={tripDates(card.starts_on, card.ends_on)} badges={<PhotoBadge label={tr('Trip')} />} height={220}>
        {footer}
      </HeroCard>
    );
  }
  return (
    <Glass style={{ gap: space.sm }}>
      <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 6, alignItems: 'center' }}>
        <Pill tone="accent" icon="airplane-outline" label={card.city} />
        <T v="small" style={{ fontWeight: '700' }}>{tripDates(card.starts_on, card.ends_on)}</T>
      </View>
      {footer}
    </Glass>
  );
}

function TripView({ trip }: { trip: Trip }) {
  const counts = [
    trip.going === 1 ? tr('1 other person is going then') : tr('{n} other people are going then', { n: String(trip.going) }),
    trip.plans === 1 ? tr('1 plan to join there') : tr('{n} plans to join there', { n: String(trip.plans) }),
  ].join(' · ');
  if (trip.scene) {
    return (
      <HeroCard scene={trip.scene} title={trip.city_name} subtitle={tripDates(trip.starts_on, trip.ends_on)} badges={<PhotoBadge label={tr('Trip')} />} height={220}>
        {trip.note ? <T>{trip.note}</T> : null}
        <T v="small" soft>{counts}</T>
      </HeroCard>
    );
  }
  return (
    <Glass style={{ gap: space.sm }}>
      <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 6, alignItems: 'center' }}>
        <Pill tone="accent" icon="airplane-outline" label={trip.city_name} />
        <T v="small" style={{ fontWeight: '700' }}>{tripDates(trip.starts_on, trip.ends_on)}</T>
      </View>
      {trip.note ? <T>{trip.note}</T> : null}
      <T v="small" soft>
        {[
          trip.going === 1 ? tr('1 other person is going then') : tr('{n} other people are going then', { n: String(trip.going) }),
          trip.plans === 1 ? tr('1 plan to join there') : tr('{n} plans to join there', { n: String(trip.plans) }),
        ].join(' · ')}
      </T>
    </Glass>
  );
}

/** One of this person's trips. */
export function TripCard({ id }: { id: string }) {
  const [trip, setTrip] = useState<Trip | null>(null);
  const [gone, setGone] = useState(false);
  useEffect(() => {
    api.trips.get(id).then(setTrip, (e: any) => e?.status === 404 && setGone(true));
  }, [id]);
  if (gone) return <Closed label={tr('This trip was removed.')} />;
  if (!trip) return null;
  return <TripView trip={trip} />;
}
