# AquaSense Go Backend

REST API backend สำหรับ AquaSense TDS Flutter app เขียนด้วย **Go + Gin + MySQL**

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.22 |
| HTTP Framework | [Gin](https://github.com/gin-gonic/gin) |
| Database | MySQL 8.0 |
| Auth | JWT (HS256) + bcrypt |
| Driver | go-sql-driver/mysql |

---

## Project Structure

```
aquasense-go-backend/
├── cmd/server/main.go            # Entry point — wires all layers + registers routes
├── internal/
│   ├── config/config.go          # Load config from .env / env vars
│   ├── database/database.go      # MySQL connection pool
│   ├── middleware/middleware.go   # JWT auth + CORS
│   ├── models/models.go          # All data models (match Flutter fromJson/toJson)
│   ├── repository/repository.go  # DB access layer (Auth, Farm, Sensor, Notification)
│   ├── service/service.go        # Business logic (AuthService, AiService, SubscriptionService)
│   └── handlers/handlers.go      # HTTP handlers (1 handler per endpoint group)
├── scripts/
│   └── schema.sql                # MySQL schema + seed data
├── go.mod
├── .env.example
└── README.md
```

---

## Prerequisites

- **Go 1.25+**
- **MySQL 8.0+**

---

## Quick Start

### macOS

```bash
brew install go mysql
brew services start mysql
mysql -u root -p < scripts/schema.sql
cp .env.example .env   # แก้ไข DB_PASSWORD และ JWT_SECRET
go mod tidy
make run
```

### Windows

```powershell
# 1. ติดตั้ง Go จาก https://go.dev/dl/ และ MySQL จาก https://dev.mysql.com/downloads/installer/
# 2. สร้าง DB และ seed ข้อมูล
mysql -u root -p < scripts\schema.sql
# 3. ตั้งค่า environment
copy .env.example .env   # แก้ไข DB_PASSWORD และ JWT_SECRET
# 4. Run
go mod tidy
go run cmd/server/main.go
```

### Docker (MySQL only)

```bash
docker run --name aquasense-mysql \
  -e MYSQL_ROOT_PASSWORD=root \
  -e MYSQL_DATABASE=aquasense \
  -p 3307:3306 \
  -d mysql:8.0

mysql -h 127.0.0.1 -P 3307 -u root -p < scripts/schema.sql
cp .env.example .env  # DB_PORT=3307
make run
```

Server starts on `http://localhost:8080`  
Swagger UI: `http://localhost:8080/swagger/index.html` (dev mode only)

---

## API Endpoints

All protected endpoints require `Authorization: Bearer <token>` header.

### Auth (Public)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/auth/login` | Login with email/password |
| `POST` | `/api/v1/auth/register` | Register new user |
| `POST` | `/api/v1/auth/social` | Social login (Google/Facebook) |

### Farm (Protected)

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/v1/farms` | Create farm (onboarding step 3) |
| `GET`  | `/api/v1/farms` | Get user's farm |
| `PUT`  | `/api/v1/farms/:id/location` | Update farm GPS location |
| `POST` | `/api/v1/farms/:id/sensor` | Link sensor to farm |

### Sensor (Protected)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/sensors/nearby?lat=&lng=` | Get nearby sensors sorted by distance |
| `GET` | `/api/v1/sensors/:id/latest` | Get latest sensor reading |
| `GET` | `/api/v1/sensors/:id/history?period=7d\|30d` | Get water history |
| `GET` | `/api/v1/sensors/:id/status` | Get sensor status |

### Node Management (Protected)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/nodes` | Get all nodes for current user |
| `POST` | `/api/v1/nodes` | Add a new node (max 5) |
| `PUT` | `/api/v1/nodes/:id/active` | Set node as active (used in dashboard) |
| `DELETE` | `/api/v1/nodes/:id` | Remove a node |

### Dashboard (Protected)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/dashboard/summary` | Full dashboard data |
| `GET` | `/api/v1/analytics/soil-moisture?sensor_id=&period=` | Soil moisture history |

### AI (Protected)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/ai/recommendations` | All AI recommendations |
| `GET` | `/api/v1/ai/recommendations/:id` | Single recommendation detail |
| `GET` | `/api/v1/ai/advisory-history` | Historical advisories |
| `GET` | `/api/v1/ai/crop-suggestions?tds=&soil_ph=` | Crop suggestions by TDS |

### Account (Protected)

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/v1/users/profile` | Get user profile |
| `PUT` | `/api/v1/users/profile` | Update profile |
| `GET` | `/api/v1/users/notification-settings` | Get notification settings |
| `PUT` | `/api/v1/users/notification-settings` | Save notification settings |
| `GET` | `/api/v1/subscriptions/plans` | Get subscription plans |
| `POST` | `/api/v1/subscriptions/subscribe` | Subscribe to plan |

---

## Example: Login

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"somchai@example.com","password":"password123"}'
```

Response:
```json
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": "u001",
      "first_name": "สมชาย",
      "last_name": "ใจดี",
      "email": "somchai@example.com",
      "subscription_plan": "starter"
    }
  }
}
```

## Example: Get Dashboard

```bash
curl http://localhost:8080/api/v1/dashboard/summary \
  -H "Authorization: Bearer <token>"
```

---

## Connecting Flutter to This Backend

ดู **Swagger UI** สำหรับ request/response shape ครบทุก endpoint: `http://localhost:8080/swagger/index.html`

```dart
// Dev — ใส่ใน env_dev.dart
const baseUrl = 'http://10.0.2.2:8080/api/v1'; // Android Emulator
// const baseUrl = 'http://localhost:8080/api/v1'; // iOS Simulator

// Auth header
headers: {
  'Content-Type': 'application/json',
  'Authorization': 'Bearer $token',
}

// is_first_login flow
if (authResponse.isFirstLogin) {
  // navigate to Onboarding/Farm Setup (user has no farm yet)
} else {
  // navigate to Dashboard
}
```

> **Social Login**: ส่ง `id_token` (ไม่ใช่ `access_token`) จาก Google Sign-In SDK / Apple Sign-In SDK

---

## Seed Accounts

| Email | Password | Plan |
|-------|----------|------|
| `test@gmail.com` | `12345za` | member |
| `admin@gmail.com` | `123456` | admin |
