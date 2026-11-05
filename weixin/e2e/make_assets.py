"""Makes the mini program's images from the app's. Run from the repo root: python weixin/e2e/make_assets.py

weixin/assets/suki: Suki's faces, small enough for the 2 MB package limit.
weixin/assets/tabs: tab bar icons at 81x81, each under WeChat's 40 KB limit.
"""
import math
import os

from PIL import Image, ImageChops, ImageDraw

FACES = ["portrait", "happy", "surprised", "thinking", "playful", "pouty", "chibi"]
SRC = "mobile/assets/suki"
FACE_OUT = "weixin/assets/suki"
TAB_OUT = "weixin/assets/tabs"
FACE_SIZE = 192
TAB = 81
S = 8  # Drawn large and scaled down, which smooths the edges.
BIG = TAB * S

MUTED = (169, 159, 196, 255)  # The tab bar's text color.
WHITE = (255, 255, 255, 255)
BRAND = ((122, 69, 245), (212, 92, 190))
IRIS = ((139, 232, 245), (181, 156, 255), (246, 166, 222))


def gradient(size, stops):
    """A diagonal gradient through the stops."""
    img = Image.new("RGB", (size, size))
    px = img.load()
    for y in range(size):
        for x in range(size):
            t = (x + y) / (2 * (size - 1)) * (len(stops) - 1)
            i = min(int(t), len(stops) - 2)
            f = t - i
            px[x, y] = tuple(round(stops[i][c] + (stops[i + 1][c] - stops[i][c]) * f) for c in range(3))
    return img


def circle_mask(size, inset=0):
    mask = Image.new("L", (size, size), 0)
    ImageDraw.Draw(mask).ellipse((inset, inset, size - 1 - inset, size - 1 - inset), fill=255)
    return mask


def canvas():
    return Image.new("RGBA", (BIG, BIG), (0, 0, 0, 0))


def heart(d, cx, cy, r, fill=None, outline=None, width=0):
    pts = []
    for i in range(240):
        t = 2 * math.pi * i / 240
        x = 16 * math.sin(t) ** 3
        y = 13 * math.cos(t) - 5 * math.cos(2 * t) - 2 * math.cos(3 * t) - math.cos(4 * t)
        pts.append((cx + x * r / 16, cy - y * r / 16))
    if fill:
        d.polygon(pts, fill=fill)
    if outline:
        d.line(pts + pts[:2], fill=outline, width=width, joint="curve")


def calendar(d, cx, cy, r, color, solid):
    w = max(1, round(r * 0.17))
    box = (cx - r, cy - r * 0.8, cx + r, cy + r)
    head = cy - r * 0.28
    if solid:
        d.rounded_rectangle(box, radius=r * 0.24, fill=color)
        d.rectangle((box[0] + w, head, box[2] - w, head + w * 0.8), fill=(0, 0, 0, 0))
    else:
        d.rounded_rectangle(box, radius=r * 0.24, outline=color, width=w)
        d.line((box[0], head, box[2], head), fill=color, width=w)
    for x in (cx - r * 0.5, cx + r * 0.5):
        d.line((x, cy - r * 1.08, x, cy - r * 0.55), fill=color, width=w)
    if not solid:
        for x in (cx - r * 0.45, cx, cx + r * 0.45):
            for y in (cy + r * 0.12, cy + r * 0.58):
                dot = r * 0.1
                d.ellipse((x - dot, y - dot, x + dot, y + dot), fill=color)


def person(d, cx, cy, r, color, solid):
    w = round(r * 0.17)
    head = r * 0.46
    head_box = (cx - head, cy - r, cx + head, cy - r + head * 2)
    body = (cx - r * 0.95, cy + r * 0.02, cx + r * 0.95, cy + r * 2)
    if solid:
        d.ellipse(head_box, fill=color)
        d.pieslice(body, 180, 360, fill=color)
    else:
        d.ellipse(head_box, outline=color, width=w)
        d.arc(body, 180, 360, fill=color, width=w)
        d.line((body[0] + w / 2, cy + r, body[2] - w / 2, cy + r), fill=color, width=w)


def on_badge(draw_glyph):
    """Selected: a white glyph on the brand gradient, as in the app."""
    img = gradient(BIG, BRAND).convert("RGBA")
    img.putalpha(circle_mask(BIG))
    glyph = canvas()
    draw_glyph(ImageDraw.Draw(glyph), BIG / 2, BIG / 2, BIG * 0.27, WHITE, True)
    img.alpha_composite(glyph)
    return img


def off_icon(draw_glyph):
    img = canvas()
    draw_glyph(ImageDraw.Draw(img), BIG / 2, BIG / 2, BIG * 0.36, MUTED, False)
    return img


def heart_glyph(d, cx, cy, r, color, solid):
    if solid:
        heart(d, cx, cy, r, fill=color)
    else:
        heart(d, cx, cy, r, outline=color, width=round(r * 0.17))


def suki_tab(on):
    """Her face in the iridescent ring. It's dimmed until selected."""
    ring = gradient(BIG, IRIS).convert("RGBA")
    ring.putalpha(circle_mask(BIG))
    face = Image.open(f"{SRC}/portrait.png").convert("RGB").resize((BIG, BIG), Image.LANCZOS).convert("RGBA")
    if not on:
        face = Image.blend(face, Image.new("RGBA", face.size, (27, 16, 51, 255)), 0.3)
        ring = Image.blend(ring, Image.new("RGBA", ring.size, (27, 16, 51, 255)), 0.45)
        ring.putalpha(circle_mask(BIG))
    inset = BIG // (10 if on else 14)
    img = canvas()
    img.alpha_composite(ring)
    face.putalpha(circle_mask(BIG, inset))
    img.alpha_composite(face)
    return img


def save(img, name):
    path = f"{TAB_OUT}/{name}.png"
    img.resize((TAB, TAB), Image.LANCZOS).save(path, optimize=True)
    size = os.path.getsize(path)
    assert size < 40 * 1024, f"{name} is too big for a tab icon: {size}"
    print(f"{path} {size}")


def main():
    os.makedirs(FACE_OUT, exist_ok=True)
    os.makedirs(TAB_OUT, exist_ok=True)
    for face in FACES:
        img = Image.open(f"{SRC}/{face}.png").convert("RGB").resize((FACE_SIZE, FACE_SIZE), Image.LANCZOS)
        path = f"{FACE_OUT}/{face}.jpg"
        img.save(path, quality=84, optimize=True)
        print(f"{path} {os.path.getsize(path)}")
    save(suki_tab(False), "suki")
    save(suki_tab(True), "suki-on")
    for name, glyph in (("matches", heart_glyph), ("dates", calendar), ("me", person)):
        save(off_icon(glyph), name)
        save(on_badge(glyph), f"{name}-on")


if __name__ == "__main__":
    main()
