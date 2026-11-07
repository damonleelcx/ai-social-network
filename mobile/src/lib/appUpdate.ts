import Constants from 'expo-constants';
import { Platform } from 'react-native';
import { isPlayBuild } from './playBilling';

const LATEST_RELEASE = 'https://api.github.com/repos/damonleelcx/ai-social-network/releases/latest';
export const DOWNLOAD_PAGE = 'https://match.heros-agent.space/#download';

export type AppRelease = { version: string; apkUrl: string | null };

/** "1.10.0" is newer than "1.9.0", which string order gets wrong. */
export function isNewer(candidate: string, installed: string) {
  const a = candidate.split('.').map(Number);
  const b = installed.split('.').map(Number);
  for (let i = 0; i < Math.max(a.length, b.length); i++) {
    const x = a[i] ?? 0;
    const y = b[i] ?? 0;
    if (Number.isNaN(x) || Number.isNaN(y)) return false;
    if (x !== y) return x > y;
  }
  return false;
}

/**
 * The newer release, if there is one. Any failure means no prompt: this must never get in the way.
 * Google Play updates its own build. Our APK carries a different signature, so Android wouldn't
 * install it over that build, and Play doesn't allow updates from anywhere else.
 */
export async function findUpdate(): Promise<AppRelease | null> {
  const installed = Constants.expoConfig?.version;
  if (__DEV__ || !installed || isPlayBuild) return null;
  const controller = new AbortController();
  const timer = setTimeout(() => controller.abort(), 8000);
  try {
    const res = await fetch(LATEST_RELEASE, { signal: controller.signal, headers: { Accept: 'application/vnd.github+json' } });
    if (!res.ok) return null;
    const release: { tag_name?: string; assets?: { name: string; browser_download_url: string }[] } = await res.json();
    const version = release.tag_name?.replace(/^app-v/, '');
    if (!version || !isNewer(version, installed)) return null;
    const apk = release.assets?.find((a) => a.name === 'SynAgent.apk');
    return { version, apkUrl: Platform.OS === 'android' ? (apk?.browser_download_url ?? null) : null };
  } catch {
    return null;
  } finally {
    clearTimeout(timer);
  }
}
