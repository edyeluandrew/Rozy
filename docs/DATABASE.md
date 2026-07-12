# Database & schema strategy

## Neon + auto-migrations (Go backend)

Rozy uses **SQL migrations** stored in `internal/platform/db/migrations/`.

On API startup (`AUTO_MIGRATE=true` by default), pending migrations run automatically — you do **not** need to run the migrate CLI every time during development.

```env
AUTO_MIGRATE=true
DATABASE_URL=postgresql://...
```

To disable (e.g. production with CI-managed migrations):

```env
AUTO_MIGRATE=false
```

## Zod — where it fits (and where it doesn't)

**Zod is TypeScript/JavaScript.** It does not replace Postgres or Go migrations.

| Layer | Tool | Purpose |
|-------|------|---------|
| **Database schema** | SQL migrations | Tables, indexes, enums — source of truth in Neon |
| **Go API input** | `go-playground/validator` | Validate request JSON (phone, OTP length, etc.) |
| **Admin / React forms** | **Zod** (recommended) | Form + API client validation |
| **Flutter** | Manual models / optional validators | Parse API responses safely |

If you want "schema as code" like Drizzle + Zod, that pattern is for **Node/TS** backends. Rozy's Go backend keeps **SQL migrations + validator tags** — simpler and idiomatic for Go.

## PostGIS on Neon

The first migration enables `postgis`. If Neon project doesn't support it on your plan, enable the extension in Neon dashboard → Extensions.

## Rotating credentials

Never commit `.env`. Rotate Neon password if exposed.
