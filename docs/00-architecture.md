# SynAgent architecture

SynAgent is a dating app where an AI matchmaker, Suki, does the work of finding,
introducing and helping people meet. This document is how it is built, and where each
agent-development principle lives in the code.

## Objective

**Business goal:** people in the same city go on dates they'd want to repeat.

**Success criteria:** a new person finishes onboarding by talking to Suki; introductions
are relevant and grounded in real shared interests; people who match reach a planned date;
and evaluations are used without ever being shown.

**Acceptable failure:** a background search can fail and retry tomorrow, and a coach hint
can be skipped. **Not acceptable:** a score reaching a user, unmoderated content reaching
another person, a lost message, or a duplicate introduction.

**Expected duration:** a turn takes seconds; a search takes minutes; a date lifecycle
takes days. All of it has to survive restarts.

## System

```
Mobile app (React Native: iOS, Android)
   │  HTTPS + one WebSocket that says "what changed"
   ▼
synagent-web ───────────► MongoDB (all chat messages)
   │                  └─► S3 (photo bytes, private)
   ▼
Postgres: profiles, goals, tasks, leases, checkpoints, events, schema hidden
   ▲
synagent-worker
   ├─ Turn workers ── Suki, one durable turn at a time per person
   ├─ Mission workers ── skills as task DAGs, run by specialists
   └─ Scheduler ── leader-elected: leases, promotion, timers (Cron)
        │
        ▼ tools with typed contracts
   Model · Fish Audio · OpenStreetMap · Expo push · mail relay
```

Everything runs on one k3s node in the `synagent` namespace, Postgres and MongoDB as
StatefulSets. See `deploy/`.

## The agents

The full contracts (sees, can do, cannot do) are in
[01-intents-tools-skills.md](01-intents-tools-skills.md) §4.

**Suki** is the one voice people hear. A deterministic safety filter runs before her, and
an output guard after. **Specialists** never speak to anyone: Scout, Matchmaker, Critic,
Publisher, PhotoReviewer, Coach, VenueScout, DatePlanner, DebriefAnalyst, TasteLearner,
Sentinel. **Verifiers** are code that reads the database rather than trusting a model.

### Example: `match-search`

```
gather (Scout, no model)
   └─► evaluate (Matchmaker, compat_record per candidate; verified: all evaluated)
          └─► critique (Critic, independent; verified: all ruled on)
                 └─► publish (Publisher, no model; idempotent per pair)
```

## How hidden scoring stays hidden

The rule is enforced by what code *can't do*, not by asking it to behave:

1. Scores are in a separate Postgres schema, `hidden`, reached only through
   `internal/hidden`.
2. `internal/archtest` fails the build if the API, Suki's package or the other
   people-facing packages import it, or if SQL elsewhere names the schema.
3. No tool Suki can hold returns a score (`TestSukiCannotHoldAHiddenTool`). She can't leak
   what she can't read.
4. What leaves the hidden package carries copy, not numbers (`hidden.Publishable`).
5. Suki's output passes a leak guard, and a stubborn draft is replaced with fixed text.
6. The end-to-end test records every response body of a full journey and searches them
   for score fields.

**Fairness rules** live in `internal/hidden`. There is no race, religion, politics or looks
feature; reputation moves ranking by at most ±15% and needs three separate raters; and
raters are normalised against their own average. Safety concerns go to a human and never
just lower a number.

## Learning someone's type

Every accept, pass, match, reply, date and debrief is one online logistic-regression step
over a closed feature list (`hidden.Features`). Interests generalise through categories, so
someone who keeps liking hikers will also see climbers. The Scout ranks with these weights
on both sides, since a match needs two yeses. After enough new behaviour, the TasteLearner
writes what it sees as plain notes, which the person can read and delete in Settings, and
they can reset everything. Weights are never shown.

## The principles, mapped

| Principle | Where |
|---|---|
| Objective | Above |
| Decompose | Skills in `internal/workflows/workflows.go` |
| Agent roles | `registerRoles` in `internal/workflows/workers.go`; each role sets its prompt and exactly what it sees |
| Orchestration layer | `internal/engine`: planner, scheduler, workers |
| Agent contracts | `internal/tools/contract.go`: typed input and output, gate, timeout, idempotency |
| Task graph | `tasks` and `task_deps`, with depth and size limits |
| Shared task state | `goals`, `tasks`, `checkpoints`, `events` |
| Global and local context | `Role.Context` rebuilds only what a role may see, from the database, every time |
| Durable communication | `pg_notify` wake-ups over stored state; the app's socket only says what changed |
| Execution loop | `Worker.runLLM`: observe, act, verify, checkpoint, continue |
| Durable execution | Each step is checkpointed; a crash resumes from the last one |
| Idempotency | Tool ledger with unique keys; MongoDB writes keyed per thread; one proposal per pair ever |
| Workers and queues | `FOR UPDATE SKIP LOCKED` with leases and fencing epochs; web and worker scale separately |
| Wake-ups | `workflows.Cron`, `run_after`, NOTIFY |
| Checkpoints | `checkpoints` table |
| Verification agents | `registerVerifiers`, which read the database; `critic` role is independent of the Matchmaker |
| Critic loops | The Critic must rule on every evaluation before anything publishes |
| Replanning | Deterministic `planHook`: each failure has a defined truthful outcome |
| Local failure handling | Retries with backoff per task; a failed coach hint affects nothing else |
| Escalation | `needs_attention` goals and the moderator queue (`/v1/admin/*`) |
| Autonomy control | Gates: G0 automatic, G1 the person's tap, G2 a human moderator, G3 never |
| Conflict prevention | Leases, fencing, one turn per person, one open plan per match, advisory-lock scheduler leader |
| Recursion limits | `goalLimits`: steps, tools, tokens, cost, depth, replans. Account and deployment daily caps |
| Long-term memory | Mission state, per-turn history, episodic summaries, prose notes, learned weights, each separate |
| Artifacts | Shortlists, packs and evaluations are rows, not conversation text; photos in S3 |
| Observability | `events`, `tool_calls`, `llm_calls`, `/v1/admin/goals/{id}/timeline` |
| Timeline | `/v1/admin/goals/{id}/timeline`: what each agent did and why |
| Safety boundaries | `internal/safety`, `internal/moderation`, tool allow-lists per domain, data markers around user text |
| Multi-agent evaluations | `internal/e2e`, `internal/engine/*_test.go` |
| Disaster recovery | `TestTurnSurvivesWorkerCrash`, engine crash tests, and a live pod kill during a deployment test |
| Reliability metrics | `/v1/admin/metrics`: completion, success, recovery, intervention, latency, spend |
| Continuous improvement | A failure becomes an e2e or eval case; see `internal/e2e` |

## Streaming

A turn runs in a worker, and the person's socket is on a web pod, so deltas travel by
Postgres NOTIFY to whichever pod holds the socket. Streaming can't weaken the guards:
text stays 48 characters behind the model, and the leak and NSFW checks run over the held
text before any of it is released. A failed check tells the client to clear the bubble,
and the reply goes through the normal rewrite path. The stream is a preview, so losing an
event costs nothing: the stored message replaces it.

## Storage

| Data | Where | Why |
|---|---|---|
| Accounts, profiles, workflow, safety | Postgres | Transactions and queues |
| Scores and taste | Postgres, schema `hidden` | A boundary that can be checked |
| All chat messages | MongoDB | As specified; writes are keyed since there's no transaction across the two |
| Photo bytes | S3, private | Re-encoded first so EXIF is gone; streamed only after an access check |

## Known limits

- **Single node.** Everything is on one EC2 instance, and databases use local volumes.
  Nightly dumps go to S3, so a node loss can lose up to a day.
- **NSFW text with the model down** falls back to the word list. Photos wait instead.
- **Moderation is an API, not a console**, and needs a person to act on it.
- **Legal text has not had a lawyer's review**, and it names no legal entity, address or governing law yet. See `internal/httpapi/legal.go`.
