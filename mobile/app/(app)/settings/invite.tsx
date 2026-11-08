import { useCallback, useEffect, useState } from 'react';
import { View } from 'react-native';
import * as api from '@/api/endpoints';
import type { ReferralStatus } from '@/api/types';
import { Share } from '@/lib/dialogs';
import { sharePoster } from '@/lib/poster';
import { useTopic } from '@/state/live';
import { space } from '@/theme/tokens';
import { Backdrop, Page, ScreenHeader } from '@/ui/layout';
import { Button, ErrorText, Glass, T } from '@/ui/primitives';
import { showError } from '@/ui/sheet';
import { SukiLoading } from '@/ui/suki';
import { tr } from '@/i18n';

export default function InviteScreen() {
  const [status, setStatus] = useState<ReferralStatus | null>(null);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      setStatus(await api.referral.status());
      setError(null);
    } catch (e: any) {
      setError(e.message);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);
  useTopic('credits', load);

  const share = async () => {
    if (!status) return;
    try {
      await Share.share({
        message: tr('Come try SynAgent with me. Suki, an AI winggirl, introduces you to people worth meeting. Start here: {url} (invite code: {code})', {
          url: status.link,
          code: status.code,
        }),
      });
    } catch (e) {
      showError(e);
    }
  };

  const [posting, setPosting] = useState(false);
  const poster = async () => {
    setPosting(true);
    try {
      await sharePoster(tr('Share my invite'));
    } catch (e) {
      showError(e);
    } finally {
      setPosting(false);
    }
  };

  return (
    <Backdrop>
      <ScreenHeader title={tr('Invite friends')} />
      <Page refreshing={false} onRefresh={load}>
        <ErrorText message={error} />
        {!status && !error ? <SukiLoading /> : null}
        {status ? (
          <>
            <Glass style={{ gap: space.sm }}>
              <T v="subheading">{tr('Earn {n} credits for each friend', { n: String(status.credits_per_friend) })}</T>
              <T v="small" soft>
                {tr('You get them when your friend signs up with your link or code and finishes their profile.')}
              </T>
              <T v="small" soft>{tr('Your invite code')}</T>
              <T v="heading" selectable>{status.code}</T>
              <Button label={tr('Share my invite')} icon="share-outline" full onPress={share} />
              <Button label={tr('Share a poster to WeChat Moments')} icon="image-outline" kind="secondary" full loading={posting} onPress={poster} />
              <T v="caption" soft>{tr('The poster has your QR code and invite code. Pick WeChat, then Moments, or save it and post it anywhere.')}</T>
            </Glass>
            <Glass style={{ flexDirection: 'row', gap: space.md }}>
              <View style={{ flex: 1 }}>
                <T v="subheading">{status.joined}</T>
                <T v="small" soft>{tr('Friends who signed up')}</T>
              </View>
              <View style={{ flex: 1 }}>
                <T v="subheading">{status.credits_earned}</T>
                <T v="small" soft>{tr('Credits earned')}</T>
              </View>
            </Glass>
          </>
        ) : null}
      </Page>
    </Backdrop>
  );
}
