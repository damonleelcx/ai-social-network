import { Image } from 'expo-image';
import { LinearGradient } from 'expo-linear-gradient';
import { router } from 'expo-router';
import { useEffect, useRef } from 'react';
import { Animated, Easing, useWindowDimensions, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useTheme } from '@/theme/ThemeProvider';
import { space } from '@/theme/tokens';
import { Backdrop } from '@/ui/layout';
import { Button, Glass, T } from '@/ui/primitives';
import { sukiFace } from '@/ui/suki';

export default function Welcome() {
  const t = useTheme();
  const insets = useSafeAreaInsets();
  const { width, height } = useWindowDimensions();
  const size = Math.min(width - 72, height * 0.38);
  const glow = useRef(new Animated.Value(0)).current;

  useEffect(() => {
    const loop = Animated.loop(
      Animated.sequence([
        Animated.timing(glow, { toValue: 1, duration: 2400, easing: Easing.inOut(Easing.quad), useNativeDriver: true }),
        Animated.timing(glow, { toValue: 0, duration: 2400, easing: Easing.inOut(Easing.quad), useNativeDriver: true }),
      ]),
    );
    loop.start();
    return () => loop.stop();
  }, [glow]);

  return (
    <Backdrop>
      <View style={{ flex: 1, paddingTop: insets.top + space.xxl, paddingBottom: insets.bottom + space.xl, paddingHorizontal: space.xl }}>
        <View style={{ flex: 1, alignItems: 'center', justifyContent: 'center' }}>
          <View style={{ width: size + 28, height: size + 28, alignItems: 'center', justifyContent: 'center' }}>
            <Animated.View
              style={{
                position: 'absolute',
                width: size + 28,
                height: size + 28,
                opacity: glow.interpolate({ inputRange: [0, 1], outputRange: [0.35, 0.8] }),
                transform: [{ scale: glow.interpolate({ inputRange: [0, 1], outputRange: [0.98, 1.03] }) }],
              }}
            >
              <LinearGradient
                colors={t.iris}
                start={{ x: 0, y: 0 }}
                end={{ x: 1, y: 1 }}
                style={{ flex: 1, borderRadius: (size + 28) / 2 }}
              />
            </Animated.View>
            <Image
              source={sukiFace('portrait') as any}
              style={{ width: size, height: size, borderRadius: size / 2, borderWidth: 3, borderColor: t.glassBorder }}
              contentFit="cover"
              accessibilityLabel="Suki, your AI wingwoman, winking"
            />
          </View>
          <T v="display" center style={{ marginTop: space.xl }}>
            SynAgent
          </T>
          <T v="heading" center color={t.accent} style={{ marginTop: space.xs }}>
            Meet someone, share ideas, explore together
          </T>
        </View>

        <Glass style={{ marginBottom: space.lg }}>
          <T soft center>
            I'm Suki, your AI wingwoman. Tell me about you, and I'll introduce you to people worth meeting. No swiping.
          </T>
        </Glass>
        <View style={{ gap: space.sm }}>
          <Button label="Get started" full onPress={() => router.push('/signup')} />
          <Button label="I already have an account" kind="ghost" full onPress={() => router.push('/signin')} />
        </View>
      </View>
    </Backdrop>
  );
}
