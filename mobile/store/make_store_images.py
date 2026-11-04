"""Builds Play Store and App Store images from real screenshots of the app.

    python make_store_images.py <folder of raw screenshots> <output folder>

Sizes: Play phone screenshots 1080x1920 and feature graphic 1024x500, icon 512x512.
App Store iPhone 6.9-inch 1320x2868 and 6.5-inch 1284x2778.
"""
import math
import os
import sys
from PIL import Image, ImageDraw, ImageFilter, ImageFont

HERE = os.path.dirname(os.path.abspath(__file__))
FONT_DIR = os.path.join(HERE, "fonts")
BOLD = os.path.join(FONT_DIR, "Poppins-Bold.ttf")
SEMI = os.path.join(FONT_DIR, "Poppins-Medium.ttf")
SUKI = os.path.join(HERE, "..", "assets", "suki")

INK = (42, 27, 89)
DEEP = [(38, 22, 86), (86, 42, 150), (168, 70, 170)]
LILAC = [(244, 238, 255), (226, 214, 255), (250, 226, 244)]

# Screenshot, headline, sub-line, Suki's face, background.
SLIDES = [
    ("02_onboard", "Meet Suki,\nyour AI winggirl", "Set up your profile just by talking", "playful", "deep"),
    ("05_suki_found", "Introductions,\nnot endless swiping", "People in your city she thinks you'd like", "happy", "light"),
    ("08_coach", "Never stuck\nfor words", "Private hints that only you can see", "thinking", "deep"),
    ("10_venues", "A date spot\nyou'd both love", "Real places, picked around what you share", "surprised", "light"),
    ("11_confirmed", "Plans made,\nsafety built in", "Share your plans with a friend in one tap", "happy", "deep"),
    ("16_memory", "She learns\nyour type", "Read or delete anything she remembers", "playful", "light"),
    ("15_suki_settings", "Make Suki\nyours", "Her tone, her voice, how often she checks in", "pouty", "deep"),
]

SIZES = {
    "play/phone": (1080, 1920),
    "appstore/iphone-6.9": (1320, 2868),
    "appstore/iphone-6.5": (1284, 2778),
}


def gradient(w, h, stops, angle=None):
    """A diagonal blend, with a soft iridescent glow like the jacket on her character sheet."""
    small = Image.new("RGB", (max(2, w // 8), max(2, h // 8)))
    px = small.load()
    sw, sh = small.size
    for y in range(sh):
        for x in range(sw):
            t = (x / sw * 0.45 + y / sh * 0.55) * (len(stops) - 1)
            i = min(int(t), len(stops) - 2)
            f = t - i
            px[x, y] = tuple(int(stops[i][c] + (stops[i + 1][c] - stops[i][c]) * f) for c in range(3))
    return small.resize((w, h), Image.BICUBIC)


def glow(img, cx, cy, r, color, alpha=120):
    layer = Image.new("RGBA", img.size, (0, 0, 0, 0))
    ImageDraw.Draw(layer).ellipse((cx - r, cy - r, cx + r, cy + r), fill=color + (alpha,))
    layer = layer.filter(ImageFilter.GaussianBlur(r / 2))
    img.paste(layer, (0, 0), layer)


def rounded(img, radius):
    mask = Image.new("L", (img.width * 2, img.height * 2), 0)
    ImageDraw.Draw(mask).rounded_rectangle((0, 0, mask.width - 1, mask.height - 1), radius * 2, fill=255)
    out = img.convert("RGBA")
    out.putalpha(mask.resize(img.size, Image.LANCZOS))
    return out


def phone(shot, width):
    """A slim device around the screenshot, with the Android status and nav bars cropped off
    so the same frame works for both stores."""
    top, bottom = 104, 46
    screen = shot.crop((0, top, shot.width, shot.height - bottom)).convert("RGB")
    h = int(screen.height * width / screen.width)
    screen = screen.resize((width, h), Image.LANCZOS)
    pad = int(width * 0.022)
    r = int(width * 0.085)
    body = Image.new("RGBA", (width + pad * 2, h + pad * 2), (0, 0, 0, 0))
    d = ImageDraw.Draw(body)
    d.rounded_rectangle((0, 0, body.width - 1, body.height - 1), r + pad, fill=(22, 14, 44, 255))
    d.rounded_rectangle((2, 2, body.width - 3, body.height - 3), r + pad - 2, outline=(120, 100, 190, 255), width=2)
    s = rounded(screen, r)
    body.paste(s, (pad, pad), s)
    return body


def shadowed(canvas, item, x, y, blur=40, alpha=110):
    sh = Image.new("RGBA", canvas.size, (0, 0, 0, 0))
    m = Image.new("RGBA", item.size, (20, 8, 60, alpha))
    m.putalpha(item.split()[3].point(lambda v: min(v, alpha)))
    sh.paste(m, (x, y + blur // 2), m)
    sh = sh.filter(ImageFilter.GaussianBlur(blur))
    canvas.paste(sh, (0, 0), sh)
    canvas.paste(item, (x, y), item)


def face(mood, size):
    img = Image.open(os.path.join(SUKI, mood + ".png")).convert("RGB").resize((size * 2, size * 2), Image.LANCZOS)
    mask = Image.new("L", img.size, 0)
    ImageDraw.Draw(mask).ellipse((0, 0, img.width - 1, img.height - 1), fill=255)
    ring = Image.new("RGBA", (size * 2 + 24, size * 2 + 24), (0, 0, 0, 0))
    ImageDraw.Draw(ring).ellipse((0, 0, ring.width - 1, ring.height - 1), fill=(255, 255, 255, 255))
    ring.paste(img, (12, 12), mask)
    return ring.resize((size + 12, size + 12), Image.LANCZOS)


def centered(draw, y, text, font, fill, width, spacing):
    for line in text.split("\n"):
        w = draw.textlength(line, font=font)
        draw.text(((width - w) / 2, y), line, font=font, fill=fill)
        y += font.size + spacing
    return y


def slide(shot_path, head, sub, mood, bg, size):
    W, H = size
    k = W / 1080
    dark = bg == "deep"
    img = gradient(W, H, DEEP if dark else LILAC).convert("RGBA")
    glow(img, int(W * 0.85), int(H * 0.08), int(W * 0.5), (120, 210, 255) if dark else (160, 220, 255), 90)
    glow(img, int(W * 0.1), int(H * 0.55), int(W * 0.55), (255, 120, 210), 80 if dark else 60)
    d = ImageDraw.Draw(img)
    fg = (255, 255, 255) if dark else INK
    fg2 = (232, 222, 255) if dark else (84, 62, 140)
    hf = ImageFont.truetype(BOLD, int(92 * k))
    sf = ImageFont.truetype(SEMI, int(36 * k))
    y = int(H * 0.036)
    y2 = centered(d, y, head, hf, fg, W, int(24 * k))
    y3 = centered(d, y2 + int(8 * k), sub, sf, fg2, W, 0)
    shot = Image.open(shot_path)
    avail = H - y3 - int(46 * k)
    pw = int(W * 0.76)
    p = phone(shot, pw)
    px = (W - p.width) // 2
    py = y3 + int(56 * k)
    # The phone runs off the bottom edge, so the top of the screen gets all the attention.
    shadowed(img, p, px, py)
    return img.convert("RGB")


def feature(out):
    W, H = 1024, 500
    img = gradient(W, H, DEEP).convert("RGBA")
    glow(img, 860, 60, 300, (120, 210, 255), 90)
    glow(img, 120, 460, 300, (255, 120, 210), 90)
    d = ImageDraw.Draw(img)
    d.text((56, 112), "SynAgent", font=ImageFont.truetype(BOLD, 84), fill=(255, 255, 255))
    d.text((60, 226), "Meet Suki, your AI winggirl.", font=ImageFont.truetype(SEMI, 31), fill=(236, 226, 255))
    d.text((60, 276), "Good introductions,", font=ImageFont.truetype(SEMI, 27), fill=(214, 200, 250))
    d.text((60, 314), "no endless swiping.", font=ImageFont.truetype(SEMI, 27), fill=(214, 200, 250))
    p = Image.open(os.path.join(SUKI, "portrait.png")).convert("RGB").resize((360, 360), Image.LANCZOS)
    m = Image.new("L", (720, 720), 0)
    ImageDraw.Draw(m).ellipse((0, 0, 719, 719), fill=255)
    m = m.resize((360, 360), Image.LANCZOS)
    ring = Image.new("RGBA", (380, 380), (0, 0, 0, 0))
    ImageDraw.Draw(ring).ellipse((0, 0, 379, 379), fill=(255, 255, 255, 235))
    ring.paste(p, (10, 10), m)
    shadowed(img, ring, 610, 60, blur=24, alpha=100)
    os.makedirs(os.path.join(out, "play"), exist_ok=True)
    img.convert("RGB").save(os.path.join(out, "play", "feature-graphic-1024x500.png"))


def main():
    raw, out = sys.argv[1], sys.argv[2]
    for folder, size in SIZES.items():
        dest = os.path.join(out, folder)
        os.makedirs(dest, exist_ok=True)
        for i, (name, head, sub, mood, bg) in enumerate(SLIDES, 1):
            slide(os.path.join(raw, name + ".png"), head, sub, mood, bg, size).save(os.path.join(dest, f"{i:02d}-{name.split('_', 1)[1]}.png"))
    feature(out)
    icon = Image.open(os.path.join(HERE, "..", "assets", "brand", "icon.png")).convert("RGB")
    icon.resize((512, 512), Image.LANCZOS).save(os.path.join(out, "play", "icon-512.png"))
    icon.resize((1024, 1024), Image.LANCZOS).save(os.path.join(out, "appstore", "icon-1024.png"))
    print("done")


if __name__ == "__main__":
    main()
