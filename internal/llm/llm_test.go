package llm

import "testing"

// Each host's hybrid models think unless told not to, and each wants its own switch.
func TestBodyTurnsThinkingOffPerHost(t *testing.T) {
	r := Request{Model: "m", Messages: []Message{{Role: "user", Content: "hi"}}}
	cases := []struct {
		base, key string
		want      any
	}{
		{"https://token-plan.cn-beijing.maas.aliyuncs.com/compatible-mode/v1", "enable_thinking", false},
		{"https://tokenhub.tencentmaas.com/v1", "reasoning_effort", "none"},
	}
	for _, c := range cases {
		b := New(c.base, "k").body(r, false)
		if b[c.key] != c.want {
			t.Errorf("%s: %s = %v, want %v", c.base, c.key, b[c.key], c.want)
		}
	}
	b := New("https://api.openai.com/v1", "k").body(r, false)
	if _, ok := b["enable_thinking"]; ok {
		t.Error("enable_thinking sent to a host that doesn't take it")
	}
	if _, ok := b["reasoning_effort"]; ok {
		t.Error("reasoning_effort sent to a host that doesn't take it")
	}
}
