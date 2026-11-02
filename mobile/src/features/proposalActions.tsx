import { router } from 'expo-router';
import { useState } from 'react';
import { Alert, View } from 'react-native';
import { proposals as api } from '@/api/endpoints';
import type { Proposal } from '@/api/types';
import { useLive } from '@/state/live';
import { space } from '@/theme/tokens';
import { Button } from '@/ui/primitives';
import { showError } from '@/ui/sheet';

/** Yes and Pass, with what happens afterwards. Shared by every place an introduction shows. */
export function useDecision(onDone?: (decision: 'accept' | 'pass', matchId: string) => void) {
  const { refreshProposals, refreshMatches } = useLive();
  const [busy, setBusy] = useState<'accept' | 'pass' | null>(null);
  const decide = async (p: Proposal, decision: 'accept' | 'pass') => {
    setBusy(decision);
    try {
      const matchId = await api.decide(p.id, decision);
      await Promise.all([refreshProposals(), refreshMatches()]);
      onDone?.(decision, matchId);
      if (decision === 'accept' && matchId) {
        Alert.alert(`You and ${p.person.display_name} both said yes!`, "That's a match. Go say hi.", [
          { text: 'Later', style: 'cancel' },
          { text: 'Open chat', onPress: () => router.push(`/chat/${matchId}`) },
        ]);
      }
    } catch (e) {
      showError(e);
    } finally {
      setBusy(null);
    }
  };
  return { decide, busy };
}

export function DecisionBar({
  proposal,
  decide,
  busy,
}: {
  proposal: Proposal;
  decide: (p: Proposal, d: 'accept' | 'pass') => void;
  busy: 'accept' | 'pass' | null;
}) {
  return (
    <View style={{ flexDirection: 'row', gap: space.md }}>
      <Button
        label="Pass"
        icon="close"
        kind="secondary"
        loading={busy === 'pass'}
        disabled={!!busy}
        onPress={() => decide(proposal, 'pass')}
        style={{ flex: 1 }}
      />
      <Button
        label="Yes"
        icon="heart"
        loading={busy === 'accept'}
        disabled={!!busy}
        onPress={() => decide(proposal, 'accept')}
        style={{ flex: 1.6 }}
      />
    </View>
  );
}
