import { Ionicons } from '@expo/vector-icons';
import { LinearGradient } from 'expo-linear-gradient';
import { useState, type ComponentProps, type ReactNode } from 'react';
import {
  ActivityIndicator,
  Pressable,
  Text,
  TextInput,
  View,
  type StyleProp,
  type TextInputProps,
  type TextProps,
  type ViewStyle,
} from 'react-native';
import { useTheme } from '@/theme/ThemeProvider';
import { radius, space, type, type TypeName } from '@/theme/tokens';

export type IconName = ComponentProps<typeof Ionicons>['name'];

type TProps = TextProps & {
  v?: TypeName;
  color?: string;
  soft?: boolean;
  faint?: boolean;
  center?: boolean;
};

export function T({ v = 'body', color, soft, faint, center, style, ...rest }: TProps) {
  const t = useTheme();
  const c = color ?? (faint ? t.inkFaint : soft ? t.inkSoft : t.ink);
  return <Text {...rest} style={[type[v], { color: c }, center && { textAlign: 'center' }, style]} />;
}

export function Glass({
  children,
  style,
  strong,
  padded = true,
}: {
  children?: ReactNode;
  style?: StyleProp<ViewStyle>;
  strong?: boolean;
  padded?: boolean;
}) {
  const t = useTheme();
  return (
    <View
      style={[
        {
          backgroundColor: strong ? t.glassStrong : t.glass,
          borderColor: t.glassBorder,
          borderWidth: 1,
          borderRadius: radius.lg,
          padding: padded ? space.lg : 0,
        },
        style,
      ]}
    >
      {children}
    </View>
  );
}

/** A thin holographic line, the app's signature accent. */
export function IrisRule({ style }: { style?: StyleProp<ViewStyle> }) {
  const t = useTheme();
  return (
    <LinearGradient
      colors={t.iris}
      start={{ x: 0, y: 0 }}
      end={{ x: 1, y: 0 }}
      style={[{ height: 2, borderRadius: 1 }, style]}
    />
  );
}

type ButtonProps = {
  label: string;
  onPress?: () => void;
  kind?: 'primary' | 'secondary' | 'ghost' | 'danger';
  icon?: IconName;
  loading?: boolean;
  disabled?: boolean;
  small?: boolean;
  full?: boolean;
  style?: StyleProp<ViewStyle>;
};

export function Button({ label, onPress, kind = 'primary', icon, loading, disabled, small, full, style }: ButtonProps) {
  const t = useTheme();
  const off = disabled || loading;
  const height = small ? 40 : 52;
  const fg = kind === 'primary' ? t.onBrand : kind === 'danger' ? t.bad : t.accent;
  const inner = (
    <View
      style={{
        flexDirection: 'row',
        alignItems: 'center',
        justifyContent: 'center',
        gap: space.sm,
        minHeight: height,
        paddingHorizontal: small ? space.lg : space.xl,
      }}
    >
      {loading ? (
        <ActivityIndicator color={fg} size="small" />
      ) : icon ? (
        <Ionicons name={icon} size={small ? 16 : 19} color={fg} />
      ) : null}
      <T v={small ? 'small' : 'subheading'} color={fg} style={{ fontWeight: '700' }} numberOfLines={1}>
        {label}
      </T>
    </View>
  );
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      accessibilityState={{ disabled: !!off, busy: !!loading }}
      disabled={off}
      onPress={onPress}
      style={({ pressed }) => [
        { borderRadius: radius.pill, overflow: 'hidden', opacity: off ? 0.55 : pressed ? 0.82 : 1 },
        full ? { alignSelf: 'stretch' } : { alignSelf: 'flex-start' },
        kind === 'secondary' && { backgroundColor: t.accentSoft },
        kind === 'danger' && { backgroundColor: t.badSoft },
        style,
      ]}
    >
      {kind === 'primary' ? (
        <LinearGradient colors={t.brand} start={{ x: 0, y: 0 }} end={{ x: 1, y: 1 }}>
          {inner}
        </LinearGradient>
      ) : (
        inner
      )}
    </Pressable>
  );
}

export function IconButton({
  icon,
  onPress,
  label,
  color,
  size = 22,
  filled,
  disabled,
}: {
  icon: IconName;
  onPress?: () => void;
  label: string;
  color?: string;
  size?: number;
  filled?: boolean;
  disabled?: boolean;
}) {
  const t = useTheme();
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      hitSlop={8}
      disabled={disabled}
      onPress={onPress}
      style={({ pressed }) => ({
        width: 42,
        height: 42,
        borderRadius: 21,
        alignItems: 'center',
        justifyContent: 'center',
        backgroundColor: filled ? t.accentSoft : 'transparent',
        opacity: disabled ? 0.4 : pressed ? 0.6 : 1,
      })}
    >
      <Ionicons name={icon} size={size} color={color ?? t.ink} />
    </Pressable>
  );
}

export function Pill({
  label,
  tone = 'accent',
  icon,
  style,
}: {
  label: string;
  tone?: 'accent' | 'good' | 'warn' | 'bad' | 'plain';
  icon?: IconName;
  style?: StyleProp<ViewStyle>;
}) {
  const t = useTheme();
  const map = {
    accent: [t.accentSoft, t.accent],
    good: [t.goodSoft, t.good],
    warn: [t.warnSoft, t.warn],
    bad: [t.badSoft, t.bad],
    plain: [t.glass, t.inkSoft],
  } as const;
  const [bg, fg] = map[tone];
  return (
    <View
      style={[
        {
          flexDirection: 'row',
          alignItems: 'center',
          gap: 5,
          alignSelf: 'flex-start',
          paddingHorizontal: 11,
          paddingVertical: 5,
          borderRadius: radius.pill,
          backgroundColor: bg,
        },
        style,
      ]}
    >
      {icon ? <Ionicons name={icon} size={13} color={fg} /> : null}
      <T v="caption" color={fg} style={{ fontWeight: '700' }}>
        {label}
      </T>
    </View>
  );
}

/** Errors from the API, shown where the person is looking. */
export function ErrorText({ message }: { message?: string | null }) {
  const t = useTheme();
  if (!message) return null;
  return (
    <View
      accessibilityLiveRegion="polite"
      style={{ flexDirection: 'row', gap: space.sm, padding: space.md, borderRadius: radius.sm, backgroundColor: t.badSoft }}
    >
      <Ionicons name="alert-circle" size={18} color={t.bad} />
      <T v="small" color={t.bad} style={{ flex: 1 }}>
        {message}
      </T>
    </View>
  );
}

type FieldProps = TextInputProps & {
  label?: string;
  hint?: string;
  secure?: boolean;
  style?: never;
  inputStyle?: TextInputProps['style'];
};

export function Field({ label, hint, secure, multiline, inputStyle, ...rest }: FieldProps) {
  const t = useTheme();
  const [focused, setFocused] = useState(false);
  const [shown, setShown] = useState(false);
  return (
    <View style={{ gap: 6 }}>
      {label ? (
        <T v="small" soft style={{ fontWeight: '600', marginLeft: 4 }}>
          {label}
        </T>
      ) : null}
      <View
        style={{
          flexDirection: 'row',
          alignItems: multiline ? 'flex-start' : 'center',
          borderRadius: multiline ? radius.md : radius.pill,
          borderWidth: 1.5,
          borderColor: focused ? t.accent : t.hairline,
          backgroundColor: t.glassStrong,
          paddingHorizontal: space.lg,
        }}
      >
        <TextInput
          {...rest}
          multiline={multiline}
          accessibilityLabel={rest.accessibilityLabel ?? label}
          secureTextEntry={secure && !shown}
          placeholderTextColor={t.inkFaint}
          selectionColor={t.accent}
          onFocus={(e) => {
            setFocused(true);
            rest.onFocus?.(e);
          }}
          onBlur={(e) => {
            setFocused(false);
            rest.onBlur?.(e);
          }}
          style={[
            {
              flex: 1,
              color: t.ink,
              fontSize: 16,
              paddingVertical: multiline ? 12 : 13,
              minHeight: multiline ? 96 : 50,
              textAlignVertical: multiline ? 'top' : 'center',
            },
            inputStyle,
          ]}
        />
        {secure ? (
          <Pressable
            hitSlop={10}
            accessibilityRole="button"
            accessibilityLabel={shown ? 'Hide password' : 'Show password'}
            onPress={() => setShown((s) => !s)}
          >
            <Ionicons name={shown ? 'eye-off-outline' : 'eye-outline'} size={20} color={t.inkFaint} />
          </Pressable>
        ) : null}
      </View>
      {hint ? (
        <T v="caption" faint style={{ marginLeft: 4 }}>
          {hint}
        </T>
      ) : null}
    </View>
  );
}

export function Badge({ count }: { count: number }) {
  const t = useTheme();
  if (count <= 0) return null;
  return (
    <LinearGradient
      colors={t.brand}
      start={{ x: 0, y: 0 }}
      end={{ x: 1, y: 1 }}
      style={{ minWidth: 22, height: 22, borderRadius: 11, alignItems: 'center', justifyContent: 'center', paddingHorizontal: 6 }}
    >
      <T v="caption" color={t.onBrand} style={{ fontWeight: '800' }}>
        {count > 99 ? '99+' : count}
      </T>
    </LinearGradient>
  );
}
