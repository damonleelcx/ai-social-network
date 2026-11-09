// Package workflows is the orchestration: the skills (task DAGs), the specialist
// roles behind them, the verifiers that check their work, the plan hooks that
// replan deterministically, and the timers. Durability comes from internal/engine;
// this package says what the work is.
package workflows

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/damonleelcx/ai-social-network/internal/agent"
	"github.com/damonleelcx/ai-social-network/internal/billing"
	"github.com/damonleelcx/ai-social-network/internal/chatstore"
	"github.com/damonleelcx/ai-social-network/internal/dating"
	"github.com/damonleelcx/ai-social-network/internal/engine"
	"github.com/damonleelcx/ai-social-network/internal/hidden"
	"github.com/damonleelcx/ai-social-network/internal/i18n"
	"github.com/damonleelcx/ai-social-network/internal/moderation"
	"github.com/damonleelcx/ai-social-network/internal/persona"
	"github.com/damonleelcx/ai-social-network/internal/photostore"
	"github.com/damonleelcx/ai-social-network/internal/places"
	"github.com/damonleelcx/ai-social-network/internal/profile"
	"github.com/damonleelcx/ai-social-network/internal/skills"
	"github.com/damonleelcx/ai-social-network/internal/tools"
)

// Deps are the handles workers need. Set once at startup by Init.
type Deps struct {
	Billing *billing.Service
	Pool             *pgxpool.Pool
	Store            *engine.Store
	Model            *engine.Model
	Chats            *chatstore.Store
	Photos           photostore.Store
	Places           *places.Client
	Dating           *dating.Service
	FastModel        string
	PhotoAutoApprove bool
	// DebriefDelay is how long after a date's start Suki asks how it went.
	DebriefDelay time.Duration
	HTTP         *http.Client
}

var d *Deps

// Init wires the domain to the engine. It must run before workers start.
func Init(deps *Deps) {
	d = deps
	if d.DebriefDelay == 0 {
		d.DebriefDelay = 14 * time.Hour
	}
	d.Dating.Learn = func(ctx context.Context, user, subject, kind, key string) {
		if err := hidden.Signal(ctx, d.Pool, user, subject, kind, key); err != nil {
			slog.Error("taste signal", "kind", kind, "err", err)
		}
	}
	d.Dating.Hooks = dating.Hooks{
		Coach: func(ctx context.Context, matchID, userID, trigger string) {
			if err := StartCoach(ctx, matchID, userID, trigger); err != nil {
				slog.Error("start coach", "err", err)
			}
		},
		DatePlan: func(ctx context.Context, dateID, matchID, userID string) {
			if err := start(ctx, userID, "date-plan", "date", "Find a date spot", "Find public places these two people would both enjoy, shortlist up to three and put them to both.", dateID); err != nil {
				slog.Error("start date plan", "err", err)
			}
		},
		SafetyScan: func(ctx context.Context, matchID, reportID string) {
			var subject string
			if err := d.Pool.QueryRow(ctx, `SELECT coalesce(subject_id::text,'') FROM reports WHERE id=$1`, reportID).Scan(&subject); err != nil || subject == "" {
				return
			}
			if err := start(ctx, subject, "safety-review", "safety", "Review a flagged message", "Assess a flagged message for the moderators.", reportID); err != nil {
				slog.Error("start safety review", "err", err)
			}
		},
	}
	tools.PostToConversation = d.Chats.Post
	d.Dating.Mod = &moderation.Filter{Model: d.Model, LLM: d.FastModel}
	profile.TextCheck = func(ctx context.Context, userID, text string) error {
		v := d.Dating.Mod.Check(ctx, userID, moderation.Profile, text)
		if !v.Blocked {
			return nil
		}
		d.Dating.Strike(ctx, userID, "", "profile", v, text)
		return &profile.ValidationError{Msg: moderation.ProfileBlockedMessage}
	}
}

var goalLimits = engine.Limits{MaxIterations: 30, MaxToolCalls: 60, MaxTokens: 150_000, MaxCostUSD: 1, MaxDays: 2, MaxDepth: 6, MaxTasksPerPlan: 8, MaxReplans: 2, MaxTotalTasks: 16}

// start creates a goal unless the same work is already running: at most one per
// person, skill and subject. subject may be empty.
func start(ctx context.Context, userID, skill, domain, title, objective, subject string) error {
	var running bool
	if err := d.Pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM goals WHERE user_id=$1 AND skill=$2 AND coalesce(subject_id,'')=$3 AND status IN ('planning','active'))`,
		userID, skill, subject).Scan(&running); err != nil {
		return err
	}
	if running {
		return nil
	}
	conv, err := dating.ConversationID(ctx, d.Pool, userID)
	if err != nil {
		return err
	}
	lang, _ := dating.GetSettings(ctx, d.Pool, userID)["language"].(string)
	return d.Store.CreateGoal(ctx, &engine.Goal{UserID: userID, ConversationID: conv, Title: title, Objective: objective, Domain: domain, Skill: skill,
		SubjectID: subject, Limits: goalLimits, Language: lang})
}

// StartSearch starts looking for this person. asked says they requested it, so
// they hear back even when nobody turns up; the daily search stays quiet.
func StartSearch(ctx context.Context, userID string, asked bool) error {
	subject := ""
	if asked {
		subject = "asked"
	}
	return start(ctx, userID, "match-search", "match", "Find someone", "Find people in this person's city who suit them and introduce the best.", subject)
}

// StartPhotoReview queues moderation for a new photo.
func StartPhotoReview(ctx context.Context, userID, photoID string) error {
	return start(ctx, userID, "photo-review", "photo", "Review a photo", "Check this photo is allowed on the app.", photoID)
}

// StartCoach queues a private word from Suki about one conversation.
func StartCoach(ctx context.Context, matchID, userID, trigger string) error {
	st := dating.GetSettings(ctx, d.Pool, userID)
	if st["chat_coach"] == false {
		return nil
	}
	return start(ctx, userID, "conversation-coach", "coach", "Coach a conversation", "trigger="+trigger, matchID)
}

// StartDebriefAnalysis turns a closed debrief into a hidden evaluation.
func StartDebriefAnalysis(ctx context.Context, userID, dateID string) error {
	return start(ctx, userID, "date-debrief", "debrief", "Understand how the date went", "Extract what this person said about their date.", dateID)
}

// RewardReferrer pays whoever invited this person, and the person too, and tells them both.
func RewardReferrer(ctx context.Context, userID string) error {
	paid, err := d.Billing.RewardReferral(ctx, userID)
	if err != nil || paid == nil {
		return err
	}
	lang := d.Dating.Lang(ctx, paid.UserID)
	body := i18n.T(lang, "A friend you invited just joined. I added {credits} credits, so you have {balance} now.", "credits", strconv.Itoa(paid.Credits), "balance", strconv.Itoa(paid.Balance))
	if err := d.Dating.PostFromSuki(ctx, paid.UserID, body, "happy", nil, "referral:"+userID); err != nil {
		slog.Error("referral announce", "err", err)
	}
	d.Dating.Touch(ctx, paid.UserID, "credits")
	if paid.InviteeCredits > 0 {
		lang := d.Dating.Lang(ctx, userID)
		body := i18n.T(lang, "You joined through a friend's invite, so I added {credits} credits to welcome you. You have {balance} now.", "credits", strconv.Itoa(paid.InviteeCredits), "balance", strconv.Itoa(paid.InviteeBalance))
		if err := d.Dating.PostFromSuki(ctx, userID, body, "happy", nil, "invited:"+userID); err != nil {
			slog.Error("invited announce", "err", err)
		}
		d.Dating.Touch(ctx, userID, "credits")
	}
	return nil
}

// AfterProfileChange starts matching the moment onboarding completes.
func AfterProfileChange(ctx context.Context, userID string) error {
	done, err := profile.ActivateIfComplete(ctx, d.Pool, userID)
	if err != nil || !done {
		return err
	}
	// First, and not fatal: this moment doesn't come round again.
	if err := RewardReferrer(ctx, userID); err != nil {
		slog.Error("referral reward", "err", err)
	}
	defer func() {
		if err := EnqueueSystemTurn(context.WithoutCancel(ctx), userID, agent.LookingIntent, showLookingPrompt, "looking:"+userID); err != nil {
			slog.Error("show looking for", "err", err)
		}
	}()
	if err := d.Dating.PostFromSuki(ctx, userID, "Your profile is ready, and it looks great. I'm off to find people I think you'd really like. I'll be back soon!", "happy", nil, "profile-ready"); err != nil {
		return err
	}
	return StartSearch(ctx, userID, true)
}

// ── Roles ───────────────────────────────────────────────────────────────────

// role prompts share the worker base and add a contract of their own.
func system(lang string, body string) string {
	return persona.WorkerSystem(lang, time.Now()) + "\n\n" + body
}

func init() {
	registerSkills()
	registerTools()
	registerSukiTools()
	registerRoles()
	registerVerifiers()
	for _, name := range []string{"match-search", "photo-review", "conversation-coach", "date-plan", "date-debrief", "taste-update", "safety-review"} {
		engine.RegisterPlanHook(name, planHook)
	}
}

// planHook replaces the generic LLM judge and replanner. These workflows are
// fixed, so finishing and failing are decisions code can make. Each failure path
// leaves the person in a truthful state and the goal visible to operators.
func planHook(ctx context.Context, p *engine.Planner, g *engine.Goal, t *engine.Task) (bool, error) {
	if t.Kind == "finish" {
		if err := p.Store.Complete(ctx, t, map[string]any{"summary": "all steps finished"}); err != nil {
			return true, err
		}
		return true, p.Store.SetGoalStatus(ctx, g.ID, "completed", "all steps finished")
	}
	if err := onGoalFailed(ctx, g, t.Spec.Reason); err != nil {
		return true, err
	}
	if err := p.Store.Complete(ctx, t, map[string]any{"summary": "failed: " + t.Spec.Reason}); err != nil {
		return true, err
	}
	return true, p.Store.SetGoalStatus(ctx, g.ID, "needs_attention", t.Spec.Reason)
}

func onGoalFailed(ctx context.Context, g *engine.Goal, why string) error {
	slog.Warn("goal failed", "goal", g.ID, "skill", g.Skill, "why", why)
	switch g.Skill {
	case "photo-review":
		// Never fail open: an unreviewed photo waits for a human.
		if _, err := profile.RecordReview(ctx, d.Pool, g.SubjectID, "review", "", ""); err != nil {
			return err
		}
		_, err := d.Dating.FileReport(ctx, dating.Report{SubjectID: g.UserID, Source: "sentinel", Reason: "photo_review_failed", Details: why, Key: "photo-fail:" + g.SubjectID})
		return err
	case "date-plan":
		return d.Dating.FailSearch(ctx, g.SubjectID, "The search didn't work this time.")
	case "match-search":
		if g.SubjectID == "asked" {
			return d.Dating.PostFromSuki(ctx, g.UserID, "My search hit a snag, sorry! I'll try again on my next round, so nothing's lost.", "pouty", nil, "search-fail:"+g.ID)
		}
	case "safety-review":
		// The report is already in the queue; the human sees it without the assessment.
	}
	return nil
}

// ── Skills ──────────────────────────────────────────────────────────────────

func registerSkills() {
	skills.Register(&skills.Skill{Name: "match-search", Domain: "match", Title: "Find someone", Fixed: true,
		Description: "Scout filters and ranks, Matchmaker evaluates, Critic checks, Publisher introduces.",
		Criteria:    []string{"Every candidate evaluated and ruled on", "Approved pairs introduced"},
		Milestones:  []string{"Scout", "Evaluate", "Critique", "Publish"},
		Steps: []skills.Step{
			{Key: "gather", Title: "Scout candidates", Role: "scout", Tool: "candidates_gather", Args: map[string]any{}},
			{Key: "evaluate", Title: "Evaluate compatibility", Role: "matchmaker", Deps: []string{"gather"}, Tools: []string{"compat_record"}, Verify: []string{"compat_complete"}, MaxSteps: 12,
				Instructions: "Evaluate every candidate in the pack with compat_record, then finish."},
			{Key: "critique", Title: "Check the evaluations", Role: "critic", Deps: []string{"evaluate"}, Tools: []string{"critic_verdict"}, Verify: []string{"verdicts_complete"}, MaxSteps: 12,
				Instructions: "Rule on every evaluation with critic_verdict, then finish."},
			{Key: "publish", Title: "Make introductions", Role: "publisher", Deps: []string{"critique"}, Tool: "proposal_publish", Args: map[string]any{}},
		}})
	skills.Register(&skills.Skill{Name: "photo-review", Domain: "photo", Title: "Review a photo", Fixed: true,
		Description: "A vision check, recorded before anyone else can see the photo.", Criteria: []string{"The photo has a decision"}, Milestones: []string{"Review"},
		Steps: []skills.Step{{Key: "review", Title: "Check the photo", Role: "photo_reviewer", Tool: "photo_vision_review", Args: map[string]any{}, Verify: []string{"photo_decided"}}}})
	skills.Register(&skills.Skill{Name: "conversation-coach", Domain: "coach", Title: "Coach a conversation", Fixed: true,
		Description: "A private hint and tappable suggestions.", Criteria: []string{"A hint was posted"}, Milestones: []string{"Coach"},
		Steps: []skills.Step{{Key: "coach", Title: "Write a hint", Role: "coach", Tools: []string{"chat_common_ground", "coach_suggest", "chat_stage_set", "chat_prompt_post"}, Verify: []string{"called:coach_suggest"}, MaxSteps: 5,
			Instructions: "Decide what would help this person most right now and post it with coach_suggest."}}})
	skills.Register(&skills.Skill{Name: "date-plan", Domain: "date", Title: "Find a date spot", Fixed: true,
		Description: "Search real public places, shortlist by shared interests, verify, put to both.", Criteria: []string{"A verified shortlist reached both people"}, Milestones: []string{"Search", "Shortlist"},
		Steps: []skills.Step{
			{Key: "search", Title: "Search places", Role: "venue_scout", Tool: "venue_search", Args: map[string]any{}},
			{Key: "shortlist", Title: "Choose the shortlist", Role: "date_planner", Deps: []string{"search"}, Tools: []string{"venue_shortlist"}, Verify: []string{"venues_public"}, MaxSteps: 4,
				Instructions: "Pick up to three places from the candidates and explain each with venue_shortlist."},
		}})
	skills.Register(&skills.Skill{Name: "date-debrief", Domain: "debrief", Title: "Understand how the date went", Fixed: true,
		Description: "Turn what one person said into a structured, hidden evaluation.", Criteria: []string{"The debrief was analysed"}, Milestones: []string{"Analyse"},
		Steps: []skills.Step{{Key: "analyse", Title: "Analyse the debrief", Role: "debrief_analyst", Tools: []string{"evaluation_record"}, Verify: []string{"evaluation_saved"}, MaxSteps: 4,
			Instructions: "Record one evaluation from what this person said with evaluation_record, then finish."}}})
	skills.Register(&skills.Skill{Name: "taste-update", Domain: "taste", Title: "Learn what they like", Fixed: true,
		Description: "Write learned taste as prose the person can read and delete.", Criteria: []string{"Notes reflect recent behaviour"}, Milestones: []string{"Notes"},
		Steps: []skills.Step{
			{Key: "notes", Title: "Write taste notes", Role: "taste_learner", Tools: []string{"taste_note_add"}, MaxSteps: 5, Instructions: "Save at most two short new notes with taste_note_add, or none if nothing new stands out."},
			{Key: "mark", Title: "Mark as noted", Deps: []string{"notes"}, Tool: "taste_mark_noted", Args: map[string]any{}},
		}})
	skills.Register(&skills.Skill{Name: "safety-review", Domain: "safety", Title: "Review a flagged message", Fixed: true,
		Description: "Assess a flagged message and hand it to a human.", Criteria: []string{"The report carries an assessment"}, Milestones: []string{"Assess"},
		Steps: []skills.Step{{Key: "assess", Title: "Assess the message", Role: "sentinel", Tools: []string{"safety_report"}, Verify: []string{"called:safety_report"}, MaxSteps: 3,
			Instructions: "Assess the flagged message and record your assessment with safety_report."}}})
}

func fmtList(l []string) string {
	if len(l) == 0 {
		return "(none)"
	}
	s := ""
	for i, x := range l {
		if i > 0 {
			s += ", "
		}
		s += x
	}
	return s
}

// ResetTaste is the person's "start my preferences over": learned weights and
// prose notes both go. It lives here so the API needn't touch the hidden side.
func ResetTaste(ctx context.Context, userID string) error {
	if err := hidden.ResetTaste(ctx, d.Pool, userID); err != nil {
		return err
	}
	if _, err := d.Pool.Exec(ctx, `DELETE FROM looking_reads WHERE user_id=$1`, userID); err != nil {
		return err
	}
	_, err := d.Pool.Exec(ctx, `DELETE FROM memories WHERE user_id=$1`, userID)
	return err
}
