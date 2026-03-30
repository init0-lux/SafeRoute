# Architecture

## Overview

SafeRoute is a **real-time safety system** composed of:

- A **mobile client (React Native)** for SOS, reporting, and navigation
- A **Go backend (Fiber)** for real-time coordination and APIs
- A **geospatial data layer (PostGIS)** for safety intelligence
- A **streaming + notification layer** for live SOS escalation
- A **storage pipeline** for secure, tamper-evident evidence

The system is designed to be:

- **Low-latency** (SOS-critical paths)
- **Offline-tolerant**
- **Horizontally scalable**
- **Trust-aware**

---

## High-Level Architecture

```sql
[ Mobile Client (React Native) ]
|
| HTTPS / WebSocket
↓
[ Go Backend (Fiber) ]
|
├── PostgreSQL + PostGIS (core data)
├── Redis (sessions, rate limiting)
├── Object Storage (S3/R2)
├── IPFS (anchoring, later phase)
|
├── FCM (push notifications)
└── Twilio (SMS fallback)
```

---

## Core Components

### 1. Client Layer

**Tech:** React Native (Expo)

**Responsibilities:**

- SOS trigger + background execution
- Location tracking (high frequency during SOS)
- Audio chunk recording
- Periodic image capture
- Offline storage (queued reports)
- WebSocket subscription for live sessions

---

### 2. Backend (Go + Fiber)

**Tech:** Go 1.22+, Fiber v2

**Responsibilities:**

- API handling (REST + WebSocket)
- SOS session orchestration
- Real-time fan-out to trusted contacts
- Report ingestion and lifecycle tracking
- Trust scoring + validation hooks
- Evidence processing pipeline

---

### 3. Data Layer

### PostgreSQL 15 + PostGIS

Primary database for:

- Users
- Reports
- SOS sessions
- Geospatial queries (heatmaps, clustering)

### Redis 7

Used for:

- Active SOS session state
- WebSocket pub/sub fan-out
- Rate limiting
- Retry / deferred queues

---

### 4. Real-Time Layer

### WebSockets (Fiber)

- Live SOS session updates:
    - location stream
    - audio chunk relay
    - image events

### Firebase Cloud Messaging

- Push notifications for:
    - SOS alerts
    - report updates

### Twilio (SMS Fallback)

- Emergency fallback when push fails
- Sends minimal alert payload (location + link)

---

### 5. Evidence Pipeline

### Phase 1 (MVP)

- Evidence stored in S3-compatible storage (MinIO/R2)
- SHA-256 hash generated per file
- Metadata stored in Postgres

### Phase 2

- Evidence encryption (client-side)

### Phase 3

- IPFS (Kubo):
    - Upload finalized evidence
    - Store CID in DB
    - Maintain hash integrity linkage

---

### 6. Safety Scoring Engine

Runs as part of backend services.

**Inputs:**

- Recent reports (time-decayed)
- Historical density
- Time-of-day heuristics
- Crowd proxies (POI density, road types)
- Trust-weighted user signals

**Outputs:**

- Safety score (0–100)
- Risk classification

---

### 7. Trust & Identity Layer

**Mechanisms:**

- JWT-based authentication
- Phone OTP (via Firebase Auth or similar)
- Optional verification flags (future KYC layer)

**Trust Score Factors:**

- Report history
- Corroboration
- Device consistency
- Verification level

---

## Key Flows

---

### 1. SOS Flow

```sql
User triggers SOS
↓
Backend creates session (Redis + DB)
↓
Push notification sent (FCM)
↓
WebSocket channel opened
↓
Client streams:
- location (continuous)
- audio chunks
- images (interval)
↓
Backend relays to trusted contacts
```

---

### 2. Incident Reporting Flow

```sql
User submits report
↓
Validated (Go validator)
↓
Stored in Postgres (with geospatial index)
↓
Evidence uploaded (S3/R2)
↓
Hash generated + stored
↓
Queued for scoring engine
```

---

### 3. Evidence Anchoring Flow (Phase 3)

```sql
Evidence finalized
↓
Upload to IPFS (Kubo node)
↓
Receive CID
↓
Store CID + hash in Postgres
```

---

## API Design Principles

- REST for standard operations
- WebSockets for real-time streams
- Idempotent endpoints for retries
- JWT-secured routes
- Rate-limited critical endpoints (SOS, reporting)

---

## Concurrency Model

- Goroutines for:
    - SOS session handling
    - WebSocket fan-out
    - background jobs (IPFS, scoring)
- Redis Pub/Sub for:
    - multi-instance broadcast

---

## Deployment Architecture

### Development

- Docker Compose:
    - Postgres + PostGIS
    - Redis
    - MinIO

### Production (Initial)

- Backend: single instance (scalable)
- Frontend: Vercel
- Storage: Cloudflare R2

### Scaling Path

- Horizontal scaling (multiple Go instances)
- Redis for shared state
- Load balancer in front of backend
- Separate workers for:
    - scoring
    - evidence processing

---

## Security Considerations

- JWT authentication
- Input validation (validator)
- Rate limiting (Redis)
- Encrypted evidence (future phase)
- No sensitive identity storage (KYC optional)

---

## Observability

- Structured logs (slog)
- Request tracing (future)
- Metrics (Prometheus-ready)
- Error tracking hooks

---

## Non-Goals (Current Architecture)

- Microservices split (modular monolith preferred)
- Full blockchain dependency
- Real-time ML pipelines
- Native WebRTC infra (future upgrade)

---

## Summary

This architecture prioritizes:

- **Reliability under stress (SOS)**
- **Low-latency communication**
- **Geospatial intelligence**
- **Incremental scalability**

The system is intentionally designed as a **modular monolith**, allowing rapid iteration while maintaining a clear path to distributed scaling.

---
