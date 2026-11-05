# Release pipeline

Every merge into main runs [`.github/workflows/release.yml`](../.github/workflows/release.yml). It tests, then ships whatever has changed since that part last shipped. A docs-only merge ships nothing.

| Part | Ships when these change | What happens | Needs |
|---|---|---|---|
| Server and web app | `cmd`, `internal`, `go.mod`, `go.sum`, `Dockerfile`, `deploy`, `mobile` | Builds one image, deploys it, checks the live site | AWS role (below) |
| Native app | `mobile` | Next version, Android and iOS packages, GitHub release. The Android app then offers the update | Already set up |
| Mini program | `weixin` | Uploads a new version to WeChat | WeChat setup (below) |

The order matters. If tests fail, nothing ships. If the deploy fails, the native app and mini program wait, because they shouldn't reach people before the server they talk to. A part that fails is retried by the next merge, since each run compares with what last shipped, not with the previous merge.

## Versions

The latest `app-v*` tag decides the next version, and the build gets it through `APP_VERSION`. Nothing is committed back to main, so the `version` in `mobile/app.config.ts` is only a fallback for local builds.

Merges raise the patch number. To raise minor or major, run the Release workflow by hand and choose it. That also ships whatever hasn't shipped yet.

Android's version code comes from the version: 1.3.4 is 10304. It has to keep rising, or Android refuses the update.

## Setup

**Server and web app.** Sign in to AWS and to `gh`, then run this once. It lets main in this repository deploy, with no AWS keys stored in GitHub, and sets the `AWS_DEPLOY_ROLE_ARN` and `SYN_INSTANCE` variables.

```bash
INSTANCE=i-05f4712279b04fac5 deploy/ci/bootstrap-github-role.sh
```

The role can run any command as root on that instance, because that's how deploying works here, and the node also runs other things. Anyone who can merge into main can use it, so protect main.

**Mini program.** It needs the registered mini program from [`weixin/README.md`](../weixin/README.md) first. In the WeChat console, under development settings:

1. Generate a code upload key, and store it as the `WEIXIN_UPLOAD_KEY` secret.
2. Turn off the IP allowlist for uploads, because GitHub's runners don't have fixed addresses.
3. Set the `WEIXIN_APPID` variable, and `WEIXIN_API_URL` to the ICP-filed domain.

## What it can't do

- **Mini program review.** WeChat gives a mini program's own developer no way to submit for review or publish from CI. The pipeline stops at the uploaded version. Someone then selects it in the console, submits it, and publishes it once approved.
- **Store releases.** The packages go to GitHub releases and the landing page. Nothing goes to Google Play or the App Store.
- **Database rollbacks.** Migrations run on deploy and aren't undone. A failed rollout leaves the old pods serving, but they'd be running against any migration that already ran.

## Making sure it means something

Main has no branch protection, so nothing stops a merge before its checks pass, and every merge now goes to production. Make the `test` check required on main.

If the server or mini program changed but its setup is missing, the run shows a warning rather than passing quietly.
