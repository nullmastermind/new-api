## ADDED Requirements

### Requirement: Automated tests verify no hardcoded Chinese strings in backend
The test suite SHALL include tests that scan backend Go files for hardcoded Chinese characters in user-facing messages.

#### Scenario: Backend API response test
- **WHEN** test scans controller files for Chinese characters
- **THEN** test fails if Chinese characters found in API response messages

#### Scenario: Middleware message test
- **WHEN** test scans middleware files for Chinese characters
- **THEN** test fails if Chinese characters found in error messages

#### Scenario: Service layer message test
- **WHEN** test scans service files for Chinese characters
- **THEN** test fails if Chinese characters found in user-facing messages

### Requirement: Automated tests verify frontend i18n coverage
The test suite SHALL include tests that verify all user-facing strings in frontend use the `t()` translation function.

#### Scenario: JSX string literal test
- **WHEN** test scans JSX files for string literals
- **THEN** test fails if Chinese string literals found outside of `t()` function

#### Scenario: Error message test
- **WHEN** test scans service files for error messages
- **THEN** test fails if error messages not wrapped in `t()` function

#### Scenario: Translation key existence test
- **WHEN** test extracts all `t('key')` calls from code
- **THEN** test fails if any key missing from en.json or vi.json

### Requirement: Tests verify translation completeness
The test suite SHALL verify that all translation keys have entries in all supported locale files.

#### Scenario: English translation completeness
- **WHEN** test compares keys in zh.json with en.json
- **THEN** test fails if any key in zh.json missing from en.json

#### Scenario: Vietnamese translation completeness
- **WHEN** test compares keys in zh.json with vi.json
- **THEN** test fails if any key in zh.json missing from vi.json

#### Scenario: No orphaned keys
- **WHEN** test scans locale files for unused keys
- **THEN** test warns if translation keys exist but are not used in code

### Requirement: CI/CD integration prevents i18n regressions
The CI/CD pipeline SHALL run i18n tests on every pull request to prevent hardcoded strings from being merged.

#### Scenario: PR with hardcoded Chinese string
- **WHEN** pull request contains new hardcoded Chinese string
- **THEN** CI/CD pipeline fails with clear error message

#### Scenario: PR with missing translation
- **WHEN** pull request adds new `t('key')` without adding to locale files
- **THEN** CI/CD pipeline fails with message indicating missing translation

#### Scenario: PR with proper i18n
- **WHEN** pull request properly uses `t()` and includes translations
- **THEN** CI/CD pipeline passes i18n tests

### Requirement: Test documentation and maintenance
The test suite SHALL include documentation on how to run i18n tests locally and how to fix common issues.

#### Scenario: Developer runs i18n tests locally
- **WHEN** developer runs `npm run test:i18n` or `go test ./...`
- **THEN** tests execute and provide clear output on any i18n violations

#### Scenario: Test failure provides actionable guidance
- **WHEN** i18n test fails
- **THEN** error message includes file path, line number, and suggested fix

#### Scenario: Test documentation is accessible
- **WHEN** developer reads test documentation
- **THEN** documentation explains how to add new translations and avoid common pitfalls
