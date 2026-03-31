# SafeRoute

> **A Trust-First Urban Safety Platform**

SafeRoute is a modern, trust-weighted safety layer for urban mobility. By combining real-time emergency escalation (SOS), verifiable incident reporting, and risk-aware navigation, SafeRoute transforms fragmented safety signals into a continuous safety network that helps users react instantly to emergencies and proactively avoid unsafe areas.

---

## 🌟 Product Vision & Capabilities

1. **Instant Safety (SOS Layer)**: Immediate escalation to trusted contacts with live streaming (location, audio, images) adaptive to network strength.
2. **Trustworthy Reporting (Evidence Layer)**: Anonymous, verifiable, and tamper-resistant incident reports built with encrypted endpoints.
3. **Risk-Aware Navigation (Intelligence Layer)**: Dynamic safety scoring leveraging nearby incident density, crowd proxies, temporal risk models, and user trust weightings.

---

## 🏗️ Architecture & Tech Stack 

This is a monorepo setup consisting of:

### Backend
- **Language**: Go
- **Framework**: Fiber (Fast HTTP web framework)
- **ORM**: GORM
- **Database**: PostgreSQL + **PostGIS** for high-performance geospatial queries (`geography` geometries).
- **Workers**: Custom background job manager for stale file cleanup and async operations.
- **Tooling**: Make, Docker Compose, Air (Live reload).

### Frontend
- **Framework**: React Native with **Expo** framework for rapid iOS/Android and Web deployment.

### Blockchain
- **Soroban**: Smart contract written in rust and deployed successfully on the testnet
- **Contract Address**: CAD455XRGDJRHQNBCPU2M33XABQHYP3KEBU4MK6DZQ3VEGTYGLCZSMMR
---

## 🚀 Current Implementation Status

The SafeRoute **backend core MVP** is feature-complete and test-covered.

### 📍 Reports Module
- **Endpoints**: Create, retrieve by ID, and list nearby reports.
- **Geospatial Processing**: Fully integrated PostGIS `POINT` system (`geography(POINT,4326)`).
- **Features**: Allowlisted incident categories, trust-weighted query sorting, strict pagination, and dynamic radius bounds filtering.

### 🛡️ Trust Module
- **Endpoints**: User trust profile, verification checks.
- **Engine**: Dynamic reputation calculator mapping account age, report corroborations, device consistency, and active verifications to a single dynamic variable.

### 🔒 Evidence Module
- **Endpoints**: Secure multipart uploads, metadata fetch, content download.
- **Integrity**: Enforced server-side SHA-256 evidence hashing upon stream reception. 
- **Validation**: Strict size limit and MIME-type allowlist enforcement via sniffed bytes. Local disk development storage, cleanly abstracted for S3/MinIO/IPFS.

### 🗺️ Safety Intelligence Module
- **Endpoints**: Geographic Point Safety Score, Route-based Safety Score.
- **Engine**: Fully explainable scoring layer output. Calculations ingest:
  - Trust-weighted historic + recent incident points.
  - Adjustable configurable Time-of-Day risk offsets.
- **Navigation**: Abstracted Provider Map Integration (e.g., Google Routes). Evaluates "corridor-based" route segments overlaying incident hotspots to find the safest (instead of merely fastest) routing.

---

## 🏃 Getting Started (Local Development)

### 1. Start the Backend API & Infrastructure
Ensure Docker is installed and running, then provision the stack locally:
```bash
cd apps/backend

# Initialize config
cp .env.example .env

# Stand up Postgres with PostGIS
make infra-up
make infra-init-postgis

# Build missing schema layouts
make schema-sync

# Run the API (defaults to localhost:3000)
make run
```

### 2. Verify with Postman
A pre-configured workspace collection is included at the repository root:  
`SafeRoute_Postman_Collection.json`. 

Run queries sequentially: `Health` → `Auth / Register` → `Reports` / `Safety` to populate the PostGIS tables and observe index interactions in real-time.

### 3. Start the Frontend App
```bash
cd apps/frontend

# Install dependencies
pnpm install

# Start the Expo bundler
pnpm expo start
```

---

## 🔒 Security & Privacy Principles
SafeRoute operates on a strictly "needs to know" basis:
- **End-to-end evidence chains**: Media retains backend hashes linking straight to reports ensuring integrity check guarantees.
- **No loose IDs**: Trust scoring functions operate using heuristics that preserve privacy, eschewing required physical IDs.
- **Bounded Geospatial Fetching**: Nearby queries execute via `ST_DWithin` ensuring tight bounded data constraints avoiding excessive scrapable exposures.
