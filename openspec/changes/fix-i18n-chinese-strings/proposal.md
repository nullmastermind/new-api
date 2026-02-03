## Why

The codebase currently has extensive hardcoded Chinese strings in both frontend (React) and backend (Go) code, preventing non-Chinese users from using the application effectively. While the frontend has i18n infrastructure (i18next with 6 languages), it's incompletely applied. The backend has no i18n system and returns Chinese error messages. This change will make the application fully accessible to international users by replacing Chinese strings with English and ensuring proper i18n coverage.

## What Changes

- Replace ~150 hardcoded Chinese strings in Go backend API responses with English messages
- Wrap ~500+ hardcoded Chinese strings in frontend React components with `t()` translation function
- Add English translations to existing i18n locale files (en.json)
- Ensure Vietnamese (vi.json) translations are complete for all new keys
- Add automated tests to verify i18n coverage and prevent future regressions
- Focus on incremental, high-quality manual translation (no bulk find-replace scripts)

## Capabilities

### New Capabilities
- `backend-english-messages`: Backend API responses return English messages instead of Chinese
- `frontend-i18n-coverage`: All user-facing strings in frontend use i18n translation system
- `i18n-test-coverage`: Automated tests verify i18n coverage and prevent hardcoded strings

### Modified Capabilities
<!-- No existing capabilities are being modified at the spec level -->

## Impact

**Backend:**
- `controller/user.go` (~60 strings): Authentication, registration, user management messages
- `middleware/auth.go` (~15 strings): Authorization error messages
- `middleware/distributor.go` (~15 strings): Model access and channel routing messages
- `service/error.go` (~5 strings): Error wrapper functions
- Other middleware and service files (~20 strings)

**Frontend:**
- `web/src/services/secureVerification.js`: Authentication service error messages
- `web/src/pages/Home/index.jsx`: Home page UI strings
- `web/src/pages/Setting/`: Settings pages (~15 files)
- `web/src/components/`: Auth, table, and common components (~50+ files)
- `web/src/i18n/locales/en.json`: English translations
- `web/src/i18n/locales/vi.json`: Vietnamese translations

**Testing:**
- New test files to verify i18n coverage in both frontend and backend
- CI/CD integration to prevent hardcoded strings in future PRs
