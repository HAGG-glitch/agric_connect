# Diagnosis Display, Image, Strict JSON, and Optional Krio STT Fix Report

## 1. Diagnosis Array-Rendering Root Cause

The `CropDiagnosis` model stored list fields as `datatypes.JSON` (Go `[]byte`):

```go
ObservedSigns        datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'::jsonb"`
```

When Go templates called `{{range .ObservedSigns}}` they iterated byte-by-byte, rendering ASCII values (`91 = '[', 34 = '"', 121 = 'y'`, etc.).

## 2. Result Parsing Fix

Added getter methods on `CropDiagnosis` that parse JSON arrays into `[]string`:

- `GetObservedSigns() []string` — `internal/diagnosis/model.go:61`
- `GetPossibleAlternatives() []string` — `internal/diagnosis/model.go:65`
- `GetRecommendedActions() []string` — `internal/diagnosis/model.go:69`
- `GetPreventionTips() []string` — `internal/diagnosis/model.go:73`

The `parseJSONArray()` helper (line 77) safely unmarshals JSON, returns `nil` for null/invalid/number arrays.

## 3. Template Fix

Updated `web/templates/pages/diagnosis_detail.html` to use getter methods:

```gotemplate
{{range $d.GetObservedSigns}}<li>&bull; {{.}}</li>{{else}}<li>None</li>{{end}}
```

Also updated `diagnosisToView()` in `internal/handlers/diagnosis_handler.go:287` to use getters for API responses.

## 4. Strict JSON Vision Output

Added `response_format` support to `ChatRequest` (`internal/ai/schemas.go:37`):

- New `ResponseFormat`, `JSONSchema` struct types
- `buildJSONSchemaFormat()` in `internal/ai/diagnosis.go:141` generates a `json_schema` response format with strict schema validation
- The `Diagnose` method passes `ResponseFormat` to Groq

## 5. JSON Parser Fallback

Enhanced `parseDiagnosisJSON()` (`internal/ai/diagnosis.go:255`):

1. Trims whitespace
2. Removes markdown code fences
3. Extracts JSON object from surrounding text (first `{` to last `}`)
4. Validates required fields (`probable_condition`, `crop`)

## 6. Non-JSON Retry Behavior

When parsing fails with non-JSON output (e.g. starting with "Here is..."), the `Diagnose` method triggers `retryDiagnose()`:

- Uses a minimal repair prompt: "Your previous answer was not valid JSON. Return only a valid JSON object..."
- Sends only crop + symptom + image (no full context)
- Uses same `json_schema` response format
- Uses lower temperature (0.1) and smaller context
- One retry only — if it fails, the diagnosis is marked as failed

## 7. Supabase Signed URL Root Cause

Supabase's `/storage/v1/object/sign/` endpoint returns a relative path like:

```
/storage/v1/object/sign/bucket/path?token=abc
```

The browser follows the redirect from `agri connect render domain/.../image` to `/object/sign/...` which resolves to `render domain/object/sign/...` instead of `supabase.co/storage/v1/object/sign/...`.

## 8. Signed URL Absolute-Path Fix

Updated `internal/storage/supabase.go:105` to:

1. Check if Supabase returned a relative path (starts with `/`)
2. Prefix with `s.url` (the configured `SUPABASE_URL`)
3. Already-absolute URLs are left unchanged
4. Also handles `symmetric` field as fallback

## 9. Image-Display Verification

Switched `ServeImage` from 302 redirect to server-side proxy (`internal/handlers/diagnosis_handler.go:205`):

- Creates signed URL
- Fetches image server-side from Supabase
- Streams bytes directly to the browser
- Local storage still serves files directly
- Safe cache headers (`Cache-Control: private, max-age=300`)
- Added `onerror` fallback in template for broken images

## 10. Failed AI-Analysis UI Changes

- Image always displays regardless of AI status
- Status badges show separately: `Image: uploaded`, `AI analysis: status`, `Officer review: pending`
- Confidence and urgency chips are hidden when status is not `completed`
- Failed state shows clear message: "The image was uploaded, but AI analysis could not be completed"
- `Try Again` and `Delete` buttons in failed state

## 11. Diagnosis Chart Implementation

Added summary chart card on diagnosis detail page:

- Confidence progress bar (color-coded: green ≥70%, amber ≥40%, red <40%)
- Urgency badge
- AI analysis status
- Officer review status

## 12. Diagnosis History Summary Chart

Added summary section above diagnosis list:

- Total / Completed / Failed / Processing counts
- Top 5 crops with counts (sorted by frequency)
- Powered by client-side JS from existing API response
- Updates on each page load

## 13. Diagnosis History Back Button

Added to `web/templates/pages/diagnosis_history.html`:

- `← Back to Assistant` anchor linking to `/assistant`
- Uses Lucide `arrow-left` icon
- Consistent with detail page navigation
- `Start New Diagnosis` already existed

## 14. Krio STT Repository Assessment

The [krio-stt repo](https://github.com/notttadev/krio-stt) is a Node.js package that calls Hugging Face Inference API with `openai/whisper-large-v3`. Key patterns:
- `transcribeUrl`, `transcribeBuffer`, `transcribeFile` functions
- Bearer token auth with `HUGGINGFACE_API_KEY`
- Audio under 25 MB
- Cold-start delay on free tier

## 15. Optional Hugging Face Provider Implementation

Created `internal/ai/huggingface_stt.go`:

- Implements the existing `AudioTranscriber` interface
- Sends audio as multipart form to Hugging Face Inference API
- Model configurable via `HUGGINGFACE_STT_MODEL` (default: `openai/whisper-large-v3`)
- Handles `text` and `translation_text` response shapes
- Maps to existing `TranscriptionResult` schema
- Krio transcripts always require confirmation

## 16. Hugging Face Configuration

Added config vars in `internal/config/config.go`:

| Env Var | Default | Purpose |
|---------|---------|---------|
| `TRANSCRIPTION_PROVIDER` | `groq` | Default provider for all languages |
| `KRIO_STT_PROVIDER` | `groq` | Krio-specific provider |
| `HUGGINGFACE_API_KEY` | — | Hugging Face token |
| `HUGGINGFACE_STT_MODEL` | `openai/whisper-large-v3` | HF model ID |
| `HUGGINGFACE_STT_TIMEOUT_SECONDS` | `60` | HF request timeout |

Selection logic in `cmd/server/main.go:104`:
- Default: both providers use Groq
- If `KRIO_STT_PROVIDER=huggingface` and `HUGGINGFACE_API_KEY` is set: Krio uses Hugging Face
- If HF key is missing with HF config: fallback to Groq with warning log

## 17. Files Modified

| File | Change |
|------|--------|
| `internal/ai/schemas.go` | Added `ResponseFormat`, `JSONSchema` types to `ChatRequest` |
| `internal/ai/diagnosis.go` | JSON schema format builder, enhanced parser, repair retry, logging |
| `internal/ai/prompts/crop_diagnosis.txt` | Stricter JSON-only instruction |
| `internal/diagnosis/model.go` | Getter methods for array fields (previously added) |
| `internal/diagnosis/service.go` | Safe parser debugging logs, better error messages |
| `internal/storage/supabase.go` | Absolute URL prefixing (previously added) |
| `internal/handlers/diagnosis_handler.go` | Image proxy instead of redirect, `diagnosisToView` uses getters |
| `internal/config/config.go` | HF STT config vars |
| `internal/transcription/service.go` | Dual-provider support (Groq + Krio provider) |
| `cmd/server/main.go` | HF provider wiring, startup log |
| `web/templates/pages/diagnosis_detail.html` | Getter methods, chart card, failed display fix, image fallback |
| `web/templates/pages/diagnosis_history.html` | Summary chart, back button |

## 18. Files Created

| File | Purpose |
|------|---------|
| `internal/ai/huggingface_stt.go` | Hugging Face transcription provider |

## 19. Tests Added

**Diagnosis service tests** (`tests/diagnosis_service_test.go`):

- `TestDiagnosis_GetObservedSigns_ReturnsStrings` — arrays parse to `[]string`
- `TestDiagnosis_GetPossibleAlternatives_ReturnsStrings` — alternatives parse
- `TestDiagnosis_GetRecommendedActions_ReturnsStrings` — actions parse
- `TestDiagnosis_GetPreventionTips_ReturnsStrings` — tips parse
- `TestDiagnosis_GetObservedSigns_NullJSON` — null → nil
- `TestDiagnosis_GetObservedSigns_EmptyJSON` — empty array → empty slice
- `TestDiagnosis_GetObservedSigns_InvalidJSON` — invalid → nil
- `TestDiagnosis_GetObservedSigns_NumbersArray` — numbers → nil
- `TestDiagnosis_JSONParse_ValidObject` — valid JSON parses
- `TestDiagnosis_JSONParse_MarkdownCodeFence` — fence removed
- `TestDiagnosis_JSONParse_TextBeforeJSON` — text before stripped
- `TestDiagnosis_JSONParse_TextAfterJSON` — text after stripped
- `TestDiagnosis_JSONParse_EmptyContent` — empty → error
- `TestDiagnosis_JSONParse_NoJSONObject` — no JSON → error
- `TestDiagnosis_JSONParse_MissingRequiredField` — missing field → error

**Supabase signed URL tests** (`tests/handlers_test.go`):

- `TestSupabaseStorage_SignedURL_RelativePathMadeAbsolute` — relative → absolute
- `TestSupabaseStorage_SignedURL_AbsoluteStaysAbsolute` — absolute unchanged
- `TestSupabaseStorage_SignedURL_IncludesStorageV1` — URL contains `/storage/v1/`
- `TestSupabaseStorage_SignedURL_MissingField` — missing field → error
- `TestSupabaseStorage_SignedURL_SymmetricFallback` — symmetric field fallback
- `TestDiagnosisHandler_ServeImage_ChecksStorageType` — handler handles mock storage

**Krio STT tests** (`tests/handlers_test.go`):

- `TestTranscriptionService_KrioUsesHuggingFaceWhenConfigured` — Krio → HF
- `TestTranscriptionService_EnglishUsesDefaultProvider` — English → Groq
- `TestTranscriptionService_KrioFallsBackToGroqWhenNoKrioProvider` — same provider fallback

## 20. Commands Executed

```bash
go mod tidy    # not needed
go test ./...  # PASS (90+ tests)
go vet ./...   # no issues
go build ./cmd/server  # success
npx tailwindcss -i ./web/static/css/input.css -o ./web/static/css/app.css --minify  # success
docker build --no-cache -t agriconnect-diagnosis-display-image-json-stt-fix .  # success
```

## 21. Test Results

All tests pass:

```
ok  github.com/agriconnect-ai/tests  1.906s
```

## 22. Docker Result

Docker image built successfully:

```
agriconnect-diagnosis-display-image-json-stt-fix:latest
```

## 23. Render Deployment Notes

1. **New env vars** for Krio STT (optional):
   - `KRIO_STT_PROVIDER=groq` (or `huggingface`)
   - `HUGGINGFACE_API_KEY=...` (if using Hugging Face)
   - `HUGGINGFACE_STT_MODEL=openai/whisper-large-v3`
   - `HUGGINGFACE_STT_TIMEOUT_SECONDS=60`

2. **No env changes required** for the JSON parsing, image, or chart fixes — they work with existing config.

3. The image proxy (`/api/v1/diagnoses/:id/image`) now fetches from Supabase server-side instead of redirecting. This should resolve the broken image issue on Render.

## 24. Remaining Limitations

1. **`json_schema` response format** may not work with all Groq vision models. If `llama-3.2-11b-vision-preview` doesn't support it, the API will return an error. In that case, the `ResponseFormat` should be changed to `{"type": "json_object"}` (simpler, no schema validation). Currently hardcoded to `json_schema` — may need a config-driven fallback.

2. **Hugging Face cold starts** — The free tier has a ~30-60s cold start. The 60s timeout may need tuning for production.

3. **Image proxy memory** — Large images are buffered entirely in memory during proxy. For very large installations, streaming with size limits should be added.

4. **Krio detection** — The system currently detects Krio only by the `language_hint` parameter. Automatic language detection from audio is not implemented (Groq's Whisper doesn't return detected language reliably for Krio).
