# Core Service API: MVP contract

This is the MVP contract for the Go Core Service. It defines the HTTP fields and behavior for chats, messages, businesses, and business membership. The matching machine-readable contract is [core-service.openapi.yaml](./core-service.openapi.yaml).

## Conventions and IDs

- Every entity ID (`user_id`, `chat_id`, `message_id`, `business_id`) is a positive integer.
- In PostgreSQL, store these IDs and their foreign keys as `BIGINT`; generate primary keys with `GENERATED ... AS IDENTITY`.
- In Go, represent IDs as `int64`; send them in JSON as numbers, not quoted strings.
- Do not use chat or business names as identifiers. Names are mutable display values; IDs are stable.
- Timestamps are ISO 8601 UTC strings, for example `2026-11-11T10:00:00Z`.

All routes require `Authorization: Bearer <JWT>` (or the configured authenticated session cookie). For this MVP, requests also include `user_id`. The server must validate the token/session and require that the request's `user_id` equals the authenticated user's ID. A mismatch returns `403 Forbidden`; missing or invalid authentication returns `401 Unauthorized`.

Success responses use JSON and generally have this shape:

```json
{
  "success": true,
  "data": {}
}
```

Errors use:

```json
{
  "success": false,
  "error": {
    "code": "invalid_request",
    "message": "chat_id must be a positive integer"
  }
}
```

Common statuses: `400` invalid input, `401` missing/invalid authentication, `403` not permitted, `404` resource missing or not accessible, `409` conflicting state, `500` unexpected server error.

## Endpoint overview

| Method | Path | Purpose |
|---|---|---|
| `GET` | `/health` | Check that the Core Service process is running |
| `POST` | `/chat` | Create a chat |
| `GET` | `/chat` | Read a chat and paginated messages |
| `PUT` | `/chat` | Rename a chat |
| `DELETE` | `/chat` | Delete a chat and its messages |
| `GET` | `/chats` | List the authenticated user's chats |
| `POST` | `/chat/message` | Save a message and queue AI work |
| `GET` | `/chat/message/{message_id}/events` | Receive AI response events over SSE |
| `POST` | `/business` | Create a business |
| `GET` | `/business` | List businesses owned by or shared with the user |
| `PUT` | `/business` | Replace editable business fields |
| `DELETE` | `/business` | Delete a business and its memberships |
| `POST` | `/business/member` | Add a business member |
| `DELETE` | `/business/member` | Remove a business member |

## Health check

### `GET /health`

Unauthenticated liveness endpoint for Docker health checks and orchestration. It only confirms that the Core Service process can respond; it does not prove that PostgreSQL or NATS is reachable.

Response `200 OK`:

```json
{
  "status": "healthy"
}
```

## Chat

### `POST /chat`

Creates a chat owned by the authenticated user. `chat_name` is 1-80 characters; it must be trimmed and unique per owner. `description` is optional (maximum 500 characters); `metadata` is an optional JSON object.

Request:

```json
{
  "user_id": 42,
  "chat_name": "sales-ops",
  "description": "Q4 sales support workspace",
  "metadata": {
    "context": "support"
  }
}
```

Response `201 Created`:

```json
{
  "success": true,
  "data": {
    "chat_id": 103,
    "user_id": 42,
    "chat_name": "sales-ops",
    "description": "Q4 sales support workspace",
    "created_at": "2026-11-11T10:00:00Z",
    "updated_at": "2026-11-11T10:00:00Z"
  }
}
```

### `GET /chats`

Lists chats owned by the authenticated user, newest activity first. Uses the same cursor pagination convention as `GET /chat`; `limit` defaults to 25 and must be 1-100. `last_message_preview` is truncated to at most 200 characters and may be `null` for an empty chat.

Example:

```http
GET /chats?user_id=42&limit=25
```

Response `200 OK`:

```json
{
  "success": true,
  "data": {
    "chats": [
      {
        "chat_id": 103,
        "user_id": 42,
        "chat_name": "sales-ops",
        "description": "Q4 sales support workspace",
        "created_at": "2026-11-11T10:00:00Z",
        "updated_at": "2026-11-11T10:02:00Z",
        "last_message_preview": "How can we reduce churn?",
        "last_message_at": "2026-11-11T10:02:00Z"
      }
    ],
    "next_cursor": null
  }
}
```

### `GET /chat`

Requires query parameters `user_id` and `chat_id`. `limit` defaults to 25 and must be 1-100. `cursor` is opaque and, if present, comes from the previous response. Messages are returned oldest-to-newest within the selected page; pages continue toward older messages.

Example:

```http
GET /chat?user_id=42&chat_id=103&limit=25
```

Response `200 OK`:

```json
{
  "success": true,
  "data": {
    "chat": {
      "chat_id": 103,
      "user_id": 42,
      "chat_name": "sales-ops",
      "description": "Q4 sales support workspace",
      "created_at": "2026-11-11T10:00:00Z",
      "updated_at": "2026-11-11T10:00:00Z"
    },
    "messages": [
      {
        "message_id": 9001,
        "chat_id": 103,
        "user_id": 42,
        "role": "user",
        "content": "How can we reduce churn?",
        "created_at": "2026-11-11T10:01:00Z"
      }
    ],
    "next_cursor": null
  }
}
```

### `PUT /chat`

Renames a chat. The new name is 1-80 characters and unique for that user.

Request:

```json
{
  "user_id": 42,
  "chat_id": 103,
  "new_chat_name": "sales-ops-q4"
}
```

Response `200 OK`:

```json
{
  "success": true,
  "message": "chat renamed successfully",
  "data": {
    "chat_id": 103,
    "chat_name": "sales-ops-q4"
  }
}
```

### `DELETE /chat`

Deletes the chat and its messages. The MVP implementation should perform this atomically (for example, with a foreign key `ON DELETE CASCADE`).

Request:

```json
{
  "user_id": 42,
  "chat_id": 103
}
```

Response `200 OK`:

```json
{
  "success": true,
  "message": "chat deleted successfully",
  "data": {
    "chat_id": 103
  }
}
```

## Messages and AI queue

### `POST /chat/message`

`message_content` must contain 1-20,000 characters. Core verifies that the caller owns the chat, persists the user message, publishes the job to NATS JetStream, and returns `202 Accepted`. The response means the job was accepted for processing, not that the AI reply is ready.

Request:

```json
{
  "user_id": 42,
  "chat_id": 103,
  "message_content": "We lost 8% of customers this quarter. What should we do?",
  "metadata": {
    "locale": "ru"
  }
}
```

Response `202 Accepted`:

```json
{
  "success": true,
  "data": {
    "message_id": 9002,
    "chat_id": 103,
    "status": "queued",
    "accepted_at": "2026-11-11T10:02:00Z"
  }
}
```

MVP NATS JetStream subject: `core.chat.message.created`. Payload:

```json
{
  "event_type": "chat.message.created",
  "message_id": 9002,
  "chat_id": 103,
  "user_id": 42,
  "content": "We lost 8% of customers this quarter. What should we do?",
  "request_id": "req-01JCA8Y8C4K0M0K9D5T6H7J8K9",
  "created_at": "2026-11-11T10:02:00Z"
}
```

IDs in the NATS payload are JSON integers too. NATS delivery must be acknowledged by the consumer after successful processing; retries and duplicate delivery must not create duplicate assistant messages. The AI response is delivered to the client by the SSE endpoint below.

### `GET /chat/message/{message_id}/events` (SSE)

The client uses the `message_id` returned by `POST /chat/message` to open an authenticated Server-Sent Events stream. The stream delivers the AI response as it is generated:

```http
GET /chat/message/9002/events?user_id=42
Accept: text/event-stream
Authorization: Bearer <JWT>
```

Successful response headers:

```http
HTTP/1.1 200 OK
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
X-Accel-Buffering: no
```

The stream uses these events:

- `message.status` — initial state (`queued` or `processing`)
- `message.snapshot` — current accumulated response when the stream connects/reconnects
- `message.delta` — next response text fragment, with an increasing per-message `sequence`
- `message.completed` — final assistant message is saved; includes its `assistant_message_id` and full content; the server closes the stream
- `message.failed` — processing failed; includes a stable error code and safe user-facing message; the server closes the stream

Example stream:

```text
event: message.status
data: {"message_id":9002,"status":"processing"}

event: message.snapshot
data: {"message_id":9002,"assistant_message_id":9003,"status":"processing","content":"","next_sequence":1}

event: message.delta
data: {"message_id":9002,"sequence":1,"delta":"Start by identifying "}

event: message.delta
data: {"message_id":9002,"sequence":2,"delta":"the customers most likely to churn."}

event: message.completed
data: {"message_id":9002,"assistant_message_id":9003,"status":"completed","content":"Start by identifying the customers most likely to churn."}
```

SSE framing requires a blank line after every event. While waiting for the AI, Core sends a comment heartbeat periodically (for example `: keep-alive`) so proxies do not close an idle connection. If processing has already finished before the stream is opened, Core sends `message.snapshot` and `message.completed` immediately. On reconnect, Core sends the current persisted snapshot and then any later deltas; the MVP does not promise replay of every delta from before the disconnect. The canonical full answer is always available as an assistant message via `GET /chat`.

Native browser `EventSource` cannot attach an `Authorization` header. The frontend should use `fetch` with the Bearer token and read the response body as a stream (or use an authenticated same-site cookie if the auth service supports that flow). Configure NGINX for this route with response buffering disabled, pass through `X-Accel-Buffering: no`, and set a sufficiently long read timeout. The Core handler must flush each event.

Errors discovered before sending the `200` SSE headers use the normal JSON error response (`401`, `404`, etc.). After streaming starts, HTTP status can no longer change; report processing errors using `message.failed`.

AI response events are published by the AI service to the NATS JetStream subject `ai.chat.message.events`, keyed by `message_id`. MVP event payloads:

```json
{
  "event_type": "message.delta",
  "message_id": 9002,
  "sequence": 1,
  "delta": "Start by identifying ",
  "created_at": "2026-11-11T10:02:01Z"
}
```

```json
{
  "event_type": "message.completed",
  "message_id": 9002,
  "sequence": 2,
  "content": "Start by identifying the customers most likely to churn.",
  "created_at": "2026-11-11T10:02:02Z"
}
```

For failures, the payload is:

```json
{
  "event_type": "message.failed",
  "message_id": 9002,
  "sequence": 2,
  "error": {
    "code": "ai_processing_failed",
    "message": "The assistant could not complete this response."
  },
  "created_at": "2026-11-11T10:02:02Z"
}
```

Core consumes these events, persists the assistant response/status in PostgreSQL, and forwards them to the matching open SSE stream. Sequence numbers let Core ignore duplicate or out-of-order chunks. The AI service must publish `message.completed` only after all deltas; failures publish `message.failed`. If Core is horizontally scaled, stream delivery must be routed to the Core instance holding the HTTP connection (for example via a shared NATS consumer/fan-out design); a process-local channel alone is not sufficient.

## Business

### `GET /business`

Lists businesses the user owns or belongs to, most recently updated first. Uses cursor pagination; `limit` defaults to 25 and must be 1-100. `user_id` must match the authenticated user. Each returned `BusinessRecord.user_id` is the business owner's ID.

Example:

```http
GET /business?user_id=42&limit=25
```

Response `200 OK`:

```json
{
  "success": true,
  "data": {
    "businesses": [
      {
        "business_id": 501,
        "user_id": 42,
        "business_name": "Northwind Labs",
        "description": "B2B SaaS company",
        "business_data": {
          "industry": "saas",
          "employees": 32
        },
        "created_at": "2026-11-11T10:03:00Z",
        "updated_at": "2026-11-11T10:03:00Z"
      }
    ],
    "next_cursor": null
  }
}
```

### `POST /business`

Creates a business and makes the authenticated user its owner. `business_name` is 1-120 characters; `description` is optional (maximum 1,000 characters); `business_data` is an optional JSON object that defaults to `{}`.

Request:

```json
{
  "user_id": 42,
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

Response `201 Created`:

```json
{
  "success": true,
  "data": {
    "business_id": 501,
    "user_id": 42,
    "business_name": "Northwind Labs",
    "description": "B2B SaaS company",
    "business_data": {
      "industry": "saas",
      "revenue": 2400000,
      "employees": 32,
      "region": "europe"
    },
    "created_at": "2026-11-11T10:03:00Z",
    "updated_at": "2026-11-11T10:03:00Z"
  }
}
```

### `PUT /business`

Replaces the business's editable fields. All fields shown are required so a client cannot accidentally clear data by omitting it. To clear `description`, send `null`; to clear structured data, send `{}`. Only the business owner may update it. The owner ID itself cannot be changed through this endpoint.

Request:

```json
{
  "user_id": 42,
  "business_id": 501,
  "business_name": "Northwind Group",
  "description": null,
  "business_data": {
    "industry": "saas",
    "employees": 35
  }
}
```

Response `200 OK`: same `BusinessResponse` shape as `POST /business`, containing the updated record and its new `updated_at`.

### `DELETE /business`

Deletes the business and its membership rows. It does not delete chats or messages. Only the owner may delete the business.

Request:

```json
{
  "user_id": 42,
  "business_id": 501
}
```

Response `200 OK`:

```json
{
  "success": true,
  "message": "business deleted successfully",
  "data": {
    "business_id": 501
  }
}
```

### `POST /business/member`

Adds a user to the business. Only the business owner can add members in the MVP. The role defaults to `member`; `admin` may be assigned, but does not grant owner-only business update/delete permissions.

Request:

```json
{
  "user_id": 42,
  "business_id": 501,
  "member_user_id": 77,
  "role": "member"
}
```

Adding an existing member returns `409 Conflict`; a non-existent business or target user returns `404 Not Found`.

### `DELETE /business/member`

Removes a member. Only the owner can remove other members. The owner cannot remove themselves through this endpoint; delete the business to remove the owner relationship.

Request:

```json
{
  "user_id": 42,
  "business_id": 501,
  "member_user_id": 77
}
```

Response `200 OK`:

```json
{
  "success": true,
  "message": "member removed successfully",
  "data": {
    "business_id": 501,
    "member_user_id": 77
  }
}
```
