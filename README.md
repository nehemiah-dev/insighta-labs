# Insighta Labs

A REST API that accepts a name and returns a demographic profile — predicted gender, age, age group, and nationality — by calling three external inference APIs concurrently and persisting the result.

---

## Problem it solves

Manually calling Genderize, Agify, and Nationalize separately, stitching the results together, and figuring out what to do with the data is repetitive work. Insighta Labs wraps all three into a single `POST /api/profiles` call, applies classification rules (age grouping, top-country selection, confidence scoring), deduplicates by name, and stores the result so subsequent requests for the same name are instant — no external calls needed.

---

## Tech choices

**Go + `net/http`** — chosen over a framework deliberately. Building on the standard library means understanding exactly what happens at each layer: TCP connection pooling, request/response parsing, middleware composition, context propagation.

**`sync.WaitGroup` for concurrency** — the three external API calls (Genderize, Agify, Nationalize) run concurrently in separate goroutines, each writing to its own field in a shared results struct. Total latency is the slowest of the three, not the sum of all three.

**`pgx` (no ORM)** — direct SQL via pgx's connection pool. Parameterized queries (`$1`, `$2`) for injection safety. `RETURNING` on insert to avoid a second round-trip for generated fields. `LOWER()` in SQL for case-insensitive filtering rather than post-processing in Go.

**Custom error types** — `UpstreamFailure{Service: "Genderize"}`, `ErrNotFound`, `ErrAlreadyExists` — so the handler layer can use `errors.Is`/`errors.As` to make precise routing decisions (502 vs 404 vs 200-with-message) without string matching or leaking implementation details across layers.

**Layered architecture** — `clients` (external API wrappers) → `store` (Postgres queries) → `services` (orchestration) → `handlers` (HTTP). Each layer depends only on the one below it. Adding a cache layer later means changing only `services` — handlers and store are untouched.

---

## Results / impact

- Single endpoint replaces three separate API calls for the consumer
- Duplicate name requests skip all external calls entirely — served directly from Database
- Concurrent fetch keeps p99 latency close to the slowest single upstream call rather than their sum
- Structured error responses (`{ "status": "error", "message": "..." }`) with correct HTTP status codes (400/404/422/502/503) across every failure mode
- CORS-enabled — callable from any browser origin

---

## API

### POST /api/profiles
Create a profile for a name. Returns the existing profile if the name has been seen before.

**Request**
```json
{ "name": "ella" }
```

**Response 201**
```json
{
  "status": "success",
  "data": {
    "id": "4f97ba3f-7af4-4541-9d13-fbc71a5349a4",
    "name": "ella",
    "gender": "female",
    "gender_probability": 0.99,
    "sample_size": 97632,
    "age": 47,
    "age_group": "adult",
    "country_id": "GB",
    "country_probability": 0.107,
    "created_at": "2026-06-22T00:47:22Z"
  }
}
```

**Response 200 (already exists)**
```json
{
  "status": "success",
  "message": "Profile already exists",
  "data": { "...existing profile..." }
}
```

### GET /api/profiles
List all profiles. Supports case-insensitive filters.

```
GET /api/profiles?gender=female&country_id=GB&age_group=adult
```

### GET /api/profiles/{id}
Fetch a single profile by UUID.

### DELETE /api/profiles/{id}
Delete a profile. Returns `204 No Content`.

---

## Error responses

All errors follow a consistent shape:
```json
{ "status": "error", "message": "description of what went wrong" }
```

| Status | Meaning |
|--------|---------|
| 400 | Missing or empty name |
| 422 | Name contains invalid characters |
| 404 | Profile not found |
| 502 | External API returned unusable data |
| 503 | External API unreachable |

---

## How to run

**Prerequisites**
- Go 1.22+
- PostgreSQL running locally

**1. Clone and install dependencies**
```bash
git clone https://github.com/nehemiah-dev/insighta-labs
cd insighta-labs
go mod tidy
```

**2. Create the database**
```bash
createdb insighta
psql $DATABASE_URL -f src/store/schema.sql
```

**3. Set environment variables**
```bash
export DATABASE_URL="postgres://youruser:yourpass@localhost:5432/insighta?sslmode=disable"
export PORT=8080  # optional, defaults to 8080
```

**4. Run**
```bash
go run ./src
```

**5. Try it**
```bash
curl -X POST localhost:8080/api/profiles -d '{"name":"ella"}'
curl localhost:8080/api/profiles
curl localhost:8080/api/profiles/{id}
curl -X DELETE localhost:8080/api/profiles/{id}
```