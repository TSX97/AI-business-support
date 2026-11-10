# Project guidance

## Architecture

The intended request and service flow is documented in [docs/architecture.md](docs/architecture.md):

- The React + TypeScript + Redux frontend is built with Vite.
- Browser traffic reaches NGINX over HTTPS. NGINX terminates TLS and proxies API requests to internal services over HTTP.
- The Python Auth Service uses PostgreSQL and Redis.
- The Go Core Service uses PostgreSQL and communicates with the Python AI Service through NATS.
- PostgreSQL is shared infrastructure; keep service data in the appropriate domain/schema and do not bypass service boundaries.

## Repository layout

- `frontend/` — web client.
- `backend/auth/` — Python authentication service.
- `backend/core/` — Go core service.
- `backend/ai/` — Python AI service.
- `migrations/` — PostgreSQL schema migrations.
- `nginx/` — reverse proxy and TLS configuration.
- `docs/` — architecture and API documentation.
- `docker-compose.yml` — local infrastructure and service orchestration.

## Development practices

- Keep internal service traffic on the Docker network; expose public API access through NGINX.
- Keep secrets in environment configuration, not source files. Use `.env.example` for documenting required variables.
- Update migrations and service/API documentation when changing persisted data or request contracts.
- Preserve the existing frontend scripts in `frontend/package.json`; use `pnpm build` and `pnpm lint` for frontend validation when applicable.
