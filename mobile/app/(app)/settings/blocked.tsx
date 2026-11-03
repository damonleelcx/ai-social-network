import { useCallback, useEffect, useState } from 'react';
import { View } from 'react-native';
import * as api from '@/api/endpoints';
import type { Blocked } from '@/api/types';
import { useLive } from '@/state/live';
import { space } from '@/theme/tokens';
import { Backdrop, Page, ScreenHeader } from '@/ui/layout';
import { Button, ErrorText, Glass, T } from '@/ui/primitives';
import { confirm, showError } from '@/ui/sheet';
import { SukiLoading, SukiSays } from '@/ui/suki';

export default function BlockedScreen() {
  const live = useLive();
  const [items, setItems] = useState<Blocked[] | null>(null);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      setItems(await api.safety.blocked());
      setError(null);
    } catch (e: any) {
      setError(e.message);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  return (
    <Backdrop>
      <ScreenHeader title="Blocked people" />
      <Page refreshing={false} onRefresh={load}>
        <ErrorText message={error} />
        {items === null && !error ? (
          <SukiLoading />
        ) : items?.length === 0 ? (
          <SukiSays face="happy" title="Nobody blocked" body="If anyone ever makes you uncomfortable, block them from the chat menu. They won't be told." />
        ) : (
          items?.map((b) => (
            <Glass key={b.user_id} style={{ flexDirection: 'row', alignItems: 'center', gap: space.md }}>
              <View style={{ flex: 1 }}>
                <T v="subheading">{b.display_name}</T>
              </View>
              <Button
                label="Unblock"
                small
                kind="secondary"
                onPress={async () => {
                  const yes = await confirm(`Unblock ${b.display_name}?`, 'Your old match won’t come back, but you could be introduced again.', 'Unblock');
                  if (!yes) return;
                  try {
                    await api.safety.unblock(b.user_id);
                    await load();
                    live.refreshProposals();
                  } catch (e) {
                    showError(e);
                  }
                }}
              />
            </Glass>
          ))
        )}
      </Page>
    </Backdrop>
  );
}
