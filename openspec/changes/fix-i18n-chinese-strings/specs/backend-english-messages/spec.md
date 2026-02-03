## ADDED Requirements

### Requirement: Backend API responses use English messages
The backend SHALL return all user-facing messages in English, including error messages, validation messages, success messages, and informational messages.

#### Scenario: Authentication error message in English
- **WHEN** user attempts to login with password authentication disabled
- **THEN** API returns `{"message": "Password login has been disabled by administrator", "success": false}`

#### Scenario: Validation error message in English
- **WHEN** user submits invalid parameters to an API endpoint
- **THEN** API returns `{"message": "Invalid parameters", "success": false}`

#### Scenario: Authorization error message in English
- **WHEN** user attempts an action without proper permissions
- **THEN** API returns error message in English format (e.g., "Unauthorized to perform this action")

### Requirement: Error wrapper functions return English messages
All error wrapper functions (ClaudeErrorWrapper, TaskErrorWrapper, etc.) SHALL return English error messages instead of Chinese.

#### Scenario: Upstream request failure message in English
- **WHEN** request to upstream service fails
- **THEN** error wrapper returns "Failed to request upstream address" instead of "请求上游地址失败"

#### Scenario: Network error message in English
- **WHEN** network error occurs during API call
- **THEN** error message is returned in English with masked sensitive information

### Requirement: User management messages in English
All user management operations (registration, login, profile updates, quota management) SHALL return English messages.

#### Scenario: Registration disabled message
- **WHEN** user attempts to register with registration disabled
- **THEN** API returns "Administrator has disabled new user registration"

#### Scenario: Two-factor authentication required message
- **WHEN** user with 2FA enabled attempts login
- **THEN** API returns "Please enter two-factor authentication code"

#### Scenario: Quota transfer success message
- **WHEN** administrator successfully transfers quota between users
- **THEN** API returns "Transfer successful"

### Requirement: Middleware messages in English
All middleware error messages (authentication, authorization, rate limiting, channel distribution) SHALL be in English.

#### Scenario: Token access denied message
- **WHEN** token attempts to access unauthorized model
- **THEN** middleware returns "This token is not authorized to access model {model_name}"

#### Scenario: Channel disabled message
- **WHEN** request routed to disabled channel
- **THEN** middleware returns "This channel has been disabled"

#### Scenario: IP restriction message
- **WHEN** request comes from unauthorized IP address
- **THEN** middleware returns "Your IP address is not in the token's allowed access list"

### Requirement: Service layer messages in English
All service layer messages (channel management, quota warnings, billing) SHALL be in English.

#### Scenario: Channel auto-disable notification
- **WHEN** channel is automatically disabled due to errors
- **THEN** system logs "Channel '{name}' (#{id}) has been disabled due to: {reason}"

#### Scenario: Quota warning notification
- **WHEN** user quota falls below threshold
- **THEN** notification message is "Your quota is running low. Current remaining quota: {amount}. Please recharge in time."

#### Scenario: Balance insufficient message
- **WHEN** channel balance check returns insufficient funds
- **THEN** API returns "Insufficient balance"
