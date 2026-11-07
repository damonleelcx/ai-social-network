package dating

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"

	"github.com/damonleelcx/ai-social-network/internal/chatstore"
	"github.com/damonleelcx/ai-social-network/internal/i18n"
	"github.com/damonleelcx/ai-social-network/internal/moderation"
	"github.com/damonleelcx/ai-social-network/internal/notify"
	"github.com/damonleelcx/ai-social-network/internal/profile"
	"github.com/damonleelcx/ai-social-network/internal/safety"
)

// Group plans: a buddy post with more than one spot. Saying "I'm in" joins at once while
// there's room, and everyone in it, the poster too, shares one group chat. Like everything
// else here it starts in the conversation with Suki: she brings the post as a card, the tap
// joins, and her card opens the group chat.
const (
	// A group chat stays open for a while after the plan, to share photos and say thanks.
	groupChatGrace = 48 * time.Hour
	groupPage      = 200
	// At most one push about a group's messages per person in this window.
	groupPushEvery = 30 * time.Minute
)

// buddyIn is whether $1 has said they're in on r: an offer, or a place in the group.
const buddyIn = `(EXISTS(SELECT 1 FROM buddy_offers o WHERE o.request_id=r.id AND o.user_id=$1)
	OR EXISTS(SELECT 1 FROM buddy_members bm WHERE bm.request_id=r.id AND bm.user_id=$1))`

// GroupThread is the group chat's thread in the chat store.
func GroupThread(requestID string) string { return "group:" + requestID }

// groupPeople is everyone in a group, the poster first, whatever their state.
func (s *Service) groupPeople(ctx context.Context, requestID string) ([]string, error) {
	rows, err := s.Pool.Query(ctx, `SELECT user_id FROM (
		SELECT r.user_id, r.created_at AS at FROM buddy_requests r WHERE r.id=$1
		UNION ALL SELECT bm.user_id, bm.joined_at FROM buddy_members bm WHERE bm.request_id=$1) x ORDER BY at`, requestID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, pgx.RowTo[string])
}

// groupMembers is who the viewer sees in the group: not anyone blocked either way, held,
// or gone.
func (s *Service) groupMembers(ctx context.Context, requestID, viewerID string) ([]profile.Public, error) {
	rows, err := s.Pool.Query(ctx, `SELECT x.user_id FROM (
		SELECT r.user_id, r.created_at AS at FROM buddy_requests r WHERE r.id=$1
		UNION ALL SELECT bm.user_id, bm.joined_at FROM buddy_members bm WHERE bm.request_id=$1) x
		JOIN profiles p ON p.user_id=x.user_id JOIN users u ON u.id=x.user_id
		WHERE p.status<>'held' AND u.deleted_at IS NULL
		  AND NOT EXISTS (SELECT 1 FROM blocks b WHERE (b.blocker_id=$2 AND b.blocked_id=x.user_id) OR (b.blocker_id=x.user_id AND b.blocked_id=$2))
		ORDER BY x.at`, requestID, viewerID)
	if err != nil {
		return nil, err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}
	out := []profile.Public{}
	for _, id := range ids {
		pub, err := s.publicOf(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, *pub)
	}
	return out, nil
}

// memberView is a group as someone who joined it sees it, full or not.
func (s *Service) memberView(ctx context.Context, requestID, viewerID string) (*BuddyRequest, error) {
	b, owner, err := scanBuddy(s.Pool.QueryRow(ctx, `SELECT `+buddyCols+` FROM buddy_requests r
		WHERE r.id=$1 AND r.spots>1 AND EXISTS (SELECT 1 FROM buddy_members bm WHERE bm.request_id=r.id AND bm.user_id=$2)`, requestID, viewerID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if blocked, err := s.blockedEitherWay(ctx, owner, viewerID); err != nil {
		return nil, err
	} else if blocked {
		return nil, ErrNotFound
	}
	b.Joined, b.Offered = true, true
	if b.Person, err = s.publicOf(ctx, owner); err != nil {
		return nil, err
	}
	b.Members, err = s.groupMembers(ctx, requestID, viewerID)
	return b, err
}

// joinGroup puts someone in a group while there's room. The caller has already checked they
// can see the post. Joining twice is joining once.
func (s *Service) joinGroup(ctx context.Context, requestID, userID string) error {
	var owner, kind string
	var going int
	var joinedAt time.Time
	err := pgx.BeginFunc(ctx, s.Pool, func(tx pgx.Tx) error {
		var spots int
		var status string
		var live bool
		err := tx.QueryRow(ctx, `SELECT user_id, kind, spots, status, expires_at>now() FROM buddy_requests WHERE id=$1 FOR UPDATE`, requestID).
			Scan(&owner, &kind, &spots, &status, &live)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		var already bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM buddy_members WHERE request_id=$1 AND user_id=$2)`, requestID, userID).Scan(&already); err != nil {
			return err
		}
		if already {
			return nil
		}
		if status != "open" || !live {
			return ErrNotFound
		}
		var joined int
		if err := tx.QueryRow(ctx, `SELECT count(*) FROM buddy_members WHERE request_id=$1`, requestID).Scan(&joined); err != nil {
			return err
		}
		if joined >= spots {
			return userErr("This group is full.")
		}
		if err := tx.QueryRow(ctx, `INSERT INTO buddy_members (request_id, user_id) VALUES ($1,$2) RETURNING joined_at`, requestID, userID).Scan(&joinedAt); err != nil {
			return err
		}
		if joined+1 >= spots {
			if _, err := tx.Exec(ctx, `UPDATE buddy_requests SET status='filled' WHERE id=$1`, requestID); err != nil {
				return err
			}
		}
		going = joined + 2 // everyone who had joined, this person, and the poster
		return nil
	})
	if err != nil || joinedAt.IsZero() {
		return err
	}
	name, err := s.displayName(ctx, userID)
	if err != nil {
		return err
	}
	key := fmt.Sprintf("%s:%d", userID, joinedAt.UnixNano())
	if _, _, err := s.Chats.AddChat(ctx, GroupThread(requestID), "", "system", name+" joined.", map[string]any{"event": "joined", "name": name}, "joined:"+key); err != nil {
		slog.Error("group join line", "err", err)
	}
	card := []map[string]any{{"kind": "group", "request_id": requestID}}
	lang := s.Lang(ctx, userID)
	body := i18n.T(lang, "You're in! The group chat for {plan} is open: say hi and sort out the details together. For a first meet-up, pick somewhere public and tell a friend.",
		"plan", i18n.T(lang, buddyKindNames[kind]))
	if err := s.PostFromSuki(ctx, userID, body, "happy", card, "group-in:"+requestID+":"+key); err != nil {
		slog.Error("group join announce", "err", err)
	}
	lang = s.Lang(ctx, owner)
	body = i18n.T(lang, "{name} joined your group for {plan}. {n} going so far.", "name", name, "plan", i18n.T(lang, buddyKindNames[kind]), "n", fmt.Sprint(going))
	if err := s.PostFromSuki(ctx, owner, body, "happy", card, "group-joined:"+requestID+":"+key); err != nil {
		slog.Error("group join announce", "err", err)
	}
	if err := notify.Send(ctx, s.Pool, owner, "buddy_offer", "Someone joined your group", body,
		map[string]any{"screen": "group", "id": requestID}, "group-joined:"+requestID+":"+key); err != nil {
		slog.Error("group join notify", "err", err)
	}
	s.groupChanged(ctx, requestID)
	return nil
}

// leaveGroup takes someone out of a group. It says false when they weren't in one.
func (s *Service) leaveGroup(ctx context.Context, requestID, userID string) (bool, error) {
	var joinedAt time.Time
	err := s.Pool.QueryRow(ctx, `DELETE FROM buddy_members WHERE request_id=$1 AND user_id=$2 RETURNING joined_at`, requestID, userID).Scan(&joinedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	// A spot came free.
	if _, err := s.Pool.Exec(ctx, `UPDATE buddy_requests SET status='open' WHERE id=$1 AND status='filled' AND spots>1 AND expires_at>now()`, requestID); err != nil {
		return true, err
	}
	if name, err := s.displayName(ctx, userID); err == nil {
		key := fmt.Sprintf("left:%s:%d", userID, joinedAt.UnixNano())
		if _, _, err := s.Chats.AddChat(ctx, GroupThread(requestID), "", "system", name+" left.", map[string]any{"event": "left", "name": name}, key); err != nil {
			slog.Error("group leave line", "err", err)
		}
	}
	s.notifyUser(ctx, userID, "group:"+requestID)
	s.groupChanged(ctx, requestID)
	return true, nil
}

// groupCalledOff tells everyone who joined that the poster took the plan down.
func (s *Service) groupCalledOff(ctx context.Context, requestID, ownerID, kind string) {
	ctx = context.WithoutCancel(ctx)
	name, err := s.displayName(ctx, ownerID)
	if err != nil {
		slog.Error("group off", "err", err)
		return
	}
	if _, _, err := s.Chats.AddChat(ctx, GroupThread(requestID), "", "system", name+" called off the plan.", map[string]any{"event": "called_off", "name": name}, "called-off"); err != nil {
		slog.Error("group off line", "err", err)
	}
	people, err := s.groupPeople(ctx, requestID)
	if err != nil {
		slog.Error("group off", "err", err)
		return
	}
	for _, id := range people {
		if id == ownerID {
			continue
		}
		lang := s.Lang(ctx, id)
		body := i18n.T(lang, "{name} called off the group plan for {plan}. Want me to look for something else?", "name", name, "plan", i18n.T(lang, buddyKindNames[kind]))
		if err := s.PostFromSuki(ctx, id, body, "calm", nil, "group-off:"+requestID+":"+id); err != nil {
			slog.Error("group off announce", "err", err)
		}
	}
	s.groupChanged(ctx, requestID)
}

// groupChanged tells everyone's screens to refresh the group and their plans.
func (s *Service) groupChanged(ctx context.Context, requestID string) {
	people, err := s.groupPeople(context.WithoutCancel(ctx), requestID)
	if err != nil {
		return
	}
	for _, id := range people {
		s.notifyUser(ctx, id, "group:"+requestID)
		s.notifyUser(ctx, id, "buddies")
	}
}

// groupAccess says whether this person is in the group, and whether its chat is still open:
// not called off, and not long past.
func (s *Service) groupAccess(ctx context.Context, requestID, userID string) (bool, error) {
	var open bool
	err := s.Pool.QueryRow(ctx, `SELECT r.status<>'closed' AND r.expires_at > now() - make_interval(secs => $3)
		FROM buddy_requests r WHERE r.id=$1 AND r.spots>1
		  AND (r.user_id=$2 OR EXISTS (SELECT 1 FROM buddy_members bm WHERE bm.request_id=r.id AND bm.user_id=$2))`,
		requestID, userID, groupChatGrace.Seconds()).Scan(&open)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, ErrNotFound
	}
	return open, err
}

// GroupChat is a group's chat as one person in it sees it.
type GroupChat struct {
	Request  *BuddyRequest       `json:"request"`
	Messages []chatstore.Message `json:"messages"`
	Open     bool                `json:"open"`
}

// GroupMessages is the chat after seq (the latest page when after is 0), without anything
// from people this person has blocked or who blocked them.
func (s *Service) GroupMessages(ctx context.Context, requestID, userID string, after int64) (*GroupChat, error) {
	open, err := s.groupAccess(ctx, requestID, userID)
	if err != nil {
		return nil, err
	}
	view, err := s.BuddyView(ctx, requestID, userID)
	if err != nil {
		return nil, err
	}
	var msgs []chatstore.Message
	if after > 0 {
		msgs, err = s.Chats.ChatAfter(ctx, GroupThread(requestID), after, groupPage)
	} else {
		msgs, err = s.Chats.ChatRecent(ctx, GroupThread(requestID), groupPage)
	}
	if err != nil {
		return nil, err
	}
	rows, err := s.Pool.Query(ctx, `SELECT blocked_id FROM blocks WHERE blocker_id=$1 UNION SELECT blocker_id FROM blocks WHERE blocked_id=$1`, userID)
	if err != nil {
		return nil, err
	}
	hidden, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}
	skip := map[string]bool{}
	for _, id := range hidden {
		skip[id] = true
	}
	out := []chatstore.Message{}
	for _, m := range msgs {
		if m.SenderID == "" || !skip[m.SenderID] {
			out = append(out, m)
		}
	}
	return &GroupChat{Request: view, Messages: out, Open: open}, nil
}

// SendGroupMessage posts to a group chat. Many strangers read it, so anything the filter
// blocks never goes in, and a threat or money request goes to a human as well.
func (s *Service) SendGroupMessage(ctx context.Context, requestID, userID, body, clientID string) (*chatstore.Message, error) {
	open, err := s.groupAccess(ctx, requestID, userID)
	if err != nil {
		return nil, err
	}
	if !open {
		return nil, userErr("This group chat has closed.")
	}
	if err := s.requireNotHeld(ctx, userID); err != nil {
		return nil, err
	}
	body = strings.TrimSpace(body)
	if body == "" {
		return nil, userErr("write something first")
	}
	if utf8.RuneCountInString(body) > maxMessage {
		return nil, userErr("that message is too long")
	}
	if v := s.Mod.Check(ctx, userID, moderation.Chat, body); v.Blocked {
		s.Strike(ctx, userID, "", "group", v, body)
		return nil, userErr("%s", moderation.BlockedMessage)
	}
	name, err := s.displayName(ctx, userID)
	if err != nil {
		return nil, err
	}
	var key string
	if clientID != "" {
		key = userID + ":" + clientID
	}
	thread := GroupThread(requestID)
	m, _, err := s.Chats.AddChat(ctx, thread, userID, "text", body, map[string]any{"name": name}, key)
	if err != nil {
		return nil, err
	}
	if f := safety.ScanChat(body); f != nil && f.Severity != "normal" {
		if _, err := s.FileReport(ctx, Report{SubjectID: userID, Source: "sentinel", Reason: f.Reason, Details: truncateRunes(body, 500),
			Severity: f.Severity, Key: fmt.Sprintf("scan:%s:%d", thread, m.Seq)}); err != nil {
			slog.Error("group scan report", "err", err)
		}
	}
	people, err := s.groupPeople(ctx, requestID)
	if err != nil {
		return m, nil
	}
	bucket := time.Now().Truncate(groupPushEvery).Unix()
	for _, id := range people {
		s.notifyUser(ctx, id, "group:"+requestID)
		if id == userID {
			continue
		}
		if blocked, err := s.blockedEitherWay(ctx, id, userID); err != nil || blocked {
			continue
		}
		_ = notify.Send(ctx, s.Pool, id, "message", "New message in your group", "There are new messages in your group chat.",
			map[string]any{"screen": "group", "id": requestID}, fmt.Sprintf("group-msg:%s:%s:%d", requestID, id, bucket))
	}
	return m, nil
}

// MyGroups is the groups this person is in, theirs or someone else's, while the chat is open.
func (s *Service) MyGroups(ctx context.Context, userID string) ([]*BuddyRequest, error) {
	rows, err := s.Pool.Query(ctx, `SELECT r.id FROM buddy_requests r
		WHERE r.spots>1 AND r.status<>'closed' AND r.expires_at > now() - make_interval(secs => $2)
		  AND (r.user_id=$1 OR EXISTS (SELECT 1 FROM buddy_members bm WHERE bm.request_id=r.id AND bm.user_id=$1))
		ORDER BY r.expires_at LIMIT 20`, userID, groupChatGrace.Seconds())
	if err != nil {
		return nil, err
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, err
	}
	out := []*BuddyRequest{}
	for _, id := range ids {
		b, err := s.BuddyView(ctx, id, userID)
		if errors.Is(err, ErrNotFound) {
			continue
		}
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}
