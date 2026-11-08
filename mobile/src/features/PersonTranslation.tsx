import { useEffect, useState } from 'react';
import { Pressable, View } from 'react-native';
import * as api from '@/api/endpoints';
import type { Person } from '@/api/types';
import { currentLanguage, tr } from '@/i18n';
import { space } from '@/theme/tokens';
import { T } from '@/ui/primitives';

/**
 * What someone wrote about themselves, in this reader's language. The server translates it once and
 * keeps it; until it arrives, and if it fails, the person reads as they wrote themselves.
 */
export function usePersonInLanguage(person: Person | undefined) {
  const [words, setWords] = useState<api.PersonWords | null>(null);
  const [original, setOriginal] = useState(false);
  const lang = currentLanguage();
  const id = person?.user_id;
  useEffect(() => {
    setWords(null);
    if (!id) return;
    let live = true;
    api.people
      .translation(id, lang)
      .then((r) => live && r.translated && setWords(r.words))
      .catch(() => {});
    return () => {
      live = false;
    };
  }, [id, lang]);
  const shown: Person | undefined =
    person && words && !original
      ? {
          ...person,
          bio: words.bio ?? person.bio,
          city_name: words.city_name ?? person.city_name,
          occupation: words.occupation ?? person.occupation,
          education: words.education ?? person.education,
          interests: words.interests ?? person.interests,
          prompts: words.prompts ?? person.prompts,
        }
      : person;
  return { shown, translated: !!words, original, toggle: () => setOriginal((o) => !o) };
}

/** Says Suki translated it, with the way back to their own words. */
export function TranslatedNote({ original, toggle }: { original: boolean; toggle: () => void }) {
  return (
    <View style={{ flexDirection: 'row', alignItems: 'center', gap: space.sm, marginLeft: 4 }}>
      <T v="caption" soft>
        {original ? tr('Their own words') : tr('Translated by Suki')}
      </T>
      <Pressable accessibilityRole="button" hitSlop={8} onPress={toggle}>
        <T v="caption" style={{ fontWeight: '700', textDecorationLine: 'underline' }}>
          {original ? tr('Show translation') : tr('Show original')}
        </T>
      </Pressable>
    </View>
  );
}
