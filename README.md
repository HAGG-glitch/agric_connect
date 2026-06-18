# AgriConnect AI

Agricultural advisory web application for smallholder farmers in Sierra Leone.

## Implemented Scope

- **Phase 1** — Agricultural AI Chat (Gin web server, PostgreSQL, Groq integration, English/Krio responses, conversation management, streaming)
- **Phase 2** — Agricultural Knowledge Retrieval (PostgreSQL document storage, crop/topic detection, context retrieval)
- **Phase 3** — Weather Intelligence (Sierra Leone district mapping, Open-Meteo integration, weather caching, weather-aware AI responses)
- **Phase 4** — AI Crop Image Diagnosis (image upload with drag-and-drop, crop/part/symptom form, Groq vision integration, structured AI result, diagnosis history, continue-in-chat)
- **Phase 5** — Voice Recording & Transcription (browser MediaRecorder, Groq Whisper transcription, Krio experimental warning, editable transcript insertion into chat)

## Architecture

```
cmd/server/main.go          — Application entry point
internal/config/             — Environment-based configuration
internal/database/           — PostgreSQL connection and migrations
internal/models/             — GORM models (conversation, message, document, weather cache)
internal/repositories/       — Data access layer
internal/services/           — Business logic (chat, knowledge, weather)
internal/diagnosis/          — Crop diagnosis service, repository, validation, schemas
internal/transcription/      — Audio transcription service, validation, schemas
internal/storage/            — Object storage abstraction (local + Supabase)
internal/ai/                 — Groq client, prompt builder, response orchestrator, vision, transcription
internal/weather/            — Open-Meteo client, district coordinates
internal/handlers/           — HTTP handlers (page, chat, conversation, weather, health, diagnosis, transcription)
internal/middleware/         — Request ID, recovery, anonymous user, rate limiting
web/templates/               — Gin HTML templates (layouts, pages, partials)
web/static/                  — CSS (Tailwind) and JavaScript
migrations/                  — SQL migration files
seed/                        — Agricultural knowledge base seed data
scripts/                     — Utility programs (seed.go)
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

Migrations run automatically on application startup. To run them manually:

```bash
# Up migrations
go run ./cmd/server
```

## Seeding Agricultural Documents

```bash
go run ./scripts/seed.go
```

This loads agricultural knowledge for rice, cassava, maize, and groundnut from `seed/agricultural_documents.json`.

## Tailwind Compilation

```bash
# One-time build
npx tailwindcss -i ./web/static/css/input.css -o ./web/static/css/app.css --minify

# Watch mode
npx tailwindcss -i ./web/static/css/input.css -o ./web/static/css/app.css --watch
```

## Groq Configuration

Set `GROQ_API_KEY` in your `.env` file. The application uses `llama-3.1-8b-instant` by default but any Groq-compatible model can be configured via `GROQ_CHAT_MODEL`.

Without a configured key, the UI shows a clear configuration error and the AI features are unavailable.

### Vision Model (Phase 4)

Configured via `GROQ_VISION_MODEL` (default: `llama-3.2-11b-vision-preview`). Used for crop image diagnosis. If the configured model cannot process images, the diagnosis is marked as failed.

### Transcription Model (Phase 5)

Configured via `GROQ_TRANSCRIPTION_MODEL` (default: `whisper-large-v3`). Used for voice recording transcription. Supports English and Krio language hints.

## Storage Configuration

### Local Storage (Development)

Default driver. Files stored under `LOCAL_UPLOAD_DIR` (default: `./data/uploads`). Images are served only through an ownership-checked HTTP handler, not as public static files.

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

## API Routes

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check (database connectivity) |
| GET | `/assistant` | Main assistant page |
| GET | `/diagnose` | Crop diagnosis form |
| GET | `/diagnoses` | Diagnosis history |
| GET | `/diagnoses/:id` | Diagnosis detail |
| GET | `/login` | Login page |
| GET | `/register` | Registration page |
| GET | `/officer` | Officer dashboard |
| GET | `/admin/users` | Admin user management |
| POST | `/api/v1/auth/register` | Register a new user |
| POST | `/api/v1/auth/login` | Login |
| POST | `/api/v1/auth/refresh` | Refresh access token |
| POST | `/api/v1/auth/logout` | Logout |
| GET | `/api/v1/auth/me` | Get current user profile |
| POST | `/api/v1/conversations` | Create conversation |
| GET | `/api/v1/conversations` | List conversations |
| GET | `/api/v1/conversations/:id` | Get conversation with messages |
| DELETE | `/api/v1/conversations/:id` | Delete conversation |
| POST | `/api/v1/conversations/:id/messages` | Send message (non-streaming) |
| POST | `/api/v1/conversations/:id/messages/stream` | Send message (SSE streaming) |
| GET | `/api/v1/weather?district=Bo` | Get weather for a district |
| POST | `/api/v1/diagnoses` | Submit a crop diagnosis (multipart) |
| GET | `/api/v1/diagnoses` | List diagnoses (paginated) |
| GET | `/api/v1/diagnoses/:id` | Get diagnosis detail |
| DELETE | `/api/v1/diagnoses/:id` | Delete diagnosis |
| GET | `/api/v1/diagnoses/:id/image` | Serve diagnosis image (ownership-checked) |
| POST | `/api/v1/diagnoses/:id/continue-in-chat` | Continue diagnosis in AI chat |
| POST | `/api/v1/ai/transcribe` | Transcribe audio recording (multipart) |
| GET | `/api/v1/officer/diagnoses` | Officer diagnosis queue (paginated, filterable) |
| GET | `/api/v1/officer/diagnoses/:id` | Officer get diagnosis detail with reviews |
| POST | `/api/v1/officer/diagnoses/:id/reviews` | Create review for a diagnosis |
| PUT | `/api/v1/officer/diagnoses/:id/reviews/:reviewID` | Update an existing review |
| GET | `/api/v1/admin/users` | List all users |
| PATCH | `/api/v1/admin/users/:userId/role` | Update user role |
| PATCH | `/api/v1/admin/users/:userId/status` | Update user active status |
| GET | `/api/v1/notifications` | List user notifications |
| PATCH | `/api/v1/notifications/:id/read` | Mark notification as read |

## Anonymous User Behavior

Users are identified by an HTTP-only cookie (`agriconnect_user`) set on first visit. No registration is required. Conversation and diagnosis ownership is enforced server-side. Diagnosis images are only accessible by the anonymous owner.

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
