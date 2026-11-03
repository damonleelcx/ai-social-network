import { View } from 'react-native';
import type { Settings } from '@/api/types';
import { useSession } from '@/state/session';
import { Group, SingleSelect, ToggleRow } from '@/ui/form';
import { Backdrop, Page, ScreenHeader, Section } from '@/ui/layout';
import { Glass } from '@/ui/primitives';
import { showError } from '@/ui/sheet';
import { SukiLoading } from '@/ui/suki';
import { LANGUAGE_NAMES, tr } from '@/i18n';

const THEMES = [
  { value: 'system', label: 'Match my phone' },
  { value: 'light', label: 'Light' },
  { value: 'dark', label: 'Dark' },
];
const DAILY = ['1', '2', '3', '4', '5'].map((v) => ({ value: v, label: v }));

export default function AppSettings() {
  const { settings: s, updateSettings } = useSession();
  const set = (patch: Partial<Settings>) => updateSettings(patch).catch(showError);

  return (
    <Backdrop>
      <ScreenHeader title={tr('App')} />
      {!s ? (
        <View style={{ flex: 1, justifyContent: 'center' }}>
          <SukiLoading line={tr('Getting your settings...')} />
        </View>
      ) : (
        <Page>
          <Section title={tr('Language')} hint={tr('For the app and for Suki.')}>
            <Glass>
              <SingleSelect options={LANGUAGE_NAMES} value={s.language} onChange={(v) => set({ language: v as Settings['language'] })} />
            </Glass>
          </Section>
          <Section title={tr('Theme')}>
            <Glass>
              <SingleSelect options={THEMES} value={s.theme} onChange={(v) => set({ theme: v as Settings['theme'] })} />
            </Glass>
          </Section>
          <Section title={tr('Notify me about')}>
            <Group>
              <ToggleRow label={tr('New introductions and matches')} value={s.notify_matches} onChange={(v) => set({ notify_matches: v })} />
              <ToggleRow label={tr('Messages')} value={s.notify_messages} onChange={(v) => set({ notify_messages: v })} />
              <ToggleRow label={tr('Dates')} value={s.notify_dates} onChange={(v) => set({ notify_dates: v })} />
              <ToggleRow label={tr('Things Suki wants to tell me')} value={s.notify_suki} onChange={(v) => set({ notify_suki: v })} />
              <ToggleRow label={tr('Also by email')} value={s.notify_email} onChange={(v) => set({ notify_email: v })} />
            </Group>
          </Section>
          <Section title={tr('Chats')}>
            <Group>
              <ToggleRow label={tr('Read receipts')} hint={tr("People can see when you've read their message.")} value={s.read_receipts} onChange={(v) => set({ read_receipts: v })} />
            </Group>
          </Section>
          <Section title={tr('Introductions per day')} hint={tr('The most Suki will bring you in one day. She may bring fewer.')}>
            <Glass>
              <SingleSelect options={DAILY} value={String(s.daily_matches)} onChange={(v) => set({ daily_matches: Number(v) })} />
            </Glass>
          </Section>
        </Page>
      )}
    </Backdrop>
  );
}
