# Fix AgriConnect Weather, Voice Transcription and Crop Diagnosis Navigation

The deployed AgriConnect application is live at:

```text
https://agriconnect-b0hh.onrender.com
```

Registration and authentication now work.

The current production logs show:

```text
POST /api/v1/auth/register            → 201
GET  /assistant                       → 200
GET  /api/v1/weather?district=Kono    → 429
```

There is no logged request for:

```text
POST /api/v1/ai/transcribe
```

The user also cannot find a visible crop-image upload button or navigation link.

Inspect the entire implementation before modifying it.

Do not replace the architecture and do not introduce React.

Use the existing:

* Go
* Gin
* Go HTML templates
* Tailwind CSS
* Vanilla JavaScript
* PostgreSQL
* Groq
* Open-Meteo
* Supabase Storage

Implement and test all fixes below.

---

## 1. Fix the weather endpoint returning HTTP 429

The current request is:

```text
GET /api/v1/weather?district=Kono
```

and it returns:

```text
429 Too Many Requests
```

Inspect:

```text
cmd/server/main.go
internal/middleware/
internal/handlers/weather*
internal/services/weather*
internal/weather/
web/static/js/assistant.js
web/templates/pages/assistant.html
```

Find exactly which rate limiter blocks the weather request.

### Rate-limiting requirements

1. Do not apply the same small global limit to every application route.
2. Exempt these from normal application rate limits:

```text
/health
/static/*
/favicon.ico
```

3. Do not let Render health checks consume a farmer's API allowance.
4. For authenticated users, use the authenticated user ID as the primary rate-limit key.
5. For unauthenticated users, use a correctly resolved client IP.
6. Do not treat every visitor as `::1`.
7. Configure trusted proxy handling correctly for Render.
8. Do not blindly trust arbitrary forwarded headers from untrusted clients.
9. Give weather a separate sensible limit, for example:

   * 30 requests per minute per authenticated user
10. Use the PostgreSQL weather cache before calling Open-Meteo.
11. Add a `Retry-After` header to genuine 429 responses.
12. Return a structured JSON error.
13. Do not return 429 during a normal single page load.
14. Ensure only one automatic weather request is made when the assistant page loads.
15. Prevent duplicate weather requests caused by several `DOMContentLoaded` listeners or duplicate script initialization.

Add tests proving:

* `/health` does not consume the weather allowance.
* Static files do not consume the weather allowance.
* A normal authenticated weather request returns 200.
* Several immediate duplicate requests are handled sensibly.
* Different authenticated users do not share the same limit.
* Render proxy addresses do not cause every user to share one bucket.

---

## 2. Always show weather using the district selected during registration

The farmer selected `Kono` during registration.

The assistant page should automatically show the weather for that district.

Required flow:

```text
Registration saves district
→ authenticated user opens /assistant
→ backend identifies authenticated user
→ assistant page receives user's saved district
→ frontend requests weather once
→ weather card is displayed
```

Requirements:

1. Use the authenticated user's district from PostgreSQL as the source of truth.
2. Do not rely only on `localStorage`.
3. Add the user's district to the assistant page data.
4. Alternatively, load it safely through `/api/v1/auth/me`.
5. Automatically load weather once after the assistant page initializes.
6. Display:

   * district
   * current temperature
   * humidity
   * wind
   * precipitation or rain probability
   * short forecast
   * last updated time
7. Show a loading skeleton while fetching.
8. Show a retry button on failure.
9. Do not display only the vague message `Unable to load weather`.
10. Display a useful safe message based on the status:

    * 429: `Too many weather requests. Please wait briefly and retry.`
    * 502/503: `The weather provider is temporarily unavailable.`
    * network error: `Check your internet connection and retry.`
11. If the farmer has no district, show a district selector.
12. Changing the selected district should refresh the card once.
13. Avoid duplicate fetches.
14. Keep cached weather visible if refreshing the provider fails.

Add browser and handler tests.

---

## 3. Fix the Transcribe button

The farmer can record audio, but clicking **Transcribe** does nothing.

The production logs show no:

```text
POST /api/v1/ai/transcribe
```

Therefore, inspect the complete frontend-to-backend flow.

Inspect:

```text
web/static/js/recorder.js
web/templates/partials/voice_recorder.html
web/templates/pages/assistant.html
internal/handlers/transcription*
internal/transcription/
internal/ai/transcription.go
cmd/server/main.go
```

### Frontend requirements

1. Confirm the recorder partial is rendered on `/assistant`.
2. Confirm every HTML ID used by `recorder.js` exists.
3. Confirm every element ID is unique.
4. Initialize the recorder only after the DOM is ready.
5. Avoid initializing it more than once.
6. Attach the click listener to the actual Transcribe button.
7. Make the button:

```html
<button type="button">
```

8. Preserve the recorded Blob after recording stops.
9. Disable Transcribe until a valid recording exists.
10. Build a `FormData` request.
11. Use the exact backend field name, such as:

```javascript
formData.append("audio", recordedBlob, filename);
```

12. Send:

```javascript
fetch("/api/v1/ai/transcribe", {
  method: "POST",
  credentials: "same-origin",
  body: formData
});
```

13. Send the selected language hint.
14. Show:

    * `Preparing audio…`
    * `Transcribing…`
    * success
    * error
15. Add a visible loading spinner.
16. Prevent duplicate clicks.
17. Re-enable the button after failure.
18. Display the returned transcript in an editable field.
19. Do not automatically send the transcript to the chat.
20. Keep the Krio experimental warning.
21. Allow retry and record again.
22. Log safe frontend errors to the browser console.
23. Never log raw audio or authentication cookies.

### MIME-type handling

Browsers may produce values such as:

```text
audio/webm
audio/webm;codecs=opus
audio/ogg;codecs=opus
audio/mp4
```

Normalize the MIME type safely.

Do not reject a valid WebM recording only because the browser adds a codec parameter.

The backend must still verify the file and enforce the size limit.

### Backend requirements

1. Confirm the route is registered:

```text
POST /api/v1/ai/transcribe
```

2. Confirm authentication middleware allows the authenticated farmer.
3. Confirm multipart field names match the frontend.
4. Return structured JSON.
5. Return clear statuses:

   * 400 for invalid input
   * 401 for authentication failure
   * 413 for oversized audio
   * 415 for unsupported audio
   * 429 for transcription rate limit
   * 502/503 for provider failure
6. Do not retain audio after processing.
7. Do not expose Groq error responses or API keys.
8. Add a request timeout.
9. Ensure `GROQ_API_KEY` and `GROQ_TRANSCRIPTION_MODEL` are loaded in production.

Add HTTP tests for:

* successful transcription
* missing audio
* valid WebM with codec parameter
* unsupported type
* oversized audio
* provider failure
* authenticated request
* unauthenticated request
* Krio confirmation requirement

---

## 4. Make crop-image diagnosis visible

The user cannot find a button or place to upload a crop image.

Confirm these routes exist and work:

```text
GET  /diagnose
GET  /diagnoses
GET  /diagnoses/:id

POST /api/v1/diagnoses
```

Open the route directly during testing:

```text
/diagnose
```

### Navigation requirements

Add a clearly visible item called:

```text
Diagnose Crop
```

to:

1. Desktop navigation
2. Desktop sidebar
3. Mobile vertical hamburger menu
4. Farmer dashboard quick actions
5. Assistant page quick actions

Use a Lucide icon such as:

```text
camera
image
scan
leaf
```

Also add a visible button near the assistant input area:

```text
Upload crop photo
```

This button should navigate to:

```text
/diagnose
```

Do not force the full diagnosis form into the chat input unless the existing architecture was designed for that.

### Diagnosis form requirements

The `/diagnose` page must visibly contain:

* Image drag-and-drop area
* Browse image button
* Mobile camera-compatible file input
* Image preview
* Replace image
* Remove image
* Crop selector
* District
* Language
* Plant part
* Symptom description
* Submit button
* Upload progress
* Analysis state
* Validation messages

Use:

```html
<input
  type="file"
  accept="image/jpeg,image/png,image/webp"
  capture="environment"
>
```

`capture="environment"` may help mobile users open the rear camera, but normal file browsing must still work.

### Authorization requirements

* Farmers can submit their own diagnosis.
* Farmers can only see their own diagnoses.
* Officers and admins may view diagnoses only according to existing authorization rules.
* Do not expose Supabase credentials.
* Keep the bucket private.
* Use ownership-checked signed URLs.

Add tests that confirm:

* `/diagnose` returns 200 for the farmer.
* The navigation contains `/diagnose`.
* The mobile menu contains `/diagnose`.
* The upload input exists.
* Valid images can be submitted.
* Unauthorized diagnosis access is rejected.

---

## 5. Fix navigation visibility

Inspect all navigation templates and role conditions.

Ensure an authenticated farmer can see:

```text
AI Assistant
Diagnose Crop
Diagnosis History
Weather
Notifications
Profile
Logout
```

Requirements:

1. Desktop navigation should remain horizontal.
2. Mobile menu should be vertical and collapsible.
3. The hamburger button should open and close the menu.
4. Active route highlighting should work.
5. Restricted officer/admin links must remain role-protected.
6. Do not hide farmer links because of an incorrect authentication-data key.
7. Do not rely on JavaScript alone for critical navigation.
8. Use real links such as:

```html
<a href="/diagnose">Diagnose Crop</a>
```

---

## 6. Add frontend diagnostics

During development, add safe diagnostics for failed requests.

For weather:

```text
status code
safe response message
request path
```

For transcription:

```text
button click detected
recording Blob exists
Blob type
Blob size
request started
response status
```

Do not log:

* audio content
* JWTs
* cookies
* Groq keys
* Supabase keys
* farmer-sensitive data

Remove excessive debug logging before production, but retain useful structured errors.

---

## 7. Verify Render environment variables

Inspect and document the exact variable names used by the code.

Confirm Render contains the required values for:

```env
APP_ENV=production
APP_URL=https://agriconnect-b0hh.onrender.com
GIN_MODE=release

GROQ_API_KEY=
GROQ_TRANSCRIPTION_MODEL=
GROQ_VISION_MODEL=

WEATHER_REQUEST_TIMEOUT_SECONDS=
WEATHER_CACHE_DURATION_MINUTES=

MAX_AUDIO_SIZE_MB=
ALLOWED_AUDIO_TYPES=

MAX_IMAGE_SIZE_MB=
ALLOWED_IMAGE_TYPES=

STORAGE_DRIVER=supabase
SUPABASE_URL=
SUPABASE_SECRET_KEY=
SUPABASE_STORAGE_BUCKET=crop-diagnosis-images
```

Use only the names actually loaded by the application.

Do not add duplicate environment-variable names.

Do not place real secrets in `.env.example`.

---

## 8. Required regression testing

Run:

```bash
go mod tidy
go test ./...
go vet ./...
go build ./cmd/server

npm install
npx tailwindcss \
  -i ./web/static/css/input.css \
  -o ./web/static/css/app.css \
  --minify

docker build --no-cache -t agriconnect-feature-fix .
```

When Docker is available, run the production image and manually test:

### Farmer registration

```text
Register with district Kono
→ Redirect to assistant
→ Kono weather loads automatically
```

### Weather

```text
Open assistant
→ one weather request
→ response 200
→ forecast card displayed
```

### Voice

```text
Record
→ stop
→ play recording
→ click Transcribe
→ POST /api/v1/ai/transcribe appears in logs
→ editable transcript displayed
```

### Diagnosis

```text
Open navigation
→ click Diagnose Crop
→ /diagnose loads
→ upload JPEG/PNG/WebP
→ preview appears
→ diagnosis request is submitted
```

### Mobile

```text
Open hamburger menu
→ menu is vertical
→ Diagnose Crop link visible
→ weather visible
→ voice recorder usable
```

---

## 9. Final report

Create:

```text
FEATURE_INTEGRATION_FIX_REPORT.md
```

Include:

1. Weather root cause
2. Rate-limiter root cause
3. Rate-limiter changes
4. Automatic district-weather changes
5. Voice-transcription root cause
6. Recorder frontend changes
7. Transcription backend changes
8. Diagnosis-navigation root cause
9. Navigation changes
10. Diagnosis-page changes
11. Tests added
12. Commands executed
13. Test results
14. Docker results
15. Render environment variables required
16. Remaining limitations

Do not only inspect or explain the failures.

Implement, test and report the fixes.

Do not rewrite unrelated AI or USSD logic.



# Additional UI Fixes — Diagnosis Navigation, Chat Upload Button and Favicon

The crop diagnosis page works when opened directly:

```text
https://agriconnect-b0hh.onrender.com/diagnose
```

Therefore, do not rebuild the crop-diagnosis backend or change the AI diagnosis logic.

The remaining problem is that users cannot easily discover or access the diagnosis feature from the interface.

Also, the browser tab currently has no AgriConnect favicon.

Inspect the existing navigation templates, assistant template, static assets and shared layout before editing.

---

## 1. Add “Diagnose Crop” to every farmer navigation area

Add a clearly visible navigation item:

```text
Diagnose Crop
```

that links to:

```text
/diagnose
```

It must appear in all appropriate farmer navigation areas:

1. Desktop header navigation
2. Desktop sidebar
3. Mobile hamburger menu
4. Farmer dashboard quick actions
5. Assistant page quick actions

Use a suitable Lucide icon such as:

```text
camera
scan-line
image-plus
leaf
```

Recommended label:

```text
Diagnose Crop
```

Recommended accessibility label:

```text
Upload a crop image for diagnosis
```

Do not use `href="#"`.

Use a real link:

```html
<a href="/diagnose">
  Diagnose Crop
</a>
```

### Role visibility

The link should be visible to authenticated farmers.

It may also be visible to officers and administrators when their existing permissions allow diagnosis access.

Do not expose officer-only or administrator-only links to farmers.

Do not hide the diagnosis link because of an incorrect template authentication key.

---

## 2. Add an image-upload shortcut in the assistant chat area

Add a visible crop-image button near the chat input and microphone button.

Suggested layout:

```text
[Upload crop photo] [Microphone] [Message input] [Send]
```

On smaller screens, keep the controls usable without making the chat input too narrow.

The button should:

* Display a camera or image-plus icon
* Include accessible text or an accessible label
* Navigate to `/diagnose`
* Not immediately open or submit a hidden diagnosis form
* Not send an image directly as a normal text-chat message
* Not interfere with the microphone button
* Not interfere with the Send button
* Use `type="button"` when implemented as a button
* Use a normal anchor when no JavaScript is needed

Preferred implementation:

```html
<a
  href="/diagnose"
  aria-label="Upload a crop image for diagnosis"
  title="Diagnose crop from photo"
>
  <i data-lucide="image-plus"></i>
  <span>Upload crop photo</span>
</a>
```

On narrow mobile screens, the visible label may be shortened while retaining the full `aria-label` and tooltip.

Example mobile display:

```text
[Photo icon] [Mic icon] [Message input] [Send]
```

The photo icon must still be easy to understand.

---

## 3. Add a diagnosis quick-action card

On the assistant page or farmer dashboard, add a visible quick-action card:

```text
Diagnose a Crop Problem
Upload a clear crop photo and describe the symptoms.
```

The card should link to:

```text
/diagnose
```

Include:

* Crop or leaf icon
* Clear heading
* Short explanation
* Call-to-action such as `Start diagnosis`
* Responsive styling
* Keyboard focus state
* Hover state

Do not duplicate the full diagnosis form on the assistant page.

---

## 4. Ensure the mobile menu contains the diagnosis link

The mobile hamburger menu must contain a vertical item:

```text
Diagnose Crop
```

Expected farmer mobile menu:

```text
AI Assistant
Diagnose Crop
Diagnosis History
Weather
Notifications
Profile
Logout
```

Requirements:

* Vertically arranged
* Full-width links
* Active-route styling
* Diagnosis link closes the menu when clicked
* Keyboard accessible
* Visible focus ring
* Does not overflow small screens

The `/diagnose` item should show an active state when the current path is `/diagnose`.

---

## 5. Add diagnosis history navigation

Also ensure users can access:

```text
/diagnoses
```

Use a label such as:

```text
Diagnosis History
```

This should appear near `Diagnose Crop` in:

* Desktop navigation
* Sidebar
* Mobile menu
* Farmer dashboard

Use an appropriate Lucide icon such as:

```text
history
clipboard-list
file-clock
```

---

## 6. Verify the diagnosis page upload interface

Because `/diagnose` already opens, inspect the page and confirm it visibly contains:

* Drag-and-drop image area
* Browse-image button
* Mobile camera-compatible input
* Image preview
* Replace-image action
* Remove-image action
* Crop selector
* District
* Language
* Plant-part selector
* Symptom description
* Submit button
* Upload progress
* Validation messages

The file input should support:

```html
<input
  type="file"
  accept="image/jpeg,image/png,image/webp"
  capture="environment"
>
```

Do not make `capture` mandatory. Users must still be able to select existing images from their device.

---

## 7. Add an AgriConnect favicon

The application currently requests:

```text
/favicon.ico
```

and receives HTTP 404.

Add a proper AgriConnect favicon and connect it to every HTML page through the shared base layout.

First inspect the repository for an existing:

* AgriConnect logo
* Leaf icon
* Brand mark
* SVG logo
* PNG logo

Reuse the existing brand asset when suitable.

If no suitable icon exists, create a simple original AgriConnect favicon using:

* Green background or green agricultural accent
* White or light leaf/sprout symbol
* Simple shapes that remain recognizable at small sizes
* No complicated text
* No copied third-party brand logo

Create at least:

```text
web/static/favicon.svg
web/static/favicon.ico
```

Optionally also create:

```text
web/static/apple-touch-icon.png
```

Do not use a Lucide icon through JavaScript as the browser favicon. The favicon must be a real static asset.

---

## 8. Add favicon tags to the shared HTML layout

Add the favicon references inside the shared `<head>` used by all pages:

```html
<link
  rel="icon"
  type="image/svg+xml"
  href="/static/favicon.svg?v={{ .AssetVersion }}"
>

<link
  rel="alternate icon"
  href="/static/favicon.ico?v={{ .AssetVersion }}"
>

<link
  rel="apple-touch-icon"
  href="/static/apple-touch-icon.png?v={{ .AssetVersion }}"
>
```

Use only the assets that actually exist.

Also add:

```html
<meta name="theme-color" content="#166534">
```

Use the existing AgriConnect brand colour when it differs.

Ensure the favicon appears on:

* Landing page
* Login
* Registration
* Assistant
* Diagnosis form
* Diagnosis history
* Officer pages
* Administrator pages

Do not repeat favicon tags separately in every page. Put them in the shared layout or shared head partial.

---

## 9. Fix favicon static-file serving

Confirm Gin serves the favicon assets correctly.

The following must return HTTP 200:

```text
/static/favicon.svg
/static/favicon.ico
```

Optionally support the conventional root route:

```text
/favicon.ico
```

It may redirect or serve the same static file.

Example safe route:

```go
router.StaticFile(
    "/favicon.ico",
    "./web/static/favicon.ico",
)
```

Do not create duplicate conflicting static routes.

Ensure the Dockerfile copies all favicon assets into the final runtime image.

---

## 10. Cache-busting

The production site already uses asset version query parameters.

Apply the same versioning to favicon references so browsers do not retain the old missing favicon result.

Example:

```html
href="/static/favicon.svg?v={{ .AssetVersion }}"
```

Confirm that `.AssetVersion` is available in the shared layout.

Do not hardcode a random version manually on every deployment when a build-version mechanism already exists.

---

## 11. Visual styling requirements

The diagnosis shortcuts must match the current AgriConnect design:

* Green agricultural colour palette
* Rounded controls
* Clear hover states
* Visible keyboard focus states
* Comfortable spacing
* Mobile-friendly tap targets
* Lucide icons aligned with text
* No horizontal overflow
* Consistent active navigation style

The chat upload-photo shortcut should look important but must not overpower the normal Send button.

---

## 12. Tests

Add or update tests verifying:

1. `/diagnose` returns HTTP 200.
2. `/diagnoses` returns the expected page.
3. Farmer desktop navigation contains `/diagnose`.
4. Farmer desktop navigation contains `/diagnoses`.
5. Mobile menu contains `/diagnose`.
6. Mobile menu contains `/diagnoses`.
7. Assistant page contains an upload-photo shortcut.
8. The upload-photo shortcut links to `/diagnose`.
9. The diagnosis form contains a file input.
10. The file input accepts JPEG, PNG and WebP.
11. The shared layout contains favicon links.
12. `/static/favicon.svg` returns HTTP 200.
13. `/static/favicon.ico` returns HTTP 200.
14. `/favicon.ico` no longer returns HTTP 404.
15. Favicon files exist in the production Docker image.
16. Farmer navigation does not expose unauthorized officer/admin links.

Do not make live Groq, Supabase or Open-Meteo requests in these tests.

---

## 13. Manual browser verification

After implementation, verify on desktop and mobile:

### Desktop

```text
Open assistant
→ Diagnose Crop visible in navigation
→ Upload crop photo visible beside chat controls
→ Click it
→ /diagnose opens
```

### Mobile

```text
Open hamburger menu
→ Diagnose Crop visible
→ menu items vertically arranged
→ click Diagnose Crop
→ menu closes
→ /diagnose opens
```

### Favicon

```text
Open landing page
→ favicon visible in browser tab
Open assistant
→ same favicon visible
Open diagnosis page
→ same favicon visible
```

Test with a hard refresh or private/incognito window because browsers cache favicons aggressively.

---

## 14. Required commands

Run:

```bash
go test ./...
go vet ./...
go build ./cmd/server

npm install
npx tailwindcss \
  -i ./web/static/css/input.css \
  -o ./web/static/css/app.css \
  --minify

docker build --no-cache -t agriconnect-ui-fix .
```

Start the production image when Docker is available and confirm that navigation and favicon assets work.

---

## 15. Final report

Update:

```text
FEATURE_INTEGRATION_FIX_REPORT.md
```

Include:

1. Why the diagnosis feature was not discoverable
2. Navigation templates modified
3. Assistant upload-photo shortcut added
4. Mobile-menu changes
5. Diagnosis-history link changes
6. Favicon assets created
7. Shared layout changes
8. Static route changes
9. Docker asset-copy verification
10. Tests added
11. Commands executed
12. Test results
13. Manual browser results

Do not rebuild the crop-diagnosis backend.

The `/diagnose` route already works. Fix discoverability, navigation, assistant shortcuts and favicon integration.