import { LinearGradient } from 'expo-linear-gradient';
import { router, useLocalSearchParams } from 'expo-router';
import { useCallback, useEffect, useState } from 'react';
import { ScrollView, View } from 'react-native';
import { useScreenSize } from '@/ui/screenSize';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import * as api from '@/api/endpoints';
import type { Match, Proposal } from '@/api/types';
import { DecisionBar, useDecision } from '@/features/proposalActions';
import { useTopic } from '@/state/live';
import { useTheme } from '@/theme/ThemeProvider';
import { space } from '@/theme/tokens';
import { Backdrop } from '@/ui/layout';
import { PersonDetails, personLine, PhotoPager, SharedChips } from '@/ui/person';
import { TranslatedNote, usePersonInLanguage } from '@/features/PersonTranslation';
import { Button, Glass, IconButton, Pill, T } from '@/ui/primitives';
import { SukiAvatar, SukiLoading, SukiSays } from '@/ui/suki';
import { tr } from '@/i18n';

/** Someone's profile, opened from an introduction (?proposal=) or a match (?match=). */
export default function PersonScreen() {
  const t = useTheme();
  const insets = useSafeAreaInsets();
  const { height } = useScreenSize();
  const params = useLocalSearchParams<{ proposal?: string; match?: string }>();
  const [proposal, setProposal] = useState<Proposal | null>(null);
  const [match, setMatch] = useState<Match | null>(null);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    try {
      if (params.proposal) setProposal(await api.proposals.get(params.proposal));
      else if (params.match) setMatch(await api.matches.get(params.match));
      setError(null);
    } catch (e: any) {
      setError(e.message);
    }
  }, [params.match, params.proposal]);

  useEffect(() => {
    load();
  }, [load]);
  useTopic('proposals', load);

  const { decide, busy } = useDecision((decision, matchId) => {
    if (decision === 'pass' || !matchId) router.back();
    else load();
  });

  const { shown: person, translated, original, toggle } = usePersonInLanguage(proposal?.person ?? match?.person);
  const back = (
    <View style={{ position: 'absolute', top: insets.top + space.sm, left: space.sm }}>
      <View style={{ backgroundColor: t.glassStrong, borderRadius: 21 }}>
        <IconButton icon="chevron-back" label={tr('Back')} onPress={() => router.back()} />
      </View>
    </View>
  );

  if (!person) {
    return (
      <Backdrop>
        <View style={{ flex: 1, justifyContent: 'center' }}>
          {error ? <SukiSays face="pouty" title={tr("I can't open this profile")} body={error} /> : <SukiLoading line={tr('Pulling up their profile...')} />}
        </View>
        {back}
      </Backdrop>
    );
  }

  const open = proposal && proposal.status === 'open' && !proposal.my_decision;

  return (
    <Backdrop>
      <ScrollView contentContainerStyle={{ paddingBottom: (open ? 110 : 40) + insets.bottom }}>
        <View>
          <PhotoPager ids={person.photo_ids ?? []} height={Math.round(height * 0.58)} />
          <LinearGradient
            pointerEvents="none"
            colors={['transparent', 'rgba(25,12,60,0.82)']}
            style={{ position: 'absolute', left: 0, right: 0, bottom: 0, paddingTop: 56, paddingHorizontal: space.xl, paddingBottom: space.lg }}
          >
            <T v="display" color="#FFFFFF">
              {person.display_name}
            </T>
            <T color="rgba(255,255,255,0.9)">{personLine(person)}</T>
          </LinearGradient>
        </View>
        <View style={{ padding: space.lg, gap: space.lg }}>
          {proposal?.pitch ? (
            <Glass style={{ flexDirection: 'row', gap: space.md }}>
              <SukiAvatar mood="playful" size={40} />
              <View style={{ flex: 1, gap: space.sm }}>
                <T v="small">{proposal.pitch}</T>
                <SharedChips shared={proposal.shared} />
              </View>
            </Glass>
          ) : null}
          {proposal?.my_decision === 'accept' && !proposal.match_id ? (
            <Pill tone="good" icon="heart" label={tr('You said yes. Waiting on {name}.', { name: person.display_name })} />
          ) : null}
          {proposal?.match_id ? (
            <Button label={tr('Chat with {name}', { name: person.display_name })} icon="chatbubbles" full onPress={() => router.replace(`/chat/${proposal.match_id}`)} />
          ) : null}
          {translated ? <TranslatedNote original={original} toggle={toggle} /> : null}
          <PersonDetails person={person} />
        </View>
      </ScrollView>
      {back}
      {open && proposal ? (
        <View
          style={{
            position: 'absolute',
            left: 0,
            right: 0,
            bottom: 0,
            padding: space.lg,
            paddingBottom: insets.bottom + space.md,
            backgroundColor: t.sheet,
          }}
        >
          <DecisionBar proposal={proposal} decide={decide} busy={busy} />
        </View>
      ) : null}
    </Backdrop>
  );
}
