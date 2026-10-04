# InsideOut Flutter client

Material 3 app that talks to the existing Go `/api/v1` API. Builds for
web, iOS, and Android. Railway production serves this tree (`client/`
Dockerfile + nginx).

## Run

The Go server must be reachable. Local default:

```bash
# from repo root, with .env exported
./scripts/dev.sh -C server go run ./cmd/insideout

cd client
flutter run -d chrome
flutter run -d ios
flutter run -d android
```

Against the hosted API:

```bash
flutter run -d chrome --dart-define=API_BASE=https://<your-app>.up.railway.app/api/v1   # your deployment's host
```

Register or sign in. Tokens are stored with `flutter_secure_storage`.
The server still sets httpOnly cookies (unused by this client).

The UI defaults to **zh-CN**. Use the translate and theme icons in the
top-right cluster to switch to en-US or dark mode. Signed-in chrome is
the bottom dock (Capture / Ideas / Version / Commit / Roadmap / Evidence).

## Checks

```bash
flutter analyze --no-fatal-infos
flutter test
```
