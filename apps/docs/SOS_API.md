# SafeRoute SOS API

## What This Document Covers

This file is the concrete API reference for the currently implemented SOS backend.

It covers:

- the REST routes
- the WebSocket route
- exact input and output shapes
- how to test the WebSocket flow
- what "real-time" means in the current implementation

## Current Reality

The SOS WebSocket route is real-time in this specific sense:

- the reporter sends a location message immediately over WebSocket
- the backend validates it immediately
- the backend persists it immediately to `location_pings`
- the backend immediately returns an ack

What it does **not** do yet:

- broadcast the live location to watchers
- maintain a multi-user session hub
- use Redis to track active stream state

So right now it is real-time reporter-to-backend location ingest, not full real-time multi-party streaming yet.

## Base URLs

HTTP base:

```text
http://localhost:8080/api/v1
```

WebSocket base:

```text
ws://localhost:8080/api/v1
```

## Authentication

All currently implemented SOS routes require the authenticated reporter.

Auth transport:

- cookie-based session auth
- the same auth cookies used by `/auth/*` routes are used for SOS

For REST requests:

- Postman or the browser cookie jar should automatically send the auth cookie after login/register

For WebSocket requests:

- Postman may reuse the cookie jar automatically
- if it does not, add the auth cookie manually in the handshake headers

## Route Summary

### `POST /api/v1/sos/start`

Purpose:

- create a new SOS session for the authenticated user

Rules:

- authenticated reporter only
- only one active SOS session is allowed per user at a time

Request body:

```json
{}
```

Success: `201 Created`

```json
{
  "session": {
    "id": "sos-123",
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "active",
    "started_at": "2026-03-31T12:00:00Z",
    "ended_at": null
  }
}
```

Error cases:

- `401`

```json
{
  "error": "unauthorized"
}
```

- `409`

```json
{
  "error": "active sos session already exists"
}
```

### `GET /api/v1/sos/:id`

Purpose:

- fetch the current state of one SOS session

Rules:

- authenticated reporter only
- only the owner of the session can read it

Request body:

- none

Success: `200 OK`

```json
{
  "session": {
    "id": "sos-123",
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "active",
    "started_at": "2026-03-31T12:00:00Z",
    "ended_at": null
  }
}
```

Error cases:

- `401`

```json
{
  "error": "unauthorized"
}
```

- `403`

```json
{
  "error": "forbidden"
}
```

- `404`

```json
{
  "error": "sos session not found"
}
```

### `POST /api/v1/sos/:id/end`

Purpose:

- end an active SOS session

Rules:

- authenticated reporter only
- only the owner of the session can end it

Request body:

```json
{}
```

Success: `200 OK`

```json
{
  "session": {
    "id": "sos-123",
    "user_id": "550e8400-e29b-41d4-a716-446655440000",
    "status": "ended",
    "started_at": "2026-03-31T12:00:00Z",
    "ended_at": "2026-03-31T12:05:00Z"
  }
}
```

Error cases:

- `401`

```json
{
  "error": "unauthorized"
}
```

- `403`

```json
{
  "error": "forbidden"
}
```

- `404`

```json
{
  "error": "sos session not found"
}
```

- `409`

```json
{
  "error": "sos session already ended"
}
```

## WebSocket Route

### `WS /api/v1/sos/:id/stream`

Purpose:

- accept live location messages from the reporter

Rules:

- authenticated reporter only
- only the owner of the session can connect
- the session must still be active

### Handshake Requirements

The request must:

- use WebSocket upgrade
- include the valid auth cookie
- target a session that belongs to the authenticated user

If any of that fails, the upgrade or stream will be rejected.

### Client Message Format

The server currently expects JSON shaped like this:

```json
{
  "lat": 12.9716,
  "lng": 77.5946,
  "ts": "2026-03-31T12:00:00Z"
}
```

Field meaning:

- `lat`
  - latitude, must be between `-90` and `90`
- `lng`
  - longitude, must be between `-180` and `180`
- `ts`
  - optional RFC3339 timestamp
  - if omitted or zero, the backend uses current UTC time

### Server Ack

On success, the server replies with:

```json
{
  "status": "accepted",
  "recorded_at": "2026-03-31T12:00:00Z"
}
```

Meaning:

- the ping passed validation
- the session was still valid
- the backend persisted the point

### WebSocket Error Messages

The server currently writes a simple error payload before closing or continuing depending on the error:

```json
{
  "error": "sos session already ended"
}
```

Common error messages:

- `sos session not found`
- `forbidden`
- `sos session already ended`
- `latitude must be between -90 and 90`
- `longitude must be between -180 and 180`

Behavior:

- invalid lat/lng: error frame, connection stays open
- ended session / forbidden / not found: error frame, then connection closes

## How To Check The WebSocket Part

### Option 1: Postman

Use the updated Postman collection:

1. register or login first
2. run `Start SOS Session`
3. copy or reuse the generated `sos_session_id`
4. open `Stream SOS Location (WebSocket)`
5. connect
6. send:

```json
{
  "lat": 12.9716,
  "lng": 77.5946,
  "ts": "2026-03-31T12:00:00Z"
}
```

Expected response:

```json
{
  "status": "accepted",
  "recorded_at": "2026-03-31T12:00:00Z"
}
```

If Postman does not send the auth cookie automatically during the WebSocket handshake, add it manually as a `Cookie` header.

### Option 2: Verify Persistence In Postgres

After sending a WebSocket location message, verify the ping directly in Postgres:

```bash
docker exec -it saferoute-postgres psql -U postgres -d saferoute -c "
SELECT
  session_id,
  ST_Y(location) AS lat,
  ST_X(location) AS lng,
  recorded_at,
  created_at
FROM location_pings
ORDER BY created_at DESC
LIMIT 5;
"
```

If the WebSocket ack returned `accepted` and you see the point in `location_pings`, then the currently implemented real-time ingest path is working correctly.

## Example End-To-End Check

Minimal flow:

1. login
2. `POST /api/v1/sos/start`
3. connect `WS /api/v1/sos/:id/stream`
4. send a location JSON payload
5. receive `accepted` ack
6. optionally confirm the row in Postgres
7. `POST /api/v1/sos/:id/end`

## Current Limitations

The current SOS API still does not do these things:

- stream location to trusted contacts in real time
- expose a watcher subscription channel
- use Redis for active-session coordination
- attach evidence during the stream
- support audio or media streaming

That is why the answer to "is it having real time location?" is:

- yes for live ingest and persistence
- no for live multi-user broadcast yet
