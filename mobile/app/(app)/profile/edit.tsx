import DateTimePicker, { DateTimePickerAndroid } from '@react-native-community/datetimepicker';
import { router } from 'expo-router';
import { useEffect, useState } from 'react';
import { Platform, Pressable, View } from 'react-native';
import * as api from '@/api/endpoints';
import type { Profile, Prompt } from '@/api/types';
import { fullDate, GENDERS, GOALS, HABITS, KIDS, WANTS_KIDS, ymd } from '@/lib/format';
import { useSession } from '@/state/session';
import { useTheme } from '@/theme/ThemeProvider';
import { space } from '@/theme/tokens';
import { MultiSelect, SingleSelect, TagInput } from '@/ui/form';
import { Backdrop, Page, ScreenHeader, Section } from '@/ui/layout';
import { Button, ErrorText, Field, Glass, IconButton, Pill, T } from '@/ui/primitives';
import { confirm, Sheet, showError } from '@/ui/sheet';
import { SukiAvatar, SukiLoading } from '@/ui/suki';

const PROMPT_IDEAS = [
  'My ideal Sunday',
  'The way to my heart is',
  "I'm weirdly good at",
  'A green flag I look for',
  'Two truths and a lie',
  'Something I want to try this year',
];

type Form = {
  display_name: string;
  pronouns: string;
  bio: string;
  occupation: string;
  education: string;
  religion: string;
  politics: string;
  gender: string;
  seeking: string[];
  age_min: string;
  age_max: string;
  goal: string;
  height_cm: string;
  smoking: string;
  drinking: string;
  kids: string;
  wants_kids: string;
  languages: string[];
  interests: string[];
  prompts: Prompt[];
  dealbreakers: Record<string, string[]>;
};

function toForm(p: Profile): Form {
  return {
    display_name: p.display_name ?? '',
    pronouns: p.pronouns ?? '',
    bio: p.bio ?? '',
    occupation: p.occupation ?? '',
    education: p.education ?? '',
    religion: p.religion ?? '',
    politics: p.politics ?? '',
    gender: p.gender ?? '',
    seeking: p.seeking ?? [],
    age_min: p.age_min ? String(p.age_min) : '',
    age_max: p.age_max ? String(p.age_max) : '',
    goal: p.goal ?? '',
    height_cm: p.height_cm ? String(p.height_cm) : '',
    smoking: p.smoking ?? '',
    drinking: p.drinking ?? '',
    kids: p.kids ?? '',
    wants_kids: p.wants_kids ?? '',
    languages: p.languages ?? [],
    interests: p.interests ?? [],
    prompts: p.prompts ?? [],
    dealbreakers: p.dealbreakers ?? {},
  };
}

const same = (a: unknown, b: unknown) => JSON.stringify(a) === JSON.stringify(b);

/** PATCH only what changed. An enum can't be set back to empty, so an unset one is never sent. */
function diff(before: Form, after: Form) {
  const patch: Record<string, unknown> = {};
  (Object.keys(after) as (keyof Form)[]).forEach((k) => {
    if (same(before[k], after[k])) return;
    if (k === 'age_min' || k === 'age_max' || k === 'height_cm') {
      if (after[k] !== '') patch[k] = Number(after[k]);
    } else if (k === 'prompts') {
      patch[k] = after.prompts.map((p) => ({ q: p.q.trim(), a: p.a.trim() })).filter((p) => p.q && p.a);
    } else if (k === 'dealbreakers') {
      patch[k] = Object.fromEntries(Object.entries(after.dealbreakers).filter(([, v]) => v.length));
    } else {
      patch[k] = after[k];
    }
  });
  return patch;
}

function Birthdate({ value, onSaved }: { value: string; onSaved: () => void }) {
  const t = useTheme();
  const [iosOpen, setIosOpen] = useState(false);
  const eighteenAgo = new Date();
  eighteenAgo.setFullYear(eighteenAgo.getFullYear() - 18);
  const [draft, setDraft] = useState(eighteenAgo);

  const save = async (d: Date) => {
    const s = ymd(d);
    const yes = await confirm(
      `Is ${fullDate(s)} right?`,
      "You can only set your birthday once, so please check it. SynAgent is for people 18 and over.",
      'Yes, save it',
    );
    if (!yes) return;
    try {
      await api.profile.patch({ birthdate: s });
      onSaved();
    } catch (e) {
      showError(e);
      onSaved();
    }
  };

  if (value) {
    return (
      <View style={{ gap: 4 }}>
        <T v="small" soft style={{ fontWeight: '600', marginLeft: 4 }}>
          Birthday
        </T>
        <T style={{ marginLeft: 4 }}>{fullDate(value)}</T>
        <T v="caption" faint style={{ marginLeft: 4 }}>
          Set once. Contact support if it's wrong.
        </T>
      </View>
    );
  }
  return (
    <View style={{ gap: 6 }}>
      <T v="small" soft style={{ fontWeight: '600', marginLeft: 4 }}>
        Birthday
      </T>
      <Pressable
        accessibilityRole="button"
        accessibilityLabel="Set your birthday"
        onPress={() => {
          if (Platform.OS === 'ios') setIosOpen(true);
          else {
            DateTimePickerAndroid.open({
              value: eighteenAgo,
              mode: 'date',
              maximumDate: new Date(),
              onChange: (e, d) => {
                if (e.type === 'set' && d) save(d);
              },
            });
          }
        }}
        style={{ paddingVertical: space.sm, paddingHorizontal: 4 }}
      >
        <T color={t.accent} style={{ fontWeight: '700' }}>
          Set your birthday
        </T>
      </Pressable>
      {Platform.OS === 'ios' ? (
        <Sheet visible={iosOpen} onClose={() => setIosOpen(false)} title="Your birthday">
          <DateTimePicker value={draft} mode="date" display="spinner" maximumDate={new Date()} onChange={(_, d) => d && setDraft(d)} />
          <Button
            label="Continue"
            full
            onPress={() => {
              setIosOpen(false);
              save(draft);
            }}
          />
        </Sheet>
      ) : null}
    </View>
  );
}

function City({ name, onSaved }: { name: string; onSaved: () => void }) {
  const [query, setQuery] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  return (
    <View style={{ gap: space.sm }}>
      <Field
        label="City"
        placeholder={name || 'Where do you live?'}
        value={query}
        onChangeText={setQuery}
        hint={name ? `Now: ${name}` : 'Add the country if I can’t find it.'}
        autoCorrect={false}
      />
      {query.trim() ? (
        <Button
          label="Set this city"
          small
          kind="secondary"
          loading={busy}
          onPress={async () => {
            setBusy(true);
            setError(null);
            try {
              await api.profile.setCity(query.trim());
              setQuery('');
              onSaved();
            } catch (e: any) {
              setError(e.message);
            } finally {
              setBusy(false);
            }
          }}
        />
      ) : null}
      <ErrorText message={error} />
    </View>
  );
}

export default function EditProfile() {
  const t = useTheme();
  const { refreshMe } = useSession();
  const [profile, setProfile] = useState<Profile | null>(null);
  const [before, setBefore] = useState<Form | null>(null);
  const [form, setForm] = useState<Form | null>(null);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [acceptingBio, setAcceptingBio] = useState(false);

  const load = async () => {
    try {
      const { profile: p } = await api.profile.get();
      const f = toForm(p);
      setProfile(p);
      setBefore(f);
      // Keep edits in progress if this is a reload after a sub-action.
      setForm((cur) => (cur && before ? { ...f, ...diffToForm(before, cur) } : f));
      setError(null);
    } catch (e: any) {
      setError(e.message);
    }
  };

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  if (!form || !before || !profile) {
    return (
      <Backdrop>
        <ScreenHeader title="Edit profile" />
        <View style={{ flex: 1, justifyContent: 'center', padding: space.xl }}>
          {error ? <ErrorText message={error} /> : <SukiLoading line="Getting your profile..." />}
        </View>
      </Backdrop>
    );
  }

  const set = <K extends keyof Form>(k: K, v: Form[K]) => setForm((f) => (f ? { ...f, [k]: v } : f));
  const patch = diff(before, form);
  const dirty = Object.keys(patch).length > 0;

  const save = async () => {
    setSaving(true);
    setError(null);
    try {
      await api.profile.patch(patch);
      await refreshMe();
      router.back();
    } catch (e: any) {
      setError(e.message);
    } finally {
      setSaving(false);
    }
  };

  const afterSubAction = () => {
    load();
    refreshMe().catch(() => {});
  };

  const dealbreaker = (field: string) => form.dealbreakers[field] ?? [];
  const setDealbreaker = (field: string, v: string[]) => set('dealbreakers', { ...form.dealbreakers, [field]: v });

  return (
    <Backdrop>
      <ScreenHeader title="Edit profile" />
      <Page>
        <Glass style={{ flexDirection: 'row', gap: space.md, alignItems: 'center' }}>
          <SukiAvatar mood="playful" size={44} />
          <T v="small" soft style={{ flex: 1 }}>
            You can also just tell me. "I'm a nurse now" works fine.
          </T>
        </Glass>

        <Section title="About you">
          <Glass style={{ gap: space.lg }}>
            <Field label="Name" value={form.display_name} onChangeText={(v) => set('display_name', v)} maxLength={40} autoCapitalize="words" />
            <Field label="Pronouns" value={form.pronouns} onChangeText={(v) => set('pronouns', v)} maxLength={30} autoCapitalize="none" />
            <Birthdate value={profile.birthdate} onSaved={afterSubAction} />
            <SingleSelect label="Gender" options={GENDERS} value={form.gender} onChange={(v) => set('gender', v)} />
            <City name={profile.city_name} onSaved={afterSubAction} />
            <Field label="Work" value={form.occupation} onChangeText={(v) => set('occupation', v)} maxLength={80} />
            <Field label="Studied" value={form.education} onChangeText={(v) => set('education', v)} maxLength={80} />
          </Glass>
        </Section>

        <Section title="Bio">
          <Glass style={{ gap: space.md }}>
            {profile.bio_draft ? (
              <View style={{ gap: space.sm, padding: space.md, borderRadius: 14, backgroundColor: t.cardTint }}>
                <Pill label="Suki's draft" icon="sparkles" />
                <T v="small" style={{ fontStyle: 'italic' }}>
                  {profile.bio_draft}
                </T>
                <Button
                  label="Use this"
                  small
                  loading={acceptingBio}
                  onPress={async () => {
                    setAcceptingBio(true);
                    try {
                      await api.profile.acceptBio();
                      setForm((f) => (f ? { ...f, bio: profile.bio_draft } : f));
                      afterSubAction();
                    } catch (e) {
                      showError(e);
                    } finally {
                      setAcceptingBio(false);
                    }
                  }}
                />
              </View>
            ) : null}
            <Field value={form.bio} onChangeText={(v) => set('bio', v)} multiline maxLength={600} placeholder="A few lines that sound like you" accessibilityLabel="Bio" />
          </Glass>
        </Section>

        <Section title="Who you want to meet">
          <Glass style={{ gap: space.lg }}>
            <MultiSelect label="I'd like to meet" options={GENDERS} value={form.seeking} onChange={(v) => set('seeking', v)} />
            <View style={{ flexDirection: 'row', gap: space.md }}>
              <View style={{ flex: 1 }}>
                <Field label="Youngest" keyboardType="number-pad" value={form.age_min} maxLength={3} onChangeText={(v) => set('age_min', v.replace(/\D/g, ''))} />
              </View>
              <View style={{ flex: 1 }}>
                <Field label="Oldest" keyboardType="number-pad" value={form.age_max} maxLength={3} onChangeText={(v) => set('age_max', v.replace(/\D/g, ''))} />
              </View>
            </View>
            <SingleSelect label="Looking for" options={GOALS} value={form.goal} onChange={(v) => set('goal', v)} />
          </Glass>
        </Section>

        <Section title="Lifestyle">
          <Glass style={{ gap: space.lg }}>
            <Field label="Height (cm)" keyboardType="number-pad" value={form.height_cm} maxLength={3} onChangeText={(v) => set('height_cm', v.replace(/\D/g, ''))} />
            <SingleSelect label="Smoking" options={HABITS} value={form.smoking} onChange={(v) => set('smoking', v)} />
            <SingleSelect label="Drinking" options={HABITS} value={form.drinking} onChange={(v) => set('drinking', v)} />
            <SingleSelect label="Kids now" options={KIDS} value={form.kids} onChange={(v) => set('kids', v)} />
            <SingleSelect label="Kids in the future" options={WANTS_KIDS} value={form.wants_kids} onChange={(v) => set('wants_kids', v)} />
            <Field label="Religion" value={form.religion} onChangeText={(v) => set('religion', v)} maxLength={40} hint="Optional, and only if you want it to matter." />
            <Field label="Politics" value={form.politics} onChangeText={(v) => set('politics', v)} maxLength={40} hint="Optional, same deal." />
          </Glass>
        </Section>

        <Section title="Languages and interests">
          <Glass style={{ gap: space.lg }}>
            <TagInput label="Languages" placeholder="English, Korean" value={form.languages} onChange={(v) => set('languages', v)} />
            <TagInput label="Interests" placeholder="Climbing, jazz, ramen" value={form.interests} onChange={(v) => set('interests', v)} hint="At least 3, so I have something to work with." />
          </Glass>
        </Section>

        <Section title="Prompts" hint="Up to 6. These are what people actually read.">
          {form.prompts.map((p, i) => (
            <Glass key={i} style={{ gap: space.md }}>
              <View style={{ flexDirection: 'row', alignItems: 'flex-start', gap: space.xs }}>
                <View style={{ flex: 1 }}>
                  <Field
                    label="Question"
                    value={p.q}
                    maxLength={120}
                    onChangeText={(v) => set('prompts', form.prompts.map((x, j) => (j === i ? { ...x, q: v } : x)))}
                  />
                </View>
                <IconButton icon="trash-outline" label="Remove this prompt" onPress={() => set('prompts', form.prompts.filter((_, j) => j !== i))} />
              </View>
              <Field
                label="Your answer"
                value={p.a}
                multiline
                maxLength={300}
                onChangeText={(v) => set('prompts', form.prompts.map((x, j) => (j === i ? { ...x, a: v } : x)))}
              />
            </Glass>
          ))}
          {form.prompts.length < 6 ? (
            <Button
              label="Add a prompt"
              icon="add"
              small
              kind="secondary"
              onPress={() => {
                const used = new Set(form.prompts.map((p) => p.q));
                const q = PROMPT_IDEAS.find((x) => !used.has(x)) ?? '';
                set('prompts', [...form.prompts, { q, a: '' }]);
              }}
            />
          ) : null}
        </Section>

        <Section title="Dealbreakers" hint="I'll never introduce you to someone on the wrong side of these.">
          <Glass style={{ gap: space.lg }}>
            <MultiSelect label="Won't date someone who smokes..." options={HABITS} value={dealbreaker('smoking')} onChange={(v) => setDealbreaker('smoking', v)} />
            <MultiSelect label="Won't date someone who drinks..." options={HABITS} value={dealbreaker('drinking')} onChange={(v) => setDealbreaker('drinking', v)} />
            <MultiSelect label="Won't date someone who..." options={KIDS} value={dealbreaker('kids')} onChange={(v) => setDealbreaker('kids', v)} />
            <MultiSelect label="On future kids, not someone who..." options={WANTS_KIDS} value={dealbreaker('wants_kids')} onChange={(v) => setDealbreaker('wants_kids', v)} />
            <TagInput label="Religions to rule out" placeholder="Type and tap done" value={dealbreaker('religion')} onChange={(v) => setDealbreaker('religion', v)} />
            <TagInput label="Politics to rule out" placeholder="Type and tap done" value={dealbreaker('politics')} onChange={(v) => setDealbreaker('politics', v)} />
          </Glass>
        </Section>

        <ErrorText message={error} />
        <Button label={dirty ? 'Save changes' : 'Nothing to save yet'} full disabled={!dirty} loading={saving} onPress={save} />
      </Page>
    </Backdrop>
  );
}

function diffToForm(before: Form, cur: Form): Partial<Form> {
  const out: Partial<Form> = {};
  (Object.keys(cur) as (keyof Form)[]).forEach((k) => {
    if (!same(before[k], cur[k])) (out as any)[k] = cur[k];
  });
  return out;
}

