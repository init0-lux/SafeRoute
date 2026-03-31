# SafeRoute

> **A Trust-First Urban Safety Platform**

SafeRoute is a modern, trust-weighted safety layer for urban mobility. By combining real-time emergency escalation (SOS), verifiable incident reporting, and risk-aware navigation, SafeRoute transforms fragmented safety signals into a continuous safety network that helps users react instantly to emergencies and proactively avoid unsafe areas.

---

## 🌟 Product Vision & Capabilities

1. **Instant Safety (SOS Layer)**: Immediate escalation to trusted contacts with live streaming (location, audio, images) adaptive to network strength.
2. **Trustworthy Reporting (Evidence Layer)**: Anonymous, verifiable, and tamper-resistant incident reports built with encrypted endpoints.
3. **Risk-Aware Navigation (Intelligence Layer)**: Dynamic safety scoring leveraging nearby incident density, crowd proxies, temporal risk models, and user trust weightings.
4. **Immutable Chain of Custody**: Cryptographically secure evidence timestamping on the Stellar Blockchain to guarantee admissibility without compromising privacy.

---

## 🏗️ Architecture & Tech Stack 

This is a monorepo setup consisting of:

### Backend
- **Language**: Go
- **Framework**: Fiber (Fast HTTP web framework)
- **ORM**: GORM
- **Database**: PostgreSQL + **PostGIS** for high-performance geospatial queries (`geography` geometries).
- **Workers**: Custom background job manager linking to the Stellar Blockchain.
- **Tooling**: Make, Docker Compose, Air (Live reload).

### Frontend
- **Framework**: React Native with **Expo** framework for rapid iOS/Android deployment.

### Blockchain (Soroban)
- **Engine**: Rust-based Smart Contract running on the Stellar network.
- **Status**: Live on Testnet.
- **Contract Address**: `CAD455XRGDJRHQNBCPU2M33XABQHYP3KEBU4MK6DZQ3VEGTYGLCZSMMR`

---

## 🚀 Current Implementation Status

### 📍 Reports Module
- **Endpoints**: Create, retrieve by ID, and list nearby reports.
- **Geospatial Processing**: PostGIS `POINT` bounds filtering (`geography(POINT,4326)`).

### 🛡️ Trust Module
- **Engine**: Dynamic reputation calculator mapping account age, report corroborations, and device consistency to a single dynamic variable.

### 🔒 Evidence Module & Blockchain Relayer 
- **Integrity Integration**: Upon stream reception, the backend calculates a server-side `SHA-256` hash.
- **Zero-Web3 UX**: The backend utilizes an automated **Relayer Wallet** to pay gas fees and submit the `<ReportID>` and `<EvidenceHash>` to the Soroban Smart Contract. Users do not need wallets or crypto.
- **Local Indexer**: A Go worker polls the Stellar Blockchain for confirmation events and updates the DB to show a "Verified on Chain" status natively in the app.

### 🗺️ Safety Intelligence Module
- **Engine**: Fully explainable scoring layer output utilizing trust-weighted historic points and Time-of-Day offsets.
- **Navigation**: Abstracted Provider Map Integration (e.g., Google Routes) overlaid with our incident hotspots.

---

## ⚖️ Compliance & Data Security (India)

SafeRoute operates on a strictly "needs to know" basis and complies heavily with Indian legal standards:
- **BNS/BSA 2023 (Evidentiary Standards)**: End-to-end evidence chains. Media files retain their on-chain hashes to act as a cryptographically secure digital receipt, ensuring they are admissible in court.
- **DPDP Act (Right to be Forgotten)**: Private media objects are kept entirely off-chain in localized cloud buckets. Since *only* the `SHA-256` hash lives on the public ledger, deleting the cloud file immediately renders the immutable on-chain hash an undecipherable orphan, fully satisfying digital privacy laws without breaking the blockchain.

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

### 2. Blockchain Setup (Deterministic Sandbox)
You do not need to install Rust or the Stellar CLI locally. We use Dockerized make targets for full determinism.

```bash
cd apps/backend

# Configures the testnet and safely generates + funds your local relayer keypair
make soroban-setup

# Compiles the Rust smart contract into a WASM binary
make soroban-build

# Deploys the built WASM array to the Testnet
make soroban-deploy
```

### 3. Start the Frontend App
```bash
cd apps/frontend

# Install dependencies
pnpm install

# Start the Expo bundler
pnpm expo start
```
