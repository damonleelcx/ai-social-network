import { Ionicons } from '@expo/vector-icons';
import { Image } from 'expo-image';
import { LinearGradient } from 'expo-linear-gradient';
import { useState } from 'react';
import { FlatList, Pressable, View, type StyleProp, type ViewStyle } from 'react-native';
import { useScreenSize } from '@/ui/screenSize';
import { mediaSource } from '@/api/client';
import type { Person } from '@/api/types';
import { GOALS, HABITS, KIDS, labelOf, WANTS_KIDS } from '@/lib/format';
import { useTheme } from '@/theme/ThemeProvider';
import { radius, space } from '@/theme/tokens';
import { Glass, Pill, T } from './primitives';
import { Chip } from './form';
import { tr } from '@/i18n';

/** An authed photo, with a soft placeholder if there's none or it won't load. */
export function MediaImage({
  id,
  style,
  radiusPx = 0,
}: {
  id?: string | null;
  style?: StyleProp<ViewStyle>;
  radiusPx?: number;
}) {
  const t = useTheme();
  const [failed, setFailed] = useState(false);
  return (
    <View style={[{ backgroundColor: t.cardTint, overflow: 'hidden', borderRadius: radiusPx }, style]}>
      {id && !failed ? (
        <Image
          source={mediaSource(id)}
          style={{ width: '100%', height: '100%' }}
          contentFit="cover"
          transition={160}
          cachePolicy="disk"
          accessibilityIgnoresInvertColors
          onError={() => setFailed(true)}
        />
      ) : (
        <View style={{ flex: 1, alignItems: 'center', justifyContent: 'center' }}>
          <Ionicons name="person" size={36} color={t.inkFaint} />
        </View>
      )}
    </View>
  );
}

export function PersonAvatar({ id, size = 56 }: { id?: string | null; size?: number }) {
  const t = useTheme();
  return (
    <LinearGradient
      colors={t.iris}
      start={{ x: 0, y: 0 }}
      end={{ x: 1, y: 1 }}
      style={{ width: size, height: size, borderRadius: size / 2, padding: 2 }}
    >
      <MediaImage id={id} radiusPx={size / 2} style={{ flex: 1 }} />
    </LinearGradient>
  );
}

export function PhotoPager({ ids, height, onPress }: { ids: string[]; height: number; onPress?: () => void }) {
  const { width } = useScreenSize();
  const t = useTheme();
  const [page, setPage] = useState(0);
  const list = ids.length ? ids : [''];
  return (
    <View style={{ height }}>
      <FlatList
        horizontal
        pagingEnabled
        data={list}
        keyExtractor={(x, i) => x || String(i)}
        showsHorizontalScrollIndicator={false}
        onMomentumScrollEnd={(e) => setPage(Math.round(e.nativeEvent.contentOffset.x / width))}
        renderItem={({ item }) => (
          <Pressable onPress={onPress} disabled={!onPress} accessibilityLabel={tr('Photo')}>
            <MediaImage id={item} style={{ width, height }} />
          </Pressable>
        )}
      />
      {list.length > 1 ? (
        <View style={{ position: 'absolute', top: 10, left: 12, right: 12, flexDirection: 'row', gap: 4 }} pointerEvents="none">
          {list.map((_, i) => (
            <View
              key={i}
              style={{
                flex: 1,
                height: 3,
                borderRadius: 2,
                backgroundColor: i === page ? t.onBrand : 'rgba(255,255,255,0.45)',
              }}
            />
          ))}
        </View>
      ) : null}
    </View>
  );
}

export function personLine(p: Person) {
  return [p.age ? `${p.age}` : '', p.pronouns, p.city_name].filter(Boolean).join(' · ');
}

/** Everything about a person below their photos. Words only, never a score. */
export function PersonDetails({ person }: { person: Person }) {
  const details: [string, string][] = (
    [
      [tr('Looking for'), labelOf(GOALS, person.goal)],
      [tr('Work'), person.occupation],
      [tr('Studied'), person.education],
      [tr('Height'), person.height_cm ? tr('{name} cm', { name: person.height_cm }) : ''],
      [tr('Smokes'), labelOf(HABITS, person.smoking)],
      [tr('Drinks'), labelOf(HABITS, person.drinking)],
      [tr('Kids'), labelOf(KIDS, person.kids)],
      [tr('Future kids'), labelOf(WANTS_KIDS, person.wants_kids)],
      [tr('Speaks'), (person.languages ?? []).join(', ')],
    ] as [string, string][]
  ).filter(([, v]) => !!v);
  return (
    <View style={{ gap: space.lg }}>
      {person.bio ? (
        <Glass>
          <T v="body">{person.bio}</T>
        </Glass>
      ) : null}
      {(person.prompts ?? []).map((p, i) => (
        <Glass key={i} style={{ gap: 6 }}>
          <T v="small" soft style={{ fontWeight: '700' }}>
            {p.q}
          </T>
          <T v="heading" style={{ fontWeight: '600' }}>
            {p.a}
          </T>
        </Glass>
      ))}
      {person.interests?.length ? (
        <View style={{ gap: space.sm }}>
          <T v="label" soft style={{ marginLeft: 4 }}>{tr('Into')}</T>
          <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 8 }}>
            {person.interests.map((x) => (
              <Chip key={x} label={x} />
            ))}
          </View>
        </View>
      ) : null}
      {details.length ? (
        <Glass style={{ gap: space.md }}>
          {details.map(([k, v]) => (
            <View key={k} style={{ flexDirection: 'row', gap: space.md }}>
              <T v="small" soft style={{ width: 92 }}>
                {k}
              </T>
              <T v="small" style={{ flex: 1, fontWeight: '600' }}>
                {v}
              </T>
            </View>
          ))}
        </Glass>
      ) : null}
    </View>
  );
}

export function SharedChips({ shared }: { shared?: string[] | null }) {
  if (!shared?.length) return null;
  return (
    <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 6 }}>
      {shared.map((s) => (
        <Pill key={s} label={s} icon="sparkles" />
      ))}
    </View>
  );
}

export const cardRadius = radius.xl;
