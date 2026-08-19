---
description: Deploy AlphaForge backend and frontend correctly
---

# Deploy AlphaForge

## Backend

```bash
# 1. Reclaim Docker build cache (58G disk fills quickly)
docker builder prune -f

# 2. Build and start backend
docker compose build backend && docker compose up -d backend
```

- **Never** run `go build` on the host and `docker cp` the binary — host glibc is incompatible with Alpine musl.
- **Never** `go build` or `apk add build-base` inside a running container.
- Uncommitted `backend/migrations/*.up.sql` are executed during Docker build. Move WIP migration files before building.
- Binary path in container: `/app/alphaforge`.

## Frontend

```bash
# Build
npm run build

# Deploy (overlay copy; nginx must reload)
docker cp frontend/dist/. alphaforge-frontend:/usr/share/nginx/html/
docker exec alphaforge-frontend nginx -s reload
```

- The frontend container runs as non-root. Old `assets/` are not removed by `docker cp`, so old JS chunks accumulate.
- Before major deployments, clean old chunks as root:

  ```bash
  docker exec -u root alphaforge-frontend rm -rf /usr/share/nginx/html/assets
  ```

- Cloudflare caches `.js` chunks for 4h (`max-age=14400`), but `index.html` is DYNAMIC (no cache). New HTML points to new chunks, so as long as container assets are clean, users will get the new bundle.
- Verify: `curl -sI https://<host>/` should return `Cache-Control: no-cache`.

## Verification

```bash
curl -s http://localhost:8080/healthz
curl -s http://localhost:8080/readyz
```
