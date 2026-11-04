import { router, useLocalSearchParams } from 'expo-router';
import { useState } from 'react';
import { View } from 'react-native';
import { auth } from '@/api/endpoints';
import { GoogleSignInButton } from '@/features/GoogleSignInButton';
import { useSession } from '@/state/session';
import { space } from '@/theme/tokens';
import { Backdrop, Page, ScreenHeader } from '@/ui/layout';
import { Button, ErrorText, Field, Pill, T } from '@/ui/primitives';
import { SukiAvatar } from '@/ui/suki';
import { tr } from '@/i18n';

export default function SignIn() {
  const { adoptToken } = useSession();
  const params = useLocalSearchParams<{ reset?: string; email?: string }>();
  const [email, setEmail] = useState(params.email ?? '');
  const [password, setPassword] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const submit = async () => {
    setBusy(true);
    setError(null);
    try {
      const r = await auth.signIn(email.trim(), password);
      await adoptToken(r.token);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <Backdrop>
      <ScreenHeader title={tr('Welcome back')} />
      <Page>
        <View style={{ alignItems: 'center', gap: space.sm, paddingVertical: space.md }}>
          <SukiAvatar mood="playful" size={96} ring={3} />
          <T soft center>{tr("There you are. I was hoping you'd come back.")}</T>
        </View>
        {params.reset ? <Pill tone="good" icon="checkmark-circle" label={tr('Password changed. Sign in with the new one.')} /> : null}
        <Field
          label={tr('Email')}
          value={email}
          onChangeText={setEmail}
          autoComplete="email"
          textContentType="username"
          keyboardType="email-address"
          autoCapitalize="none"
          autoCorrect={false}
        />
        <Field
          label={tr('Password')}
          value={password}
          onChangeText={setPassword}
          secure
          autoComplete="current-password"
          textContentType="password"
          autoCapitalize="none"
          onSubmitEditing={submit}
        />
        <ErrorText message={error} />
        <View style={{ gap: space.sm }}>
          <GoogleSignInButton />
          <Button label={tr('Sign in')} full loading={busy} disabled={!email.trim() || !password} onPress={submit} />
          <Button label={tr('I forgot my password')} kind="ghost" full onPress={() => router.push('/forgot')} />
        </View>
      </Page>
    </Backdrop>
  );
}
