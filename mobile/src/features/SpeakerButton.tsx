import { Ionicons } from '@expo/vector-icons';
import { ActivityIndicator, Pressable } from 'react-native';
import { voice, useVoice, type ClipKey } from '@/lib/voice';
import { useTheme } from '@/theme/ThemeProvider';
import { Button } from '@/ui/primitives';
import { showError } from '@/ui/sheet';
import { tr } from '@/i18n';

function toggle(key: ClipKey, active: boolean) {
  if (active) voice.stop();
  else voice.play(key, { tapped: true }).catch(showError);
}

/** The small speaker on a Suki bubble. Tap to hear her, tap again to stop. */
export function SpeakerButton({ seq }: { seq: number }) {
  const t = useTheme();
  const { clip, phase } = useVoice();
  const active = clip === seq;
  const loading = active && phase === 'loading';
  const playing = active && phase === 'playing';
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={loading ? tr('Getting Suki’s voice. Tap to cancel') : playing ? tr('Stop') : tr('Hear Suki say this')}
      accessibilityState={{ busy: loading }}
      hitSlop={10}
      onPress={() => toggle(seq, active)}
      style={({ pressed }) => ({
        width: 26,
        height: 26,
        borderRadius: 13,
        alignItems: 'center',
        justifyContent: 'center',
        backgroundColor: active ? t.accentSoft : 'transparent',
        opacity: pressed ? 0.6 : 1,
      })}
    >
      {loading ? (
        <ActivityIndicator size="small" color={t.accent} />
      ) : (
        <Ionicons name={playing ? 'stop' : 'volume-medium-outline'} size={playing ? 14 : 17} color={playing ? t.accent : t.inkFaint} />
      )}
    </Pressable>
  );
}

/** Settings' "Hear Suki" button: the fixed greeting line. */
export function HearSukiButton() {
  const { clip, phase } = useVoice();
  const active = clip === 'sample';
  return (
    <Button
      label={active && phase === 'playing' ? tr('Stop') : tr('Hear Suki')}
      icon={active && phase === 'playing' ? 'stop' : 'volume-medium-outline'}
      kind="secondary"
      small
      loading={active && phase === 'loading'}
      onPress={() => toggle('sample', active)}
    />
  );
}
