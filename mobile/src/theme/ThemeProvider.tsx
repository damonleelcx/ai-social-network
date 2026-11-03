import { createContext, useContext, useEffect, useMemo, type ReactNode } from 'react';
import { Appearance, useColorScheme } from 'react-native';
import { useSession } from '@/state/session';
import { dark, light, type Tokens } from './tokens';

const Ctx = createContext<Tokens>(light);

export function ThemeProvider({ children }: { children: ReactNode }) {
  const { settings, cachedTheme } = useSession();
  const choice = settings?.theme ?? cachedTheme ?? 'system';

  // Setting it at the OS level means alerts and native pickers follow too.
  useEffect(() => {
    Appearance.setColorScheme(choice === 'system' ? 'unspecified' : choice);
  }, [choice]);

  const system = useColorScheme();
  const scheme = choice === 'system' ? system : choice;
  const tokens = useMemo(() => (scheme === 'dark' ? dark : light), [scheme]);

  return <Ctx.Provider value={tokens}>{children}</Ctx.Provider>;
}

export function useTheme() {
  return useContext(Ctx);
}
