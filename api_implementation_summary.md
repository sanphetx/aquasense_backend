# AquaSense Backend — API Implementation Summary

> **วัตถุประสงค์เอกสารนี้**: สรุปสถานะการ implement ที่เป็นจริงในโค้ด — source of truth สำหรับ backend team  
> ออกแบบคู่กับ `app_implementation_summary.md` ของ Flutter — ชื่อ + format เดียวกัน แต่ track layer Go backend  
> **API Reference สำหรับ Flutter team**: Swagger UI `http://localhost:8080/swagger/index.html` — Section 8 ของไฟล์นี้คือ Feature Status สำหรับ frontend planning  
> **อัปเดตล่าสุด**: 12 พฤษภาคม 2026 — **v1.2**: Data Migration Static → MySQL (AI recommendations, Subscription Plans, Crop Suggestions ย้ายจาก hard-coded Go → DB tables จริง)

---

## 0. สรุปสถานะ Endpoint ทั้งหมด

| สัญลักษณ์ | ความหมาย |
|----------|---------|
| ✅ | implement ครบ + ทดสอบได้จริง |
| 🔶 | endpoint มีแต่ข้อมูลเป็น seed / mock / stub |
| ⏳ | รอ dependency ภายนอก (IoT team, Payment, SMTP) |
| ❌ | ยังไม่ implement เลย |

### 0.1 Auth Endpoints

| Method | Path | Handler | สถานะ | หมายเหตุ |
|--------|------|---------|-------|---------|
| `POST` | `/api/v1/auth/login` | `AuthHandler.Login` | ✅ | JWT + bcrypt |
| `POST` | `/api/v1/auth/register` | `AuthHandler.Register` | ✅ | display_name split, isFirstLogin |
| `POST` | `/api/v1/auth/social` | `AuthHandler.SocialLogin` | ✅ | Google JWKS + Apple RSA verify จริง |
| `POST` | `/api/v1/auth/forgot-password` | `AuthHandler.ForgotPassword` | 🔶 | response สำเร็จแต่ไม่ส่ง email (TODO: SMTP) |

### 0.2 Farm Endpoints (Protected)

| Method | Path | Handler | สถานะ | หมายเหตุ |
|--------|------|---------|-------|---------|
| `POST` | `/api/v1/farms` | `FarmHandler.CreateFarm` | ✅ | Upsert — 1 user 1 farm, re-read จาก DB หลัง update |
| `GET` | `/api/v1/farms` | `FarmHandler.GetFarm` | ✅ | ORDER BY created_at DESC |
| `PUT` | `/api/v1/farms/:id/location` | `FarmHandler.UpdateLocation` | ✅ | validate lat/lng range |
| `POST` | `/api/v1/farms/:id/sensor` | `FarmHandler.LinkSensor` | ✅ | Anti-IDOR: check user_id + farm_id |

### 0.3 Sensor Endpoints (Protected)

| Method | Path | Handler | สถานะ | หมายเหตุ |
|--------|------|---------|-------|---------|
| `GET` | `/api/v1/sensors/nearby` | `SensorHandler.GetNearbySensors` | 🔶 | ข้อมูล seed — รอ IoT real data |
| `GET` | `/api/v1/sensors/:id/latest` | `SensorHandler.GetSensorLatest` | 🔶 | ข้อมูล seed |
| `GET` | `/api/v1/sensors/:id/history` | `SensorHandler.GetSensorHistory` | 🔶 | ข้อมูล seed, period 7d/30d |
| `GET` | `/api/v1/sensors/:id/status` | `SensorHandler.GetSensorStatus` | 🔶 | ข้อมูล seed |

### 0.4 Node Endpoints (Protected)

| Method | Path | Handler | สถานะ | หมายเหตุ |
|--------|------|---------|-------|---------|
| `GET` | `/api/v1/nodes` | `NodeHandler.GetNodes` | ✅ | Preload("Sensor") — ไม่มี N+1 |
| `POST` | `/api/v1/nodes` | `NodeHandler.AddNode` | ✅ | max 5 nodes, duplicate check, DB Transaction |
| `PUT` | `/api/v1/nodes/:id/active` | `NodeHandler.SetActiveNode` | ✅ | DB Transaction — sync active_sensor_id ใน farm |
| `DELETE` | `/api/v1/nodes/:id` | `NodeHandler.RemoveNode` | ✅ | clear active_sensor_id ถ้าลบ active node |

### 0.5 Dashboard Endpoints (Protected)

| Method | Path | Handler | สถานะ | หมายเหตุ |
|--------|------|---------|-------|---------|
| `GET` | `/api/v1/dashboard/summary` | `SensorHandler.GetDashboardSummary` | 🔶 | concurrent fetch, seed sensor data |
| `GET` | `/api/v1/analytics/soil-moisture` | `SensorHandler.GetSoilMoistureHistory` | 🔶 | seed data |

### 0.6 AI Endpoints (Protected)

| Method | Path | Handler | สถานะ | หมายเหตุ |
|--------|------|---------|-------|---------|
| `GET` | `/api/v1/ai/recommendations` | `AiHandler.GetRecommendations` | ✅ | DB-backed (ai_recommendations table) |
| `GET` | `/api/v1/ai/recommendations/:id` | `AiHandler.GetRecommendationDetail` | ✅ | DB-backed |
| `GET` | `/api/v1/ai/advisory-history` | `AiHandler.GetAdvisoryHistory` | ✅ | DB-backed |
| `GET` | `/api/v1/ai/crop-suggestions` | `AiHandler.GetCropSuggestions` | ✅ | filter by TDS range จาก crop_suggestions table |

### 0.7 Account Endpoints (Protected)

| Method | Path | Handler | สถานะ | หมายเหตุ |
|--------|------|---------|-------|---------|
| `GET` | `/api/v1/users/profile` | `AccountHandler.GetProfile` | ✅ | |
| `PUT` | `/api/v1/users/profile` | `AccountHandler.UpdateProfile` | ✅ | |
| `GET` | `/api/v1/users/notification-settings` | `AccountHandler.GetNotificationSettings` | ✅ | |
| `PUT` | `/api/v1/users/notification-settings` | `AccountHandler.SaveNotificationSettings` | ✅ | Upsert, map แทน struct (fix false/0 ignored) |
| `GET` | `/api/v1/subscriptions/plans` | `AccountHandler.GetSubscriptionPlans` | ✅ | DB-backed (subscription_plans table) |
| `POST` | `/api/v1/subscriptions/subscribe` | `AccountHandler.Subscribe` | 🔶 | DB update user.subscription_plan, ไม่มี payment |

### 0.8 Admin Endpoints (Protected + Admin Role)

| Method | Path | Handler | สถานะ | หมายเหตุ |
|--------|------|---------|-------|---------|
| `GET` | `/api/v1/admin/users/:user_id/summary` | `SensorHandler.GetUserDashboardSummary` | ✅ | admin ดู dashboard ของ user อื่นได้ |

---

## 1. Repository Layer

### 1.1 AuthRepository

| Method | สถานะ | หมายเหตุ |
|--------|-------|---------|
| `FindByEmail(email)` | ✅ | case-insensitive |
| `FindByID(id)` | ✅ | |
| `CreateUser(req)` | ✅ | bcrypt hash, UUID v4 |
| `UpdateUser(id, updates)` | ✅ | |
| `CheckPassword(hash, plain)` | ✅ | bcrypt.CompareHashAndPassword |
| `FindByProviderID(providerUserID)` | ✅ | Social login lookup |
| `HasFarm(userID)` | ✅ | ใช้ตรวจ isFirstLogin |

### 1.2 FarmRepository

| Method | สถานะ | หมายเหตุ |
|--------|-------|---------|
| `CreateOrUpdateFarm(userID, req)` | ✅ | Upsert + re-read จาก DB (fix stale data) |
| `GetFarmByUserID(userID)` | ✅ | ORDER BY created_at DESC |
| `UpdateFarmLocation(farmID, userID, lat, lng)` | ✅ | Anti-IDOR |
| `LinkSensor(farmID, userID, sensorID)` | ✅ | Anti-IDOR, validate sensor exists |

### 1.3 SensorRepository

| Method | สถานะ | หมายเหตุ |
|--------|-------|---------|
| `GetNearbySensors(lat, lng)` | 🔶 | WHERE deleted_at IS NULL, seed data |
| `GetSensorByID(id)` | 🔶 | seed data |
| `GetSensorHistory(id, days)` | 🔶 | seed data, empty slice (ไม่ใช่ null) |
| `GetSensorStatus(id)` | 🔶 | seed data |

### 1.4 NodeRepository

| Method | สถานะ | หมายเหตุ |
|--------|-------|---------|
| `GetUserNodes(userID)` | ✅ | Preload("Sensor") |
| `AddNode(userID, sensorID)` | ✅ | DB Transaction, duplicate + capacity check |
| `SetActiveNode(nodeID, userID)` | ✅ | DB Transaction — deactivate others + sync farm.active_sensor_id |
| `RemoveNode(nodeID, userID)` | ✅ | clear active_sensor_id ถ้าลบ active node |

### 1.5 AiRepository

| Method | สถานะ | หมายเหตุ |
|--------|-------|---------|
| `GetRecommendations()` | ✅ | DB-backed |
| `GetRecommendationByID(id)` | ✅ | DB-backed |
| `GetAdvisoryHistory()` | ✅ | DB-backed |
| `GetCropSuggestions(tds, soilPH)` | ✅ | filter by min_tds/max_tds range |

### 1.6 SubscriptionPlanRepository

| Method | สถานะ | หมายเหตุ |
|--------|-------|---------|
| `GetAll()` | ✅ | ORDER BY sort_order |
| `FindByID(id)` | ✅ | |

### 1.7 AccountRepository (ใน AuthRepository)

| Method | สถานะ | หมายเหตุ |
|--------|-------|---------|
| `GetNotificationSettings(userID)` | ✅ | |
| `SaveNotificationSettings(userID, settings)` | ✅ | Upsert ด้วย map (fix false/0 ignored) |
| `Subscribe(userID, planID)` | 🔶 | อัปเดต subscription_plan column เท่านั้น |

---

## 2. Service Layer

| Service | สถานะ | หมายเหตุ |
|---------|-------|---------|
| `AuthService` | ✅ | Login, Register, SocialLogin, ForgotPassword (stub), JWT generate/verify |
| `AiService` | ✅ | Delegate ไป AiRepository (ย้ายจาก static data → DB ใน v1.2) |
| `SubscriptionService` | ✅ | Delegate ไป SubscriptionPlanRepository (ย้ายจาก static data → DB ใน v1.2) |
| `oauth.VerifyGoogleToken()` | ✅ | call Google tokeninfo API จริง + ตรวจ audience |
| `oauth.VerifyAppleToken()` | ✅ | verify RSA signature ผ่าน Apple JWKS |

---

## 3. Database Tables

| Table | สถานะ | หมายเหตุ |
|-------|-------|---------|
| `users` | ✅ | UUID PK, FK → subscription_plans (ON DELETE SET NULL) |
| `farms` | ✅ | FK → users (CASCADE), FK → sensors (SET NULL) |
| `sensors` | ✅ | status ENUM safe/warning/danger |
| `water_records` | ✅ | FK → sensors (CASCADE), index on date |
| `notification_settings` | ✅ | PK = user_id (1:1 กับ users) |
| `user_nodes` | ✅ | uniqueIndex(user_id, sensor_id), max 5 per user |
| `subscription_plans` | ✅ | seed: free / starter / pro |
| `ai_recommendations` | ✅ | seed: 4 recommendations + reason_chips JSON |
| `crop_suggestions` | ✅ | seed: 3 crops + min_tds/max_tds range |

---

## 4. Sentinel Errors (ใน repository.go)

| Error | HTTP Status | ใช้ใน |
|-------|------------|-------|
| `ErrFarmNotFound` | 404 | FarmHandler, SensorHandler |
| `ErrNodeNotFound` | 404 | NodeHandler |
| `ErrSensorNotFound` | 404 | FarmHandler, NodeHandler |
| `ErrNodeDuplicate` | 409 | NodeHandler |
| `ErrNodeCapacity` | 400 | NodeHandler |

---

## 5. Middleware

| Middleware | สถานะ | หมายเหตุ |
|-----------|-------|---------|
| `AuthMiddleware(secret)` | ✅ | inject `user_id` + `role` + `email` ใน Gin context |
| `AdminMiddleware()` | ✅ | ตรวจ role == "admin" จาก context |

---

## 6. Feature ที่ยังไม่มี (Backlog)

| Feature | สถานะ | เหตุที่ยัง block |
|---------|-------|----------------|
| Email SMTP (Forgot Password) | ❌ | ยังไม่มี SMTP service config |
| IoT Real Sensor Data | ⏳ | รอทีม hardware ส่ง data จริง |
| Payment Gateway | ❌ | ยังไม่ได้ integrate ใด |
| Push Notification (Firebase) | ❌ | ยังไม่ได้ setup FCM |
| LINE Notify | ❌ | ยังไม่ได้ integrate LINE API |
| Swagger Annotations | ✅ | `docs/swagger.yaml` generate ครบ — run `make swag` เมื่อแก้ handler |

---

## 8. Feature Status สำหรับ Frontend Planning

> Flutter team ใช้ section นี้ประกอบการตัดสินใจ implement  
> ดู request/response shape ครบ: **Swagger UI** `http://localhost:8080/swagger/index.html`

| Feature | Status | หมายเหตุ |
|---------|--------|--------|
| Auth Email/Password | ✅ Ready | |
| Auth Google / Apple | ✅ Ready | ใช้ `id_token` (ไม่ใช่ `access_token`) |
| Auth Forgot Password | 🔶 Stub | response สำเร็จแต่ไม่ส่ง email จริง |
| Farm CRUD | ✅ Ready | |
| Node Management | ✅ Ready | max 5 nodes |
| Sensor Data | 🔶 Seed data | รอ IoT team — API format ถูกต้องแล้ว |
| Dashboard | 🔶 Seed data | ดึงข้อมูลได้ แต่ยังเป็น mock sensor |
| AI Recommendations | ✅ Ready | ข้อมูลจาก DB จริง |
| Subscription Plans | 🔶 No payment | สมัครได้แต่ไม่มีจ่ายเงินจริง |
| Notification Settings | ✅ Ready | |
| Push Notification | ❌ Not implemented | |
| LINE Notify | ❌ Not implemented | |

> 🔶 หมายความว่า **เรียก API ได้ปกติ** response format ถูกต้อง — ไม่ต้องแก้โค้ด Flutter เมื่อต่อระบบจริง

---

## 7. Changelog

### v1.3 — 12 พฤษภาคม 2026 (Swagger + Vibe Coding Environment)
- เพิ่ม Swagger annotations ครบทุก handler (20 endpoints) — `make swag` generate `docs/swagger.yaml` สมบูรณ์
- สร้าง `.github/copilot-instructions.md`, `prompts/endpoint-implementation.prompt.md` สำหรับ vibe coding
- ย้าย Feature Status และ Flutter Integration Notes จาก `FRONTEND_CONTRACT.md` → `api_implementation_summary.md` + `README.md`
- `FRONTEND_CONTRACT.md` และ `contract-sync.prompt.md` ถูกย้าย — Swagger UI ทำหน้าที่แทน

### v1.2 — 12 พฤษภาคม 2026 (Data Migration: Static → MySQL)
- ย้าย AI recommendations, subscription plans, crop suggestions จาก hard-coded → DB tables จริง
- เพิ่ม 3 tables: `subscription_plans`, `ai_recommendations`, `crop_suggestions`
- เพิ่ม FK: `users.subscription_plan → subscription_plans.id` (ON DELETE SET NULL)
- Bug fixes: stale CreateFarm data, false/0 ignored in SaveSettings, N+1 GetUserNodes, null vs [] GetSensorHistory, stale active_sensor_id after RemoveNode

### v1.1 — 11 พฤษภาคม 2026 (Initial Architecture)
- ระบบ Auth ครบ: Email/Password + Google + Apple Social Login
- Farm, Sensor, Node Management, Dashboard, AI, Subscription, Account
- Security: Anti-IDOR, DB Transactions, Sentinel Errors, JWT fail-fast ใน production
- Layered Architecture: Handler → Service → Repository อย่างเคร่งครัด
