package safety

import "testing"

func TestClassify(t *testing.T) {
	cases := map[string]Class{
		"I want to kill myself":                   SelfHarm,
		"I don't want to be here anymore":         SelfHarm,
		"死にたい":                                    SelfHarm,
		"I don't feel safe right now":             InDanger,
		"he's following me":                       InDanger,
		"I\u2019m 16":                             Underage,
		"I\u2019m sixteen":                        Underage,
		"I don\u2019t want to be here anymore":    SelfHarm,
		"i wanna die":                             SelfHarm,
		"I'm 5 years sober":                       None,
		"I'm 16":                                  Underage,
		"I am 15 years old":                       Underage,
		"i'm in high school":                      Underage,
		"what's my score?":                        ScoreAsk,
		"how did Sam rate me":                     ScoreAsk,
		"what did they say about me":              ScoreAsk,
		"they keep threatening me":                Harass,
		"she asked me for money":                  Scam,
		// Must not trigger. Getting these wrong would hold real adults' accounts.
		"I'm 5'10":                                None,
		"I'm 15 minutes away":                     None,
		"I'm 28 and love jazz":                    None,
		"I feel safe right now":                   None,
		"I went to high school in Ohio":           None,
		"I had a great time last night":           None,
		"what's your favorite date spot":          None,
		"I scored a table at the jazz club":       None,
	}
	for text, want := range cases {
		if got := Classify(text); got != want {
			t.Errorf("%q: got %q, want %q", text, got, want)
		}
	}
}

func TestMaskSecrets(t *testing.T) {
	if got := MaskSecrets("my password is hunter2!"); got == "my password is hunter2!" {
		t.Fatal("password survived")
	}
	if got := MaskSecrets("I forgot my passport"); got != "I forgot my passport" {
		t.Fatalf("over-masked: %s", got)
	}
}

func TestLeaksScore(t *testing.T) {
	for _, s := range []string{"Your rating is great", "You're an 87% match", "they rated you highly", "I'd give them 8/10", "Their reputation is low", "compatibility score: 80"} {
		if !LeaksScore(s) {
			t.Errorf("should be caught: %s", s)
		}
	}
	for _, s := range []string{"You both love jazz", "Saturday at 8 works", "I don't share how people are evaluated"} {
		if LeaksScore(s) {
			t.Errorf("clean text caught: %s", s)
		}
	}
}

func TestScanChat(t *testing.T) {
	if f := ScanChat("can you send me $200 for rent"); f == nil || f.Severity != "high" {
		t.Errorf("money: %+v", f)
	}
	if f := ScanChat("I know where you live"); f == nil || f.Severity != "high" {
		t.Errorf("threat: %+v", f)
	}
	if f := ScanChat("add me on telegram"); f == nil || f.Severity != "normal" {
		t.Errorf("move: %+v", f)
	}
	for _, s := range []string{"I'll find you at the bar", "You'll regret skipping dessert", "want to grab coffee Saturday?"} {
		if f := ScanChat(s); f != nil {
			t.Errorf("innocent message flagged: %q %+v", s, f)
		}
	}
	if f := ScanChat("I\u2019ll kill you"); f == nil {
		t.Error("a curly apostrophe hid a threat")
	}
	if f := ScanChat("see you Saturday"); f != nil {
		t.Errorf("innocent message flagged: %+v", f)
	}
}
