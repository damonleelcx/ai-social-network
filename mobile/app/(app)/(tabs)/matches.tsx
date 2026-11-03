import { LinearGradient } from 'expo-linear-gradient';
import { router } from 'expo-router';
import { useState } from 'react';
import { FlatList, Pressable, RefreshControl, View } from 'react-native';
import type { Match, Proposal } from '@/api/types';
import { DecisionBar, useDecision } from '@/features/proposalActions';
import { shortTime, STAGE_LABEL } from '@/lib/format';
import { useLive } from '@/state/live';
import { useTheme } from '@/theme/ThemeProvider';
import { radius, space } from '@/theme/tokens';
import { Backdrop, TabHeader } from '@/ui/layout';
import { MediaImage, PersonAvatar, personLine, SharedChips } from '@/ui/person';
import { Badge, Glass, T } from '@/ui/primitives';
import { FadeIn, SukiAvatar, SukiLoading, SukiSays } from '@/ui/suki';
import { tr } from '@/i18n';

function Introduction({ proposal }: { proposal: Proposal }) {
  const t = useTheme();
  const { decide, busy } = useDecision();
  const p = proposal.person;
  return (
    <FadeIn>
      <Glass padded={false} style={{ overflow: 'hidden', borderRadius: radius.xl }}>
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={tr("See {name}'s profile", { name: p.display_name })}
          onPress={() => router.push(`/person?proposal=${proposal.id}`)}
        >
          <MediaImage id={p.photo_ids?.[0]} style={{ aspectRatio: 4 / 5 }} />
          <LinearGradient
            colors={['transparent', 'rgba(25,12,60,0.86)']}
            style={{ position: 'absolute', left: 0, right: 0, bottom: 0, paddingTop: 64, paddingHorizontal: space.xl, paddingBottom: space.lg }}
          >
            <T v="title" color="#FFFFFF">
              {p.display_name}
            </T>
            <T color="rgba(255,255,255,0.88)">{personLine(p)}</T>
          </LinearGradient>
        </Pressable>
        <View style={{ padding: space.lg, gap: space.md }}>
          <View style={{ flexDirection: 'row', gap: space.sm }}>
            <SukiAvatar mood="playful" size={36} />
            <View style={{ flex: 1, backgroundColor: t.cardTint, borderRadius: radius.md, borderTopLeftRadius: 6, padding: space.md }}>
              <T v="small">{proposal.pitch}</T>
            </View>
          </View>
          <SharedChips shared={proposal.shared} />
          <DecisionBar proposal={proposal} decide={decide} busy={busy} />
        </View>
      </Glass>
    </FadeIn>
  );
}

function MatchRow({ match }: { match: Match }) {
  const p = match.person;
  const stage = tr(STAGE_LABEL[match.stage] ?? '');
  return (
    <Pressable
      accessibilityRole="button"
      accessibilityLabel={tr('Chat with {name}{value}', { name: p.display_name, value: match.unread ? tr(', new messages') : '' })}
      onPress={() => router.push(`/chat/${match.id}`)}
      style={({ pressed }) => ({ opacity: pressed ? 0.7 : 1 })}
    >
      <Glass style={{ flexDirection: 'row', alignItems: 'center', gap: space.md, padding: space.md }}>
        <PersonAvatar id={p.photo_ids?.[0]} size={60} />
        <View style={{ flex: 1, gap: 2 }}>
          <View style={{ flexDirection: 'row', alignItems: 'center', gap: space.sm }}>
            <T v="subheading" numberOfLines={1} style={{ flex: 1 }}>
              {p.display_name}
            </T>
            <T v="caption" faint>
              {shortTime(match.last_message_at ?? match.created_at)}
            </T>
          </View>
          <T v="small" soft numberOfLines={1} style={match.unread ? { fontWeight: '700' } : undefined}>
            {match.last_preview || tr('Say hi. I know you want to.')}
          </T>
          {stage ? (
            <T v="caption" faint numberOfLines={1}>
              {stage}
            </T>
          ) : null}
        </View>
        <Badge count={match.unread} />
      </Glass>
    </Pressable>
  );
}

type Row =
  | { type: 'heading'; key: string; title: string; hint?: string }
  | { type: 'proposal'; key: string; proposal: Proposal }
  | { type: 'match'; key: string; match: Match };

export default function Matches() {
  const live = useLive();
  const [refreshing, setRefreshing] = useState(false);
  const t = useTheme();

  const rows: Row[] = [];
  if (live.proposals.length) {
    rows.push({ type: 'heading', key: 'h-intro', title: 'Waiting for you', hint: "People I think you should meet. They only find out if you both say yes." });
    for (const p of live.proposals) rows.push({ type: 'proposal', key: p.id, proposal: p });
  }
  if (live.matches.length) {
    rows.push({ type: 'heading', key: 'h-match', title: 'Your matches' });
    for (const m of live.matches) rows.push({ type: 'match', key: m.id, match: m });
  }

  const onRefresh = async () => {
    setRefreshing(true);
    await Promise.all([live.refreshProposals(), live.refreshMatches()]);
    setRefreshing(false);
  };

  const loading = !live.loaded.proposals || !live.loaded.matches;

  return (
    <Backdrop>
      <TabHeader title={tr('Matches')} subtitle={tr('Good introductions, not endless swiping.')} />
      <FlatList
        data={rows}
        keyExtractor={(r) => r.key}
        contentContainerStyle={{ paddingHorizontal: space.lg, paddingBottom: space.xxl, gap: space.md, flexGrow: 1 }}
        refreshControl={<RefreshControl refreshing={refreshing} onRefresh={onRefresh} tintColor={t.accent} colors={[t.accent]} />}
        ListEmptyComponent={
          loading ? (
            <SukiLoading line={tr('Checking on your introductions...')} />
          ) : (
            <SukiSays
              face="thinking"
              size={132}
              title={tr('Nobody here yet')}
              body={tr("I'm looking. When I find someone worth your time, I'll bring them right here. The more you tell me, the better I get.")}
            />
          )
        }
        renderItem={({ item }) => {
          if (item.type === 'heading') {
            return (
              <View style={{ marginTop: space.sm, gap: 2, paddingHorizontal: 4 }}>
                <T v="label" soft>
                  {item.title}
                </T>
                {item.hint ? (
                  <T v="caption" faint>
                    {item.hint}
                  </T>
                ) : null}
              </View>
            );
          }
          if (item.type === 'proposal') return <Introduction proposal={item.proposal} />;
          return <MatchRow match={item.match} />;
        }}
      />
    </Backdrop>
  );
}
