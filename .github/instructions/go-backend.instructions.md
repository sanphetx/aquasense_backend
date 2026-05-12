---
description: "Use when writing Go backend code in the AquaSense project. Covers layered architecture, Gin handlers, GORM queries, database models, security, JWT, OAuth, IDOR prevention, and all coding patterns."
applyTo: "internal/**/*.go"
---
# Go Backend — Complete Coding Guidelines

---

## 1. Layered Architecture (ห้ามข้าม layer)

```
Handler (bind JSON, validate) → Service (business logic) → Repository (GORM queries) → MySQL
```

| Layer | ✅ ต้องทำ | ❌ ห้ามทำ |
|-------|----------|----------|
| **Handler** | bind JSON, validate input, map error → HTTP status | เรียก DB ตรง, มี business logic, import `gorm` |
| **Service** | JWT generate, OAuth verify, data transform, validation rules | import `gin`, เรียก DB ตรง, return HTTP status |
| **Repository** | GORM CRUD, raw SQL, transaction | มี business logic, return HTTP status, import `gin` |

---

## 2. Response Helpers

ใช้ helper functions ที่มีอยู่แล้ว **ห้ามสร้างใหม่**:

```go
ok(c, data)           // 200 — ดึงข้อมูลสำเร็จ
created(c, data)      // 201 — สร้างข้อมูลสำเร็จ
badRequest(c, msg)    // 400 — input ไม่ถูกต้อง
unauthorized(c, msg)  // 401 — ไม่มี token / token หมดอายุ
notFound(c, msg)      // 404 — resource ไม่พบ
serverError(c, err)   // 500 — unexpected error (log error, ไม่ expose ให้ client)
```

### Response JSON Format (ห้ามเปลี่ยน structure)

```json
// Success — ต้องมี "success": true + "data"
{ "success": true, "data": { ... } }

// Error — ต้องมี "success": false + "error" (string เท่านั้น)
{ "success": false, "error": "human-readable message" }
```

---

## 3. Error Handling — Sentinel Errors

```go
// ─── Repository (กำหนด error ไว้ด้านบนไฟล์) ─────────────────────────────────
var ErrFarmNotFound = errors.New("farm not found")
var ErrSensorNotFound = errors.New("sensor not found")

func (r *FarmRepository) GetFarmByID(farmID, userID string) (*models.Farm, error) {
    var farm models.Farm
    err := r.db.Where("id = ? AND user_id = ?", farmID, userID).First(&farm).Error
    if errors.Is(err, gorm.ErrRecordNotFound) {
        return nil, ErrFarmNotFound  // ✅ คืน sentinel error
    }
    return &farm, err
}

// ─── Handler (map error → HTTP status) ───────────────────────────────────────
func (h *FarmHandler) GetFarm(c *gin.Context) {
    userID := c.GetString("userID")
    farm, err := h.repo.GetFarmByID(farmID, userID)
    if err != nil {
        switch {
        case errors.Is(err, repository.ErrFarmNotFound):
            notFound(c, "farm not found")
        default:
            serverError(c, err)
        }
        return
    }
    ok(c, farm.ToJSON())
}
```

| กฎ | ✅ ถูกต้อง | ❌ ผิด |
|----|----------|--------|
| เปรียบเทียบ error | `errors.Is(err, gorm.ErrRecordNotFound)` | `err == gorm.ErrRecordNotFound` |
| คืน error จาก repo | `return nil, ErrFarmNotFound` | `return nil, fmt.Errorf("farm not found")` |
| error message | lowercase, no period | `"Farm Not Found."` |

---

## 4. IDOR Prevention — กฎเหล็ก

> **ทุก protected endpoint ต้องใช้ user_id จาก JWT ควบคู่กับ resource ID เสมอ**

```go
// ✅ ถูกต้อง — ป้องกันคนอื่นเข้าถึง
userID := c.GetString("userID")
db.Where("id = ? AND user_id = ?", farmID, userID).First(&farm)

// ✅ ถูกต้อง — list ข้อมูลเฉพาะของ user นั้น
db.Where("user_id = ?", userID).Find(&farms)

// ❌ ห้ามเด็ดขาด — ใครก็เข้าถึงได้ถ้ารู้ ID
db.Where("id = ?", farmID).First(&farm)

// ❌ ห้าม — รับ user_id จาก request body/param
userID := c.Param("userID") // NEVER trust client-provided user ID
```

**ข้อยกเว้น**: Admin endpoints ที่ผ่าน AdminMiddleware แล้ว → ใช้ target user_id จาก param ได้

---

## 5. Handler Pattern — Template

```go
func (h *XxxHandler) DoSomething(c *gin.Context) {
    // 1. อ่าน userID จาก JWT context
    userID := c.GetString("userID")

    // 2. Bind + validate request body
    var req models.SomeRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        badRequest(c, err.Error())
        return
    }

    // 3. เรียก service/repository
    result, err := h.service.DoSomething(userID, &req)

    // 4. Map error → HTTP status
    if err != nil {
        switch {
        case errors.Is(err, repository.ErrXxxNotFound):
            notFound(c, "xxx not found")
        default:
            serverError(c, err)
        }
        return
    }

    // 5. Return success
    ok(c, result)
}
```

### Context Values from JWT Middleware

```go
userID := c.GetString("userID")  // UUID string — เจ้าของ resource
email  := c.GetString("email")   // email ที่ login
role   := c.GetString("role")    // "user" หรือ "admin"
```

**ห้ามใช้ `c.Param()` หรือ `c.Query()` เป็น user identifier** — ต้องใช้จาก JWT เท่านั้น

---

## 6. Adding a New Endpoint — 6 ขั้นตอน (ห้ามข้าม)

### 1. เพิ่ม Request/Response struct → `models/models.go`

```go
type CreateXxxRequest struct {
    Name string `json:"name" binding:"required"`
}
```

### 2. เพิ่ม repository method → `repository/repository.go`

```go
func (r *XxxRepository) Create(userID string, req *models.CreateXxxRequest) (*models.Xxx, error) {
    item := models.Xxx{ID: uuid.NewString(), UserID: userID, Name: req.Name}
    if err := r.db.Create(&item).Error; err != nil {
        return nil, err
    }
    return &item, nil
}
```

### 3. เพิ่ม service method (ถ้ามี business logic) → `service/service.go`

### 4. เพิ่ม handler → `handlers/handlers.go`

### 5. เพิ่ม route → `router/router.go`

### 6. Wire DI → `cmd/server/main.go`

```go
xxxRepo := repository.NewXxxRepository(db)
xxxHandler := handlers.NewXxxHandler(xxxRepo)
```

---

## 7. GORM — Model Definition

```go
type MyModel struct {
    ID   string `gorm:"primaryKey;type:varchar(36)"` // UUID v4
    Name string `gorm:"type:varchar(255);not null"`
    models.BaseModel // embeds created_at, updated_at, deleted_at, created_by, updated_by
}
```

| กฎ | ✅ ถูกต้อง | ❌ ผิด |
|----|----------|--------|
| Tag | `gorm:"type:varchar(100)"` | `db:"type:varchar(100)"` |
| Primary Key | `gorm:"primaryKey;type:varchar(36)"` + UUID | auto-increment int |
| Nullable | ใช้ pointer `*string`, `*float64` | ใช้ zero value |
| Audit fields | embed `models.BaseModel` | สร้าง created_at เอง |
| JSON column | `gorm:"type:json"` + string type | `[]string` ตรงๆ |

---

## 8. GORM — Foreign Key Constraints

```go
// ─── One-to-Many (User มีหลาย Farm) ─────────────────────────────────────────
type Farm struct {
    ID     string `gorm:"primaryKey;type:varchar(36)"`
    UserID string `gorm:"type:varchar(36);not null;index"`
    User   User   `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE,OnUpdate:CASCADE"`
}

// ─── Nullable FK (User → SubscriptionPlan) ───────────────────────────────────
type User struct {
    SubscriptionPlan    string            `gorm:"type:varchar(50)"`
    SubscriptionPlanRef *SubscriptionPlan `gorm:"foreignKey:SubscriptionPlan;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
}
```

| Strategy | ใช้เมื่อ | ตัวอย่าง |
|----------|---------|---------|
| `CASCADE` | child ไม่มีความหมายถ้า parent ถูกลบ | Farm→User, UserNode→User |
| `SET NULL` | child ยังคงอยู่ได้ FK เป็น null | User→SubscriptionPlan, Farm→Sensor |

---

## 9. GORM — Transaction Pattern

```go
// ✅ ถูกต้อง — ใช้ tx (ไม่ใช่ db) ภายใน transaction
err := r.db.Transaction(func(tx *gorm.DB) error {
    if err := tx.Model(&models.UserNode{}).
        Where("user_id = ? AND is_active = ?", userID, true).
        Update("is_active", false).Error; err != nil {
        return err // auto rollback
    }
    if err := tx.Model(&models.UserNode{}).
        Where("id = ? AND user_id = ?", nodeID, userID).
        Update("is_active", true).Error; err != nil {
        return err
    }
    return nil // auto commit
})

// ❌ ผิด — ใช้ db แทน tx
r.db.Model(&models.UserNode{}).Where(...).Update(...)
r.db.Model(&models.Farm{}).Where(...).Update(...)
```

| กรณี | ต้องใช้ Transaction? |
|------|-------------------|
| Write 1 table | ❌ ไม่ต้อง |
| Write 2+ tables ที่ต้อง consistent | ✅ บังคับ |
| Read-only queries | ❌ ไม่ต้อง |

---

## 10. GORM — AutoMigrate & Seed

### AutoMigrate Order (FK dependency — parent ก่อน child)

```go
db.AutoMigrate(
    &models.SubscriptionPlan{},   // 1. ไม่มี FK
    &models.Sensor{},             // 2. ไม่มี FK
    &models.User{},               // 3. FK → SubscriptionPlan
    &models.Farm{},               // 4. FK → User, Sensor
    &models.WaterRecord{},        // 5. FK → Sensor
    &models.NotificationSettings{}, // 6. FK → User
    &models.UserNode{},           // 7. FK → User, Sensor
    &models.AiRecommendation{},   // 8. ไม่มี FK
    &models.CropSuggestion{},     // 9. ไม่มี FK
)
```

### Seed Pattern (idempotent)

```go
func seedXxx(db *gorm.DB) {
    var count int64
    db.Model(&models.Xxx{}).Count(&count)
    if count > 0 {
        return // already seeded
    }
    records := []models.Xxx{{ID: "xxx001", Name: "..."}}
    for _, r := range records {
        db.Create(&r)
    }
}
```

| กฎ Seed | รายละเอียด |
|---------|-----------|
| Idempotent | check count ก่อน — รันซ้ำไม่ duplicate |
| FK order | seed parent ก่อน child (plans → users → farms) |
| Fixed IDs | ใช้ fixed string ID (e.g. `"s001"`, `"u001"`) |
| Password | bcrypt hash — ห้าม plaintext |

---

## 11. GORM — Soft Delete

| สถานการณ์ | GORM จัดการให้? | ต้องทำเอง? |
|-----------|---------------|------------|
| `db.Delete(&model)` | ✅ set deleted_at | — |
| `db.Where(...).Find(&list)` | ✅ filter deleted_at IS NULL | — |
| Raw SQL / Exec | ❌ | ต้องเพิ่ม `WHERE deleted_at IS NULL` |
| ต้องการเห็น deleted records | — | ใช้ `db.Unscoped()` |

---

## 12. JWT Authentication

| Setting | ค่า |
|---------|-----|
| Algorithm | **HMAC-SHA256** (`golang-jwt/jwt/v5`) |
| Payload | `user_id`, `email`, `role` |
| Expiry | `JWT_EXPIRE_HOURS` (default 24h) |
| Secret | `JWT_SECRET` env var |

### Production Guard — Fail-Fast

```go
// ✅ GIN_MODE=release + ไม่ตั้ง JWT_SECRET → crash ทันที
if jwtSecret == "" && ginMode == "release" {
    log.Fatal("[FATAL] JWT_SECRET env var must be set in production")
}

// ❌ ห้าม fallback ใช้ default secret ใน production
if jwtSecret == "" { jwtSecret = "default-secret" } // NEVER
```

### AuthMiddleware Flow

```
Authorization: Bearer <token>
  → Parse JWT + validate signature + check expiry
  → c.Set("userID", ...), c.Set("email", ...), c.Set("role", ...)
  → c.Next()
```

### AdminMiddleware

```go
// ต้องวางหลัง AuthMiddleware เสมอ
// ตรวจ role == "admin" → ถ้าไม่ใช่ → 403 Forbidden
```

---

## 13. Social Login — Google OAuth

```
Flutter → id_token → Backend
  → GET https://oauth2.googleapis.com/tokeninfo?id_token=<token>
  → Validate: email_verified == "true", aud == GOOGLE_CLIENT_ID, email != ""
  → FindOrCreate user → Generate JWT → return
```

```go
// ✅ ต้องตรวจทั้ง 3 เงื่อนไข
if info.EmailVerified != "true" { return errors.New("email not verified") }
if info.Aud != s.googleClientID { return errors.New("invalid audience") }
if info.Email == "" { return errors.New("email is empty") }
```

---

## 14. Social Login — Apple Sign In

```
Flutter → id_token → Backend
  → Fetch Apple JWKS (cached 24h): GET https://appleid.apple.com/auth/keys
  → Find matching key by `kid` header → RSA verify
  → Validate: iss == "https://appleid.apple.com", aud == APPLE_CLIENT_ID, exp > now
  → FindOrCreate user → Generate JWT → return
```

### JWKS Caching

```go
var (
    cachedKeys  []AppleKey
    cacheExpiry time.Time
    cacheMutex  sync.RWMutex
)
const jwksCacheDuration = 24 * time.Hour
```

---

## 15. OAuth HTTP Client

```go
// ✅ ถูกต้อง — มี timeout ป้องกัน goroutine leak
var oauthClient = &http.Client{Timeout: 10 * time.Second}

// ❌ ผิด — default client ไม่มี timeout
resp, err := http.Get("https://oauth2.googleapis.com/tokeninfo?...")
```

---

## 16. Security Rules Summary

- **ห้ามเก็บ password เป็น plaintext** — ใช้ `bcrypt.GenerateFromPassword()` เสมอ
- **ห้ามส่ง internal error ให้ client** — `serverError(c, err)` log แต่ตอบ generic message
- **ห้ามใช้ `c.Param()` เป็น user identity** — อ่านจาก JWT context เท่านั้น
- **ห้ามปิด token expiry** — ทุก token ต้องมี `exp` claim
- **ห้าม log JWT secret / password / token** — log แค่ user ID + action

### Production Checklist

| # | รายการ | เหตุผล |
|---|--------|--------|
| 1 | ตั้ง `JWT_SECRET` (32+ chars) | ป้องกัน token forgery |
| 2 | ตั้ง `GIN_MODE=release` | ปิด debug logs |
| 3 | ตั้ง `CORS_ALLOWED_ORIGINS` (specific domains) | ป้องกัน CSRF |
| 4 | ตั้ง `GOOGLE_CLIENT_ID` + `APPLE_CLIENT_ID` | validate OAuth audience |
| 5 | ปิด AutoMigrate | ใช้ migration tool แทน |
| 6 | ไม่ commit `.env` | credentials leak |
| 7 | bcrypt cost ≥ 10 | brute-force protection |

---

## 17. Connection Pool

```go
sqlDB.SetMaxOpenConns(25)
sqlDB.SetMaxIdleConns(10)
sqlDB.SetConnMaxLifetime(5 * time.Minute)
```

ไม่ต้องแก้ค่าเหล่านี้ยกเว้นมี load testing results
