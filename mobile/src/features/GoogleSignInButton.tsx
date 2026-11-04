import { useState } from 'react';
import * as api from '@/api/endpoints';
import { ApiError } from '@/api/client';
import { googleIdToken, googleSignInAvailable } from '@/lib/googleSignIn';
import { useSession } from '@/state/session';
import { Button, ErrorText } from '@/ui/primitives';
import { confirm } from '@/ui/sheet';
import { tr } from '@/i18n';

/** Signs in, or signs up for someone new once they've accepted the terms. */
export function GoogleSignInButton() {
  const { adoptToken } = useSession();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  if (!googleSignInAvailable) return null;

  const press = async () => {
    setBusy(true);
    setError(null);
    try {
      const token = await googleIdToken();
      if (!token) return;
      let r;
      try {
        r = await api.auth.google(token, false);
      } catch (e) {
        if (!(e instanceof ApiError) || e.message !== tr('please accept the terms to continue')) throw e;
        const yes = await confirm(
          tr('Welcome to SynAgent'),
          tr('By continuing, you confirm you are 18 or over and accept the Terms of Use and Privacy Policy.'),
          tr('Continue'),
        );
        if (!yes) return;
        r = await api.auth.google(token, true);
      }
      await adoptToken(r.token);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <ErrorText message={error} />
      <Button label={tr('Continue with Google')} icon="logo-google" kind="secondary" full loading={busy} onPress={press} />
    </>
  );
}
