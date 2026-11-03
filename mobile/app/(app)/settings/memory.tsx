import { useCallback, useEffect, useState } from 'react';
import { View } from 'react-native';
import * as api from '@/api/endpoints';
import type { Memory } from '@/api/types';
import { useSession } from '@/state/session';
import { space } from '@/theme/tokens';
import { Backdrop, Page, ScreenHeader } from '@/ui/layout';
import { Button, ErrorText, Glass, IconButton, T } from '@/ui/primitives';
import { confirm, showError } from '@/ui/sheet';
import { SukiLoading, SukiSays } from '@/ui/suki';
import { tr } from '@/i18n';

export default function MemoryScreen() {
  const { settings } = useSession();
  const [items, setItems] = useState<Memory[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [resetting, setResetting] = useState(false);

  const load = useCallback(async () => {
    try {
      setItems(await api.memories.list());
      setError(null);
    } catch (e: any) {
      setError(e.message);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const remove = async (m: Memory) => {
    setItems((all) => all?.filter((x) => x.id !== m.id) ?? null);
    try {
      await api.memories.remove(m.id);
    } catch (e) {
      showError(e);
      load();
    }
  };

  const reset = async () => {
    const yes = await confirm(tr('Start over?'), tr("I'll forget my notes about you and everything I've learned about your taste. This can't be undone."), tr('Forget it all'), true);
    if (!yes) return;
    setResetting(true);
    try {
      await api.memories.reset();
      await load();
    } catch (e) {
      showError(e);
    } finally {
      setResetting(false);
    }
  };

  return (
    <Backdrop>
      <ScreenHeader title={tr('What Suki remembers')} />
      <Page refreshing={false} onRefresh={load}>
        <ErrorText message={error} />
        {settings && !settings.memory_enabled ? (
          <Glass>
            <T v="small" soft>{tr("Long-term memory is off, so I'm not adding anything new.")}</T>
          </Glass>
        ) : null}
        {items === null && !error ? (
          <SukiLoading line={tr('Going through my notes...')} />
        ) : items?.length === 0 ? (
          <SukiSays face="thinking" title={tr('A blank page')} body={tr("I haven't written anything down yet. As we talk, my notes will show up here, and you can delete any of them.")} />
        ) : (
          items?.map((m) => (
            <Glass key={m.id} style={{ flexDirection: 'row', alignItems: 'center', gap: space.sm, paddingRight: space.sm }}>
              <T v="small" style={{ flex: 1 }}>
                {m.note}
              </T>
              <IconButton icon="trash-outline" size={19} label={tr('Delete note: {note}', { note: m.note })} onPress={() => remove(m)} />
            </Glass>
          ))
        )}
        <View style={{ marginTop: space.md }}>
          <Button label={tr('Forget everything and start over')} kind="danger" full loading={resetting} onPress={reset} />
        </View>
      </Page>
    </Backdrop>
  );
}
