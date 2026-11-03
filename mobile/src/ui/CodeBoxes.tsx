import { useRef, useState, type ComponentRef } from 'react';
import { Pressable, TextInput, View } from 'react-native';
import { useTheme } from '@/theme/ThemeProvider';
import { radius, space } from '@/theme/tokens';
import { T } from './primitives';
import { tr } from '@/i18n';

/** Six boxes over one real text input, so paste and autofill still work. */
export function CodeBoxes({ value, onChange, autoFocus }: { value: string; onChange: (v: string) => void; autoFocus?: boolean }) {
  const t = useTheme();
  const input = useRef<ComponentRef<typeof TextInput>>(null);
  const [focused, setFocused] = useState(false);
  return (
    <Pressable onPress={() => input.current?.focus()} accessibilityLabel={tr('Six digit code')} style={{ alignItems: 'center' }}>
      <View style={{ flexDirection: 'row', gap: space.sm }}>
        {Array.from({ length: 6 }, (_, i) => {
          const active = focused && i === Math.min(value.length, 5);
          return (
            <View
              key={i}
              style={{
                width: 46,
                height: 58,
                borderRadius: radius.sm,
                alignItems: 'center',
                justifyContent: 'center',
                backgroundColor: t.glassStrong,
                borderWidth: active ? 2 : 1.5,
                borderColor: active ? t.accent : t.hairline,
              }}
            >
              <T v="title">{value[i] ?? ''}</T>
            </View>
          );
        })}
      </View>
      <TextInput
        ref={input}
        value={value}
        autoFocus={autoFocus}
        onChangeText={(v) => onChange(v.replace(/\D/g, '').slice(0, 6))}
        keyboardType="number-pad"
        textContentType="oneTimeCode"
        autoComplete="sms-otp"
        maxLength={6}
        onFocus={() => setFocused(true)}
        onBlur={() => setFocused(false)}
        caretHidden
        style={{ position: 'absolute', opacity: 0.01, top: 0, bottom: 0, left: 0, right: 0, color: 'transparent' }}
      />
    </Pressable>
  );
}
