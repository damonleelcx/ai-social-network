import { Image } from 'expo-image';
import { LinearGradient } from 'expo-linear-gradient';
import { useEffect, useRef, type ReactNode } from 'react';
import { Animated, Easing, View, type ImageSourcePropType, type StyleProp, type ViewStyle } from 'react-native';
import type { Mood } from '@/api/types';
import { useTheme } from '@/theme/ThemeProvider';
import { space } from '@/theme/tokens';
import { T } from './primitives';
import { tr } from '@/i18n';

const FACES: Record<string, ImageSourcePropType> = {
  happy: require('../../assets/suki/happy.png'),
  surprised: require('../../assets/suki/surprised.png'),
  thinking: require('../../assets/suki/thinking.png'),
  playful: require('../../assets/suki/playful.png'),
  pouty: require('../../assets/suki/pouty.png'),
  portrait: require('../../assets/suki/portrait.png'),
  chibi: require('../../assets/suki/chibi.png'),
};

export function sukiFace(mood?: Mood | 'portrait' | 'chibi' | string | null, moodAvatar = true) {
  if (!moodAvatar || !mood) return FACES.portrait;
  return FACES[mood] ?? FACES.portrait;
}

/** Suki in her iridescent ring. */
export function SukiAvatar({
  mood,
  size = 40,
  moodAvatar = true,
  ring = 2,
  style,
}: {
  mood?: Mood | 'portrait' | 'chibi' | string | null;
  size?: number;
  moodAvatar?: boolean;
  ring?: number;
  style?: StyleProp<ViewStyle>;
}) {
  const t = useTheme();
  return (
    <LinearGradient
      colors={t.iris}
      start={{ x: 0, y: 0 }}
      end={{ x: 1, y: 1 }}
      style={[{ width: size, height: size, borderRadius: size / 2, padding: ring }, style]}
    >
      <Image
        source={sukiFace(mood, moodAvatar) as any}
        style={{ flex: 1, borderRadius: size / 2 }}
        contentFit="cover"
        accessibilityLabel="Suki"
        transition={120}
      />
    </LinearGradient>
  );
}

export function TypingDots({ color }: { color?: string }) {
  const t = useTheme();
  const anim = useRef(new Animated.Value(0)).current;
  useEffect(() => {
    const loop = Animated.loop(
      Animated.timing(anim, { toValue: 1, duration: 1200, easing: Easing.linear, useNativeDriver: true }),
    );
    loop.start();
    return () => loop.stop();
  }, [anim]);
  return (
    <View style={{ flexDirection: 'row', gap: 5, alignItems: 'center', height: 18 }} accessibilityLabel={tr('Typing')}>
      {[0, 1, 2].map((i) => (
        <Animated.View
          key={i}
          style={{
            width: 7,
            height: 7,
            borderRadius: 3.5,
            backgroundColor: color ?? t.accent,
            opacity: anim.interpolate({
              inputRange: [0, (0.2 + i * 0.2) * 0.5, 0.2 + i * 0.2, Math.min(0.2 + i * 0.2 + 0.2, 0.99), 1],
              outputRange: [0.3, 0.45, 1, 0.45, 0.3],
            }),
            transform: [
              {
                translateY: anim.interpolate({
                  inputRange: [0, 0.2 + i * 0.2, 1],
                  outputRange: [0, -3, 0],
                }),
              },
            ],
          }}
        />
      ))}
    </View>
  );
}

/** Empty and loading states are Suki talking, not a grey box. */
export function SukiSays({
  face = 'chibi',
  title,
  body,
  children,
  size = 120,
}: {
  face?: string;
  title: string;
  body?: string;
  children?: ReactNode;
  size?: number;
}) {
  return (
    <View style={{ alignItems: 'center', paddingVertical: space.xxl, paddingHorizontal: space.lg, gap: space.md }}>
      <SukiAvatar mood={face} size={size} ring={3} />
      <T v="heading" center>
        {title}
      </T>
      {body ? (
        <T v="body" soft center>
          {body}
        </T>
      ) : null}
      {children}
    </View>
  );
}

export function SukiLoading({ line = tr('One second, let me look.') }: { line?: string }) {
  return (
    <View style={{ alignItems: 'center', paddingVertical: space.huge, gap: space.md }}>
      <SukiAvatar mood="thinking" size={72} />
      <TypingDots />
      <T v="small" soft center>
        {line}
      </T>
    </View>
  );
}

/** Fades a new thing in. Native driver only. */
export function FadeIn({ children }: { children: ReactNode }) {
  const v = useRef(new Animated.Value(0)).current;
  useEffect(() => {
    Animated.timing(v, { toValue: 1, duration: 220, useNativeDriver: true }).start();
  }, [v]);
  return (
    <Animated.View
      style={{
        opacity: v,
        transform: [{ translateY: v.interpolate({ inputRange: [0, 1], outputRange: [8, 0] }) }],
      }}
    >
      {children}
    </Animated.View>
  );
}
