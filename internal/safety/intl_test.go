package safety

import "testing"

func TestClassifyIntl(t *testing.T) {
	cases := map[string]Class{
		"我想自杀":                    SelfHarm,
		"我不想活了":                   SelfHarm,
		"我真的活不下去了":                SelfHarm,
		"죽고 싶어요":                  SelfHarm,
		"死にたい":                    SelfHarm,
		"no quiero vivir más":     SelfHarm,
		"他一直在跟踪我":                 InDanger,
		"我现在不安全":                  InDanger,
		"지금 안전하지 않아요":             InDanger,
		"助けて":                     InDanger,
		"estoy en peligro":        InDanger,
		"我才16岁":                   Underage,
		"我今年十五岁":                  Underage,
		"我是未成年":                   Underage,
		"我是高中生":                   Underage,
		"저는 16살이에요":               Underage,
		"私は16歳です":                 Underage,
		"tengo 16 años":           Underage,
		"soy menor de edad":       Underage,
		"那他给我打了多少分？":              ScoreAsk,
		"我的分数是多少":                 ScoreAsk,
		"내 점수 알려줘":                ScoreAsk,
		"私のスコアは？":                 ScoreAsk,
		"¿cuál es mi puntuación?": ScoreAsk,
		"他一直骚扰我":                  Harass,
		"계속 협박해요":                 Harass,
		"他让我转账给他":                 Scam,
		"他叫我转账":                   Scam,
		"me pidió dinero":         Scam,

		// Ordinary things people say.
		"想死你了":                         None,
		"我17岁的时候搬到了上海":                 None,
		"我17岁那年开始学钢琴":                  None,
		"我儿子今年16岁":                     None,
		"我喜欢吃火锅":                       None,
		"17살 때 서울로 이사했어요":              None,
		"tengo 15 años de experiencia": None,
		"我今年二十八岁":                      None,
		"我今年28岁":                       None,
	}
	for in, want := range cases {
		if got := Classify(in); got != want {
			t.Errorf("%q: got %q, want %q", in, got, want)
		}
	}
}

func TestLeaksScoreIntl(t *testing.T) {
	for _, s := range []string{"你们的匹配度是87", "他的评分很高", "90%匹配", "궁합 점수는 80이에요", "あなたのスコアは高いです", "tu puntuación es alta"} {
		if !LeaksScore(s) {
			t.Errorf("%q should count as a leak", s)
		}
	}
	for _, s := range []string{"约会后的反馈是私密的，不会显示具体分数", "你们都喜欢爵士乐", "今天天气很好"} {
		if LeaksScore(s) {
			t.Errorf("%q isn't a leak", s)
		}
	}
}
