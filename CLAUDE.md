# Working on this repository

This is Sam's fork of ezBookkeeping (upstream mayswind/ezbookkeeping), with family books, projects,
merchants and other additions.

## The Android app is in another repository

The Android app lives in **https://github.com/ssiuuk/ezbookkeeping-android**. It shares no code with
this repository; it calls this server's HTTP API (`/api/v1/...`), so API changes here can break it.

- When you change a request or response the app uses (models in `pkg/models`, routes in
  `cmd/webserver.go`, handlers in `pkg/api`), check the app's `data/` package
  (`app/src/main/java/com/ssiuuk/ezbookkeeping/data/`) and update it in that repository too.
- For a feature that needs both sides, open the server pull request here, and an app pull request
  there whose `SERVER_REF` file names this pull request's branch, so the app's CI tests them together.
  Merge the server one first, then set `SERVER_REF` back to `main` in the app one and merge it.
- `.github/workflows/notify-android-app.yml` starts the app's emulator test whenever `main` changes
  (it needs the `ANDROID_REPO_TOKEN` secret). The app's test also runs nightly, and opens an issue in
  the app repository when it fails against the latest server.
