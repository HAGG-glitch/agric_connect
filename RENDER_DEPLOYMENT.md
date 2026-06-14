# AgriConnect Render Deployment

This guide explains how to deploy AgriConnect on Render for user testing.

---

## Architecture

| Service          | Provider        | Plan  |
|------------------|-----------------|-------|
| Web Service      | Render (Docker) | Free  |
| PostgreSQL       | Render          | Free  |
| Object Storage   | Supabase        | Free  |
| AI/LLM           | Groq            | Pay   |
| Weather          | Open-Meteo      | Free  |
| Voice AI         | Groq (Whisper)  | Pay   |

---

## Prerequisites

- GitHub repository pushed and accessible to Render
- Render account (sign up at https://render.com)
- Supabase account with a project and a private storage bucket named `crop-diagnosis-images`
- Groq API key (sign up at https://console.groq.com)

---

## Deployment Steps

### 1. Push the repository to GitHub

```bash
git push -u origin main
```

### 2. Create a Render Blueprint

1. Sign in to [Render Dashboard](https://dashboard.render.com).
2. Click **New +** → **Blueprint**.
3. Connect your GitHub repository.
4. Select the `agric_connect` repository.
5. Render will automatically read `render.yaml`.
6. Review the resources (Web Service + PostgreSQL).
7. Click **Apply**.

### 3. Enter secret environment variables

After applying the Blueprint, Render will prompt for secret values. Enter:

| Variable              | Description                           |
|-----------------------|---------------------------------------|
| `GROQ_API_KEY`        | Your Groq API key                     |
| `SUPABASE_URL`        | Your Supabase project URL             |
| `SUPABASE_SECRET_KEY` | Your Supabase service role secret     |
| `JWT_ACCESS_SECRET`   | Random long string (min 32 chars)     |
| `JWT_REFRESH_SECRET`  | Random long string (min 32 chars)     |

Generate secrets with:

```bash
openssl rand -base64 32
```

### 4. Deploy

1. Click **Deploy**.
2. Monitor the build logs for any errors.
3. Wait for the deployment to complete.

### 5. Verify the deployment

```bash
curl https://<your-service>.onrender.com/health
```

Expected response:
```json
{"status":"healthy","database":"connected"}
```

---

## Required Environment Variables

### Non-secret (set in `render.yaml`)

| Variable                    | Value                    | Purpose                               |
|-----------------------------|--------------------------|---------------------------------------|
| `APP_ENV`                   | `production`             | Application environment               |
| `APP_PORT`                  | `8080`                   | Internal port (Render provides `PORT`)|
| `COOKIE_SECURE`             | `true`                   | Secure cookies in production          |
| `COOKIE_SAME_SITE`          | `lax`                    | CSRF protection                       |
| `STORAGE_DRIVER`            | `supabase`               | Production image storage              |
| `SUPABASE_STORAGE_BUCKET`   | `crop-diagnosis-images`  | Supabase bucket name                  |
| `MAX_IMAGE_SIZE_MB`         | `5`                      | Max upload image size                 |
| `MAX_AUDIO_SIZE_MB`         | `10`                     | Max upload audio size                 |
| `MIN_IMAGE_WIDTH`           | `256`                    | Minimum image width for analysis      |
| `MIN_IMAGE_HEIGHT`          | `256`                    | Minimum image height for analysis     |
| `MAX_IMAGE_PIXELS`          | `25000000`               | Maximum total pixels (25MP)           |

### Secret (entered manually in Render dashboard)

| Variable              | Required | Purpose                            |
|-----------------------|----------|------------------------------------|
| `GROQ_API_KEY`        | Yes      | Groq API authentication            |
| `SUPABASE_URL`        | Yes      | Supabase project URL               |
| `SUPABASE_SECRET_KEY` | Yes      | Supabase service role key          |
| `JWT_ACCESS_SECRET`   | Yes      | JWT access token signing           |
| `JWT_REFRESH_SECRET`  | Yes      | JWT refresh token signing          |

### Render-provided

| Variable        | Source                           |
|-----------------|----------------------------------|
| `PORT`          | Render assigns at runtime        |
| `DATABASE_URL`  | From Render PostgreSQL via Blueprint |
| `ASSET_VERSION` | Render `deployId` for cache busting |

---

## Public Routes

| Method | Path                          | Description           |
|--------|-------------------------------|-----------------------|
| GET    | `/health`                     | Health check          |
| GET    | `/`                           | Home / Assistant      |
| GET    | `/assistant`                  | AI chat               |
| GET    | `/diagnose`                   | Crop diagnosis form   |
| GET    | `/diagnoses`                  | Diagnosis history     |
| GET    | `/diagnoses/:id`              | Diagnosis details     |
| GET    | `/login`                      | Login page            |
| GET    | `/register`                   | Registration page     |
| GET    | `/officer`                    | Officer dashboard     |
| GET    | `/officer/diagnoses`          | Officer diagnosis list|
| GET    | `/officer/diagnoses/:id`      | Officer diagnosis view|
| GET    | `/admin/users`                | Admin user management |
| POST   | `/api/v1/auth/register`       | Register user         |
| POST   | `/api/v1/auth/login`          | Login                 |
| POST   | `/api/v1/auth/refresh`        | Refresh token         |
| POST   | `/api/v1/auth/logout`         | Logout                |
| GET    | `/api/v1/auth/me`             | Current user          |
| POST   | `/api/v1/diagnoses`           | Create diagnosis      |
| GET    | `/api/v1/diagnoses`           | List diagnoses        |
| GET    | `/api/v1/diagnoses/:id`       | Get diagnosis         |
| DELETE | `/api/v1/diagnoses/:id`       | Delete diagnosis      |
| GET    | `/api/v1/diagnoses/:id/image` | Serve diagnosis image |
| POST   | `/api/v1/diagnoses/:id/continue-in-chat` | Continue in chat |
| POST   | `/api/v1/ai/transcribe`       | Transcribe audio      |
| POST   | `/api/v1/conversations`       | Create conversation   |
| GET    | `/api/v1/conversations`       | List conversations    |
| POST   | `/api/v1/conversations/:id/messages` | Send message   |
| GET    | `/api/v1/weather`             | Get weather           |
| GET    | `/api/v1/officer/diagnoses`   | Officer diagnosis API |
| GET    | `/api/v1/officer/diagnoses/:id` | Officer diagnosis detail |
| POST   | `/api/v1/officer/diagnoses/:id/reviews` | Create review |
| PUT    | `/api/v1/officer/diagnoses/:id/reviews/:rid` | Update review |
| GET    | `/api/v1/admin/users`         | List users            |
| PATCH  | `/api/v1/admin/users/:id/role` | Update user role     |
| PATCH  | `/api/v1/admin/users/:id/status` | Update user status  |
| GET    | `/api/v1/notifications`       | List notifications    |
| PATCH  | `/api/v1/notifications/:id/read` | Mark notification read |

---

## Troubleshooting

### No open port detected
Ensure the application reads the `PORT` environment variable. Render sets `PORT` automatically.

### Application binding to localhost
The application binds to `0.0.0.0` by default (Gin's `":" + port` syntax). No change needed.

### Database connection refused
- Verify `DATABASE_URL` is set correctly (Render provides it via Blueprint)
- Check if the database and web service are in the same region
- Render's free PostgreSQL uses SSL — the current driver handles this

### Migration failure
- Check Render build logs for migration errors
- Migrations use `CREATE TABLE IF NOT EXISTS` and are idempotent
- Migration files must be in the `migrations/` directory

### Missing template or static file
- Verify the Docker image contains `web/templates/`, `web/static/`, and `migrations/`
- Run `docker run --rm agriconnect-render ls /app/web/templates` to verify locally

### Stale static assets
- `ASSET_VERSION` (from Render deploy ID) adds cache-busting query parameter
- Clear browser cache if CSS/JS doesn't update

### Supabase 401 or 403
- Verify `SUPABASE_URL` and `SUPABASE_SECRET_KEY` are entered correctly
- Ensure the storage bucket name matches `crop-diagnosis-images`
- The bucket must be private (not public)

### Groq authentication failure
- Verify `GROQ_API_KEY` is entered in Render secrets
- Key should start with `gsk_`

### Health check failure
- `GET /health` returns `{"status":"healthy"}` when the app is ready
- If it returns 503, the database connection failed — check `DATABASE_URL`
- No external API calls are made during health checks

### Secure-cookie login issue
- `COOKIE_SECURE=true` requires HTTPS
- Render provides HTTPS automatically behind the proxy
- `APP_URL` should match the Render service URL

---

## Testing Checklist

- [ ] `GET /health` returns `200 {"status":"healthy","database":"connected"}`
- [ ] Home page loads at `GET /`
- [ ] Mobile hamburger menu toggles sidebar
- [ ] Registration works at `/register`
- [ ] Login works at `/login`
- [ ] AI assistant responds at `/assistant`
- [ ] Weather works at `/api/v1/weather?district=bombali`
- [ ] Crop upload works at `/diagnose`
- [ ] Image displays after diagnosis via `/api/v1/diagnoses/:id/image`
- [ ] Voice transcription works via `/api/v1/ai/transcribe`
- [ ] Officer routes (`/officer`) are protected by role
- [ ] Admin routes (`/admin/users`) are protected by role
- [ ] No secrets appear in browser source or Render logs

---

## Manual Blueprint Alternative

If the Blueprint fails, create resources manually:

1. **PostgreSQL**: New → PostgreSQL → Select free plan → Same region
2. **Web Service**: New → Web Service → Select repo → Runtime: Docker → Health path: `/health`
3. Copy the `DATABASE_URL` from the PostgreSQL dashboard to the Web Service environment
4. Add all environment variables from the tables above
5. Deploy
