import { saveJson } from '@/lib/saveJson';
import { useCallback, useEffect, useState } from 'react';
import { View } from 'react-native';
import { API_URL, legalUrl } from '@/api/client';
import * as api from '@/api/endpoints';
import { googleIdToken } from '@/lib/googleSignIn';
import type { Session } from '@/api/types';
import { shortTime } from '@/lib/format';
import { openLegal } from '@/lib/links';
import { useSession } from '@/state/session';
import { space } from '@/theme/tokens';
import { Group, NavRow } from '@/ui/form';
import { Backdrop, Page, ScreenHeader, Section } from '@/ui/layout';
import { Button, ErrorText, Field, Glass, Pill, T } from '@/ui/primitives';
import { confirm, Sheet, showError } from '@/ui/sheet';
import { tr } from '@/i18n';

function describeAgent(ua: string) {
  if (!ua) return tr('Unknown device');
  if (/okhttp|android/i.test(ua)) return tr('Android phone');
  if (/darwin|cfnetwork|iphone|ios/i.test(ua)) return tr('iPhone');
  return ua.length > 40 ? `${ua.slice(0, 40)}...` : ua;
}

function PasswordSheet({ visible, onClose }: { visible: boolean; onClose: () => void }) {
  const [current, setCurrent] = useState('');
  const [next, setNext] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [done, setDone] = useState(false);
  const close = () => {
    setCurrent('');
    setNext('');
    setError(null);
    setDone(false);
    onClose();
  };
  return (
    <Sheet visible={visible} onClose={close} title={tr('Change password')}>
      {done ? (
        <>
          <Pill tone="good" icon="checkmark-circle" label={tr('Changed. Your other devices were signed out.')} />
          <Button label={tr('Done')} full onPress={close} />
        </>
      ) : (
        <>
          <Field label={tr('Current password')} value={current} onChangeText={setCurrent} secure autoCapitalize="none" autoComplete="current-password" />
          <Field label={tr('New password')} value={next} onChangeText={setNext} secure autoCapitalize="none" autoComplete="new-password" hint={tr('At least 10 characters.')} />
          <ErrorText message={error} />
          <Button
            label={tr('Change password')}
            full
            loading={busy}
            disabled={!current || !next}
            onPress={async () => {
              setBusy(true);
              setError(null);
              try {
                await api.auth.changePassword(current, next);
                setDone(true);
              } catch (e: any) {
                setError(e.message);
              } finally {
                setBusy(false);
              }
            }}
          />
        </>
      )}
    </Sheet>
  );
}

function DeleteSheet({ visible, onClose }: { visible: boolean; onClose: () => void }) {
  const { forget, user } = useSession();
  const passwordless = user?.has_password === false;
  const [password, setPassword] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  return (
    <Sheet visible={visible} onClose={onClose} title={tr('Delete your account')}>
      <T soft>{tr("This removes your profile, photos, matches and chats for good. It can't be undone. Type your password to confirm.")}</T>
      {passwordless ? <T soft>{tr('Confirm with your Google account to finish.')}</T> : null}
      {passwordless ? null : <Field label={tr('Password')} value={password} onChangeText={setPassword} secure autoCapitalize="none" autoComplete="current-password" /> }
      <ErrorText message={error} />
      <Button
        label={tr('Delete my account')}
        kind="danger"
        full
        loading={busy}
        disabled={!passwordless && !password}
        onPress={async () => {
          setBusy(true);
          setError(null);
          try {
            if (passwordless) {
              const token = await googleIdToken();
              if (!token) {
                setBusy(false);
                return;
              }
              await api.auth.deleteWithGoogle(token);
            } else {
              await api.auth.deleteAccount(password);
            }
            // Close first: a modal left open would sit on top of the welcome screen.
            onClose();
            await forget();
          } catch (e: any) {
            setError(e.message);
            setBusy(false);
          }
        }}
      />
      <Button label={tr('Keep my account')} kind="ghost" full onPress={onClose} />
    </Sheet>
  );
}

export default function AccountSettings() {
  const { user, signOut, forget } = useSession();
  const [sessions, setSessions] = useState<Session[] | null>(null);
  const [sessionError, setSessionError] = useState<string | null>(null);
  const [passwordOpen, setPasswordOpen] = useState(false);
  const [deleteOpen, setDeleteOpen] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [signingOut, setSigningOut] = useState(false);

  const loadSessions = useCallback(async () => {
    try {
      setSessions(await api.auth.sessions());
      setSessionError(null);
    } catch (e: any) {
      setSessionError(e.message);
    }
  }, []);

  useEffect(() => {
    loadSessions();
  }, [loadSessions]);

  const revoke = async (s: Session) => {
    const body = s.current ? tr('This signs you out on this phone.') : tr('That device gets signed out right away.');
    if (!(await confirm(tr('Sign out this session?'), body, tr('Sign out'), true))) return;
    try {
      await api.auth.revokeSession(s.id);
      if (s.current) await forget();
      else await loadSessions();
    } catch (e) {
      showError(e);
    }
  };

  const exportData = async () => {
    setExporting(true);
    try {
      const data = await api.auth.exportData();
      const json = JSON.stringify(data, null, 2);
      await saveJson('synagent-data.json', json, tr('Your SynAgent data'));
    } catch (e) {
      showError(e);
    } finally {
      setExporting(false);
    }
  };

  return (
    <Backdrop>
      <ScreenHeader title={tr('Account')} />
      <Page refreshing={false} onRefresh={loadSessions}>
        <Section title={tr('Signed in as')}>
          <Glass>
            <T>{user?.email}</T>
          </Glass>
        </Section>

        {user?.has_password === false ? null : (
          <Group>
            <NavRow icon="key-outline" label={tr('Change password')} onPress={() => setPasswordOpen(true)} />
          </Group>
        )}

        <Section title={tr('Active sessions')}>
          <ErrorText message={sessionError} />
          {(sessions ?? []).map((s) => (
            <Glass key={s.id} style={{ flexDirection: 'row', alignItems: 'center', gap: space.md }}>
              <View style={{ flex: 1, gap: 2 }}>
                <View style={{ flexDirection: 'row', gap: space.sm, alignItems: 'center', flexWrap: 'wrap' }}>
                  <T v="subheading">{describeAgent(s.user_agent)}</T>
                  {s.current ? <Pill tone="good" label={tr('This phone')} /> : null}
                </View>
                <T v="caption" faint>{tr('Last seen {time} {value}', { time: shortTime(s.last_seen_at), value: s.ip ? ` · ${s.ip}` : '' })}</T>
              </View>
              <Button label={tr('Revoke')} small kind="secondary" onPress={() => revoke(s)} />
            </Glass>
          ))}
        </Section>

        <Section title={tr('Your data')} hint={tr("A JSON file of your profile, your words and Suki's notes.")}>
          <Button label={tr('Export my data')} icon="download-outline" kind="secondary" full loading={exporting} onPress={exportData} />
        </Section>

        <Section title={tr('Legal')}>
          <Group>
            <NavRow label={tr('Terms')} onPress={() => openLegal(legalUrl('terms'))} />
            <NavRow label={tr('Privacy policy')} onPress={() => openLegal(legalUrl('privacy'))} />
            <NavRow label={tr('Contact')} onPress={() => openLegal(legalUrl('contact'))} />
            <NavRow label={tr('How account deletion works')} onPress={() => openLegal(legalUrl('delete-account'))} />
          </Group>
        </Section>

        <Button
          label={tr('Sign out')}
          kind="secondary"
          full
          loading={signingOut}
          onPress={async () => {
            setSigningOut(true);
            await signOut();
          }}
        />
        <Button label={tr('Delete my account')} kind="danger" full onPress={() => setDeleteOpen(true)} />
        {__DEV__ ? (
          <T v="caption" faint center>{tr('API: {API_URL}', { API_URL: API_URL })}</T>
        ) : null}
      </Page>

      <PasswordSheet visible={passwordOpen} onClose={() => setPasswordOpen(false)} />
      <DeleteSheet visible={deleteOpen} onClose={() => setDeleteOpen(false)} />
    </Backdrop>
  );
}
