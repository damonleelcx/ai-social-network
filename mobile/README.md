# SynAgent mobile

The iOS and Android client: Expo, React Native, TypeScript, expo-router. It needs Node 22 or newer.

The API contract is [`../docs/03-api.md`](../docs/03-api.md).

## Run against a local API

`EXPO_PUBLIC_API_URL` picks the server. It defaults to `https://synagent.heros-agent.space`.
An Android emulator reaches your machine at `10.0.2.2`.

```sh
npm ci
EXPO_PUBLIC_API_URL=http://10.0.2.2:8080 npx expo run:android
```

On PowerShell, set it first: `$env:EXPO_PUBLIC_API_URL = "http://10.0.2.2:8080"`.

Debug builds allow cleartext only to `10.0.2.2`, `localhost` and `127.0.0.1`
(`plugins/withDebugNetworking.js`). Release builds are HTTPS-only. A physical phone therefore needs
`adb reverse tcp:8080 tcp:8080` with `http://localhost:8080`, and the same for Metro on 8081.

### Without Metro

Debug APKs embed the JS bundle, so an installed one runs on its own. The API URL is
fixed when the bundle is built. If Metro is running, the app uses Metro instead.

```sh
npx expo prebuild --platform android
cd android && EXPO_PUBLIC_API_URL=http://10.0.2.2:8080 ./gradlew assembleDebug
adb install -r app/build/outputs/apk/debug/app-debug.apk
```

Gradle needs JDK 17 as `JAVA_HOME`. It reads its proxy from Java system properties, not from
`HTTPS_PROXY`, so behind a proxy pass `-Dhttps.proxyHost=... -Dhttps.proxyPort=...` and the same
in `GRADLE_OPTS` for the wrapper's download.

The debug build works from a normal Windows checkout, without `subst`. Two settings keep it under
the 260-character path limit: `plugins/withDebugNetworking.js` turns on CMake unity builds, and
`react-native.config.js` leaves out worklets, reanimated and gesture-handler, which the app doesn't
use. Don't use `subst` here: Node's realpath sees through it, and Gradle fails with "different roots".

An emulator started from a shell with `HTTP_PROXY` set sends `10.0.2.2` through that proxy, and the
app hangs on its splash screen. Clear the variable, or start the emulator from Android Studio.

## Checks

```sh
npx tsc --noEmit
npx expo export --platform android
npx expo export --platform ios
```

## iOS and store builds

`eas.json` has `development`, `preview` and `production` profiles. iOS needs a Mac or EAS.

Push needs an EAS project. Set `EAS_PROJECT_ID` at build time. Without it the app skips
push registration, and notifications still arrive in-app over the socket.

## Art

The icon, adaptive icon, splash and notification glyph in `assets/brand` come from
`assets/suki/portrait.png` through `scripts/generate_brand.py` (needs Pillow).
