# 🏗️ AquaSense Backend — Architecture & Developer Guide

โปรเจกต์นี้เขียนด้วย **Go (Golang)** ใช้ **Gin** เป็น HTTP Framework, **GORM** เป็น ORM และ **MySQL** เป็นฐานข้อมูล  
ออกแบบตามหลัก **Layered Architecture** แยก Handler → Service → Repository อย่างชัดเจน

---

## 🔄 การเปลี่ยนแปลงล่าสุด

### v1.2 — 2026-05-11 (Data Migration: Static → MySQL)

**เป้าหมาย**: ย้าย mock data ทั้งหมด (AI recommendations, subscription plans, crop suggestions) จาก hard-coded Go code ไปเก็บใน MySQL tables จริง เพื่อให้ข้อมูลแสดงใน DBeaver ERD และจัดการผ่าน DB ได้

| ไฟล์ | การเปลี่ยนแปลง |
|------|---------------|
| `models/models.go` | เพิ่ม 3 DB models: `SubscriptionPlan`, `AiRecommendation`, `CropSuggestion` |
| `models/models.go` | เพิ่ม FK relationship `User.SubscriptionPlanRef → SubscriptionPlan` (ON DELETE SET NULL) |
| `repository/repository.go` | เพิ่ม `SubscriptionPlanRepository` (GetAll, FindByID) |
| `repository/repository.go` | เพิ่ม `AiRepository` (GetRecommendations, GetRecommendationByID, GetAdvisoryHistory, GetCropSuggestions) |
| `service/service.go` | `AiService` เปลี่ยนจาก static data → ใช้ `AiRepository` query จาก DB |
| `service/service.go` | `SubscriptionService` เปลี่ยนจาก static data → ใช้ `SubscriptionPlanRepository` query จาก DB |
| `service/service.go` | ลบ `staticRecommendations`, `staticCropSuggestions`, `staticPlans` ทั้งหมด |
| `database/database.go` | AutoMigrate เพิ่ม `SubscriptionPlan`, `AiRecommendation`, `CropSuggestion` |
| `database/database.go` | เพิ่ม `seedPlans()` — seed 3 plans (free/starter/pro) ถ้ายังไม่มี |
| `database/database.go` | เพิ่ม `seedAiData()` — seed 4 AI recommendations + 3 crop suggestions |
| `database/database.go` | เปลี่ยนลำดับ AutoMigrate: `SubscriptionPlan` → `Sensor` → `User` → ... (FK dependency order) |
| `database/database.go` | เปลี่ยนลำดับ seed: `seedPlans()` → `seedAiData()` → `seedAdmin()` (FK dependency order) |
| `cmd/server/main.go` | เพิ่ม DI wiring: `AiRepository`, `SubscriptionPlanRepository` → inject เข้า services |
| `scripts/schema.sql` | เพิ่ม CREATE TABLE: `subscription_plans`, `ai_recommendations`, `crop_suggestions` |
| `scripts/schema.sql` | เพิ่ม FK constraint: `users.subscription_plan → subscription_plans.id` |
| `scripts/schema.sql` | ย้าย `subscription_plans` CREATE + INSERT ขึ้นก่อน `users` (FK dependency) |
| `scripts/schema.sql` | เพิ่ม seed data สำหรับ 3 ตารางใหม่ |

#### Bug Fixes (v1.2)

| Bug | ไฟล์ | แก้ไข |
|-----|------|-------|
| `CreateFarm` upsert คืน stale data | `repository.go` | เพิ่ม re-read จาก DB หลัง update |
| `SaveSettings` ignore false/0 values | `repository.go` | ใช้ `map` แทน struct สำหรับ `Updates()` |
| `GetNearbySensors` คืน soft-deleted sensors | `repository.go` | เพิ่ม `WHERE deleted_at IS NULL` |
| `GetSensorHistory` คืน `null` แทน `[]` | `repository.go` | คืน empty slice แทน nil |
| `RemoveNode` ไม่ sync `active_sensor_id` | `repository.go` | เพิ่ม logic clear active_sensor_id เมื่อลบ active node |
| `GetUserNodes` N+1 query | `repository.go` | ใช้ `Preload("Sensor")` |
| `LinkSensor` ไม่ validate sensor exists | `handlers.go` | เพิ่ม switch/case handle `ErrSensorNotFound` |
| GORM tags ใช้ `db:` แทน `gorm:` | `models.go` | เปลี่ยนเป็น `gorm:` tags ทั้งหมด |
| `GetFarmByUserID` ไม่ sort | `repository.go` | เพิ่ม `ORDER BY created_at DESC` |

---

### v1.1 — 2026-05-11 (Initial Architecture)

| ไฟล์ | การเปลี่ยนแปลง |
|------|---------------|
| `models/models.go` | เพิ่ม `BaseModel` struct embed ใน User, Farm, Sensor, UserNode |
| `models/models.go` | เพิ่ม `UserNode` model + `NodeJSON` response |
| `models/models.go` | เพิ่ม `Pagination` + `PaginatedResponse` struct |
| `models/models.go` | `RegisterRequest` รองรับ `display_name` string เดียว split อัตโนมัติ |
| `models/models.go` | `SocialLoginRequest` เปลี่ยน `access_token` → `id_token` + เพิ่ม `SocialUserInfo` struct |
| `service/oauth.go` | **ใหม่** — `VerifyGoogleToken()` call Google tokeninfo API จริง + `VerifyAppleToken()` verify RSA signature จาก Apple JWKS |
| `service/service.go` | `AuthService` เพิ่ม `googleClientID` + `appleClientID` fields |
| `service/service.go` | `SocialLogin()` เปลี่ยนจาก stub → verify จริงกับ Google/Apple |
| `repository/repository.go` | เพิ่ม `NodeRepository` + `HasFarm()` |
| `repository/repository.go` | ห่อ `AddNode`, `SetActiveNode` ด้วย **DB Transaction** กัน Race condition |
| `repository/repository.go` | ใช้ **Sentinel Errors** (`ErrFarmNotFound`, `ErrSensorNotFound`) แทน fmt.Errorf |
| `handlers/handlers.go` | เพิ่ม `NodeHandler` + helper functions + concurrent dashboard |
| `handlers/handlers.go` | ปิดช่องโหว่ **IDOR** (บังคับเช็ค `userID` ควบคู่ `farmID` เสมอ) |
| `handlers/handlers.go` | ลบ Dashboard Fallback (`s001`) → คืน 404 หากยังไม่มีฟาร์ม |
| `config/config.go` | เพิ่ม `GoogleClientID` + `AppleClientID` fields |
| `database/database.go` | AutoMigrate เพิ่ม `UserNode` table |
| `router/router.go` | เพิ่ม node routes |
| `middleware/middleware.go` | inject `role` เข้า context |
| `.env.example` | เพิ่ม `GOOGLE_CLIENT_ID`, `APPLE_CLIENT_ID`, และอธิบาย `CORS_ALLOWED_ORIGINS` สำหรับ Prod |

---

## 🛡️ มาตรฐานการพัฒนา (Coding Standards & Security)

เพื่อให้ระบบพร้อมสำหรับ Production ทีมพัฒนาต้องยึดมาตรฐานต่อไปนี้:

1. **Security & Authorization (การป้องกัน IDOR)**
   - **ห้าม** อัปเดตหรือดึงข้อมูลโดยใช้แค่ `id` ของ Resource เด็ดขาด
   - **ต้อง** แนบ `user_id` จาก JWT Token เข้าไปใน SQL Query เสมอ (เช่น `Where("id = ? AND user_id = ?", farmID, userID)`)
2. **Database Transactions**
   - การทำงานที่กระทบหลายตาราง (เช่น ตั้ง Active Node แล้วอัปเดต Active Sensor ของ Farm) **ต้อง** ห่อด้วย `db.Transaction(func(tx *gorm.DB) error { ... })` เพื่อป้องกันข้อมูลไม่ตรงกัน
3. **Foreign Key Constraints**
   - ใช้ `constraint:OnUpdate:CASCADE,OnDelete:CASCADE` ใน GORM tags เพื่อไม่ให้เกิดขยะ (Orphan records) เมื่อ User หรือ Farm ถูกลบ
4. **Sentinel Errors**
   - เลิกใช้ `fmt.Errorf("...")` คืนกลับไปหา Handler
   - กำหนดค่าตัวแปร Error ไว้ด้านบนของ Repository (เช่น `var ErrFarmNotFound = errors.New(...)`) และให้ Handler ใช้ `errors.Is(err, repository.ErrFarmNotFound)` ในการแยกประเภท HTTP Status (404, 400, 500)
5. **CORS & Environment**
   - ในโหมด Production (GIN_MODE=release) ห้ามตั้ง `CORS_ALLOWED_ORIGINS=*` และ `JWT_SECRET` ต้องอ่านจาก Environment Variable เท่านั้น (หากไม่ตั้งค่า ระบบจะ Crash ป้องกันการลืม)

---

## 🚦 สถานะความพร้อมของแต่ละระบบ

| ระบบ | สถานะ | คำอธิบาย |
|------|-------|----------|
| **Auth — Email/Password** | ✅ พร้อมใช้ | Login, Register, JWT Token, bcrypt |
| **Auth — Forgot Password** | 🔶 โครงสร้างพร้อม | Endpoint มีแล้ว แต่ยังไม่ส่ง email จริง (TODO: SMTP) |
| **Auth — Social Login (Google)** | ✅ พร้อมใช้ | verify id_token จริงผ่าน Google tokeninfo API, ตรวจ audience |
| **Auth — Social Login (Apple)** | ✅ พร้อมใช้ | verify id_token จริงผ่าน Apple JWKS (RSA signature), ตรวจ issuer + audience |
| **isFirstLogin Detection** | ✅ พร้อมใช้ | ตรวจจาก DB จริงว่า user มี farm หรือไม่ |
| **Farm Management** | ✅ พร้อมใช้ | CRUD แปลงเกษตร, อัปเดตพิกัด, เชื่อมต่อ Sensor |
| **Sensor / IoT Data** | ⏳ รอทีม IoT | API พร้อม แต่ข้อมูลมาจาก Seed data — ยังไม่มี Sensor จริง |
| **Dashboard** | 🔶 ใช้ได้กับ Seed data | ดึง sensor+history พร้อมกัน (concurrent) แต่ข้อมูลยังเป็น seed |
| **AI Recommendations** | ✅ DB-backed | 4 รายการใน `ai_recommendations` table + 3 crop suggestions ใน `crop_suggestions` table |
| **Subscription** | 🔶 DB-backed, ไม่มี payment | 3 plans ใน `subscription_plans` table, FK จาก users |
| **Notification Settings** | ✅ พร้อมใช้ | CRUD ตั้งค่าแจ้งเตือน upsert ใน DB |
| **Push Notification / LINE** | ❌ ยังไม่มี | ต้องต่อ Firebase + LINE Notify |
| **Node Management** | ✅ พร้อมใช้ | CRUD เชื่อมต่อ/ยกเลิก/เปลี่ยน Active Node, max 5 nodes, DB transaction |
| **User Profile** | ✅ พร้อมใช้ | ดู/แก้ไขโปรไฟล์ |
| **Admin Dashboard** | ✅ พร้อมใช้ | Admin ดู Dashboard ของ user คนอื่นได้ |

```
✅ พร้อมใช้จริง:  Auth (email/password, Google, Apple), Farm, Node Management, Profile, Notifications, Admin, isFirstLogin, AI (DB-backed), Subscription Plans (DB-backed)
🔶 Mock/Stub:     Dashboard (seed data), Forgot Password  
⏳ รอทีมอื่น:     IoT Sensor (รอทีม hardware)
❌ ยังไม่มี:      Push Notification, LINE Notify, Payment Gateway
```

> **หมายเหตุสำหรับ Frontend Developer**: ระบบที่เป็น 🔶 Mock ทุกตัว **เรียก API ได้ปกติ** — response format ถูกต้องตาม spec แล้ว เมื่อต่อระบบจริงไม่ต้องแก้โค้ดฝั่ง Flutter

---

## 📦 Libraries ที่ใช้ (Direct Dependencies)

| Library | เวอร์ชัน | หน้าที่ |
|---------|---------|--------|
| `gin-gonic/gin` | v1.12.0 | Web framework หลัก |
| `gin-contrib/cors` | v1.7.7 | จัดการ CORS headers |
| `gorm.io/gorm` | v1.31.1 | ORM สำหรับ MySQL |
| `gorm.io/driver/mysql` | v1.6.0 | MySQL driver |
| `golang-jwt/jwt/v5` | v5.2.1 | JWT Token |
| `google/uuid` | v1.6.0 | UUID v4 Primary Key |
| `joho/godotenv` | v1.5.1 | อ่านค่าจาก `.env` |
| `go.uber.org/zap` | v1.28.0 | Structured Logger |
| `golang.org/x/crypto` | v0.50.0 | bcrypt hash password |
| `swaggo/swag` | v1.16.6 | สร้างเอกสาร Swagger |
| `swaggo/gin-swagger` | v1.6.1 | แสดง Swagger UI |

---

## 🗄️ ตารางในฐานข้อมูล (Database Tables)

### 1. `users`
| Column | Type | คำอธิบาย |
|--------|------|---------|
| `id` | VARCHAR(36) PK | UUID |
| `first_name` | VARCHAR(100) | ชื่อ |
| `last_name` | VARCHAR(100) | นามสกุล |
| `email` | VARCHAR(255) UNIQUE | อีเมล |
| `phone` | VARCHAR(20) | เบอร์โทร |
| `birth_date` | DATE | วันเกิด |
| `password_hash` | VARCHAR(255) | bcrypt hash |
| `subscription_plan` | VARCHAR(50) FK → subscription_plans | `free`, `starter`, `pro` |
| `avatar_url` | VARCHAR(255) | URL รูป (nullable) |
| `role` | VARCHAR(20) | `user` หรือ `admin` |
| `created_at`, `updated_at`, `deleted_at` | DATETIME | audit fields (BaseModel) |

### 2. `farms`
| Column | Type | คำอธิบาย |
|--------|------|---------|
| `id` | VARCHAR(36) PK | UUID |
| `user_id` | VARCHAR(36) FK → users | เจ้าของ |
| `name` | VARCHAR(255) | ชื่อแปลง |
| `area_size_rai` | DECIMAL(10,2) | ขนาด (ไร่) |
| `crop_type` | VARCHAR(50) | ชนิดพืช |
| `yield_ton_per_rai` | DECIMAL(10,3) | ผลผลิตต่อไร่ (nullable) |
| `avg_price_baht_per_kg` | DECIMAL(10,2) | ราคาเฉลี่ย (nullable) |
| `distribution_channels` | JSON | ช่องทางจำหน่าย |
| `soil_ph` | DECIMAL(4,2) | ค่า pH ดิน (nullable) |
| `soil_problems` | JSON | ปัญหาดิน |
| `water_source` | VARCHAR(100) | แหล่งน้ำ |
| `latitude`, `longitude` | DECIMAL(10,7) | พิกัด GPS (nullable) |
| `active_sensor_id` | VARCHAR(36) | Sensor ที่เชื่อมอยู่ (nullable) |

### 3. `sensors`
| Column | Type | คำอธิบาย |
|--------|------|---------|
| `id` | VARCHAR(36) PK | UUID |
| `name` | VARCHAR(255) | ชื่อ Sensor |
| `latitude`, `longitude` | DECIMAL(10,7) | ตำแหน่งติดตั้ง |
| `status` | ENUM | `safe`, `warning`, `danger` |
| `tds_value` | DECIMAL(10,2) | ค่า TDS (ppm) |
| `temperature` | DECIMAL(5,2) | อุณหภูมิน้ำ (nullable) |
| `ph` | DECIMAL(4,2) | ค่า pH (nullable) |

### 4. `water_records`
| Column | Type | คำอธิบาย |
|--------|------|---------|
| `id` | BIGINT PK AUTO_INCREMENT | ลำดับ |
| `sensor_id` | VARCHAR(36) FK → sensors | Sensor ที่วัด |
| `date` | DATETIME | วันที่วัด |
| `tds`, `ph`, `temperature`, `soil_moisture` | DECIMAL | ค่าวัด |
| `status` | ENUM | `safe`, `warning`, `danger` |

### 5. `notification_settings`
| Column | Type | คำอธิบาย |
|--------|------|---------|
| `user_id` | VARCHAR(36) PK FK → users | เจ้าของ |
| `push_enabled` | TINYINT(1) | เปิด/ปิด push |
| `tds_threshold` | DECIMAL(10,2) | ค่า TDS แจ้งเตือน (default 400) |
| `line_enabled` | TINYINT(1) | เปิด/ปิด LINE |
| `daily_summary_time` | VARCHAR(20) | `none`, `morning`, `evening`, `both` |

### 6. `user_nodes`
| Column | Type | คำอธิบาย |
|--------|------|--------|
| `id` | VARCHAR(36) PK | UUID |
| `user_id` | VARCHAR(36) FK → users | เจ้าของ |
| `sensor_id` | VARCHAR(36) FK → sensors | Sensor ที่เชื่อมต่อ |
| `is_active` | TINYINT(1) | Node ที่ active อยู่ (ได้ 1 ตัว) |
| `created_at`, `updated_at`, `deleted_at` | DATETIME | audit fields |

### 7. `subscription_plans` *(v1.2)*
| Column | Type | คำอธิบาย |
|--------|------|--------|
| `id` | VARCHAR(36) PK | plan ID (`free`, `starter`, `pro`) |
| `name` | VARCHAR(50) | ชื่อแผน |
| `price` | VARCHAR(50) | ราคาแสดงผล (เช่น `฿59`) |
| `period` | VARCHAR(50) | ระยะเวลา (เช่น `/ฤดูกาล`) |
| `features` | JSON | รายการฟีเจอร์ (array of strings) |
| `recommended` | TINYINT(1) | แสดง badge "แนะนำ" |
| `sort_order` | INT | ลำดับแสดงผล |

### 8. `ai_recommendations` *(v1.2)*
| Column | Type | คำอธิบาย |
|--------|------|--------|
| `id` | VARCHAR(36) PK | recommendation ID |
| `title` | VARCHAR(255) | หัวข้อคำแนะนำ |
| `body` | TEXT | เนื้อหาคำแนะนำฉบับเต็ม |
| `type` | VARCHAR(50) | ประเภท: `tds_alert`, `tds_danger`, `crop_suggestion`, `fertilizer` |
| `reason_chips` | JSON | array of `{label, category}` สำหรับ Explainable AI |
| `confidence_score` | DECIMAL(3,2) | ระดับความเชื่อมั่น (0.00–1.00) |
| `created_at`, `updated_at`, `deleted_at` | DATETIME | audit fields (BaseModel) |

### 9. `crop_suggestions` *(v1.2)*
| Column | Type | คำอธิบาย |
|--------|------|--------|
| `id` | VARCHAR(36) PK | crop ID |
| `name` | VARCHAR(100) | ชื่อพืช (EN) |
| `name_th` | VARCHAR(100) | ชื่อพืช (TH) |
| `estimated_price_per_kg` | DECIMAL(10,2) | ราคาประมาณ (บาท/กก.) |
| `reason` | TEXT | เหตุผลแนะนำ |
| `icon` | VARCHAR(10) | emoji icon |
| `min_tds`, `max_tds` | DECIMAL(10,2) | ช่วง TDS ที่เหมาะสม |
| `sort_order` | INT | ลำดับแสดงผล |

---

## 🔐 ระบบ Security & Middleware

### JWT Authentication
- Login → Server สร้าง **JWT Token** (อายุ `JWT_EXPIRE_HOURS`, default 24 ชม.)
- Protected routes ส่ง Header: `Authorization: Bearer <token>`
- Token เก็บ: `user_id`, `email`, `role`

### Middleware
| Middleware | ตำแหน่ง | หน้าที่ |
|-----------|--------|--------|
| **CORS** | ทุก route | อนุญาต Flutter เรียก API ข้ามโดเมน |
| **AuthMiddleware** | `/api/v1/*` (protected) | ตรวจ JWT → inject `userID`, `email`, `role` เข้า context |
| **AdminMiddleware** | `/api/v1/admin/*` | ตรวจ `role == "admin"` → 403 ถ้าไม่ใช่ |

---

## 🚀 API Endpoints ทั้งหมด

### Public (ไม่ต้อง Login)
| Method | Path | หน้าที่ | สถานะ |
|--------|------|--------|-------|
| `POST` | `/api/v1/auth/login` | เข้าสู่ระบบ | ✅ |
| `POST` | `/api/v1/auth/register` | สมัครสมาชิก (รองรับ display_name) | ✅ |
| `POST` | `/api/v1/auth/social` | Social Login (idempotent stub) | 🔶 Mock |
| `POST` | `/api/v1/auth/forgot-password` | รีเซ็ตรหัสผ่าน | 🔶 ยังไม่ส่ง email |
| `GET` | `/health` | Health check | ✅ |
| `GET` | `/swagger/*` | Swagger UI | ✅ |

### Protected (ต้อง JWT)
| Method | Path | หน้าที่ | สถานะ |
|--------|------|--------|-------|
| `POST` | `/api/v1/farms` | สร้างแปลงเกษตร | ✅ |
| `GET` | `/api/v1/farms` | ดูแปลงของตัวเอง | ✅ |
| `PUT` | `/api/v1/farms/:id/location` | อัปเดตพิกัด GPS | ✅ |
| `POST` | `/api/v1/farms/:id/sensor` | เชื่อมต่อ Sensor กับแปลง | ✅ |
| `GET` | `/api/v1/sensors/nearby?lat=&lng=` | ค้นหา Sensor ใกล้เคียง (Haversine) | ⏳ seed data |
| `GET` | `/api/v1/sensors/:id/latest` | ค่าล่าสุด | ⏳ seed data |
| `GET` | `/api/v1/sensors/:id/history?period=7d` | ประวัติ 7d/30d | ⏳ seed data |
| `GET` | `/api/v1/sensors/:id/status` | สถานะ Sensor | ⏳ seed data |
| `GET` | `/api/v1/dashboard/summary` | Dashboard รวม (concurrent fetch) | 🔶 seed data |
| `GET` | `/api/v1/analytics/soil-moisture?period=7d` | ประวัติความชื้นดิน | ⏳ seed data |
| `GET` | `/api/v1/ai/recommendations` | คำแนะนำ AI ทั้งหมด | ✅ DB |
| `GET` | `/api/v1/ai/recommendations/:id` | คำแนะนำ AI ตาม ID | ✅ DB |
| `GET` | `/api/v1/ai/advisory-history` | ประวัติคำแนะนำ (ย้อนหลัง 7 วัน) | ✅ DB |
| `GET` | `/api/v1/ai/crop-suggestions?tds=` | แนะนำพืชตามค่า TDS | ✅ DB |
| `GET` | `/api/v1/users/profile` | ดูโปรไฟล์ | ✅ |
| `PUT` | `/api/v1/users/profile` | แก้ไขโปรไฟล์ | ✅ |
| `GET` | `/api/v1/users/notification-settings` | ดูตั้งค่าแจ้งเตือน | ✅ |
| `PUT` | `/api/v1/users/notification-settings` | บันทึกตั้งค่าแจ้งเตือน (upsert) | ✅ |
| `GET` | `/api/v1/subscriptions/plans` | ดูแผนสมาชิก | ✅ DB (ไม่มี payment) |
| `POST` | `/api/v1/subscriptions/subscribe` | สมัครแผน | ✅ DB (ไม่มี payment) |
| `GET` | `/api/v1/nodes` | ดูรายการ Node | ✅ |
| `POST` | `/api/v1/nodes` | เพิ่ม Node (max 5) | ✅ |
| `PUT` | `/api/v1/nodes/:id/active` | ตั้ง Active Node (DB transaction) | ✅ |
| `DELETE` | `/api/v1/nodes/:id` | ยกเลิก Node | ✅ |

### Admin (ต้อง JWT + role=admin)
| Method | Path | หน้าที่ | สถานะ |
|--------|------|--------|-------|
| `GET` | `/api/v1/admin/users/:user_id/summary` | ดู Dashboard ของ user อื่น | ✅ |

---

## 📂 โครงสร้างไฟล์

```text
aquasense-backend/
├── cmd/server/main.go          ← Entry point: DI + Graceful Shutdown
├── internal/
│   ├── config/config.go        ← อ่านค่า .env เก็บเป็น struct + DSN()
│   ├── database/database.go    ← MySQL connect + AutoMigrate + seedAdmin + seedPlans + seedAiData
│   ├── logger/logger.go        ← zap logger (dev=console, prod=JSON)
│   ├── middleware/middleware.go ← AuthMiddleware (JWT→context) + AdminMiddleware
│   ├── models/models.go        ← DB models + JSON shapes + Request/Response types
│   ├── repository/repository.go← CRUD layer (Auth, Farm, Sensor, Notification, Node, AI, SubscriptionPlan)
│   ├── router/router.go        ← Route definitions + CORS + Swagger
│   └── service/service.go      ← Business logic (Auth, AI DB-backed, Subscription DB-backed)
├── docs/                       ← Swagger auto-generated files
├── scripts/schema.sql          ← SQL สร้างตาราง + seed data
├── .env                        ← Config จริง (ห้าม commit!)
├── .env.example                ← Template สำหรับคนใหม่
├── .gitignore
├── Makefile                    ← make run / build / test / swag / tidy
└── go.mod                      ← Dependencies
```

### อธิบายแต่ละไฟล์

#### `cmd/server/main.go`
- โหลด Config → เชื่อม DB → สร้าง Repo → Service → Handler → Router
- Graceful Shutdown: รอ request เก่าเสร็จ (5 วินาที) ก่อนปิด

#### `internal/database/database.go`
- Connection pool: max 25, idle 10, lifetime 5 นาที
- AutoMigrate (ตามลำดับ FK): **SubscriptionPlan** → **Sensor** → User → Farm → WaterRecord → NotificationSettings → UserNode → **AiRecommendation** → **CropSuggestion**
- Seed order: `seedPlans()` → `seedAiData()` → `seedAdmin()` (FK dependency)
- Seed admin: `admin@gmail.com` / `123456` (role: admin, plan: pro)
- Seed plans: free / starter / pro (3 records)
- Seed AI: 4 recommendations + 3 crop suggestions

#### `internal/models/models.go`
- **BaseModel**: `created_at`, `updated_at`, `deleted_at`, `created_by`, `updated_by`
- **DB Models**: User, Farm, Sensor, WaterRecord, NotificationSettings, UserNode, **SubscriptionPlan**, **AiRecommendation**, **CropSuggestion**
- **JSON Models**: UserJSON, FarmJSON, SensorJSON, NodeJSON, WaterRecordJSON, ...
- **Request Models**: LoginRequest, RegisterRequest (display_name support), CreateFarmRequest, ...
- **Response**: APIResponse, ErrorResponse, Pagination, PaginatedResponse

#### `internal/repository/repository.go`
- `AuthRepository`: FindByEmail, FindByID, CreateUser, CheckPassword, UpdateProfile, UpdateSubscription, **HasFarm**
- `FarmRepository`: CreateFarm, GetFarmByUserID, **GetFarmByID**, UpdateLocation, LinkSensor
- `SensorRepository`: GetNearbySensors (Haversine SQL), GetSensorLatest, GetSensorHistory
- `NotificationRepository`: GetSettings (default ถ้าไม่มี), SaveSettings (upsert)
- `NodeRepository`: GetUserNodes, AddNode (max 5), SetActiveNode (transaction), RemoveNode
- `AiRepository`: GetRecommendations, GetRecommendationByID, GetAdvisoryHistory, GetCropSuggestions (TDS filter)
- `SubscriptionPlanRepository`: GetAll, FindByID

#### `internal/service/service.go`
- `AuthService`: Login (isFirstLogin จาก HasFarm), Register (split display_name), SocialLogin (idempotent), ForgotPassword (stub)
- `AiService`: query `AiRepository` → GetRecommendations, GetRecommendationByID, GetAdvisoryHistory (7d), GetCropSuggestions (TDS filter)
- `SubscriptionService`: query `SubscriptionPlanRepository` → GetPlans, ValidatePlan

#### `internal/handlers/handlers.go`
- Helper functions: `ok`, `created`, `badRequest`, `unauthorized`, `notFound`, `serverError`
- `AuthHandler`, `FarmHandler`, `SensorHandler` (concurrent dashboard), `AccountHandler`, `AiHandler`, `NodeHandler`
- `buildDashboardSummary`: ใช้ `sync.WaitGroup` ดึง sensor + history พร้อมกัน

#### `internal/router/router.go`
- CORS: AllowOrigins=`*` (ปรับใน production), MaxAge=12h
- Groups: Public `/auth/*` | Protected `/` (JWT) | Admin `/admin` (JWT+role)

#### `internal/middleware/middleware.go`
- `AuthMiddleware`: parse JWT → inject `userID`, `email`, `role` เข้า Gin context
- `AdminMiddleware`: ตรวจ `role == "admin"` → 403 ถ้าไม่ใช่

---

## 🔄 Flow การทำงาน

```
Flutter App → HTTP Request
    ↓
Router (URL matching)
    ↓
Middleware (CORS → AuthMiddleware → AdminMiddleware)
    ↓
Handler (bind JSON, validate, call service/repo)
    ↓
Service (business logic: JWT, AI, Subscription)
    ↓
Repository (GORM queries → MySQL)
    ↓
MySQL Database
```

---

## 🏃 วิธีรันโปรเจค

```bash
# 1. Copy config
cp .env.example .env

# 2. แก้ไข DB_PASSWORD ใน .env

# 3. สร้าง DB และ seed
mysql -u root < scripts/schema.sql

# 4. รัน server
make run          # หรือ go run cmd/server/main.go

# 5. ทดสอบ
curl http://localhost:8080/health
```

### Makefile
| คำสั่ง | หน้าที่ |
|--------|--------|
| `make run` | รัน dev server |
| `make build` | Build → `bin/aquasense-api` |
| `make test` | รัน unit tests |
| `make swag` | สร้าง Swagger docs |
| `make tidy` | จัดระเบียบ go.mod/go.sum |

---

## 📝 Seed Data

- **Admin**: `admin@gmail.com` / `123456` (role: admin, plan: pro)
- **Users**: `somchai@example.com`, `malee@example.com` / `password123`
- **Sensors**: 5 ตัว (s001–s005) พร้อมพิกัด + ค่า TDS
- **Water Records**: ประวัติ 7 วัน ของ s001 + s002
- **Farm**: "แปลงนาหัวทุ่ง" 12.5 ไร่ เชื่อมกับ s001
- **Subscription Plans**: free / starter (฿59, recommended) / pro (฿199)
- **AI Recommendations**: 4 รายการ (tds_alert, crop_suggestion, tds_danger, fertilizer)
- **Crop Suggestions**: 3 รายการ (ถั่วเขียว, ข้าวโพดหวาน, ข้าว) พร้อม TDS range
