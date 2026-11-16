package agent

import (
	"strings"
	"unicode"

	"github.com/damonleelcx/ai-social-network/internal/llm"
)

// repeatWindow is how many of her recent messages a new unprompted one is compared with.
const repeatWindow = 8

// repeatSimilarity is how alike two messages may be before the new one counts as saying the
// same thing again. Character pairs work the same for Chinese, which has no spaces, and English.
const repeatSimilarity = 0.6

// repeats returns the earlier message of hers that a draft says again, or "". A model told
// "don't repeat yourself" still sends the same nudge every day, so this is checked by code.
func repeats(draft string, msgs []llm.Message) string {
	body, _, _ := parseReply(draft)
	now := bigrams(body)
	if len(now) == 0 {
		return ""
	}
	seen := 0
	for i := len(msgs) - 1; i >= 0 && seen < repeatWindow; i-- {
		if msgs[i].Role != "assistant" || msgs[i].Content == "" || len(msgs[i].ToolCalls) > 0 {
			continue
		}
		seen++
		if dice(now, bigrams(msgs[i].Content)) >= repeatSimilarity {
			return msgs[i].Content
		}
	}
	return ""
}

// bigrams counts adjacent pairs of letters and digits, ignoring case, spaces, punctuation and emoji.
func bigrams(s string) map[string]int {
	var rs []rune
	for _, r := range strings.ToLower(s) {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			rs = append(rs, r)
		}
	}
	out := map[string]int{}
	for i := 0; i+1 < len(rs); i++ {
		out[string(rs[i:i+2])]++
	}
	return out
}

// dice is the Sørensen–Dice coefficient of two bigram multisets: 1 when they're the same.
func dice(a, b map[string]int) float64 {
	na, nb, both := 0, 0, 0
	for k, n := range a {
		na += n
		both += min(n, b[k])
	}
	for _, n := range b {
		nb += n
	}
	if na+nb == 0 {
		return 0
	}
	return 2 * float64(both) / float64(na+nb)
}
