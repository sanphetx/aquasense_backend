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

- **Go 1.22+** — `brew install go`
- **MySQL 8.0+** — `brew install mysql`

---

## Quick Start

### 1. Install Go
```bash
brew install go
```

### 2. Setup MySQL
```bash
# Start MySQL
brew services start mysql

# Create database and tables (with seed data)
mysql -u root -p < scripts/schema.sql
```

### 3. Configure Environment
```bash
cp .env.example .env
# Edit .env with your MySQL password and JWT secret
```

### 4. Install Go dependencies
```bash
go mod tidy
```

### 5. Run the server
```bash
go run ./cmd/server
```

Server starts on `http://localhost:8080`

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

In your Flutter app's `env_dev.dart`, update `apiBaseUrl`:

```dart
class DevEnv implements EnvConfig {
  @override String get apiBaseUrl => 'http://localhost:8080/api/v1';
}
```

> On Android emulator use `http://10.0.2.2:8080/api/v1`
> On iOS simulator use `http://localhost:8080/api/v1`

---

## Seed Accounts

| Email | Password | Plan |
|-------|----------|------|
| `test@gmail.com` | `12345za` | member |
| `admin@gmail.com` | `123456` | admin |
