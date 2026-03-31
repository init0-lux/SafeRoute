# SafeRoute Trusted Contacts

## What This Document Is For

This file explains the dedicated trusted-contacts module:

- why it is separate from auth
- how the request-based flow works
- which routes exist today
- how the current SOS notification architecture should build on top of it

## Why This Is Separate From Auth

Trusted contacts are related to identity, but they are not part of login/session management.

Auth is responsible for:

- registering a user
- signing a user in
- maintaining the cookie-based session

The trusted-contacts module is responsible for:

- creating a pending trusted-contact invitation
- accepting that invitation
- storing the accepted trusted-contact relationship
- later serving as the source of truth for SOS notification targets

This separation keeps auth focused and makes the SOS safety network easier to evolve on its own.

## Product Flow

The current backend now treats trusted-contact addition as a request flow instead of an instant write.

### Step 1: Owner Creates A Request

The authenticated SafeRoute user creates a trusted-contact request with:

- contact name
- phone
- optional email

The backend stores a `trusted_contact_requests` row with:

- `status = pending`
- a hashed invitation token
- an expiry time

### Step 2: Request Is Delivered

The backend does not yet send SMS or email directly, but the returned accept token is meant to be used by a notification layer.

The intended production flow is:

1. create the request
2. build an SMS/email/deep link using the request id and token
3. send that link to the target contact

### Step 3: Contact Accepts

When the recipient accepts, the backend:

1. validates the request id
2. validates the invitation token
3. checks that the request is still pending and not expired
4. creates the actual `trusted_contacts` row
5. marks the request as `accepted`

Only accepted trusted contacts should be treated as real SOS recipients.

## Current Routes

Base path: `/api/v1/trusted-contacts`

### `POST /requests`

Creates a new pending trusted-contact request.

Authentication:

- required

Request body:

```json
{
  "name": "Emergency Contact",
  "phone": "+91 88888 22222",
  "email": "helper@example.com"
}
```

Success response: `201 Created`

```json
{
  "request": {
    "id": "request-id",
    "user_id": "user-id",
    "name": "Emergency Contact",
    "phone": "+918888822222",
    "email": "helper@example.com",
    "status": "pending",
    "expires_at": "2026-04-07T12:00:00Z",
    "created_at": "2026-03-31T12:00:00Z"
  },
  "accept_token": "plain-token-for-dev-flow"
}
```

Notes:

- the plain `accept_token` is returned right now so local development and Postman testing are easy
- in production, that token should be embedded into an SMS/email/deep link, not exposed broadly in app UI
- the database stores only the hashed token, not the raw token

Common errors:

- `400` for invalid name or phone
- `401` if the requester is not authenticated
- `409` if an accepted contact already exists for that phone or a pending request is already active

### `POST /requests/:id/accept`

Accepts a pending trusted-contact request.

Authentication:

- not required right now
- authorization is token-based

Request body:

```json
{
  "token": "plain-token-from-link"
}
```

Success response: `201 Created`

```json
{
  "request": {
    "id": "request-id",
    "user_id": "user-id",
    "name": "Emergency Contact",
    "phone": "+918888822222",
    "email": "helper@example.com",
    "status": "accepted",
    "expires_at": "2026-04-07T12:00:00Z",
    "responded_at": "2026-03-31T12:05:00Z",
    "created_at": "2026-03-31T12:00:00Z"
  },
  "contact": {
    "id": "contact-id",
    "user_id": "user-id",
    "request_id": "request-id",
    "name": "Emergency Contact",
    "phone": "+918888822222",
    "email": "helper@example.com",
    "accepted_at": "2026-03-31T12:05:00Z",
    "created_at": "2026-03-31T12:05:00Z"
  }
}
```

Common errors:

- `400` for missing/invalid token
- `404` if the request id does not exist
- `409` if the request is already processed, expired, or the contact already exists

### `DELETE /:id`

Deletes an accepted trusted contact for the authenticated owner.

Authentication:

- required

Request body:

- none

Success response: `200 OK`

```json
{
  "status": "deleted"
}
```

Common errors:

- `401` if the owner is not authenticated
- `404` if that contact does not belong to the authenticated user

## Current Schema Shape

The module uses two tables:

- `trusted_contact_requests`
  - pending or processed invitation state
- `trusted_contacts`
  - the accepted relationship used by SOS

That split matters because:

- a request can exist before a relationship is trusted
- requests can expire or be accepted later
- SOS should only fan out to accepted contacts

## How This Fits SOS

This module should feed SOS in two separate stages:

1. account setup stage
   - user builds an accepted trusted-contact network
2. SOS event stage
   - backend notifies those accepted contacts for a specific live session

That second stage should use a different per-session viewer token from the invitation token used here. The invitation token grants acceptance of the ongoing trusted-contact relationship. The SOS viewer token should grant access to one live emergency session only.
