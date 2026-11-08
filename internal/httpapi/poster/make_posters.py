"""Draws the invite poster backgrounds, one per language: bg-<lang>.jpg in this folder.

The server adds each person's QR code and invite code into the white panel (see poster.go,
whose layout constants must match QR_* and CODE_Y here). Run from the repo root on Windows,
where the fonts below are installed:  python internal/httpapi/poster/make_posters.py
"""
import os
from PIL import Image, ImageDraw, ImageFilter, ImageFont

HERE = os.path.dirname(os.path.abspath(__file__))
ROOT = os.path.abspath(os.path.join(HERE, "..", "..", ".."))
W, H = 1080, 1920
PANEL = (190, 1060, 890, 1800)  # the white panel
QR_X, QR_Y, QR_SIZE = 300, 1150, 480  # where the server draws the QR code
CODE_Y = 1700  # centre line of the invite code, drawn by the server

FONTS = r"C:\Windows\Fonts"
FACES = {
    "zh": ("msyhbd.ttc", "msyh.ttc"),
    "en": ("msyhbd.ttc", "msyh.ttc"),
    "es": ("msyhbd.ttc", "msyh.ttc"),
    "ko": ("malgunbd.ttf", "malgun.ttf"),
    "ja": ("YuGothB.ttc", "YuGothM.ttc"),
}
# Words for finding company, on purpose: the same poster is used on the China site.
COPY = {
    "zh": ("和 Suki 一起找搭子", "看电影、吃饭、旅行、运动，\n告诉 Suki 你想做什么，她帮你找到合拍的人。", "微信扫码，或用浏览器打开"),
    "en": ("Find your people with Suki", "Movies, meals, trips, sports.\nTell Suki what you want to do; she finds people who fit.", "Scan to join"),
    "ko": ("Suki와 함께 메이트 찾기", "영화, 식사, 여행, 운동.\n하고 싶은 걸 말하면 Suki가 잘 맞는 사람을 찾아줘요.", "스캔해서 시작하기"),
    "ja": ("Suki と一緒に仲間さがし", "映画、ごはん、旅行、スポーツ。\nやりたいことを伝えれば、Suki が気の合う人を見つけます。", "スキャンして参加"),
    "es": ("Encuentra tu gente con Suki", "Cine, comidas, viajes, deporte.\nDile a Suki qué quieres hacer y ella encuentra a quien encaje.", "Escanea para unirte"),
}


def font(name, size):
    return ImageFont.truetype(os.path.join(FONTS, name), size)


def gradient():
    top, mid, bottom = (20, 10, 44), (52, 24, 110), (122, 40, 120)
    img = Image.new("RGB", (W, H))
    px = img.load()
    for y in range(H):
        t = y / (H - 1)
        a, b, u = (top, mid, t / 0.55) if t < 0.55 else (mid, bottom, (t - 0.55) / 0.45)
        c = tuple(int(a[i] + (b[i] - a[i]) * u) for i in range(3))
        for x in range(W):
            px[x, y] = c
    return img


def portrait(size):
    face = Image.open(os.path.join(ROOT, "mobile", "assets", "suki", "playful.png")).convert("RGBA").resize((size, size), Image.LANCZOS)
    mask = Image.new("L", (size, size), 0)
    ImageDraw.Draw(mask).ellipse((0, 0, size - 1, size - 1), fill=255)
    face.putalpha(mask)
    return face


def centred(draw, y, text, f, fill, spacing=10):
    for i, line in enumerate(text.split("\n")):
        w = draw.textlength(line, font=f)
        draw.text(((W - w) / 2, y + i * (f.size + spacing)), line, font=f, fill=fill)


def make(lang):
    bold, regular = FACES[lang]
    head, sub, scan = COPY[lang]
    img = gradient()
    # A soft pink glow behind Suki.
    glow = Image.new("RGBA", (W, H), (0, 0, 0, 0))
    ImageDraw.Draw(glow).ellipse((190, 120, 890, 820), fill=(255, 92, 138, 120))
    img = Image.alpha_composite(img.convert("RGBA"), glow.filter(ImageFilter.GaussianBlur(90)))
    d = ImageDraw.Draw(img)
    # Ring, then Suki.
    d.ellipse((290, 150, 790, 650), fill=(255, 255, 255, 255))
    d.ellipse((298, 158, 782, 642), fill=(255, 92, 138, 255))
    img.alpha_composite(portrait(470), (305, 165))
    d = ImageDraw.Draw(img)
    hf = font(bold, 64 if lang in ("zh", "ko", "ja") else 60)
    sf = font(regular, 34)
    centred(d, 720, head, hf, (255, 255, 255))
    centred(d, 830, sub, sf, (221, 208, 255), spacing=14)
    # The panel the QR code and invite code go in.
    d.rounded_rectangle(PANEL, radius=48, fill=(255, 255, 255))
    lf = font(regular, 30)
    w = d.textlength(scan, font=lf)
    d.text(((W - w) / 2, 1085), scan, font=lf, fill=(120, 104, 160))
    brand = font(bold, 30)
    w = d.textlength("SynAgent · Suki", font=brand)
    d.text(((W - w) / 2, 1835), "SynAgent · Suki", font=brand, fill=(221, 208, 255))
    img.convert("RGB").save(os.path.join(HERE, f"bg-{lang}.jpg"), quality=86, optimize=True, progressive=True)


for lang in COPY:
    make(lang)
    print("drew", lang)
