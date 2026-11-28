# CLAOJ API reference

`docs.go` is the checked-in swaggo definition of the Go API. It is generated
from handler annotations and is **not** served at runtime (no Swagger UI route
is registered) — treat `api/router.go` as the source of truth for routes.

## Regenerate

```bash
cd claoj
go install github.com/swaggo/swag/cmd/swag@latest
swag init --parseDependency --parseInternal
```

This rewrites `docs/docs.go` (and `swagger.json` / `swagger.yaml` when
emitted). The API base path is `/api`.

## Authentication

Handlers accept any of:

| Method | How |
|---|---|
| Session cookie | httpOnly `access_token` (browser default) |
| Bearer JWT | `Authorization: Bearer <jwt>` |
| API token | `Authorization: <64 hex chars>` (from `/user/api-token`) |

Mutations made with cookies also need the double-submit CSRF header:
`X-CSRF-Token: <csrf_token cookie>`.

Admin routes require Django permission codenames (`judge.*`), not a v2 role
flag; superuser bypasses the check.

## Errors

```json
{ "error": "human-readable message" }
```

`401` missing/invalid token, `403` missing permission or CSRF, `404` unknown
resource, `429` rate-limited.