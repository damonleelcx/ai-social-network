import { LinearGradient } from 'expo-linear-gradient';
import { router, useLocalSearchParams } from 'expo-router';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { FlatList, KeyboardAvoidingView, ScrollView, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import * as api from '@/api/endpoints';
import type { BuddyRequest, GroupMessage } from '@/api/types';
import { ChatInput } from '@/features/ChatInput';
import { KINDS, groupLine, PersonRow, Plan, planOf } from '@/features/BuddyCards';
import { shortTime, uuid } from '@/lib/format';
import { useForeground, useTopic } from '@/state/live';
import { useSession } from '@/state/session';
import { useTheme } from '@/theme/ThemeProvider';
import { radius, space } from '@/theme/tokens';
import { Backdrop } from '@/ui/layout';
import { PersonAvatar } from '@/ui/person';
import { ErrorText, Glass, IconButton, IrisRule, T } from '@/ui/primitives';
import { confirm, Sheet, SheetAction, showError } from '@/ui/sheet';
import { SukiLoading, SukiSays } from '@/ui/suki';
import { tr } from '@/i18n';

/** What happened in the group, in this person's language. */
function eventLine(m: GroupMessage) {
  const name = m.meta?.name ?? '';
  switch (m.meta?.event) {
    case 'joined':
      return tr('{name} joined.', { name });
    case 'left':
      return tr('{name} left.', { name });
    case 'called_off':
      return tr('{name} called off the plan.', { name });
    default:
      return m.body;
  }
}

const merge = (a: GroupMessage[], b: GroupMessage[]) => {
  const map = new Map<number, GroupMessage>();
  for (const m of a) map.set(m.seq, m);
  for (const m of b) map.set(m.seq, m);
  return [...map.values()].sort((x, y) => x.seq - y.seq);
};

function Bubble({ msg, mine, photo }: { msg: GroupMessage; mine: boolean; photo?: string }) {
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
        <T v="caption" faint style={{ marginRight: 4 }}>{shortTime(msg.created_at)}</T>
      </View>
    );
  }
  return (
    <View style={{ flexDirection: 'row', alignItems: 'flex-end', gap: space.sm }}>
      <PersonAvatar id={photo} size={28} />
      <View style={{ flex: 1, alignItems: 'flex-start', gap: 3 }}>
        <T v="caption" soft style={{ marginLeft: 4, fontWeight: '700' }}>{msg.meta?.name ?? ''}</T>
        <View
          style={{
            maxWidth: '88%',
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
        <T v="caption" faint style={{ marginLeft: 4 }}>{shortTime(msg.created_at)}</T>
      </View>
    </View>
  );
}

/** A group plan's chat. Suki's card opens it; everyone in the plan is here. */
export default function GroupScreen() {
  const insets = useSafeAreaInsets();
  const { id } = useLocalSearchParams<{ id: string }>();
  const { user } = useSession();
  const [request, setRequest] = useState<BuddyRequest | null>(null);
  const [messages, setMessages] = useState<GroupMessage[]>([]);
  const last = useRef(0);
  const [open, setOpen] = useState(true);
  const [loaded, setLoaded] = useState(false);
  const [gone, setGone] = useState(false);
  const [text, setText] = useState('');
  const [sending, setSending] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [menu, setMenu] = useState(false);
  const [people, setPeople] = useState(false);

  const load = useCallback(async () => {
    if (!id) return;
    try {
      const c = await api.groups.chat(id, last.current);
      setRequest(c.request);
      setOpen(c.open);
      setMessages((prev) => {
        const all = merge(prev, c.messages);
        last.current = all.length ? all[all.length - 1].seq : 0;
        return all;
      });
      setGone(false);
    } catch (e: any) {
      if (e?.status === 404) setGone(true);
      else setError(e.message);
    } finally {
      setLoaded(true);
    }
  }, [id]);

  useEffect(() => {
    load();
  }, [load]);
  useTopic((x) => x === `group:${id}`, load);
  useForeground(load);

  const photoOf = useMemo(() => {
    const m = new Map<string, string | undefined>();
    for (const p of request?.members ?? []) m.set(p.user_id, p.photo_ids?.[0]);
    return m;
  }, [request]);

  const mine = !!request?.person && request.person.user_id === user?.id;

  const send = async () => {
    const body = text.trim();
    if (!body || !id) return;
    setSending(true);
    setError(null);
    try {
      const m = await api.groups.send(id, body, uuid());
      setText('');
      setMessages((prev) => merge(prev, [m]));
    } catch (e: any) {
      setError(e.message);
    } finally {
      setSending(false);
    }
  };

  const leave = async () => {
    setMenu(false);
    if (!id || !(await confirm(tr('Leave the group?'), tr("You'll leave its group chat too."), tr('Leave'), true))) return;
    try {
      await api.buddies.withdraw(id);
      router.back();
    } catch (e) {
      showError(e);
    }
  };

  const callOff = async () => {
    setMenu(false);
    if (!id || !(await confirm(tr('Call off this plan?'), tr('Everyone who joined is told, and the group chat closes.'), tr('Call it off'), true))) return;
    try {
      await api.buddies.close(id);
      await load();
    } catch (e) {
      showError(e);
    }
  };

  const back = () => (router.canGoBack() ? router.back() : router.replace('/suki'));

  if (gone) {
    return (
      <Backdrop>
        <View style={{ paddingTop: insets.top + space.xs, paddingHorizontal: space.sm }}>
          <IconButton icon="chevron-back" label={tr('Back')} onPress={back} />
        </View>
        <View style={{ flex: 1, justifyContent: 'center' }}>
          <SukiSays face="pouty" title={tr("I can't open this group")} body={tr("You're not in this group any more, or the plan has ended.")} />
        </View>
      </Backdrop>
    );
  }

  return (
    <Backdrop>
      <View style={{ paddingTop: insets.top + space.xs }}>
        <View style={{ flexDirection: 'row', alignItems: 'center', paddingHorizontal: space.sm, paddingBottom: space.sm, gap: space.xs }}>
          <IconButton icon="chevron-back" label={tr('Back')} onPress={back} />
          <View style={{ flex: 1 }}>
            <T v="subheading" numberOfLines={1}>
              {request ? `${tr(KINDS[request.kind] ?? 'Something else')} · ${request.when}` : ' '}
            </T>
            <T v="caption" soft numberOfLines={1}>
              {request ? groupLine(request.going, request.spots) : ' '}
            </T>
          </View>
          <IconButton icon="people-outline" label={tr("Who's going")} onPress={() => setPeople(true)} />
          <IconButton icon="ellipsis-horizontal" label={tr('More')} onPress={() => setMenu(true)} />
        </View>
        <IrisRule style={{ opacity: 0.55 }} />
      </View>

      <KeyboardAvoidingView behavior="padding" style={{ flex: 1 }}>
        {!loaded ? (
          <View style={{ flex: 1, justifyContent: 'center' }}>
            <SukiLoading line={tr('Opening your chat...')} />
          </View>
        ) : (
          <FlatList
            inverted
            data={[...messages].reverse()}
            keyExtractor={(m) => String(m.seq)}
            keyboardShouldPersistTaps="handled"
            keyboardDismissMode="on-drag"
            contentContainerStyle={{ padding: space.lg, gap: space.md, flexGrow: 1 }}
            ListFooterComponent={request ? <Glass style={{ marginBottom: space.md }}><Plan {...planOf(request, true)} /></Glass> : undefined}
            renderItem={({ item }) =>
              item.role === 'text' ? (
                <Bubble msg={item} mine={item.sender_id === user?.id} photo={item.sender_id ? photoOf.get(item.sender_id) : undefined} />
              ) : (
                <T v="caption" faint center style={{ paddingHorizontal: space.xl }}>{eventLine(item)}</T>
              )
            }
          />
        )}
        <View style={{ paddingHorizontal: space.md, paddingBottom: insets.bottom + space.sm, gap: space.sm }}>
          {open ? (
            <>
              <ErrorText message={error} />
              <ChatInput value={text} onChangeText={setText} onSend={send} disabled={sending} placeholder={tr('Message the group')} />
            </>
          ) : (
            <Glass>
              <T v="small" soft center>{tr('This group chat has closed.')}</T>
            </Glass>
          )}
        </View>
      </KeyboardAvoidingView>

      <Sheet visible={people} onClose={() => setPeople(false)}>
        <ScrollView style={{ maxHeight: 420 }} contentContainerStyle={{ gap: space.md, padding: space.md }}>
          <T v="small" soft>{tr('Report or block anyone here from their menu. Someone you block disappears from this chat for you.')}</T>
          {(request?.members ?? []).filter((p) => p.user_id !== user?.id).map((p) => (
            <PersonRow key={p.user_id} person={p} onBlocked={load} />
          ))}
        </ScrollView>
      </Sheet>
      <Sheet visible={menu} onClose={() => setMenu(false)}>
        {mine ? (
          open ? <SheetAction icon="close-circle-outline" label={tr('Call it off')} danger onPress={callOff} /> : null
        ) : (
          <SheetAction icon="exit-outline" label={tr('Leave the group')} danger onPress={leave} />
        )}
        <SheetAction
          icon="sparkles-outline"
          label={tr('Ask Suki for ideas')}
          onPress={() => {
            setMenu(false);
            router.navigate('/suki');
          }}
        />
      </Sheet>
    </Backdrop>
  );
}
