import { Image } from 'expo-image';
import { LinearGradient } from 'expo-linear-gradient';
import { useEffect, useRef, useState, type ReactNode } from 'react';
import { Platform, Pressable, View } from 'react-native';
import { useTheme } from '@/theme/ThemeProvider';
import { radius, space } from '@/theme/tokens';
import { API_URL } from '@/api/client';
import { MediaImage, PersonAvatar } from '@/ui/person';
import { T } from '@/ui/primitives';

/** A scene picture's address. They show no one, so they need no sign-in and never change. */
export const sceneUri = (key: string) => `${API_URL}/v1/scenes/${key}.jpg`;

const SCENE_RETRY_MS = 3000;
const SCENE_RETRIES = 40;

/** A scene picture. A new one takes a while to draw and 404s until then, so this keeps asking. */
function SceneImage({ id }: { id: string }) {
  const [attempt, setAttempt] = useState(0);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  useEffect(
    () => () => {
      if (timer.current) clearTimeout(timer.current);
    },
    [],
  );
  return (
    <Image
      // On the web a failed image never loads a new source, so each try is a new one.
      key={attempt}
      source={{ uri: attempt ? `${sceneUri(id)}?try=${attempt}` : sceneUri(id) }}
      style={{ position: 'absolute', top: 0, left: 0, right: 0, bottom: 0 }}
      contentFit="cover"
      cachePolicy="disk"
      // On the web a fade can stall and leave the picture invisible.
      transition={Platform.OS === 'web' ? 0 : 160}
      onError={() => {
        if (attempt < SCENE_RETRIES) timer.current = setTimeout(() => setAttempt((a) => a + 1), SCENE_RETRY_MS);
      }}
    />
  );
}

/**
 * A card picture first: a person Suki recommends, or a plan with a picture of what it's about
 * and the person by their name. Name and details sit over the picture's foot, what to do
 * underneath. Introductions, plans, groups and trips all use it, so they look of a piece.
 */
export function HeroCard({
  photo,
  scene,
  title,
  subtitle,
  badges,
  corner,
  onPress,
  pressLabel,
  height = 300,
  children,
}: {
  photo?: string | null;
  /** A picture of what the plan is about. With it, the person's photo becomes a small round one by their name. */
  scene?: string;
  title: string;
  subtitle?: string;
  /** Small labels over the top of the photo: what kind of plan, how full it is. */
  badges?: ReactNode;
  /** Top right over the photo, such as the report and block menu. */
  corner?: ReactNode;
  onPress?: () => void;
  pressLabel?: string;
  height?: number;
  children?: ReactNode;
}) {
  const t = useTheme();
  const picture = (
    <View style={{ height }}>
      {scene ? (
        <SceneImage key={scene} id={scene} />
      ) : (
        <MediaImage id={photo} style={{ position: 'absolute', top: 0, left: 0, right: 0, bottom: 0 }} />
      )}
      <LinearGradient
        colors={['rgba(8,4,24,0)', 'rgba(8,4,24,0.15)', 'rgba(8,4,24,0.85)']}
        locations={[0, 0.45, 1]}
        style={{ position: 'absolute', left: 0, right: 0, bottom: 0, height: height * 0.6 }}
      />
      {badges ? <View style={{ position: 'absolute', top: space.md, left: space.md, right: 56, flexDirection: 'row', flexWrap: 'wrap', gap: 6 }}>{badges}</View> : null}
      <View style={{ position: 'absolute', left: space.lg, right: space.lg, bottom: space.md, flexDirection: 'row', alignItems: 'center', gap: space.md }}>
        {scene && photo ? <PersonAvatar id={photo} size={52} /> : null}
        <View style={{ flex: 1, gap: 2 }}>
          <T v="title" color="#fff" numberOfLines={1} style={{ textShadowColor: 'rgba(0,0,0,0.35)', textShadowRadius: 8 }}>
            {title}
          </T>
          {subtitle ? (
            <T v="small" color="rgba(255,255,255,0.9)" numberOfLines={1}>
              {subtitle}
            </T>
          ) : null}
        </View>
      </View>
    </View>
  );
  return (
    <View
      style={{
        borderRadius: radius.xl,
        overflow: 'hidden',
        backgroundColor: t.glassStrong,
        borderWidth: 1,
        borderColor: t.glassBorder,
      }}
    >
      {onPress ? (
        <Pressable accessibilityRole="button" accessibilityLabel={pressLabel ?? title} onPress={onPress}>
          {picture}
        </Pressable>
      ) : (
        picture
      )}
      {corner ? <View style={{ position: 'absolute', top: space.sm, right: space.sm }}>{corner}</View> : null}
      {children ? <View style={{ padding: space.lg, gap: space.md }}>{children}</View> : null}
    </View>
  );
}

/** A label that reads on top of any photo. */
export function PhotoBadge({ label }: { label: string }) {
  return (
    <View style={{ backgroundColor: 'rgba(8,4,24,0.55)', borderRadius: 999, paddingHorizontal: 10, paddingVertical: 4, borderWidth: 1, borderColor: 'rgba(255,255,255,0.18)' }}>
      <T v="caption" color="#fff" style={{ fontWeight: '700' }}>
        {label}
      </T>
    </View>
  );
}
