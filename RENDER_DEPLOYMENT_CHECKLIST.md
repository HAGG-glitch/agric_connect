# AgriConnect Render Deployment Checklist

## Before Deployment

- [x] Tests pass (`go test ./...`)
- [x] `go vet ./...` passes
- [x] `go build ./...` succeeds
- [x] Docker image builds (`docker build -t agriconnect-render .`)
- [x] `.env` is in `.gitignore` (not committed)
- [x] `.env.example` contains placeholders only (no real secrets)
- [x] No real secrets in repository (verified with `git grep`)
- [x] Tailwind CSS built in Docker image
- [x] All templates copied into Docker image
- [x] All JavaScript files copied into Docker image
- [x] Migration SQL files copied into Docker image
- [x] AI prompt files copied into Docker image
- [x] CA certificates installed in Docker image
- [x] Server binary built and included in Docker image
- [x] Health endpoint returns JSON
- [x] Cache busting via `ASSET_VERSION` env var
- [x] `render.yaml` created with Blueprint specification
- [x] `.dockerignore` created to exclude unnecessary files
- [x] Application binds to `0.0.0.0:$PORT`
- [x] Graceful shutdown implemented (SIGINT/SIGTERM)
- [x] HTTP server timeouts configured

## Render Configuration

- [ ] Web service created (Docker runtime)
- [ ] PostgreSQL database created
- [ ] Web service and database in same region
- [ ] `DATABASE_URL` connected from PostgreSQL (Blueprint automates this)
- [ ] `GROQ_API_KEY` entered as secret
- [ ] `SUPABASE_URL` entered as secret
- [ ] `SUPABASE_SECRET_KEY` entered as secret
- [ ] `JWT_ACCESS_SECRET` entered as secret
- [ ] `JWT_REFRESH_SECRET` entered as secret
- [ ] `STORAGE_DRIVER` set to `supabase`
- [ ] `COOKIE_SECURE` set to `true`
- [ ] Health check path set to `/health`
- [ ] Blueprint resources match expected services

## After Deployment

- [ ] `/health` returns 200 with `{"status":"healthy"}`
- [ ] Home page loads
- [ ] Mobile hamburger menu works
- [ ] Registration works (POST `/api/v1/auth/register`)
- [ ] Login works (POST `/api/v1/auth/login`)
- [ ] AI assistant responds
- [ ] Weather endpoint returns data
- [ ] Crop upload works
- [ ] Diagnosis result shows analysis
- [ ] Private image loads via signed URL
- [ ] Voice transcription works
- [ ] Officer routes protected (returns 401/403 for non-officer)
- [ ] Admin routes protected (returns 401/403 for non-admin)
- [ ] No secrets appear in browser source
- [ ] No critical errors in Render logs
