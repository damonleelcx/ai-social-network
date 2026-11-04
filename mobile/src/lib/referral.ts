import { Platform } from 'react-native';

const KEY = 'synagent.ref';

/** The code someone arrived with: /app/?ref=CODE. Kept until they sign up, even across a reload. */
export function pendingReferral(): string {
  if (Platform.OS !== 'web') return '';
  try {
    return window.localStorage.getItem(KEY) ?? '';
  } catch {
    return '';
  }
}

/** Once they've signed up, so that the next person on this browser doesn't inherit it. */
export function clearReferral() {
  if (Platform.OS !== 'web') return;
  try {
    window.localStorage.removeItem(KEY);
  } catch {
    // Nothing to clear.
  }
}

// At import, because the router drops the query when it redirects to sign-in.
if (Platform.OS === 'web') {
  try {
    const ref = new URLSearchParams(window.location.search).get('ref');
    if (ref) window.localStorage.setItem(KEY, ref);
  } catch {
    // Without storage they can still type the code in.
  }
}
