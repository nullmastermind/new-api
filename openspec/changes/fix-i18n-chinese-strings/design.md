## Context

The codebase is a bilingual API gateway system (new-api) with:
- **Backend**: Go (Gin framework) with ~150 hardcoded Chinese strings in API responses
- **Frontend**: React with i18next infrastructure supporting 6 languages (zh, en, fr, ru, ja, vi)
- **Current State**: Frontend has i18n setup but incomplete coverage (~500+ hardcoded Chinese strings); backend has no i18n system
- **Constraints**: 
  - Must maintain backward compatibility with existing API consumers
  - Cannot use bulk find-replace scripts (quality over speed)
  - Must support English and Vietnamese as primary non-Chinese languages
  - Frontend already has i18next infrastructure; backend should stay language-agnostic

## Goals / Non-Goals

**Goals:**
- Replace all backend Chinese strings with English messages
- Wrap all frontend user-facing strings with `t()` translation function
- Ensure complete English and Vietnamese translations
- Add automated tests to prevent i18n regressions
- Maintain high translation quality through manual review

**Non-Goals:**
- Adding i18n library to Go backend (keep backend simple, English-only)
- Translating console.log or debug messages (developer-facing only)
- Translating all 6 languages immediately (focus on en, vi first)
- Creating a translation management system
- Changing API response structure or error codes

## Decisions

### Decision 1: Backend English-Only (No Go i18n Library)

**Choice**: Replace Chinese strings with English directly in Go code, without adding i18n library.

**Rationale**:
- Standard practice: APIs typically return English messages
- Frontend already has i18n infrastructure to translate for users
- Simpler implementation: no new dependencies or complexity
- Backend stays language-agnostic and maintainable

**Alternatives Considered**:
- **Add go-i18n library**: Rejected due to added complexity, maintenance burden, and unnecessary when frontend can handle translation
- **Return error codes only**: Rejected because human-readable messages improve developer experience

### Decision 2: Incremental Manual Translation (No Bulk Scripts)

**Choice**: Manually translate each string with context awareness, file by file.

**Rationale**:
- Higher quality translations that consider context
- Avoids awkward machine translations
- Allows for proper error message formatting
- Easier to review and test incrementally

**Alternatives Considered**:
- **Bulk find-replace with regex**: Rejected due to risk of incorrect translations and loss of context
- **Machine translation API**: Rejected due to quality concerns and need for human review anyway

### Decision 3: Priority-Based Implementation Order

**Choice**: Implement in phases: Backend critical paths → Frontend services → Frontend UI (by priority).

**Rationale**:
- Backend changes affect all API consumers (highest impact)
- Authentication/security services are critical user flows
- UI components can be done incrementally with lower risk
- Allows for testing and validation at each phase

**Alternatives Considered**:
- **All at once**: Rejected due to high risk and difficulty in testing
- **Frontend first**: Rejected because backend Chinese messages would still break non-Chinese users

### Decision 4: Test Strategy - Static Analysis + Runtime Tests

**Choice**: Combine static analysis (regex scanning) with runtime tests (translation key validation).

**Rationale**:
- Static analysis catches hardcoded strings at build time
- Runtime tests verify translation completeness
- CI/CD integration prevents regressions
- Low maintenance overhead

**Alternatives Considered**:
- **Manual code review only**: Rejected due to human error and scalability issues
- **Runtime-only tests**: Rejected because they don't catch hardcoded strings until execution

### Decision 5: Translation Key Naming Convention

**Choice**: Use hierarchical dot notation matching Chinese text structure (e.g., `error.auth.passwordDisabled`).

**Rationale**:
- Matches existing i18next patterns in codebase
- Easy to organize and find translations
- Supports namespacing for different modules
- Compatible with i18next extraction tools

**Alternatives Considered**:
- **Flat keys**: Rejected due to namespace collisions and poor organization
- **File-based keys**: Rejected due to complexity when components move

## Risks / Trade-offs

### Risk 1: Breaking Changes for API Consumers
**Risk**: Existing API consumers may parse Chinese error messages.
**Mitigation**: 
- Document the change in release notes
- Maintain error codes unchanged (only message text changes)
- Consider adding `Accept-Language` header support in future if needed

### Risk 2: Incomplete Translation Coverage
**Risk**: Missing translations cause fallback to Chinese or broken UI.
**Mitigation**:
- Automated tests verify all keys exist in en.json and vi.json
- CI/CD blocks PRs with missing translations
- Fallback language remains Chinese (zh) for graceful degradation

### Risk 3: Translation Quality Issues
**Risk**: Manual translation may have inconsistencies or errors.
**Mitigation**:
- Review translations in context during implementation
- Use consistent terminology (create glossary if needed)
- Test with native speakers for Vietnamese translations
- Allow for iterative improvements post-launch

### Risk 4: Large PR Size and Review Difficulty
**Risk**: Changing 500+ strings creates massive PRs that are hard to review.
**Mitigation**:
- Break into multiple PRs by phase (backend, frontend services, frontend UI)
- Each PR focuses on specific files or modules
- Include before/after examples in PR descriptions
- Automated tests reduce manual review burden

### Risk 5: Merge Conflicts During Long Implementation
**Risk**: Other developers may add new Chinese strings during implementation.
**Mitigation**:
- Implement in short iterations (1-2 days per phase)
- Add CI/CD tests early to catch new violations
- Communicate with team about i18n requirements
- Document i18n guidelines for new code

## Migration Plan

### Phase 1: Backend API Messages (Days 1-2)
1. Create branch `fix/i18n-backend`
2. Replace Chinese → English in priority order:
   - `controller/user.go` (authentication, registration)
   - `middleware/auth.go` (authorization)
   - `middleware/distributor.go` (model access)
   - `service/error.go` (error wrappers)
   - Other middleware and service files
3. Test API responses manually with Postman/curl
4. Create PR with before/after examples
5. Merge after review

### Phase 2: Frontend Services (Day 3)
1. Create branch `fix/i18n-frontend-services`
2. Wrap `t()` in `services/secureVerification.js`
3. Add translation keys to `en.json` and `vi.json`
4. Test authentication flows (login, 2FA, passkey)
5. Create PR and merge

### Phase 3: Frontend UI - Critical (Days 4-6)
1. Create branch `fix/i18n-frontend-critical`
2. Wrap `t()` in:
   - Auth components (Login, Register, 2FA)
   - Home page
   - Settings pages
3. Run `bun run i18n:extract` to update locale files
4. Translate new keys to English and Vietnamese
5. Test critical user flows
6. Create PR and merge

### Phase 4: Frontend UI - Remaining (Days 7-11)
1. Create branch `fix/i18n-frontend-remaining`
2. Wrap `t()` in remaining components:
   - Table components
   - Other pages
   - Modals and forms
3. Run `bun run i18n:extract`
4. Translate new keys
5. Comprehensive testing
6. Create PR and merge

### Phase 5: Automated Tests (Day 12)
1. Create branch `fix/i18n-tests`
2. Add backend test to scan for Chinese characters
3. Add frontend test to verify i18n coverage
4. Add translation completeness tests
5. Integrate into CI/CD pipeline
6. Create PR and merge

### Rollback Strategy
- Each phase is independently deployable
- If issues found, revert specific PR
- Backend changes are low-risk (only message text)
- Frontend changes can be rolled back without affecting backend
- No database migrations or schema changes required

## Open Questions

1. **Vietnamese Translation Source**: Who will provide Vietnamese translations? Should we use professional translation service or community contributors?
   - **Resolution needed before Phase 3**

2. **Error Code Standardization**: Should we standardize error codes alongside message changes?
   - **Decision**: Out of scope for this change, can be separate improvement

3. **Logging Language**: Should internal logs remain in Chinese or switch to English?
   - **Decision**: Keep logs as-is (developer-facing), focus on user-facing messages only

4. **Future Language Support**: When should we translate to remaining languages (fr, ru, ja)?
   - **Decision**: After en/vi are complete and stable, can be separate effort
