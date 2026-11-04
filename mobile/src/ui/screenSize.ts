import { Platform, useWindowDimensions } from 'react-native';

/** On a big screen the web app stays a phone-width column (app/_layout.tsx). */
export const WEB_WIDTH = 520;

/** The room the app actually has, which on the web isn't the whole window. */
export function useScreenSize() {
  const { width, height } = useWindowDimensions();
  return { width: Platform.OS === 'web' ? Math.min(width, WEB_WIDTH) : width, height };
}
