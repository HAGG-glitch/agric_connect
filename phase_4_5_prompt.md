# AgriConnect AI — Phase 4 and Phase 5 Integration Prompt

You are a senior Go full-stack engineer working inside the existing AgriConnect AI repository.

Phases 1–3 are already implemented and must remain working:

- Agricultural AI chat using Gin, PostgreSQL, GORM, and Groq
- English and Krio text responses
- Agricultural knowledge retrieval
- Sierra Leone district weather using Open-Meteo
- Anonymous browser-user ownership
- Responsive Go templates, Tailwind CSS, HTMX, and vanilla JavaScript
- Docker, migrations, tests, and documentation

Your task is to inspect the existing repository and integrate:

- **Phase 4: AI Crop Image Diagnosis**
- **Phase 5: Voice Recording and Speech-to-Text Transcription**

Do not merely produce a plan. Modify the repository, create missing files, connect all layers, add migrations and tests, compile the frontend, run the Go test/build commands, and update the documentation.

---

## 1. Mandatory working rules

1. Read `RECOVERY_REPORT.md` and inspect the complete repository before editing.
2. Run the current test suite and build first to establish a baseline.
3. Preserve all working Phase 1–3 behavior.
4. Use the existing application entry point, router, configuration, database connection, Groq client, dependency injection, logging, and error conventions.
5. Do not create a second server, duplicate router, duplicate database connection, or duplicate configuration loader.
6. Never expose Groq or Supabase credentials in HTML or JavaScript.
7. Keep all AI model names configurable through environment variables.
8. Continue using the anonymous UUID cookie as the owner of diagnoses and conversations.
9. Do not present AI diagnosis as certain.
10. Do not invent pesticide names or dosages.
11. Do not claim Krio transcription is fully reliable.
12. Do not add empty placeholder files.
13. Do not remove existing tests to make the build pass.
14. Record all work in `PHASE_4_5_REPORT.md`.

Before modification, run:

```bash
go test ./...
go vet ./...
go build ./cmd/server
```

Record the baseline results in `PHASE_4_5_REPORT.md`.

---

# PHASE 4 — AI CROP IMAGE DIAGNOSIS

## 2. Required farmer workflow

A farmer must be able to:

1. Open `/diagnose`.
2. Select a crop, district, preferred language, and affected plant part.
3. Describe symptoms and field conditions.
4. Upload a JPEG, PNG, or WebP crop image.
5. Preview, replace, or remove the image before submission.
6. Submit the case.
7. See upload and analysis progress.
8. Receive a structured preliminary AI result.
9. View diagnosis history.
10. Open a diagnosis detail page.
11. Delete their own diagnosis.
12. Continue the diagnosis in the existing AI assistant.

The AI request must use both the image and the farmer's written field information.

The interface must clearly display:

> This is a preliminary AI assessment and may be incorrect. Confirm serious or uncertain crop problems with a qualified agricultural extension officer.

---

## 3. Suggested package additions

Integrate equivalent packages into the existing architecture:

```text
internal/
├── diagnosis/
│   ├── model.go
│   ├── repository.go
│   ├── service.go
│   ├── schemas.go
│   └── validator.go
├── transcription/
│   ├── service.go
│   ├── schemas.go
│   └── validator.go
├── storage/
│   ├── interfaces.go
│   ├── local.go
│   └── supabase.go
├── handlers/
│   ├── diagnosis_handler.go
│   └── transcription_handler.go
└── ai/
    ├── diagnosis.go
    ├── transcription.go
    └── prompts/
        └── crop_diagnosis.txt

web/
├── templates/
│   ├── pages/
│   │   ├── diagnose.html
│   │   ├── diagnosis_history.html
│   │   └── diagnosis_detail.html
│   └── partials/
│       ├── diagnosis_form.html
│       ├── diagnosis_result.html
│       ├── diagnosis_card.html
│       ├── image_preview.html
│       ├── voice_recorder.html
│       └── transcription_result.html
└── static/js/
    ├── diagnosis.js
    └── recorder.js

migrations/
├── 000002_create_crop_diagnoses.up.sql
└── 000002_create_crop_diagnoses.down.sql
```

Do not duplicate an equivalent existing package. Follow the repository's established organization.

---

## 4. Configuration additions

Extend the configuration loader and `.env.example`:

```env
GROQ_VISION_MODEL=
GROQ_TRANSCRIPTION_MODEL=

STORAGE_DRIVER=local
LOCAL_UPLOAD_DIR=./data/uploads

SUPABASE_URL=
SUPABASE_SERVICE_ROLE_KEY=
SUPABASE_STORAGE_BUCKET=crop-diagnosis-images

MAX_IMAGE_SIZE_MB=5
MAX_AUDIO_SIZE_MB=10
MAX_RECORDING_SECONDS=60

ALLOWED_IMAGE_TYPES=image/jpeg,image/png,image/webp
ALLOWED_AUDIO_TYPES=audio/webm,audio/wav,audio/mpeg,audio/mp4,audio/ogg

DIAGNOSIS_REQUEST_TIMEOUT_SECONDS=90
TRANSCRIPTION_REQUEST_TIMEOUT_SECONDS=90
```

Rules:

- `STORAGE_DRIVER` must support `local` and `supabase`.
- Development may use local storage.
- Production may use private Supabase Storage.
- Never log service-role keys.
- Production must fail clearly when required provider configuration is missing.
- Do not hard-code vision or transcription model names.

---

## 5. Storage abstraction

Create a testable storage interface:

```go
type ObjectStorage interface {
    Save(ctx context.Context, input SaveObjectInput) (StoredObject, error)
    Delete(ctx context.Context, path string) error
    SignedURL(ctx context.Context, path string, expiry time.Duration) (string, error)
}
```

### Local storage

- Store files under `LOCAL_UPLOAD_DIR`.
- Generate random, non-guessable object paths.
- Do not expose the upload directory as a public static folder.
- Serve images only through an ownership-checked handler.
- Prevent path traversal.
- Delete the stored image when its diagnosis is deleted.

### Supabase Storage

- Use a private bucket.
- Upload only through the Go backend.
- Keep the service-role key on the backend.
- Store only the object path in PostgreSQL.
- Generate short-lived signed URLs for authorized owners.
- Delete the object when the diagnosis is deleted.

Recommended object path:

```text
anonymous-users/{user_id}/diagnoses/{diagnosis_id}/{random_filename}
```

Do not use the original filename as the storage key.

---

## 6. Database migration

Create `crop_diagnoses` with UUID primary keys:

```sql
CREATE TABLE crop_diagnoses (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,

    crop VARCHAR(100) NOT NULL,
    district VARCHAR(100),
    preferred_language VARCHAR(20) NOT NULL DEFAULT 'english',

    plant_part VARCHAR(100),
    symptom_description TEXT NOT NULL,
    symptoms_started_at DATE,
    affected_percentage NUMERIC(5,2),

    recent_weather TEXT,
    fertiliser_history TEXT,
    pesticide_history TEXT,

    image_storage_path TEXT NOT NULL,
    image_original_name VARCHAR(255),
    image_content_type VARCHAR(100) NOT NULL,
    image_size_bytes BIGINT NOT NULL,
    image_sha256 VARCHAR(64),

    probable_condition VARCHAR(255),
    confidence NUMERIC(5,2),
    confidence_label VARCHAR(20),
    description TEXT,

    observed_signs JSONB NOT NULL DEFAULT '[]'::jsonb,
    possible_alternatives JSONB NOT NULL DEFAULT '[]'::jsonb,
    recommended_actions JSONB NOT NULL DEFAULT '[]'::jsonb,
    prevention_tips JSONB NOT NULL DEFAULT '[]'::jsonb,

    urgency VARCHAR(20),
    requires_expert_review BOOLEAN NOT NULL DEFAULT TRUE,
    disclaimer TEXT,

    raw_ai_result JSONB,
    model VARCHAR(150),

    status VARCHAR(30) NOT NULL DEFAULT 'processing',
    error_message TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

Create indexes on:

- `user_id`
- `created_at`
- `crop`
- `probable_condition`
- `status`

Allowed application values:

```text
status: processing, completed, failed
urgency: low, medium, high, urgent
confidence_label: low, medium, high
```

Create both up and down migrations. Ensure Go models match the migration exactly.

---

## 7. Diagnosis form fields

Collect:

```text
crop
district
preferred_language
plant_part
symptom_description
symptoms_started_at
affected_percentage
recent_weather
fertiliser_history
pesticide_history
image
```

Initial supported crops:

- Rice
- Cassava
- Maize
- Groundnut
- Cocoa
- Coffee
- Oil Palm
- Tomato
- Pepper
- Other

Plant parts:

- Whole plant
- Leaf
- Stem
- Root
- Fruit
- Seed
- Flower
- Bark
- Tuber
- Pod
- Other

Validation:

- Crop, symptoms, and image are required.
- District must be one of the existing supported Sierra Leone districts.
- Language must be `english` or `krio`.
- Affected percentage must be between 0 and 100.
- Enforce configured size limits.

---

## 8. Server-side image validation

Implement strict backend validation:

1. Limit the multipart request body before parsing.
2. Do not trust the extension or browser MIME type.
3. Detect JPEG, PNG, or WebP from file signatures.
4. Reject SVG and unsupported files.
5. Decode image metadata to confirm it is a real image.
6. Reject zero or unreasonable dimensions.
7. Add a reasonable maximum pixel-count limit.
8. Calculate SHA-256.
9. Generate a random server filename.
10. Never preserve user-controlled paths.
11. Never render uploaded content as HTML.

Client-side preview is only a convenience; backend validation is authoritative.

---

## 9. Groq vision integration

Reuse or extend the existing Groq client.

Create a testable interface:

```go
type CropDiagnosisAI interface {
    Diagnose(ctx context.Context, input DiagnosisAIInput) (DiagnosisAIResult, error)
}
```

The request must include:

- image bytes or provider-compatible image representation
- crop
- district
- plant part
- symptom description
- symptoms start date
- affected percentage
- recent weather
- fertiliser history
- pesticide history
- preferred language
- relevant Phase 2 agricultural documents

Use the model configured by `GROQ_VISION_MODEL`.

If the configured provider/model cannot process images:

- return a clear configuration error
- mark the diagnosis failed
- do not fabricate a result

---

## 10. Crop diagnosis prompt

Create `internal/ai/prompts/crop_diagnosis.txt` with rules equivalent to:

```text
You are AgriConnect AI's crop-health screening assistant for farmers
in Sierra Leone.

Analyze the image together with the farmer's field information.
This is preliminary screening, not laboratory confirmation.

Rules:
1. Never claim complete certainty.
2. Explain when the image is unclear or insufficient.
3. Include reasonable alternative causes.
4. Do not invent pesticide names or dosages.
5. When chemicals may be relevant, tell the farmer to follow a locally
   registered product label, use protective equipment, and consult an
   agricultural extension officer.
6. Mark serious, fast-spreading, uncertain, toxic, or high-impact cases
   for expert review.
7. Use simple English when language is English.
8. Use clear, simple Krio when language is Krio.
9. Do not invent unusual Krio terminology.
10. Return only JSON matching the required schema.
```

---

## 11. Structured diagnosis result

Require and validate a JSON response equivalent to:

```json
{
  "crop": "Cassava",
  "probable_condition": "Cassava Mosaic Disease",
  "confidence": 78,
  "confidence_label": "medium",
  "description": "The visible leaf pattern may be consistent with cassava mosaic disease.",
  "observed_signs": [
    "yellow-green mosaic pattern",
    "distorted leaves"
  ],
  "possible_alternatives": [
    "nutrient deficiency",
    "herbicide damage"
  ],
  "recommended_actions": [
    "Separate severely affected plants",
    "Do not reuse stems from affected plants",
    "Request extension-officer confirmation"
  ],
  "prevention_tips": [
    "Use healthy planting stems",
    "Inspect fields regularly"
  ],
  "urgency": "medium",
  "requires_expert_review": true,
  "disclaimer": "This is a preliminary AI assessment and may be incorrect."
}
```

Server validation must:

- reject malformed JSON
- reject missing required fields
- clamp confidence to 0–100
- validate confidence label and urgency
- limit array lengths and string lengths
- insert the standard disclaimer when missing
- avoid returning raw provider errors

Confidence is model-estimated confidence, not scientific accuracy.

---

## 12. Diagnosis routes

Create and connect:

```text
GET    /diagnose
GET    /diagnoses
GET    /diagnoses/:id

POST   /api/v1/diagnoses
GET    /api/v1/diagnoses
GET    /api/v1/diagnoses/:id
DELETE /api/v1/diagnoses/:id

GET    /api/v1/diagnoses/:id/image
POST   /api/v1/diagnoses/:id/continue-in-chat
```

### `POST /api/v1/diagnoses`

1. Read the anonymous owner UUID from middleware.
2. Limit and parse multipart form data.
3. Validate all fields and the image.
4. Generate the diagnosis UUID.
5. Save the image through `ObjectStorage`.
6. Insert a processing record.
7. Retrieve relevant agricultural knowledge.
8. Call the vision model.
9. Validate the structured result.
10. Update the record to completed.
11. On provider failure, update it to failed.
12. Clean up the stored image if database creation fails.
13. Return safe JSON.

### List and read

- Return only the current anonymous user's records.
- Support pagination.
- Sort newest first.
- Verify ownership on detail access.
- Never expose `raw_ai_result` publicly.

### Delete

- Verify ownership.
- Delete the record.
- Delete the stored image.
- Log partial cleanup failures safely.

### Protected image route

- Verify ownership before serving or signing.
- Set the correct image content type.
- Prevent arbitrary path access.

### Continue in chat

- Verify ownership.
- Create or select a conversation.
- Insert a concise safe diagnosis summary as context.
- Do not insert raw AI JSON.
- Return the conversation ID.

---

## 13. Diagnosis frontend

Create responsive pages that match the current AgriConnect design.

### `/diagnose`

Include:

- Safety notice
- Crop, district, language, and plant-part selectors
- Symptom fields
- Drag-and-drop image area
- Browse button
- Image preview
- Replace and remove actions
- Upload progress
- Analysis progress
- Validation errors
- Submit button

### Result page

Display:

- Uploaded image
- Probable condition
- Preliminary confidence
- Description
- Observed signs
- Possible alternatives
- Recommended actions
- Prevention tips
- Urgency
- Expert-review requirement
- Disclaimer
- Continue in AI chat
- View history

### History page

Display:

- Thumbnail
- Crop
- Probable condition
- Confidence
- Urgency
- Status
- Date
- View and delete actions

Update the existing navigation with:

- AI Assistant
- Diagnose Crop
- Diagnosis History
- Weather

Do not create a second navigation system.

---

## 14. Diagnosis JavaScript

Create or extend `web/static/js/diagnosis.js`.

Required behavior:

- drag and drop
- file picker
- client preview
- replace/remove image
- client size/type warning
- prevent duplicate submissions
- disable submit while active
- show upload/analysis status
- safe error rendering
- redirect or render result on success
- no raw AI-generated HTML
- no frontend secrets

Backend validation remains authoritative.

---

# PHASE 5 — VOICE INPUT AND TRANSCRIPTION

## 15. Transcription architecture

Create a testable interface:

```go
type AudioTranscriber interface {
    Transcribe(ctx context.Context, input TranscriptionInput) (TranscriptionResult, error)
}
```

Suggested result:

```go
type TranscriptionResult struct {
    Text             string
    DetectedLanguage string
    Model            string
    DurationSeconds  float64
    RequiresReview   bool
}
```

Use `GROQ_TRANSCRIPTION_MODEL` from configuration.

Reuse the existing Groq base URL and API key.

---

## 16. Audio validation

Support configured audio types such as:

- WebM
- WAV
- MP3
- M4A/MP4 audio
- OGG

Rules:

1. Limit request size before parsing.
2. Reject empty audio.
3. Validate type server-side where practical.
4. Enforce configured maximum size.
5. Generate no public object.
6. Use temporary files or in-memory buffers safely.
7. Delete temporary audio after success or failure.
8. Do not retain audio by default.
9. Do not trust the original filename.
10. Prevent path traversal.

Recommended browser recording maximum: 60 seconds.

---

## 17. Transcription endpoint

Create:

```text
POST /api/v1/ai/transcribe
Content-Type: multipart/form-data
```

Input:

```text
audio
language_hint
```

Allowed language hints:

- `english`
- `krio`
- `auto`

Response:

```json
{
  "transcript": "Di rice leaf dem get brown spot",
  "detected_language": "unknown",
  "requires_confirmation": true,
  "experimental_krio": true
}
```

Rules:

- Never automatically send the transcript to the AI.
- The farmer must edit or confirm it first.
- Krio transcription must always require confirmation.
- Do not invent a detected language when the provider does not return one.
- Delete temporary audio after processing.
- Return stable public errors.

---

## 18. Browser recorder

Create `web/static/js/recorder.js` and integrate it into the existing assistant page.

Required UI:

- Microphone button
- Recording timer
- Recording indicator
- Stop button
- Cancel button
- Audio playback
- Transcribe button
- Editable transcript field
- Use transcript button
- Retry action
- Experimental Krio notice

Required behavior:

1. Check `navigator.mediaDevices` support.
2. Request permission only after user action.
3. Use `MediaRecorder` when supported.
4. Select a supported recording MIME type.
5. Stop automatically at the configured duration.
6. Release microphone tracks on stop and cancel.
7. Create an audio Blob.
8. Allow playback.
9. Upload through multipart form data.
10. Insert the returned text into the normal chat input.
11. Do not automatically send it.
12. Allow editing before sending.
13. Revoke object URLs.
14. Prevent concurrent recordings.
15. Handle permission denial.
16. Handle unsupported browsers.
17. Handle empty recording and provider failure.

The microphone control must be keyboard and screen-reader accessible.

Display:

> Krio voice transcription is experimental. Please review and correct the transcript before sending it.

Store only the final confirmed text in `ai_messages`. Do not store raw audio by default.

---

## 19. Rate limiting and security

Extend rate limiting so that:

- diagnosis submissions use a stricter limit than normal page requests
- transcription requests use a stricter limit than normal chat
- static files are not unnecessarily blocked

Implement stable public errors for:

### Diagnosis

- image too large
- unsupported image type
- invalid image
- missing crop
- missing symptoms
- unsupported district
- storage unavailable
- vision model unavailable
- invalid AI result
- diagnosis not found
- unauthorized access

### Transcription

- audio too large
- unsupported audio type
- empty audio
- microphone permission denied
- browser unsupported
- transcription model unavailable
- transcription failure
- empty transcript

Log internal details with request IDs. Do not expose provider bodies, credentials, stack traces, or database errors.

---

## 20. Tests

Use mocks. Do not call live Groq or Supabase in tests.

### Diagnosis service tests

- valid diagnosis flow
- unsupported image type
- oversized image
- invalid image bytes
- missing symptoms
- unsupported district
- storage failure
- database failure after storage
- provider failure
- invalid structured response
- confidence clamping
- urgency validation
- completed persistence
- failed persistence
- ownership checks
- deletion cleanup
- continue-in-chat summary

### Diagnosis handler tests

- multipart parsing
- request limits
- anonymous ownership
- protected image access
- unauthorized access
- pagination
- safe error responses

### Transcription service tests

- valid transcription
- unsupported audio type
- oversized audio
- empty audio
- provider failure
- empty transcript
- Krio requires confirmation
- temporary cleanup
- no permanent audio storage

### Transcription handler tests

- valid multipart request
- missing audio
- invalid language hint
- safe errors
- body-size limits

Also run all existing Phase 1–3 tests. Do not remove regression tests.

---

## 21. Docker and build updates

Update the Dockerfile and Compose setup safely:

- Copy new templates, JS, prompts, and migrations.
- Compile Tailwind after adding new templates.
- Create writable local upload/temp directories.
- Keep the runtime user non-root.
- Use a Docker volume for local uploads when needed, for example:

```text
agriconnect_uploads:/app/data/uploads
```

- Do not expose the upload directory as a static public folder.
- Do not bake secrets into the image.
- Preserve the existing health check.

Update the Makefile only where necessary. Do not create duplicate commands.

---

## 22. Documentation

Update `README.md` with:

- Phase 4 architecture and workflow
- Phase 5 architecture and workflow
- Vision model configuration
- Transcription model configuration
- Local storage configuration
- Supabase Storage configuration
- Upload limits
- Diagnosis API routes
- Transcription API route
- Browser microphone requirements
- Krio transcription limitations
- Security behavior
- New tests
- Docker volume behavior

Known limitations must include:

- AI image diagnosis may be incorrect
- image quality affects results
- several conditions can cause similar symptoms
- expert confirmation is required for serious cases
- Krio voice transcription is experimental
- no native Krio text-to-speech
- audio is not retained by default
- full authentication is not yet implemented
- extension-officer review is not yet implemented

Update `PHASE_4_5_REPORT.md` with every major decision, file change, test, and result.

---

## 23. Acceptance criteria

Do not finish until these work.

### Phase 4

1. `/diagnose` renders.
2. Image preview works.
3. Backend image validation works.
4. Storage interface saves the image.
5. Diagnosis record is created.
6. Agricultural context is retrieved.
7. Groq vision receives image and field information.
8. Structured JSON is validated.
9. Result is saved and displayed safely.
10. History works.
11. Ownership works.
12. Delete works.
13. Protected image access works.
14. Continue-in-chat works.
15. Failures are recorded safely.
16. No credential appears in frontend code.

### Phase 5

1. Microphone button appears.
2. Permission is requested only after user action.
3. Recording starts, stops, and cancels.
4. Timer works.
5. Audio playback works.
6. Audio upload is validated.
7. Groq transcription is called from the backend.
8. Transcript appears in an editable field.
9. Transcript is not sent automatically.
10. Corrected text can be sent through normal chat.
11. Krio experimental warning appears.
12. Temporary audio is deleted.
13. Unsupported browsers and permission denial are handled.
14. No credential appears in frontend code.

### Regression

1. Existing English chat still works.
2. Existing Krio text chat still works.
3. Knowledge retrieval still works.
4. Weather still works.
5. Conversation persistence still works.
6. Anonymous ownership still works.
7. Docker still works.
8. Existing tests still pass.

---

## 24. Required verification

Run:

```bash
go mod tidy
go test ./...
go vet ./...
go build ./cmd/server
npm install
npx tailwindcss -i ./web/static/css/input.css -o ./web/static/css/app.css --minify
docker compose config
docker compose build
```

When possible, also run:

```bash
docker compose up -d
```

Verify:

```text
GET /health
GET /assistant
GET /diagnose
GET /diagnoses
```

Do not claim a command succeeded unless it was actually executed.

---

## 25. Final response

After implementation, report:

1. Initial gaps discovered
2. Existing files modified
3. New files created
4. Migration added
5. Storage architecture implemented
6. Diagnosis workflow implemented
7. Vision integration implemented
8. Voice recorder implemented
9. Transcription integration implemented
10. Routes added
11. UI changes
12. Tests added
13. Commands executed
14. Test results
15. Build results
16. Docker results
17. Remaining limitations
18. Exact commands I should run locally

Also ensure `PHASE_4_5_REPORT.md` contains the same grounded information.

Do not stop after auditing or describing the work.

Inspect, implement, connect, test, and document Phase 4 and Phase 5.
