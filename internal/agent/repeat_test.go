package agent

import (
	"testing"

	"github.com/damonleelcx/ai-social-network/internal/llm"
)

// The nudges from the screenshots that kept coming back, day after day.
func TestRepeatsCatchesTheSameNudge(t *testing.T) {
	sent := []llm.Message{
		{Role: "assistant", Content: "Daphne之前问你在深圳探索过最酷的地方，你还没回呢！要不要聊聊这个？🤔"},
		{Role: "assistant", Content: "Damon，你和Daphne匹配成功啦！要不要聊聊你在深圳发现的最酷角落？🤔"},
		{Role: "assistant", Content: "Damon，你和Daphne正在规划见面呢！要不要现在给她发条消息聊聊周末安排？😊"},
	}
	cases := []struct {
		draft  string
		repeat bool
	}{
		{"Damon，你和Daphne正在规划见面呢！要不要现在给她发条消息聊聊周末安排？😊\n<<mood=happy|replies=发消息;稍后再说>>", true},
		{"Damon, 你和 Daphne 正在规划见面呢 —— 要不要现在给她发条消息，聊聊周末安排?", true},
		{"Damon，你和Daphne匹配成功啦！要不要现在给她发条消息打个招呼？😊", true},
		{"周末想不想找人一起打德州扑克？我在搭子广场看到一个周六晚上的局，要看看吗？", false},
		{"你上次说喜欢可爱的女生，要不要给资料加一张近照？照片多的人回复率更高。", false},
		{"Hey Damon, want to plan something with Daphne this weekend?", false},
	}
	for _, c := range cases {
		if got := repeats(c.draft, sent) != ""; got != c.repeat {
			t.Errorf("repeats(%q) = %v, want %v", c.draft, got, c.repeat)
		}
	}
}

// Only her own recent finished messages count: not theirs, and not tool calls.
func TestRepeatsLooksOnlyAtHerRecentMessages(t *testing.T) {
	same := "要不要现在给她发条消息聊聊周末安排？"
	if repeats(same, []llm.Message{{Role: "user", Content: same}}) != "" {
		t.Error("their own words counted as her repeat")
	}
	old := []llm.Message{{Role: "assistant", Content: same}}
	for i := 0; i < repeatWindow; i++ {
		old = append(old, llm.Message{Role: "assistant", Content: "完全不同的一句话，编号" + string(rune('A'+i))})
	}
	if repeats(same, old) != "" {
		t.Error("a message older than the window counted")
	}
	if repeats("", old) != "" {
		t.Error("an empty draft counted as a repeat")
	}
}
