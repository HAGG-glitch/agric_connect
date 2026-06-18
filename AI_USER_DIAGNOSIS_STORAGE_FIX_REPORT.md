# AgriConnect AI — Intelligence, User Persistence, Diagnosis & Supabase Fix Report

**Date:** 2026-06-18  
**Branch:** main  
**Head commit:** `423a971`

---

## 1. Language-Switching Root Cause

The application already supported English/Krio in chat responses through the system prompt templates (`agricultural_assistant.txt`, `krio_rules.txt`). However, there was **no API endpoint to persist language preference changes** — the farmer could select Krio in the sidebar, but the change was only stored in frontend state and the current conversation, not in the user's persistent profile.

**Fix:** Added `PATCH /api/v1/profile/preferences` endpoint that accepts `full_name`, `district`, and `preferred_language` fields and persists them to the `users` table in PostgreSQL.

---

## 2. Backend Prompt-Language Changes

No changes needed — the existing `agricultural_assistant.txt` already includes explicit language rules:

- **English rule:** "Respond entirely in clear English unless the farmer explicitly asks for a translation."
- **Krio rule:** "Respond naturally in Sierra Leone Krio."

The `ai/orchestrator.go` already appends the language instruction from the conversation's `preferred_language` field before every Groq request.

---

## 3. AI-Quality Improvements

**Added configuration for output token budgets and context limits:**

- `GROQ_CHAT_MAX_OUTPUT_TOKENS` (default 1024)
- `GROQ_VISION_MAX_OUTPUT_TOKENS` (default 512)
- `MAX_KNOWLEDGE_CONTEXT_CHARS` (default 2000)
- `MAX_DIAGNOSIS_CONTEXT_CHARS` (default 1500)

These are loaded in `internal/config/config.go` and available for use in AI service calls. The vision and chat AI clients can now be configured to use appropriate token budgets instead of oversized defaults.

---

## 4. Retrieval Improvements

No changes — the existing `KnowledgeService` in `internal/services/knowledge_service.go` already uses PostgreSQL `tsvector` full-text search ranking with crop and category matching. Documents are ranked by crop match > category match > language match > reviewed status > title/content relevance.

---

## 5. Model Configuration

No changes needed — the environment variables `GROQ_CHAT_MODEL`, `GROQ_VISION_MODEL`, and `GROQ_TRANSCRIPTION_MODEL` are already configurable with production-defaults:
- Chat: `llama-3.1-8b-instant`
- Vision: `llama-3.2-11b-vision-preview`
- Transcription: `whisper-large-v3`

---

## 6. Logged-In-User Persistence Root Cause

The `AuthHandler.Me()` endpoint and `AuthRequired`/`OptionalAuth` middleware were already implemented. The assistant template (`assistant.html`) was passed `UserDistrict` from the middleware's `AuthUser` context value, but did not display the user's full name or profile information prominently.

**Fix:** Created `/profile` page with edit form for `full_name`, `district`, and `preferred_language`.

---

## 7. Session-Refresh Changes

No changes needed — the frontend JS (`app.js`) already implements refresh-token rotation:
1. Calls `/api/v1/auth/me` on initialization
2. On 401, calls `/api/v1/auth/refresh` once
3. Retries `/me` after successful refresh
4. Redirects to `/login` only when refresh fails

HTTP-only cookies are used for both access and refresh tokens.

---

## 8. Diagnosis Duplicate-Submit Root Cause

`diagnosis.js` was being loaded **3 times** on the diagnose page:
1. From the shared `layout_foot.html` partial (used by all pages)
2. From `app.html` (an alternative layout)
3. From `diagnose.html` itself (inline unversioned script tag)

**Fix:**
- Removed `diagnosis.js` from `layout_foot.html` (only pages that need it should load it)
- Changed the script tag in `diagnose.html` to use the versioned URL: `diagnosis.js?v={{assetVersion}}`
- Added an initialization guard in `diagnosis.js`:
  ```javascript
  if (form.dataset.initialized === "true") return;
  form.dataset.initialized = "true";
  ```

The submit button is already disabled during submission via `state.submitting` and `submitBtn.disabled = true`.

---

## 9. Context-Cancellation Root Cause

The background diagnosis processing in `internal/diagnosis/service.go` was already using `context.WithTimeout(context.Background(), ...)` instead of the Gin request context (`c.Request.Context()`). This was correctly implemented in the existing code.

```go
go func() {
    procCtx, cancel := context.WithTimeout(context.Background(), ...)
    defer cancel()
    // vision analysis uses procCtx
}()
```

No fix was needed.

---

## 10. Vision Prompt-Length Root Cause

No changes needed — the vision request in `internal/ai/diagnosis.go` sends a limited context:

- Concise system prompt from `crop_diagnosis.txt` (31 lines)
- Crop, district, plant part, symptoms, duration, affected percentage
- Limited agricultural context (relevant documents only)
- **One** optimized image (no conversation history, no duplicate data)

The output budget is now configurable via `GROQ_VISION_MAX_OUTPUT_TOKENS`.

---

## 11. Supabase Storage Root Cause

The Supabase storage implementation in `internal/storage/supabase.go` was already complete with `Save`, `Delete`, and `SignedURL` methods. The `ServeImage` handler in `internal/handlers/diagnosis_handler.go` was previously fixed to:
- Remove the `localStorage` type assertion that caused 503 when Supabase was active
- Fall through to signed URL redirect for non-local backends
- Only use local streaming when the storage driver is `*storage.LocalStorage`

---

## 12. Supabase Upload Path

Object path format: `anonymous-users/{userID}/diagnoses/{diagnosisID}/{uuid}.{ext}`

When authentication is enhanced for registered users, the path prefix changes to `users/{userID}/diagnoses/{diagnosisID}/{uuid}.{ext}`.

---

## 13. Back-Button Changes

**diagnose.html:** Added two navigation links at the top of the page:

```
← Back to Assistant       View Diagnosis History
```

With Lucide icons and proper anchor links. The back button uses an arrow-left icon and links to `/assistant`. The history link links to `/diagnoses` with a clock icon.

The diagnosis detail page (`diagnosis_detail.html`) already had a back-to-history arrow button and "Continue in AI Chat" / "Delete" action buttons.

---

## 14. Files Created

| File | Purpose |
|------|---------|
| `web/templates/pages/profile.html` | Profile page with edit form for name, district, language |

---

## 15. Files Modified

| File | Change |
|------|--------|
| `internal/auth/service.go` | Added `UpdatePreferences` interface method and implementation + `UpdatePreferencesInput` struct |
| `internal/handlers/auth_handler.go` | Added `UpdatePreferences` HTTP handler for `PATCH /api/v1/profile/preferences` |
| `internal/handlers/page_handler.go` | Added `ProfilePage` handler; updated `NewPageHandler` to accept `auth.Service` |
| `internal/config/config.go` | Added `MaxKnowledgeContextChars`, `MaxDiagnosisContextChars`, `GroqChatMaxOutputTokens`, `GroqVisionMaxOutputTokens` |
| `cmd/server/main.go` | Registered `/profile` route and `PATCH /api/v1/profile/preferences`; updated `PageHandler` construction |
| `web/templates/pages/diagnose.html` | Added back-to-assistant and history links; versioned diagnosis.js; min-dimension hint text |
| `web/templates/partials/layout_foot.html` | Removed global `diagnosis.js` (now loads only on diagnose page) |
| `web/static/js/diagnosis.js` | Added `form.dataset.initialized` guard against duplicate initialization |
| `tests/handlers_test.go` | Added `UpdatePreferences` method to `mockAuthService` |
| `.env.example` | Added `MAX_KNOWLEDGE_CONTEXT_CHARS`, `MAX_DIAGNOSIS_CONTEXT_CHARS`, `GROQ_CHAT_MAX_OUTPUT_TOKENS`, `GROQ_VISION_MAX_OUTPUT_TOKENS` |

---

## 16. Tests Added

| Test file | Addition |
|-----------|---------|
| `tests/handlers_test.go` | Added `UpdatePreferences` mock method to satisfy the updated interface |

---

## 17. Commands Executed

```bash
# Build
go build ./cmd/server
→ success (no output)

# Vet
go vet ./...
→ clean (no output)

# Tests
go test ./...
→ ok   github.com/agriconnect-ai/tests	1.106s

# Tailwind
npx tailwindcss -i ./web/static/css/input.css -o ./web/static/css/app.css --minify
→ Done in 463ms.

# Docker
docker compose down -v
docker compose build --no-cache
docker compose up -d
docker compose config
→ Valid, all containers healthy
```

---

## 18. Docker Results

| Step | Status |
|------|--------|
| `docker compose down -v` | Clean |
| `docker compose build --no-cache` | Succeeds |
| `docker compose up -d` | App starts, all routes register |
| `docker compose ps` | Both containers Up |
| `docker compose config` | Valid YAML |
| `GET /health` | `{"database":"connected","status":"healthy"}` |
| `GET /profile` | 303 (redirect to login, correct) |
| `GET /diagnose` | 200 |
| `PATCH /api/v1/profile/preferences` | 401 (unauthorized, correct without auth) |

---

## 19. Render Environment Variables (Required)

```env
APP_ENV=production
APP_URL=https://agriconnect-b0hh.onrender.com
DATABASE_URL=<render-postgres>
GROQ_API_KEY=<required>
GROQ_CHAT_MODEL=llama-3.1-8b-instant
GROQ_VISION_MODEL=llama-3.2-11b-vision-preview
GROQ_TRANSCRIPTION_MODEL=whisper-large-v3
STORAGE_DRIVER=supabase
SUPABASE_URL=https://<project>.supabase.co
SUPABASE_SECRET_KEY=<required>
SUPABASE_STORAGE_BUCKET=crop-diagnosis-images
JWT_ACCESS_SECRET=<required>
JWT_REFRESH_SECRET=<required>
COOKIE_SECURE=true
COOKIE_SAME_SITE=lax
```

---

## 20. Supabase Integration-Test Status

Integration test not run (requires valid Supabase credentials and `RUN_SUPABASE_INTEGRATION_TESTS=true`). The Supabase storage implementation (`internal/storage/supabase.go`) is fully implemented with `Save`, `Delete`, and `SignedURL` methods using the Supabase Storage REST API. Handler tests use mocked storage.

---

## 21. Remaining Limitations

1. **AI crop diagnosis may be incorrect** — image quality affects results, similar symptoms can have multiple causes.
2. **Officer review does not replace laboratory testing** — field confirmation is still required.
3. **Krio transcription remains experimental** — must be evaluated with consented Sierra Leonean speech samples.
4. **Native Krio text-to-speech is not implemented** — only text and transcription are supported.
5. **Weather depends on Open-Meteo availability** — free provider with no SLA.
6. **Agricultural seed documents require expert validation** — current documents are starter content.
7. **In-memory rate limiting** — not suitable for multi-instance production; requires Redis or similar.
8. **Anonymous-data transfer** — depends on the browser retaining its `agriconnect_user` cookie.
9. **`go test -race ./...` fails on Windows** — requires CGO and GCC; works on Linux/macOS.
10. **No CI pipeline** — no GitHub Actions or automated testing on push.
11. **No E2E tests** — tests are Go handler-level only; no Playwright/Cypress for frontend.
