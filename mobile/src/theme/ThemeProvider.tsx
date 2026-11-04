import * as SystemUI from 'expo-system-ui';
import { createContext, useContext, useEffect, useMemo, type ReactNode } from 'react';
import { Appearance, Platform, useColorScheme } from 'react-native';
import { useSession } from '@/state/session';
import { dark, light, type Tokens } from './tokens';

const Ctx = createContext<Tokens>(light);

export function ThemeProvider({ children }: { children: ReactNode }) {
  const { settings, cachedTheme, language } = useSession();
  const choice = settings?.theme ?? cachedTheme ?? 'system';

  // Setting it at the OS level means alerts and native pickers follow too.
  useEffect(() => {
    // Browsers have no such setting.
    Appearance.setColorScheme?.(choice === 'system' ? 'unspecified' : choice);
  }, [choice]);

  const system = useColorScheme();
  const scheme = choice === 'system' ? system : choice;
  // A new object when the language changes: every screen reads this context, so they all
  // render again with the new copy, and the navigator stays where it is.
  // eslint-disable-next-line react-hooks/exhaustive-deps
  const tokens = useMemo(() => ({ ...(scheme === 'dark' ? dark : light) }), [scheme, language]);

  // Scrollbars and the date input are the browser's own.
  useEffect(() => {
    if (Platform.OS === 'web') document.documentElement.style.colorScheme = scheme === 'dark' ? 'dark' : 'light';
  }, [scheme]);

  // The window behind the app shows during transitions and while the keyboard moves.
  useEffect(() => {
    SystemUI.setBackgroundColorAsync(tokens.bg[0]).catch(() => {});
  }, [tokens]);

  return <Ctx.Provider value={tokens}>{children}</Ctx.Provider>;
}

export function useTheme() {
  return useContext(Ctx);
}
