import { useCallback, useEffect, useState } from 'react';
import { Pressable, View } from 'react-native';
import * as api from '@/api/endpoints';
import { GENDERS, GOALS, HABITS, KIDS, labelOf, WANTS_KIDS } from '@/lib/format';
import { useLive } from '@/state/live';
import { useTheme } from '@/theme/ThemeProvider';
import { space } from '@/theme/tokens';
import { Button, Glass, T } from '@/ui/primitives';
import { showError } from '@/ui/sheet';
import { tr } from '@/i18n';

/** A dealbreaker as words: "Smoking: Yes, Sometimes". */
function redFlag(field: string, values: string[]) {
  const label: Record<string, [string, { value: string; label: string }[] | null]> = {
    smoking: [tr('Smokes'), HABITS],
    drinking: [tr('Drinks'), HABITS],
    kids: [tr('Kids'), KIDS],
    wants_kids: [tr('Future kids'), WANTS_KIDS],
    religion: [tr('Religion'), null],
    politics: [tr('Politics'), null],
  };
  const [name, options] = label[field] ?? [field, null];
  return `${name}: ${values.map((v) => (options ? labelOf(options, v) : v)).join(', ')}`;
}

function Section({ title, empty, children }: { title: string; empty: boolean; children: React.ReactNode }) {
  return (
    <View style={{ gap: 6 }}>
      <T v="label" soft>{title}</T>
      {empty ? <T v="small" faint>{tr('Nothing yet.')}</T> : children}
    </View>
  );
}

function Line({ text, onRemove }: { text: string; onRemove?: () => void }) {
  const t = useTheme();
  return (
    <View style={{ flexDirection: 'row', alignItems: 'center', gap: space.sm }}>
      <T style={{ flex: 1 }}>{text}</T>
      {onRemove ? (
        <Pressable accessibilityRole="button" accessibilityLabel={tr('Remove')} hitSlop={10} onPress={onRemove}>
          <T v="small" color={t.inkFaint}>✕</T>
        </Pressable>
      ) : null}
    </View>
  );
}

/** Something Suki noticed, with their own words under it so they can see why she thinks so. */
function ReadLine({ line, onRemove }: { line: api.LookingLine; onRemove: () => void }) {
  const t = useTheme();
  return (
    <View style={{ flexDirection: 'row', alignItems: 'flex-start', gap: space.sm }}>
      <View style={{ flex: 1, gap: 2 }}>
        <T>{line.text}</T>
        <T v="small" faint style={{ fontStyle: 'italic' }}>{`“${line.said}”`}</T>
      </View>
      <Pressable accessibilityRole="button" accessibilityLabel={tr('Remove')} hitSlop={10} onPress={onRemove}>
        <T v="small" color={t.inkFaint}>✕</T>
      </Pressable>
    </View>
  );
}

/**
 * What Suki understands this person is looking for, from what they told her and nothing else:
 * the basics they set, who draws them in, what they'd steer clear of, and the notes she keeps. One tap removes a line; anything else they change by telling her, which is what the
 * button does.
 */
export function LookingForCard() {
  const { sendToSuki } = useLive();
  const [lf, setLf] = useState<api.LookingFor | null>(null);
  const load = useCallback(() => api.lookingFor.get().then(setLf, () => {}), []);
  useEffect(() => {
    load();
  }, [load]);
  if (!lf) return null;
  const must = [
    lf.must.seeking.length ? tr('Meeting: {who}', { who: lf.must.seeking.map((g) => labelOf(GENDERS, g)).join(', ') }) : '',
    lf.must.age_min && lf.must.age_max ? tr('Aged {min} to {max}', { min: String(lf.must.age_min), max: String(lf.must.age_max) }) : '',
    lf.must.goal ? tr('Looking for: {goal}', { goal: labelOf(GOALS, lf.must.goal) }) : '',
    lf.must.city ? tr('In or near {city}', { city: lf.must.city }) : '',
  ].filter(Boolean);
  const flags = Object.entries(lf.red_flags);
  const read = lf.read ?? [];
  const toward = read.filter((l) => l.kind === 'toward');
  const away = read.filter((l) => l.kind === 'away');
  const remove = (fn: () => Promise<unknown>) => async () => {
    try {
      await fn();
      await load();
    } catch (e) {
      showError(e);
    }
  };
  return (
    <Glass strong style={{ gap: space.md }}>
      <T v="subheading">{tr("What I think you're looking for")}</T>
      <T v="small" soft>{tr("Only from what you've told me. Tap ✕ to take something out, or tell me what to change.")}</T>
      <Section title={tr('The basics')} empty={!must.length}>
        {must.map((m) => (
          <Line key={m} text={m} />
        ))}
      </Section>
      <Section title={tr('Drawn to')} empty={!toward.length}>
        {toward.map((l) => (
          <ReadLine key={l.id} line={l} onRemove={remove(() => api.lookingFor.dropRead(l.id))} />
        ))}
      </Section>
      <Section title={tr('Steering clear of')} empty={!flags.length && !away.length}>
        {away.map((l) => (
          <ReadLine key={l.id} line={l} onRemove={remove(() => api.lookingFor.dropRead(l.id))} />
        ))}
        {flags.map(([field, values]) => (
          <Line key={field} text={redFlag(field, values)} onRemove={remove(() => api.lookingFor.dropRedFlag(field))} />
        ))}
      </Section>
      <Section title={tr('Nice to have')} empty={!lf.nice.length}>
        {lf.nice.map((n) => (
          <Line key={n.id} text={n.note} onRemove={remove(() => api.memories.remove(n.id))} />
        ))}
      </Section>
      <Button label={tr('Change something')} icon="create-outline" kind="secondary" small onPress={() => sendToSuki(tr("I'd like to change what I'm looking for."))} />
    </Glass>
  );
}
