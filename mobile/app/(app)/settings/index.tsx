import { router } from 'expo-router';
import { Group, NavRow } from '@/ui/form';
import { Backdrop, Page, ScreenHeader } from '@/ui/layout';

export default function Settings() {
  return (
    <Backdrop>
      <ScreenHeader title="Settings" />
      <Page>
        <Group>
          <NavRow icon="phone-portrait-outline" label="App" hint="Language, theme, notifications, read receipts" onPress={() => router.push('/settings/app')} />
          <NavRow icon="sparkles-outline" label="Suki" hint="How she talks, what she remembers, coaching" onPress={() => router.push('/settings/suki')} />
          <NavRow icon="key-outline" label="Account" hint="Password, sessions, your data, legal" onPress={() => router.push('/settings/account')} />
        </Group>
      </Page>
    </Backdrop>
  );
}
