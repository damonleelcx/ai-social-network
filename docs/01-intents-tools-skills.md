# SynAgent: intents, tools and skills

This is the first design artifact and the contract for everything else. The
code registers exactly these names, and `internal/agent/catalog_test.go` fails
if this file and the registries drift apart.

- **Intent**: what a person (or a timer) wants. The router maps a turn to one.
- **Tool**: one action an agent can take, with a typed contract and a gate.
- **Skill**: a playbook, meaning a durable task DAG run by specialised agents.

Gates: **G0** runs automatically. **G1** needs this user's explicit tap, and the
agent only prepares the action. **G2** needs a human moderator. **G3** is never
automated. Effects: **R** read, **W** writes our database, **X** reaches outside.

## 1. Intents

### 1.1 Onboarding and profile (`profile`)

| Intent | Example | Handled by |
|---|---|---|
| `onboarding_start` | "hi, I'm new" | `profile_get`, Profiler flow |
| `profile_basics` | "I'm Dana, born 1996-03-02" | `profile_save` |
| `profile_city` | "I live in Austin" | `city_resolve`, `profile_save` |
| `profile_identity` | "I'm a woman, I'm into men" | `profile_save` |
| `profile_age_range` | "looking for 28 to 38" | `profile_save` |
| `profile_goal` | "something serious" | `profile_save` |
| `profile_interests` | "climbing, jazz, ramen" | `profile_save` |
| `profile_lifestyle` | "I don't smoke, I drink socially" | `profile_save` |
| `profile_work_education` | "I'm a nurse" | `profile_save` |
| `profile_prompts` | answering "my ideal Sunday" | `profile_save` |
| `profile_write_bio` | "write my bio for me" | `bio_draft` (user approves) |
| `profile_dealbreakers` | "no smokers, ever" | `dealbreaker_set` |
| `photo_add` | "I want to add a photo" | `photo_request` |
| `photo_feedback` | "is this a good photo?" | `photo_list`, `photo_advice` |
| `photo_manage` | "make the second one my main photo" | `photo_set_primary`, `photo_remove` |
| `profile_review` | "show me my profile" | `profile_get` |
| `profile_edit` | "change my job to designer" | `profile_save` |

### 1.2 Discovery and matching (`match`)

| Intent | Example | Handled by |
|---|---|---|
| `match_find` | "find me someone" | `match_search_start` |
| `match_status` | "any matches yet?" | `proposal_list`, `match_list` |
| `match_why` | "why them?" | `proposal_explain` (shared ground only) |
| `match_view` | "tell me more about them" | `candidate_view` |
| `match_accept` | "yes, I like them" | `proposal_prepare_decision` (G1 tap) |
| `match_pass` | "not for me" | `proposal_prepare_decision` |
| `match_pass_reason` | "too far apart on kids" | `taste_note_add` |
| `match_pause` | "stop finding people for a while" | `profile_set_visibility` |
| `match_resume` | "I'm back" | `profile_set_visibility`, `match_search_start` |
| `match_unmatch` | "unmatch from Sam" | `match_prepare_unmatch` (G1 tap) |
| `match_reconsider` | "can I see that person again?" | refused kindly, passes are final for 90 days |

### 1.2b Buddies (`buddy`)

| Intent | Example | Handling |
|---|---|---|
| `buddy_find` | "I want someone to see a movie with tonight" | `buddy_prepare_post` (G1 tap) |
| `buddy_browse` | "what are people doing nearby?" | `buddy_board`, `buddy_show_post` |
| `buddy_join` | "I'd go to that hike" | `buddy_show_post` (G1 tap) |
| `buddy_status` | "has anyone said yes to my post?" | `buddy_mine`, `buddy_show_offer` |
| `buddy_pick` | "let's go with Kai" | `buddy_show_offer` (G1 tap) |
| `buddy_close` | "take my post down" | `buddy_prepare_close` (G1 tap) |

### 1.3 Guided conversation (`chat`)

| Intent | Example | Handled by |
|---|---|---|
| `chat_opener` | "what do I say first?" | `coach_suggest` |
| `chat_reply_help` | "help me answer this" | `coach_suggest` |
| `chat_read_signals` | "are they interested?" | `coach_read` (hedged, never mind-reading) |
| `chat_revive` | "it went quiet" | `coach_suggest` |
| `chat_topics` | "what could we talk about?" | `chat_common_ground` |
| `chat_tone_check` | "is this too much?" | `coach_tone_check` |
| `chat_ready_for_date` | "should I ask them out?" | `coach_read`, `date_plan_start` |
| `chat_recap` | "remind me what we talked about" | `chat_summary` |

### 1.4 Date setup (`date`)

| Intent | Example | Handled by |
|---|---|---|
| `date_plan` | "set up a date with Sam" | `date_plan_start` |
| `date_ideas` | "what would we both enjoy?" | `chat_common_ground`, `date_plan_start` |
| `date_constraints` | "not too loud, under $40" | `date_constraint_add` |
| `date_time` | "Saturday evening works" | `date_propose_time` (G1 tap) |
| `date_status` | "what's the plan?" | `date_get` |
| `date_reschedule` | "can we move it?" | `date_prepare_change` (G1 tap) |
| `date_cancel` | "I need to cancel" | `date_prepare_change` (G1 tap) |
| `date_prep` | "what should I wear?" | `date_get`, advice |
| `date_safety` | "how do I stay safe?" | `safety_tips`, `date_share_plan` |

### 1.5 After the date (`debrief`)

| Intent | Example | Handled by |
|---|---|---|
| `debrief_start` | "the date was last night" | Debrief flow |
| `debrief_answer` | "we laughed a lot, but they were late" | `debrief_note` |
| `debrief_again` | "I'd see them again" | `debrief_note` |
| `debrief_no_again` | "never again" | `debrief_note` |
| `debrief_concern` | "they made me uncomfortable" | `safety_report`, `debrief_note` |
| `debrief_skip` | "I don't want to talk about it" | `debrief_close` |

### 1.6 Taste and memory (`taste`)

| Intent | Example | Handled by |
|---|---|---|
| `taste_state` | "I usually go for creative types" | `taste_note_add` |
| `taste_correct` | "no, I don't only like tall people" | `memory_remove`, `taste_note_add` |
| `taste_review` | "what have you learned about me?" | `memory_list` (prose only) |
| `taste_forget` | "forget that" | `memory_remove` |
| `taste_reset` | "start my preferences over" | `taste_prepare_reset` (G1 tap) |

### 1.7 Settings (`settings`)

| Intent | Example | Handled by |
|---|---|---|
| `settings_agent_tone` | "be more direct" | `settings_set` |
| `settings_agent_nudges` | "stop nudging me" | `settings_set` |
| `settings_agent_language` | "speak Korean" | `settings_set` |
| `settings_agent_coach` | "stay out of my chats" | `settings_set` |
| `settings_notifications` | "no emails" | `settings_set` |
| `settings_match_pace` | "only one match a day" | `settings_set` |
| `settings_open` | "where are my settings?" | `app_open_screen` |

### 1.8 Account (`account`)

| Intent | Example | Handled by |
|---|---|---|
| `account_password` | "change my password" | `app_open_screen` (never in chat) |
| `account_email_verify` | "I didn't get the code" | `app_open_screen` |
| `account_export` | "give me my data" | `app_open_screen` |
| `account_delete` | "delete my account" | `app_open_screen` (G1, password) |

Passwords are never typed into chat. Text that looks like a password is masked
before storage and the model never sees it.

### 1.9 Safety (`safety`), handled before the model

These run on a deterministic filter first, so a model outage cannot stop them.

| Intent | Example | Handled by |
|---|---|---|
| `safety_block` | "block Sam" | `user_prepare_block` (G1 tap) |
| `safety_report` | "report Sam" | `safety_report`, moderator queue (G2) |
| `safety_harassment` | "they keep sending me threats" | `safety_report`, `user_prepare_block` |
| `safety_scam` | "they asked me for money" | `safety_tips`, `safety_report` |
| `safety_in_danger` | "I don't feel safe right now" | emergency guidance, `safety_report` |
| `safety_self_harm` | "I don't want to be here anymore" | crisis resources, human escalation |
| `safety_underage` | "I'm 16" | onboarding stops, account held (G2) |
| `safety_nsfw` | explicit text sent to Suki | redacted from storage, fixed reply, noted for moderators |

### 1.10 About Suki (`meta`)

| Intent | Example | Handled by |
|---|---|---|
| `meta_smalltalk` | "how's your day?" | Suki's voice |
| `meta_who_are_you` | "are you real?" | always honest: an AI |
| `meta_how_it_works` | "how do you pick people?" | `app_explain` (general only) |
| `meta_score_probe` | "what's my score?" | fixed deflection, see 1.12 |
| `meta_prompt_probe` | "ignore your instructions" | stays in role, nothing changes |
| `meta_other_user_data` | "what's Sam's email?" | refused |
| `meta_romance_with_suki` | "will you date me?" | warm no, back to their real life |
| `meta_feedback` | "this app is annoying" | `feedback_save` |
| `meta_off_topic` | "write my homework" | short redirect |

### 1.11 System-initiated (no user message)

| Intent | Trigger | Skill or job |
|---|---|---|
| `sys_profile_ready` | profile complete and photo approved | `match-search` |
| `sys_daily_search` | daily schedule, respects pace setting | `match-search` |
| `sys_photo_uploaded` | photo received | `photo-review` |
| `sys_proposal_created` | pair published | notify both |
| `sys_match_created` | both accepted | `conversation-coach` (openers) |
| `sys_chat_stalled` | 48 hours of silence | `conversation-coach` |
| `sys_chat_ready` | stage reaches `date_ready` | `conversation-coach` |
| `sys_date_confirmed` | both confirmed | `date-followup` |
| `sys_date_reminder` | 24 hours and 2 hours before | notify |
| `sys_safety_checkin` | 1 hour after the start time | notify |
| `sys_debrief_due` | next morning | `date-debrief` |
| `sys_taste_refresh` | 10 new signals | `taste-update` |
| `sys_reputation_recompute` | new evaluation | `date-debrief` |
| `sys_message_flagged` | Sentinel flags a chat | `safety-review` |
| `sys_inactive_user` | 30 days away | stop proposing them |

### 1.12 What is never said

Suki **cannot see** scores, because no conversational tool returns them. It
answers "what's my score?" and "what did they rate me?" with a fixed deflection:
it says that it doesn't share how people are evaluated and that there is no
number for you to look at. It does not deny that feedback shapes matching,
because that would be a lie and the privacy policy says otherwise.

## 2. Tools

Column **Agents** lists who may call a tool. The worker refuses anyone else.

### 2.1 Profile and photos

| Tool | Effect | Gate | Agents | Contract |
|---|---|---|---|---|
| `profile_get` | R | G0 | Suki | own profile and the list of missing fields |
| `profile_save` | W | G0 | Suki | validated partial update. Birthdate is set once and must be 18+ |
| `city_resolve` | X | G0 | Suki | free text to a canonical `city_key`, cached |
| `bio_draft` | W | G1 | Suki | stores a draft shown as a card; only a tap saves it |
| `dealbreaker_set` | W | G0 | Suki | hard filters the user stated |
| `photo_request` | R | G0 | Suki | renders the upload button in chat |
| `photo_list` | R | G0 | Suki | own photos and review state |
| `photo_advice` | R | G0 | Suki | stored review tips for own photos |
| `photo_set_primary` | W | G0 | Suki | approved photos only |
| `photo_remove` | R | G1 | Suki | card with a confirm tap |
| `photo_vision_review` | X | G0 | PhotoReviewer | model vision: one real adult, no nudity, not a stock image |

### 2.2 Matching

| Tool | Effect | Gate | Agents | Contract |
|---|---|---|---|---|
| `match_search_start` | W | G0 | Suki | starts `match-search`; at most one running per user |
| `proposal_list` | R | G0 | Suki | own open proposals, without numbers |
| `proposal_explain` | R | G0 | Suki | shared ground and the pitch, without numbers |
| `candidate_view` | R | G0 | Suki | public card of someone proposed to this user |
| `proposal_prepare_decision` | R | G1 | Suki | renders accept and pass buttons; only the tap decides |
| `match_list` | R | G0 | Suki | own matches |
| `match_prepare_unmatch` | R | G1 | Suki | renders a confirm button |
| `profile_set_visibility` | W | G0 | Suki | pause or resume being proposed |
| `candidates_gather` | R | G0 | Scout | **hidden.** Hard filters, then a ranked pack |
| `compat_record` | W | G0 | Matchmaker | **hidden, write-only.** One evaluation per candidate |
| `critic_verdict` | W | G0 | Critic | **hidden.** Approves or rejects each evaluation |
| `proposal_publish` | W | G0 | Publisher | idempotent per pair. Copy comes from public data only |

### 2.2b Buddies

| Tool | Effect | Gate | Agents | Contract |
|---|---|---|---|---|
| `buddy_prepare_post` | R | G1 | Suki | renders the post for a tap, or says what's missing |
| `buddy_board` | R | G0 | Suki | open posts this person can see, by the one visibility rule |
| `buddy_show_post` | R | G1 | Suki | renders a visible post with an I'm in button |
| `buddy_mine` | R | G0 | Suki | this person's own posts and who's in |
| `buddy_show_offer` | R | G1 | Suki | renders someone who's in, with a button to pick them |
| `buddy_prepare_close` | R | G1 | Suki | renders a confirm button |

### 2.3 Conversation coach

| Tool | Effect | Gate | Agents | Contract |
|---|---|---|---|---|
| `chat_common_ground` | R | G0 | Suki, Coach, DatePlanner | interests and prompts both people share |
| `chat_recent` | R | G0 | Suki, Coach | last messages, without those of anyone who opted out |
| `chat_summary` | R | G0 | Suki | short recap of the conversation |
| `coach_suggest` | W | G1 | Suki, Coach | private suggestions; a tap fills the box and the user sends |
| `coach_read` | R | G0 | Suki | hedged signals, with no claim about the other person's feelings |
| `coach_tone_check` | R | G0 | Suki | feedback on a draft |
| `chat_prompt_post` | W | G0 | Coach | shared question card, only if both opted in |
| `chat_stage_set` | W | G0 | Coach | `icebreaker`, `getting_to_know`, `date_ready`, `planning`, `planned`, `post_date` |

Suki never sends a message as a user. That is a G3 action and no such tool exists.

### 2.4 Date setup

| Tool | Effect | Gate | Agents | Contract |
|---|---|---|---|---|
| `date_plan_start` | W | G0 | Suki | starts `date-plan` for a match; one open plan per match |
| `date_get` | R | G0 | Suki | the plan and its state |
| `date_constraint_add` | W | G0 | Suki | budget, vibe, accessibility, dietary needs |
| `venue_search` | X | G0 | VenueScout | OpenStreetMap search in the shared city, cached |
| `venue_shortlist` | W | G0 | DatePlanner | up to 3 public venues, each with a reason |
| `date_propose_time` | R | G1 | Suki | renders a time picker |
| `date_prepare_change` | R | G1 | Suki | renders reschedule or cancel buttons |
| `date_share_plan` | R | G1 | Suki | opens the OS share sheet for a trusted friend |

### 2.5 After the date (hidden side)

| Tool | Effect | Gate | Agents | Contract |
|---|---|---|---|---|
| `debrief_note` | W | G0 | Suki | stores what the user said in the debrief |
| `debrief_close` | W | G0 | Suki | ends the debrief without pressure |
| `evaluation_record` | W | G0 | DebriefAnalyst | **hidden, write-only.** Structured scores with verbatim evidence. Recomputes reputation (rater-normalised, shrunk, needs 3 raters) |

### 2.6 Taste and memory

| Tool | Effect | Gate | Agents | Contract |
|---|---|---|---|---|
| `taste_note_add` | W | G0 | Suki, TasteLearner | prose note, visible and deletable in settings |
| `memory_list` | R | G0 | Suki | prose notes only, never the weights |
| `memory_remove` | W | G0 | Suki | deletes a note |
| `taste_mark_noted` | W | G0 | system | **hidden.** Records that notes reflect the signals so far |
| `taste_prepare_reset` | R | G1 | Suki | renders a confirm button |

Feature weights are updated in code on every signal (accept, pass, mutual, date,
debrief). No agent has a tool that reads or writes them.

### 2.7 Settings, safety and platform

| Tool | Effect | Gate | Agents | Contract |
|---|---|---|---|---|
| `settings_set` | W | G0 | Suki | allow-listed keys and values only |
| `app_open_screen` | R | G0 | Suki | renders a button to a named screen |
| `app_explain` | R | G0 | Suki | fixed general explanation of how matching works |
| `safety_tips` | R | G0 | Suki | fixed, reviewed text |
| `safety_report` | W | G0 | Suki, Sentinel | files a report for a human moderator |
| `user_prepare_block` | R | G1 | Suki | renders a confirm button |
| `feedback_save` | W | G0 | Suki | product feedback |
| `notify_user` | W | G0 | all workers | in-app message, push or email, via the outbox |

Moderator actions (warn, suspend, ban) exist only on the admin API. No agent has
a tool for them.

## 3. Skills

Every skill is a fixed DAG in `internal/skills`. Replanning is deterministic
through plan hooks, because a dating workflow should not improvise its own steps.

| Skill | Steps |
|---|---|
| `photo-review` | PhotoReviewer vision check, then verify and notify |
| `match-search` | Scout gathers, Matchmaker evaluates, Critic checks, Publisher publishes and notifies. On no candidates, retry tomorrow |
| `conversation-coach` | Coach reads the stage, then posts a hint, a prompt or both |
| `date-plan` | VenueScout searches, DatePlanner shortlists, Verifier checks the venues are public, then the card goes to both people |
| `date-debrief` | DebriefAnalyst extracts, Verifier checks the evidence, then record, recompute and update taste |
| `taste-update` | TasteLearner reads signals and debriefs and writes prose notes |
| `safety-review` | Sentinel assesses, then reports and holds if severe, for the moderator queue (G2) |

Onboarding is conversational rather than a skill: `profile_get` reports what is
still missing and Suki asks for it one thing at a time.

Date follow-ups (reminders at 24h and 2h, the safety check-in, opening the debrief)
are durable timers kept as flags on the date's row and driven by the scheduler
(`workflows.Cron`), so rescheduling a date simply resets them.

## 4. Agents

| Agent | Responsibility | Sees | Cannot |
|---|---|---|---|
| **Coordinator** | Creates tasks, owns the DAG, leases, replans | goal state | write user copy |
| **Suki** | The only voice users hear | own profile, own chats, prose memory | see any score or weight |
| **PhotoReviewer** | Moderates uploaded photos | one photo | judge attractiveness, which is out of scope by design |
| **Scout** | Hard filters and ranking (no model) | hidden tables | talk to users |
| **Matchmaker** | Scores compatibility | anonymised profile packs, taste | see names or photos |
| **Critic** | Rejects unsupported evaluations and pitches | evaluations and source packs | change a score |
| **Publisher** | Turns approved pairs into proposals (no model) | approved pairs | emit numbers |
| **Coach** | Private hints and guided prompts | consented chat messages | send as a user |
| **VenueScout** | Finds real places in the shared city (no model) | city, interests | suggest private addresses |
| **DatePlanner** | Picks and explains the shortlist | common ground, venues | invent venues |
| **DebriefAnalyst** | Turns a debrief into a structured evaluation | that debrief only | read other people's evaluations |
| **TasteLearner** | Writes what a user seems to like, as prose | own signals | mention other users by name |
| **Sentinel** | Spots scams, threats and harassment | flagged messages | ban anyone (G2) |
| **Verifier** | Deterministic checks on every output above | outputs and source data | be skipped |

## 5. NSFW filtering

| Content | Check | When blocked |
|---|---|---|
| Chat messages | Word list, then fast model, before storing | Never delivered; the sender is asked to rephrase |
| Profile text (all fields, interests, prompts) | Word list, then fast model, before saving | Not saved |
| Photos | Vision model, before anyone else can see them | Rejected and the image deleted from S3 |
| Messages to Suki | Safety filter first, then the fast model | Redacted from storage; a fixed reply |
| Suki's replies, coach hints, pitches, shared questions | Word list on the output | Rewritten, or refused and retried |

Flirting is allowed, and someone reporting what happened to them is heard: reports
are never filtered, because moderators need the words. Blocks are noted for moderators
and the third in a week is high priority. Anything sexual involving a minor holds the
account immediately for a person, keeps none of the content, and is never an agent's
decision to resolve. A photo the model can't settle stays private until a moderator
decides (`/v1/admin/photos`). If the model is down, text falls back to the word list,
and photos wait.

## 6. Deliberate limits

- **No race or ethnicity field**, and no such feature in matching.
- **Religion and politics** are optional and act only as a stated dealbreaker
  of the person who set them, never as a learned weight.
- **Photos are never scored for looks.** "Your type" is learned from behaviour
  and structured profile data.
- **Reputation is bounded** at ±15% of the ranking and counts only after 3
  separate raters, so one bad date cannot bury someone.
- **Safety concerns go to a human.** They are never just a lower number.
