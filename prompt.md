# AgriConnect Render Deployment Preparation

You are a senior Go, Docker, PostgreSQL and Render deployment engineer working inside the existing AgriConnect repository.

Your task is to inspect the entire codebase, understand how the application currently builds and runs, and make all changes required to deploy AgriConnect successfully as a Render Web Service for user testing.

Do not only provide instructions.

Inspect, implement, configure, test and document the deployment.

---

# 1. Important ownership boundary

Joshua owns all AI and USSD functionality.

Do not redesign, rewrite or take ownership of:

* Groq agricultural chat
* AI prompts
* AI response generation
* Agricultural knowledge retrieval
* Crop-image AI analysis
* Vision-model calls
* Voice transcription
* Krio transcription
* AI streaming
* AI model selection
* USSD menus
* USSD session handling
* USSD provider integration
* USSD-to-AI communication

You may modify configuration, dependency wiring, startup behavior and deployment compatibility only when necessary for hosting.

Do not change the existing AI or USSD business logic unless a specific deployment defect prevents the application from starting.

---

# 2. Inspect the entire repository first

Before editing anything, inspect:

```text
go.mod
go.sum
Dockerfile
docker-compose.yml
render.yaml
.env.example
.gitignore
.dockerignore
Makefile
package.json
tailwind.config.js

cmd/
internal/
migrations/
configs/
web/templates/
web/static/
scripts/
tests/

README.md
RECOVERY_REPORT.md
PHASE_4_5_REPORT.md
FINAL_COMPLETION_REPORT.md
```

Some files might not exist. Record which deployment-related files already exist and which must be created.

Inspect and understand:

1. The actual Go application entry point
2. The migration command and migration binary
3. How configuration is loaded
4. How the application reads its port
5. How Gin starts the HTTP server
6. How PostgreSQL is initialized
7. How Redis is used, if still required
8. How templates are loaded
9. How static files are served
10. How Tailwind CSS is compiled
11. How JavaScript files are copied into the final image
12. How Groq environment variables are loaded
13. How Supabase credentials are loaded
14. How authentication secrets are loaded
15. How storage drivers are selected
16. How the `/health` endpoint behaves
17. Whether startup automatically runs migrations
18. Whether local filesystem paths are assumed
19. Whether any localhost-only addresses are hardcoded
20. Whether any secrets are present in tracked files

Do not assume filenames, binary names or commands. Determine them from the repository.

---

# 3. Run and record the baseline

Before changing the code, run:

```bash
git status
go mod tidy
go test ./...
go vet ./...
go build ./...
docker compose config
```

When Node dependencies are present, also run:

```bash
npm install
npx tailwindcss -i ./web/static/css/input.css -o ./web/static/css/app.css --minify
```

If Docker is available, run:

```bash
docker build -t agriconnect-render-baseline .
```

Record:

* Commands that pass
* Commands that fail
* Exact failure messages
* Missing files
* Missing environment variables
* Current application start command
* Current migration command

Create or update:

```text
RENDER_DEPLOYMENT_REPORT.md
```

Include a baseline section before making changes.

---

# 4. Make the Go server compatible with Render

Render must be able to reach the application through the port assigned at runtime.

Update the application so it:

1. Reads `PORT` first.
2. Falls back to the existing local port when `PORT` is absent.
3. Binds to `0.0.0.0`.
4. Does not bind only to `localhost` or `127.0.0.1`.
5. Logs the listening address without logging secrets.
6. Supports graceful shutdown.
7. Stops cleanly when Render sends a termination signal.
8. Uses reasonable HTTP server timeouts.

The equivalent behavior should be:

```go
port := os.Getenv("PORT")
if port == "" {
    port = existingConfiguredPort
}

address := "0.0.0.0:" + port
```

Do not create a second Gin router.

Do not create a second server entry point.

Modify the existing application startup path.

---

# 5. Verify the health endpoint

The application must expose:

```text
GET /health
```

Requirements:

* Return HTTP 200 when the application is ready.
* Return JSON.
* Do not require authentication.
* Do not call Groq.
* Do not call Open-Meteo.
* Do not require Supabase to answer.
* Avoid leaking database credentials or configuration.
* Make the response fast.
* Include a simple status such as `healthy`.
* If a database-readiness check already exists, use a short timeout.
* Do not let a slow external AI provider make Render mark the service unhealthy.

Example response:

```json
{
  "status": "healthy"
}
```

Add or update tests for `/health`.

---

# 6. Create a production-ready Dockerfile

Inspect the current Dockerfile and improve it instead of blindly replacing it.

Use a multi-stage build where appropriate.

The Docker build must:

1. Download Go dependencies.
2. Compile the migration binary if the project uses one.
3. Compile the server binary.
4. Install Node dependencies only in a build stage.
5. Compile Tailwind CSS.
6. Copy templates into the final image.
7. Copy static JavaScript, CSS, icons and images into the final image.
8. Copy required configuration files.
9. Copy migration files.
10. Include CA certificates for HTTPS calls to Groq, Supabase and Open-Meteo.
11. Avoid placing source-only tools in the final image.
12. Avoid placing `.env` in the image.
13. Avoid embedding credentials as Docker build arguments.
14. Use a non-root runtime user where compatible with the application.
15. Use predictable working directories.
16. ensure the final image starts the correct binary.

Confirm the final image contains all of these at runtime:

```text
Go server binary
migration binary or migration command
HTML templates
Tailwind output CSS
JavaScript files
Lucide integration
migration SQL files
required prompt files
required configuration files
```

The application must not fail on Render because a template, CSS file, JavaScript file, migration or AI prompt is missing from the final image.

---

# 7. Add a safe production startup process

Inspect the current migration process.

Use the actual existing migration implementation.

Implement a production startup flow equivalent to:

```text
Apply pending database migrations
→ Stop immediately if migration fails
→ Start the Go server
```

This can be implemented using:

* Render’s supported pre-deploy migration command, or
* a small entrypoint script, or
* the existing migration-and-start command

Choose the approach that best matches the repository.

Requirements:

1. Migrations must be idempotent.
2. Existing migrations must not be rewritten after deployment.
3. Migration failures must stop the deployment.
4. The server must not start against an outdated schema.
5. Migration logs must not reveal credentials.
6. The startup command must use the actual server binary.
7. Do not run destructive down-migrations during deployment.
8. Do not automatically delete or recreate production data.

Create a script only if it improves reliability, for example:

```text
scripts/render-start.sh
```

Make it executable in the Docker image.

---

# 8. Create or repair `render.yaml`

Create a root-level:

```text
render.yaml
```

Use the current official Render Blueprint specification.

The Blueprint should define:

## Web service

* Type: public web service
* Runtime: Docker
* Repository Dockerfile
* Health check path: `/health`
* Automatic deployment from the selected branch
* Production environment
* Correct Docker context and Dockerfile path
* Appropriate region selection or clear documentation
* Web service name such as `agriconnect`

## PostgreSQL database

Define or reference a Render PostgreSQL database such as:

```text
agriconnect-db
```

Configure the web service’s `DATABASE_URL` from the database connection string using the supported Blueprint database reference.

Do not hardcode the database password.

## Public non-secret environment values

Add appropriate non-secret values such as:

```env
APP_ENV=production
COOKIE_SECURE=true
COOKIE_SAME_SITE=lax
STORAGE_DRIVER=supabase
SUPABASE_STORAGE_BUCKET=crop-diagnosis-images
MAX_IMAGE_SIZE_MB=5
MAX_AUDIO_SIZE_MB=10
MIN_IMAGE_WIDTH=256
MIN_IMAGE_HEIGHT=256
MAX_IMAGE_PIXELS=25000000
```

Use the exact environment-variable names currently loaded by the application.

Do not invent duplicate configuration names.

## Secret environment values

Mark secrets so the user must enter them in Render rather than committing them:

```env
GROQ_API_KEY
SUPABASE_URL
SUPABASE_SECRET_KEY
JWT_ACCESS_SECRET
JWT_REFRESH_SECRET
```

Also include any other secret that the inspected code genuinely requires.

Do not include real values.

Use the appropriate Blueprint mechanism for dashboard-supplied secret values.

## Optional services

If Redis is genuinely required by the current code:

* Determine whether it is required for startup or optional.
* Add the appropriate Render key-value service or document the manual setup.
* Do not add Redis merely because an old file mentions it.
* If Redis is optional, make the application degrade safely without it.

Validate `render.yaml` using an available Render CLI, schema validator or careful comparison with the current Blueprint specification.

Do not claim validation passed unless it was actually validated.

---

# 9. Correct production environment handling

Inspect every environment variable loaded by the application.

Update `.env.example` so it contains placeholders only.

It must never contain:

* Real Groq keys
* Real Supabase secret keys
* Database passwords
* JWT secrets
* USSD provider secrets
* Personal phone numbers
* Production callback tokens

Ensure `.gitignore` contains:

```gitignore
.env
.env.local
.env.production
```

Ensure `.dockerignore` excludes:

```text
.env
.env.*
.git
.gitignore
data/uploads
temporary audio
local test artifacts
coverage files
```

Do not exclude `.env.example`.

Create a complete production environment-variable table in the deployment documentation.

For each variable, document:

* Name
* Required or optional
* Example format without secret values
* Purpose
* Whether Render supplies it
* Whether it must be entered manually

Use the environment variables actually found in the codebase.

---

# 10. PostgreSQL compatibility

Ensure the application can connect to Render PostgreSQL using one complete:

```env
DATABASE_URL
```

Requirements:

1. Support a standard PostgreSQL connection URL.
2. Do not require hardcoded localhost values in production.
3. Do not split the URL incorrectly.
4. Support SSL options present in the Render connection string.
5. Use sensible connection-pool settings.
6. Avoid opening excessive database connections on a small testing instance.
7. Close the database connection during graceful shutdown.
8. Apply migrations before serving traffic.
9. Use the Render database’s internal connection string when referenced through the Blueprint.
10. Keep local Docker Compose PostgreSQL working.

Add a test for configuration parsing where practical.

Do not replace PostgreSQL with Supabase Database unless the repository already uses it.

Supabase remains the crop-image storage provider unless the codebase indicates otherwise.

---

# 11. Supabase production storage

For Render hosting, do not depend on the container’s local filesystem for permanent crop images.

Ensure production uses:

```env
STORAGE_DRIVER=supabase
```

Verify:

* Supabase image upload works.
* Private signed URLs work.
* Image deletion works.
* The private bucket remains private.
* `GET /api/v1/diagnoses/:id/image` does not return 503 simply because local storage is disabled.
* The Supabase secret stays server-side.
* The browser never receives the secret key.
* PostgreSQL stores only the storage object path.
* Signed URLs are not stored permanently.
* Ownership checks happen before generating a signed URL.

If the Supabase implementation is still incomplete, finish the deployment-critical storage functions and add tests.

Do not modify the AI diagnosis logic.

---

# 12. Static files and mobile navigation deployment

The rendered production site must include the latest CSS and JavaScript, including the mobile hamburger menu.

Verify:

1. Tailwind CSS builds during Docker image creation.
2. The final CSS file is copied into the runtime image.
3. All JavaScript modules are copied.
4. The hamburger menu JavaScript is loaded.
5. Lucide icons initialize.
6. Desktop navigation remains horizontal.
7. Mobile navigation is vertical and collapsible.
8. Browser caching does not permanently show an older CSS or JavaScript build.

Implement a simple cache-busting strategy if the project does not already have one.

Acceptable approaches include:

* Content-hashed asset filenames, or
* An asset version derived from the deployment commit, or
* Version query parameters generated from a build version

Do not disable all browser caching globally without reason.

HTML pages should not become permanently cached during active user testing.

Confirm the Docker image contains the newly built static assets rather than stale files from a previous build layer.

---

# 13. Production cookies and proxy behavior

Render terminates HTTPS before forwarding traffic to the application.

Ensure production authentication works behind a reverse proxy.

Requirements:

* Secure cookies enabled in production
* HTTP-only authentication cookies
* Appropriate SameSite behavior
* Correct cookie path
* No hardcoded localhost cookie domain
* Respect forwarded HTTPS information where needed
* Avoid redirect loops
* Do not trust arbitrary proxy headers from untrusted direct clients
* APP_URL or equivalent production URL must be configurable
* Logout must clear cookies correctly

Do not implement authentication from scratch if it already exists.

Only make it deployment-compatible.

---

# 14. Logging and error handling

Production logs must go to standard output and standard error.

Requirements:

* Use structured logs where the project already supports them.
* Log startup stages.
* Log successful database connection without credentials.
* Log migration success or failure.
* Log the selected storage driver without secrets.
* Log the server port.
* Do not log Groq keys.
* Do not log Supabase keys.
* Do not log JWT secrets.
* Do not log database passwords.
* Do not log full voice transcripts by default.
* Do not log uploaded image bytes.
* Return safe user-facing errors.
* Keep enough error context for Render logs.

Do not write essential logs only to local files because the Render container filesystem is not the primary logging system.

---

# 15. Preserve local development

Render changes must not break local development.

The following should continue working:

```bash
go run ./cmd/server
docker compose up --build
go test ./...
```

Local development may continue using:

```env
STORAGE_DRIVER=local
LOCAL_UPLOAD_DIR=./data/uploads
```

Production must support:

```env
STORAGE_DRIVER=supabase
```

Keep the same application and codebase for both environments.

Do not create a separate Render-only copy of the application.

---

# 16. Do not commit secrets

Search the repository for accidentally committed secrets.

Inspect:

```bash
git grep -n "sb_secret_"
git grep -n "gsk_"
git grep -n "GROQ_API_KEY="
git grep -n "SUPABASE_SECRET_KEY="
git grep -n "JWT_ACCESS_SECRET="
git grep -n "JWT_REFRESH_SECRET="
```

Also inspect Git status and tracked environment files.

If a real secret appears:

1. Remove it from tracked files.
2. Replace it with a placeholder.
3. State clearly in the report that the exposed key should be rotated.
4. Do not print the complete key in the report.
5. Do not attempt to silently hide the incident.

Do not rewrite Git history unless explicitly authorized.

---

# 17. Add deployment documentation

Create:

```text
RENDER_DEPLOYMENT.md
```

The guide must include:

## Render resources

* One Render Web Service
* One Render PostgreSQL database
* Supabase private storage
* Groq API
* Open-Meteo
* Optional Redis or key-value service only if genuinely required

## Manual dashboard deployment

Document:

1. Push repository to GitHub.
2. Sign in to Render.
3. Create a Blueprint from `render.yaml`, or create the services manually.
4. Select the correct branch.
5. Select a region.
6. Keep the web service and database in the same region.
7. Enter all secret environment variables.
8. Deploy.
9. Watch build logs.
10. Confirm migrations.
11. Open `/health`.
12. Test registration and login.
13. Test the AI assistant.
14. Test crop diagnosis.
15. Test voice transcription.
16. Test Supabase image display.
17. Test mobile navigation.
18. Configure the USSD callback URL manually if the existing USSD provider requires it.

Do not implement or change the USSD logic.

Only document the final public callback URL pattern based on the route already present in the repository.

## Required secret variables

List every required secret discovered during inspection.

Do not place values in the guide.

## Testing checklist

Include exact public routes to test.

## Troubleshooting

Include solutions for:

* No open port detected
* Application binding to localhost
* Database connection refused
* Migration failure
* Missing template
* Missing CSS or JavaScript
* Stale static assets
* Supabase 401 or 403
* Groq authentication failure
* Health check failure
* Secure-cookie login issue
* Incorrect APP_URL
* Service starts locally but fails in Docker

---

# 18. Add a deployment checklist

Create:

```text
RENDER_DEPLOYMENT_CHECKLIST.md
```

Include checkboxes for:

## Before deployment

* Tests pass
* Docker image builds
* `.env` ignored
* No real secrets in repository
* Tailwind CSS compiled
* Templates copied
* Static JavaScript copied
* Migrations included
* Health endpoint works
* Supabase signed URLs work

## Render configuration

* Web service created
* PostgreSQL created
* Same region selected
* Database URL connected
* Groq key added
* Supabase URL added
* Supabase secret key added
* JWT secrets added
* Storage driver set to Supabase
* Health path set to `/health`

## After deployment

* `/health` returns 200
* Home page loads
* Mobile hamburger menu works
* Registration works
* Login works
* AI assistant works
* Weather works
* Crop upload works
* Diagnosis result works
* Private image loads
* Voice transcription works
* Officer route protection works
* Admin route protection works
* USSD callback route is publicly reachable if implemented
* No secrets appear in browser source
* No critical errors appear in Render logs

---

# 19. Test the final deployment image locally

Run:

```bash
go mod tidy
go test ./...
go test -race ./...
go vet ./...
go build ./...
npm install
npx tailwindcss -i ./web/static/css/input.css -o ./web/static/css/app.css --minify
docker compose config
docker build --no-cache -t agriconnect-render .
```

If Docker is available, run the image using safe local test environment values.

Confirm:

* The container starts.
* Migrations run.
* The server binds to the supplied `PORT`.
* `/health` returns 200.
* Templates load.
* CSS loads.
* JavaScript loads.
* The mobile hamburger menu works.
* The application connects to PostgreSQL.
* Startup fails safely when required variables are missing.

Do not call paid Groq endpoints automatically in normal unit tests.

Do not call live Supabase automatically unless an explicitly enabled integration-test mode exists.

---

# 20. Optional Render integration test

If Render credentials or CLI access are already configured in the environment, validate the Blueprint.

Do not ask for or print personal Render API keys.

If deployment access is not available:

* Prepare the repository completely.
* Validate everything possible locally.
* Provide the exact manual dashboard steps.
* Do not claim that the application was deployed.

If actual deployment access is available and authorized:

1. Deploy the Blueprint.
2. Monitor logs.
3. Verify `/health`.
4. Verify public pages.
5. Record the generated public service URL.
6. Do not expose secret values.
7. Record the actual deployment result.

---

# 21. Final acceptance criteria

The Render preparation is complete only when:

* The entire repository was inspected.
* The server binds to `0.0.0.0:$PORT`.
* `/health` returns HTTP 200.
* The Docker image builds.
* The Docker image contains templates and static assets.
* Tailwind builds during deployment.
* The latest hamburger-menu JavaScript is deployed.
* PostgreSQL connects through `DATABASE_URL`.
* Migrations run before the server starts.
* Supabase is used for production crop images.
* No real secrets are committed.
* `render.yaml` exists and uses valid current syntax.
* Render secrets are represented as placeholders.
* Local development still works.
* Existing AI and USSD logic remains intact.
* Tests pass.
* Deployment documentation exists.
* A deployment checklist exists.
* The final report honestly states what was and was not executed.

---

# 22. Final report

Update:

```text
RENDER_DEPLOYMENT_REPORT.md
```

The final report must contain:

1. Repository structure inspected
2. Existing deployment problems found
3. Files created
4. Files modified
5. Port-binding changes
6. Dockerfile changes
7. Migration-startup changes
8. `render.yaml` configuration
9. PostgreSQL configuration
10. Supabase production-storage status
11. Static-asset build and cache-busting status
12. Environment variables required
13. Secret scan results
14. Tests executed
15. Test results
16. Docker build result
17. Local container result
18. Blueprint validation result
19. Actual Render deployment result, if authorized
20. Remaining manual dashboard steps
21. Known limitations
22. Final public-testing checklist

Do not only summarize what should be done.

Make the deployment changes, run the available checks, fix failures and prepare the repository for Render.

Do not mark a step as passed unless you executed it successfully.
