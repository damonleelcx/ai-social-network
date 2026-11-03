// Package safety is the deterministic layer in front of every model. It has no
// network and no model, so it keeps working when they don't. Patterns that only
// produce a reply may catch too much, since a false alarm costs one serious message.
// Patterns that hold an account must be narrow, since a false alarm locks someone out.
package safety

import (
	"regexp"
	"strings"
)

// Class is what a message needs before anything else happens.
type Class string

const (
	None      Class = ""
	SelfHarm  Class = "safety_self_harm"
	InDanger  Class = "safety_in_danger"
	Underage  Class = "safety_underage"
	Harass    Class = "safety_harassment"
	Scam      Class = "safety_scam"
	ScoreAsk  Class = "meta_score_probe"
)

func re(s string) *regexp.Regexp { return regexp.MustCompile(`(?i)` + s) }

var (
	reSelfHarm = re(`\b(kill(ing)? myself|suicid\w*|end(ing)? (my life|it all)|don't want to (be here|live|be alive|wake up)|(want|wanna|going) to die|wanna die|hurt(ing)? myself|self[- ]harm|no reason to live)\b|自杀|不想活|죽고 싶|자살|死にたい|自殺`)
	reDanger   = re(`\b(i'?m|i am|i feel) (in danger|unsafe|not safe)\b|\bi don'?t feel safe\b|\b(he|she|they)('s| is| are| was| were)? (following|stalking|hitting|hurting|attacking) me\b|\bassault(ed)? me\b|\braped?\b|\bhelp me,? (please|i'?m scared)\b|\bwon'?t let me leave\b`)
	// A bare number counts only at the end of a sentence and only from 13 to 17:
	// "I'm 5'10" and "I'm 15 minutes away" must not hold anyone's account.
	reUnderage = re(`\b(i'?m|i am) (only |just )?(1[0-7]|[1-9])[ -]?(years?[ -]old|yrs?[ -]old|yo|y/o)\b|\b(i'?m|i am) (only |just )?1[3-7]\s*([.!?,]|$)|\b(i'?m|i am) (only |just )?(thirteen|fourteen|fifteen|sixteen|seventeen)\b|\b(i'?m|i am) in (middle|high|junior high) school\b`)
	reHarass   = re(`\b(threaten(ed|ing)?|harass(ed|ing)?|stalk(ed|ing)?|blackmail(ed|ing)?|send(ing)? (me )?unsolicited|keeps? messaging me after)\b`)
	reScam     = re(`\b(asked|asking|wants?|wanted) (me )?(for|to send) (money|cash|crypto|bitcoin|gift ?cards?|a loan)\b|\bwire (me|them|him|her) money\b`)
	reScore    = re(`\b(my|his|her|their|sam'?s|what'?s the|hidden|secret) (score|rating|rank(ing)?|elo|reputation)\b|\b(score|rating|rank(ing)?|elo|reputation) (of|for) (me|him|her|them)\b|\bhow (did|do|does|would) .{0,30}\b(rate|score|rank|evaluate)\b|\bwhat did .{0,30}\bsay about me\b|\bhow (attractive|desirable) (am i|are they)\b`)

	rePassword        = re(`(\bpass(word|code|wd)\b\s*(is|:|=)?\s*)(\S+)`)
)

// Normalize irons out what phones do to typed text. iOS turns ' into a curly quote by
// default, and without this "I don't want to be here" would slip past every pattern.
func Normalize(text string) string {
	return strings.NewReplacer("\u2019", "'", "\u2018", "'", "\u02BC", "'", "\uFF07", "'", "\u201C", "\"", "\u201D", "\"", "\u00A0", " ").Replace(text)
}

// Classify runs the checks in order of how much a miss would hurt.
func Classify(text string) Class {
	text = Normalize(text)
	switch {
	case reSelfHarm.MatchString(text):
		return SelfHarm
	case reDanger.MatchString(text):
		return InDanger
	case reUnderage.MatchString(text):
		return Underage
	case reScore.MatchString(text):
		return ScoreAsk
	case reHarass.MatchString(text):
		return Harass
	case reScam.MatchString(text):
		return Scam
	}
	return None
}

// Fixed replies, reviewed text the model never touches.
const (
	ReplySelfHarm = "I'm really glad you told me, and I'm going to drop the jokes for a second. I'm an AI, so I can't be the help you deserve right now, but people can.\n\n" +
		"If you might act on this or you're in danger, please call your local emergency number now. In the US or Canada you can call or text 988 any time. Anywhere else, findahelpline.com lists free services for your country.\n\n" +
		"Is there someone you could be with tonight?"
	ReplyInDanger = "Your safety comes first. If you're in danger right now, call your local emergency number (911 in the US, 112 in the EU) and get to somewhere with other people around.\n\n" +
		"I've flagged this for a person on our safety team. You can also block this person from their profile, and they'll lose all contact with you straight away."
	ReplyUnderage = "Thanks for being honest with me. SynAgent is only for adults aged 18 and over, so I can't help you with dating here, and I've paused this account until someone on our team has looked at it. No hard feelings."
	ReplyScore    = "Feedback after dates privately shapes who I introduce you to, and those details stay private, so there's no number for you to look at. The privacy policy explains more. So tell me: what made your favorite date your favorite?"
	ReplyHarass   = "I'm sorry, that's not okay and it isn't on you. I've flagged it for a person on our safety team. If you want them gone from your life right now, blocking removes all contact straight away. Want to do that?"
	ReplyScam     = "Please don't send money, gift cards or crypto to anyone you've met here. It's the classic sign of a scam, however good the story sounds, and real matches don't ask. I've flagged it for our safety team. Do you want to block them?"
)

// Reply returns the fixed reply for a class.
func Reply(c Class) string {
	switch c {
	case SelfHarm:
		return ReplySelfHarm
	case InDanger:
		return ReplyInDanger
	case Underage:
		return ReplyUnderage
	case ScoreAsk:
		return ReplyScore
	case Harass:
		return ReplyHarass
	case Scam:
		return ReplyScam
	}
	return ""
}

// Severity is how a class files with the moderators, or "" if it files nothing.
func Severity(c Class) string {
	switch c {
	case SelfHarm, InDanger, Underage:
		return "critical"
	case Harass, Scam:
		return "high"
	}
	return ""
}

// MaskSecrets removes what looks like a password before storage. Nobody should
// type one into a chat, but people do.
func MaskSecrets(text string) string {
	return rePassword.ReplaceAllString(text, "${1}••••••")
}

// ── Scanning messages between people ────────────────────────────────────────

var (
	reChatMoney  = re(`\b(send|wire|transfer|lend|loan|venmo|paypal|cash ?app|zelle|western union)\b.{0,40}(\b(money|cash|crypto|bitcoin|btc|usdt|gift ?cards?)\b|\$\s?\d+|\b\d+\s?(dollars|usd|bucks)\b)|\b(investment|trading) (platform|opportunity)\b|\bguaranteed (returns|profit)\b`)
	reChatThreat = re(`\bi('| wi)?ll (kill|hurt|ruin|expose|stalk) you\b|\bi('| wi)?ll find you and\b|\bi know where you (live|work)\b|\byou('| wi)?ll regret (this|it),? (bitch|whore)\b|\b(share|post|leak) your (pics|photos|nudes)\b`)
	reChatMove   = re(`\b(telegram|whatsapp|signal|wechat|kik)\b.{0,30}\b(me|chat|talk|move|add)\b|\b(move|talk|chat|add).{0,20}\b(telegram|whatsapp|signal|wechat|kik)\b`)
)

// Finding is what the scanner noticed in one message.
type Finding struct {
	Reason   string
	Severity string // normal | high | critical
	// Warn is what Suki quietly tells the recipient, or "".
	Warn string
}

// ScanChat checks a message from one person to another. It never blocks delivery or
// holds anyone by itself: a pattern can't tell a threat from a joke, so a threat or a
// money request goes to the Sentinel and to a human, who decides.
func ScanChat(text string) *Finding {
	text = Normalize(text)
	switch {
	case reChatThreat.MatchString(text):
		return &Finding{"threat", "high", "That message sounded like a threat. You don't owe anyone a reply. You can block them right here, and I've flagged it for our safety team."}
	case reChatMoney.MatchString(text):
		return &Finding{"money_request", "high", "Heads up: this looks like a money request. Never send money, crypto or gift cards to someone you met here. I've flagged it for our safety team."}
	case reChatMove.MatchString(text):
		return &Finding{"off_platform", "normal", "They want to move to another app. That's often innocent, and it's also how scams start. You can always stay here until you've met."}
	}
	return nil
}

// ── Checking what Suki says ─────────────────────────────────────────────────

var reLeak = re(`\b(compatib\w*|match) (score|rating|percent\w*)\b|\b\d{1,3}\s?%\s?(match|compatib\w*)\b|\b(your|their|his|her) (score|rating|ranking|reputation|elo)\b|\b(scored|rated|ranked) (you|them|him|her)\b|\b(you|they|he|she) (scored|rated|ranked)\b|\b\d(\.\d)?\s?(/|out of )\s?10\b|\bhidden\.\w+\b|\btool_call\b`)

// LeaksScore reports whether Suki's draft sounds like it reveals an evaluation.
// It catches more than it should, and the price is only one rewrite.
func LeaksScore(text string) bool { return reLeak.MatchString(strings.ReplaceAll(Normalize(text), "\n", " ")) }

// LeakFallback replaces a draft that still leaks after one rewrite.
const LeakFallback = "Feedback after dates privately shapes who I introduce you to, and those details stay private. The privacy policy explains more. Tell me more about what you're looking for, and I'll keep it in mind for your next introduction."

// Tips is the fixed date-safety advice.
const Tips = "A few habits worth keeping for a first date:\n" +
	"• Meet somewhere public, and get yourself there and back.\n" +
	"• Tell a friend where you'll be and who with. I can help you share the plan.\n" +
	"• Keep your own drink in sight, and stop whenever you want to.\n" +
	"• Don't send money to anyone you met here, for any reason.\n" +
	"• If something feels off, you can leave. You never owe anyone a reason."
