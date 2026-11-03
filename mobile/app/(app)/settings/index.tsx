import { router } from 'expo-router';
import { Group, NavRow } from '@/ui/form';
import { Backdrop, Page, ScreenHeader } from '@/ui/layout';
import { tr } from '@/i18n';

export default function Settings() {
  return (
    <Backdrop>
      <ScreenHeader title={tr('Settings')} />
      <Page>
        <Group>
          <NavRow icon="phone-portrait-outline" label={tr('App')} hint={tr('Language, theme, notifications, read receipts')} onPress={() => router.push('/settings/app')} />
          <NavRow icon="sparkles-outline" label="Suki" hint={tr('How she talks, what she remembers, coaching')} onPress={() => router.push('/settings/suki')} />
          <NavRow icon="key-outline" label={tr('Account')} hint={tr('Password, sessions, your data, legal')} onPress={() => router.push('/settings/account')} />
        </Group>
      </Page>
    </Backdrop>
  );
}
