# AgriConnect AI — Final Completion Prompt

You are a senior Go full-stack engineer working inside the existing AgriConnect AI repository.

Read these files first:

```text
RECOVERY_REPORT.md
PHASE_4_5_REPORT.md
README.md
```

The current project already includes:

- Gin backend
- PostgreSQL and GORM
- Go HTML templates
- Tailwind CSS
- HTMX and vanilla JavaScript
- Groq agricultural chat
- English and Krio text support
- Agricultural knowledge retrieval
- Open-Meteo weather integration
- Crop-image diagnosis
- Voice recording and Groq transcription
- Anonymous browser identity
- Local image storage
- A partial Supabase storage implementation
- Docker and Docker Compose
- Existing automated tests

The current report lists these remaining limitations:

1. Diagnosis and transcription handlers lack HTTP-level tests.
2. Supabase image serving returns 503 when `localStorage` is nil.
3. Uploaded images have byte-size validation but no minimum dimensions or maximum pixel-count checks.
4. `docker compose build` and `docker compose up` were not executed.
5. Krio voice transcription is experimental.
6. Full authentication is missing.
7. The extension-officer review workflow is missing.

Your task is to complete every technically actionable item, preserve all working Phase 1–5 functionality, execute the required verification commands, and update the project reports honestly.

Do not stop after auditing, planning, or explaining.

Implement, connect, test, run, and document the work.

---

## 1. Non-Negotiable Rules

1. Inspect the full repository before editing.
2. Preserve all existing chat, knowledge, weather, diagnosis, transcription, conversation, and storage behavior.
3. Do not rewrite the frontend in React.
4. Do not create another application entry point.
5. Do not duplicate the router, database connection, Groq client, storage abstraction, or configuration loader.
6. Keep handlers thin, business logic in services, and database access in repositories.
7. Use interfaces for external providers and mocks in tests.
8. Do not expose Groq or Supabase credentials to HTML or JavaScript.
9. Do not place real credentials in `.env.example`.
10. Do not weaken validation to make tests pass.
11. Do not remove existing tests.
12. Do not make the Supabase bucket public.
13. Do not claim Krio transcription is fully accurate.
14. Do not claim AI diagnosis is scientifically confirmed.
15. Do not invent pesticide dosage.
16. Do not claim Docker success unless Docker commands were actually executed successfully.
17. Update `PHASE_4_5_REPORT.md`.
18. Create `FINAL_COMPLETION_REPORT.md`.

---

## 2. Baseline Audit

Before making changes, run:

```bash
git status
go test ./...
go vet ./...
go build ./cmd/server
docker compose config
```

Record the baseline results in `FINAL_COMPLETION_REPORT.md`.

Also document:

- Current storage-driver selection logic
- Current Supabase implementation state
- Current diagnosis image-serving flow
- Current authentication behavior
- Current anonymous ownership behavior
- Current officer/admin capabilities
- Whether Docker is available
- Whether `.env.example` contains real secrets

If real credentials are present in `.env.example`:

1. Remove them immediately.
2. Replace them with empty placeholders.
3. Check Git history.
4. State that the key must be rotated if it was committed, pushed, shared, uploaded, or logged.
5. Never print the secret in logs or reports.

---

## 3. Correct Environment Configuration

The committed `.env.example` must contain placeholders only.

Use this structure:

```env
APP_ENV=development
APP_PORT=8081
APP_URL=http://localhost:8081

DATABASE_URL=postgresql://postgres:postgres@db:5432/agriconnect?sslmode=disable

GROQ_API_KEY=
GROQ_BASE_URL=https://api.groq.com/openai/v1
GROQ_CHAT_MODEL=
GROQ_VISION_MODEL=
GROQ_TRANSCRIPTION_MODEL=

STORAGE_DRIVER=local
LOCAL_UPLOAD_DIR=./data/uploads

SUPABASE_URL=
SUPABASE_SECRET_KEY=
SUPABASE_SERVICE_ROLE_KEY=
SUPABASE_STORAGE_BUCKET=crop-diagnosis-images

MAX_IMAGE_SIZE_MB=5
MAX_AUDIO_SIZE_MB=10
MIN_IMAGE_WIDTH=256
MIN_IMAGE_HEIGHT=256
MAX_IMAGE_PIXELS=25000000

JWT_ACCESS_SECRET=
JWT_REFRESH_SECRET=
JWT_ACCESS_DURATION=15m
JWT_REFRESH_DURATION=168h

COOKIE_SECURE=false
COOKIE_DOMAIN=
COOKIE_SAME_SITE=lax
```

Requirements:

- Real values belong only in `.env` or deployment environment variables.
- `.gitignore` must ignore `.env`, `.env.local`, and `.env.production`.
- Prefer `SUPABASE_SECRET_KEY`.
- Support `SUPABASE_SERVICE_ROLE_KEY` only as a legacy fallback.
- Never log either key.

---

## 4. Complete Supabase Storage

The current report says Supabase image serving returns 503 when `localStorage` is nil.

Fix the design so handlers depend only on the generic storage abstraction.

### 4.1 Storage interface

Preserve or extend the existing interface to support:

```go
type ObjectStorage interface {
    Save(ctx context.Context, input SaveObjectInput) (StoredObject, error)
    Delete(ctx context.Context, path string) error
    SignedURL(ctx context.Context, path string, expiry time.Duration) (string, error)
}
```

If local streaming needs another capability, use a separate interface such as:

```go
type ObjectReader interface {
    Open(ctx context.Context, path string) (io.ReadCloser, ObjectMetadata, error)
}
```

Do not type-cast the handler to `*storage.LocalStorage`.

### 4.2 Supabase implementation

Fully implement `internal/storage/supabase.go`.

Use:

```text
SUPABASE_URL
SUPABASE_SECRET_KEY
SUPABASE_STORAGE_BUCKET
```

Implement:

#### Save

- Context-aware request
- Validated image bytes only
- Backend-generated object path
- Correct content type
- No overwriting
- Typed errors
- Bounded response reading
- No secret logging

#### SignedURL

- Private signed URL
- Default expiry around five minutes
- Validate non-empty object path
- Never store signed URLs in PostgreSQL

#### Delete

- Delete the exact object
- Validate the object path
- Handle non-2xx responses
- Never accept bucket names from browser input

Use the new Supabase secret key through the `apikey` header.

Do not treat `sb_secret_...` as a legacy bearer JWT.

### 4.3 Image-serving route

Fix:

```text
GET /api/v1/diagnoses/:id/image
```

Required flow:

1. Resolve current authenticated or anonymous user.
2. Load diagnosis.
3. Verify owner, assigned officer, or admin authorization.
4. Read the stored image path.
5. For local storage, stream the file safely.
6. For Supabase, generate a short-lived signed URL and redirect or return it consistently.
7. Set safe headers.
8. Do not return 503 merely because local storage is not selected.

For local streaming, set:

```text
Content-Type
X-Content-Type-Options: nosniff
```

Prevent path traversal.

### 4.4 Supabase and storage tests

Add mocked tests for:

- Upload success
- Signed URL success
- Delete success
- Upload non-2xx response
- Signed URL non-2xx response
- Delete non-2xx response
- Malformed provider response
- Empty object path
- Context cancellation
- Local image streaming
- Unauthorized image access
- Missing diagnosis image
- Supabase failure
- Diagnosis deletion invoking storage deletion

No live Supabase calls in normal unit tests.

---

## 5. Add Missing Diagnosis Handler Tests

Use Gin HTTP test utilities and mocked services.

Add tests for at least:

1. Successful multipart diagnosis submission
2. Missing image
3. Unsupported image type
4. Oversized body
5. Invalid district
6. Unauthorized diagnosis read
7. Unauthorized diagnosis delete
8. Image-serving ownership check
9. Supabase signed-URL redirect or response
10. Pagination validation
11. Service/provider failure returns safe public error
12. Continue-in-chat ownership check

Verify:

- Status codes
- JSON structure
- Public-safe error messages
- Ownership enforcement
- Request limits

Do not call live Groq, Supabase, or Open-Meteo services.

---

## 6. Add Missing Transcription Handler Tests

Add HTTP-level tests for:

1. Successful multipart transcription
2. Missing audio
3. Unsupported audio type
4. Oversized request
5. Invalid language hint
6. Provider failure
7. Empty transcript
8. Krio requires confirmation
9. Temporary-file cleanup path
10. Safe public error response

Verify that:

- Transcript is not automatically sent
- Krio sets `requires_confirmation=true`
- Internal provider errors do not leak
- Audio is not permanently stored

---

## 7. Add Image Dimension and Pixel Validation

Add configuration:

```env
MIN_IMAGE_WIDTH=256
MIN_IMAGE_HEIGHT=256
MAX_IMAGE_PIXELS=25000000
```

Server-side validation must:

1. Decode image configuration safely.
2. Reject width below minimum.
3. Reject height below minimum.
4. Reject zero dimensions.
5. Compute pixel count with overflow-safe arithmetic.
6. Reject pixel count above maximum.
7. Keep existing byte-size validation.
8. Keep JPEG, PNG, and WebP signature validation.
9. Reject SVG.
10. Return clear validation messages.

Add tests for:

- Valid dimensions
- Width too small
- Height too small
- Pixel count too large
- Malformed image
- Signature mismatch
- Overflow protection

Update the diagnosis form to say:

```text
Use a clear image of at least 256 × 256 pixels.
```

---

## 8. Krio Voice Transcription Safeguards

This limitation cannot be honestly marked as fully solved without real evaluation.

Implement safeguards instead:

1. Keep the visible notice: `Krio voice transcription is experimental.`
2. Always set `requires_confirmation=true` for Krio.
3. Never automatically send the transcript.
4. Require user review or editing.
5. Provide retry recording.
6. Preserve normal text input.
7. Do not retain raw audio by default.
8. Do not log transcript contents in normal logs.
9. Add optional feedback controls:
   - Accurate
   - Needs correction
10. Store feedback only after explicit user action.

If feedback is persisted, add a migration similar to:

```sql
CREATE TABLE transcription_feedback (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    language_hint VARCHAR(20) NOT NULL,
    rating VARCHAR(30) NOT NULL,
    correction_length INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Do not store raw audio or duplicate transcript content in this feedback table.

Document that real Krio accuracy must be evaluated using consented Sierra Leonean speech samples.

---

## 9. Implement Full Authentication

Replace anonymous-only access with real authentication while preserving anonymous-data migration.

Use:

- Phone number
- Password
- JWT access token
- Refresh token
- HTTP-only cookies
- Role-based authorization

Roles:

```text
farmer
officer
admin
```

### 9.1 Database migrations

Add `users`:

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY,
    full_name VARCHAR(200) NOT NULL,
    phone_number VARCHAR(30) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    district VARCHAR(100),
    preferred_language VARCHAR(20) NOT NULL DEFAULT 'english',
    role VARCHAR(20) NOT NULL DEFAULT 'farmer',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Add `refresh_tokens`:

```sql
CREATE TABLE refresh_tokens (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Add indexes.

### 9.2 Registration

Routes:

```text
GET  /register
POST /api/v1/auth/register
```

Fields:

- Full name
- Phone number
- District
- Preferred language
- Password

Requirements:

- Normalize phone number consistently
- Reject duplicate phone numbers
- Hash password with bcrypt
- Never store plain passwords
- Default role to farmer
- Issue access and refresh cookies
- Return safe errors

### 9.3 Login and session routes

Create:

```text
GET  /login
POST /api/v1/auth/login
POST /api/v1/auth/refresh
POST /api/v1/auth/logout
GET  /api/v1/auth/me
```

Requirements:

- Verify bcrypt hash
- Reject inactive users
- Rotate refresh tokens
- Store only refresh-token hashes
- Revoke on logout
- Use HTTP-only cookies
- Use Secure cookies in production
- Apply configured SameSite mode
- Do not expose JWTs to browser JavaScript unless strictly necessary

### 9.4 Authentication middleware

Create middleware that:

1. Reads access token from cookie.
2. Validates signature and expiry.
3. Loads user and role.
4. Rejects inactive users.
5. Places user ID, role, district, and language in Gin context.

Add role middleware:

```go
RequireRole("officer", "admin")
RequireRole("admin")
```

### 9.5 Anonymous-data migration

Existing anonymous users may own conversations and diagnoses.

After registration or login:

- Read anonymous cookie UUID
- Transfer conversations and diagnoses to authenticated user ID
- Transfer transcription feedback if implemented
- Use a database transaction
- Do not claim records already owned by another authenticated account
- Rotate or clear anonymous identity after transfer
- Add tests

### 9.6 Authorization rules

Farmer:

- Own conversations and diagnoses only
- Submit diagnoses
- Use chat and transcription
- Read officer review of their diagnoses

Officer:

- Assigned-district diagnosis queue
- Review district diagnoses
- Add comments and recommendations
- Cannot change admin roles

Admin:

- Manage users and roles
- Access all diagnoses and reviews
- View analytics

---

## 10. Implement Extension-Officer Review Workflow

### 10.1 Database migration

Add:

```sql
CREATE TABLE diagnosis_reviews (
    id UUID PRIMARY KEY,
    diagnosis_id UUID NOT NULL REFERENCES crop_diagnoses(id) ON DELETE CASCADE,
    officer_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    review_status VARCHAR(30) NOT NULL,
    confirmed_condition VARCHAR(255),
    officer_comment TEXT,
    recommendation TEXT,
    urgency VARCHAR(20),
    requires_field_visit BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Statuses:

```text
pending
in_review
confirmed
needs_more_information
field_visit_required
closed
```

Add indexes.

Prevent uncontrolled duplicate active reviews.

### 10.2 Diagnosis workflow statuses

Normalize statuses:

```text
processing
ai_completed
awaiting_review
under_review
reviewed
failed
```

Keep AI output separate from human review.

### 10.3 Officer routes

Create:

```text
GET  /officer
GET  /officer/diagnoses
GET  /officer/diagnoses/:id

GET  /api/v1/officer/diagnoses
GET  /api/v1/officer/diagnoses/:id
POST /api/v1/officer/diagnoses/:id/reviews
PUT  /api/v1/officer/diagnoses/:id/reviews/:reviewID
```

Requirements:

- Officer or admin role required
- Officer sees assigned district only
- Admin sees all
- Officer can claim a case
- Officer can request more information
- Officer can confirm or revise condition
- Officer can recommend field visit
- Farmer sees submitted review
- Actor and timestamp recorded

### 10.4 Farmer diagnosis display

Display two separate sections:

#### AI assessment

- Probable condition
- Confidence
- Recommendations
- Disclaimer

#### Extension-officer review

- Review status
- Confirmed condition
- Officer recommendation
- Field visit requirement
- Reviewed date

Make the distinction visually clear.

### 10.5 Admin user management

Create:

```text
GET   /admin/users
PATCH /api/v1/admin/users/:id/role
PATCH /api/v1/admin/users/:id/status
```

Requirements:

- Admin only
- Validate roles
- Prevent demoting the final active admin
- Record audit events
- Prevent self-elevation through profile editing

### 10.6 Audit logs

Add:

```sql
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY,
    actor_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    entity_type VARCHAR(100) NOT NULL,
    entity_id UUID,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Record:

- Role changes
- User status changes
- Diagnosis claims
- Review creation and update
- Diagnosis deletion
- Storage deletion failures

Never store secrets, passwords, raw audio, or image data in audit metadata.

---

## 11. Notifications

Add a simple notification table if one does not exist:

```sql
CREATE TABLE notifications (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    message TEXT NOT NULL,
    notification_type VARCHAR(50) NOT NULL,
    is_read BOOLEAN NOT NULL DEFAULT FALSE,
    entity_type VARCHAR(100),
    entity_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Create notifications when:

- Officer starts review
- Officer requests more information
- Review is completed
- Field visit is recommended

Routes:

```text
GET   /api/v1/notifications
PATCH /api/v1/notifications/:id/read
```

Verify ownership.

---

## 12. UI Work

Use existing Go templates, Tailwind, HTMX, and JavaScript.

Do not introduce React.

Add:

### Public

- `/login`
- `/register`

### Farmer

- Profile menu
- Logout
- Review status on diagnosis detail
- Officer review section
- Notifications

### Officer

- Dashboard
- Pending diagnosis queue
- Filters by crop, urgency, and status
- Secure diagnosis detail
- AI result panel
- Farmer symptom information
- Review form
- Field-visit option

### Admin

- User list
- Role selector
- Active/inactive control
- Counts for farmers, officers, diagnoses, and pending reviews

Use Lucide icons and the current green design system.

---

## 13. Authentication and Review Tests

Add HTTP-level tests for authentication:

- Registration success
- Duplicate phone
- Invalid password
- Login success
- Invalid password
- Inactive user
- Refresh success
- Refresh rotation
- Logout revocation
- `/me`
- Cookie attributes
- Anonymous-data transfer

Add role tests:

- Farmer blocked from officer routes
- Officer blocked from admin routes
- Admin allowed
- Inactive user blocked

Add officer review tests:

- Officer sees assigned district queue
- Officer blocked from unrelated district
- Admin sees all
- Claim case
- Create review
- Update review
- Farmer reads completed review
- Farmer cannot create review
- Duplicate active review prevented

No live external provider calls.

---

## 14. Docker Build and Runtime Verification

If Docker is available, execute:

```bash
docker compose down -v
docker compose build --no-cache
docker compose up -d
docker compose ps
docker compose logs --no-color --tail=200 app
```

Then verify:

```bash
curl -i http://localhost:8081/health
curl -i http://localhost:8081/login
curl -i http://localhost:8081/register
curl -i http://localhost:8081/assistant
curl -i http://localhost:8081/diagnose
```

Also verify migrations applied.

Run smoke tests for:

- Registration
- Login
- Conversation creation
- Weather route
- Diagnosis page
- Transcription validation
- Officer authorization

Do not make paid Groq calls automatically unless explicit integration-test configuration is present.

If Docker is unavailable:

- Record the exact error
- Do not claim build or startup success
- Still run `docker compose config`

---

## 15. Optional Supabase Integration Test

Add an integration test or script disabled by default:

```env
RUN_SUPABASE_INTEGRATION_TESTS=false
```

When enabled with valid credentials:

1. Upload a generated small image under a test prefix.
2. Generate signed URL.
3. Confirm response.
4. Delete object.
5. Always attempt cleanup.

Never log the secret.

---

## 16. Required Verification

Run:

```bash
go mod tidy
go test ./...
go test -race ./...
go vet ./...
go build ./cmd/server
npm install
npx tailwindcss -i ./web/static/css/input.css -o ./web/static/css/app.css --minify
docker compose config
```

When Docker is available:

```bash
docker compose build --no-cache
docker compose up -d
docker compose ps
```

Fix all failures.

Do not ignore race-detector failures.

---

## 17. Reports

Update:

```text
PHASE_4_5_REPORT.md
```

Create:

```text
FINAL_COMPLETION_REPORT.md
```

The final report must include:

1. Baseline status
2. Supabase changes
3. Image-serving fix
4. Handler tests added
5. Image-dimension validation
6. Krio safeguards
7. Authentication design
8. Anonymous-data migration
9. Officer workflow
10. Admin functions
11. Notifications
12. Migrations
13. Routes
14. Files created
15. Files modified
16. Tests added
17. Commands executed
18. Exact command results
19. Docker results
20. Supabase test status
21. Remaining limitations

Be honest about anything not executed.

---

## 18. Limitations That Must Remain Documented

Even after implementation, keep these limitations:

1. AI crop diagnosis may be incorrect.
2. Image quality affects results.
3. Similar symptoms can have multiple causes.
4. Officer review does not replace laboratory testing.
5. Krio transcription remains experimental until evaluated with consented Sierra Leonean speech samples.
6. Native Krio text-to-speech is not implemented.
7. Weather depends on Open-Meteo availability.
8. Agricultural seed documents require expert validation.
9. In-memory rate limiting is not suitable for multi-instance production.
10. Anonymous-data transfer depends on the browser retaining its anonymous cookie.

Do not remove these limitations merely to make the project appear complete.

---

## 19. Acceptance Criteria

The task is complete only when:

### Supabase

- `Save`, `Delete`, and `SignedURL` are real implementations.
- No handler depends directly on `LocalStorage`.
- Supabase image serving no longer returns 503 because local storage is nil.
- Bucket remains private.
- `.env.example` contains no real credentials.

### Handler tests

- Diagnosis handler tests exist and pass.
- Transcription handler tests exist and pass.
- Authentication tests exist and pass.
- Officer-review tests exist and pass.
- Existing tests still pass.
- Race tests pass.

### Image validation

- Byte-size checks work.
- Signature checks work.
- Minimum dimensions work.
- Maximum pixel count works.
- Malformed images are rejected.

### Authentication

- Register works.
- Login works.
- Refresh rotation works.
- Logout revokes refresh token.
- HTTP-only cookies are used.
- Roles are enforced.
- Anonymous data transfers safely.

### Officer workflow

- Officer queue works.
- District authorization works.
- Human review is stored separately from AI output.
- Farmer can see completed review.
- Admin can manage roles safely.
- Audit logs are written.

### Docker

- Compose configuration is valid.
- Build and startup are executed when Docker is available.
- `/health` succeeds after startup.
- Failures are reported honestly.

---

## 20. Final Response Format

After completing the repository, respond with:

1. What was incomplete
2. What was implemented
3. Files created
4. Files modified
5. Migrations added
6. Routes added
7. Tests added
8. Commands executed
9. Test results
10. Build results
11. Docker results
12. Supabase results
13. Remaining limitations
14. Exact local run commands
15. Required production environment variables

Do not only recommend the next task.

Complete the repository.
