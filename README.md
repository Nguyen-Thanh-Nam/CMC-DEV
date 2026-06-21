# EASM (External Attack Surface Management) Homework Project

This repository contains the completed homework sessions 3-7 for the EASM application, implementing database migration to MySQL, asynchronous scanning, advanced validation, unit testing, premium frontend dashboard, Docker containerization, and automated CI/CD security workflows.

---

## 🏗️ Architecture & Features

This project is built following **Clean Architecture** patterns:
- **Handlers (HTTP Layer)**: Handles path parsing, response marshalling, and error mapping.
- **Service Layer**: Manages business logic, caching, audit logs, and coordinates scanning tasks.
- **Validator**: Advanced parameter, IP, domain, pagination, and SQL injection query parameter validators.
- **Storage Layer**: Dynamic persistence layer implementing full CRUD and pagination query builders.
- **Scanners**: Interface-driven scanner engines running asynchronously on thread-safe workers.

### Completed Homeworks (Bài 1 to Bài 7)

#### 1. Database Migration to MySQL (Bài 1)
- Auto-running migrations via SQL script (`migrations/001_create_tables.up.sql`).
- Persistent storage for assets, scan jobs, and results in MySQL.
- Automatic connection retries and graceful error exits on startup.

#### 2. Expanded Scan API (Bài 2)
- Interface-driven scanners under `internal/scanner/`:
  - `dns`: DNS records (A, AAAA, MX, NS, TXT).
  - `whois`: Registry information lookup.
  - `ip`: IP Geolocation and autonomous system (ASN) lookup.
  - `port`: Active port scanning (TCP connection probe) with **strict safety checks disallowing public IP scans**.
  - `ssl`: SSL/TLS cert extraction, cipher suite verification, and grading.
  - `tech`: Web technology fingerprinting from meta tags and headers.
- New endpoints for type-specific assets results:
  - `GET /assets/{id}/dns`
  - `GET /assets/{id}/whois`
  - `GET /assets/{id}/subdomains`

#### 3. Unit Tests & Benchmarking (Bài 3)
- Model, scanner, service, memory storage, and validator tests.
- Benchmarks for validator and model performance.
- Automated tests passing 100%.

#### 4. CORS & Premium Frontend (Bài 4)
- Dark glassmorphic premium UI dashboard at `frontend/`.
- CORS middleware in backend allowing cross-origin requests.

#### 5. CI/CD & Security Scans (Bài 5 - Bonus)
- GitHub Actions workflow (`.github/workflows/ci.yml`) compiling, testing, and running `gosec` security scanning.

#### 6. Docker Compose Deployment (Bài 6 - Bonus)
- Production multi-stage `Dockerfile` and `docker-compose.yml` orchestrating MySQL, REST API, and Nginx frontend in a single stack.

#### 7. Advanced EASM Features (Bài 7 - Bonus)
- **Asset Tags**: Comma-separated tags, search, and filtering in the UI/database.
- **Scheduled Scans**: Auto-passive scanning triggered for active assets periodically.

---

## 🚀 Setup & Execution Guide

### 1. Prerequisites
- Docker & Docker Compose
- Go 1.23+ (optional, for local development)

### 2. Environment Variables (.env)
Create a `.env` file in the `backend/` directory or run with default environment settings.
```env
DB_HOST=db
DB_PORT=3306
DB_USER=root
DB_PASSWORD=password
DB_NAME=mini_asm
SERVER_PORT=8080
USE_DB=true
```

### 3. Deploying via Docker Compose
To build and spin up the complete application stack (MySQL Database, Go API Backend, and Nginx Frontend):
```bash
docker-compose up --build -d
```
- **Go API Backend**: accessible at `http://localhost:8080`
- **Dashboard UI**: accessible at `http://localhost:3000`

### 4. Running Unit Tests locally
Navigate to the `backend/` directory:
```bash
cd backend
go test -v ./...
```

To view code coverage:
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

---

## 📡 API Testing Examples

### Create Domain Asset:
```bash
curl -X POST http://localhost:8080/assets \
  -H "Content-Type: application/json" \
  -d '{"name":"google.com","type":"domain"}'
```

### Trigger Scan:
```bash
curl -X POST http://localhost:8080/assets/{asset_id}/scan \
  -H "Content-Type: application/json" \
  -d '{"scan_type":"dns"}'
```

### Get Scan Results:
```bash
curl http://localhost:8080/scan-jobs/{job_id}/results
```
