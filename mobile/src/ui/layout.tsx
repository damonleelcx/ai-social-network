import { LinearGradient } from 'expo-linear-gradient';
import { router } from 'expo-router';
import { StatusBar } from 'expo-status-bar';
import type { ReactNode } from 'react';
import {
  KeyboardAvoidingView,
  Platform,
  RefreshControl,
  ScrollView,
  StyleSheet,
  View,
  type ScrollViewProps,
  type StyleProp,
  type ViewStyle,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useSession } from '@/state/session';
import { useTheme } from '@/theme/ThemeProvider';
import { space } from '@/theme/tokens';
import { IconButton, IrisRule, T } from './primitives';
import { showError } from './sheet';

/** The lilac (or deep violet) gradient everything sits on. */
export function Backdrop({ children }: { children?: ReactNode }) {
  const t = useTheme();
  return (
    <View style={{ flex: 1, backgroundColor: t.bg[0] }}>
      <StatusBar style={t.statusBar} />
      <LinearGradient
        colors={t.bg}
        locations={[0, 0.4, 0.75, 1]}
        start={{ x: 0, y: 0 }}
        end={{ x: 1, y: 1 }}
        style={StyleSheet.absoluteFill}
        pointerEvents="none"
      />
      {children}
    </View>
  );
}

/** One tap between light and dark. "Match my phone" lives in Settings > App. */
export function ThemeToggle() {
  const t = useTheme();
  const { setTheme } = useSession();
  const toDark = t.scheme === 'light';
  return (
    <IconButton
      icon={toDark ? 'moon-outline' : 'sunny-outline'}
      label={toDark ? 'Switch to dark mode' : 'Switch to light mode'}
      onPress={() => setTheme(toDark ? 'dark' : 'light').catch(showError)}
    />
  );
}

export function ScreenHeader({
  title,
  right,
  back = true,
}: {
  title: string;
  right?: ReactNode;
  back?: boolean;
}) {
  const insets = useSafeAreaInsets();
  return (
    <View style={{ paddingTop: insets.top + space.xs }}>
      <View style={{ flexDirection: 'row', alignItems: 'center', paddingHorizontal: space.sm, minHeight: 52, gap: space.xs }}>
        {back ? (
          <IconButton
            icon="chevron-back"
            label="Back"
            onPress={() => (router.canGoBack() ? router.back() : router.replace('/suki'))}
          />
        ) : (
          <View style={{ width: space.md }} />
        )}
        <T v="heading" numberOfLines={1} style={{ flex: 1 }} accessibilityRole="header">
          {title}
        </T>
        {right}
      </View>
      <IrisRule style={{ opacity: 0.55 }} />
    </View>
  );
}

/** A big title for the top level of a tab. */
export function TabHeader({ title, subtitle, right }: { title: string; subtitle?: string; right?: ReactNode }) {
  const insets = useSafeAreaInsets();
  return (
    <View style={{ paddingTop: insets.top + space.md, paddingHorizontal: space.xl, paddingBottom: space.md }}>
      <View style={{ flexDirection: 'row', alignItems: 'center' }}>
        <T v="title" style={{ flex: 1 }} accessibilityRole="header">
          {title}
        </T>
        {right}
      </View>
      {subtitle ? (
        <T v="small" soft style={{ marginTop: 2 }}>
          {subtitle}
        </T>
      ) : null}
    </View>
  );
}

/** A scrollable page body with sensible padding, keyboard handling and pull-to-refresh. */
export function Page({
  children,
  refreshing,
  onRefresh,
  contentStyle,
  bottomInset = true,
  ...rest
}: ScrollViewProps & {
  children: ReactNode;
  refreshing?: boolean;
  onRefresh?: () => void;
  contentStyle?: StyleProp<ViewStyle>;
  bottomInset?: boolean;
}) {
  const t = useTheme();
  const insets = useSafeAreaInsets();
  return (
    <KeyboardAvoidingView style={{ flex: 1 }} behavior="padding" keyboardVerticalOffset={Platform.OS === 'ios' ? 0 : 0}>
      <ScrollView
        {...rest}
        keyboardShouldPersistTaps="handled"
        contentContainerStyle={[
          { padding: space.xl, gap: space.lg, paddingBottom: (bottomInset ? insets.bottom : 0) + space.xxl },
          contentStyle,
        ]}
        refreshControl={
          onRefresh ? (
            <RefreshControl refreshing={!!refreshing} onRefresh={onRefresh} tintColor={t.accent} colors={[t.accent]} />
          ) : undefined
        }
      >
        {children}
      </ScrollView>
    </KeyboardAvoidingView>
  );
}

export function Section({ title, hint, children }: { title?: string; hint?: string; children: ReactNode }) {
  return (
    <View style={{ gap: space.sm }}>
      {title ? (
        <T v="label" soft style={{ marginLeft: 4 }}>
          {title}
        </T>
      ) : null}
      {children}
      {hint ? (
        <T v="caption" faint style={{ marginLeft: 4 }}>
          {hint}
        </T>
      ) : null}
    </View>
  );
}
