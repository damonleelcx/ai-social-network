package httpapi

import (
	"bytes"
	"image/jpeg"
	"os"
	"testing"
)

func TestInvitePoster(t *testing.T) {
	for _, lang := range []string{"zh", "en", "ko", "ja", "es", "fr"} {
		raw, err := renderPoster(lang, "https://suki.zhiyuwujie.top/?ref=ANDGQX3Z", "ANDGQX3Z")
		if err != nil {
			t.Fatalf("%s: %v", lang, err)
		}
		img, err := jpeg.Decode(bytes.NewReader(raw))
		if err != nil {
			t.Fatalf("%s: %v", lang, err)
		}
		if b := img.Bounds(); b.Dx() != 1080 || b.Dy() != 1920 {
			t.Fatalf("%s: size %v", lang, b)
		}
		// The QR code is drawn: its corner finder pattern is dark.
		if r, g, b, _ := img.At(posterQRX+10, posterQRY+10).RGBA(); r > 0x6000 || g > 0x6000 || b > 0x8000 {
			t.Fatalf("%s: no QR code at the top left of its box", lang)
		}
		if out := os.Getenv("POSTER_OUT"); out != "" && lang == "zh" {
			_ = os.WriteFile(out, raw, 0o644)
		}
	}
}
