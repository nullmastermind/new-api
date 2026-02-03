## ADDED Requirements

### Requirement: All user-facing strings use i18n translation
The frontend SHALL wrap all user-facing strings with the `t()` translation function from the `useTranslation()` hook.

#### Scenario: Error message uses translation
- **WHEN** error occurs in authentication service
- **THEN** error message is wrapped with `t('error.key')` instead of hardcoded Chinese string

#### Scenario: UI label uses translation
- **WHEN** component renders a label or button text
- **THEN** text is wrapped with `t('label.key')` instead of hardcoded string

#### Scenario: Toast notification uses translation
- **WHEN** `showError()` or `showSuccess()` is called
- **THEN** message parameter uses `t('message.key')` instead of hardcoded string

### Requirement: Translation keys exist in locale files
All translation keys used in the frontend SHALL have corresponding entries in locale files for English (en.json) and Vietnamese (vi.json).

#### Scenario: English translation exists
- **WHEN** component uses `t('some.key')`
- **THEN** `en.json` contains entry for `"some.key": "English translation"`

#### Scenario: Vietnamese translation exists
- **WHEN** component uses `t('some.key')`
- **THEN** `vi.json` contains entry for `"some.key": "Vietnamese translation"`

#### Scenario: Chinese translation preserved
- **WHEN** component uses `t('some.key')`
- **THEN** `zh.json` contains entry for `"some.key": "中文翻译"` (original Chinese text)

### Requirement: Authentication service messages use i18n
All error messages in `services/secureVerification.js` SHALL use the `t()` function.

#### Scenario: Verification code required message
- **WHEN** user submits empty verification code
- **THEN** error thrown is `new Error(t('请输入验证码或备用码'))`

#### Scenario: Verification failed message
- **WHEN** verification API call fails
- **THEN** error message uses `t('验证失败')` instead of hardcoded string

#### Scenario: Passkey verification cancelled message
- **WHEN** user cancels passkey verification
- **THEN** error message uses `t('Passkey 验证被取消')` instead of hardcoded string

### Requirement: Page components use i18n
All page components (Home, Settings, About, etc.) SHALL use `t()` for all user-facing text.

#### Scenario: Home page error message
- **WHEN** home page content fails to load
- **THEN** error message uses `t('加载首页内容失败...')` instead of hardcoded string

#### Scenario: Settings page labels
- **WHEN** settings page renders form labels
- **THEN** all labels use `t('label.key')` format

#### Scenario: Empty state messages
- **WHEN** page displays empty state
- **THEN** empty state description uses `t('description.key')`

### Requirement: Table and form components use i18n
All table components, form components, and modals SHALL use `t()` for labels, placeholders, and messages.

#### Scenario: Table column headers
- **WHEN** table renders column headers
- **THEN** headers use `t('column.header.key')` instead of hardcoded strings

#### Scenario: Form validation messages
- **WHEN** form validation fails
- **THEN** validation message uses `t('validation.message.key')`

#### Scenario: Modal dialog text
- **WHEN** modal dialog is displayed
- **THEN** title and content use `t()` function

### Requirement: Console log messages remain in original language
Console log messages and debug statements MAY remain in Chinese or English as they are for developer use only.

#### Scenario: Debug console log
- **WHEN** developer debugging code logs to console
- **THEN** console.log messages are not required to use i18n

#### Scenario: Error logging
- **WHEN** error is logged for debugging purposes
- **THEN** log message can be in any language (not user-facing)
