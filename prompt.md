# AgriConnect MVP Platform Completion and Full-System Testing Prompt

You are working inside the existing AgriConnect repository.

Your task is to complete the remaining **non-AI platform features** needed for the AgriConnect MVP presentation and then run full-system testing.

Do not create a new project.

Do not replace the existing architecture.

Do not introduce React, Next.js, FastAPI, Firebase, or a second backend.

Use the existing stack:

- Go
- Gin
- Go HTML templates
- Tailwind CSS
- Vanilla JavaScript
- PostgreSQL
- GORM
- Supabase Storage
- Render deployment
- Groq AI integration
- Existing authentication
- Existing crop diagnosis
- Existing weather integration
- Existing voice transcription
- Existing English/Krio preference system

---

# 1. Important ownership boundary

Joshua owns:

- AI chat
- AI prompts
- Groq model selection
- Vision diagnosis
- Voice transcription
- Krio transcription
- Weather intelligence
- USSD
- USSD-to-AI logic

Do not rewrite Joshua’s AI or USSD work.

Do not create a second AI assistant.

Do not create duplicate diagnosis or transcription systems.

You may connect platform pages to the existing AI routes.

You may repair integration only if it blocks the MVP flow.

---

# 2. Read current reports before editing

Before making changes, inspect these reports if they exist:

```text
README.md
RECOVERY_REPORT.md
PHASE_4_5_REPORT.md
FEATURE_INTEGRATION_FIX_REPORT.md
AI_USER_DIAGNOSIS_STORAGE_FIX_REPORT.md
VISION_MODEL_FIX_REPORT.md
DIAGNOSIS_DISPLAY_IMAGE_STT_FIX_REPORT.md
SCHEMA_AND_IMAGE_PROXY_FIX_REPORT.md
CURRENT_PRODUCTION_BUGFIX_REPORT.md
```

Record which files exist.

The latest diagnosis/STT reports indicate that these features already exist or were recently fixed:

- Farmer authentication
- Profile preferences
- English/Krio language persistence
- AI chat
- Weather
- Voice transcription
- Optional Hugging Face Krio STT provider
- Crop diagnosis upload
- Supabase image storage
- Supabase image proxy
- Strict JSON vision result parsing
- Diagnosis result page
- Diagnosis charts
- Diagnosis history
- Diagnosis history back button
- Favicon
- Mobile navigation

Do not redo these unless inspection proves they are broken.

---

# 3. Pre-flight production bug check

Before implementing new MVP features, quickly verify the current branch still passes the core bugfix checks.

Run or inspect tests for:

```text
Diagnosis image route
Diagnosis detail page
Diagnosis history page
Voice transcription
AI chat
Weather
Profile preferences
```

Also check that the diagnosis detail confidence chart no longer performs unsafe Go template comparisons like:

```gotemplate
{{ ge $d.Confidence 70 }}
```

If this bug still exists, fix it before MVP platform work.

The diagnosis confidence styling should be prepared in the Go view model, not compared directly in the template.

---

# 4. MVP presentation target

The full AgriConnect MVP must demonstrate this flow:

```text
Farmer registers
→ Farmer logs in
→ Farmer sees personalized dashboard
→ Farmer asks AI a farming question
→ Farmer sees weather for saved district
→ Farmer uploads crop image
→ AI creates preliminary diagnosis
→ Extension officer reviews diagnosis
→ Farmer receives notification
→ Farmer reads officer recommendation
→ Farmer checks market prices
→ Farmer opens farming learning resources
→ Administrator manages users and roles
```

Joshua will separately demonstrate USSD.

Your job is to complete the platform surrounding the AI.

---

# 5. Existing AI routes must be reused

Inspect the actual routes in the codebase.

Likely existing routes include:

```text
GET  /assistant

POST /api/v1/conversations
GET  /api/v1/conversations
GET  /api/v1/conversations/:id
POST /api/v1/conversations/:id/messages
POST /api/v1/conversations/:id/messages/stream

GET  /api/v1/weather

GET  /diagnose
GET  /diagnoses
GET  /diagnoses/:id
POST /api/v1/diagnoses
GET  /api/v1/diagnoses
GET  /api/v1/diagnoses/:id
GET  /api/v1/diagnoses/:id/image
POST /api/v1/diagnoses/:id/retry
POST /api/v1/diagnoses/:id/continue-in-chat

POST /api/v1/ai/transcribe
```

Confirm the real routes.

Do not create duplicate routes like:

```text
/api/chat2
/api/diagnose2
/new-assistant
/another-ai
```

---

# 6. Authentication and roles

Authentication already exists. Verify it and extend it only where needed.

Required roles:

```text
farmer
officer
admin
```

Confirm these routes work:

```text
GET  /register
GET  /login
GET  /profile

POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/logout
POST /api/v1/auth/refresh
GET  /api/v1/auth/me
PATCH /api/v1/profile/preferences
```

Required behavior:

- Registration creates a farmer by default.
- Public registration must not allow choosing `admin` or `officer`.
- Passwords are hashed.
- Tokens use HTTP-only cookies.
- Refresh works.
- Logout clears cookies.
- Inactive users cannot log in.
- Farmer identity persists after refresh.
- Farmer district and preferred language persist.
- Farmers cannot access officer/admin routes.
- Officers cannot access admin routes.
- Admins can access admin routes.
- Backend authorization is final, not just hidden UI links.

Do not rebuild authentication from scratch.

---

# 7. Build or complete farmer dashboard

Create or complete:

```text
/dashboard
```

After login, a farmer may land on `/dashboard` or `/assistant`, depending on existing app flow. If the existing flow already sends users to `/assistant`, add a clear dashboard navigation link.

The farmer dashboard must show:

## Farmer identity

- Full name
- Phone number or masked phone number
- District
- Preferred language
- Role
- Profile link
- Logout action

## Quick actions

```text
Ask AI              → /assistant
Diagnose Crop       → /diagnose
Diagnosis History   → /diagnoses
Weather             → /assistant or dashboard weather card
Market Prices       → /market-prices
Learning Resources  → /resources
Notifications       → /notifications
Profile             → /profile
```

## Recent activity

Show:

- Recent AI conversations
- Recent diagnoses
- Latest diagnosis status
- Latest officer-review status
- Unread notification count
- Current weather summary for saved district

Do not call Groq directly from the dashboard.

Use existing APIs and routes.

---

# 8. Build extension-officer workflow

Create or complete:

```text
/officer
/officer/diagnoses
/officer/diagnoses/:id
```

The officer workflow must allow an extension officer to review AI diagnosis cases.

Officer dashboard should show:

- Pending reviews
- Cases under review
- Completed reviews
- Urgent cases
- Cases requiring field visit
- Diagnoses from officer’s assigned district

Filters:

- District
- Crop
- Date
- Urgency
- Review status

Officer actions:

- Open diagnosis
- View uploaded crop image
- View farmer symptoms
- View AI assessment
- Claim case
- Confirm AI suspected condition
- Correct AI suspected condition
- Add officer comment
- Add recommendation
- Request more information
- Recommend field visit
- Complete review

Review statuses:

```text
pending
in_review
confirmed
needs_more_information
field_visit_required
closed
```

Important rule:

```text
Do not overwrite the AI assessment.
Store officer review separately.
```

---

# 9. Officer-review database

Inspect whether `diagnosis_reviews` already exists.

If missing, add a migration.

Suggested fields:

```text
id
diagnosis_id
officer_id
review_status
confirmed_condition
officer_comment
recommendation
urgency
requires_field_visit
created_at
updated_at
```

Add indexes:

```text
diagnosis_id
officer_id
review_status
created_at
```

Prevent uncontrolled duplicate active reviews for the same diagnosis.

Use repository and service layers.

Do not put database queries directly inside handlers or templates.

---

# 10. Officer API routes

Implement or complete:

```text
GET  /api/v1/officer/diagnoses
GET  /api/v1/officer/diagnoses/:id
POST /api/v1/officer/diagnoses/:id/claim
POST /api/v1/officer/diagnoses/:id/reviews
PUT  /api/v1/officer/diagnoses/:id/reviews/:reviewID
```

Requirements:

- Officer or admin role required.
- Officers see only authorized district cases.
- Admins can see all cases.
- Farmers cannot use officer endpoints.
- Validate review statuses.
- Return structured JSON errors.
- Create audit records for officer actions.

---

# 11. Show officer review to farmer

Update diagnosis detail page.

Show two separate sections:

## AI assessment

- Probable condition
- Confidence
- Observed signs
- Possible alternatives
- Recommended actions
- Prevention tips
- Urgency
- AI disclaimer

## Extension-officer review

- Review status
- Confirmed condition
- Officer comment
- Officer recommendation
- Field visit requirement
- Review date

Make it visually clear:

```text
AI screening result
Human extension-officer review
```

Do not overwrite or hide the original AI result.

---

# 12. Build notifications

Create or complete:

```text
/notifications
GET   /api/v1/notifications
PATCH /api/v1/notifications/:id/read
PATCH /api/v1/notifications/read-all
```

Create notifications when:

- Officer claims a diagnosis
- Officer requests more information
- Officer completes a review
- Officer recommends a field visit
- Admin changes account status when appropriate

Suggested fields:

```text
id
user_id
title
message
notification_type
is_read
entity_type
entity_id
created_at
```

Requirements:

- Farmers see only their own notifications.
- Notification count appears in dashboard/sidebar/header.
- Notification links open the related diagnosis.
- Mark-as-read works.
- Read-all works.
- No user can read another user’s notifications.

---

# 13. Build admin MVP

Create or complete:

```text
/admin
/admin/users
/admin/diagnoses
/admin/reviews
/admin/audit-logs
```

Admin dashboard should show:

- Total farmers
- Total officers
- Active users
- Inactive users
- Total diagnoses
- Pending reviews
- Completed reviews
- Recent audit activity

Admin user management:

```text
GET   /api/v1/admin/users
PATCH /api/v1/admin/users/:id/role
PATCH /api/v1/admin/users/:id/status
```

Security requirements:

- Admin only.
- Users cannot promote themselves.
- Farmers cannot open admin pages.
- Officers cannot perform admin actions.
- Prevent disabling or demoting the final active admin.
- Record role/status changes in audit logs.
- Public registration must never expose admin/officer role selection.

---

# 14. Add audit logs

Create or complete audit logs.

Suggested fields:

```text
id
actor_user_id
action
entity_type
entity_id
metadata
created_at
```

Record:

- Role changes
- User activation/deactivation
- Officer claim
- Officer review create/update
- Diagnosis deletion
- Storage deletion failure
- Important authentication failures

Never store:

- Plain passwords
- Password hashes
- JWTs
- Refresh tokens
- Groq keys
- Supabase keys
- Hugging Face keys
- Raw audio
- Uploaded image contents

---

# 15. Build market-prices MVP

Create:

```text
/market-prices
```

MVP commodities:

```text
rice
cassava
groundnut
palm oil
cocoa
coffee
```

Suggested fields:

```text
id
commodity
market_name
district
price
currency
unit
source
price_date
is_verified
created_by
created_at
updated_at
```

Use:

```text
Currency: SLE
```

Example units:

```text
per kg
per bag
per bushel
per litre
```

Farmer features:

- View latest prices
- Filter by commodity
- Filter by district
- See market name
- See date last updated
- See verified/unverified status

Officer/admin features:

- Add price
- Update price
- Mark verified

Ordinary farmers cannot publish official prices.

Seed demo market data.

Clearly label it:

```text
Demonstration market data for MVP testing.
```

Do not present demo prices as official national live market data.

---

# 16. Build farming resources MVP

Create:

```text
/resources
/resources/:id
```

Resource categories:

```text
planting
pests
disease
soil
fertiliser
irrigation
harvesting
storage
```

Suggested fields:

```text
id
title
crop
category
language
summary
content
source
reviewed
published
created_by
created_at
updated_at
```

Farmer features:

- Browse resources
- Filter by crop
- Filter by category
- Filter by language
- Open resource detail
- See source and reviewed status

Admin features:

- Create resource
- Edit resource
- Publish/unpublish resource
- Mark reviewed

Important:

Inspect whether the existing `agricultural_documents` table can safely power both AI retrieval and farmer resources.

Prefer reusing it when practical.

Do not expose internal prompt-only content.

---

# 17. Navigation

Create one consistent role-aware navigation system.

## Farmer navigation

```text
Dashboard
AI Assistant
Diagnose Crop
Diagnosis History
Market Prices
Learning Resources
Notifications
Profile
Logout
```

## Officer navigation

```text
Officer Dashboard
Diagnosis Queue
Market Prices
Learning Resources
Notifications
Profile
Logout
```

## Admin navigation

```text
Admin Dashboard
Users
Diagnoses
Reviews
Market Prices
Learning Resources
Audit Logs
Profile
Logout
```

Requirements:

- Desktop navigation is clear.
- Mobile menu is vertical and collapsible.
- Active route is highlighted.
- Links render by role.
- Backend authorization remains final security.
- Restricted routes reject unauthorized users.

---

# 18. Presentation UX polish

The MVP should look polished enough for a university presentation.

Add:

- Empty states
- Loading states
- Error states
- Success messages
- Confirmation dialogs
- Responsive cards/tables
- Accessible buttons
- Visible focus states
- Mobile-friendly forms
- Clear page headings
- Breadcrumbs/back buttons
- Consistent AgriConnect green branding
- Lucide icons

Do not spend time on unnecessary animations before core flows work.

---

# 19. Demo seed data

Create a controlled demo seed command or seed file.

Seed:

- One admin
- One officer
- Two farmers
- Market prices
- Farming resources
- Optional sample diagnosis/review records only if safe

Do not hardcode production passwords.

Use environment variables or generated temporary credentials.

Do not run demo seed automatically in production.

Use explicit opt-in:

```env
RUN_DEMO_SEED=false
```

or an explicit command.

Document exactly how to create demo users.

---

# 20. Demo guide

Create:

```text
MVP_DEMO_GUIDE.md
```

Document the full demo sequence.

## Farmer demo

```text
Login
→ Dashboard
→ Ask AI
→ View weather
→ Diagnose crop
→ View diagnosis result
→ Check notifications
→ Read officer review
→ View market prices
→ Open learning resource
```

## Officer demo

```text
Login
→ Open diagnosis queue
→ Claim farmer case
→ Review AI result
→ Submit recommendation
→ Mark complete
```

## Farmer again

```text
Login
→ Notification appears
→ Open reviewed diagnosis
→ Read officer recommendation
```

## Admin demo

```text
Login
→ View admin dashboard
→ View users
→ Assign officer role
→ View diagnoses and review counts
→ View audit logs
```

Joshua will separately demonstrate USSD.

---

# 21. Full-system testing requirements

Add automated tests for the new platform work.

## Farmer dashboard tests

- Requires authentication
- Shows user identity
- Shows district
- Shows preferred language
- Shows quick links
- Shows recent diagnoses
- Shows review status
- Shows notification count

## Authorization tests

- Farmer blocked from officer routes
- Farmer blocked from admin routes
- Officer allowed on officer routes
- Officer blocked from admin routes
- Admin allowed on admin routes
- Inactive user blocked

## Officer workflow tests

- Officer sees assigned-district cases
- Officer cannot access unauthorized district
- Officer can claim case
- Officer can create review
- Officer can update review
- Farmer can view completed review
- Farmer cannot create officer review
- AI result remains unchanged

## Notification tests

- Review creates notification
- Farmer sees own notifications
- Farmer cannot see another user's notification
- Mark-as-read works
- Read-all works

## Admin tests

- Admin lists users
- Admin changes role
- Admin changes account status
- User cannot promote self
- Final active admin is protected
- Audit event is created

## Market price tests

- Farmer can list prices
- Filtering works
- Farmer cannot create price
- Officer/admin can create price
- Invalid values rejected

## Resource tests

- Farmer can list published resources
- Unpublished resources hidden
- Admin can create and publish resources
- Filtering works

## Regression tests

Verify these still work:

```text
Registration
Login
Profile preferences
AI chat
English response
Krio response
Weather card
Crop diagnosis upload
Diagnosis image display
Diagnosis history
Diagnosis charts
Voice transcription
Continue in AI Chat
Mobile navigation
Favicon
```

Do not call live Groq, Supabase, Hugging Face, Open-Meteo, or USSD providers in normal unit tests.

Use mocks.

---

# 22. Database and Render safety

The Render PostgreSQL instance is shared, but AgriConnect uses its own logical database:

```text
agriconnect
```

Do not access or modify:

```text
savwise_ai
market_pay
```

Requirements:

- Use only `DATABASE_URL`.
- Confirm migrations target `agriconnect`.
- Do not create another Render PostgreSQL service.
- Do not reset the database.
- Do not drop existing tables.
- Add forward-safe migrations.
- Preserve all existing users, conversations, diagnoses, images, weather cache, and AI data.

---

# 23. Supabase safety

Crop images are stored in Supabase Storage.

Do not create another image storage system.

Requirements:

- Keep bucket private.
- Keep service key server-side.
- Do not expose Supabase secret in JavaScript.
- Use existing image proxy or signed URL system.
- Officers/admins can view images only through authorized backend checks.
- Do not store signed URLs in PostgreSQL.
- Store only object paths.

---

# 24. Required commands

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

docker compose config
docker build --no-cache -t agriconnect-mvp-platform .
```

When Docker is available:

```bash
docker compose down
docker compose build --no-cache
docker compose up -d
docker compose ps
docker compose logs --tail=200
```

Do not claim Docker or Render tests passed unless actually executed.

---

# 25. Manual route testing

Manually test:

```text
/health
/login
/register
/profile
/dashboard
/assistant
/diagnose
/diagnoses
/diagnoses/:id
/market-prices
/resources
/resources/:id
/notifications
/officer
/officer/diagnoses
/admin
/admin/users
/admin/diagnoses
/admin/reviews
/admin/audit-logs
```

---

# 26. MVP acceptance criteria

The platform is ready for full-system testing when:

## Farmer

- Can register
- Can log in
- Sees dashboard
- Sees name, district, language
- Opens AI assistant
- Sees weather
- Uploads crop image
- Views diagnosis result
- Views diagnosis history
- Receives notification
- Reads officer review
- Views market prices
- Views learning resources

## Officer

- Can log in
- Opens officer dashboard
- Sees diagnosis queue
- Opens crop image
- Reads AI assessment
- Claims case
- Adds review
- Recommends action or field visit
- Completes review

## Admin

- Can log in
- Views admin dashboard
- Views users
- Changes roles
- Activates/deactivates users
- Views diagnoses/reviews
- Views audit logs
- Manages market prices
- Manages resources

## Integration

- Existing AI routes are reused
- Existing diagnosis result is displayed to officers
- Farmer receives officer feedback
- Notifications work
- Navigation is role-aware
- Authorization is enforced
- Tests pass
- Docker build passes
- Render deployment remains functional

---

# 27. Final report

Create:

```text
AGRICONNECT_MVP_PLATFORM_COMPLETION_REPORT.md
```

Include:

1. Reports inspected
2. Existing features preserved
3. Missing features discovered
4. Database migrations added
5. Files created
6. Files modified
7. Routes added
8. Farmer dashboard implementation
9. Officer review implementation
10. Notification implementation
11. Admin implementation
12. Market price implementation
13. Learning resource implementation
14. AI integration points reused
15. Authorization rules
16. Demo seed instructions
17. Demo guide created
18. Tests added
19. Commands executed
20. Test results
21. Docker result
22. Render deployment notes
23. Manual testing results
24. Remaining limitations

Do not stop after planning.

Implement the remaining MVP platform, connect it to the existing AI routes, run tests, and create the final report.
