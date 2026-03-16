# Verifying Deployed Version (Railway / any host)

If the client shows an "older" UI after deploy, use this to confirm what is actually running and fix caching.

## 1. Check what build is running

After each deploy, the server exposes the build in the health response:

```bash
curl -s https://YOUR-RAILWAY-URL/health | jq .
```

Example:

```json
{
  "status": "ok",
  "service": "checkdepot",
  "version": "1.0.0",
  "build_version": "a1b2c3d"
}
```

- **`build_version`** is set at Docker build time:
  - On **Railway**: from `RAILWAY_GIT_COMMIT_SHA` (short commit), so you can match it to the commit you deployed.
  - Locally: from `git rev-parse --short HEAD` or `"local"` / `"dev"`.

If you deploy a new commit and `build_version` in the response **does not change**, the new image is not running (e.g. old deployment still active, or build cache produced the same binary).

## 2. Force Railway to build and run the new image

- **Redeploy**: In Railway, trigger a new deployment from the correct branch/commit.
- **Clear build cache**: In the service → **Settings** or **Deployments**, use **Clear build cache** (or equivalent) so the next build does not reuse old Docker layers, then redeploy.
- **Confirm source**: In the deployment that is “active”, check that the commit SHA matches the one you expect. If the active deployment is an older commit, promote or redeploy from the latest.

## 3. Avoid browser cache (embedded UI)

The app now sends `Cache-Control: no-cache, no-store, must-revalidate` for HTML and JS/CSS so that after a new deploy, the next request gets the new files without a hard refresh.

If you still see old UI:

- Do a **hard refresh**: Ctrl+Shift+R (Windows/Linux) or Cmd+Shift+R (Mac).
- Or open the app in an **incognito/private** window.

## 4. If `build_version` is still wrong after a fresh deploy

- Ensure the **Dockerfile** build runs with the latest source (no `.dockerignore` excluding `web/` or the wrong context).
- On Railway, ensure the build uses the commit you expect (e.g. correct branch, no stale cache). Re-run the build with cache cleared and compare the new `build_version` to your git log.
