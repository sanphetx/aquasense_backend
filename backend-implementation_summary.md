# AquaSense Backend — สรุปสถานะการ Implement ปัจจุบัน

> **วัตถุประสงค์เอกสารนี้**: สรุปสถานะการ implement ที่เป็นจริงใน Go code ใช้อ้างอิงเป็น source of truth คู่กับ ARCHITECTURE.md
> **วันที่จัดทำ**: 12 พฤษภาคม 2026
> **สถานะ**: Production-ready (Auth, Farm, Node, AI, Profile) + Seed data (Sensor/IoT)
> **อัปเดตล่าสุด**: 12 พฤษภาคม 2026 — **v1.3**: Security Hardening + Seed Demo Users/Sensors/WaterRecords

---

## 0. สรุปสถานะ Requirement ทั้งหมด

> หมายเหตุ: ระบบที่มีเครื่องหมาย ⏳ **เรียก API ได้ปกติ** — response format ถูกต้องตาม spec แล้ว เมื่อต่อระบบจริงไม่ต้องแก้โค้ดฝั่ง Flutter

### 0.1 ระบบที่ยังไม่ implement / ยังไม่สมบูรณ์

| ระบบ | Requirement | สถานะ | หมายเหตุ |
|------|------------|--------|----------|
| **Auth** | Forgot Password — ส่ง email จริง | 🔶 Stub | Endpoint มี, handler return success mock — ยังไม่มี SMTP integration |
| **Auth** | OTP Verification (email/SMS) | ❌ ยังไม่มี | Frontend มี `OtpVerifyPage` แต่ backend ไม่มี OTP endpoint |
| **Sensor** | Real-time data จาก IoT hardware | ⏳ รอทีม IoT | API พร้อม, ข้อมูลมาจาก seed — รอ hardware ส่งค่าจริง |
| **Dashboard** | Real-time WebSocket push | ❌ ยังไม่มี | ใช้ HTTP polling (pull-to-refresh) อยู่ |
| **Notification** | Push Notification (Firebase FCM) | ❌ ยังไม่มี | Settings CRUD พร้อม, แต่ยังไม่ส่ง push จริง |
| **Notification** | LINE Notify integration | ❌ ยังไม่มี | Toggle เปิด/ปิดได้ แต่ไม่มี LINE OAuth + send |
| **Subscription** | Payment Gateway (PromptPay/Credit) | ❌ ยังไม่มี | เปลี่ยน plan ใน DB ได้ แต่ไม่มีการเก็บเงินจริง |
| **AI** | ML Model inference จริง | ❌ ยังไม่มี | ข้อมูลมาจาก seed table — ไม่มี model prediction |
| **Export** | CSV/PDF generation | ❌ ยังไม่มี | Frontend มี UI แต่ backend ไม่มี endpoint |
| **Admin** | User management (CRUD users) | ❌ ยังไม่มี | มีแค่ดู dashboard ของ user อื่น |

### 0.2 ระบบที่ implement ครบแล้ว

| ระบบ | Requirement | สถานะ |
|------|------------|--------|
| **Auth — Email/Password** | Register + Login + JWT + bcrypt | ✅ |
| **Auth — Social Login (Google)** | verify id_token ผ่าน Google tokeninfo API + audience check | ✅ |
| **Auth — Social Login (Apple)** | verify id_token ผ่าน Apple JWKS RSA + issuer/audience check | ✅ |
| **Auth — isFirstLogin** | ตรวจจาก DB ว่า user มี farm หรือไม่ → `is_first_login: true/false` | ✅ |
| **Auth — Production Guard** | JWT_SECRET ไม่ตั้งใน release mode → crash ทันที | ✅ |
| **Farm — CRUD** | สร้าง/ดู/อัปเดตพิกัด/เชื่อม sensor | ✅ |
| **Farm — IDOR Prevention** | ทุก query แนบ user_id จาก JWT | ✅ |
| **Sensor — Nearby Search** | Haversine formula SQL query + distance sort | ✅ (seed data) |
| **Sensor — Latest/History/Status** | ดึงจาก water_records + sensors table | ✅ (seed data) |
| **Dashboard — Summary** | Concurrent fetch (sync.WaitGroup) sensor + history | ✅ (seed data) |
| **Analytics — Soil Moisture** | Query water_records by period (7d/30d) | ✅ (seed data) |
| **AI — Recommendations** | CRUD จาก ai_recommendations table | ✅ (DB-backed) |
| **AI — Advisory History** | Filter ย้อนหลัง 7 วัน | ✅ (DB-backed) |
| **AI — Crop Suggestions** | Filter by TDS range (min_tds/max_tds) | ✅ (DB-backed) |
| **Node — Management** | เพิ่ม/ลบ/เปลี่ยน active (max 5, DB transaction) | ✅ |
| **Profile — View/Edit** | ดู/แก้ไข first_name, last_name, phone, avatar_url | ✅ |
| **Notification Settings** | CRUD upsert (push, LINE toggle, TDS threshold, daily_summary_time) | ✅ |
| **Subscription — Plans** | ดูแผน 3 ระดับจาก DB | ✅ (DB-backed) |
| **Subscription — Subscribe** | เปลี่ยน plan ของ user ใน DB | ✅ (ไม่มี payment) |
| **Admin — View User Dashboard** | Admin ดู summary ของ user อื่นได้ | ✅ |
| **Middleware — Auth** | JWT parse → inject userID, email, role | ✅ |
| **Middleware — Admin** | ตรวจ role == "admin" → 403 | ✅ |
| **CORS** | Configurable origins (env var) | ✅ |
| **Swagger** | Auto-generated docs at /swagger/* | ✅ |
| **Health Check** | GET /health → 200 | ✅ |
| **Graceful Shutdown** | รอ 5 วินาทีก่อนปิด | ✅ |
| **Seed Data** | Auto-seed plans, sensors, AI, users, farm, admin | ✅ |

---

## 1. สรุป API Endpoints ทั้งหมด

### Public (ไม่ต้อง Login) — 6 endpoints

| # | Method | Path | หน้าที่ | สถานะ |
|---|--------|------|--------|-------|
| 1 | `POST` | `/api/v1/auth/login` | เข้าสู่ระบบ (Email/Password) | ✅ |
| 2 | `POST` | `/api/v1/auth/register` | สมัครสมาชิก (รองรับ display_name) | ✅ |
| 3 | `POST` | `/api/v1/auth/social` | Social Login (Google / Apple) | ✅ |
| 4 | `POST` | `/api/v1/auth/forgot-password` | รีเซ็ตรหัสผ่าน | 🔶 stub |
| 5 | `GET` | `/health` | Health check | ✅ |
| 6 | `GET` | `/swagger/*` | Swagger UI | ✅ |

### Protected (ต้อง JWT) — 20 endpoints

| # | Method | Path | หน้าที่ | สถานะ |
|---|--------|------|--------|-------|
| 7 | `POST` | `/api/v1/farms` | สร้างแปลงเกษตร (upsert) | ✅ |
| 8 | `GET` | `/api/v1/farms` | ดูแปลงของตัวเอง | ✅ |
| 9 | `PUT` | `/api/v1/farms/:id/location` | อัปเดตพิกัด GPS | ✅ |
| 10 | `POST` | `/api/v1/farms/:id/sensor` | เชื่อมต่อ Sensor กับแปลง | ✅ |
| 11 | `GET` | `/api/v1/sensors/nearby?lat=&lng=` | ค้นหา Sensor ใกล้เคียง (Haversine) | ⏳ |
| 12 | `GET` | `/api/v1/sensors/:id/latest` | ค่าล่าสุดของ Sensor | ⏳ |
| 13 | `GET` | `/api/v1/sensors/:id/history?period=7d` | ประวัติ 7d/30d | ⏳ |
| 14 | `GET` | `/api/v1/sensors/:id/status` | สถานะ Sensor | ⏳ |
| 15 | `GET` | `/api/v1/dashboard/summary` | Dashboard รวม (concurrent) | 🔶 |
| 16 | `GET` | `/api/v1/analytics/soil-moisture?period=7d` | ประวัติความชื้นดิน | ⏳ |
| 17 | `GET` | `/api/v1/ai/recommendations` | คำแนะนำ AI ทั้งหมด | ✅ |
| 18 | `GET` | `/api/v1/ai/recommendations/:id` | คำแนะนำ AI ตาม ID | ✅ |
| 19 | `GET` | `/api/v1/ai/advisory-history` | ประวัติคำแนะนำ (7 วัน) | ✅ |
| 20 | `GET` | `/api/v1/ai/crop-suggestions?tds=` | แนะนำพืชตามค่า TDS | ✅ |
| 21 | `GET` | `/api/v1/users/profile` | ดูโปรไฟล์ | ✅ |
| 22 | `PUT` | `/api/v1/users/profile` | แก้ไขโปรไฟล์ | ✅ |
| 23 | `GET` | `/api/v1/users/notification-settings` | ดูตั้งค่าแจ้งเตือน | ✅ |
| 24 | `PUT` | `/api/v1/users/notification-settings` | บันทึกตั้งค่าแจ้งเตือน (upsert) | ✅ |
| 25 | `GET` | `/api/v1/subscriptions/plans` | ดูแผนสมาชิก | ✅ |
| 26 | `POST` | `/api/v1/subscriptions/subscribe` | สมัครแผน | ✅ |

### Node Management — 4 endpoints

| # | Method | Path | หน้าที่ | สถานะ |
|---|--------|------|--------|-------|
| 27 | `GET` | `/api/v1/nodes` | ดูรายการ Node ทั้งหมด | ✅ |
| 28 | `POST` | `/api/v1/nodes` | เพิ่ม Node (max 5) | ✅ |
| 29 | `PUT` | `/api/v1/nodes/:id/active` | ตั้ง Active Node (DB transaction) | ✅ |
| 30 | `DELETE` | `/api/v1/nodes/:id` | ยกเลิก Node (soft delete) | ✅ |

### Admin — 1 endpoint

| # | Method | Path | หน้าที่ | สถานะ |
|---|--------|------|--------|-------|
| 31 | `GET` | `/api/v1/admin/users/:user_id/summary` | ดู Dashboard ของ user อื่น | ✅ |

**รวม: 31 endpoints** (✅ 22 | 🔶 2 | ⏳ 5 | ❌ 2 missing from frontend needs)

---

## 2. Request Flow

```
Flutter App → HTTP Request (+ Bearer Token)
    │
    ▼
┌─────────────────────────────────────────────────┐
│ Router (router.go)                              │
│  ├── /health                → Health check      │
│  ├── /swagger/*             → Swagger UI        │
│  ├── /api/v1/auth/*         → Public            │
│  ├── /api/v1/* (protected)  → AuthMiddleware    │
│  └── /api/v1/admin/*        → AdminMiddleware   │
└─────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────┐
│ Middleware Chain                                 │
│  1. CORS (gin-contrib/cors)                     │
│  2. AuthMiddleware → parse JWT → inject context │
│     c.Set("userID", ...)                        │
│     c.Set("email", ...)                         │
│     c.Set("role", ...)                          │
│  3. AdminMiddleware → check role == "admin"     │
└─────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────┐
│ Handler (handlers.go)                           │
│  • Bind JSON / Query / Param                    │
│  • Validate input                               │
│  • Call Service or Repository                   │
│  • Map error → HTTP status (errors.Is)          │
│  • Return JSON: {success, data/error}           │
└─────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────┐
│ Service (service.go + oauth.go)                 │
│  • AuthService: JWT generate, Social verify     │
│  • AiService: query AiRepository               │
│  • SubscriptionService: validate plan           │
└─────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────┐
│ Repository (repository.go)                      │
│  • GORM queries → MySQL                         │
│  • Sentinel errors (ErrFarmNotFound, ...)       │
│  • DB Transactions (multi-table writes)         │
│  • IDOR: WHERE user_id = ? AND id = ?           │
└─────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────┐
│ MySQL 8.0+ (9 tables)                           │
│  • FK constraints (CASCADE / SET NULL)          │
│  • Soft delete (deleted_at IS NULL)             │
│  • Connection pool: 25 open, 10 idle, 5min max  │
└─────────────────────────────────────────────────┘
```

---

## 3. รายละเอียดแต่ละ Module

---

### 3.1 Auth Module

**Files**: `handlers.go` (AuthHandler), `service.go` (AuthService), `oauth.go`, `repository.go` (AuthRepository)

**Endpoints implemented**:

#### POST `/api/v1/auth/login`

| ขั้นตอน | รายละเอียด |
|---------|-----------|
| 1. Bind | `LoginRequest{email, password}` |
| 2. Service | FindByEmail → bcrypt.CompareHashAndPassword → GenerateToken |
| 3. Response | `{token, user: UserJSON, is_first_login: bool}` |
| 4. isFirstLogin | ตรวจจาก `HasFarm(userID)` — ถ้าไม่มี farm → true |

**Request/Response**:
```json
// Request
{"email": "somchai@example.com", "password": "password123"}

// Success Response (200)
{
  "success": true,
  "data": {
    "token": "eyJhbG...",
    "user": {
      "id": "u001",
      "first_name": "สมชาย",
      "last_name": "ใจดี",
      "email": "somchai@example.com",
      "subscription_plan": "starter",
      "role": "user"
    },
    "is_first_login": false
  }
}
```

#### POST `/api/v1/auth/register`

| ขั้นตอน | รายละเอียด |
|---------|-----------|
| 1. Bind | `RegisterRequest{email, password, first_name, last_name, display_name, phone, birth_date}` |
| 2. Service | Check duplicate email → bcrypt hash → CreateUser → GenerateToken |
| 3. display_name | ถ้ามี → split เป็น first_name + last_name อัตโนมัติ |
| 4. Default plan | `"free"` |
| 5. Response | `{token, user: UserJSON, is_first_login: true}` |

#### POST `/api/v1/auth/social`

| ขั้นตอน | รายละเอียด |
|---------|-----------|
| 1. Bind | `SocialLoginRequest{provider: "google"|"apple", id_token: "..."}` |
| 2. Service | switch provider → VerifyGoogleToken / VerifyAppleToken |
| 3. Google | GET `https://oauth2.googleapis.com/tokeninfo?id_token=` → check aud, email_verified |
| 4. Apple | Fetch JWKS (cached 24h) → find kid → RSA verify → check iss, aud |
| 5. FindOrCreate | ค้นหา user by email → ถ้าไม่มี → CreateUser (no password) |
| 6. Response | `{token, user: UserJSON, is_first_login: bool}` |

**Error Cases**:
| Error | HTTP Status | เหตุ |
|-------|-------------|------|
| Token verification failed | 401 | id_token ไม่ถูกต้อง / หมดอายุ |
| Invalid provider | 400 | provider ไม่ใช่ "google" หรือ "apple" |
| Email not verified (Google) | 401 | email_verified != "true" |
| Invalid audience | 401 | aud ไม่ตรงกับ CLIENT_ID |

#### POST `/api/v1/auth/forgot-password`

| สถานะ | รายละเอียด |
|-------|-----------|
| 🔶 Stub | Bind email → return success ทุกกรณี (ไม่ส่ง email จริง) |
| TODO | ต้อง integrate SMTP (SendGrid / AWS SES) |

---

### 3.2 Farm Module

**Files**: `handlers.go` (FarmHandler), `repository.go` (FarmRepository)

**Endpoints implemented**:

#### POST `/api/v1/farms` — สร้าง/อัปเดตแปลง (Upsert)

| ขั้นตอน | รายละเอียด |
|---------|-----------|
| 1. Bind | `CreateFarmRequest{name, area_size_rai, crop_type, ...}` |
| 2. IDOR | userID จาก JWT context (ห้ามจาก request body) |
| 3. Logic | ถ้า user มี farm อยู่แล้ว → update, ถ้าไม่มี → create |
| 4. Re-read | หลัง upsert → re-read จาก DB เพื่อ return ข้อมูลล่าสุด |
| 5. Response | `{farm: FarmJSON}` (201 Created / 200 OK) |

#### GET `/api/v1/farms` — ดูแปลงของตัวเอง

| ขั้นตอน | รายละเอียด |
|---------|-----------|
| 1. IDOR | `WHERE user_id = ?` (จาก JWT) |
| 2. Sort | `ORDER BY created_at DESC` |
| 3. Response | `{farms: []FarmJSON}` |

#### PUT `/api/v1/farms/:id/location` — อัปเดตพิกัด GPS

| ขั้นตอน | รายละเอียด |
|---------|-----------|
| 1. IDOR | `WHERE id = ? AND user_id = ?` |
| 2. Bind | `{latitude, longitude}` |
| 3. Response | `{farm: FarmJSON}` |

#### POST `/api/v1/farms/:id/sensor` — เชื่อมต่อ Sensor

| ขั้นตอน | รายละเอียด |
|---------|-----------|
| 1. IDOR | `WHERE id = ? AND user_id = ?` |
| 2. Validate | ตรวจว่า sensor_id มีอยู่จริงใน sensors table |
| 3. Update | `farms.active_sensor_id = sensor_id` |
| 4. Error | sensor ไม่พบ → 404 `ErrSensorNotFound` |

---

### 3.3 Sensor Module

**Files**: `handlers.go` (SensorHandler), `repository.go` (SensorRepository)

**สถานะ**: ⏳ API ทำงานได้ แต่ข้อมูลมาจาก seed (รอ IoT hardware ส่งค่าจริง)

#### GET `/api/v1/sensors/nearby?lat=&lng=`

| ขั้นตอน | รายละเอียด |
|---------|-----------|
| 1. Query params | `lat`, `lng` (float64) |
| 2. SQL | Haversine formula → sort by distance ASC |
| 3. Filter | `WHERE deleted_at IS NULL` (manual — raw SQL) |
| 4. Response | `{sensors: []SensorJSON}` พร้อม `distance_km` |

#### GET `/api/v1/sensors/:id/latest`

| Response | `{sensor: SensorJSON}` (tds_value, ph, temperature, status) |
|----------|-----------|

#### GET `/api/v1/sensors/:id/history?period=7d`

| ขั้นตอน | รายละเอียด |
|---------|-----------|
| 1. Period | `7d` (default) หรือ `30d` |
| 2. SQL | `WHERE sensor_id = ? AND date >= ? ORDER BY date DESC` |
| 3. Empty | คืน `[]` (empty slice) ไม่ใช่ `null` |

#### GET `/api/v1/sensors/:id/status`

| Response | `{status: "safe"|"warning"|"danger", tds_value: 350, ...}` |
|----------|-----------|

---

### 3.4 Dashboard Module

**Files**: `handlers.go` (SensorHandler.GetDashboardSummary)

**สถานะ**: 🔶 ทำงานได้ แต่ข้อมูลจาก seed

#### GET `/api/v1/dashboard/summary`

| ขั้นตอน | รายละเอียด |
|---------|-----------|
| 1. IDOR | อ่าน userID → GetFarmByUserID → ดึง active_sensor_id |
| 2. No farm | คืน 404 "farm not found" (ไม่ fallback) |
| 3. Concurrent | `sync.WaitGroup` ดึง sensor latest + history 7d **พร้อมกัน** |
| 4. Response | `{sensor: SensorJSON, history: []WaterRecordJSON, farm: FarmJSON}` |

---

### 3.5 AI Module

**Files**: `handlers.go` (AiHandler), `service.go` (AiService), `repository.go` (AiRepository)

**สถานะ**: ✅ DB-backed (ข้อมูลจาก ai_recommendations + crop_suggestions tables)

#### GET `/api/v1/ai/recommendations`

| Response | `{recommendations: []AiRecommendationJSON}` — ทั้งหมด sorted by created_at DESC |
|----------|-----------|

#### GET `/api/v1/ai/recommendations/:id`

| Response | `{recommendation: AiRecommendationJSON}` — รวม reason_chips, confidence_score |
|----------|-----------|
| Error | ไม่พบ → 404 |

#### GET `/api/v1/ai/advisory-history`

| Logic | `WHERE created_at >= 7 วันที่แล้ว ORDER BY created_at DESC` |
|-------|-----------|

#### GET `/api/v1/ai/crop-suggestions?tds=350`

| Logic | `WHERE min_tds <= ? AND max_tds >= ? ORDER BY sort_order` |
|-------|-----------|
| No tds param | คืนทั้งหมด |

---

### 3.6 Node Management Module

**Files**: `handlers.go` (NodeHandler), `repository.go` (NodeRepository)

**สถานะ**: ✅ Production-ready (DB Transaction, max 5 limit)

#### GET `/api/v1/nodes`

| Logic | `WHERE user_id = ? AND deleted_at IS NULL` + `Preload("Sensor")` |
|-------|-----------|
| Response | `{nodes: []NodeJSON}` (includes sensor details + is_active flag) |

#### POST `/api/v1/nodes`

| ขั้นตอน | รายละเอียด |
|---------|-----------|
| 1. Bind | `{sensor_id: "s001"}` |
| 2. Validate | sensor_id ต้องมีอยู่จริง |
| 3. Limit | count nodes ≥ 5 → 400 "maximum 5 nodes" |
| 4. Duplicate | sensor ซ้ำ → 400 "sensor already linked" |
| 5. Create | `UserNode{id, user_id, sensor_id, is_active: false}` |
| 6. Response | `{node: NodeJSON}` (201) |

#### PUT `/api/v1/nodes/:id/active`

| ขั้นตอน | รายละเอียด |
|---------|-----------|
| 1. IDOR | `WHERE id = ? AND user_id = ?` |
| 2. **Transaction** | Step 1: deactivate all → Step 2: activate target → Step 3: sync `farms.active_sensor_id` |
| 3. Atomicity | ถ้า step ใดพัง → rollback ทั้งหมด |

#### DELETE `/api/v1/nodes/:id`

| ขั้นตอน | รายละเอียด |
|---------|-----------|
| 1. IDOR | `WHERE id = ? AND user_id = ?` |
| 2. Soft delete | GORM set deleted_at |
| 3. Sync | ถ้าลบ active node → clear `farms.active_sensor_id` |

---

### 3.7 Account Module

**Files**: `handlers.go` (AccountHandler), `repository.go` (AuthRepository)

#### GET `/api/v1/users/profile`

| Response | `{user: UserJSON}` — first_name, last_name, email, phone, birth_date, avatar_url, subscription_plan |
|----------|-----------|

#### PUT `/api/v1/users/profile`

| Bind | `{first_name, last_name, phone, avatar_url}` (partial update) |
|------|-----------|
| IDOR | update WHERE id = userID (จาก JWT) |

#### GET/PUT `/api/v1/users/notification-settings`

| GET | คืน settings (ถ้าไม่มี → สร้าง default: push=true, tds=400, line=false, time=none) |
|-----|-----------|
| PUT | Upsert ด้วย `map[string]interface{}` (รองรับ false/0 values) |

---

### 3.8 Subscription Module

**Files**: `handlers.go` (AccountHandler), `service.go` (SubscriptionService), `repository.go` (SubscriptionPlanRepository)

#### GET `/api/v1/subscriptions/plans`

| Response | `{plans: [free, starter, pro]}` — sorted by sort_order |
|----------|-----------|

#### POST `/api/v1/subscriptions/subscribe`

| ขั้นตอน | รายละเอียด |
|---------|-----------|
| 1. Bind | `{plan_id: "starter"}` |
| 2. Validate | plan_id ต้องมีใน subscription_plans table |
| 3. Update | `users.subscription_plan = plan_id` |
| 4. ⚠️ No payment | เปลี่ยน plan ทันทีไม่มีการเก็บเงิน |

---

### 3.9 Admin Module

**Files**: `handlers.go` (SensorHandler.GetAdminDashboard)

#### GET `/api/v1/admin/users/:user_id/summary`

| Middleware | AuthMiddleware + AdminMiddleware (role == "admin") |
|-----------|-----------|
| Logic | เหมือน dashboard/summary แต่ใช้ target user_id จาก param |
| IDOR | ปลอดภัย — AdminMiddleware กั้นไว้ |

---

## 4. Database — ตารางและ Seed Data

### 4.1 ตารางทั้งหมด (9 tables)

| # | Table | Records (seed) | FK Parent |
|---|-------|---------------|-----------|
| 1 | `subscription_plans` | 3 (free/starter/pro) | — |
| 2 | `sensors` | 5 (s001–s005) | — |
| 3 | `users` | 3 (u001, u002, admin) | → subscription_plans |
| 4 | `farms` | 1 (f001) | → users, → sensors |
| 5 | `water_records` | 14 (7d × 2 sensors) | → sensors |
| 6 | `notification_settings` | 0 (created on demand) | → users |
| 7 | `user_nodes` | 0 (created on demand) | → users, → sensors |
| 8 | `ai_recommendations` | 4 | — |
| 9 | `crop_suggestions` | 3 | — |

### 4.2 Seed Data Summary

| ข้อมูล | ID | รายละเอียด |
|--------|-----|-----------|
| **Admin** | (auto UUID) | `admin@gmail.com` / ENV password / role: admin / plan: pro |
| **User 1** | u001 | `somchai@example.com` / password123 / plan: starter |
| **User 2** | u002 | `malee@example.com` / password123 / plan: pro |
| **Farm** | f001 | "แปลงนาหัวทุ่ง" 12.5 ไร่ / user: u001 / sensor: s001 |
| **Sensor 1** | s001 | "ลำคลองใหญ่" / TDS 350 / safe |
| **Sensor 2** | s002 | "ท่อระบายน้ำหลัก" / TDS 520 / warning |
| **Sensor 3** | s003 | "คลองชลประทาน" / TDS 720 / danger |
| **Sensor 4** | s004 | "สระเก็บน้ำฝั่งเหนือ" / TDS 280 / safe |
| **Sensor 5** | s005 | "แหล่งน้ำบาดาล" / TDS 310 / safe |
| **Water Records** | (auto) | 7 วัน × s001 + s002 (TDS varies) |
| **AI Recs** | ai001–ai004 | tds_alert, crop_suggestion, tds_danger, fertilizer |
| **Crops** | crop001–003 | ถั่วเขียว, ข้าวโพดหวาน, ข้าว |
| **Plans** | free/starter/pro | ฟรี / ฿59/ฤดูกาล / ฿199/ปี |

### 4.3 Seed Order (FK dependency)

```
seedPlans()      →  3 subscription plans (free, starter, pro)
    │
    ▼
seedSensors()    →  5 sensors + seedWaterRecords() (14 records)
    │
    ▼
seedAiData()     →  4 AI recommendations + 3 crop suggestions
    │
    ▼
seedDemoUsers()  →  2 users (u001, u002) + 1 farm (f001)
    │
    ▼
seedAdmin()      →  1 admin user (admin@gmail.com)
```

---

## 5. Security Implementation

### 5.1 Authentication Flow

```
┌─ Email/Password Login ─────────────────────────────────────────────┐
│  email + password → bcrypt verify → JWT (HS256, 24h) → response   │
└───────────────────────────────────────────────────────────────────┘

┌─ Google Social Login ──────────────────────────────────────────────┐
│  id_token → GET googleapis.com/tokeninfo → check:                 │
│    ✓ email_verified == "true"                                     │
│    ✓ aud == GOOGLE_CLIENT_ID                                      │
│    ✓ email != ""                                                  │
│  → FindOrCreate user → JWT → response                            │
└───────────────────────────────────────────────────────────────────┘

┌─ Apple Social Login ───────────────────────────────────────────────┐
│  id_token → fetch Apple JWKS (cached 24h) → match kid →           │
│  RSA verify signature → check:                                    │
│    ✓ iss == "https://appleid.apple.com"                           │
│    ✓ aud == APPLE_CLIENT_ID                                       │
│    ✓ exp > now                                                    │
│  → FindOrCreate user → JWT → response                            │
└───────────────────────────────────────────────────────────────────┘
```

### 5.2 IDOR Prevention (ทุก Protected endpoint)

| Pattern | ตัวอย่าง |
|---------|---------|
| Get own resource | `WHERE user_id = ?` (userID จาก JWT) |
| Get specific resource | `WHERE id = ? AND user_id = ?` (ห้ามแค่ id) |
| Admin override | ผ่าน AdminMiddleware แล้วใช้ target user_id จาก param |

### 5.3 Production Guards

| Guard | Trigger | ผลลัพธ์ |
|-------|---------|---------|
| JWT_SECRET empty + release mode | Server startup | `log.Fatal` → crash |
| CORS_ALLOWED_ORIGINS=* + release | ⚠️ Warning เท่านั้น | ต้อง manual แก้ |
| ADMIN_DEFAULT_PASSWORD empty | Seed time | Generate random + log warn |

---

## 6. Dependency Injection — Wiring Order

```go
// cmd/server/main.go — ลำดับ DI (ห้ามสลับ)

// 1. Config
cfg := config.Load()

// 2. Database
db := database.Connect(cfg.DSN(), true, cfg.AdminDefaultPassword)

// 3. Repositories
authRepo    := repository.NewAuthRepository(db)
farmRepo    := repository.NewFarmRepository(db)
sensorRepo  := repository.NewSensorRepository(db)
notifRepo   := repository.NewNotificationRepository(db)
nodeRepo    := repository.NewNodeRepository(db)
aiRepo      := repository.NewAiRepository(db)
planRepo    := repository.NewSubscriptionPlanRepository(db)

// 4. Services
authService := service.NewAuthService(authRepo, cfg.JWTSecret, cfg.JWTExpireHours,
                                       cfg.GoogleClientID, cfg.AppleClientID)
aiService   := service.NewAiService(aiRepo)
subService  := service.NewSubscriptionService(planRepo)

// 5. Handlers
authHandler    := handlers.NewAuthHandler(authService)
farmHandler    := handlers.NewFarmHandler(farmRepo)
sensorHandler  := handlers.NewSensorHandler(sensorRepo, farmRepo)
accountHandler := handlers.NewAccountHandler(authRepo, notifRepo, subService)
aiHandler      := handlers.NewAiHandler(aiService)
nodeHandler    := handlers.NewNodeHandler(nodeRepo, farmRepo)

// 6. Router
r := router.Setup(authHandler, farmHandler, sensorHandler,
                   accountHandler, aiHandler, nodeHandler, cfg)
```

---

## 7. Libraries & Versions

| Library | Version | หน้าที่ |
|---------|---------|--------|
| `gin-gonic/gin` | v1.12.0 | HTTP framework |
| `gin-contrib/cors` | v1.7.7 | CORS middleware |
| `gorm.io/gorm` | v1.31.1 | ORM |
| `gorm.io/driver/mysql` | v1.6.0 | MySQL driver |
| `golang-jwt/jwt/v5` | v5.2.1 | JWT Token |
| `google/uuid` | v1.6.0 | UUID v4 Primary Key |
| `joho/godotenv` | v1.5.1 | .env loader |
| `go.uber.org/zap` | v1.28.0 | Structured logger |
| `golang.org/x/crypto` | v0.50.0 | bcrypt |
| `swaggo/swag` | v1.16.6 | Swagger docs |
| `swaggo/gin-swagger` | v1.6.1 | Swagger UI |

---

## 8. Error Handling — Sentinel Errors ที่กำหนดไว้

| Error Variable | HTTP Status | ใช้ที่ |
|---------------|-------------|-------|
| `ErrFarmNotFound` | 404 | GetFarm, UpdateLocation, LinkSensor |
| `ErrSensorNotFound` | 404 | LinkSensor, AddNode |
| `ErrNodeNotFound` | 404 | SetActiveNode, RemoveNode |
| `ErrMaxNodesReached` | 400 | AddNode (≥ 5) |
| `ErrSensorAlreadyLinked` | 400 | AddNode (duplicate) |
| `ErrInvalidPlan` | 400 | Subscribe (plan not found) |
| `ErrEmailDuplicate` | 400 | Register |
| `ErrInvalidCredentials` | 401 | Login |

---

## 9. Frontend ↔ Backend Mapping

> เปรียบเทียบ Frontend pages กับ Backend endpoints ที่ต้องเรียก

| Frontend Page | Backend Endpoint(s) | สถานะ Backend |
|---------------|--------------------|----|
| SplashPage (session check) | ไม่ต้องเรียก (token อยู่ใน local) | — |
| LoginPage | `POST /auth/login` | ✅ |
| LoginPage (Google) | `POST /auth/social` (provider: google) | ✅ |
| LoginPage (Apple) | `POST /auth/social` (provider: apple) | ✅ |
| RegisterPage | `POST /auth/register` | ✅ |
| OtpVerifyPage | ❌ **ไม่มี endpoint** | ❌ ต้อง implement |
| LocationSetupPage | `PUT /farms/:id/location` + `GET /sensors/nearby` | ✅ |
| LocationSetupPage (GPS) | Client-side only (geolocator) | — |
| DashboardPage | `GET /dashboard/summary` | 🔶 seed |
| DashboardPage (AI cards) | `GET /ai/recommendations` | ✅ |
| DashboardPage (Crop Planner) | `GET /ai/crop-suggestions` | ✅ |
| FarmLocationPage | `GET /sensors/nearby` + `GET /sensors/:id/latest` | ⏳ seed |
| FarmLocationPage (connect) | `POST /farms/:id/sensor` | ✅ |
| WaterAnalyticsPage | `GET /sensors/:id/history` + `GET /analytics/soil-moisture` | ⏳ seed |
| WaterAnalyticsPage (export) | ❌ **ไม่มี endpoint** | ❌ ต้อง implement |
| NotificationsPage | `GET/PUT /users/notification-settings` | ✅ |
| AccountPage (profile) | `GET/PUT /users/profile` | ✅ |
| AccountPage (SSO linking) | ❌ **ไม่มี endpoint** | ❌ ต้อง implement |
| SubscriptionPage | `GET /subscriptions/plans` + `POST /subscriptions/subscribe` | ✅ (no payment) |
| AdvisoryDetailPage | `GET /ai/recommendations/:id` | ✅ |
| NodeManagementPage | `GET/POST/PUT/DELETE /nodes/*` | ✅ |
| PlantingPlanPage | `GET /ai/crop-suggestions` + `GET /ai/recommendations` | ✅ |

---

## 10. สิ่งที่ต้อง Implement ต่อ (Priority)

| # | รายการ | Priority | ผลกระทบ Frontend |
|---|--------|----------|-----------------|
| 1 | OTP Verification endpoint | 🔴 High | `OtpVerifyPage` เรียกไม่ได้ |
| 2 | SSO Linking endpoint (ผูก Google/Apple เพิ่ม) | 🔴 High | AccountPage "บัญชีที่เชื่อมต่อ" ไม่ทำงาน |
| 3 | Forgot Password (SMTP) | 🟡 Medium | dialog กรอก email ยัง mock |
| 4 | Export CSV/PDF endpoint | 🟡 Medium | WaterAnalyticsPage Export ไม่ทำงาน |
| 5 | Push Notification (FCM) | 🟡 Medium | toggle เปิด/ปิดได้แต่ไม่ส่งจริง |
| 6 | LINE Notify integration | 🟠 Low | toggle mock |
| 7 | Payment Gateway | 🟠 Low | Subscription เปลี่ยนฟรีอยู่ |
| 8 | Real-time WebSocket | 🟠 Low | ใช้ pull-to-refresh ไปก่อน |
| 9 | AI ML Model inference | 🟠 Low | ข้อมูลจาก seed table ยังใช้ได้ |
