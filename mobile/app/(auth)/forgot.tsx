import { router } from 'expo-router';
import { useState } from 'react';
import { auth } from '@/api/endpoints';
import { CodeBoxes } from '@/ui/CodeBoxes';
import { Backdrop, Page, ScreenHeader } from '@/ui/layout';
import { Button, ErrorText, Field, T } from '@/ui/primitives';

export default function Forgot() {
  const [step, setStep] = useState<'email' | 'code'>('email');
  const [email, setEmail] = useState('');
  const [code, setCode] = useState('');
  const [password, setPassword] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const sendCode = async () => {
    setBusy(true);
    setError(null);
    try {
      await auth.forgot(email.trim());
      setStep('code');
    } catch (e: any) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };

  const reset = async () => {
    setBusy(true);
    setError(null);
    try {
      await auth.reset(email.trim(), code, password);
      router.replace({ pathname: '/signin', params: { reset: '1', email: email.trim() } });
    } catch (e: any) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <Backdrop>
      <ScreenHeader title="Forgot your password?" />
      <Page>
        {step === 'email' ? (
          <>
            <T soft>It happens. Tell me your email and I'll send you a code.</T>
            <Field
              label="Email"
              value={email}
              onChangeText={setEmail}
              autoComplete="email"
              keyboardType="email-address"
              autoCapitalize="none"
              autoCorrect={false}
              onSubmitEditing={sendCode}
            />
            <ErrorText message={error} />
            <Button label="Send me a code" full loading={busy} disabled={!email.trim()} onPress={sendCode} />
          </>
        ) : (
          <>
            <T soft>If {email.trim()} has an account, a 6-digit code is on its way. Enter it with your new password.</T>
            <CodeBoxes value={code} onChange={setCode} autoFocus />
            <Field
              label="New password"
              value={password}
              onChangeText={setPassword}
              secure
              autoComplete="new-password"
              textContentType="newPassword"
              autoCapitalize="none"
              hint="At least 10 characters."
            />
            <ErrorText message={error} />
            <Button label="Set new password" full loading={busy} disabled={code.length !== 6 || !password} onPress={reset} />
            <Button label="Use a different email" kind="ghost" full onPress={() => setStep('email')} />
          </>
        )}
      </Page>
    </Backdrop>
  );
}
