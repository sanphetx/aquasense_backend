# คำสั่งสำหรับ GitHub Copilot — AquaSense Backend

## บริบทของโปรเจกต์

ระบบ Backend API สำหรับแอป AquaSense TDS (ตรวจวัดคุณภาพน้ำสำหรับเกษตรกร)
เขียนด้วย **Go 1.25** ใช้ **Gin v1.12** เป็น HTTP Framework, **GORM v1.31** เป็น ORM และ **MySQL 8.0+** เป็นฐานข้อมูล
ให้อ่าน `ARCHITECTURE.md` ก่อนเสมอเพื่อทำความเข้าใจ flow, API endpoints, และสถานะแต่ละระบบ ก่อนที่จะเขียน code ใดๆ

---

## กฎสถาปัตยกรรม

### Layered Architecture — ห้ามข้าม layer เด็ดขาด

```
Handler (HTTP layer)  →  Service (business logic)  →  Repository (GORM queries)  →  MySQL
```

| Layer | หน้าที่ | ห้ามทำ |
|-------|--------|--------|
| **Handler** | bind JSON, validate, map error → HTTP status | ❌ เรียก DB ตรง, ❌ มี business logic |
| **Service** | JWT, OAuth verify, data transformation | ❌ import `gin`, ❌ เรียก DB ตรง |
| **Repository** | GORM CRUD operations | ❌ มี business logic, ❌ return HTTP status |

### โครงสร้างโฟลเดอร์

```
aquasense-backend/
├── cmd/server/main.go              # Entry point: DI wiring + Graceful Shutdown (5s)
├── internal/
│   ├── config/config.go            # อ่าน .env → Config struct + DSN()
│   ├── database/database.go        # MySQL connect + AutoMigrate + Seed functions
│   ├── handlers/handlers.go        # HTTP handlers (Auth, Farm, Sensor, Account, AI, Node)
│   ├── logger/logger.go            # zap logger (dev=console, prod=JSON)
│   ├── middleware/middleware.go     # AuthMiddleware (JWT→context) + AdminMiddleware
│   ├── models/models.go            # DB models + JSON shapes + Request/Response types
│   ├── repository/repository.go    # CRUD layer (Auth, Farm, Sensor, Notification, Node, AI, SubscriptionPlan)
│   ├── router/router.go            # Route groups + CORS + Swagger
│   └── service/
│       ├── service.go              # Business logic (Auth, AI, Subscription)
│       └── oauth.go                # Google tokeninfo + Apple JWKS verification
├── scripts/schema.sql              # DDL + seed data (production deploy)
├── docs/                           # Swagger auto-generated
├── .env                            # Config จริง (ห้าม commit!)
├── .env.example                    # Template
└── Makefile                        # make run / build / test / swag / tidy
```

---

## Dependency Injection — Manual Wiring

DI ทั้งหมดอยู่ใน `cmd/server/main.go` — สร้าง instance ตามลำดับ dependency:

```go
// ลำดับ DI (ห้ามสลับ)
db → repos → services → handlers → router

// ตัวอย่าง
aiRepo := repository.NewAiRepository(db)
aiService := service.NewAiService(aiRepo)
aiHandler := handlers.NewAiHandler(aiService)
```

**กฎ DI**:
- ห้ามใช้ global variable สำหรับ db หรือ service
- ทุก struct ต้องรับ dependency ผ่าน constructor (`NewXxx(...)`)
- Handler รับ Service, Service รับ Repository, Repository รับ `*gorm.DB`

---

## Environment Variables

| ตัวแปร | Default | คำอธิบาย | Production |
|--------|---------|----------|------------|
| `DB_HOST` | `localhost` | MySQL host | ต้องตั้ง |
| `DB_PORT` | `3306` | MySQL port | ต้องตั้ง |
| `DB_NAME` | `aquasense` | Database name | ต้องตั้ง |
| `DB_USER` | `root` | MySQL user | ต้องตั้ง |
| `DB_PASSWORD` | _(empty)_ | MySQL password | ต้องตั้ง |
| `JWT_SECRET` | _(insecure dev)_ | HMAC signing key | **บังคับ** (crash ถ้าไม่ตั้ง) |
| `JWT_EXPIRE_HOURS` | `24` | Token lifetime | ปรับตามต้องการ |
| `SERVER_PORT` | `8080` | HTTP listen port | ปรับตามต้องการ |
| `GIN_MODE` | `debug` | Gin mode | ต้องเป็น `release` |
| `CORS_ALLOWED_ORIGINS` | `*` | CORS origins | **ห้าม `*`** |
| `GOOGLE_CLIENT_ID` | _(empty)_ | OAuth audience check | ต้องตั้ง |
| `APPLE_CLIENT_ID` | _(empty)_ | OAuth audience check | ต้องตั้ง |
| `ADMIN_DEFAULT_PASSWORD` | _(random UUID)_ | Seed admin password | ต้องตั้ง |

### กฎ Fail-Fast — Production Guard

```go
// ถ้า GIN_MODE=release + ไม่ตั้ง JWT_SECRET → server crash ทันที
if jwtSecret == "" && ginMode == "release" {
    log.Fatal("[FATAL] JWT_SECRET env var must be set in production")
}
```

นี่คือ **fail-fast อย่างตั้งใจ** — ดีกว่า fallback ใช้ default secret แล้วเกิด token leak

---

## Response Format — Unified JSON

ทุก endpoint ต้องตอบ JSON format เดียวกัน:

```json
// Success
{ "success": true, "data": { ... } }

// Error
{ "success": false, "error": "error message" }
```

### Helper Functions (ห้ามสร้างใหม่)

```go
ok(c, data)           // 200 OK
created(c, data)      // 201 Created
badRequest(c, msg)    // 400 Bad Request
unauthorized(c, msg)  // 401 Unauthorized
notFound(c, msg)      // 404 Not Found
serverError(c, err)   // 500 Internal Server Error
```

---

## Error Handling — Sentinel Errors

**ห้ามใช้ `fmt.Errorf("...")` คืนจาก Repository ไป Handler**
กำหนด error ไว้ด้านบนของ Repository → Handler ใช้ `errors.Is()` แยก HTTP status:

```go
// ✅ ถูกต้อง — Repository
var ErrFarmNotFound = errors.New("farm not found")

// ✅ ถูกต้อง — Handler
if errors.Is(err, repository.ErrFarmNotFound) {
    notFound(c, "farm not found")
    return
}
serverError(c, err)

// ❌ ผิด — ห้ามใช้ == เปรียบเทียบ error
if err == gorm.ErrRecordNotFound { ... }

// ❌ ผิด — ห้ามใช้ fmt.Errorf คืนจาก repo
return fmt.Errorf("farm not found")
```

---

## IDOR Prevention — กฎเหล็ก

> **ห้าม** query ด้วยแค่ resource ID เด็ดขาด — **ต้อง** แนบ `user_id` จาก JWT เสมอ

```go
// ✅ ถูกต้อง — ป้องกัน IDOR
db.Where("id = ? AND user_id = ?", farmID, userID).First(&farm)

// ❌ ผิด — user คนอื่นเข้าถึงได้
db.Where("id = ?", farmID).First(&farm)
```

ทุก protected endpoint ต้องอ่าน `userID` จาก JWT context:
```go
userID := c.GetString("userID")
```

---

## Database — GORM Conventions

### AutoMigrate Order (FK dependency)

```go
// ต้องเรียงตาม FK — parent ก่อน child
SubscriptionPlan → Sensor → User → Farm → WaterRecord → NotificationSettings → UserNode → AiRecommendation → CropSuggestion
```

### Seed Order (FK dependency)

```go
seedPlans()     → seedSensors()   → seedAiData()   → seedDemoUsers() → seedAdmin()
// plans ก่อน users (FK: users.subscription_plan → subscription_plans.id)
// sensors ก่อน farms (FK: farms.active_sensor_id → sensors.id)
```

### Transaction — Multi-table Writes

```go
// ✅ ต้องห่อ Transaction เมื่อ write หลายตาราง
err := db.Transaction(func(tx *gorm.DB) error {
    if err := tx.Model(&UserNode{}).Where(...).Update(...).Error; err != nil {
        return err // auto rollback
    }
    if err := tx.Model(&Farm{}).Where(...).Update(...).Error; err != nil {
        return err
    }
    return nil // auto commit
})
```

### FK Constraints

```go
// ✅ ทุก FK ต้องมี constraint
User *User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE,OnUpdate:CASCADE"`

// OnDelete:CASCADE → child ถูกลบตาม parent
// OnDelete:SET NULL → child ยังอยู่แต่ FK เป็น null (เช่น subscription_plan)
```

---

## Build & Run

```bash
make run      # go run cmd/server/main.go
make build    # → bin/aquasense-api
make test     # unit tests
make swag     # regenerate Swagger docs
make tidy     # go mod tidy
```

### เริ่มต้นโปรเจคบนเครื่องใหม่

```bash
cp .env.example .env           # 1. สร้าง config
# แก้ DB_PASSWORD ใน .env      # 2. ตั้งค่า DB
mysql -u root -e "CREATE DATABASE IF NOT EXISTS aquasense"  # 3. สร้าง DB
make run                       # 4. Server จะ AutoMigrate + seed อัตโนมัติ
curl http://localhost:8080/health  # 5. ทดสอบ
```

---

## การตั้งชื่อ

| ประเภท | รูปแบบ | ตัวอย่าง |
|--------|--------|---------|
| Handler struct | `*Handler` | `AuthHandler`, `FarmHandler` |
| Service struct | `*Service` | `AuthService`, `AiService` |
| Repository struct | `*Repository` | `FarmRepository`, `AiRepository` |
| Constructor | `New*` | `NewAuthHandler(...)`, `NewFarmRepository(...)` |
| DB Model | คำนามเดี่ยว PascalCase | `User`, `Farm`, `Sensor` |
| JSON Response | `*JSON` | `UserJSON`, `FarmJSON`, `SensorJSON` |
| Request struct | `*Request` | `LoginRequest`, `CreateFarmRequest` |
| Sentinel Error | `Err*` | `ErrFarmNotFound`, `ErrSensorNotFound` |
| Config field | PascalCase | `JWTSecret`, `GoogleClientID` |

---

## กฎทั่วไป

- **ห้ามมี business logic ใน Handler** — logic ต้องอยู่ใน Service หรือ Repository
- **ห้ามเรียก DB โดยตรงใน Handler** — ต้องผ่าน Repository เสมอ
- **ห้ามใช้ `==` เปรียบเทียบ error** — ใช้ `errors.Is()` เท่านั้น
- **ห้ามใช้แค่ resource ID query** — ต้องแนบ `user_id` จาก JWT (IDOR prevention)
- **ห้าม commit `.env`** — ใช้ `.env.example` เป็น template
- ใช้ **UUID v4** (`google/uuid`) สำหรับ primary key ทุก model
- ใช้ **BaseModel** embed ใน model ที่ต้องการ audit fields (created_at, updated_at, deleted_at)
- **Soft Delete** ใช้ `gorm.DeletedAt` — GORM filter `deleted_at IS NULL` อัตโนมัติ
- Raw SQL ต้องเพิ่ม `WHERE deleted_at IS NULL` เอง

---

## กฎการอัปเดต ARCHITECTURE.md

> **บังคับใช้ทุก session ที่มีการเปลี่ยนแปลง code — ห้ามละเมิด**

### หลักการ: Code First, Document After

**ก่อนจะเพิ่มรายการใน ARCHITECTURE.md ต้อง implement ลง Go code จริงก่อนเสมอ**

### ลำดับขั้นตอนที่ถูกต้อง

```
1. อ่าน ARCHITECTURE.md → เข้าใจ requirement
2. เขียน/แก้ code ใน internal/ จริง
3. ตรวจ go build ./... → ไม่มี compile error
4. จึงอัปเดต ARCHITECTURE.md → เพิ่ม changelog + อัปเดตตาราง
```

### สัญลักษณ์สถานะ

| สัญลักษณ์ | ความหมาย |
|----------|---------|
| ✅ | พร้อมใช้จริง — code ทำงานได้ครบ |
| 🔶 | Mock/Stub — endpoint มีแต่ยังไม่ integrate ระบบจริง |
| ⏳ | รอทีมอื่น — API พร้อมแต่รอ data จริง |
| ❌ | ยังไม่มี — ต้อง implement |

### ห้ามทำ

```
❌ อัปเดต ARCHITECTURE.md ก่อนเขียน code
❌ ทำเครื่องหมาย ✅ เพราะ "มี struct แล้ว" แต่ยังไม่มี endpoint
❌ ทำเครื่องหมาย ✅ เพราะ "มี route แล้ว" แต่ handler ยัง return 501
❌ copy สถานะจาก README.md มาเป็น ✅ โดยไม่ตรวจ code จริง
```
