// Package chatstore keeps every chat message in MongoDB: people talking to each
// other, and people talking to Suki. Postgres keeps only what the workflow needs
// (a match's stage, its last-message time).
//
// There is no transaction spanning the two databases, so every write is keyed:
// a retry after a crash lands on the same document. seq comes from a per-thread
// counter, so messages have a stable order and a client can ask for "after N".
package chatstore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

const (
	collAgent    = "agent_messages"
	collChat     = "chat_messages"
	collCounters = "counters"
)

type Message struct {
	Thread    string         `bson:"thread" json:"thread_id"`
	Seq       int64          `bson:"seq" json:"seq"`
	SenderID  string         `bson:"sender_id,omitempty" json:"sender_id,omitempty"`
	Role      string         `bson:"role" json:"role"` // user | assistant | event (Suki); text | prompt | date | system (people)
	Body      string         `bson:"body" json:"body"`
	Meta      map[string]any `bson:"meta,omitempty" json:"meta,omitempty"`
	Key       string         `bson:"key,omitempty" json:"-"`
	ReadAt    *time.Time     `bson:"read_at,omitempty" json:"read_at,omitempty"`
	CreatedAt time.Time      `bson:"created_at" json:"created_at"`
}

type Store struct {
	client *mongo.Client
	db     *mongo.Database
}

// Open connects, waits for the server (bounded) and ensures the indexes. The
// unique partial index on (thread, key) is what makes writes idempotent.
func Open(ctx context.Context, url, dbName string) (*Store, error) {
	client, err := mongo.Connect(options.Client().ApplyURI(url).SetServerSelectionTimeout(5 * time.Second))
	if err != nil {
		return nil, err
	}
	var pingErr error
	for i := 0; i < 15; i++ {
		if pingErr = client.Ping(ctx, nil); pingErr == nil {
			break
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	if pingErr != nil {
		return nil, fmt.Errorf("mongodb unreachable: %w", pingErr)
	}
	s := &Store{client: client, db: client.Database(dbName)}
	for _, name := range []string{collAgent, collChat} {
		_, err := s.db.Collection(name).Indexes().CreateMany(ctx, []mongo.IndexModel{
			{Keys: bson.D{{Key: "thread", Value: 1}, {Key: "seq", Value: 1}}, Options: options.Index().SetUnique(true)},
			{Keys: bson.D{{Key: "thread", Value: 1}, {Key: "key", Value: 1}},
				Options: options.Index().SetUnique(true).SetPartialFilterExpression(bson.D{{Key: "key", Value: bson.D{{Key: "$type", Value: "string"}}}})},
		})
		if err != nil {
			return nil, fmt.Errorf("mongodb indexes: %w", err)
		}
	}
	return s, nil
}

func (s *Store) Close(ctx context.Context) error { return s.client.Disconnect(ctx) }

func (s *Store) Ping(ctx context.Context) error { return s.client.Ping(ctx, nil) }

// Drop removes the whole database. Tests only.
func (s *Store) Drop(ctx context.Context) error { return s.db.Drop(ctx) }

func (s *Store) nextSeq(ctx context.Context, coll, thread string) (int64, error) {
	var doc struct {
		Seq int64 `bson:"seq"`
	}
	err := s.db.Collection(collCounters).FindOneAndUpdate(ctx,
		bson.D{{Key: "_id", Value: coll + ":" + thread}},
		bson.D{{Key: "$inc", Value: bson.D{{Key: "seq", Value: int64(1)}}}},
		options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)).Decode(&doc)
	return doc.Seq, err
}

// add inserts a message once per key. created is false when the key already
// existed: a retried request or a replayed task, returning the first message.
func (s *Store) add(ctx context.Context, coll string, m *Message) (*Message, bool, error) {
	c := s.db.Collection(coll)
	if m.Key != "" {
		var prev Message
		err := c.FindOne(ctx, bson.D{{Key: "thread", Value: m.Thread}, {Key: "key", Value: m.Key}}).Decode(&prev)
		if err == nil {
			return &prev, false, nil
		}
		if !errors.Is(err, mongo.ErrNoDocuments) {
			return nil, false, err
		}
	}
	seq, err := s.nextSeq(ctx, coll, m.Thread)
	if err != nil {
		return nil, false, err
	}
	m.Seq = seq
	m.CreatedAt = time.Now().UTC()
	if _, err := c.InsertOne(ctx, m); err != nil {
		if mongo.IsDuplicateKeyError(err) && m.Key != "" {
			var prev Message
			if e := c.FindOne(ctx, bson.D{{Key: "thread", Value: m.Thread}, {Key: "key", Value: m.Key}}).Decode(&prev); e == nil {
				return &prev, false, nil
			}
		}
		return nil, false, err
	}
	return m, true, nil
}

func (s *Store) after(ctx context.Context, coll, thread string, after int64, limit int) ([]Message, error) {
	cur, err := s.db.Collection(coll).Find(ctx,
		bson.D{{Key: "thread", Value: thread}, {Key: "seq", Value: bson.D{{Key: "$gt", Value: after}}}},
		options.Find().SetSort(bson.D{{Key: "seq", Value: 1}}).SetLimit(int64(limit)))
	if err != nil {
		return nil, err
	}
	out := []Message{}
	return out, decodeInto(ctx, cur, &out)
}

// recent returns the last n messages, oldest first.
func (s *Store) recent(ctx context.Context, coll, thread string, n int) ([]Message, error) {
	cur, err := s.db.Collection(coll).Find(ctx, bson.D{{Key: "thread", Value: thread}},
		options.Find().SetSort(bson.D{{Key: "seq", Value: -1}}).SetLimit(int64(n)))
	if err != nil {
		return nil, err
	}
	out := []Message{}
	if err := decodeInto(ctx, cur, &out); err != nil {
		return nil, err
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out, nil
}

func decodeInto(ctx context.Context, cur *mongo.Cursor, out *[]Message) error {
	defer cur.Close(ctx)
	for cur.Next(ctx) {
		var m Message
		if err := cur.Decode(&m); err != nil {
			return err
		}
		*out = append(*out, m)
	}
	return cur.Err()
}

// ── Suki's conversations ────────────────────────────────────────────────────

func (s *Store) AddAgent(ctx context.Context, convID, role, body string, meta map[string]any, key string) (*Message, bool, error) {
	return s.add(ctx, collAgent, &Message{Thread: convID, Role: role, Body: body, Meta: meta, Key: key})
}

func (s *Store) AgentAfter(ctx context.Context, convID string, after int64, limit int) ([]Message, error) {
	return s.after(ctx, collAgent, convID, after, limit)
}

func (s *Store) AgentRecent(ctx context.Context, convID string, n int) ([]Message, error) {
	return s.recent(ctx, collAgent, convID, n)
}

// RedactAgent replaces a message's text, so filtered content isn't kept or replayed
// into later prompts.
func (s *Store) RedactAgent(ctx context.Context, convID string, seq int64, body string) error {
	_, err := s.db.Collection(collAgent).UpdateOne(ctx, bson.D{{Key: "thread", Value: convID}, {Key: "seq", Value: seq}},
		bson.D{{Key: "$set", Value: bson.D{{Key: "body", Value: body}}}})
	return err
}

// Post satisfies engine.Conversations. A failed post is logged by the caller's
// timeline, never fatal to a task: the work it announces is already saved.
func (s *Store) Post(ctx context.Context, convID, _ string, content string, meta map[string]any, key string) {
	_, _, _ = s.AddAgent(ctx, convID, "assistant", content, meta, key)
}

// Recent satisfies engine.Conversations.
func (s *Store) Recent(ctx context.Context, convID string, n int) string {
	ms, err := s.AgentRecent(ctx, convID, n)
	if err != nil {
		return ""
	}
	var b strings.Builder
	for _, m := range ms {
		who := "Person"
		if m.Role == "assistant" {
			who = "Suki"
		}
		fmt.Fprintf(&b, "%s: %s\n", who, m.Body)
	}
	return b.String()
}

// ── Conversations between people ────────────────────────────────────────────

// AddChat stores a message in a match's thread. role is text, prompt, date or
// system; senderID is empty for anything Suki posts.
func (s *Store) AddChat(ctx context.Context, matchID, senderID, role, body string, meta map[string]any, key string) (*Message, bool, error) {
	return s.add(ctx, collChat, &Message{Thread: matchID, SenderID: senderID, Role: role, Body: body, Meta: meta, Key: key})
}

func (s *Store) ChatAfter(ctx context.Context, matchID string, after int64, limit int) ([]Message, error) {
	return s.after(ctx, collChat, matchID, after, limit)
}

func (s *Store) ChatRecent(ctx context.Context, matchID string, n int) ([]Message, error) {
	return s.recent(ctx, collChat, matchID, n)
}

// MarkRead marks the other person's messages up to seq as read.
func (s *Store) MarkRead(ctx context.Context, matchID, readerID string, upto int64) (int64, error) {
	now := time.Now().UTC()
	res, err := s.db.Collection(collChat).UpdateMany(ctx, bson.D{
		{Key: "thread", Value: matchID},
		{Key: "seq", Value: bson.D{{Key: "$lte", Value: upto}}},
		{Key: "sender_id", Value: bson.D{{Key: "$exists", Value: true}, {Key: "$ne", Value: readerID}}},
		{Key: "read_at", Value: bson.D{{Key: "$exists", Value: false}}},
	}, bson.D{{Key: "$set", Value: bson.D{{Key: "read_at", Value: now}}}})
	if err != nil {
		return 0, err
	}
	return res.ModifiedCount, nil
}

// Unread counts the other person's unread messages in one thread.
func (s *Store) Unread(ctx context.Context, matchID, readerID string) (int64, error) {
	return s.db.Collection(collChat).CountDocuments(ctx, bson.D{
		{Key: "thread", Value: matchID},
		{Key: "sender_id", Value: bson.D{{Key: "$exists", Value: true}, {Key: "$ne", Value: readerID}}},
		{Key: "read_at", Value: bson.D{{Key: "$exists", Value: false}}},
	})
}

// ── Account deletion and export ─────────────────────────────────────────────

// DeleteThreads removes the messages and counters of these conversations and
// matches. Account deletion calls it before the Postgres rows go, so a failure
// stops the deletion and it can be retried.
func (s *Store) DeleteThreads(ctx context.Context, convIDs, matchIDs []string) error {
	del := func(coll string, ids []string) error {
		if len(ids) == 0 {
			return nil
		}
		if _, err := s.db.Collection(coll).DeleteMany(ctx, bson.D{{Key: "thread", Value: bson.D{{Key: "$in", Value: ids}}}}); err != nil {
			return err
		}
		keys := make([]string, len(ids))
		for i, id := range ids {
			keys[i] = coll + ":" + id
		}
		_, err := s.db.Collection(collCounters).DeleteMany(ctx, bson.D{{Key: "_id", Value: bson.D{{Key: "$in", Value: keys}}}})
		return err
	}
	if err := del(collAgent, convIDs); err != nil {
		return err
	}
	return del(collChat, matchIDs)
}

// Export returns one person's own words, for the data export: all of their
// conversation with Suki, and only messages they sent in chats with others.
func (s *Store) Export(ctx context.Context, userID, convID string, matchIDs []string) (agent, chats []Message, err error) {
	if agent, err = s.after(ctx, collAgent, convID, 0, 100000); err != nil {
		return nil, nil, err
	}
	chats = []Message{}
	if len(matchIDs) == 0 {
		return agent, chats, nil
	}
	cur, err := s.db.Collection(collChat).Find(ctx, bson.D{
		{Key: "thread", Value: bson.D{{Key: "$in", Value: matchIDs}}},
		{Key: "sender_id", Value: userID},
	}, options.Find().SetSort(bson.D{{Key: "created_at", Value: 1}}))
	if err != nil {
		return nil, nil, err
	}
	return agent, chats, decodeInto(ctx, cur, &chats)
}
