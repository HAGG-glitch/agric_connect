# Production Bugfix Summary — 18 June 2026

Four bugs identified from Render production logs have been fixed across 7 files.

## Bug 1 — Transcription language_hint sent verbatim to Groq

**Symptoms:** `language=english` in multipart form causes Groq 400 error (expects ISO `en`).

**Files changed:** `internal/ai/transcription.go`

**Fix:** Added `normalizeTranscriptionLanguage()` that maps:
- `english` / `en` → `"en"` (include field)
- `auto` / `krio` / empty → omit the `language` field entirely

The multipart write site now uses the normalized result instead of `input.LanguageHint` directly.

## Bug 2 — assistant.js runs on diagnosis pages

**Symptoms:** Diagnosis pages fire `GET /api/v1/conversations` calls logged as "failed to load conversation"; unnecessary chat JS executes on all pages because `layout_foot.html` loads `assistant.js` globally.

**Files changed:** `web/static/js/assistant.js`, `web/templates/pages/assistant.html`

**Fix:** Added guard:
```js
document.addEventListener('DOMContentLoaded', () => {
  const assistantRoot = document.querySelector("[data-assistant-root]");
  if (!assistantRoot) return;
  …
});
```

`data-assistant-root="true"` attribute placed on the `<div id="app">` in `assistant.html` only. `recorder.js` was **not** changed — it defines `window.Recorder` in an IIFE but only calls `Recorder.init()` from the `voice_recorder` partial (included solely in `assistant.html`).

## Bug 3 — Groq vision requests rejected as too long

**Symptoms:** Vision API returns "input_length" / 400 errors; hardcoded `MaxTokens: 2000` and raw 5 MB images create huge base64 payloads; knowledge context is unbounded.

**Files changed:** `internal/ai/diagnosis.go`, `internal/diagnosis/service.go`, `cmd/server/main.go`

**Fix — three parts:**

1. **MaxTokens per config** — `cropDiagnosisAI` struct now holds `maxTokens` (was hardcoded 2000). `NewCropDiagnosisAI(client, model, maxTokens)` reads `cfg.GroqVisionMaxOutputTokens` (default 512). Wire call updated in `main.go`.

2. **Image pre-compression** — New `optimizeImageForAI()` in `service.go` re-encodes JPEG/PNG images >300 KB as JPEG Q60 before passing to the AI layer, cutting base64 payload drastically. New `compressImage()` re-encodes at Q30 for the retry path.

3. **Knowledge context cap** — `knowledgeCtx` is truncated to `cfg.MaxDiagnosisContextChars` (default 1500) before building the AI input.

4. **Single retry on input_length** — If first Diagnose call returns an error containing `"input_length"`, the image is re-compressed at Q30 and retried once before falling back to `"failed"` status.

## Bug 4 — CropDiagnosis BeforeCreate has wrong GORM signature

**Symptoms:** GORM log warning "BeforeCreate signature doesn't match BeforeCreateInterface" (ignored at runtime but indicates future incompatibility).

**Files changed:** `internal/diagnosis/model.go`, `internal/models/agricultural_document.go`

**Fix:** Changed signature from `BeforeCreate(_ interface{}) error` to `BeforeCreate(tx *gorm.DB) error` for both `CropDiagnosis` and `AgriculturalDocument` (same issue). Added `"gorm.io/gorm"` import where missing.

## Verification

| Check | Result |
|-------|--------|
| `go test ./...` | PASS |
| `go vet ./...` | PASS |
| `go build ./cmd/server` | PASS |
| `npx tailwindcss` | PASS |
| `docker build` | PASS (94.6 MB) |

All changes have been committed and the Docker image is ready for deployment.
