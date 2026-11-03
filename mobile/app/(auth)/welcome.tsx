import { Image } from 'expo-image';
import { LinearGradient } from 'expo-linear-gradient';
import { router } from 'expo-router';
import { useEffect, useRef, useState } from 'react';
import { Animated, Easing, useWindowDimensions, View } from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useTheme } from '@/theme/ThemeProvider';
import { space } from '@/theme/tokens';
import { Backdrop, LanguagePicker, ThemeToggle } from '@/ui/layout';
import { Button, Glass, T } from '@/ui/primitives';
import { sukiFace } from '@/ui/suki';
import { MAP_ASPECT, MAP_CAPTION_HEIGHT, WorldMap } from '@/ui/WorldMap';
import { tr } from '@/i18n';

const SMALLEST_SUKI = 72;

export default function Welcome() {
  const t = useTheme();
  const insets = useSafeAreaInsets();
  const { width, height } = useWindowDimensions();
  // Small phones can't fit her at full size, so she gives up whatever the words need.
  const [room, setRoom] = useState(height);
  const [wordsHeight, setWordsHeight] = useState(0);
  const [titleHeight, setTitleHeight] = useState(0);
  const size = Math.max(SMALLEST_SUKI, Math.min(width - 72, height * 0.38, room - wordsHeight - 28));
  // The map takes what's left once she's at her smallest, and small phones go without it.
  const mapRoom = room - SMALLEST_SUKI - 28 - space.xl - titleHeight - space.md - MAP_CAPTION_HEIGHT;
  const mapWidth = Math.min(width - space.xl * 2, mapRoom * MAP_ASPECT);
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
          <View style={{ paddingTop: space.xl }} onLayout={(e) => setWordsHeight(e.nativeEvent.layout.height)}>
            <View onLayout={(e) => setTitleHeight(e.nativeEvent.layout.height)}>
              <T v="display" center>
                SynAgent
              </T>
              <T v="heading" center color={t.accent} style={{ marginTop: space.xs }}>{tr('Meet someone, share ideas, explore together')}</T>
            </View>
            {mapWidth >= 140 ? (
              <View style={{ marginTop: space.md }}>
                <WorldMap width={mapWidth} />
              </View>
            ) : null}
          </View>
        </View>

        <Glass style={{ marginBottom: space.lg }}>
          <T soft center>{tr("I'm Suki, your AI wingwoman. Tell me about you, and I'll introduce you to people worth meeting. No swiping.")}</T>
        </Glass>
        <View style={{ gap: space.sm }}>
          <Button label={tr('Get started')} full onPress={() => router.push('/signup')} />
          <Button label={tr('I already have an account')} kind="ghost" full onPress={() => router.push('/signin')} />
        </View>
      </View>
    </Backdrop>
  );
}
