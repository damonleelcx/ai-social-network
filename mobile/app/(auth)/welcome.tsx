import { Image } from 'expo-image';
import { LinearGradient } from 'expo-linear-gradient';
import { router } from 'expo-router';
import { useEffect, useRef, useState } from 'react';
import { Animated, Easing, View } from 'react-native';
import { useScreenSize } from '@/ui/screenSize';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useTheme } from '@/theme/ThemeProvider';
import { space } from '@/theme/tokens';
import { Backdrop, LanguagePicker, ThemeToggle } from '@/ui/layout';
import { Button, Glass, T } from '@/ui/primitives';
import { sukiFace } from '@/ui/suki';
import { MAP_ASPECT, WorldMap } from '@/ui/WorldMap';
import { tr } from '@/i18n';

const SMALLEST_SUKI = 64;
// Below this the intro paragraph goes, so the map keeps its full width.
const ROOM_FOR_INTRO = 720;

export default function Welcome() {
  const t = useTheme();
  const insets = useSafeAreaInsets();
  const { width, height } = useScreenSize();
  // Small phones can't fit her at full size, so she gives up whatever the words need.
  const [room, setRoom] = useState(height);
  const [wordsHeight, setWordsHeight] = useState(0);
  // The map is the biggest thing here: edge to edge, and she never outgrows it.
  const mapHeight = width / MAP_ASPECT;
  const size = Math.max(SMALLEST_SUKI, Math.min(mapHeight - 28, room - wordsHeight - 28));
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
        <View style={{ position: 'absolute', zIndex: 1, top: insets.top + space.sm, right: space.lg, flexDirection: 'row', gap: space.sm }}>
          <LanguagePicker />
          <ThemeToggle />
        </View>
        <View
          style={{ flex: 1, alignItems: 'center', justifyContent: 'center' }}
          onLayout={(e) => setRoom(e.nativeEvent.layout.height)}
        >
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
              accessibilityLabel={tr('Suki, your AI wingwoman, winking')}
            />
          </View>
          <View
            style={{ paddingTop: space.lg, alignSelf: 'stretch', marginHorizontal: -space.xl }}
            onLayout={(e) => setWordsHeight(e.nativeEvent.layout.height)}
          >
            <View style={{ paddingHorizontal: space.xl }}>
              <T v="display" center>
                SynAgent
              </T>
              <T v="heading" center color={t.accent} style={{ marginTop: space.xs }}>{tr('Meet someone, share ideas, explore together')}</T>
            </View>
            <View style={{ marginTop: space.md }}>
              <WorldMap width={width} />
            </View>
          </View>
        </View>

        {height >= ROOM_FOR_INTRO ? (
          <Glass style={{ marginBottom: space.lg }}>
            <T soft center>{tr("I'm Suki, your AI wingwoman. Tell me about you, and I'll introduce you to people worth meeting. No swiping.")}</T>
          </Glass>
        ) : null}
        <View style={{ gap: space.sm }}>
          <Button label={tr('Get started')} full onPress={() => router.push('/signup')} />
          <Button label={tr('I already have an account')} kind="ghost" full onPress={() => router.push('/signin')} />
        </View>
      </View>
    </Backdrop>
  );
}
