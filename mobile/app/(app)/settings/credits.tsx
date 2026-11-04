import { useCallback, useEffect, useState } from 'react';
import { View } from 'react-native';
import * as api from '@/api/endpoints';
import type { CreditStatus, CreditUsage } from '@/api/types';
import { BuyCredits } from '@/features/SukiCards';
import { shortTime } from '@/lib/format';
import { useTopic } from '@/state/live';
import { space } from '@/theme/tokens';
import { Backdrop, Page, ScreenHeader } from '@/ui/layout';
import { ErrorText, Glass, T } from '@/ui/primitives';
import { SukiLoading } from '@/ui/suki';
import { tr } from '@/i18n';

const REASON: Record<string, string> = {
  purchase: 'Bought credits',
  turn: 'Message to Suki',
  refund: "Refund: Suki couldn't answer",
  grant: 'Added by SynAgent',
  referral: 'A friend joined',
};

export default function CreditsScreen() {
  const [status, setStatus] = useState<CreditStatus | null>(null);
  const [usage, setUsage] = useState<CreditUsage[]>([]);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      const [s, u] = await Promise.all([api.credits.status(), api.credits.usage()]);
      setStatus(s);
      setUsage(u);
      setError(null);
    } catch (e: any) {
      setError(e.message);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);
  useTopic('credits', load);

  return (
    <Backdrop>
      <ScreenHeader title={tr('Credits')} />
      <Page refreshing={false} onRefresh={load}>
        <ErrorText message={error} />
        {!status && !error ? <SukiLoading /> : null}
        {status ? (
          <>
            <Glass>
              <T v="small" soft>
                {status.free
                  ? tr('Messages to Suki are free right now. You have used {used} of your {free} free matches.', {
                      used: String(status.matches_used),
                      free: String(status.free_matches),
                    })
                  : tr('Each message to Suki uses {n} credit. Chatting with your matches is free, and so is anything about your safety or your account.', {
                      n: String(status.cost_per_message),
                    })}
              </T>
            </Glass>
            <BuyCredits status={status} />
          </>
        ) : null}
        {usage.map((u, i) => (
          <Glass key={i} style={{ flexDirection: 'row', alignItems: 'center', gap: space.md }}>
            <View style={{ flex: 1 }}>
              <T v="small">{tr(REASON[u.reason] ?? u.reason)}</T>
              <T v="small" soft>{shortTime(u.created_at)}</T>
            </View>
            <T v="subheading">{u.delta > 0 ? `+${u.delta}` : u.delta}</T>
          </Glass>
        ))}
      </Page>
    </Backdrop>
  );
}
