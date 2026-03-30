# SafeRoute Backend Auth

## What This Document Is For

This file explains how authentication currently works in the SafeRoute backend:

- what auth flow is implemented
- which routes exist today
- how JWTs and cookies are used
- how to protect future routes with reusable middleware

This doc reflects the current backend code, not an aspirational future design.

## Current Auth Model

The backend currently uses:

- password-based login
- bcrypt for password hashing
- JWT access and refresh tokens
- httpOnly cookies for session transport

The main pieces are:

- `internal/auth/service.go`
  - register and login business logic
- `internal/auth/repository.go`
  - user persistence via GORM
- `internal/auth/session.go`
  - JWT issuance, parsing, and cookie handling
- `internal/auth/middleware.go`
  - reusable `VerifyUser()` middleware for protected routes
- `internal/auth/handler.go`
  - Fiber route handlers under `/api/v1/auth`

Trusted contacts now live in a separate backend module and are documented in `apps/docs/TRUSTED_CONTACTS.md`.

## Session Design

The backend issues two JWTs after successful register or login:

- access token
  - short-lived
  - used for protected API routes
  - read from the access cookie
- refresh token
  - longer-lived
  - used only to mint a fresh token pair
  - read from the refresh cookie

Both tokens are sent to the browser as `httpOnly` cookies, so frontend code should not try to read them directly from JavaScript.

## Cookie Behavior

Cookie names and behavior come from env config:

- `AUTH_ACCESS_COOKIE_NAME`
  - default: `saferoute_access`
- `AUTH_REFRESH_COOKIE_NAME`
  - default: `saferoute_refresh`
- `AUTH_COOKIE_DOMAIN`
  - default: empty
- `AUTH_COOKIE_SAME_SITE`
  - default: `Lax`
- `AUTH_COOKIE_SECURE`
  - default: `false` in development, `true` in production fallback logic

Current cookie properties:

- `HttpOnly: true`
- `Path: /`
- `Expires` set to token expiry
- `SameSite` configurable
- `Secure` configurable

## JWT Configuration

The current defaults are:

- access token TTL
  - `JWT_ACCESS_TTL=15m`
- refresh token TTL
  - `JWT_REFRESH_TTL=168h`

Secrets:

- `JWT_ACCESS_SECRET`
- `JWT_REFRESH_SECRET`

These must be different in real environments. The development defaults are only placeholders.

## Implemented Routes

Base path: `/api/v1/auth`

### `POST /register`

Creates a user, hashes the password, and immediately signs the user in by setting auth cookies.

Request body:

```json
{
  "phone": "+91 99999 11111",
  "email": "person@example.com",
  "password": "supersecret"
}
```

Notes:

- `phone` is required
- `password` must be at least 8 characters
- `email` is optional
- phone normalization currently removes whitespace only

Success response: `201 Created`

```json
{
  "user": {
    "id": "uuid-or-generated-id",
    "phone": "+919999911111",
    "email": "person@example.com",
    "trust_score": 0.3,
    "verified": false
  }
}
```

Side effects:

- sets access cookie
- sets refresh cookie

Common errors:

- `400` if phone or password is invalid
- `409` if user already exists

### `POST /login`

Authenticates a user using phone and password, then issues a fresh auth cookie pair.

Request body:

```json
{
  "phone": "+919999911111",
  "password": "supersecret"
}
```

Success response: `200 OK`

```json
{
  "user": {
    "id": "user-id",
    "phone": "+919999911111",
    "trust_score": 0.3,
    "verified": false
  }
}
```

Common errors:

- `401` for invalid credentials

### `POST /refresh`

Reads the refresh token cookie, validates it, reloads the user, and issues a new access/refresh pair.

Request body:

- none

Success response: `200 OK`

```json
{
  "user": {
    "id": "user-id",
    "phone": "+919999911111",
    "trust_score": 0.3,
    "verified": false
  }
}
```

Side effects:

- rotates both cookies by issuing a fresh pair
- clears cookies if the refresh token is invalid or the user no longer exists

Common errors:

- `401` if the refresh cookie is missing or invalid

### `POST /logout`

Clears both auth cookies.

Request body:

- none

Success response: `200 OK`

```json
{
  "status": "logged_out"
}
```

### `GET /me`

Returns the currently authenticated user.

Authentication:

- requires a valid access token cookie

Success response: `200 OK`

```json
{
  "user": {
    "id": "user-id",
    "phone": "+919999911111",
    "trust_score": 0.3,
    "verified": false
  }
}
```

Common errors:

- `401` if the access cookie is missing, invalid, or the user cannot be found

## Reusable Middleware

Protected routes should use the auth middleware instead of duplicating cookie/JWT parsing logic.

Middleware entry point:

- `auth.NewMiddleware(service, sessions).VerifyUser()`

After verification, the current user is stored in Fiber locals and can be retrieved with:

- `auth.CurrentUser(c)`

Example:

```go
authMiddleware := auth.NewMiddleware(authService, sessionManager)

api.Get("/protected", authMiddleware.VerifyUser(), func(c *fiber.Ctx) error {
	user, ok := auth.CurrentUser(c)
	if !ok {
		return c.SendStatus(fiber.StatusUnauthorized)
	}

	return c.JSON(fiber.Map{
		"user_id": user.ID,
		"phone":   user.Phone,
	})
})
```

## Error Mapping

Current auth errors map like this:

- `invalid request body`
  - `400`
- `phone is required`
  - `400`
- `password must be at least 8 characters`
  - `400`
- `invalid credentials`
  - `401`
- `unauthorized`
  - `401`
- `user already exists`
  - `409`
- unknown internal failure
  - `500`

## Environment Variables

Auth-related env vars currently used by the backend:

```env
JWT_ACCESS_SECRET=dev-access-secret-change-me
JWT_REFRESH_SECRET=dev-refresh-secret-change-me
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=168h
AUTH_ACCESS_COOKIE_NAME=saferoute_access
AUTH_REFRESH_COOKIE_NAME=saferoute_refresh
AUTH_COOKIE_DOMAIN=
AUTH_COOKIE_SAME_SITE=Lax
AUTH_COOKIE_SECURE=false
```

The backend also loads `.env` from the current working directory during startup when available.

## Current Limitations

This auth slice is intentionally minimal. It does not yet include:

- CSRF protection for cookie-based auth flows
- refresh-token revocation storage
- device/session management
- forgot-password or reset-password flow
- phone OTP login
- verification-provider integration

Those are the next security and product upgrades to layer on top of this base.
