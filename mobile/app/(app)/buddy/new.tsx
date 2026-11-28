import { router } from 'expo-router';
import { useState } from 'react';
import { View } from 'react-native';
import * as api from '@/api/endpoints';
import { KINDS } from '@/features/BuddyCards';
import { tr } from '@/i18n';
import { space } from '@/theme/tokens';
import { SingleSelect, ToggleRow } from '@/ui/form';
import { Backdrop, Page, ScreenHeader } from '@/ui/layout';
import { Button, ErrorText, Field, Glass, T } from '@/ui/primitives';

const HOURS = [
  { value: '12', label: 'About tonight' },
  { value: '24', label: 'A day' },
  { value: '72', label: 'A few days' },
  { value: '168', label: 'A week' },
  { value: '336', label: 'Two weeks' },
] as const;

const SPOTS = ['1', '2', '3', '5', '8'] as const;

const KINDS_OPTIONS = Object.entries(KINDS).map(([value, label]) => ({ value, label }));

export default function NewBuddy() {
  const [idea, setIdea] = useState('');
  const [kind, setKind] = useState('meal');
  const [when, setWhen] = useState('');
  const [note, setNote] = useState('');
  const [place, setPlace] = useState('');
  const [hours, setHours] = useState('24');
  const [spots, setSpots] = useState('1');
  const [same, setSame] = useState(false);
  const [drafting, setDrafting] = useState(false);
  const [posting, setPosting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const fill = async () => {
    const text = idea.trim();
    if (!text) {
      setError(tr('Say what you want to do, in a sentence.'));
      return;
    }
    setDrafting(true);
    setError(null);
    try {
      const draft = await api.buddies.draft(text);
      if (draft.kind) setKind(draft.kind);
      if (draft.when) setWhen(draft.when);
      if (draft.note) setNote(draft.note);
      if (draft.place) setPlace(draft.place);
      if (draft.expires_in_hours) setHours(String(nearestHours(draft.expires_in_hours)));
      if (draft.spots) setSpots(String(nearestSpots(draft.spots)));
      setSame(!!draft.same_gender);
    } catch (e: any) {
      setError(tr(e?.message || "I couldn't draft that just now. You can fill the form yourself."));
    } finally {
      setDrafting(false);
    }
  };

  const post = async () => {
    if (!when.trim()) {
      setError(tr('Say when, in a few words.'));
      return;
    }
    setPosting(true);
    setError(null);
    try {
      await api.buddies.post({
        kind,
        when: when.trim(),
        note: note.trim(),
        place: place.trim() || undefined,
        expires_in_hours: Number(hours) || 24,
        same_gender: same,
        spots: Number(spots) || 1,
      });
      router.back();
    } catch (e: any) {
      setError(tr(e?.message || 'Something went wrong. Please try again.'));
      setPosting(false);
    }
  };

  return (
    <Backdrop>
      <ScreenHeader title={tr('Post a plan')} />
      <Page>
        <Glass style={{ gap: space.md }}>
          <T v="subheading">{tr('What do you want to do?')}</T>
          <T v="small" soft>{tr('One sentence is enough. I can fill in the rest, and nothing is posted until you say so.')}</T>
          <Field
            label={tr('Your idea')}
            value={idea}
            onChangeText={setIdea}
            placeholder={tr('Ramen this Friday, two people')}
            multiline
          />
          <Button label={tr('Fill it in for me')} icon="sparkles-outline" kind="secondary" small loading={drafting} onPress={fill} />
        </Glass>

        <Glass style={{ gap: space.lg }}>
          <SingleSelect label={tr('Kind')} options={KINDS_OPTIONS} value={kind} onChange={setKind} />
          <Field label={tr('When')} value={when} onChangeText={setWhen} placeholder={tr('Saturday evening')} />
          <Field label={tr('Note')} value={note} onChangeText={setNote} placeholder={tr('What you want to do')} multiline />
          <Field label={tr('Place')} value={place} onChangeText={setPlace} placeholder={tr('Optional')} />
          <SingleSelect label={tr('How long it stays up')} options={HOURS} value={hours} onChange={setHours} />
          <SingleSelect label={tr('How many can join')} options={SPOTS.map((value) => ({ value, label: value }))} value={spots} onChange={setSpots} />
          <ToggleRow label={tr('Only people of my gender')} value={same} onChange={setSame} />
          <ErrorText message={error} />
          <Button label={tr('Post it')} icon="paper-plane" loading={posting} onPress={post} />
        </Glass>
        <View />
      </Page>
    </Backdrop>
  );
}

function nearestHours(n: number) {
  let best: string = HOURS[0].value;
  for (const h of HOURS) {
    if (Math.abs(Number(h.value) - n) < Math.abs(Number(best) - n)) best = h.value;
  }
  return best;
}

function nearestSpots(n: number) {
  let best: string = SPOTS[0];
  for (const s of SPOTS) {
    if (Math.abs(Number(s) - n) < Math.abs(Number(best) - n)) best = s;
  }
  return best;
}
