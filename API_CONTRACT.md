# PulsePoll API Contract

The frontend talks to the Go service at `/api` (or `VITE_API_URL/api` in production).

## Authentication

- `POST /auth/signup` - `{ name, email, password }` -> `{ token, user }`
- `POST /auth/login` - `{ email, password }` -> `{ token, user }`
- `GET /auth/me` - bearer token required -> `{ user }`

## Polls

- `GET /polls` - bearer token required; returns the current user's polls.
- `POST /polls` - bearer token required; `{ question, options: string[], closesAt?: ISODate }`.
- `GET /polls/:slug` - public; returns poll metadata, options, and current vote totals.
- `POST /polls/:slug/votes` - public; `{ optionId }`; returns fresh results. The backend assigns a durable anonymous voter cookie and allows one vote per browser per poll.
- `GET /polls/:slug/events` - public Server-Sent Events stream. Each `results` event contains the same current poll result shape as `GET /polls/:slug`.

All polls use this public shape:

```json
{
  "id": "mongodb-id",
  "slug": "shareable-slug",
  "question": "Which feature should we ship next?",
  "options": [{ "id": "option-id", "label": "Realtime dashboards", "votes": 12 }],
  "totalVotes": 12,
  "createdAt": "2026-09-19T00:00:00Z",
  "closesAt": null,
  "isClosed": false,
  "isOwner": false
}
```

Errors use `{ "error": "clear, safe message" }`.
