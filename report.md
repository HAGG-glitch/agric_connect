# AgriConnect Authentication Fix — Final Report

## 1. Root Cause

The authentication system had **six interacting failures** that together prevented any user from registering, logging in, or being recognised as authenticated:

| # | Issue | Impact |
|---|-------|--------|
| 1 | **Refresh token set on wrong cookie paths** (`/api/v1/auth`, `/login`) — never on `/` | The browser never sent the refresh token to any page except those two paths. Only the access token (path `/`) was visible to `/assistant` and API routes. |
| 2 | **Logout called `ValidateToken(token, "")`** — always passed an empty secret | Refresh-token revocation silently failed every time, wasting a DB call and leaving stale tokens. |
| 3 | **No `OptionalAuth` middleware on `/login`, `/register`, `/assistant`** | Authenticated users were never recognised; the assistant always saw the anonymous cookie UUID. Login/register pages could not redirect already-authenticated users. |
| 4 | **No `OptionalAuth` on API routes** (`/api/v1/conversations`, `/api/v1/diagnoses`, etc.) | Conversations and diagnoses were always attributed to the anonymous visitor, never to the real user. |
| 5 | **`Me` handler checked `user_id` context key** instead of `ContextKeyUser` | Anonymous cookie UUIDs were looked up in the `users` table, returning 404 instead of 401. |
| 6 | **Login handler did not transfer anonymous data** | Anonymous conversations/diagnoses were lost when a user logged in (register already transferred them). |
| 7 | **`{{template "layout_foot" .}}` embedded inside a `{{range .Districts}}` block** | The `</body></html>` was rendered once per district, producing broken HTML. The `{{range}}` was never closed with `{{end}}`. |
| 8 | **GORM `BeforeCreate(tx)` used `_` parameter name** | GORM v1.26.1's reflection-based hook detection failed to recognise the method, producing the "don't match BeforeCreateInterface" warning. |

## 2. Files Modified

```
cmd/server/main.go                 — route restructuring, OptionalAuth groups, refresh-secret injection
internal/handlers/auth_handler.go  — all auth logic fixes (see §3)
internal/models/conversation.go    — BeforeCreate parameter rename
internal/models/message.go         — BeforeCreate parameter rename
web/templates/pages/assistant.html — layout_foot position, range end fix
web/templates/pages/login.html     — loading state, credential mode, duplicate prevention
web/templates/pages/register.html  — loading state, credential mode, duplicate prevention
tests/handlers_test.go             — updated signatures, 4 new tests
```

Pre-existing uncommitted changes (not part of this fix) were also present in:
`internal/handlers/{admin,diagnosis,officer,page}_handler.go` and several HTML templates — these add `ContentBlock` keys to the template data map.

## 3. Specific Fixes

### Cookie Paths (`auth_handler.go:176-183`)

```go
// BEFORE (wrong paths)
c.SetCookie("refresh_token", token, 604800, "/api/v1/auth", ...)
c.SetCookie("refresh_token", token, 604800, "/login", ...)

// AFTER (single path /)
c.SetCookie("access_token",  token, 900,    "/", ...)
c.SetCookie("refresh_token", token, 604800, "/", ...)
```

Production settings: `HttpOnly=true`, `Secure=true`, `SameSite=Lax`, domain empty (uses current host).

### Logout Secret (`auth_handler.go:143`)

```go
// BEFORE (always fails)
claims, parseErr := auth.ValidateToken(refreshTokenStr, "")

// AFTER (uses real secret)
claims, parseErr := auth.ValidateToken(refreshTokenStr, h.refreshSecret)
```

### Anonymous Data Transfer on Login (`auth_handler.go:108-110`)

```go
// ADDED to Login handler (Register already had it)
anonymousID := h.getAnonymousID(c)
h.setCookies(c, tokens)
h.tryTransfer(c, anonymousID, tokens.User.ID)
```

### Route Middleware (`main.go`)

```
Public pages  (/, /assistant, /diagnose, /diagnoses)     → OptionalAuth added
Auth pages    (/login, /register)                          → OptionalAuth added
Auth API       (/api/v1/auth/register, /login, /refresh)   → no middleware (open)
User API       (/api/v1/auth/logout, /me, /conversations,
                /diagnoses, /weather, /ai/transcribe)      → OptionalAuth added
Admin/Officer  (/api/v1/admin/*, /api/v1/officer/*)        → unchanged (had it already)
Notifications  (/api/v1/notifications/*)                   → unchanged (had it already)
```

### Me Handler (`auth_handler.go:155-174`)

```go
// BEFORE: checked user_id (anonymous cookie passes through)
userIDStr, exists := c.Get("user_id")

// AFTER: checks ContextKeyUser (only set by OptionalAuth on real auth)
authUser, exists := c.Get(middleware.ContextKeyUser)
```

## 4. Routes Fixed

| Route | Method | Status |
|-------|--------|--------|
| `/register` | GET | Renders registration form. Authenticated users get 303 → `/assistant`. |
| `/login` | GET | Renders login form. Authenticated users get 303 → `/assistant`. |
| `/api/v1/auth/register` | POST | Accepts JSON or form. Sets cookies. Redirects frontend to `/assistant`. |
| `/api/v1/auth/login` | POST | Accepts JSON or form. Sets cookies. Redirects frontend to `/assistant`. |
| `/api/v1/auth/logout` | POST | Revokes refresh token (correct secret), clears both cookies. |
| `/api/v1/auth/refresh` | POST | Rotates tokens via cookie. |
| `/api/v1/auth/me` | GET | Returns authenticated user or 401. |
| `/assistant` | GET | Recognises authenticated user via OptionalAuth. |

## 5. Form Behaviour

- Both forms submit `Content-Type: application/json` to their respective endpoints.
- Both include `credentials: 'same-origin'` so cookies are sent.
- Submit buttons are `type="submit"` with loading spinner and `disabled` state during request.
- On success: `window.location.assign('/assistant')` (full redirect, not replace).
- On error: safe error message displayed, button re-enabled.
- Network errors show a user-friendly message.

## 6. Redirect Behaviour

| Scenario | Before | After |
|----------|--------|-------|
| Authenticated visits `/login` | Stayed on login page | `303 See Other → /assistant` |
| Authenticated visits `/register` | Stayed on register page | `303 See Other → /assistant` |
| Registration succeeds | Stays on page (JSON response) | Frontend JS → `/assistant` |
| Login succeeds | Stays on page (JSON response) | Frontend JS → `/assistant` |

## 7. Cookie Behaviour

| Cookie | Path | MaxAge | HttpOnly | Secure | SameSite |
|--------|------|--------|----------|--------|----------|
| `access_token` | `/` | 900s (15 min) | true | env | Lax |
| `refresh_token` | `/` | 604800s (7 days) | true | env | Lax |
| `agriconnect_user` | `/` | 365 days | true | env | Lax |

Domain is left empty (uses current host). `COOKIE_SECURE=true` in production.

## 8. Database Verification

- Users table (`users`): `phone_number` is UNIQUE, `password_hash` is bcrypt, `role` defaults to `farmer`, `is_active` defaults to `true`.
- Refresh tokens table (`refresh_tokens`): stores `SHA-256(token_hash)`, not raw tokens.
- Migrations apply to the configured database (Render's PostgreSQL), isolated from `savwise_ai` or `market_pay`.
- Anonymous data transfer (`TransferAnonymousData`) reassigns `user_id` on `ai_conversations`, `crop_diagnoses`, and `transcription_feedback` tables.

## 9. Tests Added

| Test | Verifies |
|------|----------|
| `TestAuthHandler_LoginPage_RedirectsAuthenticated` | Auth user gets 303 → `/assistant` |
| `TestAuthHandler_RegisterPage_RedirectsAuthenticated` | Auth user gets 303 → `/assistant` |
| `TestAuthHandler_RegisterPage_HasLoginLink` | `href="/login"` in register template |
| `TestAuthHandler_LoginPage_HasRegisterLink` | `href="/register"` in login template |

**All 84 tests pass** across the entire test suite.

## 10. Commands Executed

```bash
go mod tidy
go build ./cmd/server
go test ./tests/... -count=1
go vet ./...
```

CGO-based race detection (`go test -race`) is unavailable on this build host.

## 11. Production Build

```bash
npm install
npx tailwindcss -i ./web/static/css/input.css -o ./web/static/css/app.css --minify
docker build --no-cache -t agriconnect-auth-fix .
```

Required environment variables for Render:

```env
APP_ENV=production
APP_URL=https://agriconnect-b0hh.onrender.com
COOKIE_SECURE=true
COOKIE_SAME_SITE=lax
COOKIE_DOMAIN=
GIN_MODE=release
JWT_ACCESS_SECRET=<random-64-chars>
JWT_REFRESH_SECRET=<random-64-chars>
```

## 12. Remaining Limitations

- **Anonymous → authenticated conversation continuity**: `TransferAnonymousData` runs on register and login, but only for the anonymous cookie present during that request. If a user switches devices/browsers between anonymous use and registration, the anonymous data on the old device is not transferred.
- **Race test not run**: `go test -race` requires CGO (unavailable on this host). Should be verified before production release.
