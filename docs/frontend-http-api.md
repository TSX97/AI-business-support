# Frontend HTTP API

The frontend sends same-origin JSON requests. Nginx routes `/api/auth/*` to
`auth-service` and `/api/core/*` to `core-service`, stripping the `/api/auth`
or `/api/core` prefix before forwarding the request. For example, the frontend
request `/api/core/chats` reaches Core as `/chats`.

## Requests made by the frontend

| Method | Path | Request body | Purpose | Backend status |
|---|---|---|---|---|
| `POST` | `/api/auth/register` | `{ "name": string, "email": string, "password": string }` | Register an account | Not implemented yet |
| `POST` | `/api/auth/login` | `{ "email": string, "password": string }` | Log in | Not implemented yet |
| `POST` | `/api/core/chats` | `{ "content": string }` | Create a chat and submit its first user message | Planned; not implemented |
| `POST` | `/api/core/chats/{chat_id}/messages` | `{ "content": string }` | Submit a message to an existing chat | Planned; not implemented |

Successful chat requests are expected to return JSON in this shape:

```json
{
  "chat_id": 42,
  "reply": {
    "content": "Assistant response"
  }
}
```

The current Core service only exposes placeholder `/`, `/health`, and `POST
/api` handlers. It does not implement the planned chat endpoints. The chat
screen reports that the feature is not connected when the server returns 404;
it does not generate or display mock AI answers.

## Local chat history

The frontend supports creating and switching between multiple conversations.
Until Core provides an API for listing and retrieving chat history, the
frontend stores conversation titles, messages, the selected conversation, and
the associated Core chat IDs in this browser's `localStorage`. This is a
device-local prototype: it does not synchronize across browsers or devices,
and clearing site data removes the saved history. No authentication tokens are
stored as part of chat history.

## Authentication request details

Both authentication requests use the `Content-Type: application/json` header.
Registration sends `name`, `email`, and `password`; login sends `email` and
`password`. The frontend currently reads an optional `message` field from
error responses.

## Notes

- The frontend does not currently send requests to Redis, NATS, or PostgreSQL;
  those are backend infrastructure services.
- Requests are same-origin, so the browser uses the page's current scheme and
  host. In local development, use the Nginx origin to reach the `/api/*` routes.
- The planned chat response contract should be implemented by the Core service
  before the chat can return real assistant responses.
