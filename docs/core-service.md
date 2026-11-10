# Core Service API: MVP-ready contract

This document is the tightened MVP contract for the Go Core Service. It keeps the original business flow, but makes the fields, IDs, auth assumptions and response shapes much more concrete.

## Core principles

1. Use `uuid` as the canonical ID for chats, messages and businesses.
2. Keep `chat_name` as a human-friendly display name, but never as the primary lookup key in production logic.
3. All private actions require a valid JWT or session cookie.
4. `user_id` can be present in the request body for MVP simplicity, but it must be checked against the authenticated session.
5. The Core Service persists data in PostgreSQL and pushes AI work to NATS.

## Auth/session model

All endpoints below require:

- `Authorization: Bearer <jwt>`
- or a valid authenticated session cookie

The service must validate the session before any chat/business operation.

For MVP, the server rule is:

- `user_id` from the request body must match the authenticated user
- otherwise return `401` or `403`

This keeps the API easy to test and still safe enough for the first version.

## Field standards

- `user_id`: UUID string
- `chat_id`: UUID string
- `business_id`: UUID string
- `message_id`: UUID string
- `chat_name`: 1-80 characters, trimmed, unique per user
- `business_name`: 1-120 characters
- `message_content`: 1-20000 characters
- `description`: optional string, max length 500/1000 depending on entity
- `metadata`: JSON object, optional
- `business_data`: JSON object, optional

## Response conventions

- `Content-Type: application/json`
- success objects follow:

```json
{
  "success": true,
  "data": {
    "...": "..."
  }
}
```

- validation or permission failures follow:

```json
{
  "success": false,
  "error": {
    "code": "invalid_request",
    "message": "chat_id is required when chat_name is missing",
    "details": {}
  }
}
```

## Endpoint summary

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/chat` | Create a new chat |
| `GET` | `/chat` | Read a chat and messages with pagination |
| `PUT` | `/chat` | Rename a chat |
| `DELETE` | `/chat` | Delete a chat |
| `POST` | `/chat/message` | Save a user message and queue AI work |
| `POST` | `/business` | Create a business |
| `POST` | `/business/member` | Add a user to a business |
| `DELETE` | `/business/member` | Remove a user from a business |

## Chat API

### `POST /chat`

Create a chat for the authenticated user.

Request:

```json
{
  "user_id": "11111111-1111-4111-8111-111111111111",
  "chat_name": "sales-ops",
  "description": "Q4 sales support workspace",
  "metadata": {
    "company_id": "22222222-2222-4222-8222-222222222222",
    "context": "support"
  }
}
```

Response `201`:

```json
{
  "success": true,
  "data": {
    "chat_id": "33333333-3333-4333-8333-333333333333",
    "user_id": "11111111-1111-4111-8111-111111111111",
    "chat_name": "sales-ops",
    "description": "Q4 sales support workspace",
    "created_at": "2026-11-10T12:00:00Z",
    "updated_at": "2026-11-10T12:00:00Z"
  }
}
```

### `GET /chat`

Load a chat and its messages with pagination.

Query params:

- `user_id` required
- `chat_id` optional if `chat_name` is provided
- `chat_name` optional if `chat_id` is provided
- `limit` default 25, max 100
- `cursor` optional opaque pagination cursor

Example request:

```http
GET /chat?user_id=11111111-1111-4111-8111-111111111111&chat_id=33333333-3333-4333-8333-333333333333&limit=25
```

Response `200`:

```json
{
  "success": true,
  "data": {
    "chat": {
      "chat_id": "33333333-3333-4333-8333-333333333333",
      "user_id": "11111111-1111-4111-8111-111111111111",
      "chat_name": "sales-ops",
      "description": "Q4 sales support workspace",
      "created_at": "2026-11-10T12:00:00Z",
      "updated_at": "2026-11-10T12:00:00Z"
    },
    "messages": [
      {
        "message_id": "44444444-4444-4444-8444-444444444444",
        "chat_id": "33333333-3333-4333-8333-333333333333",
        "user_id": "11111111-1111-4111-8111-111111111111",
        "role": "user",
        "content": "We lost 8% of customers this quarter.",
        "created_at": "2026-11-10T12:01:00Z"
      }
    ],
    "next_cursor": null
  }
}
```

### `PUT /chat`

Rename a chat.

Request:

```json
{
  "user_id": "11111111-1111-4111-8111-111111111111",
  "chat_id": "33333333-3333-4333-8333-333333333333",
  "new_chat_name": "sales-ops-q4",
  "description": "Updated Q4 support workspace"
}
```

Response `200`:

```json
{
  "success": true,
  "message": "chat renamed successfully",
  "data": {
    "chat_id": "33333333-3333-4333-8333-333333333333",
    "chat_name": "sales-ops-q4"
  }
}
```

### `DELETE /chat`

Delete a chat and its messages.

Request:

```json
{
  "user_id": "11111111-1111-4111-8111-111111111111",
  "chat_id": "33333333-3333-4333-8333-333333333333"
}
```

Response `200`:

```json
{
  "success": true,
  "message": "chat deleted successfully",
  "data": {
    "chat_id": "33333333-3333-4333-8333-333333333333"
  }
}
```

## Chat message API

### `POST /chat/message`

Persist a message and queue it for AI processing.

Request:

```json
{
  "user_id": "11111111-1111-4111-8111-111111111111",
  "chat_id": "33333333-3333-4333-8333-333333333333",
  "message_content": "We lost 8% of customers this quarter. What should we do?",
  "metadata": {
    "company_id": "22222222-2222-4222-8222-222222222222",
    "locale": "ru"
  }
}
```

Response `202`:

```json
{
  "success": true,
  "data": {
    "message_id": "55555555-5555-4555-8555-555555555555",
    "chat_id": "33333333-3333-4333-8333-333333333333",
    "status": "queued",
    "accepted_at": "2026-11-10T12:03:00Z"
  }
}
```

This endpoint must:

1. validate session
2. verify the user belongs to the chat
3. insert the message in PostgreSQL
4. publish a task to NATS JetStream
5. return `202 Accepted` immediately

NATS payload contract for MVP:

```json
{
  "event_type": "chat.message.created",
  "message_id": "55555555-5555-4555-8555-555555555555",
  "chat_id": "33333333-3333-4333-8333-333333333333",
  "user_id": "11111111-1111-4111-8111-111111111111",
  "business_id": "22222222-2222-4222-8222-222222222222",
  "content": "We lost 8%...",
  "request_id": "a8cb6f7d-4dd0-450b-9e62-bc1a0d3ad519",
  "created_at": "2026-11-10T12:03:00Z"
}
```

The AI worker then processes the message and returns a result via the same pipeline or a subsequent API callback.

## Business API

### `POST /business`

Create a business record. MVP assumes one owner and many members.

Request:

```json
{
  "user_id": "11111111-1111-4111-8111-111111111111",
  "business_name": "Northwind Labs",
  "description": "B2B SaaS company",
  "business_data": {
    "industry": "saas",
    "revenue": 2400000,
    "employees": 32,
    "region": "europe"
  }
}
```

Response `201`:

```json
{
  "success": true,
  "data": {
    "business_id": "66666666-6666-4666-8666-666666666666",
    "user_id": "11111111-1111-4111-8111-111111111111",
    "business_name": "Northwind Labs",
    "description": "B2B SaaS company",
    "business_data": {
      "industry": "saas",
      "revenue": 2400000,
      "employees": 32,
      "region": "europe"
    },
    "created_at": "2026-11-10T12:04:00Z",
    "updated_at": "2026-11-10T12:04:00Z"
  }
}
```

### `POST /business/member`

Add a member to a business.

Request:

```json
{
  "user_id": "11111111-1111-4111-8111-111111111111",
  "business_id": "66666666-6666-4666-8666-666666666666",
  "member_user_id": "77777777-7777-4777-8777-777777777777",
  "role": "member"
}
```

Rules for MVP:

- only `user_id` who owns the business can add members
- `role` is optional and defaults to `member`
- admin role can be added later

### `DELETE /business/member`

Remove a member from a business.

Request:

```json
{
  "user_id": "11111111-1111-4111-8111-111111111111",
  "business_id": "66666666-6666-4666-8666-666666666666",
  "member_user_id": "77777777-7777-4777-8777-777777777777"
}
```

Response `200`:

```json
{
  "success": true,
  "message": "member removed successfully",
  "data": {
    "business_id": "66666666-6666-4666-8666-666666666666",
    "member_user_id": "77777777-7777-4777-8777-777777777777"
  }
}
```

## What is MVP-ready here

This version is now good enough to start implementation because it fixes the major unstable points:

- IDs are explicit and consistent (`uuid`)
- resources have clear field names
- `chat_name` is no longer the sole identity key
- NATS payload shape is defined
- authorization is explicit
- business member rules are stated simply
- JSON examples are concrete and easy to test

## What still needs to be decided before production

- whether `user_id` in the body stays for MVP or is removed after auth middleware is fully enforced
- whether all writes should use path params instead of JSON bodies for some endpoints
- whether the AI result should be returned via websocket, SSE, or a second polling endpoint
- whether members need role-based permissions beyond owner/admin/member
- whether chats can belong to multiple businesses or only one business

## Final assessment

This contract is now around `8/10` as an MVP draft.

It is not yet production-grade, but it is strong enough to start building the first working Go Core service with PostgreSQL + NATS and a junior developer can implement it without guessing the main shapes and contracts.
