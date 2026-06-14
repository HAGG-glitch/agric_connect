# AgriConnect AI Repository Recovery and Phase 1–3 Completion Prompt

You are a senior Go full-stack engineer working inside an existing, partially generated AgriConnect AI repository.

The previous coding session ended before the project could be fully organized or packaged. Some files may be incomplete, duplicated, misplaced, disconnected, or only partially implemented.

Your task is to inspect the current repository, recover the useful work, organize every relevant file into the correct architecture, add all missing Phase 1–3 files, connect the scattered implementations, and leave the project compiling and runnable.

Do not merely explain what should be done. Perform the repository reorganization and implementation.

---

## 1. Current Product

AgriConnect AI is a responsive agricultural advisory web application for farmers in Sierra Leone.

The current milestone covers only:

### Phase 1 — Agricultural AI Chat

- Gin web server
- PostgreSQL connection
- Anonymous browser user identity
- Agricultural AI chat through Groq
- English and Krio response modes
- Conversation creation, history, loading, deletion, and persistence
- Streaming or reliable non-streaming fallback
- Responsive HTML interface

### Phase 2 — Agricultural Knowledge Retrieval

- Agricultural documents stored in PostgreSQL
- Initial content for rice, cassava, maize, and groundnut
- Crop and topic detection
- Relevant document retrieval
- Retrieved knowledge added to the Groq prompt
- Source metadata stored and shown where appropriate

### Phase 3 — Weather Intelligence

- Sierra Leone district coordinate mapping
- Open-Meteo integration
- Current weather and seven-day forecast
- PostgreSQL weather caching
- Weather-aware agricultural AI responses
- The AI must never invent current weather

Do not implement Phase 4 crop-image diagnosis or Phase 5 voice input in this task.

You may create clean interfaces and extension points for those later phases, but do not add incomplete image-diagnosis or audio code.

---

## 2. Technology Stack

Use the following stack:

### Backend

- Go
- Gin
- PostgreSQL
- GORM
- Groq API
- Open-Meteo API

### Frontend

- Go HTML templates
- Tailwind CSS
- HTMX where useful
- Vanilla JavaScript
- Lucide icons

### Infrastructure

- Docker
- Docker Compose
- SQL migrations
- Environment-based configuration

Do not migrate the project to React, Next.js, Vue, FastAPI, or another backend framework.

Node.js may be used only for Tailwind CSS compilation and frontend development tooling.

---

## 3. Critical Recovery Rules

Follow these rules before changing the repository:

1. Inspect the entire repository recursively.
2. Read all existing Go, HTML, CSS, JavaScript, SQL, JSON, YAML, Docker, environment, and Markdown files.
3. Run `git status` when the repository uses Git.
4. Do not delete useful existing work.
5. Do not overwrite a more complete implementation with a smaller placeholder.
6. Identify duplicate files and determine which implementation is most complete.
7. Merge useful logic when two scattered files implement different parts of the same feature.
8. Update package names, imports, template paths, static asset paths, route registrations, dependency injection, and configuration after moving files.
9. Do not leave disconnected files that are never imported, registered, rendered, or called.
10. Do not leave two competing application entry points.
11. Do not leave duplicate route registrations.
12. Do not leave hard-coded secrets.
13. Do not expose the Groq API key to HTML or browser JavaScript.
14. Do not silently ignore compile errors, migration errors, template errors, or failed tests.
15. Do not stop after reorganizing folders. Complete missing Phase 1–3 behavior.
16. Preserve the current visual work where it is useful, but convert it into reusable Gin templates and static assets.
17. If an existing file cannot be integrated safely, document it in `RECOVERY_REPORT.md`.
18. Do not create fake production data or claim expert validation for sample agricultural content.

Before editing, create `RECOVERY_REPORT.md` and record:

- Existing repository structure
- Files discovered
- Duplicate or conflicting implementations
- Missing Phase 1–3 components
- Planned moves and merges
- Important assumptions

Update the report after the work is complete.

---

## 4. Target Repository Structure

Organize the project toward this structure:

```text
agriconnect-ai/
├── cmd/
│   └── server/
│       └── main.go
│
├── internal/
│   ├── config/
│   │   └── config.go
│   │
│   ├── database/
│   │   ├── postgres.go
│   │   └── migrations.go
│   │
│   ├── models/
│   │   ├── conversation.go
│   │   ├── message.go
│   │   ├── agricultural_document.go
│   │   └── weather_cache.go
│   │
│   ├── repositories/
│   │   ├── conversation_repository.go
│   │   ├── message_repository.go
│   │   ├── knowledge_repository.go
│   │   └── weather_repository.go
│   │
│   ├── services/
│   │   ├── chat_service.go
│   │   ├── conversation_service.go
│   │   ├── knowledge_service.go
│   │   └── weather_service.go
│   │
│   ├── ai/
│   │   ├── client.go
│   │   ├── assistant.go
│   │   ├── orchestrator.go
│   │   ├── schemas.go
│   │   └── prompts/
│   │       ├── agricultural_assistant.txt
│   │       └── krio_rules.txt
│   │
│   ├── weather/
│   │   ├── client.go
│   │   └── districts.go
│   │
│   ├── handlers/
│   │   ├── page_handler.go
│   │   ├── chat_handler.go
│   │   ├── conversation_handler.go
│   │   ├── weather_handler.go
│   │   └── health_handler.go
│   │
│   ├── middleware/
│   │   ├── anonymous_user.go
│   │   ├── request_id.go
│   │   ├── recovery.go
│   │   └── rate_limit.go
│   │
│   └── validation/
│       └── validation.go
│
├── web/
│   ├── templates/
│   │   ├── layouts/
│   │   │   └── app.html
│   │   ├── pages/
│   │   │   └── assistant.html
│   │   └── partials/
│   │       ├── sidebar.html
│   │       ├── chat_message.html
│   │       ├── chat_history.html
│   │       ├── weather_card.html
│   │       ├── loading_indicator.html
│   │       └── error_message.html
│   │
│   └── static/
│       ├── css/
│       │   ├── input.css
│       │   └── app.css
│       └── js/
│           ├── app.js
│           └── assistant.js
│
├── migrations/
│   ├── 000001_create_ai_tables.up.sql
│   └── 000001_create_ai_tables.down.sql
│
├── seed/
│   └── agricultural_documents.json
│
├── tests/
│   ├── chat_service_test.go
│   ├── knowledge_service_test.go
│   └── weather_service_test.go
│
├── scripts/
│   └── seed.go
│
├── Dockerfile
├── docker-compose.yml
├── Makefile
├── package.json
├── tailwind.config.js
├── go.mod
├── go.sum
├── .env.example
├── .gitignore
├── README.md
└── RECOVERY_REPORT.md
```

This is the target architecture, not a command to create empty files.

If an existing structure is already clean and equivalent, retain it and document the mapping instead of moving files unnecessarily.

---

## 5. Repository Inspection Procedure

Perform the following steps in order.

### Step 1: Inventory

List all existing files and classify each one as:

- Complete and correctly placed
- Complete but misplaced
- Incomplete but reusable
- Duplicate
- Obsolete
- Missing dependency
- Missing required Phase 1–3 implementation

Record the classification in `RECOVERY_REPORT.md`.

### Step 2: Find Entry Points

Locate every possible application entry point, including:

- `main.go`
- `cmd/**/main.go`
- old server files
- temporary prototype servers
- duplicated router setup

Choose one production entry point:

```text
cmd/server/main.go
```

Connect all application dependencies through this entry point.

Do not keep multiple active servers.

### Step 3: Trace Connections

For every major feature, trace the complete path:

```text
Route
→ Handler
→ Service
→ Repository or Integration Client
→ Database or External API
→ Response
→ Frontend rendering
```

Repair all broken paths.

### Step 4: Move and Merge

Move misplaced files into their correct folders.

When duplicate implementations exist:

- Compare completeness
- Preserve useful logic
- Merge carefully
- Remove duplicate registrations
- Update imports
- Update tests
- Document the decision

### Step 5: Compile Early

After the initial organization, run:

```bash
go mod tidy
go test ./...
go vet ./...
go build ./cmd/server
```

Fix structural and compile problems before adding more functionality.

---

## 6. Required Configuration

Create or repair `internal/config/config.go`.

Load configuration from environment variables.

Required variables:

```env
APP_ENV=development
APP_PORT=8080
APP_URL=http://localhost:8080

DATABASE_URL=postgresql://postgres:postgres@db:5432/agriconnect?sslmode=disable

GROQ_API_KEY=
GROQ_BASE_URL=https://api.groq.com/openai/v1
GROQ_CHAT_MODEL=
GROQ_REQUEST_TIMEOUT_SECONDS=60

OPEN_METEO_BASE_URL=https://api.open-meteo.com/v1
WEATHER_CACHE_MINUTES=20

COOKIE_SECURE=false
COOKIE_DOMAIN=
COOKIE_SAME_SITE=lax

MAX_MESSAGE_LENGTH=4000
MAX_CONTEXT_MESSAGES=12
RATE_LIMIT_REQUESTS_PER_MINUTE=20
```

Requirements:

- Validate required values.
- Parse integer, boolean, and duration values safely.
- Never log secret values.
- Development mode may start without a Groq key.
- When Groq is unavailable, the UI must show a clear configuration error.
- Production mode must fail fast when required secrets are missing.

Create or repair `.env.example`.

Do not create a committed `.env` containing real secrets.

---

## 7. Database and Migrations

Use PostgreSQL UUID primary keys.

Create or repair migrations for these tables.

### `ai_conversations`

```sql
CREATE TABLE ai_conversations (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,
    title VARCHAR(200) NOT NULL,
    preferred_language VARCHAR(20) NOT NULL DEFAULT 'english',
    district VARCHAR(100),
    crop VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### `ai_messages`

```sql
CREATE TABLE ai_messages (
    id UUID PRIMARY KEY,
    conversation_id UUID NOT NULL
        REFERENCES ai_conversations(id)
        ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL,
    content TEXT NOT NULL,
    language VARCHAR(20),
    model VARCHAR(150),
    input_tokens INTEGER,
    output_tokens INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Allowed roles:

- user
- assistant
- system
- tool

### `agricultural_documents`

```sql
CREATE TABLE agricultural_documents (
    id UUID PRIMARY KEY,
    crop VARCHAR(100),
    category VARCHAR(100) NOT NULL,
    title VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    language VARCHAR(20) NOT NULL DEFAULT 'english',
    source TEXT,
    reviewed BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### `weather_cache`

```sql
CREATE TABLE weather_cache (
    district VARCHAR(100) PRIMARY KEY,
    response JSONB NOT NULL,
    fetched_at TIMESTAMPTZ NOT NULL
);
```

Create useful indexes for:

- `ai_conversations.user_id`
- `ai_conversations.updated_at`
- `ai_messages.conversation_id`
- `agricultural_documents.crop`
- `agricultural_documents.category`

Requirements:

- Up and down migrations must both work.
- Models must match the database.
- Repositories must use context-aware database operations.
- The application must not silently auto-create a conflicting schema.
- Use either migrations as the source of truth or a carefully controlled migration runner.
- Document the chosen approach.

---

## 8. Anonymous Browser User

Full registration and JWT authentication are not part of Phase 1–3.

Implement anonymous browser identity.

Requirements:

1. When a browser first opens the application, generate a UUID.
2. Store it in an HTTP-only cookie named:

```text
agriconnect_user
```

3. Apply appropriate SameSite settings.
4. Use `Secure` in production.
5. Use the UUID as the conversation owner.
6. Verify ownership on every conversation read, update, message, and delete operation.
7. Never accept a user ID supplied by frontend JavaScript.
8. Structure code so the anonymous user can later be replaced by authenticated user middleware.

---

## 9. Phase 1 — Agricultural AI Chat

Complete the entire chat flow.

### Required routes

```text
GET    /assistant
GET    /health

POST   /api/v1/conversations
GET    /api/v1/conversations
GET    /api/v1/conversations/:id
DELETE /api/v1/conversations/:id

POST   /api/v1/conversations/:id/messages
POST   /api/v1/conversations/:id/messages/stream
```

### Conversation creation

Request:

```json
{
  "preferred_language": "krio",
  "district": "Bo",
  "crop": "Cassava"
}
```

Validate:

- `preferred_language` must be `english` or `krio`
- district must be supported
- crop must be either empty or in the supported crop list

### Message processing

The handler must:

1. Read the anonymous user from middleware.
2. Verify conversation ownership.
3. Validate message length.
4. Save the user message.
5. Retrieve relevant agricultural knowledge.
6. Determine whether weather data is required.
7. Retrieve weather safely when needed.
8. Build the Groq request.
9. Include only a limited number of recent messages.
10. Call Groq.
11. Save the assistant response.
12. Update conversation title after the first meaningful question.
13. Update `updated_at`.
14. Return the assistant message.
15. Preserve the saved user message when Groq fails.

### Message limits

Use:

```text
Minimum length: 2 characters
Maximum length: 4,000 characters
Maximum context messages: configurable, default 12
```

### Groq client

Create a reusable Groq client with:

- Environment-configured base URL
- Environment-configured model
- Request timeout
- Context cancellation
- Error status decoding
- Streaming support
- Token usage parsing where available
- No secret logging
- Testable interface

Do not hard-code a Groq model name.

### System prompt

Create:

```text
internal/ai/prompts/agricultural_assistant.txt
```

The prompt must establish:

- Agricultural focus
- Sierra Leone context
- English and Krio response behavior
- No invented current weather
- No invented current market prices
- No invented pesticide dosage
- Chemical safety guidance
- Uncertainty disclosure
- Escalation to an extension officer
- Simple farmer-friendly explanations

Create:

```text
internal/ai/prompts/krio_rules.txt
```

Include starter terms such as:

```text
crop disease = sik wey de affect di plant
symptoms = sign dem wey di plant de show
fertiliser = plant food
pest = bad insect or animal wey de spoil di crop
treatment = wetin yu kin do fo control di problem
prevention = wetin yu kin do fo stop di problem
harvest = time fo pull or gather di crop
soil = gron
rainfall = ren
```

Clearly document that the Krio glossary requires community and language-expert review.

---

## 10. Phase 2 — Agricultural Knowledge Retrieval

Create or repair the agricultural knowledge system.

Seed sample records for:

- Rice
- Cassava
- Maize
- Groundnut

Include categories:

- Planting
- Disease
- Pests
- Soil
- Fertiliser
- Harvesting
- Storage

Every seed record must contain:

- Crop
- Category
- Title
- Content
- Language
- Source
- Reviewed flag

Sample content must state that it is academic prototype material requiring expert validation.

### Retrieval behavior

The knowledge service must:

1. Normalize the farmer's question.
2. Detect crop names using explicit aliases and keyword matching.
3. Detect likely categories.
4. Search PostgreSQL.
5. Rank exact crop and category matches higher.
6. Limit the number and total length of documents.
7. Return structured context and source metadata.
8. Avoid unrelated documents.
9. Add the selected context to the AI request.
10. Keep retrieval behind a testable service interface.

Do not add vector search, embeddings, or `pgvector` in this phase.

---

## 11. Phase 3 — Weather Intelligence

Support these Sierra Leone districts:

- Bo
- Bombali
- Bonthe
- Falaba
- Kailahun
- Kambia
- Karene
- Kenema
- Koinadugu
- Kono
- Moyamba
- Port Loko
- Pujehun
- Tonkolili
- Western Area Urban
- Western Area Rural

Create:

```go
type DistrictCoordinates struct {
    Name      string
    Latitude  float64
    Longitude float64
}
```

Keep the mapping in:

```text
internal/weather/districts.go
```

Use reasonable central coordinates and document the source or approximation.

### Weather route

```text
GET /api/v1/weather?district=Bo
```

Return:

- District
- Current temperature
- Relative humidity
- Current precipitation
- Wind speed
- Seven-day daily minimum temperature
- Seven-day daily maximum temperature
- Rain probability
- Total precipitation
- Fetch time
- Whether the result came from cache

### Weather cache

Use the `weather_cache` table.

Processing order:

1. Validate district.
2. Check cached record.
3. Return cache if younger than the configured duration.
4. Otherwise call Open-Meteo.
5. Validate provider response.
6. Save the response.
7. Return the new weather object.
8. Return a clear service error if no valid cache exists and the provider fails.

Do not call Open-Meteo when a valid cache exists.

### AI weather orchestration

Create one orchestrator responsible for deciding when weather is required.

Preferred approach:

- Use Groq tool/function calling when supported by the configured model.

Required fallback:

- Detect weather-related questions in the backend.
- Fetch weather before the model request.
- Add the weather data to the prompt.
- Do not allow the model to invent current conditions.

Possible weather-intent terms include:

- Weather
- Rain
- Rainfall
- Temperature
- Humidity
- Tomorrow
- This week
- Planting conditions
- Dry spell
- Ren
- Hot
- Cold

The backend must validate the district itself.

Never execute arbitrary tools, commands, or URLs supplied by the model.

Limit tool iterations.

---

## 12. Frontend Organization

The repository may contain a scattered prototype with:

- A root `index.html`
- One large `app.js`
- One large `styles.css`
- Static dashboard markup
- LocalStorage-based mock data

Recover useful design elements, but reorganize the frontend for Gin.

### Required template structure

```text
web/templates/layouts/app.html
web/templates/pages/assistant.html
web/templates/partials/sidebar.html
web/templates/partials/chat_message.html
web/templates/partials/chat_history.html
web/templates/partials/weather_card.html
web/templates/partials/loading_indicator.html
web/templates/partials/error_message.html
```

### Required static assets

```text
web/static/css/input.css
web/static/css/app.css
web/static/js/app.js
web/static/js/assistant.js
```

### Page requirements

The `/assistant` page must provide:

#### Sidebar

- AgriConnect AI brand
- New conversation button
- Conversation history
- English/Krio selector
- District selector
- Crop selector
- Weather shortcut

#### Main chat area

- Welcome state
- Suggested agricultural questions
- User and assistant message bubbles
- Loading state
- Streaming state
- Error state
- Retry action
- Message input
- Send button
- Safety disclaimer
- Source display when knowledge records were used

#### Weather area

- Selected district
- Current conditions
- Rain probability
- Temperature
- Humidity
- Seven-day summary
- Refresh behavior that respects backend caching

### Responsive behavior

The interface must work on:

- Desktop
- Tablet
- Mobile

On mobile:

- Conversation sidebar becomes a drawer
- Message input stays accessible
- Selectors remain usable
- Weather cards stack vertically
- Content does not overflow horizontally

### Design

Use:

```text
Primary green: #2E7D32
Secondary green: #4CAF50
Accent gold: #FFC107
Background: #F5F7F5
Surface: #FFFFFF
Text: #1F2937
Muted text: #6B7280
Border: #E5E7EB
```

Use Tailwind CSS and Lucide icons.

Do not use emoji as the primary icon system.

### Browser safety

- Escape all AI content.
- Do not inject raw model-generated HTML.
- If rendering Markdown, use a safe renderer and sanitizer.
- Prevent duplicate submissions.
- Disable the send button during active requests.
- Cancel or stop work when the browser disconnects.
- Handle SSE parsing correctly.
- Show clear failures without losing existing messages.

---

## 13. Streaming Requirements

Complete the streaming endpoint when possible.

Use Server-Sent Events.

Supported events:

```text
event: status
data: {"message":"Searching agricultural resources"}

event: status
data: {"message":"Checking weather for Bo"}

event: token
data: {"text":"Yu"}

event: complete
data: {"message_id":"uuid"}

event: error
data: {"message":"The AI service is temporarily unavailable"}
```

Requirements:

- Set correct SSE headers.
- Flush events immediately.
- Stop generation after client cancellation.
- Accumulate the final assistant text safely.
- Save the assistant message only when a complete response exists.
- Send a clean error event on failure.
- Provide a non-streaming fallback route.

---

## 14. Middleware and Security

Implement or repair:

- Request ID middleware
- Panic recovery
- Anonymous user cookie middleware
- Rate limiting
- Request body size limits
- Input validation
- Safe error responses
- Ownership checks
- HTTP client timeouts
- Secure cookie settings
- Structured logs
- No secret logging

Do not return database errors or provider response bodies directly to users.

Use stable public error messages and log internal details with request IDs.

---

## 15. Docker and Development Tools

Create or repair:

### `Dockerfile`

Use a multi-stage build.

Stages should:

1. Install frontend dependencies.
2. Compile Tailwind CSS.
3. Build the Go binary.
4. Copy templates, static assets, prompts, migrations, and seed data.
5. Run as a non-root user.

### `docker-compose.yml`

Include:

- `app`
- `db`

Expose:

```text
Application: 8080
PostgreSQL: 5432
```

The application must wait for PostgreSQL, run migrations, and start.

### `Makefile`

Provide:

```text
make dev
make build
make test
make vet
make migrate-up
make migrate-down
make seed
make css
make docker-up
make docker-down
```

Repair existing commands rather than creating conflicting alternatives.

---

## 16. Health Check

Create:

```text
GET /health
```

Successful response:

```json
{
  "status": "ok",
  "database": "connected"
}
```

Do not expose:

- Database URL
- Groq key
- Groq model secrets
- Environment secrets
- Internal stack traces

Return an appropriate non-200 status when the database is unavailable.

---

## 17. Testing

Use interfaces and mocks so tests do not call live services.

Create or repair tests for:

### Knowledge retrieval

- Finds cassava documents from a cassava question
- Finds disease documents from a symptom question
- Ranks exact crop/category matches higher
- Limits context size
- Does not return unrelated documents

### Weather service

- Rejects unsupported districts
- Returns a valid cached response
- Refreshes expired cache
- Avoids provider calls when cache is valid
- Handles provider failure
- Returns stale cache only when an explicit safe fallback policy exists

### Chat service

- Saves user message
- Applies English prompt behavior
- Applies Krio prompt behavior
- Includes agricultural context
- Includes weather context when required
- Limits recent history
- Saves assistant response
- Preserves user message when Groq fails
- Rejects unauthorized conversation ownership

### Handlers

At minimum, verify:

- Anonymous cookie creation
- Conversation ownership checks
- Message validation
- Health route
- Unsupported district error
- Groq-unavailable response

Run:

```bash
go test ./...
go vet ./...
go build ./cmd/server
```

Fix every failure.

---

## 18. README

Create or repair `README.md`.

It must explain:

1. Project purpose
2. Implemented Phase 1–3 scope
3. Architecture
4. Folder structure
5. Prerequisites
6. Environment setup
7. Running locally
8. Running with Docker
9. Database migrations
10. Seeding agricultural documents
11. Tailwind compilation
12. Groq configuration
13. Open-Meteo integration
14. Running tests
15. API routes
16. Anonymous user behavior
17. Security notes
18. Known limitations
19. Future Phase 4 and Phase 5 extension points

Known limitations must include:

- AI advice can be incorrect
- Krio output requires community review
- Seed agricultural material requires expert validation
- Weather depends on an external provider
- No crop-image diagnosis yet
- No speech input yet
- No complete authentication yet
- No extension-officer review workflow yet

---

## 19. Missing File Policy

When a required file is missing:

- Create it.
- Implement it completely enough for Phase 1–3.
- Connect it to the application.
- Add tests when it contains business logic.

Do not create empty placeholder files.

When an existing file is incomplete:

- Finish it.
- Preserve useful code.
- Remove dead code.
- Update all references.

When an existing file is in the wrong folder:

- Move it.
- Update package declarations.
- Update imports.
- Update template/static references.
- Update Docker copy paths.
- Update tests.

When two files conflict:

- Choose or merge the stronger design.
- Avoid duplicated behavior.
- Record the decision in `RECOVERY_REPORT.md`.

---

## 20. Completion Acceptance Criteria

Do not finish until all relevant conditions pass.

### Repository

- Files are organized coherently.
- One active application entry point exists.
- No required feature is left disconnected.
- No duplicate route registration exists.
- No secret is committed.

### Application

- `docker compose up --build` starts the database and application.
- `/health` confirms database connectivity.
- `/assistant` renders correctly.
- An anonymous user cookie is created.
- A conversation can be created.
- A conversation list can be loaded.
- Conversation messages remain after refresh.
- A conversation can be deleted by its owner.
- Another anonymous browser cannot access it.

### AI

- English mode works.
- Krio mode sends the correct prompt instructions.
- Agricultural context is retrieved and included.
- Responses are saved.
- Groq failure produces a safe UI error.
- Current weather is never invented.

### Weather

- Supported district validation works.
- Open-Meteo data is transformed correctly.
- Cache is used.
- Seven-day forecast displays.
- Weather questions receive real weather context.

### Frontend

- Desktop layout works.
- Tablet layout works.
- Mobile layout works.
- Streaming or fallback chat works.
- Duplicate message submission is prevented.
- Errors and loading states are visible.
- AI output is rendered safely.

### Quality

- `go mod tidy` succeeds.
- `go test ./...` succeeds.
- `go vet ./...` succeeds.
- `go build ./cmd/server` succeeds.
- Tailwind compilation succeeds.
- Docker build succeeds.
- README instructions are reproducible.

---

## 21. Final Work Report

After completing the repository, respond with:

1. Initial problems discovered
2. Files moved
3. Files merged
4. Files created
5. Files removed, if any, and why
6. Architecture implemented
7. Database migrations completed
8. Routes completed
9. AI integration completed
10. Knowledge retrieval completed
11. Weather integration completed
12. Frontend connections completed
13. Tests added
14. Commands executed
15. Test and build results
16. Remaining limitations
17. Exact commands I should run locally

Also ensure `RECOVERY_REPORT.md` contains this information in the repository.

---

## 22. Start Now

Begin by inspecting the repository.

Do not ask me to identify which scattered files belong together. Determine that from their contents, imports, routes, templates, and intended Phase 1–3 architecture.

Do not stop at an audit.

Organize, repair, connect, implement, test, and document the project.
