import { Ionicons } from '@expo/vector-icons';
import { LinearGradient } from 'expo-linear-gradient';
import { useState, type ReactNode } from 'react';
import { Pressable, Switch, View } from 'react-native';
import { useTheme } from '@/theme/ThemeProvider';
import { radius, space } from '@/theme/tokens';
import { Badge, Button, Field, Glass, T, type IconName } from './primitives';

type Option = { value: string; label: string };

export function Chip({ label, on, onPress, onRemove }: { label: string; on?: boolean; onPress?: () => void; onRemove?: () => void }) {
  const t = useTheme();
  const body = (
    <View style={{ flexDirection: 'row', alignItems: 'center', gap: 6, paddingHorizontal: 14, paddingVertical: 9 }}>
      <T v="small" color={on ? t.onBrand : t.ink} style={{ fontWeight: '600' }}>
        {label}
      </T>
      {onRemove ? <Ionicons name="close" size={14} color={on ? t.onBrand : t.inkSoft} /> : null}
    </View>
  );
  return (
    <Pressable
      accessibilityRole={onRemove ? 'button' : 'checkbox'}
      accessibilityState={{ checked: !!on }}
      accessibilityLabel={onRemove ? `Remove ${label}` : label}
      disabled={!onPress && !onRemove}
      onPress={onRemove ?? onPress}
      style={({ pressed }) => ({
        borderRadius: radius.pill,
        overflow: 'hidden',
        opacity: pressed ? 0.75 : 1,
        // Selected chips keep the border too, or they sit shorter than their neighbours.
        borderWidth: 1,
        borderColor: on ? 'transparent' : t.hairline,
        backgroundColor: on ? undefined : t.glassStrong,
      })}
    >
      {on ? (
        <LinearGradient colors={t.brand} start={{ x: 0, y: 0 }} end={{ x: 1, y: 1 }}>
          {body}
        </LinearGradient>
      ) : (
        body
      )}
    </Pressable>
  );
}

export function SingleSelect({
  label,
  options,
  value,
  onChange,
}: {
  label?: string;
  options: readonly Option[];
  value?: string | null;
  onChange: (v: string) => void;
}) {
  return (
    <View style={{ gap: 8 }}>
      {label ? (
        <T v="small" soft style={{ fontWeight: '600', marginLeft: 4 }}>
          {label}
        </T>
      ) : null}
      <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 8 }}>
        {options.map((o) => (
          <Chip key={o.value} label={o.label} on={value === o.value} onPress={() => onChange(o.value)} />
        ))}
      </View>
    </View>
  );
}

export function MultiSelect({
  label,
  options,
  value,
  onChange,
}: {
  label?: string;
  options: readonly Option[];
  value: string[];
  onChange: (v: string[]) => void;
}) {
  return (
    <View style={{ gap: 8 }}>
      {label ? (
        <T v="small" soft style={{ fontWeight: '600', marginLeft: 4 }}>
          {label}
        </T>
      ) : null}
      <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 8 }}>
        {options.map((o) => {
          const on = value.includes(o.value);
          return (
            <Chip
              key={o.value}
              label={o.label}
              on={on}
              onPress={() => onChange(on ? value.filter((v) => v !== o.value) : [...value, o.value])}
            />
          );
        })}
      </View>
    </View>
  );
}

/** A free-text list: type, add, tap to remove. */
export function TagInput({
  label,
  placeholder,
  value,
  onChange,
  max = 25,
  hint,
}: {
  label: string;
  placeholder: string;
  value: string[];
  onChange: (v: string[]) => void;
  max?: number;
  hint?: string;
}) {
  const [draft, setDraft] = useState('');
  const commit = () => {
    const parts = draft
      .split(',')
      .map((s) => s.trim())
      .filter(Boolean);
    if (!parts.length) return;
    const next = [...value];
    for (const p of parts) {
      if (next.length < max && !next.some((x) => x.toLowerCase() === p.toLowerCase())) next.push(p);
    }
    onChange(next);
    setDraft('');
  };
  return (
    <View style={{ gap: 8 }}>
      <Field
        label={label}
        placeholder={placeholder}
        value={draft}
        hint={hint}
        onChangeText={setDraft}
        onSubmitEditing={commit}
        onBlur={commit}
        submitBehavior="submit"
        returnKeyType="done"
      />
      {draft.trim() ? <Button label={`Add "${draft.trim()}"`} icon="add" small kind="secondary" onPress={commit} /> : null}
      {value.length ? (
        <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 8 }}>
          {value.map((v) => (
            <Chip key={v} label={v} onRemove={() => onChange(value.filter((x) => x !== v))} />
          ))}
        </View>
      ) : null}
    </View>
  );
}

export function ToggleRow({
  label,
  hint,
  value,
  onChange,
  disabled,
}: {
  label: string;
  hint?: string;
  value: boolean;
  onChange: (v: boolean) => void;
  disabled?: boolean;
}) {
  const t = useTheme();
  return (
    <View style={{ flexDirection: 'row', alignItems: 'center', gap: space.md, paddingVertical: space.sm + 2 }}>
      <View style={{ flex: 1 }}>
        <T v="body" style={{ fontWeight: '500' }}>
          {label}
        </T>
        {hint ? (
          <T v="caption" faint style={{ marginTop: 2 }}>
            {hint}
          </T>
        ) : null}
      </View>
      <Switch
        accessibilityLabel={label}
        value={value}
        disabled={disabled}
        onValueChange={onChange}
        trackColor={{ false: t.hairline, true: t.accent }}
        thumbColor="#FFFFFF"
      />
    </View>
  );
}

export function NavRow({
  icon,
  label,
  hint,
  onPress,
  badge,
  danger,
  right,
}: {
  icon?: IconName;
  label: string;
  hint?: string;
  onPress?: () => void;
  badge?: number;
  danger?: boolean;
  right?: ReactNode;
}) {
  const t = useTheme();
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      onPress={onPress}
      disabled={!onPress}
      style={({ pressed }) => ({
        flexDirection: 'row',
        alignItems: 'center',
        gap: space.md,
        paddingVertical: space.md,
        opacity: pressed ? 0.6 : 1,
      })}
    >
      {icon ? (
        <View
          style={{
            width: 36,
            height: 36,
            borderRadius: 18,
            alignItems: 'center',
            justifyContent: 'center',
            backgroundColor: danger ? t.badSoft : t.accentSoft,
          }}
        >
          <Ionicons name={icon} size={18} color={danger ? t.bad : t.accent} />
        </View>
      ) : null}
      <View style={{ flex: 1 }}>
        <T v="body" color={danger ? t.bad : undefined} style={{ fontWeight: '600' }}>
          {label}
        </T>
        {hint ? (
          <T v="caption" faint style={{ marginTop: 1 }}>
            {hint}
          </T>
        ) : null}
      </View>
      {badge ? <Badge count={badge} /> : null}
      {right ?? (onPress ? <Ionicons name="chevron-forward" size={18} color={t.inkFaint} /> : null)}
    </Pressable>
  );
}

/** A glass group of rows with hairlines between them. */
export function Group({ children }: { children: ReactNode }) {
  const t = useTheme();
  const items = (Array.isArray(children) ? children.flat() : [children]).filter(Boolean);
  return (
    <Glass style={{ paddingVertical: space.xs }}>
      {items.map((c, i) => (
        <View key={i} style={i > 0 ? { borderTopWidth: 1, borderTopColor: t.hairline } : undefined}>
          {c}
        </View>
      ))}
    </Glass>
  );
}
