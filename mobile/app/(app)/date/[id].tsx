import DateTimePicker, { DateTimePickerAndroid } from '@react-native-community/datetimepicker';
import { router, useLocalSearchParams } from 'expo-router';
import { useCallback, useEffect, useRef, useState } from 'react';
import { Linking, Platform, Pressable, Share, View } from 'react-native';
import * as api from '@/api/endpoints';
import type { DatePlan, Venue } from '@/api/types';
import { longDateTime } from '@/lib/format';
import { openVenueOnMap } from '@/lib/links';
import { useForeground, useLive, useTopic } from '@/state/live';
import { useTheme } from '@/theme/ThemeProvider';
import { radius, space } from '@/theme/tokens';
import { Backdrop, Page, ScreenHeader } from '@/ui/layout';
import { PersonAvatar } from '@/ui/person';
import { Button, ErrorText, Field, Glass, Pill, T } from '@/ui/primitives';
import { Sheet, showError } from '@/ui/sheet';
import { SukiAvatar, SukiLoading, SukiSays, TypingDots } from '@/ui/suki';

const HOUR = 3600 * 1000;
const DAY = 24 * HOUR;

const SAFETY_TIPS = [
  'Stay somewhere public, at least the first time.',
  'Tell a friend where you’ll be. The share button does it in one tap.',
  'Get yourself there and back. No need for a ride from them yet.',
  'Keep an eye on your drink, and on how you feel.',
  'You can leave at any time, for any reason. I’ll back you up.',
];

function kindLabel(kind: string) {
  return kind ? kind.replace(/_/g, ' ').replace(/^./, (c) => c.toUpperCase()) : '';
}

function VenueDetails({ venue }: { venue: Venue }) {
  const t = useTheme();
  return (
    <View style={{ gap: space.xs }}>
      <T v="heading">{venue.name}</T>
      {venue.kind ? <Pill label={kindLabel(venue.kind)} /> : null}
      {venue.address ? (
        <T v="small" soft>
          {venue.address}
        </T>
      ) : null}
      {venue.hours ? (
        <T v="caption" faint>
          {venue.hours}
        </T>
      ) : null}
      <View style={{ flexDirection: 'row', gap: space.lg, marginTop: 2 }}>
        <Pressable accessibilityRole="link" hitSlop={8} onPress={() => openVenueOnMap(venue)}>
          <T v="small" color={t.accent} style={{ fontWeight: '700' }}>
            Open map
          </T>
        </Pressable>
        {venue.website ? (
          <Pressable accessibilityRole="link" hitSlop={8} onPress={() => Linking.openURL(venue.website!).catch(() => {})}>
            <T v="small" color={t.accent} style={{ fontWeight: '700' }}>
              Website
            </T>
          </Pressable>
        ) : null}
      </View>
    </View>
  );
}

function SukiLine({ mood = 'happy', text }: { mood?: string; text: string }) {
  const t = useTheme();
  return (
    <View style={{ flexDirection: 'row', gap: space.sm, alignItems: 'flex-start' }}>
      <SukiAvatar mood={mood} size={42} />
      <View style={{ flex: 1, backgroundColor: t.bubbleOther, borderRadius: radius.md, borderTopLeftRadius: 6, padding: space.md }}>
        <T v="small">{text}</T>
      </View>
    </View>
  );
}

/** Android has separate date and time dialogs; iOS picks both inline, in a sheet. */
function useTimePicker(onPick: (d: Date) => void) {
  const [iosOpen, setIosOpen] = useState(false);
  const [value, setValue] = useState(() => new Date(Date.now() + 3 * DAY));

  const open = (initial?: string | null) => {
    const min = new Date(Date.now() + HOUR + 5 * 60000);
    const max = new Date(Date.now() + 89 * DAY);
    const start = initial && new Date(initial) > min ? new Date(initial) : new Date(Date.now() + 2 * DAY);
    start.setSeconds(0, 0);
    if (Platform.OS === 'ios') {
      setValue(start);
      setIosOpen(true);
      return;
    }
    DateTimePickerAndroid.open({
      value: start,
      mode: 'date',
      minimumDate: min,
      maximumDate: max,
      onChange: (e, day) => {
        if (e.type !== 'set' || !day) return;
        DateTimePickerAndroid.open({
          value: start,
          mode: 'time',
          onChange: (e2, time) => {
            if (e2.type !== 'set' || !time) return;
            const at = new Date(day);
            at.setHours(time.getHours(), time.getMinutes(), 0, 0);
            onPick(at);
          },
        });
      },
    });
  };

  const sheet =
    Platform.OS === 'ios' ? (
      <Sheet visible={iosOpen} onClose={() => setIosOpen(false)} title="When works for you?">
        <DateTimePicker
          value={value}
          mode="datetime"
          display="inline"
          minimumDate={new Date(Date.now() + HOUR + 5 * 60000)}
          maximumDate={new Date(Date.now() + 89 * DAY)}
          onChange={(_, d) => d && setValue(d)}
        />
        <Button
          label="Suggest this time"
          full
          onPress={() => {
            setIosOpen(false);
            onPick(value);
          }}
        />
      </Sheet>
    ) : null;

  return { open, sheet };
}

export default function DateScreen() {
  const t = useTheme();
  const { id, focus } = useLocalSearchParams<{ id: string; focus?: 'time' | 'change' }>();
  const live = useLive();
  const [date, setDate] = useState<DatePlan | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState<string | null>(null);
  const [cancelOpen, setCancelOpen] = useState(false);
  const [reason, setReason] = useState('');
  const focusUsed = useRef(false);

  const load = useCallback(async () => {
    if (!id) return;
    try {
      setDate(await api.dates.get(id));
      setError(null);
    } catch (e: any) {
      setError(e.message);
    }
  }, [id]);

  useEffect(() => {
    load();
  }, [load]);
  useTopic('dates', load);
  useTopic((x) => !!date && x === `chat:${date.match_id}`, load);
  useForeground(load);

  const act = async (name: string, fn: () => Promise<DatePlan>) => {
    setBusy(name);
    try {
      setDate(await fn());
      live.refreshDates();
      live.refreshMatches();
    } catch (e) {
      showError(e);
    } finally {
      setBusy(null);
    }
  };

  const picker = useTimePicker((at) => {
    if (id) act('time', () => api.dates.time(id, at.toISOString()));
  });

  // Suki's card can land here wanting the time picker straight away.
  useEffect(() => {
    if (!date || focusUsed.current || focus !== 'time') return;
    if (date.status === 'scheduling' || date.status === 'confirmed') {
      focusUsed.current = true;
      picker.open(date.proposed_at ?? date.scheduled_at);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [date, focus]);

  if (!date) {
    return (
      <Backdrop>
        <ScreenHeader title="Your date" />
        <View style={{ flex: 1, justifyContent: 'center' }}>
          {error ? (
            <SukiSays face="pouty" title="I can't open this date" body={error}>
              <Button label="Try again" kind="secondary" onPress={load} />
            </SukiSays>
          ) : (
            <SukiLoading line="Getting the plan..." />
          )}
        </View>
      </Backdrop>
    );
  }

  const name = date.person.display_name;
  const open = ['searching', 'voting', 'scheduling', 'confirmed'].includes(date.status);
  const venues = date.venues ?? [];

  return (
    <Backdrop>
      <ScreenHeader title={`Date with ${name}`} />
      <Page refreshing={false} onRefresh={load}>
        <Glass style={{ flexDirection: 'row', gap: space.md, alignItems: 'center' }}>
          <Pressable accessibilityRole="button" accessibilityLabel={`${name}'s profile`} onPress={() => router.push(`/person?match=${date.match_id}`)}>
            <PersonAvatar id={date.person.photo_ids?.[0]} size={56} />
          </Pressable>
          <View style={{ flex: 1 }}>
            <T v="subheading">{name}</T>
            <Pressable accessibilityRole="button" hitSlop={6} onPress={() => router.push(`/chat/${date.match_id}`)}>
              <T v="small" color={t.accent} style={{ fontWeight: '600' }}>
                Open your chat
              </T>
            </Pressable>
          </View>
        </Glass>

        {focus === 'change' && (date.status === 'scheduling' || date.status === 'confirmed') ? (
          <SukiLine mood="thinking" text="Plans change, no drama. Pick a new time or cancel below." />
        ) : null}

        {date.status === 'searching' ? (
          <Glass style={{ alignItems: 'center', gap: space.md, paddingVertical: space.xxl }}>
            <SukiAvatar mood="thinking" size={88} ring={3} />
            <TypingDots />
            <T v="heading" center>
              Looking for places
            </T>
            <T soft center>
              I'm searching for spots you'd both enjoy. I'll let you know the moment I have a few.
            </T>
          </Glass>
        ) : null}

        {date.status === 'voting' ? (
          <>
            <SukiLine
              mood="playful"
              text={
                date.my_vote
                  ? date.other_voted
                    ? 'You both voted. Sorting it out now.'
                    : `Nice pick. ${name} hasn't voted yet. You can change your mind until they do.`
                  : date.other_voted
                    ? `${name} has already voted. Your pick stays private until you've both chosen.`
                    : `Pick your favorite. ${name} is choosing too, and your pick stays private until you've both chosen.`
              }
            />
            {venues.map((v) => {
              const mine = date.my_vote === v.id;
              return (
                <Glass key={v.id} style={[{ gap: space.md }, mine && { borderColor: t.accent, borderWidth: 2 }]}>
                  <VenueDetails venue={v} />
                  {v.reason ? (
                    <View style={{ flexDirection: 'row', gap: space.sm }}>
                      <SukiAvatar mood="happy" size={26} ring={1.5} />
                      <T v="small" soft style={{ flex: 1, fontStyle: 'italic' }}>
                        {v.reason}
                      </T>
                    </View>
                  ) : null}
                  {mine ? (
                    <Pill tone="good" icon="checkmark-circle" label="Your pick" />
                  ) : (
                    <Button
                      label={date.my_vote ? 'Switch to this one' : 'Vote for this one'}
                      kind={date.my_vote ? 'secondary' : 'primary'}
                      small
                      loading={busy === `vote:${v.id}`}
                      disabled={!!busy}
                      onPress={() => act(`vote:${v.id}`, () => api.dates.vote(date.id, v.id))}
                    />
                  )}
                </Glass>
              );
            })}
          </>
        ) : null}

        {date.status === 'scheduling' ? (
          <>
            {date.venue ? (
              <Glass>
                <VenueDetails venue={date.venue} />
              </Glass>
            ) : null}
            {!date.proposed_at ? (
              <Glass style={{ gap: space.md }}>
                <T v="heading">When works for you?</T>
                <T soft>Suggest a time and I'll ask {name}.</T>
                <Button label="Pick a date and time" icon="calendar" full loading={busy === 'time'} onPress={() => picker.open()} />
              </Glass>
            ) : date.proposed_by_me ? (
              <Glass style={{ gap: space.md }}>
                <T v="label" soft>
                  You suggested
                </T>
                <T v="heading">{longDateTime(date.proposed_at)}</T>
                <T soft>Waiting for {name} to say yes.</T>
                <Button label="Suggest a different time" kind="secondary" small loading={busy === 'time'} onPress={() => picker.open(date.proposed_at)} />
              </Glass>
            ) : (
              <Glass style={{ gap: space.md }}>
                <T v="label" soft>
                  {name} suggested
                </T>
                <T v="heading">{longDateTime(date.proposed_at)}</T>
                <Button label="That works" icon="checkmark" full loading={busy === 'confirm'} disabled={!!busy} onPress={() => act('confirm', () => api.dates.confirm(date.id))} />
                <Button label="Suggest another time" kind="secondary" full disabled={!!busy} loading={busy === 'time'} onPress={() => picker.open(date.proposed_at)} />
              </Glass>
            )}
          </>
        ) : null}

        {date.status === 'confirmed' ? (
          <>
            <Glass style={{ gap: space.md }}>
              <Pill tone="good" icon="checkmark-circle" label="You're on!" />
              <T v="title">{longDateTime(date.scheduled_at)}</T>
              {date.venue ? <VenueDetails venue={date.venue} /> : null}
            </Glass>
            <Button
              label="Share the plan with a friend"
              icon="share-outline"
              full
              loading={busy === 'share'}
              onPress={async () => {
                setBusy('share');
                try {
                  const message = await api.dates.share(date.id);
                  await Share.share({ message });
                } catch (e) {
                  showError(e);
                } finally {
                  setBusy(null);
                }
              }}
            />
            <Glass style={{ gap: space.md }}>
              <View style={{ flexDirection: 'row', alignItems: 'center', gap: space.sm }}>
                <SukiAvatar mood="happy" size={36} />
                <T v="subheading">Before you go</T>
              </View>
              {SAFETY_TIPS.map((tip) => (
                <View key={tip} style={{ flexDirection: 'row', gap: space.sm }}>
                  <T v="small" color={t.accent}>
                    ●
                  </T>
                  <T v="small" soft style={{ flex: 1 }}>
                    {tip}
                  </T>
                </View>
              ))}
            </Glass>
            <Button label="Reschedule" icon="time-outline" kind="secondary" full loading={busy === 'time'} onPress={() => picker.open(date.scheduled_at)} />
          </>
        ) : null}

        {date.status === 'completed' ? (
          <SukiSays
            face="happy"
            size={108}
            title={`How did it go with ${name}?`}
            body={date.venue ? `You met at ${date.venue.name}${date.scheduled_at ? ` on ${longDateTime(date.scheduled_at)}` : ''}. I'd love to hear about it.` : "I'd love to hear about it."}
          >
            <Button label="Tell Suki" icon="chatbubble-ellipses" onPress={() => router.navigate('/suki')} />
          </SukiSays>
        ) : null}

        {date.status === 'cancelled' ? (
          <SukiSays face="pouty" size={108} title="This date was called off" body="It happens. You can always plan another one from your chat, if the match is still open." />
        ) : null}

        {date.status === 'failed' ? (
          <SukiSays face="pouty" size={108} title="I couldn't find a good place" body="That's on me. You can try again from your chat, or pick somewhere yourselves.">
            <Button label="Back to your chat" kind="secondary" onPress={() => router.push(`/chat/${date.match_id}`)} />
          </SukiSays>
        ) : null}

        {date.constraints?.length && open ? (
          <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 6 }}>
            {date.constraints.map((c) => (
              <Pill key={c} tone="plain" label={c} />
            ))}
          </View>
        ) : null}

        {open ? (
          <Button
            label="Cancel this date"
            kind="ghost"
            full
            onPress={() => setCancelOpen(true)}
          />
        ) : null}
        <ErrorText message={error} />
      </Page>

      {picker.sheet}

      <Sheet visible={cancelOpen} onClose={() => setCancelOpen(false)} title="Cancel this date?">
        <T soft>I'll let {name} know, kindly.</T>
        <Field label="Why? (optional)" value={reason} onChangeText={setReason} maxLength={200} placeholder="Something came up" />
        <Button
          label="Cancel the date"
          kind="danger"
          full
          loading={busy === 'cancel'}
          onPress={async () => {
            await act('cancel', () => api.dates.cancel(date.id, reason.trim() || undefined));
            setCancelOpen(false);
          }}
        />
        <Button label="Keep it" kind="ghost" full onPress={() => setCancelOpen(false)} />
      </Sheet>
    </Backdrop>
  );
}

