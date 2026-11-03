"""Seeds two fictional people on the runner's throwaway stack, then plays Leo while Mia is driven
through the app on the simulator: he says yes, answers messages, votes, confirms the time, and
moves the date into the past so the debrief opens. Runs for a bounded time.

    python3 peer.py <server log> <dir for logs> [minutes]
"""
import io, json, os, re, subprocess, sys, time, urllib.error, urllib.request, uuid
from PIL import Image, ImageDraw

API = "http://127.0.0.1:8080"
LOG, OUT = sys.argv[1], sys.argv[2]
MINUTES = int(sys.argv[3]) if len(sys.argv) > 3 else 60
PW = "demo password 123"
op = urllib.request.build_opener(urllib.request.ProxyHandler({}))


def say(*a):
    print(time.strftime("%H:%M:%S"), *a, flush=True)


def call(method, path, tok=None, body=None, raw=None):
    data = raw if raw is not None else (json.dumps(body).encode() if body is not None else None)
    r = urllib.request.Request(API + path, data=data, method=method)
    r.add_header("Content-Type", "image/jpeg" if raw is not None else "application/json")
    if tok:
        r.add_header("Authorization", "Bearer " + tok)
    try:
        with op.open(r, timeout=60) as x:
            b = x.read()
            return x.status, json.loads(b) if b[:1] == b"{" else b
    except urllib.error.HTTPError as e:
        return e.code, e.read()


def code_for(email):
    for _ in range(40):
        m = re.findall(re.escape(email) + r".{0,400}?(\d{6})", open(LOG, encoding="utf8", errors="replace").read())
        if m:
            return m[-1]
        time.sleep(0.5)
    raise SystemExit("no code for " + email)


def portrait(name, color):
    """A placeholder picture. The fake model approves it; no real person is pictured."""
    img = Image.new("RGB", (900, 1125), color)
    d = ImageDraw.Draw(img)
    d.ellipse((300, 220, 600, 520), fill=(250, 225, 205))
    d.rounded_rectangle((180, 580, 720, 1125), 90, fill=tuple(max(0, c - 50) for c in color))
    d.text((40, 40), name, fill=(255, 255, 255))
    out = io.BytesIO()
    img.save(out, "JPEG", quality=88)
    return out.getvalue()


def seed(name, gender, seeking, birth, interests, bio, job, color):
    email = name.lower() + "@demo.synagent.test"
    s, r = call("POST", "/v1/auth/signup", body={"email": email, "password": PW, "name": name, "accept_terms": True})
    if s != 201:
        raise SystemExit(f"signup {name}: {s} {r}")
    tok = r["token"]
    say(name, "verify", call("POST", "/v1/auth/verify", tok, {"code": code_for(email)})[0])
    s, r = call("PATCH", "/v1/profile", tok, {"birthdate": birth, "gender": gender, "seeking": seeking, "goal": "long_term", "age_min": 24, "age_max": 40,
                                              "interests": interests, "bio": bio, "occupation": job, "smoking": "no", "drinking": "sometimes", "wants_kids": "open",
                                              "prompts": [{"q": "Your ideal Sunday?", "a": "Record shops, then ramen, then a late jazz set."}]})
    say(name, "profile", s)
    say(name, "city", call("POST", "/v1/profile/city", tok, {"query": "Testville"})[0])
    say(name, "photo", call("POST", "/v1/photos", tok, raw=portrait(name, color))[0])
    return tok


def wait_ready(tok, name):
    for _ in range(60):
        me = call("GET", "/v1/me", tok)[1]
        ps = call("GET", "/v1/photos", tok)[1]["photos"]
        if ps and ps[0]["status"] == "approved" and not me.get("missing"):
            say(name, "ready", me["profile"]["status"])
            return
        time.sleep(2)
    raise SystemExit(f"{name} never got ready: {me.get('missing')} {ps}")


def sql(q):
    return subprocess.run(["psql", "-U", os.environ.get("USER", "runner"), "-d", "syn", "-Atc", q], capture_output=True, text=True).stdout.strip()


leo = seed("Leo", "man", ["woman"], "1994-08-20", ["live jazz", "ramen", "vinyl records"], "Sound engineer. I'll trade you a record for a ramen spot.", "Sound engineer", (70, 90, 150))
wait_ready(leo, "Leo")
mia = seed("Mia", "woman", ["man"], "1997-04-12", ["live jazz", "ramen", "hiking"], "A long hike, then a big bowl of ramen.", "Designer", (150, 80, 130))
wait_ready(mia, "Mia")
mia_id = call("GET", "/v1/me", mia)[1]["user"]["id"]
open(os.path.join(OUT, "seeded"), "w").write("yes\n")
say("seeded")

replied = 0
past = False
confirmed_at = None
deadline = time.time() + MINUTES * 60
while time.time() < deadline:
    time.sleep(2)
    for p in call("GET", "/v1/proposals", leo)[1].get("proposals", []):
        if p["person"]["display_name"] == "Mia" and not p.get("my_decision"):
            say("Leo says yes", call("POST", f"/v1/proposals/{p['id']}/decide", leo, {"decision": "accept"})[0])
    matches = call("GET", "/v1/matches", leo)[1].get("matches", [])
    if not matches:
        continue
    mid = matches[0]["id"]
    if matches[0].get("status") not in (None, "active"):
        continue
    msgs = call("GET", f"/v1/matches/{mid}/messages", leo)[1].get("messages", [])
    from_mia = [m for m in msgs if m.get("sender_id") == mia_id]
    if len(from_mia) > replied:
        replied = len(from_mia)
        call("POST", f"/v1/matches/{mid}/typing", leo, {})
        s = call("POST", f"/v1/matches/{mid}/messages", leo, {"text": "Kind of Blue, on vinyl. We should go hear a set together.", "client_id": uuid.uuid4().hex})[0]
        say("Leo replies", s)
    did = call("GET", f"/v1/matches/{mid}", leo)[1].get("match", {}).get("open_date_id")
    if not did:
        continue
    d = call("GET", f"/v1/dates/{did}", leo)[1].get("date", {})
    if d.get("status") == "voting" and not d.get("my_vote"):
        vote = call("GET", f"/v1/dates/{did}", mia)[1]["date"].get("my_vote")
        if vote:
            say("Leo votes", call("POST", f"/v1/dates/{did}/vote", leo, {"venue_id": vote})[0])
    elif d.get("status") == "scheduling" and d.get("proposed_by") == "them":
        # Long enough for the flow to see Mia waiting on him.
        time.sleep(3)
        say("Leo confirms", call("POST", f"/v1/dates/{did}/confirm", leo, {})[0])
    elif d.get("status") == "confirmed" and not past:
        confirmed_at = confirmed_at or time.time()
        # The flow needs a moment on the confirmed screen before the date is over.
        if time.time() - confirmed_at > 75:
            sql(f"UPDATE date_plans SET scheduled_at=now()-interval '2 minutes', proposed_at=now()-interval '2 minutes' WHERE id='{did}'")
            past = True
            say("date moved into the past")
say("time is up")
