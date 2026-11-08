import { router } from 'expo-router';
import { useEffect, useState } from 'react';
import { Linking, View } from 'react-native';
import { Alert, Share } from '@/lib/dialogs';
import * as api from '@/api/endpoints';
import type { Card, CreditStatus, Proposal, ReferralStatus } from '@/api/types';
import { sharePoster } from '@/lib/poster';
import { LookingForCard } from './LookingForCard';
import { BuddyOfferCard, BuddyPostCard, BuddyRequestCard, GroupCard, TripCard, TripPostCard } from './BuddyCards';
import { openNamedScreen, openPayment } from '@/lib/links';
import { buyOnPlay, isPlayBuild, playPrices, unfinishedPlayPurchases } from '@/lib/playBilling';
import { pickAndUploadPhoto, takeVerificationSelfie } from '@/lib/photos';
import { useLive, useTopic } from '@/state/live';
import { useSession } from '@/state/session';
import { useTheme } from '@/theme/ThemeProvider';
import { space } from '@/theme/tokens';
import { SharedChips } from '@/ui/person';
import { HeroCard, PhotoBadge } from '@/ui/HeroCard';
import { Button, ErrorText, Glass, Pill, T, type IconName } from '@/ui/primitives';
import { confirm, showError } from '@/ui/sheet';
import { DecisionBar, useDecision } from './proposalActions';
import { tr } from '@/i18n';

function ActionCard({
  label,
  icon,
  onPress,
  kind = 'secondary',
}: {
  label: string;
  icon: IconName;
  onPress: () => void | Promise<unknown>;
  kind?: 'primary' | 'secondary' | 'danger';
}) {
  const [busy, setBusy] = useState(false);
  return (
    <Button
      label={label}
      icon={icon}
      kind={kind}
      small
      loading={busy}
      onPress={async () => {
        setBusy(true);
        try {
          await onPress();
        } catch (e) {
          showError(e);
        } finally {
          setBusy(false);
        }
      }}
    />
  );
}

function PhotoUploadCard() {
  const { refreshPhotos } = useLive();
  const [state, setState] = useState<'idle' | 'busy' | 'done'>('idle');
  const [error, setError] = useState<string | null>(null);
  if (state === 'done') return <Pill tone="good" icon="checkmark-circle" label={tr("Got it. I'll take a look.")} />;
  return (
    <View style={{ gap: space.sm }}>
      <Button
        label={tr('Add a photo')}
        icon="image"
        small
        loading={state === 'busy'}
        onPress={async () => {
          setState('busy');
          setError(null);
          try {
            const r = await pickAndUploadPhoto();
            if (r) {
              refreshPhotos();
              setState('done');
            } else setState('idle');
          } catch (e: any) {
            setError(e.message);
            setState('idle');
          }
        }}
      />
      <ErrorText message={error} />
    </View>
  );
}

function BioDraftCard({ text }: { text: string }) {
  const t = useTheme();
  const { profile, refreshMe } = useSession();
  const [accepted, setAccepted] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const saved = accepted || profile?.bio === text;
  return (
    <Glass strong style={{ gap: space.md }}>
      <T v="label" soft>{tr('A bio for you')}</T>
      <T style={{ fontStyle: 'italic' }} color={t.ink}>
        {text}
      </T>
      {saved ? (
        <Pill tone="good" icon="checkmark-circle" label={tr("It's on your profile")} />
      ) : (
        <Button
          label={tr('Use this')}
          icon="checkmark"
          small
          loading={busy}
          onPress={async () => {
            setBusy(true);
            setError(null);
            try {
              await api.profile.acceptBio();
              setAccepted(true);
              refreshMe().catch(() => {});
            } catch (e: any) {
              setError(e.message);
            } finally {
              setBusy(false);
            }
          }}
        />
      )}
      <ErrorText message={error} />
    </Glass>
  );
}

function ProposalCard({ id }: { id: string }) {
  const t = useTheme();
  const [proposal, setProposal] = useState<Proposal | null>(null);
  const [gone, setGone] = useState(false);

  const load = () =>
    api.proposals
      .get(id)
      .then((p) => {
        setProposal(p);
        setGone(false);
      })
      .catch((e) => {
        if (e?.status === 404) setGone(true);
      });

  useEffect(() => {
    load();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id]);
  useTopic('proposals', load);
  useTopic('matches', load);

  const { sendToSuki } = useLive();
  const { decide, busy } = useDecision(() => load());

  if (gone) {
    return (
      <Glass strong>
        <T v="small" soft>{tr("This introduction isn't available any more.")}</T>
      </Glass>
    );
  }
  if (!proposal) {
    return <Glass strong style={{ height: 120, backgroundColor: t.cardTint }} />;
  }
  const p = proposal.person;
  const open = proposal.status === 'open' && !proposal.my_decision;
  const expired = proposal.status === 'expired' || new Date(proposal.expires_at).getTime() < Date.now();
  return (
    <HeroCard
      photo={p.photo_ids?.[0]}
      title={[p.display_name, p.age ? String(p.age) : ''].filter(Boolean).join(', ')}
      subtitle={[p.occupation, p.city_name].filter(Boolean).join(' · ')}
      badges={
        <>
          <PhotoBadge label={tr('Suki recommends')} />
          {p.verified ? <PhotoBadge label={tr('Verified')} /> : null}
        </>
      }
      onPress={() => router.push(`/person?proposal=${proposal.id}`)}
      pressLabel={tr("See {name}'s profile", { name: p.display_name })}
      height={340}
    >
      {proposal.pitch ? (
        <T numberOfLines={4} style={{ fontStyle: 'italic' }}>
          {proposal.pitch}
        </T>
      ) : null}
      <SharedChips shared={proposal.shared} />
      {open && !expired ? (
        <Button
          label={tr('Ask Suki about {name}', { name: p.display_name })}
          icon="chatbubble-ellipses-outline"
          kind="secondary"
          small
          onPress={() => sendToSuki(tr('Tell me more about {name}. What should I know before I decide?', { name: p.display_name }))}
        />
      ) : null}
      {proposal.match_id ? (
        <Button label={tr('Open chat')} icon="chatbubbles" small onPress={() => router.push(`/chat/${proposal.match_id}`)} />
      ) : proposal.my_decision === 'accept' ? (
        <Pill tone="good" icon="heart" label={tr('You said yes. Waiting on {name}.', { name: p.display_name })} />
      ) : proposal.my_decision === 'pass' ? (
        <Pill tone="plain" label={tr('You passed on this one.')} />
      ) : expired || !open ? (
        <Pill tone="plain" label={tr('This introduction has ended.')} />
      ) : (
        <DecisionBar proposal={proposal} decide={decide} busy={busy} />
      )}
    </HeroCard>
  );
}

function ConfirmCard({ card }: { card: Extract<Card, { kind: 'confirm' }> }) {
  const live = useLive();
  const { refreshMe } = useSession();
  const [done, setDone] = useState(false);
  if (done) return <Pill tone="good" icon="checkmark-circle" label={tr('Done.')} />;

  const copy: Record<typeof card.action, [string, string]> = {
    remove_photo: [tr('Remove this photo?'), tr('It comes off your profile right away.')],
    unmatch: [tr('Unmatch?'), tr('The chat ends for both of you, and any open date plan is cancelled.')],
    block: [tr('Block this person?'), tr("You'll stop seeing each other, and any match ends at once.")],
    reset_taste: [tr('Start over?'), tr('I forget my notes and everything I learned about your taste.')],
    buddy_close: [tr('Take this post down?'), tr("People can't join it any more.")],
    trip_remove: [tr('Remove this trip?'), tr("Plans there stop reaching you. Plans you posted for it stay up.")],
  };
  const [title, body] = copy[card.action] ?? [tr('Are you sure?'), ''];

  const run = async () => {
    switch (card.action) {
      case 'remove_photo':
        if (!card.photo_id) return;
        await api.photos.remove(card.photo_id);
        live.refreshPhotos();
        refreshMe().catch(() => {});
        break;
      case 'unmatch':
        if (!card.match_id) return;
        await api.matches.unmatch(card.match_id);
        live.refreshMatches();
        live.refreshDates();
        break;
      case 'block':
        if (!card.user_id) return;
        await api.safety.block(card.user_id);
        live.refreshMatches();
        live.refreshProposals();
        live.refreshDates();
        break;
      case 'reset_taste':
        await api.memories.reset();
        break;
      case 'buddy_close':
        if (!card.request_id) return;
        await api.buddies.close(card.request_id);
        break;
      case 'trip_remove':
        if (!card.trip_id) return;
        await api.trips.remove(card.trip_id);
        break;
      default:
        return;
    }
    setDone(true);
  };

  return (
    <ActionCard
      label={tr(card.label)}
      icon="alert-circle-outline"
      kind="danger"
      onPress={async () => {
        if (await confirm(title, body, tr(card.label), true)) await run();
      }}
    />
  );
}

/** A live selfie for the verified badge, compared with their main photo and never kept. */
function SelfieCard() {
  const { refreshMe } = useSession();
  const [state, setState] = useState<'idle' | 'busy' | 'done'>('idle');
  const [reason, setReason] = useState<string | null>(null);
  if (state === 'done') return <Pill tone="good" icon="shield-checkmark" label={tr("You're verified.")} />;
  return (
    <Glass strong style={{ gap: space.sm }}>
      <T v="subheading">{tr('Get the verified badge')}</T>
      <T v="small" soft>{tr("One face check: take a selfie now, facing the camera in good light. Our AI provider compares it with your main photo, then it's deleted and only the result is kept. Tapping the button means you agree to this check.")}</T>
      {reason ? <T v="small" color="#ff8a9a">{tr(reason)}</T> : null}
      <Button
        label={tr('Agree and take a selfie')}
        icon="camera-outline"
        small
        loading={state === 'busy'}
        onPress={async () => {
          setState('busy');
          setReason(null);
          try {
            const r = await takeVerificationSelfie();
            if (r?.verified) {
              setState('done');
              refreshMe().catch(() => {});
              return;
            }
            if (r?.reason) setReason(r.reason);
          } catch (e) {
            showError(e);
          }
          setState('idle');
        }}
      />
    </Glass>
  );
}

/** Suki's way to invite friends: the invite to send, and a poster for WeChat Moments. */
function InviteCard() {
  const [status, setStatus] = useState<ReferralStatus | null>(null);
  const [busy, setBusy] = useState<'' | 'link' | 'poster'>('');
  useEffect(() => {
    api.referral.status().then(setStatus, () => {});
  }, []);
  if (!status) return null;
  const run = (which: 'link' | 'poster', fn: () => Promise<unknown>) => async () => {
    setBusy(which);
    try {
      await fn();
    } catch (e) {
      showError(e);
    } finally {
      setBusy('');
    }
  };
  return (
    <Glass strong style={{ gap: space.sm }}>
      <T v="subheading">{tr('Earn {n} credits for each friend', { n: String(status.credits_per_friend) })}</T>
      <T v="small" soft>{tr('You get them when your friend signs up with your link or code and finishes their profile.')}</T>
      <Button
        label={tr('Share my invite')}
        icon="share-outline"
        small
        loading={busy === 'link'}
        onPress={run('link', () =>
          Share.share({
            message: tr('Come try SynAgent with me. Suki, an AI winggirl, introduces you to people worth meeting. Start here: {url} (invite code: {code})', { url: status.link, code: status.code }),
          }),
        )}
      />
      <Button label={tr('Share a poster to WeChat Moments')} icon="image-outline" kind="secondary" small loading={busy === 'poster'} onPress={run('poster', () => sharePoster(tr('Share my invite')))} />
    </Glass>
  );
}

const PLAY_LISTING = 'https://play.google.com/store/apps/details?id=space.herosagent.synagent';

function BuyCreditsCard() {
  const [status, setStatus] = useState<CreditStatus | null>(null);
  const load = () => {
    api.credits.status().then(setStatus, () => {});
  };
  useEffect(load, []);
  useTopic('credits', load);
  return status ? <BuyCredits status={status} /> : null;
}

/** Their balance and the packs. Paying happens on the provider's page. */
export function BuyCredits({ status }: { status: CreditStatus }) {
  if (isPlayBuild) return <BuyOnPlay status={status} />;
  return <BuyWithCheckout status={status} />;
}

function BuyOnPlay({ status }: { status: CreditStatus }) {
  const [prices, setPrices] = useState<Record<string, string>>({});
  const [busy, setBusy] = useState<string | null>(null);
  const [note, setNote] = useState<string | null>(null);
  const ids = status.google_products.map((p) => p.id).join(',');

  useEffect(() => {
    if (!status.google_enabled) return;
    playPrices(ids.split(',')).then(setPrices, () => {});
    // Anything paid for that never reached the server, because the app closed first.
    unfinishedPlayPurchases()
      .then((list) => Promise.all(list.filter((p) => !p.pending).map((p) => api.credits.verifyGoogle(p.productId, p.token))))
      .catch(() => {});
  }, [status.google_enabled, ids]);

  return (
    <Glass style={{ gap: space.sm }}>
      <T v="small" style={{ fontWeight: '700' }}>
        {tr('{n} credits left', { n: String(status.balance) })}
      </T>
      {!status.google_enabled ? <T v="small" soft>{tr("Buying credits isn't open yet.")}</T> : null}
      {status.google_enabled
        ? status.google_products.map((p) => (
            <Button
              key={p.id}
              small
              kind="secondary"
              icon="card-outline"
              label={prices[p.id] ? tr('{n} credits for {price}', { n: String(p.credits), price: prices[p.id] }) : tr('{n} credits', { n: String(p.credits) })}
              loading={busy === p.id}
              onPress={async () => {
                setBusy(p.id);
                setNote(null);
                try {
                  const bought = await buyOnPlay(p.id, status.google_account);
                  if (bought) {
                    const res = await api.credits.verifyGoogle(bought.productId, bought.token);
                    if (bought.pending || res.pending) setNote(tr("Your payment is still pending. The credits arrive once it's done."));
                  }
                } catch (e) {
                  showError(e);
                } finally {
                  setBusy(null);
                }
              }}
            />
          ))
        : null}
      {note ? <T v="small" soft>{note}</T> : null}
    </Glass>
  );
}

/** Where to buy when it can't be done here. Credits belong to the account, so they arrive here too. */
function BuyElsewhere({ status }: { status: CreditStatus }) {
  if (!status.google_enabled && !status.wechat_enabled) {
    return <T v="small" soft>{tr("Buying credits isn't open yet.")}</T>;
  }
  return (
    <>
      <T v="small" soft>{tr("Credits can't be bought here. Buy them with this account and they'll show up here too.")}</T>
      {status.google_enabled ? (
        <Button
          small
          kind="secondary"
          icon="logo-google-playstore"
          label={tr('Buy on Google Play')}
          onPress={() => Linking.openURL(PLAY_LISTING).catch(showError)}
        />
      ) : null}
      {status.wechat_enabled ? <T v="small" soft>{tr('Or buy them in the SynAgent mini program in WeChat.')}</T> : null}
    </>
  );
}

function BuyWithCheckout({ status }: { status: CreditStatus }) {
  const [busy, setBusy] = useState<string | null>(null);
  return (
    <Glass style={{ gap: space.sm }}>
      <T v="small" style={{ fontWeight: '700' }}>
        {tr('{n} credits left', { n: String(status.balance) })}
      </T>
      {!status.payments_enabled ? (
        <BuyElsewhere status={status} />
      ) : (
        status.packages.map((p) => (
          <Button
            key={p.id}
            small
            kind="secondary"
            icon="card-outline"
            label={tr('{n} credits for {price}', { n: String(p.credits), price: `$${(p.amount_cents / 100).toFixed(2)}` })}
            loading={busy === p.id}
            onPress={async () => {
              setBusy(p.id);
              try {
                await openPayment(await api.credits.checkout(p.id));
              } catch (e) {
                showError(e);
              } finally {
                setBusy(null);
              }
            }}
          />
        ))
      )}
    </Glass>
  );
}

export function SukiCard({ card }: { card: Card }) {
  const { matches } = useLive();
  const { user } = useSession();
  switch (card.kind) {
    case 'photo_upload':
      return <PhotoUploadCard />;
    case 'buy_credits':
      return <BuyCreditsCard />;
    case 'buddy_post':
      return <BuddyPostCard card={card} />;
    case 'buddy_request':
      return <BuddyRequestCard id={card.request_id} />;
    case 'buddy_offer':
      return <BuddyOfferCard id={card.request_id} userId={card.user_id} />;
    case 'group':
      return <GroupCard id={card.request_id} />;
    case 'invite':
      return <InviteCard />;
    case 'looking_for':
      return <LookingForCard />;
    case 'selfie':
      return <SelfieCard />;
    case 'trip_post':
      return <TripPostCard card={card} />;
    case 'trip':
      return <TripCard id={card.trip_id} />;
    case 'bio_draft':
      return <BioDraftCard text={card.text} />;
    case 'proposal':
      return <ProposalCard id={card.proposal_id} />;
    case 'match': {
      const name = matches.find((m) => m.id === card.match_id)?.person.display_name;
      return (
        <ActionCard
          kind="primary"
          icon="chatbubbles"
          label={name ? tr('Say hi to {name}', { name: name }) : tr('Open the chat')}
          onPress={() => router.push(`/chat/${card.match_id}`)}
        />
      );
    }
    case 'date':
      return (
        <ActionCard
          kind="primary"
          icon="calendar"
          label={card.focus === 'time' ? tr('Pick a time') : card.focus === 'change' ? tr('Change the plan') : tr('Open the date')}
          onPress={() =>
            router.push(card.focus ? `/date/${card.date_id}?focus=${card.focus}` : `/date/${card.date_id}`)
          }
        />
      );
    case 'share_plan':
      return (
        <ActionCard
          icon="share-outline"
          label={tr('Share with a friend')}
          onPress={async () => {
            const message = await api.dates.share(card.date_id);
            await Share.share({ message });
          }}
        />
      );
    case 'open_screen':
      return (
        <ActionCard
          icon="arrow-forward"
          label={card.label}
          onPress={() => {
            if (card.screen === 'verify_email') {
              if (user?.email_verified) Alert.alert(tr('All set'), tr('Your email is already verified.'));
              return;
            }
            openNamedScreen(card.screen);
          }}
        />
      );
    case 'confirm':
      return <ConfirmCard card={card} />;
    default:
      return null;
  }
}
