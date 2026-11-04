import { Ionicons } from '@expo/vector-icons';
import { router } from 'expo-router';
import { useState } from 'react';
import { Pressable, View } from 'react-native';
import { legalUrl } from '@/api/client';
import { auth } from '@/api/endpoints';
import { openLegal } from '@/lib/links';
import { GoogleSignInButton } from '@/features/GoogleSignInButton';
import { useSession } from '@/state/session';
import { useTheme } from '@/theme/ThemeProvider';
import { space } from '@/theme/tokens';
import { Backdrop, Page, ScreenHeader } from '@/ui/layout';
import { Button, ErrorText, Field, T } from '@/ui/primitives';
import { currentLanguage, tr } from '@/i18n';

export default function SignUp() {
  const t = useTheme();
  const { adoptToken } = useSession();
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [accepted, setAccepted] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const submit = async () => {
    setBusy(true);
    setError(null);
    try {
      const r = await auth.signUp({
        name: name.trim(),
        email: email.trim(),
        password,
        accept_terms: accepted,
        language: currentLanguage(),
      });
      await adoptToken(r.token);
    } catch (e: any) {
      setError(e.message);
    } finally {
      setBusy(false);
    }
  };

  return (
    <Backdrop>
      <ScreenHeader title={tr('Nice to meet you')} />
      <Page>
        <T soft>{tr("A few details and you're in. The rest we'll cover by chatting.")}</T>
        <Field
          label={tr('Your first name')}
          value={name}
          onChangeText={setName}
          autoComplete="given-name"
          textContentType="givenName"
          autoCapitalize="words"
          maxLength={40}
          returnKeyType="next"
        />
        <Field
          label={tr('Email')}
          value={email}
          onChangeText={setEmail}
          autoComplete="email"
          textContentType="emailAddress"
          keyboardType="email-address"
          autoCapitalize="none"
          autoCorrect={false}
          returnKeyType="next"
        />
        <Field
          label={tr('Password')}
          value={password}
          onChangeText={setPassword}
          secure
          autoComplete="new-password"
          textContentType="newPassword"
          autoCapitalize="none"
          hint={tr('At least 10 characters.')}
        />
        <Pressable
          accessibilityRole="checkbox"
          accessibilityState={{ checked: accepted }}
          accessibilityLabel={tr('I am 18 or older and I agree to the terms and privacy policy')}
          onPress={() => setAccepted((a) => !a)}
          style={{ flexDirection: 'row', gap: space.md, alignItems: 'flex-start', paddingVertical: space.xs }}
        >
          <Ionicons name={accepted ? 'checkbox' : 'square-outline'} size={26} color={accepted ? t.accent : t.inkFaint} />
          <T v="small" soft style={{ flex: 1 }}>
            {tr("I'm 18 or older, and I agree to the {terms} and {privacy}.")
              .split(/(\{terms\}|\{privacy\})/)
              .map((part) =>
                part === '{terms}' ? (
                  <T key={part} v="small" color={t.accent} style={{ fontWeight: '700' }} onPress={() => openLegal(legalUrl('terms'))}>
                    {tr('terms')}
                  </T>
                ) : part === '{privacy}' ? (
                  <T key={part} v="small" color={t.accent} style={{ fontWeight: '700' }} onPress={() => openLegal(legalUrl('privacy'))}>
                    {tr('privacy policy')}
                  </T>
                ) : (
                  part
                ),
              )}
          </T>
        </Pressable>
        <ErrorText message={error} />
        <View style={{ gap: space.sm }}>
          <Button
            label={tr('Create my account')}
            full
            loading={busy}
            disabled={!name.trim() || !email.trim() || !password || !accepted}
            onPress={submit}
          />
          <GoogleSignInButton />
          <Button label={tr('I already have an account')} kind="ghost" full onPress={() => router.replace('/signin')} />
        </View>
      </Page>
    </Backdrop>
  );
}
