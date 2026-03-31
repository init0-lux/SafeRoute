# Current SOS Flow Implementation

This document describes the end-to-end SOS flow as implemented in the SafeRoute backend, specifically using the **Expo Push Notification** system for alerts.

## 1. Flow Initiation (The Reporter)

*   **Endpoint**: `POST /api/v1/sos/start` (Authenticated)
*   **Logic**:
    1.  Validates that the user does not already have an active session.
    2.  Creates a new `SOSSession` record in the `sos_sessions` table with status `active`.
    3.  Triggers the **Notification Fanout** (see below) in the background.
*   **Response**: Returns the `session_id`, `status`, and `started_at` timestamp.

## 2. Notification Fanout (Alerting Contacts)

As soon as a session starts, the backend executes the following:
*   **Lookup**: Queries the `trusted_contacts` table for all contacts added by the reporter.
*   **Push Token Matching**: Joins the `trusted_contacts` with the `users` table based on the phone number to retrieve the **Expo Push Tokens** (`expo_push_token`) of those contacts.
*   **Viewer Grants**: Generates a unique `SOSViewerGrant` and a **Viewer Token** for each contact. This allows them to view the live stream without logging in as the reporter.
*   **Delivery**: Uses the `ExpoPushSender` to send a high-priority push notification to each contact's device.
    *   **Payload**: Includes `SOSSessionID` and the unique `ViewerToken` to enable deep-linking in the app.

## 3. Real-time Location Tracking (The Reporter)

*   **Connection**: The Reporter's app opens a **WebSocket** connection to:
    `ws://<host>/api/v1/sos/:session_id/stream`
*   **Action**: The app sends location pings (JSON: `{"lat": ..., "lng": ..., "ts": ...}`) via the WebSocket.
*   **Persistence**: Every ping is saved to the `location_pings` table for history and evidence.
*   **Broadcast**: The `SOS.Service` immediately broadcasts these pings to any active viewers.

## 4. Live Viewing (The Trusted Contact)

*   **Alert**: The contact receives a push notification on their mobile device: *"SafeRoute SOS Alert! [Name] triggered SOS..."*
*   **Action**: Tapping the notification opens the app and navigates to the viewer screen using the `session_id` and `token` from the notification payload.
*   **Streaming**: The app connects to the **Server-Sent Events (SSE)** endpoint:
    `GET /api/v1/sos/viewer/stream?token=<viewer_token>`
*   **Updates**: 
    1.  The backend validates the `viewer_token`.
    2.  The app receives a "ready" event with session details.
    3.  As the Reporter sends WebSocket pings, the backend pushes them to the Contact via the SSE stream in real-time.

## 5. Termination

*   **Endpoint**: `POST /api/v1/sos/:id/end` (Authenticated)
*   **Action**: 
    *   The session status is updated to `ended`.
    *   The `ended_at` timestamp is set.
    *   The live stream (WebSocket and SSE) is closed for all participants.

---

## Technical Stack & Configuration

*   **Backend**: Go (Fiber framework)
*   **Database**: PostgreSQL + PostGIS (Geospatial storage)
*   **Real-time**: WebSockets (Reporter) and SSE (Viewers)
*   **Notifications**: Expo Push API (`exp.host`)
*   **Auth**: Handled via HTTP-only session cookies.

## API Endpoints Summary

| Method | Path | Description |
|---|---|---|
| `POST` | `/api/v1/auth/push-token` | Register a device's Expo Push Token |
| `POST` | `/api/v1/sos/start` | Start SOS and notify contacts |
| `GET` | `/api/v1/sos/:id` | Get session status |
| `POST` | `/api/v1/sos/:id/end` | Stop SOS session |
| `GET` | `/api/v1/sos/:id/stream` | WebSocket for Reporter pings |
| `GET` | `/api/v1/sos/viewer/stream`| SSE for Viewer updates (uses `token` query) |
| `POST` | `/api/v1/sos/:id/viewers`| Manually create a viewer grant |
