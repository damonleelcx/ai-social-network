import { forwardRef, type ComponentRef } from 'react';
import { TextInput, View } from 'react-native';
import { useTheme } from '@/theme/ThemeProvider';
import { radius, space } from '@/theme/tokens';
import { IconButton } from '@/ui/primitives';
import { tr } from '@/i18n';

type Props = {
  value: string;
  onChangeText: (v: string) => void;
  onSend: () => void;
  placeholder: string;
  disabled?: boolean;
};

/** The composer shared by both chats. */
export const ChatInput = forwardRef<ComponentRef<typeof TextInput>, Props>(function ChatInput({ value, onChangeText, onSend, placeholder, disabled }, ref) {
  const t = useTheme();
  const canSend = !!value.trim() && !disabled;
  return (
    <View
      style={{
        flexDirection: 'row',
        alignItems: 'flex-end',
        gap: space.xs,
        backgroundColor: t.glassStrong,
        borderRadius: radius.lg,
        borderWidth: 1,
        borderColor: t.hairline,
        paddingLeft: space.lg,
        paddingRight: space.xs,
        paddingVertical: space.xs,
      }}
    >
      <TextInput
        ref={ref}
        value={value}
        onChangeText={onChangeText}
        placeholder={placeholder}
        placeholderTextColor={t.inkFaint}
        selectionColor={t.accent}
        multiline
        editable={!disabled}
        maxLength={4000}
        accessibilityLabel={placeholder}
        style={{ flex: 1, color: t.ink, fontSize: 16, lineHeight: 21, maxHeight: 120, paddingTop: 10, paddingBottom: 10 }}
      />
      <IconButton icon="arrow-up-circle" size={34} label={tr('Send')} color={canSend ? t.accent : t.inkFaint} disabled={!canSend} onPress={onSend} />
    </View>
  );
});
