"""Builds the app icon, adaptive icon, splash and notification glyph from
assets/suki/portrait.png. Run from mobile/: python scripts/generate_brand.py
Needs Pillow. The results are committed, so this only runs when the art changes."""
from pathlib import Path
from PIL import Image, ImageDraw, ImageFilter

ROOT = Path(__file__).resolve().parent.parent
OUT = ROOT / "assets" / "brand"
OUT.mkdir(parents=True, exist_ok=True)
portrait = Image.open(ROOT / "assets" / "suki" / "portrait.png").convert("RGB")

CYAN, LILAC, PINK = (139, 232, 245), (181, 156, 255), (246, 166, 222)


def lerp(a, b, t):
    return tuple(int(x + (y - x) * t) for x, y in zip(a, b))


def diagonal(size, stops):
    """A diagonal gradient through evenly spaced colour stops."""
    small = 256
    img = Image.new("RGB", (small, small))
    px = img.load()
    n = len(stops) - 1
    for y in range(small):
        for x in range(small):
            t = (x + y) / (2 * (small - 1)) * n
            i = min(int(t), n - 1)
            px[x, y] = lerp(stops[i], stops[i + 1], t - i)
    return img.resize((size, size), Image.BICUBIC)


def circle_mask(size):
    scale = 4
    m = Image.new("L", (size * scale, size * scale), 0)
    ImageDraw.Draw(m).ellipse((0, 0, size * scale - 1, size * scale - 1), fill=255)
    return m.resize((size, size), Image.LANCZOS)


def medallion(size, ring):
    """Suki's portrait in a circle, inside an iridescent ring, on transparency."""
    out = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    out.paste(diagonal(size, [CYAN, LILAC, PINK]).convert("RGBA"), (0, 0), circle_mask(size))
    inner = size - ring * 2
    face = portrait.resize((inner, inner), Image.LANCZOS).filter(ImageFilter.UnsharpMask(2, 60, 2))
    out.paste(face.convert("RGBA"), (ring, ring), circle_mask(inner))
    return out


FIELD = [(237, 229, 255), (205, 184, 255), (246, 200, 236)]

# iOS / default icon: opaque, medallion on a lilac-violet field.
icon = diagonal(1024, FIELD).convert("RGBA")
icon.alpha_composite(medallion(864, 26), (80, 80))
icon.convert("RGB").save(OUT / "icon.png")

# Android adaptive icon: 108dp canvas, 66dp always visible, so the circle sits in the safe zone.
fg = Image.new("RGBA", (1024, 1024), (0, 0, 0, 0))
fg.alpha_composite(medallion(640, 20), (192, 192))
fg.save(OUT / "adaptive-foreground.png")
diagonal(1024, FIELD).save(OUT / "adaptive-background.png")

# Splash: the medallion alone; the background colour comes from app.config.ts.
medallion(800, 22).save(OUT / "splash.png")


def cat(size, pad):
    """The cat hair clip from her sheet as a flat glyph, white on transparent."""
    scale = 4
    s = size * scale
    p = pad * scale
    img = Image.new("L", (s, s), 0)
    d = ImageDraw.Draw(img)
    w = s - 2 * p
    top = p + int(w * 0.20)
    d.ellipse((p, top, s - p, s - p), fill=255)
    d.polygon([(p + int(w * .03), p), (p + int(w * .44), top + int(w * .10)), (p + int(w * .03), top + int(w * .34))], fill=255)
    d.polygon([(s - p - int(w * .03), p), (s - p - int(w * .44), top + int(w * .10)), (s - p - int(w * .03), top + int(w * .34))], fill=255)
    eye = int(w * .06)
    ey = top + int(w * .34)
    for ex in (p + int(w * .32), s - p - int(w * .32)):
        d.ellipse((ex - eye, ey - eye, ex + eye, ey + eye), fill=0)
    alpha = img.resize((size, size), Image.LANCZOS)
    out = Image.new("RGBA", (size, size), (255, 255, 255, 0))
    out.putalpha(alpha)
    return out


cat(96, 8).save(OUT / "notification-icon.png")
mono = Image.new("RGBA", (1024, 1024), (0, 0, 0, 0))
mono.alpha_composite(cat(520, 0), (252, 252))
mono.save(OUT / "adaptive-monochrome.png")
print("done")
