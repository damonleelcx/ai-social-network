import { File, Paths } from 'expo-file-system';
import * as Sharing from 'expo-sharing';
import { useCallback, useEffect, useState } from 'react';
import { Share, View } from 'react-native';
import { API_URL, legalUrl } from '@/api/client';
import * as api from '@/api/endpoints';
import type { Session } from '@/api/types';
import { shortTime } from '@/lib/format';
import { openLegal } from '@/lib/links';
import { useSession } from '@/state/session';
import { space } from '@/theme/tokens';
import { Group, NavRow } from '@/ui/form';
import { Backdrop, Page, ScreenHeader, Section } from '@/ui/layout';
import { Button, ErrorText, Field, Glass, Pill, T } from '@/ui/primitives';
import { confirm, Sheet, showError } from '@/ui/sheet';

function describeAgent(ua: string) {
  if (!ua) return 'Unknown device';
  if (/okhttp|android/i.test(ua)) return 'Android phone';
  if (/darwin|cfnetwork|iphone|ios/i.test(ua)) return 'iPhone';
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
    <Sheet visible={visible} onClose={close} title="Change password">
      {done ? (
        <>
          <Pill tone="good" icon="checkmark-circle" label="Changed. Your other devices were signed out." />
          <Button label="Done" full onPress={close} />
        </>
      ) : (
        <>
          <Field label="Current password" value={current} onChangeText={setCurrent} secure autoCapitalize="none" autoComplete="current-password" />
          <Field label="New password" value={next} onChangeText={setNext} secure autoCapitalize="none" autoComplete="new-password" hint="At least 10 characters." />
          <ErrorText message={error} />
          <Button
            label="Change password"
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
  const { forget } = useSession();
  const [password, setPassword] = useState('');
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  return (
    <Sheet visible={visible} onClose={onClose} title="Delete your account">
      <T soft>This removes your profile, photos, matches and chats for good. It can't be undone. Type your password to confirm.</T>
      <Field label="Password" value={password} onChangeText={setPassword} secure autoCapitalize="none" autoComplete="current-password" />
      <ErrorText message={error} />
      <Button
        label="Delete my account"
        kind="danger"
        full
        loading={busy}
        disabled={!password}
        onPress={async () => {
          setBusy(true);
          setError(null);
          try {
            await api.auth.deleteAccount(password);
            // Close first: a modal left open would sit on top of the welcome screen.
            onClose();
            await forget();
          } catch (e: any) {
            setError(e.message);
            setBusy(false);
          }
        }}
      />
      <Button label="Keep my account" kind="ghost" full onPress={onClose} />
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
    const body = s.current ? 'This signs you out on this phone.' : 'That device gets signed out right away.';
    if (!(await confirm('Sign out this session?', body, 'Sign out', true))) return;
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
      const file = new File(Paths.cache, 'synagent-data.json');
      if (file.exists) file.delete();
      file.create();
      file.write(json);
      if (await Sharing.isAvailableAsync()) {
        await Sharing.shareAsync(file.uri, { mimeType: 'application/json', dialogTitle: 'Your SynAgent data' });
      } else {
        await Share.share({ message: json });
      }
    } catch (e) {
      showError(e);
    } finally {
      setExporting(false);
    }
  };

  return (
    <Backdrop>
      <ScreenHeader title="Account" />
      <Page refreshing={false} onRefresh={loadSessions}>
        <Section title="Signed in as">
          <Glass>
            <T>{user?.email}</T>
          </Glass>
        </Section>

        <Group>
          <NavRow icon="key-outline" label="Change password" onPress={() => setPasswordOpen(true)} />
        </Group>

        <Section title="Active sessions">
          <ErrorText message={sessionError} />
          {(sessions ?? []).map((s) => (
            <Glass key={s.id} style={{ flexDirection: 'row', alignItems: 'center', gap: space.md }}>
              <View style={{ flex: 1, gap: 2 }}>
                <View style={{ flexDirection: 'row', gap: space.sm, alignItems: 'center', flexWrap: 'wrap' }}>
                  <T v="subheading">{describeAgent(s.user_agent)}</T>
                  {s.current ? <Pill tone="good" label="This phone" /> : null}
                </View>
                <T v="caption" faint>
                  Last seen {shortTime(s.last_seen_at)}
                  {s.ip ? ` · ${s.ip}` : ''}
                </T>
              </View>
              <Button label="Revoke" small kind="secondary" onPress={() => revoke(s)} />
            </Glass>
          ))}
        </Section>

        <Section title="Your data" hint="A JSON file of your profile, your words and Suki's notes.">
          <Button label="Export my data" icon="download-outline" kind="secondary" full loading={exporting} onPress={exportData} />
        </Section>

        <Section title="Legal">
          <Group>
            <NavRow label="Terms" onPress={() => openLegal(legalUrl('terms'))} />
            <NavRow label="Privacy policy" onPress={() => openLegal(legalUrl('privacy'))} />
            <NavRow label="How account deletion works" onPress={() => openLegal(legalUrl('delete-account'))} />
          </Group>
        </Section>

        <Button
          label="Sign out"
          kind="secondary"
          full
          loading={signingOut}
          onPress={async () => {
            setSigningOut(true);
            await signOut();
          }}
        />
        <Button label="Delete my account" kind="danger" full onPress={() => setDeleteOpen(true)} />
        {__DEV__ ? (
          <T v="caption" faint center>
            API: {API_URL}
          </T>
        ) : null}
      </Page>

      <PasswordSheet visible={passwordOpen} onClose={() => setPasswordOpen(false)} />
      <DeleteSheet visible={deleteOpen} onClose={() => setDeleteOpen(false)} />
    </Backdrop>
  );
}
