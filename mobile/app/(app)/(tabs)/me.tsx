import { LinearGradient } from 'expo-linear-gradient';
import { router } from 'expo-router';
import { useCallback, useEffect, useState } from 'react';
import { View } from 'react-native';
import { Share } from '@/lib/dialogs';
import * as api from '@/api/endpoints';
import { missingLabel } from '@/lib/format';
import { useLive, useTopic } from '@/state/live';
import { useSession } from '@/state/session';
import { useTheme } from '@/theme/ThemeProvider';
import { radius, space } from '@/theme/tokens';
import { Group, NavRow, ToggleRow } from '@/ui/form';
import { Backdrop, Page, TabHeader, ThemeToggle } from '@/ui/layout';
import { PersonAvatar } from '@/ui/person';
import { Button, Glass, Pill, T } from '@/ui/primitives';
import { showError } from '@/ui/sheet';
import { SukiAvatar } from '@/ui/suki';
import { tr } from '@/i18n';

const DOWNLOAD_PAGE = 'https://match.heros-agent.space/#download';

async function inviteFriends() {
  try {
    await Share.share({ message: tr('Come try SynAgent with me. Suki, an AI wingwoman, introduces you to people worth meeting. Download it here: {url}', { url: DOWNLOAD_PAGE }) });
  } catch (e) {
    showError(e);
  }
}

/** Nine things make a profile whole on the server (see profile.Missing). */
const PROFILE_STEPS = 9;

function Completeness({ missing }: { missing: string[] }) {
  const t = useTheme();
  const waitingOnReview = missing.some((m) => m.includes('review'));
  const left = missing.filter((m) => !m.includes('review')).length;
  const filled = Math.max(0, Math.min(1, (PROFILE_STEPS - missing.length) / PROFILE_STEPS));
  if (!missing.length) return null;
  return (
    <Glass style={{ gap: space.md }}>
      <View style={{ flexDirection: 'row', gap: space.md, alignItems: 'center' }}>
        <SukiAvatar mood="thinking" size={44} />
        <View style={{ flex: 1 }}>
          <T v="subheading">{left === 0 && waitingOnReview ? tr('Almost there') : tr('A little more and we can start')}</T>
          <T v="small" soft>
            {left === 0 && waitingOnReview ? tr("I'm checking your photo.") : tr("I can't introduce you until these are done.")}
          </T>
        </View>
      </View>
      <View style={{ height: 8, borderRadius: 4, backgroundColor: t.accentSoft, overflow: 'hidden' }}>
        <LinearGradient
          colors={t.iris}
          start={{ x: 0, y: 0 }}
          end={{ x: 1, y: 0 }}
          style={{ height: 8, width: `${Math.round(filled * 100)}%`, borderRadius: 4 }}
        />
      </View>
      <View style={{ flexDirection: 'row', flexWrap: 'wrap', gap: 6 }}>
        {missing.map((m) => (
          <Pill key={m} tone={m.includes('review') ? 'warn' : 'plain'} label={missingLabel(m)} />
        ))}
      </View>
      {left > 0 ? (
        <View style={{ flexDirection: 'row', gap: space.sm, flexWrap: 'wrap' }}>
          <Button label={tr('Finish with Suki')} small onPress={() => router.navigate('/suki')} />
          <Button label={tr('Fill it in myself')} small kind="secondary" onPress={() => router.push('/profile/edit')} />
        </View>
      ) : null}
    </Glass>
  );
}

const STATUS: Record<string, { label: string; tone: 'good' | 'warn' | 'plain' | 'bad' }> = {
  onboarding: { label: 'Getting set up', tone: 'plain' },
  active: { label: 'Visible', tone: 'good' },
  paused: { label: 'Paused', tone: 'warn' },
  held: { label: 'Under review', tone: 'bad' },
};

export default function Me() {
  const { user, profile, missing, refreshMe } = useSession();
  const live = useLive();
  const [refreshing, setRefreshing] = useState(false);
  const [pausing, setPausing] = useState(false);
  const [balance, setBalance] = useState<number | null>(null);
  const loadBalance = useCallback(() => {
    api.credits.status().then((s) => setBalance(s.balance), () => {});
  }, []);
  useEffect(loadBalance, [loadBalance]);
  useTopic('credits', loadBalance);

  const main = live.photos.find((p) => p.is_primary) ?? live.photos[0];
  const status = STATUS[profile?.status ?? ''];
  const place = [profile?.age ? String(profile.age) : '', profile?.city_name].filter(Boolean).join(' · ');

  return (
    <Backdrop>
      <TabHeader title={tr('Me')} right={<ThemeToggle />} />
      <Page
        bottomInset={false}
        contentStyle={{ paddingTop: space.xs }}
        refreshing={refreshing}
        onRefresh={async () => {
          setRefreshing(true);
          await Promise.allSettled([refreshMe(), live.refreshPhotos(), live.refreshNotifications()]);
          setRefreshing(false);
        }}
      >
        <Glass style={{ flexDirection: 'row', gap: space.lg, alignItems: 'center', borderRadius: radius.xl }}>
          {main ? <PersonAvatar id={main.id} size={88} /> : <SukiAvatar mood="pouty" size={88} ring={3} />}
          <View style={{ flex: 1, gap: 4 }}>
            <T v="heading">{profile?.display_name || user?.name || tr('You')}</T>
            {place ? (
              <T v="small" soft>
                {place}
              </T>
            ) : null}
            {!main ? (
              <T v="small" soft>{tr("No photo yet. That's my face, not yours.")}</T>
            ) : null}
            {status ? <Pill tone={status.tone} label={tr(status.label)} /> : null}
          </View>
        </Glass>

        <Completeness missing={missing} />

        {profile?.status === 'held' ? (
          <Glass>
            <T v="small" soft>{tr("A person on our team is reviewing your account. You won't get new introductions until they're done.")}</T>
          </Glass>
        ) : null}

        <Group>
          <NavRow icon="create-outline" label={tr('Edit profile')} hint={tr('Or just tell Suki')} onPress={() => router.push('/profile/edit')} />
          <NavRow
            icon="images-outline"
            label={tr('Photos')}
            hint={live.photos.length ? undefined : tr('Add your first one')}
            onPress={() => router.push('/profile/photos')}
          />
          <NavRow icon="notifications-outline" label={tr('Notifications')} badge={live.badges.me} onPress={() => router.push('/notifications')} />
          <NavRow icon="person-add-outline" label={tr('Invite friends')} hint={tr('Send them the download link')} onPress={inviteFriends} />
          <NavRow
            icon="card-outline"
            label={tr('Credits')}
            hint={balance === null ? undefined : tr('{n} credits left', { n: String(balance) })}
            onPress={() => router.push('/settings/credits')}
          />
          <NavRow icon="settings-outline" label={tr('Settings')} onPress={() => router.push('/settings')} />
        </Group>

        {profile && (profile.status === 'active' || profile.status === 'paused') ? (
          <Group>
            <ToggleRow
              label={tr('Pause introductions')}
              hint={tr('You keep your matches and chats. I just stop looking.')}
              value={profile.status === 'paused'}
              disabled={pausing}
              onChange={async (paused) => {
                setPausing(true);
                try {
                  await api.profile.pause(paused);
                  await refreshMe();
                } catch (e) {
                  showError(e);
                } finally {
                  setPausing(false);
                }
              }}
            />
          </Group>
        ) : null}
      </Page>
    </Backdrop>
  );
}
