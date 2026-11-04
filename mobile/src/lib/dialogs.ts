import { type AlertButton, Alert as NativeAlert, Platform, Share as NativeShare } from 'react-native';
import { tr } from '@/i18n';

// react-native-web's Alert does nothing, and its Share rejects where the browser can't share.

function webAlert(title: string, message?: string, buttons?: AlertButton[], _options?: unknown) {
  const text = message ? `${title}\n\n${message}` : title;
  const yes = buttons?.find((b) => b.style !== 'cancel');
  const no = buttons?.find((b) => b.style === 'cancel');
  if (yes && no) {
    (window.confirm(text) ? yes : no).onPress?.();
    return;
  }
  window.alert(text);
  yes?.onPress?.();
}

async function webShare({ message }: { message: string }) {
  if (navigator.share) {
    try {
      await navigator.share({ text: message });
    } catch {
      // They closed the share sheet.
    }
    return;
  }
  await navigator.clipboard.writeText(message);
  window.alert(tr('Copied. Paste it wherever you like.'));
}

export const Alert = Platform.OS === 'web' ? { alert: webAlert } : NativeAlert;
export const Share = Platform.OS === 'web' ? { share: webShare } : NativeShare;
