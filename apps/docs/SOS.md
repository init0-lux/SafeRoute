# SafeRoute SOS Module

## Status

This document defines the staged implementation plan and API contract for the SOS module.

Current status:

- schema exists for `sos_sessions` and `location_pings`
- auth middleware exists
- REST SOS lifecycle is implemented
- owner-only WebSocket location ingest is implemented
- watcher broadcast, Redis-backed session state, and notifications are not implemented yet
- Redis and Postgres are running locally

This doc started as a pre-implementation contract and is now kept aligned with the currently completed SOS increment.

For concrete route-by-route request and response examples, see `apps/docs/SOS_API.md`.

## Product Goal

The SOS module is the emergency path in SafeRoute.

When a user triggers SOS, the system should:

- create a live emergency session
- mark that session as active
- support continuous live location updates
- allow the session to end cleanly
- later notify trusted contacts and let them follow the session

## MVP Scope

The MVP SOS slice we should build toward is:

- start an SOS session
- fetch session state
- end an SOS session
- stream reporter location updates over WebSocket
- later add watcher broadcast over WebSocket

For now, the backend design assumes the reporter is an authenticated user.

## What We Are Not Building First

The first SOS increments should not try to do everything at once.

These are explicitly deferred:

- SMS delivery
- push notifications
- trusted-contact deep-link access
- watcher broadcast over WebSocket
- audio streaming
- periodic image capture
- evidence upload during SOS
- cancel window / false-trigger UX logic
- advanced session analytics

Those belong in later stages after the base session lifecycle is stable.

## Actors

### Reporter

The authenticated SafeRoute user who triggers the SOS session.

Responsibilities:

- starts the session
- sends live location updates later
- ends the session

### Watcher

A trusted contact or authorized viewer who may later receive session access.

This role is planned, but not part of the first implementation increment.

### Backend

The Go backend is responsible for:

- creating the SOS session
- persisting state to Postgres
- later storing active-session state in Redis
- later broadcasting live location updates

## Data Model

The current schema already gives us the base SOS tables.

### `sos_sessions`

Represents the SOS lifecycle itself.

Important fields:

- `id`
  - stable session identifier
- `user_id`
  - owner of the session
- `status`
  - enum: `active` or `ended`
- `started_at`
  - when SOS began
- `ended_at`
  - when SOS ended

### `location_pings`

Represents time-ordered live location points within a session.

Important fields:

- `session_id`
  - parent SOS session
- `location`
  - PostGIS point
- `recorded_at`
  - client event time
- `created_at`
  - backend insertion time

## Desired Lifecycle

### Start

1. authenticated user calls `POST /api/v1/sos/start`
2. backend creates `sos_sessions` row with `status=active`
3. backend rejects the request if the user already has an active SOS session
4. backend returns the new session
5. later increments will also register session state in Redis and notify contacts

### Live Session

During the active session:

- the reporter can stream location updates over WebSocket in this increment
- backend will persist pings in `location_pings`
- watchers may later subscribe to updates in a future increment

### End

1. authenticated user calls `POST /api/v1/sos/:id/end`
2. backend verifies ownership and current state
3. backend sets `status=ended`
4. backend sets `ended_at`
5. later increments will also clear Redis state and close live streams

## API Contract For Incremental Build

These are the routes we should build in order.

### Stage 2 Routes

#### `POST /api/v1/sos/start`

Purpose:

- create a new SOS session for the current user

Authentication:

- required

Initial request body:

```json
{}
```

We can keep this body empty in the first implementation and add optional metadata later if needed.

Initial success response:

```json
{
  "session": {
    "id": "session-id",
    "user_id": "user-id",
    "status": "active",
    "started_at": "2026-03-31T12:00:00Z",
    "ended_at": null
  }
}
```

Conflict behavior:

- if the user already has an active SOS session, return `409`

#### `GET /api/v1/sos/:id`

Purpose:

- fetch current state for a session

Authentication:

- required for the first version

Initial authorization rule:

- only the session owner can read it

Initial success response:

```json
{
  "session": {
    "id": "session-id",
    "user_id": "user-id",
    "status": "active",
    "started_at": "2026-03-31T12:00:00Z",
    "ended_at": null
  }
}
```

#### `POST /api/v1/sos/:id/end`

Purpose:

- end an active SOS session

Authentication:

- required

Initial authorization rule:

- only the session owner can end it

Initial request body:

```json
{}
```

Initial success response:

```json
{
  "session": {
    "id": "session-id",
    "user_id": "user-id",
    "status": "ended",
    "started_at": "2026-03-31T12:00:00Z",
    "ended_at": "2026-03-31T12:05:00Z"
  }
}
```

#### `WS /api/v1/sos/:id/stream`

Purpose:

- allow the session owner to push live location updates

Authentication:

- required

Initial authorization rule:

- only the session owner can connect
- the session must still be active

Initial message shape:

```json
{
  "lat": 12.9716,
  "lng": 77.5946,
  "ts": "2026-03-31T12:00:00Z"
}
```

Initial server ack:

```json
{
  "status": "accepted",
  "recorded_at": "2026-03-31T12:00:00Z"
}
```

This increment is ingest-only:

- reporter sends location
- backend validates ownership and active status
- backend persists the ping
- backend replies with an acknowledgment
- no watcher broadcast yet

## Error Shape

For the first implementation, we should keep errors simple and consistent.

Suggested cases:

- `401 unauthorized`
  - missing or invalid auth cookie
- `403 forbidden`
  - authenticated, but not allowed to access that session
- `404 not found`
  - session does not exist
- `409 conflict`
  - invalid state transition, such as ending an already ended session or starting a second active session
- `500 internal server error`
  - unexpected backend failure

## Service Design

The SOS module should follow the same structure as auth:

- `internal/sos/repository.go`
- `internal/sos/service.go`
- `internal/sos/handler.go`

Recommended responsibilities:

### Repository

- create session
- get session by id
- get active session by user
- end session
- persist location ping

### Service

- start session
- enforce one active session per user
- validate ownership
- validate transitions
- end session
- persist location pings from the reporter stream
- later coordinate Redis and watcher streaming concerns

### Handler

- parse request
- call service
- return JSON response
- use auth middleware for user verification

## Redis Role

Redis should not be part of the first REST-only increment, but it is part of the planned design.

Later Redis usage:

- `session:{id}` active marker with TTL
- optional `last_seen`
- fast active-session checks for WebSocket connections
- crash-tolerant expiration safety

## WebSocket Role

WebSocket support is intentionally a later stage.

This approved increment includes only the reporter ingest path for:

- `WS /api/v1/sos/:id/stream`

Current responsibilities:

- reporter pushes live location messages
- backend persists pings
- backend returns an ack per accepted ping
- backend rejects inactive or unauthorized sessions

Still deferred:

- watcher broadcast
- multi-connection session hub
- Redis-backed active-session checks

## Auth Rules

For the first safe implementation:

- reporter must be authenticated
- all Stage 2 routes require `VerifyUser()`
- session ownership is enforced at the service layer

Trusted-contact access should come later with a deliberate design, likely using a signed watcher token or dedicated session access grant.

## Postman Plan

The Postman collection should be updated only after the REST routes are implemented.

When we do that, we should add:

- `POST /api/v1/sos/start`
- `GET /api/v1/sos/:id`
- `POST /api/v1/sos/:id/end`
- `WS /api/v1/sos/:id/stream`

Collection notes:

- requests should rely on cookie-based auth
- include one happy-path flow
- include one unauthorized example
- include one forbidden or already-ended example later if useful

## Recommended Slow Rollout

### Stage 1

Docs only.

Deliverables:

- this file

### Stage 2

REST SOS lifecycle plus owner-only WebSocket location ingest.

Deliverables:

- repository
- service
- handler
- route wiring
- websocket ingest path
- tests

### Stage 3

Postman support for the REST routes.

Deliverables:

- collection updates

### Stage 4

Redis-backed active-session state.

Deliverables:

- Redis integration
- session state helpers

### Stage 5

WebSocket live location stream.

Deliverables:

- watcher broadcast
- session hub / multi-connection coordination

### Stage 6

Trusted-contact notification hooks.

Deliverables:

- notification service contract
- alert stubs or delivery integration

## Open Decisions

These should be decided deliberately before or during later stages:

- should `GET /api/v1/sos/:id` remain owner-only or later support signed watcher access
- do we need a cancel grace period before alerts fire
- when should evidence start attaching directly to SOS sessions

## Recommendation

The safest next step is:

- implement Stage 2 only
- include only owner-side WebSocket location ingestion
- keep it owner-authenticated only
- enforce one active session per user
- defer watcher broadcast, Redis, and notifications until the session lifecycle is tested and stable
