import { API_URL, authHeaders } from '@/api/client';
import { currentLanguage } from '@/i18n';

/** On the web the poster is downloaded, to post wherever they like. */
export async function sharePoster(_dialogTitle: string) {
  const res = await fetch(`${API_URL}/v1/referral/poster?lang=${currentLanguage()}`, { headers: authHeaders() });
  if (!res.ok) throw new Error('poster');
  const url = URL.createObjectURL(await res.blob());
  const a = document.createElement('a');
  a.href = url;
  a.download = 'synagent-invite.jpg';
  document.body.appendChild(a);
  a.click();
  a.remove();
  setTimeout(() => URL.revokeObjectURL(url), 10000);
}
