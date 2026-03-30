# SafeRoute Backend Schema

## What This Document Is For

This file explains the backend schema in product terms:

- what each table represents in the SafeRoute product
- why each field exists
- how tables connect to core user flows

This is meant to be read alongside the GORM models, not instead of them.

## Source Of Truth

The schema is currently **GORM-driven**.

- Models:
  - `apps/backend/internal/auth/model.go`
  - `apps/backend/internal/trustedcontacts/model.go`
  - `apps/backend/internal/sos/model.go`
  - `apps/backend/internal/reports/model.go`
- Sync command:
  - `make -C apps/backend schema-sync`
  - or `go run ./cmd/schema-sync` from `apps/backend`

Manual SQL migrations are not the active schema source of truth right now.

## Product Flows Covered By This Schema

The current schema supports these product capabilities:

- account creation and login
- trusted contacts for SOS
- optional identity verification and trust weighting
- incident reporting, including anonymous reports
- evidence metadata and integrity tracking
- complaint status timeline
- SOS session lifecycle and live location streaming

## Database Extensions

The schema sync ensures these PostgreSQL extensions exist:

- `pgcrypto`
  - used for `gen_random_uuid()`
- `postgis`
  - used for location columns and spatial queries

## Database Enums

The schema also creates PostgreSQL enum types for status fields that should stay within a fixed workflow:

- `sos_session_status`
  - `active`
  - `ended`
- `complaint_event_status`
  - `submitted`
  - `under_review`
  - `escalated`
  - `resolved`
- `trusted_contact_request_status`
  - `pending`
  - `accepted`
  - `cancelled`
  - `expired`

## How The Tables Map To The Product

### Auth And Identity

- `users`
  - the core account record
- `trusted_contact_requests`
  - pending invitations before someone becomes a trusted contact
- `trusted_contacts`
  - accepted trusted contacts used during SOS notification fan-out
- `user_verifications`
  - optional verification records used to improve trust without storing raw identity documents

### Reporting And Evidence

- `reports`
  - the main user-submitted incident record
- `evidence`
  - metadata for uploaded image/audio/text evidence
- `complaint_events`
  - the status timeline for a report after it is submitted

### SOS

- `sos_sessions`
  - the emergency session itself
- `location_pings`
  - the live stream of location updates during an SOS session

## Tables

### `users`

Represents a SafeRoute account. This table is the anchor for identity, trust, reporting history, and SOS ownership.

| Field | Type | Product meaning |
| --- | --- | --- |
| `id` | `uuid` | Stable internal identifier for the user across auth, reports, SOS, and verification records. |
| `phone` | `text` | Primary identity field for the product. It matters because SOS fallback and trusted-contact alerts are phone-oriented. |
| `email` | `text` | Optional secondary contact and future recovery/notification channel. Not required for the core product loop. |
| `password_hash` | `text` | Stores the login credential securely. The raw password is never stored. |
| `trust_score` | `double precision` | Product-level credibility weight used when interpreting reports and user-generated safety signals. |
| `report_count` | `bigint` | Tracks how many reports a user has submitted, which helps trust and moderation logic. |
| `corroboration_count` | `bigint` | Tracks how many of the user's reports were supported by other evidence or signals. |
| `verified` | `boolean` | Quick flag for whether the user has passed optional identity verification. |
| `verified_at` | `timestamptz` | Records when verification became valid, useful for trust calculations and auditability. |
| `created_at` | `timestamptz` | Supports account-age-based trust logic and general auditing. |

Product relationships:

- one user owns many trusted-contact requests
- one user owns many trusted contacts
- one user can have many verification attempts/records
- one user can submit many reports
- one user can start many SOS sessions

### `trusted_contact_requests`

Represents the invitation or approval step before a person becomes a real trusted contact.

| Field | Type | Product meaning |
| --- | --- | --- |
| `id` | `uuid` | Stable identifier for the invitation itself. |
| `user_id` | `uuid` | The SafeRoute user who wants to add this trusted contact. |
| `name` | `text` | Human-readable label chosen by the requester. |
| `phone` | `text` | Primary delivery target for the invitation and later SOS fallback. |
| `email` | `text` | Optional invitation and notification channel. |
| `status` | `trusted_contact_request_status` | Tracks whether the invitation is pending, accepted, cancelled, or expired. |
| `invite_token_hash` | `text` | Hashed acceptance token used for secure invite links without storing the raw token. |
| `expires_at` | `timestamptz` | Time after which the invitation should no longer be accepted. |
| `responded_at` | `timestamptz` | When the recipient accepted or otherwise completed the request lifecycle. |
| `accepted_contact_id` | `uuid` | Optional link to the accepted `trusted_contacts` row after success. |
| `created_at` | `timestamptz` | Audit timestamp for when the invitation was created. |

Why this table exists separately:

- sending an invite is not the same as having a trusted contact
- the product needs to track pending and expired invitations
- accepted SOS recipients should only come from completed requests
- invitation links need token and expiry metadata that do not belong on the final contact record

### `trusted_contacts`

Represents the accepted personal safety network for a user. These records power the SOS alert flow once an invitation has been accepted.

| Field | Type | Product meaning |
| --- | --- | --- |
| `id` | `uuid` | Internal identifier for a trusted-contact record. |
| `user_id` | `uuid` | Connects the contact back to the user who added them. |
| `request_id` | `uuid` | Optional back-reference to the request that created this accepted relationship. |
| `name` | `text` | Human-readable label shown in the app when managing contacts and sending alerts. |
| `phone` | `text` | Main emergency delivery target for SMS-based SOS notifications. |
| `email` | `text` | Optional secondary channel for future alerting or evidence sharing. |
| `accepted_at` | `timestamptz` | When the contact actually became active for SOS use. |
| `created_at` | `timestamptz` | Audit trail for when this contact was added. |

Why the unique constraint matters:

- `user_id + phone` prevents the same user from adding the same contact repeatedly

Why this table still exists in addition to `trusted_contact_requests`:

- requests describe the invitation workflow
- `trusted_contacts` is the smaller, cleaner table the SOS system should query at runtime
- accepted contacts can remain stable even after the request has served its purpose

### `user_verifications`

Represents privacy-preserving verification records for optional identity validation. The product goal here is higher trust without storing sensitive raw documents like Aadhaar numbers.

| Field | Type | Product meaning |
| --- | --- | --- |
| `id` | `uuid` | Internal identifier for a verification record. |
| `user_id` | `uuid` | Ties the verification to the user whose trust level may change. |
| `provider` | `text` | Identifies the verification provider or system used. |
| `provider_ref` | `text` | Provider-side opaque reference so the app can correlate or revoke later without storing raw identity data. |
| `proof_hash` | `text` | Hash of the signed proof/token, useful for integrity checks and deduplication. |
| `verified_at` | `timestamptz` | When the verification succeeded. |
| `revoked_at` | `timestamptz` | Allows verification to be withdrawn later without deleting history. |
| `created_at` | `timestamptz` | Audit timestamp for when this record entered the system. |

Why this table exists separately from `users`:

- `users.verified` is the fast product flag
- `user_verifications` keeps the audit trail and provider metadata behind that flag

### `reports`

Represents a submitted incident report. This is the main input to the reporting system and later to safety scoring.

| Field | Type | Product meaning |
| --- | --- | --- |
| `id` | `uuid` | Stable identifier for a report, used by status pages, timelines, and evidence. |
| `user_id` | `uuid` | Optional author link. It is nullable so anonymous reporting still works. |
| `category` | `text` | High-level type of incident, used in UI filters, analytics, and risk scoring. |
| `description` | `text` | Freeform context supplied by the user. Useful for moderation and complaint review. |
| `location` | `geometry(Point,4326)` | Exact report location used for maps, hotspot detection, and safety scoring. |
| `address` | `text` | Human-readable location string for timeline/status views and authority workflows. |
| `occurred_at` | `timestamptz` | When the incident happened. This is the product time that matters for safety scoring. |
| `created_at` | `timestamptz` | When the report was received by the backend. Useful for ingestion/audit tracking. |
| `source` | `text` | How the report entered the system, such as `app`, `sms`, or `batch`. |

Why both `occurred_at` and `created_at` exist:

- `occurred_at` is the event time for safety analysis
- `created_at` is the system time for sync, auditing, and debugging

Why `user_id` is nullable:

- anonymous reporting is a product requirement

### `evidence`

Stores metadata for evidence files attached to either a report or an SOS session. This table does not store the raw file bytes; it stores the information needed to retrieve, verify, and audit them.

| Field | Type | Product meaning |
| --- | --- | --- |
| `id` | `uuid` | Internal identifier for the evidence record. |
| `report_id` | `uuid` | Optional link to a report when evidence is attached to incident reporting. |
| `session_id` | `uuid` | Optional link to an SOS session when evidence is captured during an emergency. |
| `storage_key` | `text` | Object-storage path/key used to locate the uploaded file. |
| `sha256` | `text` | Integrity hash used for verification and tamper detection. |
| `previous_hash` | `text` | Optional link for hash-chaining, useful for an evidence chain-of-custody style audit. |
| `media_type` | `text` | MIME-style description of the file so clients know how to display or process it. |
| `size_bytes` | `bigint` | File size for validation, quotas, and download metadata. |
| `signed_at` | `timestamptz` | Optional timestamp for server-side signing or attestation. |
| `created_at` | `timestamptz` | When the evidence metadata was recorded. |

Why both `report_id` and `session_id` are nullable:

- some evidence belongs to a report
- some evidence belongs directly to a live SOS session
- the product supports both flows

Why there is no `client_encrypted` field anymore:

- the product now treats client-side encryption as mandatory for every uploaded file
- keeping a boolean flag for something that is always true adds noise and can drift from the real security contract

### `complaint_events`

Represents the lifecycle timeline of a report after submission. This powers the status/timeline experience rather than storing only a single status field on `reports`.

| Field | Type | Product meaning |
| --- | --- | --- |
| `id` | `uuid` | Internal identifier for the timeline event. |
| `report_id` | `uuid` | The report this status event belongs to. |
| `status` | `complaint_event_status` | The lifecycle state at this point in time. This is a database enum so only supported workflow states can be stored. |
| `actor` | `text` | Who caused the transition, such as the system or an authority user. |
| `note` | `text` | Optional explanation or operator note for the transition. |
| `created_at` | `timestamptz` | Timestamp used to order the timeline. |

Typical product states:

- `submitted`
- `under_review`
- `escalated`
- `resolved`

Why this is a separate table:

- the product needs a timeline and audit trail
- a single `status` column on `reports` would lose historical transitions

### `sos_sessions`

Represents a live emergency session. This is the parent record for the SOS lifecycle and its live data stream.

| Field | Type | Product meaning |
| --- | --- | --- |
| `id` | `uuid` | Stable session identifier used in deep links, live watchers, and stream routing. |
| `user_id` | `uuid` | Optional owner of the SOS session. |
| `status` | `sos_session_status` | Current session state. This is a database enum so the session cannot drift into ad hoc string values. |
| `started_at` | `timestamptz` | When the SOS was triggered. |
| `ended_at` | `timestamptz` | When the SOS was explicitly closed, if it has ended. |

Why this table exists separately from `location_pings`:

- the session is the emergency event itself
- the pings are just the stream of updates within that event

### `location_pings`

Stores the time-ordered location stream for an SOS session.

| Field | Type | Product meaning |
| --- | --- | --- |
| `id` | `uuid` | Internal identifier for one location update. |
| `session_id` | `uuid` | Parent SOS session for this ping. |
| `location` | `geometry(Point,4326)` | Actual streamed user location for map display and replay. |
| `recorded_at` | `timestamptz` | Event time from the stream, useful for playback and ordering. |
| `created_at` | `timestamptz` | Backend insertion time, useful for audit/debugging. |

Why both `recorded_at` and `created_at` exist:

- `recorded_at` reflects when the device says the point was captured
- `created_at` reflects when the backend received and stored it

## Relationship Summary

Here is the main shape of the data model:

- `users -> trusted_contact_requests`
  - one-to-many
- `users -> trusted_contacts`
  - one-to-many
- `users -> user_verifications`
  - one-to-many
- `users -> reports`
  - one-to-many, optional on the report side
- `users -> sos_sessions`
  - one-to-many, optional on the session side
- `reports -> evidence`
  - one-to-many
- `reports -> complaint_events`
  - one-to-many
- `sos_sessions -> location_pings`
  - one-to-many
- `sos_sessions -> evidence`
  - one-to-many

## Indexes And Why They Matter

Important indexes created by the current schema sync:

- `users.phone`
  - fast lookup during login and account checks
- `trusted_contact_requests.invite_token_hash` unique
  - supports secure token-based invite acceptance
- `trusted_contact_requests (user_id, phone, status)`
  - supports duplicate-pending-request prevention
- `trusted_contact_requests.expires_at`
  - supports expiry cleanup and active-request checks
- `trusted_contacts (user_id, phone)` unique
  - prevents duplicate trusted contacts per user
- `user_verifications.user_id`
  - fast lookup of a user's verification history
- `user_verifications.proof_hash` unique
  - prevents duplicate proof records
- `user_verifications (provider, provider_ref)` unique
  - prevents duplicate provider-side verification entries
- `reports.user_id`
  - supports loading a user's report history
- `reports.location` GIST
  - powers nearby-incident and hotspot queries
- `reports.occurred_at DESC`
  - supports recent-incident and time-decay queries
- `evidence.report_id`
  - supports loading all evidence for a report
- `evidence.session_id`
  - supports loading all evidence for an SOS session
- `evidence.sha256`
  - supports integrity verification lookups
- `evidence.storage_key` unique
  - prevents duplicate object references
- `complaint_events (report_id, created_at DESC)`
  - supports timeline rendering for a report
- `sos_sessions.user_id`
  - supports loading a user's SOS history
- `location_pings (session_id, recorded_at DESC)`
  - supports session replay and latest-location queries

## Geospatial Notes

The schema uses `geometry(Point,4326)` for:

- `reports.location`
- `location_pings.location`

`4326` means WGS84 latitude/longitude coordinates, which is the standard GPS coordinate system used by phones and maps.

When writing query code, prefer constructing points like this:

```sql
ST_SetSRID(ST_MakePoint(lng, lat), 4326)
```

For distance queries in meters, cast to `geography` when appropriate:

```sql
ST_DWithin(location::geography, some_point::geography, radius_in_meters)
```

Why this matters to the product:

- nearby reports drive safety scoring
- location clustering powers hotspot detection
- SOS pings need map-accurate replay and distance logic

## Current Scope

This schema is based on:

- the backend implementation plan
- the current product requirements in `PRD.md`

It includes the concrete persistent entities already described there, but it does not add speculative Phase 3 tables that were not explicitly defined.
