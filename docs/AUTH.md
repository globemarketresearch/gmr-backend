# Authentication System Documentation

## Overview

The Healthcare Market Research API uses JWT (JSON Web Token) based authentication with refresh token rotation for secure user authentication and authorization.

## Security Features

- **JWT-based Authentication**: Stateless authentication using signed JWT tokens
- **Password Hashing**: Bcrypt hashing with cost factor 12 (~250ms on modern hardware)
- **Rate Limiting**: Protection against brute-force attacks (5 attempts per 15 minutes)
- **Refresh Token Rotation**: Automatic rotation of refresh tokens for enhanced security
- **Token Revocation**: Redis-backed token storage for immediate revocation capability
- **CSRF Protection**: Optional CSRF middleware available (recommended for cookie-based auth)
- **Role-Based Access Control**: Support for admin, editor, and viewer roles

## Authentication Endpoints

### 1. POST /api/v1/auth/login

Primary authentication endpoint for user login.

**Request:**
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "token_type": "Bearer",
    "expires_in": 900,
    "user": {
      "id": 1,
      "email": "user@example.com",
      "name": "John Doe",
      "role": "admin",
      "is_active": true,
      "last_login_at": "2025-12-26T10:30:00Z",
      "created_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-12-26T10:30:00Z"
    }
  }
}
```

**Error Responses:**
- `400 Bad Request`: Invalid request body or missing fields
- `401 Unauthorized`: Invalid email or password
- `429 Too Many Requests`: Rate limit exceeded (includes `Retry-After` header)

**Features:**
- Rate limiting: 5 attempts per 15 minutes (configurable)
- Updates `last_login_at` timestamp
- Stores refresh token in Redis for revocation capability
- Returns both access and refresh tokens

**cURL Example:**
```bash
curl -X POST http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@example.com",
    "password": "Admin@123"
  }'
```

---

### 2. POST /api/v1/auth/refresh

Token refresh endpoint for obtaining new access tokens.

**Request:**
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "token_type": "Bearer",
    "expires_in": 900
  }
}
```

**Error Responses:**
- `400 Bad Request`: Missing refresh token
- `401 Unauthorized`: Invalid, expired, or revoked refresh token

**Features:**
- Automatic refresh token rotation (old token invalidated)
- Validates token hasn't been revoked (checks Redis)
- Returns new access token and new refresh token
- Old refresh token is deleted from Redis

**cURL Example:**
```bash
curl -X POST http://localhost:8081/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }'
```

---

### 3. POST /api/v1/auth/logout

Logout endpoint for invalidating refresh tokens.

**Request Headers:**
```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

**Request Body (Optional):**
```json
{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**Response (200 OK):**
```json
{
  "success": true,
  "data": {
    "message": "Logout successful"
  }
}
```

**Error Responses:**
- `401 Unauthorized`: Missing or invalid access token

**Features:**
- Requires valid access token
- Invalidates refresh token by removing from Redis
- Gracefully handles missing refresh token in request body

**cURL Example:**
```bash
curl -X POST http://localhost:8081/api/v1/auth/logout \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json" \
  -d '{
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
  }'
```

---

## JWT Token Structure

### Access Token Claims

```json
{
  "id": 1,                          // User ID
  "email": "user@example.com",      // User email
  "role": "admin",                  // User role (admin/editor/viewer)
  "token_type": "access",           // Token type
  "sub": "1",                       // Subject (User ID as string)
  "iss": "healthcare-market-research-api",  // Issuer
  "iat": 1703592000,                // Issued at (Unix timestamp)
  "exp": 1703592900,                // Expiration time (Unix timestamp)
  "nbf": 1703592000,                // Not before (Unix timestamp)
  "jti": "uuid-here"                // JWT ID (unique identifier)
}
```

### Refresh Token Claims

Same structure as access token, but with `token_type: "refresh"` and longer expiration time.

### Token Expiration Defaults

- **Access Token**: 15 minutes (configurable via `JWT_ACCESS_TOKEN_EXPIRY`)
- **Refresh Token**: 7 days / 168 hours (configurable via `JWT_REFRESH_TOKEN_EXPIRY`)

---

## Using Authentication in Requests

### Protected Endpoints

All protected endpoints require the access token in the Authorization header:

```
Authorization: Bearer <access_token>
```

**Example:**
```bash
curl -X GET http://localhost:8081/api/v1/users/me \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

### Role-Based Access

Some endpoints require specific roles:

- **Admin Only**: User management, report deletion
- **Admin or Editor**: Create/update reports
- **All Authenticated Users**: View profile, logout

---

## Authentication Flow

### Initial Authentication

```
Client                          Server                         Redis
  |                               |                              |
  |-- POST /auth/login ---------->|                              |
  |    (email, password)          |                              |
  |                               |--- Validate credentials ---->|
  |                               |                              |
  |                               |--- Generate tokens --------->|
  |                               |                              |
  |                               |--- Store refresh token ----->|
  |                               |                              |
  |<-- 200 OK -------------------|                              |
  |    (access_token,             |                              |
  |     refresh_token, user)      |                              |
```

### Token Refresh

```
Client                          Server                         Redis
  |                               |                              |
  |-- POST /auth/refresh -------->|                              |
  |    (refresh_token)            |                              |
  |                               |--- Validate token ---------->|
  |                               |                              |
  |                               |<-- Check Redis --------------|
  |                               |                              |
  |                               |--- Delete old token -------->|
  |                               |                              |
  |                               |--- Generate new tokens ----->|
  |                               |                              |
  |                               |--- Store new refresh token ->|
  |                               |                              |
  |<-- 200 OK -------------------|                              |
  |    (new access_token,         |                              |
  |     new refresh_token)        |                              |
```

### Logout

```
Client                          Server                         Redis
  |                               |                              |
  |-- POST /auth/logout --------->|                              |
  |    (Authorization header,     |                              |
  |     refresh_token)            |                              |
  |                               |--- Validate access token --->|
  |                               |                              |
  |                               |--- Delete refresh token ---->|
  |                               |                              |
  |<-- 200 OK -------------------|                              |
  |    (logout success)           |                              |
```

---

## Configuration

### Environment Variables

```bash
# JWT Authentication
JWT_SECRET=your-super-secret-key-change-in-production
JWT_ACCESS_TOKEN_EXPIRY=15m
JWT_REFRESH_TOKEN_EXPIRY=168h
JWT_ISSUER=healthcare-market-research-api

# Rate Limiting
RATE_LIMIT_LOGIN_MAX_ATTEMPTS=5
RATE_LIMIT_LOGIN_WINDOW=15m

# Redis (required for token storage)
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0
```

### Security Recommendations

1. **JWT_SECRET**: Use a strong, randomly generated secret (min 32 characters)
2. **HTTPS Only**: Always use HTTPS in production
3. **Token Storage**: Store tokens in memory or httpOnly cookies (not localStorage for sensitive apps)
4. **Password Policy**: Enforce strong passwords (min 8 characters)
5. **Rate Limiting**: Adjust based on your security requirements
6. **Token Expiry**: Balance security vs user experience

---

## Error Handling

### Common Error Responses

**401 Unauthorized:**
```json
{
  "success": false,
  "error": "Invalid email or password"
}
```

**429 Too Many Requests:**
```json
{
  "success": false,
  "error": "Too many requests. Please try again later."
}
```

**403 Forbidden (CSRF):**
```json
{
  "success": false,
  "error": "CSRF token missing"
}
```

---

## Testing

### Create Admin User

Run the seed command to create a default admin user:

```bash
go run cmd/seed/main.go
```

Default credentials:
- Email: `admin@example.com`
- Password: `Admin@123`

**Important:** Change the default password after first login!

### Test Authentication Flow

1. **Login:**
```bash
curl -X POST http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"Admin@123"}'
```

2. **Use Access Token:**
```bash
export TOKEN="<access_token_from_login>"
curl -X GET http://localhost:8081/api/v1/users/me \
  -H "Authorization: Bearer $TOKEN"
```

3. **Refresh Token:**
```bash
export REFRESH_TOKEN="<refresh_token_from_login>"
curl -X POST http://localhost:8081/api/v1/auth/refresh \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}"
```

4. **Logout:**
```bash
curl -X POST http://localhost:8081/api/v1/auth/logout \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"refresh_token\":\"$REFRESH_TOKEN\"}"
```

---

## Troubleshooting

### Token Validation Failures

1. **"Token has expired"**: Access token expired (15min default), use refresh endpoint
2. **"Invalid or expired refresh token"**: Refresh token expired or revoked, re-login required
3. **"Token has been revoked"**: Token not found in Redis, user logged out or token manually revoked

### Rate Limiting Issues

1. **"Too many requests"**: Wait for the time specified in `Retry-After` header
2. Check `RATE_LIMIT_LOGIN_WINDOW` and `RATE_LIMIT_LOGIN_MAX_ATTEMPTS` configuration
3. Rate limit is per IP address per endpoint

### Redis Connection Issues

1. Verify Redis is running: `redis-cli ping`
2. Check Redis connection in logs
3. App continues without caching if Redis unavailable, but token storage will fail

---

## API Documentation

Full API documentation with interactive testing is available via Swagger UI:

```
http://localhost:8081/swagger/index.html
```

---

## Security Considerations

### Password Security
- Bcrypt hashing with cost factor 12
- Minimum password length: 8 characters
- Never expose password hash in API responses

### Token Security
- Short-lived access tokens (15 minutes)
- Refresh token rotation on every refresh
- Token revocation via Redis
- HMAC-SHA256 signing algorithm

### Network Security
- Always use HTTPS in production
- Configure CORS appropriately
- Rate limiting on authentication endpoints
- SameSite cookies for CSRF tokens

### Operational Security
- Monitor failed login attempts
- Implement account lockout after N failed attempts (future enhancement)
- Regular security audits
- Keep dependencies updated

---

## Future Enhancements

- [ ] Account lockout after failed attempts
- [ ] Email-based password reset
- [ ] Two-factor authentication (2FA)
- [ ] Session management dashboard
- [ ] Token blacklisting for immediate revocation
- [ ] OAuth2 integration (Google, GitHub, etc.)
- [ ] Audit logging for authentication events
