import { LinearGradient } from 'expo-linear-gradient';
import { router, useFocusEffect, useLocalSearchParams } from 'expo-router';
import { useCallback, useEffect, useMemo, useRef, useState, type ComponentRef } from 'react';
import { Alert, FlatList, Keyboard, KeyboardAvoidingView, Pressable, TextInput, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { ApiError } from '@/api/client';
import * as api from '@/api/endpoints';
import type { Hint, Match, Message, ReportReason } from '@/api/types';
import { ChatInput } from '@/features/ChatInput';
import { REPORT_REASONS, shortTime, uuid } from '@/lib/format';
import { mergeBySeq, useForeground, useLive, useTopic } from '@/state/live';
import { useSession } from '@/state/session';
import { useTheme } from '@/theme/ThemeProvider';
import { radius, space } from '@/theme/tokens';
import { Backdrop } from '@/ui/layout';
import { PersonAvatar } from '@/ui/person';
import { Button, ErrorText, Field, Glass, IconButton, IrisRule, Pill, T } from '@/ui/primitives';
import { confirm, Sheet, SheetAction, showError } from '@/ui/sheet';
import { FadeIn, SukiAvatar, SukiLoading, SukiSays, TypingDots } from '@/ui/suki';

const TYPING_MS = 4000;
const TYPING_THROTTLE = 3000;
const COACH_WAIT_MS = 45000;

type Pending = { client_id: string; text: string; state: 'sending' | 'failed'; error?: string };
type Row =
  | { type: 'typing'; key: string }
  | { type: 'pending'; key: string; pending: Pending }
  | { type: 'msg'; key: string; msg: Message; showRead: boolean };

function CoachPanel({
  hints,
  asking,
  typing,
  onDismiss,
  onPick,
}: {
  hints: Hint[];
  asking: boolean;
  typing: boolean;
  onDismiss: (h: Hint) => void;
  onPick: (text: string) => void;
}) {
  const t = useTheme();
  const hint = hints[0];
  if (!hint && !asking) return null;
  // With the keyboard up the panel would cover the whole conversation, so it steps aside.
  if (typing) {
    return (
      <View style={{ flexDirection: 'row', alignItems: 'center', gap: space.sm, paddingHorizontal: space.lg, paddingVertical: 6 }}>
        <SukiAvatar mood={asking && !hint ? 'thinking' : 'playful'} size={22} />
        <T v="caption" soft>
          {asking && !hint ? 'Suki is thinking about this.' : 'Suki left you a note. Close the keyboard to see it.'}
        </T>
      </View>
    );
  }
  return (
    <FadeIn>
      <View style={{ marginHorizontal: space.md, marginTop: space.sm, borderRadius: radius.md, overflow: 'hidden' }}>
        <LinearGradient colors={t.iris} start={{ x: 0, y: 0 }} end={{ x: 1, y: 1 }} style={{ padding: 1.5, borderRadius: radius.md }}>
          <View style={{ backgroundColor: t.glassStrong, borderRadius: radius.md - 1, padding: space.md, gap: space.sm }}>
            <View style={{ flexDirection: 'row', alignItems: 'center', gap: space.sm }}>
              <SukiAvatar mood={asking && !hint ? 'thinking' : 'playful'} size={30} />
              <View style={{ flex: 1 }}>
                <T v="small" style={{ fontWeight: '700' }}>
                  Suki, just to you
                </T>
                <T v="caption" faint>
                  Only you can see this.
                </T>
              </View>
              {hint ? <IconButton icon="close" size={18} label="Dismiss this hint" onPress={() => onDismiss(hint)} /> : null}
            </View>
            {hint ? (
              <>
                <T v="small">{hint.body}</T>
                {hint.suggestions?.length ? (
                  <View style={{ gap: 6 }}>
                    {hint.suggestions.map((s) => (
                      <Pressable
                        key={s}
                        accessibilityRole="button"
                        accessibilityLabel={`Put in the message box: ${s}`}
                        onPress={() => onPick(s)}
                        style={({ pressed }) => ({
                          backgroundColor: t.accentSoft,
                          borderRadius: radius.sm,
                          paddingHorizontal: space.md,
                          paddingVertical: space.sm,
                          opacity: pressed ? 0.7 : 1,
                        })}
                      >
                        <T v="small" color={t.accent} style={{ fontWeight: '600' }}>
                          {s}
                        </T>
                      </Pressable>
                    ))}
                    <T v="caption" faint>
                      Tap one to put it in your message box. Nothing sends until you do.
                    </T>
                  </View>
                ) : null}
              </>
            ) : (
              <View style={{ flexDirection: 'row', alignItems: 'center', gap: space.sm }}>
                <TypingDots />
                <T v="small" soft>
                  Let me think about this.
                </T>
              </View>
            )}
          </View>
        </LinearGradient>
      </View>
    </FadeIn>
  );
}

function SukiNote({ msg, onAnswer }: { msg: Message; onAnswer: () => void }) {
  const t = useTheme();
  if (msg.role === 'system') {
    return (
      <T v="caption" faint center style={{ paddingHorizontal: space.xl }}>
        {msg.body}
      </T>
    );
  }
  const isPrompt = msg.role === 'prompt';
  const dateId = msg.meta?.date_id;
  const label: Record<string, string> = {
    vote: 'See the places',
    time: 'Pick a time',
    confirm: 'Check the time',
    confirmed: 'See the plan',
  };
  return (
    <View style={{ alignItems: 'center', paddingVertical: space.xs }}>
      <View
        style={{
          width: '88%',
          alignItems: 'center',
          gap: space.sm,
          padding: space.md,
          borderRadius: radius.lg,
          backgroundColor: t.cardTint,
          borderWidth: 1,
          borderColor: t.hairline,
        }}
      >
        <SukiAvatar mood={isPrompt ? 'playful' : 'happy'} size={34} />
        <T v="caption" soft style={{ fontWeight: '700' }}>
          {isPrompt ? 'Suki asks you both' : 'From Suki'}
        </T>
        <T center style={isPrompt ? { fontWeight: '600' } : undefined}>
          {msg.body}
        </T>
        {isPrompt ? <Button label="Answer" small kind="secondary" onPress={onAnswer} /> : null}
        {!isPrompt && dateId ? (
          <Button
            label={label[msg.meta?.card ?? ''] ?? 'Open the date'}
            icon="calendar"
            small
            onPress={() => router.push(`/date/${dateId}`)}
          />
        ) : null}
        <T v="caption" faint>
          {shortTime(msg.created_at)}
        </T>
      </View>
    </View>
  );
}

function Bubble({ msg, mine, showRead }: { msg: Message; mine: boolean; showRead: boolean }) {
  const t = useTheme();
  if (mine) {
    return (
      <View style={{ alignItems: 'flex-end', gap: 3 }}>
        <LinearGradient
          colors={t.brand}
          start={{ x: 0, y: 0 }}
          end={{ x: 1, y: 1 }}
          style={{ maxWidth: '82%', borderRadius: radius.md, borderBottomRightRadius: 6, paddingHorizontal: 14, paddingVertical: 10 }}
        >
          <T color={t.onBrand}>{msg.body}</T>
        </LinearGradient>
        <T v="caption" faint style={{ marginRight: 4 }}>
          {shortTime(msg.created_at)}
          {showRead ? ' · Read' : ''}
        </T>
      </View>
    );
  }
  return (
    <View style={{ alignItems: 'flex-start', gap: 3 }}>
      <View
        style={{
          maxWidth: '82%',
          backgroundColor: t.bubbleOther,
          borderWidth: 1,
          borderColor: t.glassBorder,
          borderRadius: radius.md,
          borderBottomLeftRadius: 6,
          paddingHorizontal: 14,
          paddingVertical: 10,
        }}
      >
        <T>{msg.body}</T>
      </View>
      <T v="caption" faint style={{ marginLeft: 4 }}>
        {shortTime(msg.created_at)}
      </T>
    </View>
  );
}

export default function ChatScreen() {
  const t = useTheme();
  const insets = useSafeAreaInsets();
  const { id } = useLocalSearchParams<{ id: string }>();
  const { user } = useSession();
  const live = useLive();

  const [match, setMatch] = useState<Match | null>(null);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [messages, setMessages] = useState<Message[]>([]);
  const messagesRef = useRef<Message[]>([]);
  messagesRef.current = messages;
  const [loaded, setLoaded] = useState(false);
  const [pending, setPending] = useState<Pending[]>([]);
  const [hints, setHints] = useState<Hint[]>([]);
  const [keyboardUp, setKeyboardUp] = useState(false);
  useEffect(() => {
    const show = Keyboard.addListener('keyboardDidShow', () => setKeyboardUp(true));
    const hide = Keyboard.addListener('keyboardDidHide', () => setKeyboardUp(false));
    return () => {
      show.remove();
      hide.remove();
    };
  }, []);
  const [text, setText] = useState('');
  const [sendError, setSendError] = useState<string | null>(null);
  const [typing, setTyping] = useState(false);
  const [asking, setAsking] = useState(false);
  const [menu, setMenu] = useState(false);
  const [planOpen, setPlanOpen] = useState(false);
  const [reportOpen, setReportOpen] = useState(false);

  const input = useRef<ComponentRef<typeof TextInput>>(null);
  const list = useRef<FlatList<Row>>(null);
  const typingTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const lastTypingSent = useRef(0);
  const lastRead = useRef(0);
  const focused = useRef(true);
  const askTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const hintCount = useRef(0);

  const markRead = useCallback(
    (list: Message[]) => {
      if (!focused.current || !id) return;
      const newest = list.reduce((n, m) => (m.sender_id !== user?.id ? Math.max(n, m.seq) : n), 0);
      if (newest > lastRead.current) {
        lastRead.current = newest;
        api.matches
          .read(id, newest)
          .then(() => live.refreshMatches())
          .catch(() => {});
      }
    },
    [id, live.refreshMatches, user?.id], // eslint-disable-line react-hooks/exhaustive-deps
  );

  /** Read receipts change old messages, which `after=` wouldn't return, so this takes the whole window. */
  const loadMessages = useCallback(async () => {
    if (!id) return;
    try {
      const m = await api.matches.messages(id);
      setMessages((cur) => mergeBySeq(cur, m));
      setLoaded(true);
      markRead(m);
    } catch (e: any) {
      setLoadError((prev) => prev ?? e.message);
      setLoaded(true);
    }
  }, [id, markRead]);

  const loadHints = useCallback(async () => {
    if (!id) return;
    try {
      const h = await api.matches.hints(id);
      setHints(h);
      if (h.length > hintCount.current) setAsking(false);
      hintCount.current = h.length;
    } catch {
      // The panel just stays as it was.
    }
  }, [id]);

  const loadMatch = useCallback(async () => {
    if (!id) return;
    try {
      setMatch(await api.matches.get(id));
      setLoadError(null);
    } catch (e: any) {
      setLoadError(e.message);
    }
  }, [id]);

  useEffect(() => {
    loadMatch();
    loadMessages();
    loadHints();
    return () => {
      if (typingTimer.current) clearTimeout(typingTimer.current);
      if (askTimer.current) clearTimeout(askTimer.current);
    };
  }, [loadHints, loadMatch, loadMessages]);

  useFocusEffect(
    useCallback(() => {
      focused.current = true;
      markRead(messagesRef.current);
      return () => {
        focused.current = false;
      };
    }, [markRead]),
  );

  useTopic(
    (x) => x === `chat:${id}`,
    () => {
      setTyping(false);
      loadMessages();
      loadHints();
      loadMatch();
    },
  );
  useTopic('matches', loadMatch);
  useTopic('dates', loadMatch);
  useTopic(
    (x) => x === `typing:${id}`,
    (topic) => {
      if (topic === 'reconnect') return;
      setTyping(true);
      if (typingTimer.current) clearTimeout(typingTimer.current);
      typingTimer.current = setTimeout(() => setTyping(false), TYPING_MS);
    },
  );
  useForeground(() => {
    loadMatch();
    loadMessages();
    loadHints();
  });

  const onChangeText = (v: string) => {
    setText(v);
    setSendError(null);
    if (v && id && Date.now() - lastTypingSent.current > TYPING_THROTTLE) {
      lastTypingSent.current = Date.now();
      api.matches.typing(id).catch(() => {});
    }
  };

  const deliver = async (p: Pending) => {
    if (!id) return;
    setPending((all) => all.map((x) => (x.client_id === p.client_id ? { ...x, state: 'sending', error: undefined } : x)));
    try {
      const m = await api.matches.send(id, p.text, p.client_id);
      setMessages((cur) => mergeBySeq(cur, [m]));
      setPending((all) => all.filter((x) => x.client_id !== p.client_id));
      live.refreshMatches();
    } catch (e: any) {
      if (e instanceof ApiError && e.status === 422) {
        // The server refused the content (filtered as NSFW, say), so retrying can't help.
        // The words go back in the box to be edited, with its reason shown.
        setPending((all) => all.filter((x) => x.client_id !== p.client_id));
        setText((cur) => (cur.trim() ? `${p.text}\n${cur}` : p.text));
        setSendError(e.message);
        return;
      }
      setPending((all) => all.map((x) => (x.client_id === p.client_id ? { ...x, state: 'failed', error: e.message } : x)));
    }
  };

  const send = () => {
    const value = text.trim();
    if (!value) return;
    const p: Pending = { client_id: uuid(), text: value, state: 'sending' };
    setPending((all) => [...all, p]);
    setText('');
    setSendError(null);
    list.current?.scrollToOffset({ offset: 0, animated: true });
    deliver(p);
  };

  const askSuki = async () => {
    if (!id || asking) return;
    setAsking(true);
    if (askTimer.current) clearTimeout(askTimer.current);
    // If her answer gets lost, don't leave "thinking" on screen forever.
    askTimer.current = setTimeout(() => setAsking(false), COACH_WAIT_MS);
    try {
      await api.matches.coach(id);
    } catch (e) {
      setAsking(false);
      showError(e);
    }
  };

  const dismiss = async (h: Hint) => {
    setHints((all) => all.filter((x) => x.id !== h.id));
    hintCount.current = Math.max(0, hintCount.current - 1);
    api.matches.dismissHint(h.id).catch(() => loadHints());
  };

  const person = match?.person;
  const ended = !!match && match.status !== 'active';

  const openPlan = () => {
    if (match?.open_date_id) router.push(`/date/${match.open_date_id}`);
    else setPlanOpen(true);
  };

  const leave = () => {
    live.refreshMatches();
    live.refreshDates();
    live.refreshProposals();
    if (router.canGoBack()) router.back();
    else router.replace('/matches');
  };

  const rows = useMemo<Row[]>(() => {
    const out: Row[] = [];
    if (typing) out.push({ type: 'typing', key: 'typing' });
    for (let i = pending.length - 1; i >= 0; i--) out.push({ type: 'pending', key: pending[i].client_id, pending: pending[i] });
    let readMarked = false;
    for (let i = messages.length - 1; i >= 0; i--) {
      const m = messages[i];
      const mine = m.sender_id === user?.id;
      // "Read" appears once, under the newest of my messages they've seen.
      const showRead = mine && !!m.read_at && !readMarked;
      if (showRead) readMarked = true;
      out.push({ type: 'msg', key: String(m.seq), msg: m, showRead });
    }
    return out;
  }, [messages, pending, typing, user?.id]);

  if (loadError && !match) {
    return (
      <Backdrop>
        <View style={{ paddingTop: insets.top + space.xs, paddingHorizontal: space.sm }}>
          <IconButton icon="chevron-back" label="Back" onPress={() => router.back()} />
        </View>
        <View style={{ flex: 1, justifyContent: 'center' }}>
          <SukiSays face="pouty" title="I can't open this chat" body={loadError}>
            <Button label="Try again" kind="secondary" onPress={loadMatch} />
          </SukiSays>
        </View>
      </Backdrop>
    );
  }

  return (
    <Backdrop>
      <View style={{ paddingTop: insets.top + space.xs }}>
        <View style={{ flexDirection: 'row', alignItems: 'center', paddingHorizontal: space.sm, paddingBottom: space.sm, gap: space.xs }}>
          <IconButton icon="chevron-back" label="Back" onPress={() => (router.canGoBack() ? router.back() : router.replace('/matches'))} />
          <Pressable
            accessibilityRole="button"
            accessibilityLabel={`View ${person?.display_name ?? ''}'s profile`}
            onPress={() => id && router.push(`/person?match=${id}`)}
            style={{ flex: 1, flexDirection: 'row', alignItems: 'center', gap: space.sm }}
          >
            <PersonAvatar id={person?.photo_ids?.[0]} size={42} />
            <View style={{ flex: 1 }}>
              <T v="subheading" numberOfLines={1}>
                {person?.display_name ?? ' '}
              </T>
              <T v="caption" color={typing ? t.accent : t.inkFaint} numberOfLines={1}>
                {typing ? 'Typing...' : 'View profile'}
              </T>
            </View>
          </Pressable>
          <IconButton icon="ellipsis-horizontal" label="More" onPress={() => setMenu(true)} />
        </View>
        <IrisRule style={{ opacity: 0.55 }} />
      </View>

      <KeyboardAvoidingView behavior="padding" style={{ flex: 1 }}>
        <CoachPanel
          hints={hints}
          asking={asking}
          typing={keyboardUp}
          onDismiss={dismiss}
          onPick={(s) => {
            setText(s);
            input.current?.focus();
          }}
        />

        {!loaded ? (
          <View style={{ flex: 1, justifyContent: 'center' }}>
            <SukiLoading line="Opening your chat..." />
          </View>
        ) : (
          <FlatList
            ref={list}
            inverted
            data={rows}
            keyExtractor={(r) => r.key}
            keyboardShouldPersistTaps="handled"
            keyboardDismissMode="on-drag"
            contentContainerStyle={{ padding: space.lg, gap: space.md, flexGrow: 1 }}
            ListEmptyComponent={
              <View style={{ flex: 1, justifyContent: 'center' }}>
                <SukiSays
                  face="playful"
                  size={96}
                  title={`Say hi to ${person?.display_name ?? 'them'}`}
                  body="First messages don't have to be clever. Ask about something on their profile. Stuck? Tap Ask Suki."
                />
              </View>
            }
            renderItem={({ item }) => {
              if (item.type === 'typing') {
                return (
                  <FadeIn>
                    <View style={{ alignSelf: 'flex-start', backgroundColor: t.bubbleOther, borderRadius: radius.md, borderBottomLeftRadius: 6, paddingHorizontal: 16, paddingVertical: 12 }}>
                      <TypingDots />
                    </View>
                  </FadeIn>
                );
              }
              if (item.type === 'pending') {
                const p = item.pending;
                return (
                  <View style={{ alignItems: 'flex-end', gap: 3 }}>
                    <LinearGradient
                      colors={t.brand}
                      start={{ x: 0, y: 0 }}
                      end={{ x: 1, y: 1 }}
                      style={{ maxWidth: '82%', borderRadius: radius.md, borderBottomRightRadius: 6, paddingHorizontal: 14, paddingVertical: 10, opacity: p.state === 'sending' ? 0.75 : 1 }}
                    >
                      <T color={t.onBrand}>{p.text}</T>
                    </LinearGradient>
                    {p.state === 'failed' ? (
                      <View style={{ alignItems: 'flex-end', maxWidth: '85%' }}>
                        {p.error ? (
                          <T v="caption" color={t.bad} style={{ textAlign: 'right' }}>
                            {p.error}
                          </T>
                        ) : null}
                        <View style={{ flexDirection: 'row', gap: space.md }}>
                          <Pressable hitSlop={8} accessibilityRole="button" onPress={() => setPending((a) => a.filter((x) => x.client_id !== p.client_id))}>
                            <T v="caption" soft style={{ fontWeight: '700' }}>
                              Remove
                            </T>
                          </Pressable>
                          <Pressable hitSlop={8} accessibilityRole="button" onPress={() => deliver(p)}>
                            <T v="caption" color={t.accent} style={{ fontWeight: '700' }}>
                              Tap to retry
                            </T>
                          </Pressable>
                        </View>
                      </View>
                    ) : (
                      <T v="caption" faint style={{ marginRight: 4 }}>
                        Sending...
                      </T>
                    )}
                  </View>
                );
              }
              const m = item.msg;
              if (m.role === 'text') return <Bubble msg={m} mine={m.sender_id === user?.id} showRead={item.showRead} />;
              return <SukiNote msg={m} onAnswer={() => input.current?.focus()} />;
            }}
          />
        )}

        <View style={{ paddingHorizontal: space.md, paddingBottom: insets.bottom + space.sm, gap: space.sm }}>
          {ended ? (
            <Glass>
              <T v="small" soft center>
                This chat has ended. I'm sorry. There are more people to meet.
              </T>
            </Glass>
          ) : (
            <>
              <View style={{ flexDirection: 'row', gap: space.sm }}>
                <Button label="Ask Suki" icon="sparkles" small kind="secondary" loading={asking} onPress={askSuki} />
                <Button label={match?.open_date_id ? 'Open our date' : 'Plan a date'} icon="calendar-outline" small kind="secondary" onPress={openPlan} />
              </View>
              <ErrorText message={sendError} />
              <ChatInput ref={input} value={text} onChangeText={onChangeText} onSend={send} placeholder={`Message ${person?.display_name ?? ''}`} />
            </>
          )}
        </View>
      </KeyboardAvoidingView>

      <Sheet visible={menu} onClose={() => setMenu(false)}>
        <SheetAction
          icon="person-outline"
          label="View profile"
          onPress={() => {
            setMenu(false);
            if (id) router.push(`/person?match=${id}`);
          }}
        />
        <SheetAction
          icon="heart-dislike-outline"
          label="Unmatch"
          danger
          onPress={async () => {
            setMenu(false);
            const yes = await confirm(`Unmatch ${person?.display_name ?? ''}?`, 'The chat ends for both of you, and any open date plan is cancelled.', 'Unmatch', true);
            if (!yes || !id) return;
            try {
              await api.matches.unmatch(id);
              leave();
            } catch (e) {
              showError(e);
            }
          }}
        />
        <SheetAction
          icon="ban-outline"
          label="Block"
          danger
          onPress={async () => {
            setMenu(false);
            if (!person) return;
            const yes = await confirm(`Block ${person.display_name}?`, "You'll stop seeing each other, and this match ends right away.", 'Block', true);
            if (!yes) return;
            try {
              await api.safety.block(person.user_id);
              leave();
            } catch (e) {
              showError(e);
            }
          }}
        />
        <SheetAction
          icon="flag-outline"
          label="Report"
          danger
          onPress={() => {
            setMenu(false);
            setReportOpen(true);
          }}
        />
      </Sheet>

      <PlanSheet
        visible={planOpen}
        onClose={() => setPlanOpen(false)}
        name={person?.display_name ?? ''}
        onPlan={async (constraints) => {
          if (!id) return;
          const dateId = await api.matches.startDate(id, constraints);
          setPlanOpen(false);
          live.refreshDates();
          loadMatch();
          router.push(`/date/${dateId}`);
        }}
      />

      <ReportSheet
        visible={reportOpen}
        name={person?.display_name ?? ''}
        onClose={() => setReportOpen(false)}
        onSend={async (reason, details) => {
          if (!person || !id) return;
          await api.safety.report({ user_id: person.user_id, match_id: id, reason, details });
          setReportOpen(false);
          Alert.alert('Thank you for telling us', 'A person on our team will look at this. You can also block them from this menu.');
        }}
      />
    </Backdrop>
  );
}

function PlanSheet({
  visible,
  onClose,
  name,
  onPlan,
}: {
  visible: boolean;
  onClose: () => void;
  name: string;
  onPlan: (constraints: string[]) => Promise<void>;
}) {
  const [note, setNote] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  return (
    <Sheet visible={visible} onClose={onClose} title={`A date with ${name}`}>
      <T soft>I'll find a few places near you both. You each pick a favorite, then choose a time.</T>
      <Field
        label="Anything I should keep in mind?"
        placeholder="Not too loud, under $40"
        value={note}
        onChangeText={setNote}
        maxLength={120}
        hint="Optional."
      />
      <ErrorText message={error} />
      <Button
        label="Find us a place"
        icon="sparkles"
        full
        loading={busy}
        onPress={async () => {
          setBusy(true);
          setError(null);
          try {
            await onPlan(note.trim() ? [note.trim()] : []);
            setNote('');
          } catch (e: any) {
            setError(e.message);
          } finally {
            setBusy(false);
          }
        }}
      />
    </Sheet>
  );
}

function ReportSheet({
  visible,
  name,
  onClose,
  onSend,
}: {
  visible: boolean;
  name: string;
  onClose: () => void;
  onSend: (reason: ReportReason, details: string) => Promise<void>;
}) {
  const t = useTheme();
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  return (
    <Sheet visible={visible} onClose={onClose} title={`Report ${name}`}>
      <T soft>They won't know you reported them. What happened?</T>
      <View style={{ gap: 2 }}>
        {REPORT_REASONS.map((r) => (
          <Pressable
            key={r.value}
            accessibilityRole="radio"
            accessibilityState={{ selected: reason === r.value }}
            onPress={() => setReason(r.value)}
            style={{ paddingVertical: space.sm, flexDirection: 'row', alignItems: 'center', gap: space.md }}
          >
            <View
              style={{
                width: 20,
                height: 20,
                borderRadius: 10,
                borderWidth: reason === r.value ? 6 : 1.5,
                borderColor: reason === r.value ? t.accent : t.inkFaint,
              }}
            />
            <T>{r.label}</T>
          </Pressable>
        ))}
      </View>
      <Field label="Anything else?" value={details} onChangeText={setDetails} multiline maxLength={2000} hint="Optional, and it helps." />
      <ErrorText message={error} />
      <Button
        label="Send report"
        kind="danger"
        full
        disabled={!reason}
        loading={busy}
        onPress={async () => {
          if (!reason) return;
          setBusy(true);
          setError(null);
          try {
            await onSend(reason, details.trim());
            setReason(null);
            setDetails('');
          } catch (e: any) {
            setError(e.message);
          } finally {
            setBusy(false);
          }
        }}
      />
      {/* Kept visible so it's clear reporting and leaving are separate choices. */}
      <Pill tone="plain" label="Reporting doesn't end the match. Block or unmatch for that." />
    </Sheet>
  );
}
