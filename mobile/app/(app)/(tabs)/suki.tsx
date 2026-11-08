import { router, useFocusEffect } from 'expo-router';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { AppState, FlatList, KeyboardAvoidingView, Pressable, ScrollView, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import type { Message } from '@/api/types';
import { ChatInput } from '@/features/ChatInput';
import { SpeakerButton } from '@/features/SpeakerButton';
import { SukiCard } from '@/features/SukiCards';
import { shortTime } from '@/lib/format';
import { useVoice, voice } from '@/lib/voice';
import { useLive, useSukiStreams, type PendingMessage } from '@/state/live';
import { useSession } from '@/state/session';
import { useTheme } from '@/theme/ThemeProvider';
import { radius, space } from '@/theme/tokens';
import { Backdrop } from '@/ui/layout';
import { Chip } from '@/ui/form';
import { IconButton, IrisRule, T } from '@/ui/primitives';
import { FadeIn, SukiAvatar, SukiLoading, SukiSays, TypingDots } from '@/ui/suki';
import { LinearGradient } from 'expo-linear-gradient';
import { tr } from '@/i18n';

type Row =
  | { type: 'msg'; key: string; msg: Message }
  | { type: 'pending'; key: string; pending: PendingMessage };

/** Same height as the time row under a finished bubble, so nothing moves at the handover. */
const META_ROW_HEIGHT = 26;

const STARTERS = ['Hi Suki!', 'Help me set up my profile', 'Find me someone'];

function MyBubble({ text, time, footer }: { text: string; time?: string; footer?: React.ReactNode }) {
  const t = useTheme();
  return (
    <View style={{ alignItems: 'flex-end', gap: 3 }}>
      <LinearGradient
        colors={t.brand}
        start={{ x: 0, y: 0 }}
        end={{ x: 1, y: 1 }}
        style={{ maxWidth: '82%', borderRadius: radius.md, borderBottomRightRadius: 6, paddingHorizontal: 14, paddingVertical: 10 }}
      >
        <T color={t.onBrand}>{text}</T>
      </LinearGradient>
      {footer ??
        (time ? (
          <T v="caption" faint style={{ marginRight: 4 }}>
            {time}
          </T>
        ) : null)}
    </View>
  );
}

function SukiBubble({ msg, moodAvatar, speaker }: { msg: Message; moodAvatar: boolean; speaker: boolean }) {
  const t = useTheme();
  const cards = msg.meta?.cards ?? [];
  return (
    <View style={{ flexDirection: 'row', gap: space.sm, alignItems: 'flex-start' }}>
      <SukiAvatar mood={msg.meta?.mood} moodAvatar={moodAvatar} size={42} />
      <View style={{ flex: 1, gap: space.sm, alignItems: 'flex-start' }}>
        {msg.body ? (
          <View
            style={{
              maxWidth: '94%',
              backgroundColor: t.bubbleOther,
              borderColor: t.glassBorder,
              borderWidth: 1,
              borderRadius: radius.md,
              borderTopLeftRadius: 6,
              paddingHorizontal: 14,
              paddingVertical: 10,
            }}
          >
            <T>{msg.body}</T>
          </View>
        ) : null}
        {cards.map((c, i) => (
          // A card gets the column's width, not its content's: shrunk to fit, a short card became a
          // narrow strip, and Android measured its wrapped text a line short, under the button.
          <View key={`${msg.seq}-${i}`} style={{ alignSelf: 'stretch', maxWidth: 460 }}>
            <SukiCard card={c} />
          </View>
        ))}
        <View style={{ flexDirection: 'row', alignItems: 'center', gap: space.sm, marginLeft: 4, minHeight: META_ROW_HEIGHT }}>
          <T v="caption" faint>
            {shortTime(msg.created_at)}
          </T>
          {speaker && msg.body ? <SpeakerButton seq={msg.seq} /> : null}
        </View>
      </View>
    </View>
  );
}

function StreamCursor() {
  const t = useTheme();
  const [on, setOn] = useState(true);
  useEffect(() => {
    const id = setInterval(() => setOn((v) => !v), 520);
    return () => clearInterval(id);
  }, []);
  return <T color={on ? t.iris[1] : 'transparent'}>▍</T>;
}

/** Her reply as it's being written. Laid out exactly like SukiBubble, which replaces it. */
function StreamBubble({ text }: { text: string }) {
  const t = useTheme();
  return (
    <View style={{ flexDirection: 'row', gap: space.sm, alignItems: 'flex-start' }}>
      <SukiAvatar mood="portrait" size={42} />
      <View style={{ flex: 1, gap: space.sm, alignItems: 'flex-start' }}>
        <View
          style={{
            maxWidth: '94%',
            backgroundColor: t.bubbleOther,
            borderColor: t.glassBorder,
            borderWidth: 1,
            borderRadius: radius.md,
            borderTopLeftRadius: 6,
            paddingHorizontal: 14,
            paddingVertical: 10,
          }}
        >
          <T>
            {text}
            <StreamCursor />
          </T>
        </View>
        <View style={{ height: META_ROW_HEIGHT }} />
      </View>
    </View>
  );
}

/**
 * The newest thing in the chat: replies being written, or the typing indicator. It reads the
 * streams itself, so a new word re-renders this and not the whole message list.
 */
function SukiInProgress({ typing, moodAvatar }: { typing: boolean; moodAvatar: boolean }) {
  const t = useTheme();
  const visible = useSukiStreams().filter((s) => s.text);
  if (visible.length) {
    return (
      <View style={{ gap: space.md }}>
        {visible.map((s) => (
          <StreamBubble key={s.turn} text={s.text} />
        ))}
      </View>
    );
  }
  if (!typing) return null;
  return (
    <FadeIn>
      <View style={{ flexDirection: 'row', gap: space.sm, alignItems: 'center' }}>
        <SukiAvatar mood="thinking" moodAvatar={moodAvatar} size={42} />
        <View style={{ backgroundColor: t.bubbleOther, borderRadius: radius.md, borderTopLeftRadius: 6, paddingHorizontal: 16, paddingVertical: 12 }}>
          <TypingDots />
        </View>
      </View>
    </FadeIn>
  );
}

function EventNote({ text }: { text: string }) {
  return (
    <T v="caption" faint center style={{ paddingHorizontal: space.xl }}>
      {text}
    </T>
  );
}

function FailedFooter({ pending, onRetry, onDiscard }: { pending: PendingMessage; onRetry: () => void; onDiscard: () => void }) {
  const t = useTheme();
  return (
    <View style={{ alignItems: 'flex-end', gap: 2, maxWidth: '85%' }}>
      {pending.error ? (
        <T v="caption" color={t.bad} style={{ textAlign: 'right' }}>
          {pending.error}
        </T>
      ) : null}
      <View style={{ flexDirection: 'row', gap: space.md }}>
        <Pressable onPress={onDiscard} hitSlop={8} accessibilityRole="button">
          <T v="caption" soft style={{ fontWeight: '700' }}>{tr('Remove')}</T>
        </Pressable>
        <Pressable onPress={onRetry} hitSlop={8} accessibilityRole="button">
          <T v="caption" color={t.accent} style={{ fontWeight: '700' }}>{tr('Tap to retry')}</T>
        </Pressable>
      </View>
    </View>
  );
}

export default function SukiChat() {
  const t = useTheme();
  const insets = useSafeAreaInsets();
  const { settings } = useSession();
  const live = useLive();
  const { sukiMessages, sukiPending, sukiThinking, sukiTyping, sukiStreaming, sendToSuki, retryToSuki, discardPending, setSukiFocused } = live;
  const [text, setText] = useState('');
  const list = useRef<FlatList<Row>>(null);
  const moodAvatar = settings?.mood_avatar ?? true;
  const { enabled: voiceOn } = useVoice();
  const showSpeaker = voiceOn && (settings?.suki_voice ?? true);
  // Autoplay needs the speaker buttons too, or there'd be nothing on screen to stop it with.
  const autoplay = showSpeaker && (settings?.suki_voice_auto ?? false);
  const tabFocused = useRef(false);
  const spokenUpTo = useRef<number | null>(null);

  const newestSpeakable = useMemo(() => {
    for (let i = sukiMessages.length - 1; i >= 0; i--) {
      const m = sukiMessages[i];
      if (m.role === 'assistant' && m.body) return m.seq;
    }
    return 0;
  }, [sukiMessages]);
  const newestRef = useRef(newestSpeakable);
  newestRef.current = newestSpeakable;

  useFocusEffect(
    useCallback(() => {
      setSukiFocused(true);
      tabFocused.current = true;
      // Only what arrives from here on is "new". Nothing she said while you were away gets read out.
      if (live.loaded.suki) spokenUpTo.current = newestRef.current;
      return () => {
        setSukiFocused(false);
        tabFocused.current = false;
        voice.stop();
      };
    }, [setSukiFocused, live.loaded.suki]),
  );

  useEffect(() => {
    if (!live.loaded.suki) return;
    if (spokenUpTo.current === null) {
      spokenUpTo.current = newestSpeakable;
      return;
    }
    if (newestSpeakable <= spokenUpTo.current) return;
    spokenUpTo.current = newestSpeakable;
    if (!autoplay || !tabFocused.current || AppState.currentState !== 'active') return;
    // Never talk over a clip the person asked for.
    if (voice.current().tapped) return;
    voice.play(newestSpeakable, { tapped: false }).catch(() => {
      // Autoplay fails quietly: running out of allowance shouldn't interrupt the chat.
    });
  }, [autoplay, live.loaded.suki, newestSpeakable]);

  const typing = sukiThinking || sukiTyping || sukiStreaming;
  const last = sukiMessages[sukiMessages.length - 1];
  const headerMood = sukiStreaming ? 'portrait' : typing ? 'thinking' : ([...sukiMessages].reverse().find((m) => m.role === 'assistant')?.meta?.mood ?? 'happy');

  const rows = useMemo<Row[]>(() => {
    const out: Row[] = [];
    for (let i = sukiPending.length - 1; i >= 0; i--) {
      const p = sukiPending[i];
      out.push({ type: 'pending', key: p.client_id, pending: p });
    }
    for (let i = sukiMessages.length - 1; i >= 0; i--) {
      const m = sukiMessages[i];
      out.push({ type: 'msg', key: String(m.seq), msg: m });
    }
    return out;
  }, [sukiMessages, sukiPending]);

  const replies = !typing && !sukiPending.length && last?.role === 'assistant' ? (last.meta?.replies ?? []) : [];

  const send = (value: string) => {
    if (!value.trim()) return;
    sendToSuki(value);
    setText('');
    list.current?.scrollToOffset({ offset: 0, animated: true });
  };

  return (
    <Backdrop>
      <View style={{ paddingTop: insets.top + space.sm, paddingHorizontal: space.lg, paddingBottom: space.md, flexDirection: 'row', alignItems: 'center', gap: space.md }}>
        <SukiAvatar mood={headerMood} moodAvatar={moodAvatar} size={64} ring={3} />
        <View style={{ flex: 1 }}>
          <T v="title" accessibilityRole="header">
            Suki
          </T>
          {typing ? (
            <T v="small" color={t.accent} style={{ fontWeight: '600' }}>{tr('Thinking about what you said...')}</T>
          ) : (
            <T v="small" soft>{tr('Your winggirl. Openly an AI.')}</T>
          )}
        </View>
        <IconButton icon="options-outline" label={tr("Suki's settings")} filled onPress={() => router.push('/settings/suki')} />
      </View>
      <IrisRule style={{ opacity: 0.6 }} />

      <KeyboardAvoidingView behavior="padding" style={{ flex: 1 }}>
        {!live.loaded.suki && !sukiMessages.length ? (
          <View style={{ flex: 1, justifyContent: 'center' }}>
            <SukiLoading line={tr('Finding our conversation...')} />
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
            ListHeaderComponent={<SukiInProgress typing={typing} moodAvatar={moodAvatar} />}
            ListEmptyComponent={
              <View style={{ flex: 1, justifyContent: 'center' }}>
                <SukiSays
                  face="chibi"
                  size={148}
                  title={tr("Hi, I'm Suki!")}
                  body={tr("I'm an AI matchmaker, and I'm on your side. Say hi and we'll get to know each other. No forms, I promise.")}
                />
              </View>
            }
            renderItem={({ item }) => {
              if (item.type === 'pending') {
                const p = item.pending;
                return (
                  <MyBubble
                    text={p.text}
                    footer={
                      p.state === 'failed' ? (
                        <FailedFooter pending={p} onRetry={() => retryToSuki(p.client_id)} onDiscard={() => discardPending(p.client_id)} />
                      ) : (
                        <T v="caption" faint style={{ marginRight: 4 }}>{tr('Sending...')}</T>
                      )
                    }
                  />
                );
              }
              const m = item.msg;
              if (m.role === 'user') return <MyBubble text={m.body} time={shortTime(m.created_at)} />;
              if (m.role === 'event') return <EventNote text={m.body} />;
              return <SukiBubble msg={m} moodAvatar={moodAvatar} speaker={showSpeaker} />;
            }}
          />
        )}

        <View style={{ paddingHorizontal: space.md, paddingBottom: space.sm, gap: space.sm }}>
          {replies.length || !sukiMessages.length ? (
            <ScrollView horizontal showsHorizontalScrollIndicator={false} keyboardShouldPersistTaps="handled" contentContainerStyle={{ gap: 8, paddingHorizontal: 2 }}>
              {(sukiMessages.length ? replies : STARTERS.map((s) => tr(s))).map((r) => (
                <Chip key={r} label={r} onPress={() => send(r)} />
              ))}
            </ScrollView>
          ) : null}
          <ChatInput value={text} onChangeText={setText} onSend={() => send(text)} placeholder={tr('Tell Suki anything')} />
        </View>
      </KeyboardAvoidingView>
    </Backdrop>
  );
}
