import { useEffect, useState } from 'react';
import { View } from 'react-native';
import { auth } from '@/api/endpoints';
import { useSession } from '@/state/session';
import { space } from '@/theme/tokens';
import { CodeBoxes } from '@/ui/CodeBoxes';
import { Backdrop, Page } from '@/ui/layout';
import { Button, ErrorText, T } from '@/ui/primitives';
import { SukiAvatar } from '@/ui/suki';
import { tr } from '@/i18n';

const COOLDOWN = 30;

export default function Verify() {
  const { user, refreshMe, signOut } = useSession();
  const [code, setCode] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [sent, setSent] = useState(false);
  const [cooldown, setCooldown] = useState(COOLDOWN);

  useEffect(() => {
    if (cooldown <= 0) return;
    const id = setTimeout(() => setCooldown((c) => c - 1), 1000);
    return () => clearTimeout(id);
  }, [cooldown]);

  const submit = async (value: string) => {
    if (value.length !== 6 || busy) return;
    setBusy(true);
    setError(null);
    try {
      await auth.verify(value);
      await refreshMe();
    } catch (e: any) {
      setError(e.message);
      setCode('');
    } finally {
      setBusy(false);
    }
  };

  const resend = async () => {
    setError(null);
    setSent(false);
    try {
      await auth.resend();
      setSent(true);
      setCooldown(COOLDOWN);
    } catch (e: any) {
      setError(e.message);
    }
  };

  return (
    <Backdrop>
      <Page contentStyle={{ paddingTop: 96, alignItems: 'stretch' }}>
        <View style={{ alignItems: 'center', gap: space.md }}>
          <SukiAvatar mood="happy" size={104} ring={3} />
          <T v="title" center>{tr('Check your email')}</T>
          <T soft center>{tr('I sent a 6-digit code to {value}. Type it here and we can start.', { value: user?.email ?? tr('your email') })}</T>
        </View>
        <CodeBoxes
          value={code}
          autoFocus
          onChange={(v) => {
            setCode(v);
            if (v.length === 6) submit(v);
          }}
        />
        <ErrorText message={error} />
        {sent ? (
          <T v="small" soft center>{tr('A new code is on its way.')}</T>
        ) : null}
        <Button label={tr('Confirm')} full loading={busy} disabled={code.length !== 6} onPress={() => submit(code)} />
        <Button
          label={cooldown > 0 ? tr('Send a new code in {cooldown}s', { cooldown: cooldown }) : tr('Send a new code')}
          kind="secondary"
          full
          disabled={cooldown > 0}
          onPress={resend}
        />
        <Button label={tr('Use a different account')} kind="ghost" full onPress={signOut} />
      </Page>
    </Backdrop>
  );
}
