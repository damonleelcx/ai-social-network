import { useEffect, useState } from 'react';
import { View } from 'react-native';
import * as api from '@/api/endpoints';
import { useSession } from '@/state/session';
import { space } from '@/theme/tokens';
import { SingleSelect, ToggleRow } from '@/ui/form';
import { Button, ErrorText, Field } from '@/ui/primitives';
import { Sheet } from '@/ui/sheet';
import { tr } from '@/i18n';

export const BUDDY_KINDS = [
  { value: 'movie', label: 'Movie' },
  { value: 'meal', label: 'Meal' },
  { value: 'show', label: 'Show' },
  { value: 'travel', label: 'Travel' },
  { value: 'sports', label: 'Sports' },
  { value: 'games', label: 'Games' },
  { value: 'business', label: 'Business' },
  { value: 'study', label: 'Study' },
  { value: 'other', label: 'Something else' },
] as const;

export function buddyKindLabel(kind: string) {
  return tr(BUDDY_KINDS.find((k) => k.value === kind)?.label ?? 'Something else');
}

/** Roughly when the plan will have passed. It's when the post comes down, nothing more. */
const WHEN = [
  { value: 'Tonight', label: 'Tonight', hours: 12 },
  { value: 'Tomorrow', label: 'Tomorrow', hours: 36 },
  { value: 'This weekend', label: 'This weekend', hours: 7 * 24 },
  { value: 'In the next two weeks', label: 'In the next two weeks', hours: 14 * 24 },
] as const;

export function BuddyPostSheet({ visible, onClose, onPosted }: { visible: boolean; onClose: () => void; onPosted: () => void }) {
  const { profile } = useSession();
  const [kind, setKind] = useState('movie');
  const [when, setWhen] = useState<string>('Tonight');
  const [whenText, setWhenText] = useState('');
  const [note, setNote] = useState('');
  const [sameGender, setSameGender] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (visible) setWhenText(tr(when));
    // Only the moment it opens: after that the words are theirs to change.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [visible]);

  const post = async () => {
    setBusy(true);
    setError(null);
    try {
      await api.buddies.post({
        kind,
        when: whenText.trim(),
        note: note.trim(),
        expires_in_hours: WHEN.find((w) => w.value === when)?.hours ?? 24,
        same_gender: sameGender,
      });
      setNote('');
      onPosted();
    } catch (e: any) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <Sheet visible={visible} onClose={onClose} title={tr('Post a plan')}>
      <View style={{ gap: space.md }}>
        <SingleSelect label={tr('What for?')} options={BUDDY_KINDS} value={kind} onChange={setKind} />
        <SingleSelect
          label={tr('When?')}
          options={WHEN}
          value={when}
          onChange={(v) => {
            setWhen(v);
            setWhenText(tr(v));
          }}
        />
        <Field label={tr('In your words')} value={whenText} onChangeText={setWhenText} maxLength={60} hint={tr('Add the time if you know it, like "Tonight after 7".')} />
        <Field label={tr('What do you have in mind?')} value={note} onChangeText={setNote} multiline maxLength={200} hint={tr('Everyone in your city can read this. Keep contact details out of it.')} />
        <ToggleRow label={tr('Only people of my gender')} value={sameGender} onChange={setSameGender} disabled={!profile?.gender} />
        <ErrorText message={error} />
        <Button label={tr('Post it')} icon="paper-plane" full loading={busy} disabled={!whenText.trim()} onPress={post} />
      </View>
    </Sheet>
  );
}
