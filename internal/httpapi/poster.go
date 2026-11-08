package httpapi

import (
	"bytes"
	"embed"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"net/http"
	"strings"
	"sync"

	qrcode "github.com/skip2/go-qrcode"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// The invite poster: a picture someone can post to WeChat Moments or anywhere else, with Suki on
// it and their own QR code and invite code. The backgrounds are drawn ahead of time, one per
// language (poster/make_posters.py); the QR code and the code are added here. It never says
// anything about a reward: WeChat doesn't allow offering one for sharing.

//go:embed poster/*.jpg
var posterFiles embed.FS

// Where poster/make_posters.py left room. Keep the two in step.
const (
	posterQRX, posterQRY, posterQRSize = 300, 1150, 480
	posterCodeY                        = 1700
)

var (
	posterFaceOnce sync.Once
	posterFace     font.Face
)

func codeFace() font.Face {
	posterFaceOnce.Do(func() {
		f, err := opentype.Parse(gobold.TTF)
		if err == nil {
			posterFace, _ = opentype.NewFace(f, &opentype.FaceOptions{Size: 64, DPI: 72, Hinting: font.HintingFull})
		}
	})
	return posterFace
}

// renderPoster draws one person's poster in lang (English when there's no background for it).
func renderPoster(lang, link, code string) ([]byte, error) {
	raw, err := posterFiles.ReadFile("poster/bg-" + lang + ".jpg")
	if err != nil {
		if raw, err = posterFiles.ReadFile("poster/bg-en.jpg"); err != nil {
			return nil, err
		}
	}
	bg, err := jpeg.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	canvas := image.NewRGBA(bg.Bounds())
	draw.Draw(canvas, canvas.Bounds(), bg, image.Point{}, draw.Src)

	qr, err := qrcode.New(link, qrcode.Medium)
	if err != nil {
		return nil, err
	}
	qr.DisableBorder = true
	qr.ForegroundColor = color.RGBA{0x1b, 0x10, 0x33, 0xff}
	q := qr.Image(posterQRSize)
	draw.Draw(canvas, image.Rect(posterQRX, posterQRY, posterQRX+posterQRSize, posterQRY+posterQRSize), q, image.Point{}, draw.Src)

	if face := codeFace(); face != nil {
		text := strings.Join(strings.Split(strings.ToUpper(code), ""), " ")
		d := &font.Drawer{Dst: canvas, Src: image.NewUniform(color.RGBA{0x5a, 0x2b, 0xe0, 0xff}), Face: face}
		w := d.MeasureString(text)
		d.Dot = fixed.Point26_6{X: (fixed.I(canvas.Bounds().Dx()) - w) / 2, Y: fixed.I(posterCodeY + 22)}
		d.DrawString(text)
	}

	var out bytes.Buffer
	if err := jpeg.Encode(&out, canvas, &jpeg.Options{Quality: 90}); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// referralPoster is this person's invite poster, in the language they ask for.
func (s *Server) referralPoster(w http.ResponseWriter, r *http.Request) {
	st, err := s.Billing.ReferralStatus(r.Context(), userOf(r).ID)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	lang := r.URL.Query().Get("lang")
	if lang == "" {
		lang = "en"
	}
	img, err := renderPoster(lang, st.Link, st.Code)
	if err != nil {
		handleErr(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "private, max-age=3600")
	_, _ = w.Write(img)
}
