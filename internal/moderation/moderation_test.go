package moderation

import (
	"context"
	"testing"
)

// With no model, only the word list runs. It must catch the obvious, and must not
// catch the ordinary language of dating.
func TestWordList(t *testing.T) {
	f := &Filter{}
	ctx := context.Background()
	for _, s := range []string{"send nudes", "wanna fuck tonight", "check my onlyfans", "love to sext", "裸照发我"} {
		if v := f.Check(ctx, "", Chat, s); !v.Blocked {
			t.Errorf("should block: %q", s)
		}
	}
	for _, s := range []string{
		"you're gorgeous", "I'd love to kiss you", "can't wait to see you", "looking for something casual",
		"summa cum laude", "My kid took these pics at the zoo", "send me pics of your dog", "I teach in Essex", "my cat Dickens", "a cocktail at Scunthorpe", "I love analysis", "Sussex is lovely", "assassin's creed", "the class was hard",
	} {
		if v := f.Check(ctx, "", Chat, s); v.Blocked {
			t.Errorf("should allow: %q (%s)", s, v.Category)
		}
	}
}

func TestMinorsAreFlaggedAsSuch(t *testing.T) {
	if v := (&Filter{}).Check(context.Background(), "", Suki, "underage nudes"); !v.Minors() {
		t.Fatalf("got %+v", v)
	}
}

// Told privately to Suki with no model available, she declines rather than guessing.
func TestSukiChannelFailsPolitely(t *testing.T) {
	if v := (&Filter{}).Check(context.Background(), "", Suki, "he sent me dick pics"); !v.Blocked {
		t.Fatal("should decline without a model")
	}
	if v := (&Filter{}).Check(context.Background(), "", Suki, "tell me about jazz bars"); v.Blocked {
		t.Fatal("ordinary text blocked")
	}
}

func TestOutgoingCheck(t *testing.T) {
	if !Explicit("send me your nudes") {
		t.Fatal("agent output must be caught")
	}
	if Explicit("You both love live jazz") {
		t.Fatal("clean copy caught")
	}
}
