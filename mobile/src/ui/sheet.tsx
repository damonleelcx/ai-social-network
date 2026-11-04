import { Ionicons } from '@expo/vector-icons';
import type { ReactNode } from 'react';
import { KeyboardAvoidingView, Modal, Platform, Pressable, ScrollView, View } from 'react-native';
import { Alert } from '@/lib/dialogs';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { useTheme } from '@/theme/ThemeProvider';
import { radius, space } from '@/theme/tokens';
import { IrisRule, T, type IconName } from './primitives';
import { WEB_WIDTH } from './screenSize';
import { tr } from '@/i18n';

/** A bottom sheet. Android alerts only hold three buttons, so anything longer goes here. */
export function Sheet({
  visible,
  onClose,
  title,
  children,
}: {
  visible: boolean;
  onClose: () => void;
  title?: string;
  children: ReactNode;
}) {
  const t = useTheme();
  const insets = useSafeAreaInsets();
  // A modal covers the whole window, and the web app is a column in the middle of it.
  const web = Platform.OS === 'web';
  return (
    <Modal visible={visible} transparent animationType="slide" statusBarTranslucent navigationBarTranslucent onRequestClose={onClose}>
      <KeyboardAvoidingView behavior="padding" style={{ flex: 1, justifyContent: 'flex-end', backgroundColor: web ? t.scrim : undefined }}>
        <Pressable accessibilityLabel={tr('Close')} onPress={onClose} style={{ flex: 1, backgroundColor: web ? undefined : t.scrim }} />
        <View
          style={{
            backgroundColor: t.sheet,
            borderTopLeftRadius: radius.xl,
            borderTopRightRadius: radius.xl,
            paddingTop: space.md,
            paddingHorizontal: space.xl,
            paddingBottom: insets.bottom + space.lg,
            maxHeight: '86%',
            ...(web ? { width: '100%', maxWidth: WEB_WIDTH, alignSelf: 'center' } : null),
          }}
        >
          <IrisRule style={{ width: 44, alignSelf: 'center', height: 4, borderRadius: 2, marginBottom: space.md }} />
          {title ? (
            <T v="heading" style={{ marginBottom: space.md }}>
              {title}
            </T>
          ) : null}
          <ScrollView keyboardShouldPersistTaps="handled" contentContainerStyle={{ gap: space.md }}>
            {children}
          </ScrollView>
        </View>
      </KeyboardAvoidingView>
    </Modal>
  );
}

export function SheetAction({
  icon,
  label,
  danger,
  onPress,
}: {
  icon?: IconName;
  label: string;
  danger?: boolean;
  onPress: () => void;
}) {
  const t = useTheme();
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={label}
      onPress={onPress}
      style={({ pressed }) => ({
        flexDirection: 'row',
        alignItems: 'center',
        gap: space.md,
        paddingVertical: space.sm,
        opacity: pressed ? 0.6 : 1,
      })}
    >
      {icon ? <Ionicons name={icon} size={21} color={danger ? t.bad : t.accent} /> : null}
      <T v="body" color={danger ? t.bad : t.ink} style={{ fontWeight: '600' }}>
        {label}
      </T>
    </Pressable>
  );
}

export function confirm(title: string, body: string, yes: string, destructive = false) {
  return new Promise<boolean>((resolve) => {
    Alert.alert(
      title,
      body,
      [
        { text: tr('Not now'), style: 'cancel', onPress: () => resolve(false) },
        { text: yes, style: destructive ? 'destructive' : 'default', onPress: () => resolve(true) },
      ],
      { cancelable: true, onDismiss: () => resolve(false) },
    );
  });
}

/** For an action that failed away from any form: the API's own words. */
export function showError(e: unknown) {
  const message = e instanceof Error && e.message ? e.message : tr('Something went wrong. Please try again.');
  Alert.alert(tr("That didn't work"), message);
}
