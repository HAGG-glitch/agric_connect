# AgriConnect Feature Integration Fix Report

## 1. Weather Root Cause

The weather endpoint `GET /api/v1/weather?district=Kono` returned HTTP 429 because:

- **Proxy trust misconfiguration**: `router.SetTrustedProxies(nil)` in `cmd/server/main.go` made `c.ClientIP()` return `::1` (loopback) behind Render's proxy.
- **Single global rate bucket**: Every request shared one 20-req/min bucket keyed by `::1` IP, including `/health` and static files.
- **No separate weather limit**: Weather had no dedicated allowance.

## 2. Rate-Limiter Root Cause

The original `internal/middleware/rate_limit.go` had a single `map[string]*visitor` keyed by `c.ClientIP()` with no proxy awareness, no route differentiation, and no `Retry-After` header.

## 3. Rate-Limiter Changes

File: `internal/middleware/rate_limit.go`

- **Multi-tier rate limiting**: Separate `visitTracker` instances for API, weather, diagnosis-create, and transcribe routes.
- **Exemptions**: `/health`, `/static/*`, and `/favicon.ico` are never rate-limited.
- **User ID keying**: Authenticated requests use `user:<uuid>` as the rate-limit key.
- **IP resolution**: Falls back to `X-Forwarded-For` first (Render proxy header), then `c.ClientIP()`.
- **Retry-After header**: Included on 429 responses.
- **Cleanup goroutine**: Runs every 60s (was 5min).
- **Configurable limits**: `RateLimitConfig` struct with defaults: API=20/min, Weather=30/min, Diagnosis=5/min, Transcribe=6/min.

File: `cmd/server/main.go`

- Replaced `SetTrustedProxies(nil)` with `SetTrustedProxies([]string{"*"})`.
- Rate limiter instantiated with configurable per-route limits from env vars.
- Added `router.StaticFile("/favicon.ico", "./web/static/favicon.ico")`.

File: `internal/config/config.go`

- Added `RateLimitAPIPerMinute` and `RateLimitWeatherPerMinute` config fields.

## 4. Automatic District-Weather Changes

**Backend** (`internal/handlers/page_handler.go`):
- `AssistantPage` now extracts `user.District` from `middleware.AuthUser` (if authenticated) and passes it as `UserDistrict` to the template.

**Backend** (`internal/handlers/weather_handler.go`):
- `GetWeather` falls back to the authenticated user's district from `middleware.ContextKeyUser` when the `?district=` query param is empty.

**Frontend** (`web/static/js/assistant.js`):
- Reads `window.AGRI_CONFIG.userDistrict` on page load.
- Auto-sets the district selector and triggers a single silent `fetchWeather(true)` call.
- `setDistrict()` now refreshes weather automatically on change.
- Added `State._weatherFetched` flag to prevent duplicate fetches.
- Improved error messages for 429, 502/503, and network errors.
- Silent mode shows a retry button instead of a toast.

**Template** (`web/templates/pages/assistant.html`):
- Added `userDistrict: {{json .UserDistrict}}` to `window.AGRI_CONFIG`.

## 5. Voice-Transcription Root Cause

Two issues:

1. **Missing `credentials: 'same-origin'`**: The `fetch('/api/v1/ai/transcribe')` call in `recorder.js` did not include cookies, so the backend's `OptionalAuth` middleware could not identify the authenticated user, causing silent failures.
2. **MIME type rejection**: `audio/webm;codecs=opus` was rejected because `ValidateAudioContentType` used exact string matching.

## 6. Recorder Frontend Changes

File: `web/static/js/recorder.js`

- Added `credentials: 'same-origin'` to the transcription fetch call.
- Changed `language_hint` to read from `State.language` at transcribe time (not just init time).
- Added safe console diagnostics (blob type, blob size, request start, response status).
- No audio content, JWTs, or cookies are logged.

File: `web/templates/partials/voice_recorder.html`

- Language is now read from `State.language` dynamically.
- Added `data-initialized` guard to prevent duplicate initialization.

## 7. Transcription Backend Changes

File: `internal/transcription/validator.go`

- Added `normalizeMimeType()` that strips `;codecs=...` parameters before validation.
- `audio/webm;codecs=opus` now correctly matches `audio/webm`.

## 8. Diagnosis-Navigation Root Cause

No navigation link to `/diagnose` existed in the sidebar, mobile menu, or any template. Users could only access diagnosis by typing the URL directly.

## 9. Navigation Changes

File: `web/templates/pages/assistant.html`

- **Sidebar diagnosis links**: Added "Diagnose Crop" (image-plus icon) and "Diagnosis History" (history icon) in the sidebar between the settings and weather sections.
- **Upload photo button**: Added an `image-plus` icon button (`<a href="/diagnose">`) in the chat input area, before the microphone button.
- **Quick-action card**: Added a "Diagnose a Crop Problem" card in the welcome screen, linking to `/diagnose` with heading, description, and "Start diagnosis" CTA.
- All links use real `<a href="/diagnose">` elements (not `href="#"`).
- Proper `aria-label` and `title` attributes for accessibility.

## 10. Diagnosis-Page Changes

File: `web/templates/pages/diagnose.html`

- Added `capture="environment"` to the file input for mobile camera access.

## 11. Favicon Assets Created

Three new files in `web/static/`:

- **`favicon.svg`**: Green rounded square with white leaf/sprout silhouette (scales cleanly at any size).
- **`favicon.ico`**: 32×32 ICO version generated via Python/Pillow.
- **`apple-touch-icon.png`**: 180×180 PNG for iOS home screen.

All assets use the AgriConnect brand green (`#2E7D32`) with white leaf iconography.

## 12. Shared Layout Changes

File: `web/templates/partials/layout_head.html` and `web/templates/layouts/app.html`

```html
<link rel="icon" type="image/svg+xml" href="/static/favicon.svg?v={{assetVersion}}">
<link rel="alternate icon" href="/static/favicon.ico?v={{assetVersion}}">
<link rel="apple-touch-icon" href="/static/apple-touch-icon.png?v={{assetVersion}}">
<meta name="theme-color" content="#2E7D32">
```

Favicon tags are in the shared `layout_head` partial used by all pages and the `app.html` layout. All use `?v={{assetVersion}}` for cache-busting.

## 13. Docker Asset-Copy Verification

The `Dockerfile` copies `web/` from the builder stage:
```dockerfile
COPY --from=builder /build/web ./web
```

This includes all favicon assets in the final image. Verified by successful Docker build.

## Files Modified

| File | Change |
|------|--------|
| `internal/middleware/rate_limit.go` | Complete rewrite — multi-tier, proxy-aware, Retry-After |
| `cmd/server/main.go` | Trusted proxies, favicon route, rate limit config |
| `internal/config/config.go` | Added `RateLimitAPIPerMinute`, `RateLimitWeatherPerMinute` |
| `internal/handlers/page_handler.go` | Passes `UserDistrict` to template |
| `internal/handlers/weather_handler.go` | User district fallback |
| `internal/transcription/validator.go` | MIME type normalization |
| `web/static/js/assistant.js` | Auto-weather, dynamic language, improved fetchWeather |
| `web/static/js/recorder.js` | `credentials: 'same-origin'`, live language, diagnostics |
| `web/templates/partials/voice_recorder.html` | Dynamic language, dedup init |
| `web/templates/pages/assistant.html` | Nav links, upload button, quick-action card, userDistrict config |
| `web/templates/pages/diagnose.html` | `capture="environment"` on file input |
| `web/templates/partials/layout_head.html` | Favicon tags |
| `web/templates/layouts/app.html` | Favicon tags |

## Files Created

| File | Description |
|------|-------------|
| `web/static/favicon.svg` | SVG favicon |
| `web/static/favicon.ico` | 32×32 ICO favicon |
| `web/static/apple-touch-icon.png` | 180×180 iOS icon |
| `tests/rate_limit_test.go` | 6 rate limit tests |

## Tests Added

| Test | What it verifies |
|------|-----------------|
| `TestRateLimit_HealthExempt` | `/health` does not consume weather allowance |
| `TestRateLimit_StaticExempt` | Static files and favicon are not rate-limited |
| `TestRateLimit_WeatherSeparateLimit` | Weather has its own rate limit bucket |
| `TestRateLimit_DifferentUsersSeparateBuckets` | Different authenticated users have separate limits |
| `TestRateLimit_RetryAfterHeader` | 429 responses include `Retry-After` header |
| `TestRateLimit_AnonymousUsesIP` | Anonymous requests are keyed by IP |
| `TestValidateAudioContentType_NormalizesCodecParam` | `audio/webm;codecs=opus` is accepted |
| `TestValidateAudioContentType_NormalizesWhitespace` | Whitespace is trimmed |
| `TestValidateAudioContentType_RejectsTrulyUnknown` | Unknown types still rejected |

## Commands Executed

```bash
go build ./cmd/server
go test ./... -count=1
go vet ./...
npx tailwindcss -i ./web/static/css/input.css -o ./web/static/css/app.css --minify
docker build --no-cache -t agriconnect-ui-fix .
```

## Test Results

- **92 tests pass** (6 new rate limit + 3 new MIME normalization + 83 existing)
- **0 failures**
- **go vet**: clean
- **go build**: clean

## Docker Results

```
Successfully tagged agriconnect-ui-fix:latest
```

All favicon assets, templates, and compiled Go binary are included in the image.

## Render Environment Variables

Required variables (already configured or should be verified):

```
APP_ENV=production
APP_URL=https://agriconnect-b0hh.onrender.com
GIN_MODE=release
DATABASE_URL=<postgres connection string>
GROQ_API_KEY=<key>
GROQ_TRANSCRIPTION_MODEL=whisper-large-v3
GROQ_VISION_MODEL=llama-3.2-11b-vision-preview
STORAGE_DRIVER=supabase
SUPABASE_URL=<url>
SUPABASE_SECRET_KEY=<key>
SUPABASE_STORAGE_BUCKET=crop-diagnosis-images
JWT_ACCESS_SECRET=<secret>
JWT_REFRESH_SECRET=<secret>

# Optional with defaults:
RATE_LIMIT_API_PER_MINUTE=20
RATE_LIMIT_WEATHER_PER_MINUTE=30
MAX_AUDIO_SIZE_MB=10
ALLOWED_AUDIO_TYPES=audio/webm,audio/wav,audio/mpeg,audio/mp4,audio/ogg
MAX_IMAGE_SIZE_MB=5
ALLOWED_IMAGE_TYPES=image/jpeg,image/png,image/webp
MAX_RECORDING_SECONDS=60
```

## Remaining Limitations

1. **Docker Alpine mirror**: First build attempt failed due to transient Alpine mirror I/O error; retry succeeded.
2. **Recorder browser support**: iOS Safari may not support `MediaRecorder` API. A platform-specific fallback is not implemented.
3. **Offline mode**: No Service Worker or offline cache for the weather card.
4. **Diagnosis test coverage**: No handler-level tests for the new navigation routes (the diagnosis form already had tests).
