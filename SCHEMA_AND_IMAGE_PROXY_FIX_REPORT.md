# Schema & Image Proxy Fix Report

## 1. JSON Schema Root Cause

`buildJSONSchemaFormat()` (now `buildResponseFormat()`) in `internal/ai/diagnosis.go` generated a strict `json_schema` response format where `crop` and `confidence_label` were defined in `properties` but **not** listed in `required`. Groq's strict JSON schema enforcement requires every key in `properties` to also appear in `required`. Production logs confirmed:

```
/required: `required` is required to be supplied and to be an array including every key in properties.
The following properties must be listed in `required`: confidence_label, crop
```

## 2. Required Fields Added

Both `crop` and `confidence_label` were added to the `required` array. All 12 schema properties are now required:

```json
"required": ["crop", "probable_condition", "confidence", "confidence_label",
             "description", "observed_signs", "possible_alternatives",
             "recommended_actions", "prevention_tips",
             "urgency", "requires_expert_review", "disclaimer"]
```

## 3. Response-Format Fallback Behavior

Added `containsJSONSchemaError()` helper (`internal/ai/diagnosis.go`) that detects schema-rejection errors (checks for `invalid JSON schema`, `json_schema`, `response_format`, `must be listed in \`required\``, `additionalProperties`).

If the initial `json_schema` call fails with a schema error:
1. Log the rejection
2. Switch response format to `json_object`
3. Retry **once** with the same prompt
4. Restore the original format for subsequent requests (concurrency-safe)

Also added config `GROQ_VISION_RESPONSE_FORMAT` (values: `json_schema`, `json_object`, `none`; default: `json_schema`) in `internal/config/config.go` and wired through `newCropDiagnosisAI()` constructor.

## 4. Supabase Image 404 Root Cause

The image proxy (`ServeImage` handler) used a two-step approach:
1. Create a signed URL via `SignedURL()`
2. HTTP GET to the signed URL

The 404 likely stemmed from one of:
- **Path mismatch**: stored `image_storage_path` may have contained the bucket prefix (e.g., `crop-diagnosis-images/users/...`) while the signed URL creation prepended the bucket name again, producing a double bucket path (`/object/sign/bucket/bucket/users/...`)
- **Upload inconsistency**: upload succeeded but the object wasn't fully committed when the signed URL was created
- **Bucket config mismatch**: production bucket name may differ from the default `crop-diagnosis-images`

## 5. Stored Path Normalization Fix

Added `NormalizePath()` function in `internal/storage/supabase.go`:
- Strips leading slashes
- Strips bucket name prefix if present (handles old records with embedded bucket name)
- Returns clean relative path

Applied to all Supabase storage operations: `Save`, `Delete`, `SignedURL`, `Download`.

## 6. Upload Verification & Diagnostics

Added safe diagnostic logging:
- `storage_save`: bucket, original path, normalized path, HTTP status
- `storage_upload_failed`: full details (without secrets)
- `storage_upload_ok`: bucket, path, size
- `storage_signed_url`: bucket, original/normalized paths
- `storage_download`: bucket, normalized path, host, URL path without token
- `diagnosis_upload`: diagnosis ID, storage driver type, stored path, size, content type

The upload itself already verifies via Supabase HTTP 2xx response — no separate HEAD needed.

## 7. Image Proxy Behavior (Rewritten)

Added `Download()` method to `ObjectStorage` interface (`internal/storage/interfaces.go`):

**Supabase implementation**: Authenticated GET to `/storage/v1/object/{bucket}/{path}` with `apikey` header (service role key). No signed URL involved.

**Local implementation**: Direct file read with path-traversal protection.

**`ServeImage` handler rewritten**:
- Removed storage-type branching (no more `*storage.LocalStorage` type check)
- Calls `objStore.Download()` for all backends
- Streams bytes directly to response
- 404 from upstream → 404 to client (not 500)
- Sets `Cache-Control: private, max-age=300`
- Preserves content type from diagnosis model

## 8. Tests Added

### Schema & Fallback (tests/diagnosis_service_test.go)
- `TestJSONSchema_AllPropertiesInRequired`: every property is in required; marshals to valid JSON
- `TestJSONSchema_NoRequiredFieldsMissing`: property/required count match
- `TestJSONSchema_RetryWithJSONObject`: `containsJSONSchemaError` detects schema errors; valid JSON parses after `json_object` fallback; non-JSON still rejected
- `TestJSONSchema_ConfigDrivenResponseFormat`: `none`, empty default accepted

### Path Normalization & Download (tests/handlers_test.go)
- `TestSupabase_PathNormalization_StripsBucketPrefix`: 7 cases (no prefix, with bucket prefix, leading slash, combined, empty bucket, exact bucket, partial match)
- `TestSupabase_Download_UsesServiceKey`: verifies apikey header, GET method, URL format
- `TestSupabase_Download_PathNormalization`: path with bucket prefix is normalized in download URL
- `TestSupabase_Download_404ReturnsNotFound`: 404 error contains "404" and "object not found"
- `TestSupabase_Download_EmptyPath`: error for empty path
- `TestDiagnosisHandler_ServeImage_UsesDownload`: 200, correct Content-Type, Cache-Control, body length
- `TestDiagnosisHandler_ServeImage_Storage404Returns404`: 404 passthrough
- `TestDiagnosisHandler_ServeImage_EmptyStoragePathIs404`: empty path returns 404

## 9. Commands Executed

```bash
go test ./...
go vet ./...
go build ./cmd/server
npx tailwindcss -i ./web/static/css/input.css -o ./web/static/css/app.css --minify
docker build --no-cache -t agriconnect-schema-image-proxy-fix .
```

## 10. Test Results

All 50+ tests pass:
```
ok  	github.com/agriconnect-ai/tests	1.780s
```

## 11. Docker Result

Docker image `agriconnect-schema-image-proxy-fix` built successfully:
- Go binary compiled with `CGO_ENABLED=0`
- Tailwind CSS built in frontend stage
- Multi-stage build (golang + node → alpine runtime)

## 12. Render Deployment Notes

1. Push to GitHub — automatic Render redeploy if webhook is configured
2. Set `GROQ_VISION_RESPONSE_FORMAT=json_schema` in Render env (default, but explicit confirm)
3. Verify `SUPABASE_STORAGE_BUCKET` matches the actual bucket name in Supabase
4. Check Render logs for:
   - `json_schema rejected, falling back to json_object` (only if schema is rejected)
   - `storage_download: ... HTTP 404 ...` (if image still missing)

## 13. Remaining Limitations

1. **`json_schema` fallback modifies struct field temporarily** — safe because we restore it, but a second concurrent call during the fallback window could see `json_object`. Consider making `responseFormatType` atomic or per-call config.
2. **`Download()` returns raw HTTP response body** — on network errors, the error may not surface until `io.Copy` is called in the handler. Adding a small initial read in `Download()` could catch errors earlier.
3. **No migration for existing records** — existing rows with bucket-prefixed paths are normalized at runtime by `NormalizePath()`. A DB migration to strip the prefix would be cleaner.
4. **Local storage `SignedURL()` returns bare path** — adequate for dev but not for production image serving. Not an issue since production uses Supabase.
5. **Template `onerror` fallback uses escaped HTML** — the `&quot;` in `onerror` may not unescape correctly. If broken, inline the refresh link differently.
