# PRODUCT REQUIREMENTS DOCUMENT (PRD)

## SafeRoute


## 1. Executive Summary

SafeRoute is a **trust-first urban safety platform** that combines:

1. **Real-time emergency escalation (SOS system)**
2. **Anonymous, verifiable incident reporting**
3. **Risk-aware navigation with safety scoring**

The platform transforms fragmented, unreliable safety signals into a **continuous, trust-weighted safety network**, enabling users to both **react instantly in emergencies** and **proactively avoid unsafe areas**.

---

## 2. Product Vision

To build the **default safety layer for urban mobility**, where:

- Incidents are reported instantly and credibly
- Evidence is tamper-resistant and traceable
- Users navigate cities with real-time safety intelligence

---

## 3. Core Product Pillars

### 3.1 Instant Safety (SOS Layer)

Immediate escalation to trusted contacts with live data streaming.The platform transforms fragmented, unreliable safety signals into a **continuous, trust-weighted safety network**, enabling users to both **react instantly in emergencies** and **proactively avoid unsafe areas**.

### 3.2 Trustworthy Reporting (Evidence Layer)

Anonymous, verifiable, and tamper-resistant incident reporting.

### 3.3 Risk-Aware Navigation (Intelligence Layer)

Dynamic safety scoring and route optimization.

---

## 4. Target Users

### Primary Users

- Women commuting in urban environments
- Students and working professionals

### Secondary Users

- General public
- Families (trusted contacts)
- Authorities (read-only, future)

---

## 5. Key Features

---

## 5.1 Instant SOS & Live Escalation System

### Description

A low-friction emergency trigger that:

- Alerts trusted contacts instantly
- Streams live data (location, audio, images)
- Logs evidence in parallel

### Trigger Mechanisms

- Single-tap button
- Long-press gesture
- Hardware shortcut (future)

### System Behavior

### Immediate (≤2 seconds)

- Send:
    - Live location
    - Timestamp
    - Alert signal to trusted contacts

### Continuous (adaptive)

- Location tracking (continuous)
- Audio streaming (primary channel)
- Image capture (~15 sec intervals, adaptive)

### Network-Aware Degradation

- Strong network → audio + images
- Weak network → audio only
- Very weak → location only

### Evidence Handling

- Encrypted locally
- Uploaded incrementally
- Hash-chained for integrity

### Fallback

- SMS alert (location + SOS signal only)

---

## 5.2 Trusted Contacts Network

- Users can add/remove trusted contacts
- Contacts receive:
    - Live alert
    - Access to live session
- Deep link access to session view

---

## 5.3 Incident Reporting System

### Features

- Anonymous reporting
- Minimal input (<20 seconds)
- Auto-capture:
    - Location
    - Timestamp

### Evidence Types

- Image
- Audio
- Text

### Modes

- Online submission
- Offline storage + sync
- SMS fallback (minimal data)

---

## 5.4 Digital Evidence Locker

### Capabilities

- End-to-end encryption
- Secure storage (IPFS or equivalent)
- SHA-256 hashing
- Timestamping (server-signed; blockchain optional later)

### Features

- Immutable audit trail
- Evidence verification (hash match)
- Exportable evidence bundle

---

## 5.5 Complaint Tracking System

### Lifecycle

```sql
Submitted → Under Review → Escalated → Resolved
```

---

### Features

- Real-time status updates
- Timeline view
- Audit logs

---

## 5.6 Safety Scoring Engine

### Objective

Generate a **dynamic safety score (0–100)** for locations and routes.

### Formula

S = 100 − (w₁R + w₂H + w₃T + w₄(1 − C) + w₅U)

Where:

- R = Recent incident density
- H = Historical incident density
- T = Time-based risk factor
- C = Crowd proxy
- U = Trust-weighted user signals

---

### Input Signals

### R (Recent Incidents)

- Time-decayed weighting
- Proximity-based clustering

### H (Historical Data)

- Long-term patterns
- Lower weight than recent signals

### T (Time Risk)

- Heuristic-based (e.g., late night higher risk)

### C (Crowd Proxy)

- POI density
- Road classification
- Time heuristics

### U (User Signals)

- Real-time reports
- Weighted by trust score

---

### Output

- Safety score (0–100)
- Risk category:
    - Safe
    - Moderate
    - High Risk

---

### Explainability Layer

Each score must display:

- Contributing factors
- Recent incidents
- Time-based risk insights

---

## 5.7 Safe Navigation Layer

### Features

- Route comparison:
    - Fastest route
    - Safest route
- Safety overlays on map
- Risk-aware rerouting

### Routing Cost Function

Total Cost = α × Distance + β × Risk

---

## 5.8 Predictive Safety Alerts

### Features

- Pattern-based alerts (MVP)
- Time-based risk predictions
- Route warnings before travel

---

## 5.9 Offline Support

- Local storage (IndexedDB)
- Deferred sync
- Retry queue
- Offline evidence capture

---

## 6. Trust & Reputation System

---

## 6.1 Trust Score

Each user has a trust score (0–1) based on:

- Account age
- Report history
- Corroboration rate
- Device consistency
- Verification level

---

## 6.2 Aadhaar-Based Verification (Optional)

### Objective

Increase credibility without compromising privacy.

---

### Design Principles

- No storage of Aadhaar number
- No central identity database
- Fully optional

---

### Verification Flow

1. User completes Aadhaar-based verification (via compliant provider)
2. System generates:
    - Verification proof/token
3. Store:
    - Only verification status + signed proof

---

### Trust Impact

- Unverified user → lower trust weight
- Verified user → higher trust weight

---

### Future (Advanced)

- Zero-Knowledge Proof (ZKP) integration:
    - Prove “verified human” without exposing identity

---

### User Trust Strategy

- Transparent data usage
- Revocable verification
- Clear value proposition:
    - “Verified reports are prioritized”

---

## 7. System Architecture

---

### Frontend

- Web / PWA (React / Next.js)
- Mobile support (future)

---

### Backend

- Node.js / Go
- Modular monolith (MVP)

---

### Core Services

- Reporting Service
- Evidence Service
- Trust Engine
- Safety Scoring Engine
- Notification Service (real-time)

---

### Storage

- PostgreSQL + PostGIS (geospatial)
- Object storage / IPFS (evidence)

---

### Real-Time Layer

- WebSockets / WebRTC (for streaming)
- Push notifications

---

### Sync Layer

- Service workers
- Background sync manager

---

## 8. API Design (Sample)

### Submit Report

POST /reports

### Get Report Status

GET /reports/{id}

### Trigger SOS

POST /sos/start

### Stream Session

GET /sos/{session_id}

---

## 9. Security & Privacy

- End-to-end encryption (evidence)
- Minimal data retention
- Anonymous reporting
- Secure key management
- No raw Aadhaar storage

---

## 10. Analytics & Metrics

### Core Metrics

- SOS activation success rate
- Time to alert delivery
- Report completion time
- Safety feature engagement
- Retention (7-day, 30-day)

---

## 11. Rollout Plan

### Phase 1 (MVP)

- SOS system
- Reporting system
- Basic safety scoring
- Navigation overlay

### Phase 2

- Trust scoring
- Predictive alerts
- Reputation system

### Phase 3

- ZKP-based verification
- API ecosystem
- Authority integrations

---

## 12. Risks & Mitigations

| Risk | Mitigation |
| --- | --- |
| False SOS triggers | Cancel window, gesture validation |
| Fake reports | Trust-weighted scoring |
| Data sparsity | Heuristic bootstrapping |
| Aadhaar concerns | Optional + privacy-first design |
| Network issues | Adaptive streaming + SMS fallback |

---

## 13. Non-Goals (Important)

- Full law enforcement integration (MVP)
- Social networking features
- Perfect real-time accuracy
- Heavy blockchain dependency

---

## 14. Key Differentiators

- Real-time SOS + live streaming
- Trust-weighted safety scoring
- Privacy-preserving verification
- Offline-first architecture

---

## 15. Conclusion

SafeRoute is not just an app but a **real-time safety network** that integrates emergency response, verifiable reporting, and intelligent navigation. By prioritizing trust, speed, and usability, it aims to become the foundational layer for urban safety.

---
