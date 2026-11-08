import { File, Paths } from 'expo-file-system';
import * as Sharing from 'expo-sharing';
import { API_URL, authHeaders } from '@/api/client';
import { currentLanguage } from '@/i18n';

/**
 * This person's invite poster, handed to the share sheet: WeChat there offers Moments, and
 * every other app gets the picture. The poster carries their QR code and invite code.
 */
export async function sharePoster(dialogTitle: string) {
  const file = new File(Paths.cache, `synagent-invite-${currentLanguage()}.jpg`);
  const got = await File.downloadFileAsync(`${API_URL}/v1/referral/poster?lang=${currentLanguage()}`, file, { headers: authHeaders(), idempotent: true });
  await Sharing.shareAsync(got.uri, { mimeType: 'image/jpeg', UTI: 'public.jpeg', dialogTitle });
}
