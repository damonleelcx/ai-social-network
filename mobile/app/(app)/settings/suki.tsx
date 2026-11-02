import { router } from 'expo-router';
import { useEffect, useState } from 'react';
import { View } from 'react-native';
import type { Settings } from '@/api/types';
import { HearSukiButton } from '@/features/SpeakerButton';
import { useVoice, voice } from '@/lib/voice';
import { useSession } from '@/state/session';
import { space } from '@/theme/tokens';
import { Group, NavRow, SingleSelect, ToggleRow } from '@/ui/form';
import { Backdrop, Page, ScreenHeader, Section } from '@/ui/layout';
import { Field, Glass, T } from '@/ui/primitives';
import { showError } from '@/ui/sheet';
import { SukiAvatar, SukiLoading } from '@/ui/suki';

const TONES = [
  { value: 'playful', label: 'Playful' },
  { value: 'warm', label: 'Warm' },
  { value: 'direct', label: 'Direct' },
  { value: 'calm', label: 'Calm' },
];
const EMOJI = [
  { value: 'none', label: 'None' },
  { value: 'some', label: 'Some' },
  { value: 'lots', label: 'Lots' },
];
const PROACTIVE = [
  { value: 'low', label: 'Only when I ask' },
  { value: 'normal', label: 'Now and then' },
  { value: 'high', label: 'Keep me posted' },
];

const MOOD_FACE: Record<string, string> = { playful: 'playful', warm: 'happy', direct: 'thinking', calm: 'portrait' };

export default function SukiSettings() {
  const { settings: s, updateSettings } = useSession();
  const set = (patch: Partial<Settings>) => updateSettings(patch).catch(showError);
  const [nickname, setNickname] = useState(s?.suki_nickname ?? '');
  const { enabled: voiceOn } = useVoice();

  useEffect(() => () => voice.stop(), []);

  useEffect(() => {
    if (s) setNickname(s.suki_nickname);
  }, [s?.suki_nickname]); // eslint-disable-line react-hooks/exhaustive-deps

  return (
    <Backdrop>
      <ScreenHeader title="Suki" />
      {!s ? (
        <View style={{ flex: 1, justifyContent: 'center' }}>
          <SukiLoading line="Getting your settings..." />
        </View>
      ) : (
        <Page>
          <Glass style={{ flexDirection: 'row', gap: space.md, alignItems: 'center' }}>
            <SukiAvatar mood={MOOD_FACE[s.suki_tone]} size={64} ring={3} />
            <T v="small" soft style={{ flex: 1 }}>
              These change how I come across, not who I am. I'll always be on your side, and always honest that I'm an AI.
            </T>
          </Glass>

          <Section title="How she talks">
            <Glass style={{ gap: space.lg }}>
              <SingleSelect label="Tone" options={TONES} value={s.suki_tone} onChange={(v) => set({ suki_tone: v as Settings['suki_tone'] })} />
              <SingleSelect label="Emoji" options={EMOJI} value={s.suki_emoji} onChange={(v) => set({ suki_emoji: v as Settings['suki_emoji'] })} />
              <SingleSelect label="How often she checks in" options={PROACTIVE} value={s.suki_proactive} onChange={(v) => set({ suki_proactive: v as Settings['suki_proactive'] })} />
              <Field
                label="What she calls you"
                value={nickname}
                onChangeText={setNickname}
                maxLength={30}
                placeholder="Your name, unless you say otherwise"
                onBlur={() => {
                  if (nickname !== s.suki_nickname) set({ suki_nickname: nickname.trim() });
                }}
              />
            </Glass>
          </Section>

          <Group>
            <ToggleRow label="Avatar moods" hint="Her face changes with what she's saying." value={s.mood_avatar} onChange={(v) => set({ mood_avatar: v })} />
          </Group>

          {voiceOn ? (
            <Section title="Her voice">
              <Group>
                <ToggleRow label="Speaker buttons" hint="Tap the speaker on a message to hear it." value={s.suki_voice} onChange={(v) => set({ suki_voice: v })} />
                <ToggleRow
                  label="Read new messages aloud"
                  hint="Only while you're on the Suki tab, and not when your phone is on silent."
                  value={s.suki_voice && s.suki_voice_auto}
                  disabled={!s.suki_voice}
                  onChange={(v) => set({ suki_voice_auto: v })}
                />
              </Group>
              <HearSukiButton />
            </Section>
          ) : null}

          <Section title="Your chats">
            <Group>
              <ToggleRow label="Coaching" hint="Private hints in your chats. Only you see them." value={s.chat_coach} onChange={(v) => set({ chat_coach: v })} />
              <ToggleRow label="Let Suki read my chats" hint="She reads your messages to give better hints. Off means general tips only." value={s.chat_coach_read} onChange={(v) => set({ chat_coach_read: v })} />
              <ToggleRow label="Chat prompts" hint="She can post a question for you both when a chat goes quiet." value={s.chat_prompts} onChange={(v) => set({ chat_prompts: v })} />
            </Group>
          </Section>

          <Section title="Memory">
            <Group>
              <ToggleRow label="Long-term memory" hint="She keeps notes between chats." value={s.memory_enabled} onChange={(v) => set({ memory_enabled: v })} />
              <ToggleRow label="Date debriefs" hint="She asks how it went after a date." value={s.debrief_enabled} onChange={(v) => set({ debrief_enabled: v })} />
              <NavRow icon="bookmark-outline" label="What Suki remembers" hint="See, delete or start over" onPress={() => router.push('/settings/memory')} />
            </Group>
          </Section>

          <Section title="Safety">
            <Group>
              <NavRow icon="ban-outline" label="Blocked people" onPress={() => router.push('/settings/blocked')} />
            </Group>
          </Section>
        </Page>
      )}
    </Backdrop>
  );
}
