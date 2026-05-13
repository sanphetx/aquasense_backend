# AquaSense Backend — Copilot Instructions

## Project Overview
Go REST API backend for the **AquaSense** Flutter app (TDS water-quality monitoring for farmers).
Stack: **Go 1.25 · Gin · GORM · MySQL 8** — runs on port `8080`.

---

## Architecture: Strict Layered Pattern
```
HTTP Request
  └─► Handler       (internal/handlers/handlers.go)    — parse request, call service, format response
        └─► Service (internal/service/service.go)       — business logic, no DB access
              └─► Repository (internal/repository/repository.go) — DB queries via GORM only
```
- **Never** skip layers (e.g. handler must NOT call repository directly)
- **Never** put business logic in handlers or SQL in services
- Dependency injection is wired in `cmd/server/main.go`

---

## File Map
| File | Responsibility |
|------|---------------|
| `cmd/server/main.go` | Entry point — creates all deps, injects them, starts server |
| `internal/config/config.go` | Loads env vars; all config keys are here |
| `internal/database/database.go` | GORM connection pool + AutoMigrate + seed functions |
| `internal/models/models.go` | All structs: DB models, JSON shapes, request bodies |
| `internal/repository/repository.go` | All DB queries (no business logic) |
| `internal/service/service.go` | Business logic (AuthService, AiService, SubscriptionService) |
| `internal/service/oauth.go` | Google + Apple token verification |
| `internal/handlers/handlers.go` | HTTP handlers — one struct per feature group |
| `internal/router/router.go` | Route registration + CORS + Swagger middleware |
| `internal/middleware/middleware.go` | JWT auth middleware + Admin role middleware |
| `scripts/schema.sql` | MySQL schema + seed data (source of truth for DB structure) |
| `docs/swagger.yaml` | OpenAPI spec (regenerate with `make swag`) |

---

## Coding Rules — MUST Follow

### 1. Security: Anti-IDOR Pattern
Always bind `user_id` from JWT alongside the resource ID:
```go
// ✅ Correct
db.Where("id = ? AND user_id = ?", farmID, userID)
// ❌ Wrong — allows any authenticated user to access any resource
db.Where("id = ?", farmID)
```

### 2. Database Transactions
Any operation touching multiple tables MUST use a transaction:
```go
db.Transaction(func(tx *gorm.DB) error {
    // all writes here
    return nil
})
```

### 3. Sentinel Errors
Define errors at the top of repository.go, use `errors.Is()` in handlers:
```go
// repository
var ErrFarmNotFound = errors.New("farm not found")
// handler
if errors.Is(err, repository.ErrFarmNotFound) { c.JSON(404, ...) }
```

### 4. Response Envelope
All responses use the standard wrapper from `models.go`:
```go
// Success
c.JSON(200, models.APIResponse{Success: true, Data: payload})
// Error
c.JSON(400, models.ErrorResponse{Success: false, Error: "message"})
```

### 5. JSON Field Naming
- DB models use Go naming (PascalCase)
- JSON responses use `snake_case` (match Flutter `fromJson`)
- Request bodies have `binding:"required"` tags for mandatory fields

---

## Common Commands
```bash
make run      # go run cmd/server/main.go
make build    # compile to bin/aquasense-api
make swag     # regenerate docs/swagger.yaml from annotations
make test     # go test -v ./...
make tidy     # go mod tidy
```

> **กฎ Swagger**: ทุกครั้งที่เพิ่มหรือแก้ไข handler ต้องเพิ่ม/อัปเดต annotation comment และรัน `make swag`
> Annotation ขั้นต่ำที่ต้องมีทุก handler:
> ```go
> // @Summary     ชื่อ endpoint
> // @Tags        Auth|Farm|Sensor|Node|Dashboard|AI|Account|Admin
> // @Accept      json
> // @Produce     json
> // @Security    BearerAuth   ← ยกเว้น public endpoints
> // @Param       body body models.XxxRequest true "description"
> // @Success     200 {object} models.APIResponse{data=models.XxxJSON}
> // @Failure     400 {object} models.ErrorResponse
> // @Router      /path [method]
> ```

---

## Environment Variables (see .env.example)
| Key | Default | Notes |
|-----|---------|-------|
| `SERVER_PORT` | `8080` | |
| `GIN_MODE` | `debug` | Set `release` for production |
| `DB_PORT` | `3307` | Avoid conflict with local MySQL 3306 |
| `JWT_SECRET` | — | **Required** in release mode; crashes if missing |
| `CORS_ALLOWED_ORIGINS` | `*` | Must be explicit domain in production |
| `GOOGLE_CLIENT_ID` | — | Required for Google Social Login |
| `APPLE_CLIENT_ID` | — | Required for Apple Social Login |

---

## Seed Accounts (from scripts/schema.sql)
| Email | Password | Role |
|-------|----------|------|
| `test@gmail.com` | `12345za` | user |
| `admin@gmail.com` | `123456` | admin |

---

## System Readiness
| Feature | Status |
|---------|--------|
| Auth — Email/Password | ✅ Ready |
| Auth — Google / Apple | ✅ Ready |
| Auth — Forgot Password | 🔶 Stub (no SMTP yet) |
| Farm Management | ✅ Ready |
| Node Management | ✅ Ready |
| Sensor / IoT Data | ⏳ Waiting for IoT team |
| Dashboard | 🔶 Seed data only |
| AI Recommendations | ✅ DB-backed |
| Subscription Plans | 🔶 DB-backed, no payment |
| Push Notification / LINE | ❌ Not implemented |

---

## API Reference (for Flutter team)
ดู **Swagger UI** (dev mode): `http://localhost:8080/swagger/index.html`  
ดู **Feature Status** สำหรับ frontend planning: `api_implementation_summary.md` — section 8
