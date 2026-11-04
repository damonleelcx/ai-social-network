import { router, type Href } from 'expo-router';
import { Linking, Platform } from 'react-native';
import * as WebBrowser from 'expo-web-browser';
import type { NotificationData, Venue } from '@/api/types';

/** A notification's `data.screen`, to wherever it points. */
export function openNotificationTarget(data?: NotificationData | null) {
  if (!data?.screen) return;
  switch (data.screen) {
    case 'agent':
      router.navigate('/suki');
      break;
    case 'proposal':
      if (data.id) router.push(`/person?proposal=${data.id}`);
      break;
    case 'chat':
      if (data.id) router.push(`/chat/${data.id}`);
      break;
    case 'date':
      if (data.id) router.push(`/date/${data.id}`);
      break;
  }
}

const SCREENS: Record<string, Href> = {
  settings: '/settings',
  settings_app: '/settings/app',
  settings_agent: '/settings/suki',
  settings_account: '/settings/account',
  settings_memory: '/settings/memory',
  settings_blocked: '/settings/blocked',
  profile_edit: '/profile/edit',
  photos: '/profile/photos',
  matches: '/matches',
  dates: '/dates',
};

/** Suki's `open_screen` cards. Returns false for a screen this version doesn't know. */
export function openNamedScreen(screen: string) {
  const href = SCREENS[screen];
  if (!href) return false;
  if (screen === 'matches' || screen === 'dates') router.navigate(href);
  else router.push(href);
  return true;
}

/** Payment happens on the provider's own page. On the web that replaces this one. */
export function openPayment(url: string) {
  if (Platform.OS === 'web') {
    window.location.assign(url);
    return Promise.resolve();
  }
  return WebBrowser.openBrowserAsync(url);
}

export function openLegal(url: string) {
  return WebBrowser.openBrowserAsync(url);
}

export async function openVenueOnMap(v: Venue) {
  const q = encodeURIComponent(v.name);
  const native =
    Platform.OS === 'ios'
      ? `http://maps.apple.com/?q=${q}&ll=${v.lat},${v.lon}`
      : `geo:${v.lat},${v.lon}?q=${v.lat},${v.lon}(${q})`;
  try {
    await Linking.openURL(native);
  } catch {
    await Linking.openURL(`https://www.google.com/maps/search/?api=1&query=${v.lat},${v.lon}`);
  }
}
