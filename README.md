# AgriConnect AI

Agricultural advisory web application for smallholder farmers in Sierra Leone. Features AI-powered crop diagnosis, weather intelligence, market prices, agricultural resources, and role-based dashboards for farmers, extension officers, and administrators.

## Features

- **AI Chat Assistant** — Conversational AI in English and Krio using Groq LLM
- **Crop Image Diagnosis** — Upload crop images for AI-powered symptom analysis with Groq Vision
- **Weather Intelligence** — District-specific forecasts via Open-Meteo with AI-enhanced interpretation
- **Voice Recording & Transcription** — Groq Whisper transcription with Krio language support
- **Market Prices** — Commodity price tracking by district and market (officer-managed)
- **Agricultural Resources** — Curated knowledge base with crop-specific documents
- **Role-Based Dashboards** — Tailored views for farmers, extension officers, and admins
- **Notifications** — In-app alerts for diagnosis updates, reviews, and market changes
- **Officer Review Workflow** — Claim, review, and close diagnosis cases with district scoping
- **Admin Panel** — User management, diagnosis oversight, audit logs

## Architecture

```
cmd/server/main.go          — Application entry point
internal/auth/              — Authentication, JWT, user service
internal/config/            — Environment-based configuration
internal/database/          — PostgreSQL connection and migrations
internal/models/            — GORM models
internal/repositories/      — Data access layer
internal/services/          — Business logic (chat, knowledge, weather)
internal/handlers/          — HTTP handlers (page, API, auth, admin, officer)
internal/middleware/        — Auth, rate limiting, request ID, anonymous user
internal/diagnosis/         — Crop diagnosis service, repository, validation
internal/transcription/     — Audio transcription service
internal/storage/           — Object storage abstraction (local + Supabase)
internal/ai/                — Groq client, prompts, vision, transcription
internal/weather/           — Open-Meteo client, district coordinates
internal/validation/        — Input validation utilities
web/templates/              — Gin HTML templates (layouts, pages, partials)
web/static/                 — Tailwind CSS and JavaScript
migrations/                 — SQL migration files (auto-executed on startup)
seed/                       — Agricultural knowledge base seed data
scripts/                    — Utility programs
```

## Prerequisites

- Go 1.22+
- PostgreSQL 16+
- Node.js 20+ (for Tailwind CSS compilation)
- A Groq API key (https://console.groq.com)

## Environment Setup

Copy `.env.example` to `.env` and configure:

```bash
cp .env.example .env
```

Required variables:
- `DATABASE_URL` — PostgreSQL connection string
- `GROQ_API_KEY` — Groq API key (required in production)

## Running Locally

```bash
# Install Go dependencies
go mod tidy

# Compile Tailwind CSS
npx tailwindcss -i ./web/static/css/input.css -o ./web/static/css/app.css --minify

# Start the server
go run ./cmd/server
```

The application will be available at http://localhost:8080.

## Running with Docker

```bash
docker compose up --build
```

This starts both the application and PostgreSQL database.

## Database Migrations

Migrations run automatically on application startup. They are SQL files in `migrations/` named with a `000NNN` prefix and executed in order.

## Seeding Agricultural Documents

```bash
go run ./scripts/seed.go
```

This loads agricultural knowledge for rice, cassava, maize, and groundnut from `seed/agricultural_documents.json`.

## Seeding Demo Data

On first run, the application seeds demo users and sample market prices. All demo accounts use password `demo123`:

| Role | Name | Phone |
|------|------|-------|
| Admin | Admin User | `23276100001` |
| Extension Officer | Fatmata Kamara | `23276100002` |
| Extension Officer | Amadu Sesay (Krio) | `23276100003` |
| Farmer | Demo Farmer | `23276100004` |

## Tailwind Compilation

```bash
# One-time build
npx tailwindcss -i ./web/static/css/input.css -o ./web/static/css/app.css --minify

# Watch mode
npx tailwindcss -i ./web/static/css/input.css -o ./web/static/css/app.css --watch
```

## Groq Configuration

Set `GROQ_API_KEY` in your `.env` file. The application uses `llama-3.1-8b-instant` by default but any Groq-compatible model can be configured via `GROQ_CHAT_MODEL`.

### Vision Model

Configured via `GROQ_VISION_MODEL` (default: `llama-3.2-11b-vision-preview`). Used for crop image diagnosis.

### Transcription Model

Configured via `GROQ_TRANSCRIPTION_MODEL` (default: `whisper-large-v3`). Supports English and Krio language hints.

## Storage Configuration

### Local Storage (Development)

Default driver. Files stored under `LOCAL_UPLOAD_DIR` (default: `./data/uploads`). Images are served only through an ownership-checked HTTP handler.

### Supabase Storage (Production)

Configure via:
- `STORAGE_DRIVER=supabase`
- `SUPABASE_URL` — Supabase project URL
- `SUPABASE_SERVICE_ROLE_KEY` — Service role key (never exposed to frontend)
- `SUPABASE_STORAGE_BUCKET` — Private bucket name (default: `crop-diagnosis-images`)

## Upload Limits

| Setting | Default | Description |
|---------|---------|-------------|
| `MAX_IMAGE_SIZE_MB` | 5 | Maximum crop diagnosis image size |
| `MAX_AUDIO_SIZE_MB` | 10 | Maximum voice recording size |
| `MAX_RECORDING_SECONDS` | 60 | Automatic recording stop duration |
| `ALLOWED_IMAGE_TYPES` | image/jpeg,image/png,image/webp | Accepted image MIME types |
| `ALLOWED_AUDIO_TYPES` | audio/webm,audio/wav,audio/mpeg,audio/mp4,audio/ogg | Accepted audio MIME types |

## Open-Meteo Integration

Weather data is fetched from Open-Meteo (free, no API key required). Results are cached in PostgreSQL for 20 minutes by default, configurable via `WEATHER_CACHE_MINUTES`.

## Running Tests

```bash
go test ./...
go vet ./...
go build ./cmd/server
```

## Configuration Reference

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |
| `DATABASE_URL` | — | PostgreSQL connection string |
| `GROQ_API_KEY` | — | Groq API key |
| `GROQ_CHAT_MODEL` | `llama-3.1-8b-instant` | Chat model |
| `GROQ_VISION_MODEL` | `llama-3.2-11b-vision-preview` | Vision model |
| `GROQ_TRANSCRIPTION_MODEL` | `whisper-large-v3` | Transcription model |
| `STORAGE_DRIVER` | `local` | Storage backend (`local` or `supabase`) |
| `JWT_SECRET` | auto-generated | JWT signing secret |
| `ALLOW_ANONYMOUS_ASSISTANT` | `false` | Allow unauthenticated access to AI assistant |
| `WEATHER_CACHE_MINUTES` | `20` | Weather cache duration |
| `MAX_IMAGE_SIZE_MB` | `5` | Max image upload size |
| `MAX_AUDIO_SIZE_MB` | `10` | Max audio upload size |
| `MAX_RECORDING_SECONDS` | `60` | Max recording duration |

## API Routes

### Page Routes

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/` | No | Landing page (or role-based redirect if authenticated) |
| GET | `/login` | No | Login page |
| GET | `/register` | No | Registration page |
| GET | `/assistant` | Optional | AI chat assistant |
| GET | `/dashboard` | Farmer | Farmer dashboard |
| GET | `/officer` | Officer | Officer dashboard |
| GET | `/admin` | Admin | Admin dashboard |
| GET | `/diagnose` | Any | Crop diagnosis form |
| GET | `/diagnoses` | Any | Diagnosis history |
| GET | `/diagnoses/:id` | Any | Diagnosis detail |
| GET | `/market-prices` | Any | Market prices page |
| GET | `/resources` | Any | Agricultural resources list |
| GET | `/resources/:id` | Any | Resource detail |
| GET | `/notifications` | Any | User notifications |
| GET | `/admin/diagnoses` | Admin | Admin diagnosis oversight |
| GET | `/admin/reviews` | Admin | Admin review management |
| GET | `/admin/audit-logs` | Admin | Audit log viewer |
| GET | `/admin/users` | Admin | User management |

### Authentication API

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/auth/register` | Register new user |
| POST | `/api/v1/auth/login` | Login (returns role-based redirect) |
| POST | `/api/v1/auth/refresh` | Refresh access token |
| POST | `/api/v1/auth/logout` | Logout |
| GET | `/api/v1/auth/me` | Get current user profile |
| PATCH | `/api/v1/auth/profile` | Update profile |

### AI Chat API

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/conversations` | Create conversation |
| GET | `/api/v1/conversations` | List conversations |
| GET | `/api/v1/conversations/:id` | Get conversation with messages |
| DELETE | `/api/v1/conversations/:id` | Delete conversation |
| POST | `/api/v1/conversations/:id/messages` | Send message (non-streaming) |
| POST | `/api/v1/conversations/:id/messages/stream` | Send message (SSE streaming) |

### Weather API

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/weather?district=Bo` | Get weather for a district |

### Diagnosis API

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/diagnoses` | Submit a crop diagnosis (multipart) |
| GET | `/api/v1/diagnoses` | List diagnoses (paginated) |
| GET | `/api/v1/diagnoses/:id` | Get diagnosis detail |
| DELETE | `/api/v1/diagnoses/:id` | Delete diagnosis |
| GET | `/api/v1/diagnoses/:id/image` | Serve diagnosis image (ownership-checked) |
| POST | `/api/v1/diagnoses/:id/continue-in-chat` | Continue diagnosis in AI chat |

### Transcription API

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/ai/transcribe` | Transcribe audio recording (multipart) |

### Market Prices API

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/market-prices` | Any | List market prices |
| POST | `/api/v1/market-prices` | Officer/Admin | Create market price |
| PUT | `/api/v1/market-prices/:id` | Officer/Admin | Update market price |
| DELETE | `/api/v1/market-prices/:id` | Officer/Admin | Delete market price |

### Resources API

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/v1/resources` | Any | List reviewed resources |
| GET | `/api/v1/resources/:id` | Any | Get resource detail |

### Notifications API

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/notifications` | List user notifications |
| PATCH | `/api/v1/notifications/read-all` | Mark all as read |
| PATCH | `/api/v1/notifications/:id/read` | Mark notification as read |
| GET | `/api/v1/notifications/unread-count` | Get unread notification count |

### Officer API

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/officer/diagnoses` | Diagnosis queue (paginated, filterable) |
| GET | `/api/v1/officer/diagnoses/:id` | Diagnosis detail with reviews |
| POST | `/api/v1/officer/diagnoses/:id/claim` | Claim a diagnosis case |
| POST | `/api/v1/officer/diagnoses/:id/reviews` | Create review |
| PUT | `/api/v1/officer/diagnoses/:id/reviews/:reviewID` | Update review |

### Admin API

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/admin/users` | List all users |
| PATCH | `/api/v1/admin/users/:userId/role` | Update user role |
| PATCH | `/api/v1/admin/users/:userId/status` | Update user active status |
| GET | `/api/v1/admin/diagnoses` | List all diagnoses |
| GET | `/api/v1/admin/reviews` | List all reviews |
| GET | `/api/v1/admin/audit-logs` | List audit logs |

## Security Notes

- Groq API key is never exposed to the frontend
- AI output is HTML-escaped before rendering
- Rate limiting is applied to all API routes (stricter for diagnosis/transcription)
- Input validation is enforced on all endpoints
- Database errors are not returned to users
- Request IDs are logged for debugging
- Uploaded images are stored with random filenames, never as public static files
- Image paths are path-traversal protected
- Diagnosis images require ownership verification before serving
- Audio recordings are deleted after transcription, never retained by default
- Supabase service-role key is kept server-side only
- JWT tokens are HTTP-only cookies with configurable expiration
- Role-based access control enforced on all admin and officer endpoints

## Known Limitations

- AI advice can be incorrect and should be verified with a local agricultural extension officer
- AI image diagnosis may be incorrect — image quality affects results, and several conditions can cause similar symptoms
- Expert confirmation is required for serious or uncertain cases
- Krio language output and voice transcription are experimental and require community review
- Krio voice transcription requires manual review and correction before use
- No native Krio text-to-speech is available
- Audio recordings are not retained by default
- Seed agricultural material requires expert validation before use in production
- Weather depends on an external free provider (Open-Meteo)
- In-memory rate limiting is not suitable for multi-instance production
- Anonymous-data transfer depends on the browser retaining its anonymous cookie
