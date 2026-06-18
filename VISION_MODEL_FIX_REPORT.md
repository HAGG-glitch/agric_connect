# Vision Model Fix Report — 18 June 2026

## Root Cause

`internal/ai/client.go:38` (`Client.Chat`) unconditionally overrode `req.Model` with the client's default model:

```go
req.Model = c.model  // always overwrites, regardless of req.Model
```

Since the single `aiClient` was created with `cfg.GroqChatModel` (`llama-3.1-8b-instant`), every call to `client.Chat()` — including crop diagnosis vision requests — sent `model: "llama-3.1-8b-instant"` to Groq, even though `cropDiagnosisAI.Diagnose()` had correctly set `req.Model` to the vision model.

This caused `context_length_exceeded` errors because `llama-3.1-8b-instant` cannot process image inputs and has a much smaller context window.

## Fix — `Client.Chat` model override

**File:** `internal/ai/client.go:38,78`

Changed both `Chat` and `ChatStream` to only set the client default when `req.Model` is empty:

```go
if req.Model == "" {
    req.Model = c.model
}
```

This allows `cropDiagnosisAI` to explicitly set `req.Model` to the vision model without it being silently replaced.

## Environment Variables

| Variable | Old Default | New Default | Render Setting |
|----------|-------------|-------------|----------------|
| `GROQ_VISION_MODEL` | `llama-3.2-11b-vision-preview` | unchanged | `meta-llama/llama-4-scout-17b-16e-instruct` |
| `GROQ_CHAT_MODEL` | `llama-3.1-8b-instant` | unchanged | `llama-3.1-8b-instant` |
| `MAX_DIAGNOSIS_CONTEXT_CHARS` | `1500` | **500** | override on Render |
| `GROQ_VISION_MAX_OUTPUT_TOKENS` | `512` | **300** | override on Render |

## Code Wiring Fixed

| File | Change |
|------|--------|
| `internal/ai/client.go` | `Chat`/`ChatStream` only override `req.Model` if empty |
| `cm d/server/main.go` | Startup logs for all 3 models; warns if vision model misconfigured |
| `internal/config/config.go` | Defaults reduced: `MaxDiagnosisContextChars=500`, `GroqVisionMaxOutputTokens=300` |
| `internal/ai/diagnosis.go` | Added `ImageURL` field; prefers URL over base64 encoding |
| `internal/diagnosis/service.go` | Generates signed Supabase URL from storage path; passes as `ImageURL`; falls back to base64 only on retry |

## Image Optimization Behavior

Always applied to every image before AI submission:

1. **Resize** to max dimension 768px (preserves aspect ratio, nearest-neighbor scaling)
2. **Re-encode** as JPEG at quality 75
3. Result is verified < 300 KB for typical inputs
4. Images under 768px in both dimensions are not upscaled

**Compression retry** (only if first call fails with `input_length`):
- Resize to max dimension 512px at quality 70
- Uses base64 (not URL) in case signed URL was the issue

**Original image in storage** — always the untouched upload. Optimization only affects the copy sent to AI.

## Image Input to Groq

1. **Preferred:** Short-lived signed Supabase URL (`10 min` expiry) passed directly in the message — avoids base64 overhead entirely
2. **Fallback:** Optimized JPEG as `data:image/jpeg;base64,...` — used when signed URL generation fails

Groq's API accepts image URLs up to its full request size limit, while base64 is limited to ~4 MB. Signed URLs are therefore the safer approach.

## Tests Added

| Test | File | What it proves |
|------|------|----------------|
| `TestAIClient_ChatRespectsRequestModel` | `tests/ai_client_test.go` | Chat uses req.Model when set; falls back to default only when empty |
| `TestAIClient_ModelNotOverridden` | `tests/ai_client_test.go` | Vision model passes through without being overwritten |
| `TestDiagnosis_ImageOptimizationReducesSize` | `tests/diagnosis_service_test.go` | 2000×2000 PNG → < original size, ≥ 256×256, ≤ 768×768 |
| `TestDiagnosis_ImageOptimizationPreservesSmall` | `tests/diagnosis_service_test.go` | 100×100 image is not upscaled |
| `TestDiagnosis_ContextCapped` | `tests/diagnosis_service_test.go` | Knowledge context truncated to `MaxDiagnosisContextChars` |
| `TestDiagnosis_AIUsesVisionModel` | `tests/diagnosis_service_test.go` | Diagnosis completes successfully with vision model wired in |

## Verification

```
go test ./...      → 80 tests PASS
go vet ./...       → PASS
go build ./cmd/server → PASS
npx tailwindcss    → Done in 1019ms
docker build       → agriconnect-vision-model-fix (success)
```

## Render Deployment Notes

1. Set `GROQ_VISION_MODEL=meta-llama/llama-4-scout-17b-16e-instruct` in Render env
2. Set `MAX_DIAGNOSIS_CONTEXT_CHARS=500` (already the new code default)
3. Set `GROQ_VISION_MAX_OUTPUT_TOKENS=300` (already the new code default)
4. `GROQ_CHAT_MODEL=llama-3.1-8b-instant` remains unchanged
5. Deploy the new image — startup logs will print:
   ```
   chat_model=llama-3.1-8b-instant
   vision_model=meta-llama/llama-4-scout-17b-16e-instruct
   transcription_model=whisper-large-v3
   ```
6. If vision model is misconfigured (equals chat model or is `llama-3.1-8b-instant`), a `WARNING` log is emitted at startup.
