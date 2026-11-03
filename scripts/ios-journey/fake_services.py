"""Stands in for the model and the map on the iOS journey run, in the same way that
internal/e2e/fakes_test.go does for the tests. It recognises which agent is calling from the
tools it was offered and answers like a well-behaved model. Everything around it is real."""
import json, re, sys, time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

RE_ALIAS = re.compile(r'"alias": "(c\d+)"')
RE_VENUE = re.compile(r'"id":\s*"(osm-[a-z]+-\d+)"')
RE_REPORT = re.compile(r'report_id: ([0-9a-f-]{36})')

OVERPASS = {"elements": [
    {"type": "node", "id": 1, "lat": 1, "lon": 2, "tags": {"amenity": "bar", "name": "Blue Note", "addr:street": "Main St"}},
    {"type": "node", "id": 2, "lat": 1, "lon": 2, "tags": {"amenity": "restaurant", "name": "Ramen Lab"}},
    {"type": "node", "id": 3, "lat": 1, "lon": 2, "tags": {"leisure": "park", "name": "River Park"}},
    {"type": "node", "id": 4, "lat": 1, "lon": 2, "tags": {"amenity": "cafe", "name": "Cafe One"}},
]}
NOMINATIM = [{"lat": "1.0", "lon": "2.0", "address": {"city": "Testville", "state": "Test", "country": "Testland", "country_code": "tt"}}]
VISION = {"person_visible": True, "face_visible": True, "one_main_person": True, "appears_under_18": False, "nudity_or_sexual": False,
          "violent_or_hateful": False, "contact_info_visible": False, "tip": "Nice light"}


def text_of(m):
    c = m.get("content")
    return c if isinstance(c, str) else json.dumps(c or "")


def call(cid, name, args):
    return {"id": cid, "type": "function", "function": {"name": name, "arguments": json.dumps(args)}}


def route_for(text):
    if "nurse" in text:
        return "profile_work_education"
    if "laughed" in text or "all done" in text:
        return "debrief_answer"
    return "meta_smalltalk"


def aliases(user):
    seen = []
    for a in RE_ALIAS.findall(user):
        if a not in seen and a != "you":
            seen.append(a)
    return seen


def answer(body):
    """Returns (content, tool calls) for one chat completion."""
    msgs = body.get("messages", [])
    has = {t["function"]["name"] for t in body.get("tools") or []}
    everything = "\n".join(text_of(m) for m in msgs)
    # Suki's per-call reminder isn't part of the conversation.
    if len(msgs) > 1 and text_of(msgs[-1]).startswith("If this is your reply"):
        msgs = msgs[:-1]
    last = msgs[-1]
    sys_prompt = text_of(msgs[0])
    user = next((text_of(m) for m in reversed(msgs) if m.get("role") == "user"), "")
    done = last.get("role") == "tool"

    if "image_url" in json.dumps(last.get("content") or ""):
        return json.dumps(VISION), []
    if "content moderator" in sys_prompt:
        return json.dumps({"sexual_explicit": False, "sexual_solicitation": False, "minors": False, "graphic_violence": False}), []
    if "You extract profile facts" in sys_prompt:
        return (json.dumps({"taste_note": "Goes for creative types"}) if "creative types" in user else "{}"), []
    if "Classify a message" in sys_prompt:
        return json.dumps({"intent": route_for(user)}), []

    if "compat_record" in has:
        if done:
            return "Evaluated.", []
        return "", [call(a, "compat_record", {"alias": a, "score": 88, "reasons": ["shared interests"], "shared": ["jazz"],
                                              "pitch_for_viewer": "You both light up about live jazz.",
                                              "pitch_for_candidate": "You two share a love of jazz."}) for a in aliases(user)]
    if "critic_verdict" in has:
        if done:
            return "Reviewed.", []
        return "", [call(a, "critic_verdict", {"alias": a, "verdict": "approved", "notes": "fine"}) for a in aliases(user)]
    if "venue_shortlist" in has:
        if done:
            return "Shortlisted.", []
        picks = []
        for v in RE_VENUE.findall(user):
            if v not in [p["venue_id"] for p in picks] and len(picks) < 3:
                picks.append({"venue_id": v, "reason": "You both love jazz, and it's easy to talk here."})
        return "", [call("v", "venue_shortlist", {"picks": picks})]
    if "evaluation_record" in has:
        if done:
            return "Recorded.", []
        return "", [call("e", "evaluation_record", {"chemistry": 9, "respect": 9, "conversation": 8, "reliability": 6, "profile_honesty": 6,
                                                    "see_again": True, "evidence": ["we laughed a lot"], "safety_concern": False})]
    if "taste_note_add" in has and "coach_suggest" not in has and "TasteLearner" in sys_prompt:
        return "No new notes.", []
    if "safety_report" in has and "Sentinel" in sys_prompt:
        if done:
            return "Assessed.", []
        m = RE_REPORT.search(everything)
        if not m:
            return "No report id.", []
        return "", [call("s", "safety_report", {"report_id": m.group(1), "severity": "high", "details": "Reviewed by the test."})]
    if "coach_suggest" in has and "conversation coach" in sys_prompt:
        if done:
            return "Posted.", []
        return "", [call("c", "coach_suggest", {"match_id": "a-guess-from-the-model", "note": "You both love jazz. Open with that!",
                                                "suggestions": ["What's the best live jazz you've seen?"]})]

    # Suki herself.
    if last.get("role") == "system" and "EVENT" in text_of(last):
        return "So, how did it go? <<mood=playful|replies=Loved it;It was okay;Not for me>>", []
    if done:
        return "Saved! <<mood=happy|replies=Tell me more>>", []
    if "all done" in user and "debrief_close" in has:
        return "", [call("d", "debrief_close", {"skip": False})]
    return "Tell me more! <<mood=thinking>>", []


class Handler(BaseHTTPRequestHandler):
    protocol_version = "HTTP/1.1"

    def log_message(self, *a):
        pass

    def send_json(self, value, status=200):
        raw = json.dumps(value).encode()
        self.send_response(status)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)

    def do_GET(self):
        p = self.path.split("?")[0]
        if p.startswith("/photon"):
            self.send_json({"error": "the fake map only speaks Overpass"}, 503)
        elif p.endswith("/overpass"):
            self.send_json(OVERPASS)
        else:
            self.send_json(NOMINATIM)

    def do_POST(self):
        raw = self.rfile.read(int(self.headers.get("Content-Length") or 0))
        p = self.path.split("?")[0]
        if p.endswith("/overpass"):
            return self.send_json(OVERPASS)
        if p.endswith("/push"):
            return self.send_json({"data": []})
        body = json.loads(raw or b"{}")
        try:
            content, calls = answer(body)
        except Exception as e:
            print("fake model error:", repr(e), flush=True)
            return self.send_json({"error": "fake model error"}, 400)
        print(p, "tools" if calls else "text", (content or calls[0]["function"]["name"])[:70].replace("\n", " "), flush=True)
        if not body.get("stream"):
            msg = {"role": "assistant", "content": content}
            if calls:
                msg["tool_calls"] = calls
            return self.send_json({"model": "fake", "choices": [{"message": msg, "finish_reason": "stop"}],
                                   "usage": {"prompt_tokens": 10, "completion_tokens": 5}})
        self.send_response(200)
        self.send_header("Content-Type", "text/event-stream")
        self.send_header("Connection", "close")
        self.end_headers()
        self.close_connection = True

        def send(delta):
            self.wfile.write(("data: " + json.dumps({"choices": [{"delta": delta}]}) + "\n\n").encode())
            self.wfile.flush()
        for i in range(0, len(content), 4):
            send({"content": content[i:i + 4]})
            time.sleep(0.01)
        for i, c in enumerate(calls):
            args = c["function"]["arguments"]
            half = len(args) // 2
            send({"tool_calls": [{"index": i, "id": c["id"], "type": "function", "function": {"name": c["function"]["name"], "arguments": args[:half]}}]})
            send({"tool_calls": [{"index": i, "function": {"arguments": args[half:]}}]})
        self.wfile.write(b"data: [DONE]\n\n")
        self.wfile.flush()


if __name__ == "__main__":
    port = int(sys.argv[1]) if len(sys.argv) > 1 else 9000
    ThreadingHTTPServer(("127.0.0.1", port), Handler).serve_forever()
