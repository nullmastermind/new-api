## 1. Backend - User Controller (controller/user.go)

- [x] 1.1 Replace authentication error messages (lines 33-85): "管理员关闭了密码登录" → "Password login has been disabled by administrator"
- [x] 1.2 Replace registration error messages (lines 147-262): "管理员关闭了新用户注册" → "Administrator has disabled new user registration"
- [x] 1.3 Replace user management messages (lines 321-400): "无权获取同级或更高等级用户的信息" → "Unauthorized to access user information at same or higher permission level"
- [x] 1.4 Replace user update messages (lines 606-698): "无效的参数" → "Invalid parameters"
- [x] 1.5 Replace password and deletion messages (lines 709-816): "原密码错误" → "Incorrect original password"
- [x] 1.6 Replace user operation messages (lines 840-972): "无法创建权限大于等于自己的用户" → "Cannot create user with permission level greater than or equal to your own"
- [x] 1.7 Replace verification and alert messages (lines 1001-1158): "验证码错误或已过期" → "Verification code is incorrect or expired"
- [ ] 1.8 Test all user controller endpoints with Postman/curl to verify English responses

## 2. Backend - Authentication Middleware (middleware/auth.go)

- [x] 2.1 Replace unauthorized access messages (lines 46-123): "无权进行此操作，未登录且未提供 access token" → "Unauthorized to perform this action, not logged in and no access token provided"
- [x] 2.2 Replace IP restriction messages (lines 256-260): "您的 IP 不在令牌允许访问的列表中" → "Your IP address is not in the token's allowed access list"
- [x] 2.3 Replace user ban messages (lines 273): "用户已被封禁" → "User has been banned"
- [x] 2.4 Replace group access messages (lines 284-290): "无权访问 %s 分组" → "Unauthorized to access group %s"
- [x] 2.5 Replace channel specification messages (lines 330-331): "普通用户不支持指定渠道" → "Regular users cannot specify channels"
- [ ] 2.6 Test authentication flows to verify error messages display correctly

## 3. Backend - Distributor Middleware (middleware/distributor.go)

- [x] 3.1 Replace channel validation messages (lines 41-50): "无效的渠道 Id" → "Invalid channel ID"
- [x] 3.2 Replace model access messages (lines 61-78): "该令牌无权访问任何模型" → "This token is not authorized to access any models"
- [x] 3.3 Replace playground and group messages (lines 88-93): "无效的playground请求" → "Invalid playground request"
- [x] 3.4 Replace distributor error messages (lines 136-198): "获取分组 %s 下模型 %s 的可用渠道失败" → "Failed to get available channels for model %s in group %s"
- [ ] 3.5 Test model access and channel routing with various scenarios

## 4. Backend - Error Service (service/error.go)

- [x] 4.1 Replace ClaudeErrorWrapper message (line 65): "请求上游地址失败" → "Failed to request upstream address"
- [x] 4.2 Verify error wrapper functions return English messages
- [x] 4.3 Test error handling with various upstream failure scenarios

## 5. Backend - Other Middleware and Services

- [x] 5.1 Replace secure verification messages (middleware/secure_verification.go lines 28-70): "未登录" → "Not logged in"
- [x] 5.2 Replace turnstile check messages (middleware/turnstile-check.go lines 30-73): "Turnstile token 为空" → "Turnstile token is empty"
- [x] 5.3 Replace channel service messages (service/channel.go lines 22-42): "通道「%s」（#%d）发生错误" → "Channel '%s' (#%d) encountered an error"
- [x] 5.4 Replace channel affinity messages (service/channel_affinity.go lines 198-219): "rule_name 不能为空" → "rule_name cannot be empty"
- [x] 5.5 Replace quota service messages (service/quota.go lines 482-571): "quota 不能为负数！" → "Quota cannot be negative!"
- [x] 5.6 Replace channel billing messages (controller/channel-billing.go lines 370-476): "尚未实现" → "Not yet implemented"
- [x] 5.7 Test all affected endpoints to verify English messages

## 6. Frontend - Authentication Service (web/src/services/secureVerification.js)

- [x] 6.1 Wrap error messages in t() function (lines 97-182): 'throw new Error(t("请输入验证码或备用码"))'
- [x] 6.2 Add translation keys to web/src/i18n/locales/zh.json for original Chinese text
- [x] 6.3 Add English translations to web/src/i18n/locales/en.json
- [x] 6.4 Add Vietnamese translations to web/src/i18n/locales/vi.json
- [ ] 6.5 Test authentication flows (2FA, passkey) to verify translations work

## 7. Frontend - Home Page (web/src/pages/Home/index.jsx)

- [x] 7.1 Wrap error message in t() function (line 108): 'setHomePageContent(t("加载首页内容失败..."))'
- [x] 7.2 Verify all other user-facing strings use t() function
- [x] 7.3 Add translation keys to locale files
- [ ] 7.4 Test home page in English and Vietnamese languages

## 8. Frontend - Settings Pages (web/src/pages/Setting/)

- [x] 8.1 Wrap all Chinese strings in GroupRatioSettings.jsx with t() function
- [x] 8.2 Wrap all Chinese strings in other settings page components
- [x] 8.3 Run 'bun run i18n:extract' to extract new translation keys
- [x] 8.4 Add English translations for all extracted keys
- [x] 8.5 Add Vietnamese translations for all extracted keys
- [ ] 8.6 Test settings pages in multiple languages

## 9. Frontend - Auth Components

- [x] 9.1 Wrap all Chinese strings in LoginForm.jsx with t() function
- [x] 9.2 Wrap all Chinese strings in RegisterForm.jsx with t() function
- [x] 9.3 Wrap all Chinese strings in TwoFAVerification.jsx with t() function
- [x] 9.4 Wrap all Chinese strings in SecureVerificationModal.jsx with t() function
- [x] 9.5 Wrap all Chinese strings in TwoFactorAuthModal.jsx with t() function
- [x] 9.6 Run 'bun run i18n:extract' and add translations
- [ ] 9.7 Test complete authentication flows in English and Vietnamese

## 10. Frontend - Table Components

- [x] 10.1 Identify all table components with hardcoded Chinese strings
- [x] 10.2 Wrap column headers, action buttons, and messages with t() function
- [x] 10.3 Wrap filter labels and placeholders with t() function
- [x] 10.4 Run 'bun run i18n:extract' to extract new translation keys
- [ ] 10.5 Test table components with sorting, filtering, and pagination

## 11. Frontend - Other Pages and Components

- [x] 11.1 Wrap Chinese strings in About page (web/src/pages/About/index.jsx)
- [x] 11.2 Wrap Chinese strings in UserAgreement page
- [x] 11.3 Wrap Chinese strings in PrivacyPolicy page
- [x] 11.4 Wrap Chinese strings in remaining page components
- [x] 11.5 Wrap Chinese strings in modal and form components
- [x] 11.6 Run 'bun run i18n:extract' for final extraction
- [x] 11.7 Complete all English and Vietnamese translations
- [ ] 11.8 Test all pages and components in multiple languages

## 12. Backend - I18n Tests

- [x] 12.1 Create test file: common/i18n_test.go
- [x] 12.2 Implement test to scan controller files for Chinese characters in API responses
- [x] 12.3 Implement test to scan middleware files for Chinese characters in error messages
- [x] 12.4 Implement test to scan service files for Chinese characters in user-facing messages
- [ ] 12.5 Add test to CI/CD pipeline (update .github/workflows or similar)
- [x] 12.6 Run tests locally and verify they pass
- [x] 12.7 Document how to run i18n tests in README or CONTRIBUTING.md

## 13. Frontend - I18n Tests

- [x] 13.1 Create test file: web/src/__tests__/i18n.test.js
- [x] 13.2 Implement test to scan JSX files for hardcoded Chinese string literals
- [x] 13.3 Implement test to verify all t() keys exist in en.json and vi.json
- [x] 13.4 Implement test to check for orphaned translation keys
- [x] 13.5 Add test script to package.json: "test:i18n"
- [ ] 13.6 Integrate i18n tests into CI/CD pipeline
- [ ] 13.7 Run tests locally and verify they pass
- [ ] 13.8 Document i18n testing guidelines for developers

## 14. Documentation and Final Verification

- [ ] 14.1 Update AGENTS.md with i18n guidelines for new code
- [ ] 14.2 Create i18n guide document explaining how to add new translations
- [ ] 14.3 Document translation key naming conventions
- [ ] 14.4 Perform comprehensive manual testing of all user flows in English
- [ ] 14.5 Perform comprehensive manual testing of all user flows in Vietnamese
- [ ] 14.6 Verify Chinese language still works correctly (fallback)
- [ ] 14.7 Create release notes documenting the i18n improvements
- [ ] 14.8 Review all changes for quality and consistency
