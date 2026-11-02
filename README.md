# SynAgent

An agentic dating app for iOS and Android. **Suki**, an AI matchmaker with her own
character, face and voice, gets to know you in conversation, introduces you to people in
your city, coaches your chats, finds a date spot you'd both enjoy, and learns your type
from how it all goes.

| Docs | |
|---|---|
| [Intents, tools and skills](docs/01-intents-tools-skills.md) | Everything the agents can understand and do |
| [Architecture](docs/00-architecture.md) | How it works, and agent principles mapped to code |
| [Suki's soul](docs/02-soul.md) | Her character |
| [API](docs/03-api.md) | The contract with the mobile app |
| [Mobile](mobile/README.md) | Running the app |
| [Demo video](mobile/store/demo/synagent-demo.mp4) | The whole journey on Android, narrated by Suki |
| [Store images](mobile/store/images) | Play Store and App Store assets, built from real screens by `mobile/store/make_store_images.py` |

## Run it locally

You need Go, Docker and an OpenAI-compatible model key.

```bash
docker run -d --name syn-pg -e POSTGRES_USER=syn -e POSTGRES_HOST_AUTH_METHOD=trust -e POSTGRES_DB=syn -p 127.0.0.1:55870:5432 postgres:17
```

```bash
docker run -d --name syn-mongo -p 127.0.0.1:55871:27017 mongo:7
```

Create a git-ignored `.env`, export it, then start the server:

```
SYN_DATABASE_URL=postgres://syn@127.0.0.1:55870/syn?sslmode=disable
SYN_MONGO_URL=mongodb://127.0.0.1:55871
SYN_PHOTO_DIR=./.photos
SYN_LLM_API_KEY=...
SYN_ADDR=:8080
```

```bash
go run ./cmd/synagent serve
```

Without a mail relay, emails (including verification codes) go to the server log.

## Tests

```bash
go test ./...
```

The tests use real Postgres and MongoDB through the containers above, with the model and
the map faked. They cover the whole journey, crash recovery, a model outage, NSFW
filtering, and a search of every response body for leaked scores.

## Deploy

One-time AWS setup, then release. Both are idempotent. Set `ADMIN_EMAILS` so someone can moderate.

```bash
ADMIN_EMAILS=<moderators> ZONE=<route53 zone> NODE_IP=<node ip> deploy/bootstrap-aws.sh
```

```bash
INSTANCE=i-05f4712279b04fac5 deploy/release.sh
```
