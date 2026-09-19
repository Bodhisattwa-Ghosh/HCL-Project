# PulsePoll

PulsePoll is a live polling app built for the GUVI Developer Internship task. A signed-in creator asks a question, shares its unique link, and everyone watching sees vote totals update without refreshing.

![Stack](https://img.shields.io/badge/React-18-61dafb?logo=react&logoColor=white) ![Stack](https://img.shields.io/badge/Go-Gin-00ADD8?logo=go&logoColor=white) ![Stack](https://img.shields.io/badge/MongoDB-7-47A248?logo=mongodb&logoColor=white) ![Stack](https://img.shields.io/badge/Redis-7-DC382D?logo=redis&logoColor=white)

## What it does

- Sign up and log in before creating or managing a poll.
- Create a question with 2-8 distinct choices and an optional closing time.
- Share a public URL with voters.
- Allow one vote per browser per poll using a secure, HTTP-only anonymous-voter cookie plus a MongoDB unique index.
- Update option counts and percentages in real time via Redis Pub/Sub and Server-Sent Events - no page refresh.
- Let the creator close voting manually; polls also close automatically when their chosen deadline passes.

## Architecture

```text
Browser (React + EventSource)
          |
          | HTTP / SSE
          v
Go + Gin API ----------------------> MongoDB
     |                                  |
     | JWT auth, validation             | users, polls, durable vote audit trail
     v                                  |
Redis hash counters + Pub/Sub <---------+
     |
     +--> result event --> every connected viewer
```

MongoDB is the durable source of truth. Redis is not decorative: it holds the current per-option totals used on every live read and broadcasts result changes across API instances. If a Redis counter expires or is evicted, the API reconstructs it from MongoDB's durable vote records before returning results.

## Run locally with Docker (recommended)

Prerequisites: Docker Desktop with Compose enabled.

```powershell
Copy-Item .env.example .env
# Edit .env and replace JWT_SECRET with a long random value.
docker compose up --build
```

Open [http://localhost:8088](http://localhost:8088). The browser talks to the backend through Nginx at the same origin, which keeps cookies and SSE simple during local testing.

Stop the stack with:

```powershell
docker compose down
```

To also remove local database data, run `docker compose down -v` only if you deliberately want to erase it.

## Run without Docker

You need Go 1.22+, MongoDB 7+, Redis 7+, and Node 20+.

```powershell
# Terminal 1: backend
Copy-Item backend\.env.example backend\.env
# Edit backend\.env: use your MongoDB/Redis URLs and a 32+ character JWT_SECRET.
Set-Location backend
go mod download
go run ./cmd/api

# Terminal 2: frontend
Set-Location frontend
Copy-Item .env.example .env
npm install
npm run dev
```

The frontend is available at `http://localhost:5173`, and its default `.env.example` points to `http://localhost:8080`.

## Verify it

```powershell
# Frontend production build
Set-Location frontend
npm run build

# Backend formatting and unit tests
Set-Location ..\backend
go fmt ./...
go test ./...
```

For an end-to-end real-time check:

1. Create an account and poll in one browser window.
2. Open its shared `/p/<slug>` link in a second normal/incognito window.
3. Vote in the second window.
4. Confirm the first window's totals and bars change immediately without a reload.
5. Attempt another vote from the same second browser; it should be rejected.
6. Close the poll from its owner view and confirm voting becomes unavailable for viewers.

## Backend validation and security choices

- Server-side validation limits question/options, trims whitespace, rejects duplicate choices, validates close times, and never trusts client counts.
- Passwords are bcrypt hashes; the backend issues short-scope HS256 JWT sessions.
- Creator routes require a valid bearer token; only a poll owner can close a poll.
- MongoDB unique indexes enforce unique emails, slugs, and one durable vote per `{ pollId, voterId }`.
- The frontend never sends a total; Redis increments its own counter only after the durable MongoDB vote is accepted.
- CORS is allow-listed via `FRONTEND_ORIGIN`; security headers and an HTTP-only, SameSite voter cookie are enabled.

## Project layout

```text
frontend/              React + TypeScript + Vite client
backend/               Go + Gin API
  internal/domain/     data models and public response shape
  internal/repository/ MongoDB persistence and indexes
  internal/service/    authentication, validation, vote workflow
  internal/realtime/   Redis counters and Pub/Sub broker
docker-compose.yml     local full-stack MongoDB + Redis + API + web app
API_CONTRACT.md        frontend/backend request and response contract
```

## Deploy it

The included Dockerfiles and `docker-compose.yml` can run all four required services on any Docker-capable host. For a managed deployment, use the same environment variables with:

- a static frontend host (or the included Nginx frontend container),
- a Go/Docker web-service host for `backend`,
- MongoDB Atlas or another hosted MongoDB instance, and
- managed Redis (e.g. Redis Cloud/Upstash/provider Redis).

### Railway deployment

Create four Railway services: `frontend`, `backend`, MongoDB, and Redis. Only
make `frontend` public. Keep `VITE_API_URL` blank so the Nginx container proxies
`/api` requests and Server-Sent Events to the private backend, preserving the
same-origin voter cookie.

Set the frontend service variable below (assuming the backend service is named
`backend`):

```text
BACKEND_UPSTREAM=http://${{backend.RAILWAY_PRIVATE_DOMAIN}}:${{backend.PORT}}
```

Railway provides the frontend's `PORT` automatically; the included Nginx
template listens on that port. Leave the default
`BACKEND_UPSTREAM=http://backend:8080` for local Docker Compose.

Set the backend service's MongoDB and Redis connection variables from Railway's
private service references, generate a new 32+ character `JWT_SECRET` inside
Railway, and use the public frontend domain for `FRONTEND_ORIGIN`. Use
`COOKIE_SECURE=true` and `COOKIE_SAME_SITE=lax` in production.

Set these production variables on the backend:

```text
PORT=8080
MONGODB_URI=<production MongoDB URI>
MONGODB_DATABASE=pulsepoll
REDIS_URL=<production Redis URI>
JWT_SECRET=<long random secret, 32+ chars>
JWT_TTL_HOURS=168
FRONTEND_ORIGIN=https://<your-frontend-domain>
COOKIE_SECURE=true
COOKIE_SAME_SITE=lax
```

When hosting the frontend separately, build it with `VITE_API_URL=https://<your-api-domain>`. Ensure the API host permits SSE connections through its proxy/load balancer and that it forwards `Cache-Control: no-transform` / does not buffer the `/api/polls/:slug/events` response.

If the frontend and API use different sites (not just different ports), set `COOKIE_SAME_SITE=none` alongside `COOKIE_SECURE=true` so the anonymous one-vote cookie can be sent with the voting request. The included Nginx/Docker setup keeps frontend and API under one origin and therefore uses the safer `lax` default.

## Submission checklist

See [SUBMISSION_CHECKLIST.md](SUBMISSION_CHECKLIST.md) for the final GitHub, deployment, video, and email handoff. The task requires a real public deployment and a 3-5 minute video, not merely this local project.

## Key decisions

- **SSE rather than WebSockets:** viewers only receive server-to-client updates, so SSE is lighter, has automatic reconnect support, and maps cleanly to Redis Pub/Sub.
- **MongoDB plus Redis:** MongoDB persists identities and votes; Redis performs the live count reads and broadcasts. The split makes the real-time path fast while retaining a recoverable audit trail.
- **Anonymous voter cookie:** public voting stays frictionless while the backend still enforces one vote per browser. It is intentionally not presented as a perfect identity system; users can bypass it with another browser/device.

Before recording the required video, make sure you can explain these decisions, the Redis recovery path, the duplicate-vote index, and the SSE flow in your own words.
