package tools

import (
	"context"
	"time"
)

// The generic tools every playbook may use. Game-specific tools (the studio's
// save_rules, save_module, check_module, playtest, publish_game) register
// themselves next to the script runtime they drive.
// PostToConversation is set at startup to chatstore's writer.
var PostToConversation func(ctx context.Context, convID, userID, content string, meta map[string]any, key string)

func init() {
	Register(&Tool{
		Name: "notify_user",
		Description: "Post an update into this person's conversation with Suki. " +
			"Use for meaningful milestones only, in the person's language.",
		Input:  `{"type":"object","required":["message"],"additionalProperties":false,"properties":{"message":{"type":"string","minLength":2,"maxLength":4000}}}`,
		Output: `{"type":"object","required":["posted"]}`,
		Effect: Write, Gate: G0, Timeout: 5 * time.Second,
		Run: func(ctx context.Context, env *Env, a map[string]any) (map[string]any, error) {
			if env.ConversationID == "" || PostToConversation == nil {
				return map[string]any{"posted": false}, nil
			}
			// The key makes a retried step land on the same message.
			key := Key(env, "notify", Str(a, "message"))
			PostToConversation(ctx, env.ConversationID, env.UserID, Str(a, "message"), map[string]any{"goal_id": env.GoalID, "kind": "update"}, key)
			_, _ = env.Pool.Exec(ctx, `SELECT pg_notify('syn_user', $1)`, env.UserID)
			return map[string]any{"posted": true}, nil
		},
	})
}
