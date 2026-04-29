-- AquaSense MySQL Schema
-- Run this script to create the database and all tables.
-- Compatible with MySQL 8.0+

CREATE DATABASE IF NOT EXISTS aquasense
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_unicode_ci;

USE aquasense;

-- ─── Users table ─────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS users (
  id                VARCHAR(36)   NOT NULL PRIMARY KEY,          -- UUID v4
  first_name        VARCHAR(100)  NOT NULL,
  last_name         VARCHAR(100)  NOT NULL,
  email             VARCHAR(255)  NOT NULL UNIQUE,
  phone             VARCHAR(20)   NOT NULL DEFAULT '',
  birth_date        DATE          NOT NULL,
  password_hash     VARCHAR(255)  NOT NULL,
  subscription_plan VARCHAR(20)   NOT NULL DEFAULT 'free',       -- free | starter | pro
  avatar_url        VARCHAR(500)  NULL,
  created_at        DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at        DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  INDEX idx_users_email (email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ─── Farms table ─────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS farms (
  id                     VARCHAR(36)   NOT NULL PRIMARY KEY,
  user_id                VARCHAR(36)   NOT NULL,
  name                   VARCHAR(255)  NOT NULL,
  area_size_rai          DECIMAL(10,2) NOT NULL,
  crop_type              VARCHAR(50)   NOT NULL,                  -- rice | beans | corn | other
  yield_ton_per_rai      DECIMAL(10,3) NULL,
  avg_price_baht_per_kg  DECIMAL(10,2) NULL,
  distribution_channels  JSON          NOT NULL DEFAULT (JSON_ARRAY()),
  soil_ph                DECIMAL(4,2)  NULL,
  soil_problems          JSON          NOT NULL DEFAULT (JSON_ARRAY()),
  water_source           VARCHAR(100)  NOT NULL,
  latitude               DECIMAL(10,7) NULL,
  longitude              DECIMAL(10,7) NULL,
  active_sensor_id       VARCHAR(36)   NULL,
  created_at             DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at             DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  INDEX idx_farms_user_id (user_id),
  CONSTRAINT fk_farms_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ─── Sensors table ───────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS sensors (
  id          VARCHAR(36)   NOT NULL PRIMARY KEY,
  name        VARCHAR(255)  NOT NULL,
  latitude    DECIMAL(10,7) NOT NULL,
  longitude   DECIMAL(10,7) NOT NULL,
  status      ENUM('safe', 'warning', 'danger') NOT NULL DEFAULT 'safe',
  tds_value   DECIMAL(10,2) NOT NULL DEFAULT 0,
  temperature DECIMAL(5,2)  NULL,
  ph          DECIMAL(4,2)  NULL,
  updated_at  DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ─── Water records table ─────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS water_records (
  id            BIGINT        NOT NULL AUTO_INCREMENT PRIMARY KEY,
  sensor_id     VARCHAR(36)   NOT NULL,
  date          DATETIME      NOT NULL,
  tds           DECIMAL(10,2) NOT NULL,
  ph            DECIMAL(4,2)  NULL,
  temperature   DECIMAL(5,2)  NULL,
  soil_moisture DECIMAL(5,2)  NULL,
  status        ENUM('safe', 'warning', 'danger') NOT NULL DEFAULT 'safe',

  INDEX idx_water_records_sensor_date (sensor_id, date),
  CONSTRAINT fk_water_records_sensor FOREIGN KEY (sensor_id) REFERENCES sensors(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ─── Notification settings table ─────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS notification_settings (
  user_id            VARCHAR(36) NOT NULL PRIMARY KEY,
  push_enabled       TINYINT(1)  NOT NULL DEFAULT 1,
  tds_threshold      DECIMAL(10,2) NOT NULL DEFAULT 400.00,
  line_enabled       TINYINT(1)  NOT NULL DEFAULT 0,
  daily_summary_time VARCHAR(20) NOT NULL DEFAULT 'none',         -- none | morning | evening | both

  CONSTRAINT fk_notif_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ─── Seed data ───────────────────────────────────────────────────────────────
-- Matches the mock data in MockSensorRepository exactly.

INSERT IGNORE INTO sensors (id, name, latitude, longitude, status, tds_value, temperature, ph) VALUES
  ('s001', 'Sensor 01 — ลำคลองใหญ่',        14.8820000, 100.9940000, 'safe',    350, 28.5, 6.8),
  ('s002', 'Sensor 02 — ท่อระบายน้ำหลัก',   14.8760000, 100.9910000, 'warning', 520, 30.1, 6.2),
  ('s003', 'Sensor 03 — คลองชลประทาน',       14.8840000, 100.9960000, 'danger',  720, 31.4, 5.8),
  ('s004', 'Sensor 04 — สระเก็บน้ำฝั่งเหนือ', 14.8856000, 100.9895000, 'safe',  280, 27.8, 7.0),
  ('s005', 'Sensor 05 — แหล่งน้ำบาดาล',      14.8808000, 100.9975000, 'safe',   310, 26.9, 7.2);

-- Seed 7 days of water history for s001 (TDS base 340)
INSERT IGNORE INTO water_records (sensor_id, date, tds, ph, temperature, soil_moisture, status)
SELECT
  's001',
  DATE_SUB(NOW(), INTERVAL n DAY),
  340 + (n % 5) * 25 + (n * 7 % 60),
  6.5 + (n % 3) * 0.2,
  27.0 + (n % 4),
  55.0 + (n % 6) * 3,
  CASE
    WHEN (340 + (n % 5) * 25 + (n * 7 % 60)) > 600 THEN 'danger'
    WHEN (340 + (n % 5) * 25 + (n * 7 % 60)) > 450 THEN 'warning'
    ELSE 'safe'
  END
FROM (
  SELECT 0 AS n UNION SELECT 1 UNION SELECT 2 UNION SELECT 3
  UNION SELECT 4 UNION SELECT 5 UNION SELECT 6
) AS days;

-- Seed 7 days of water history for s002 (TDS base 510)
INSERT IGNORE INTO water_records (sensor_id, date, tds, ph, temperature, soil_moisture, status)
SELECT
  's002',
  DATE_SUB(NOW(), INTERVAL n DAY),
  510 + (n % 5) * 25 + (n * 7 % 60),
  6.5 + (n % 3) * 0.2,
  27.0 + (n % 4),
  55.0 + (n % 6) * 3,
  CASE
    WHEN (510 + (n % 5) * 25 + (n * 7 % 60)) > 600 THEN 'danger'
    WHEN (510 + (n % 5) * 25 + (n * 7 % 60)) > 450 THEN 'warning'
    ELSE 'safe'
  END
FROM (
  SELECT 0 AS n UNION SELECT 1 UNION SELECT 2 UNION SELECT 3
  UNION SELECT 4 UNION SELECT 5 UNION SELECT 6
) AS days;

-- Seed demo users (matching MockAuthRepository)
-- password_hash is bcrypt of 'password123'
INSERT IGNORE INTO users (id, first_name, last_name, email, phone, birth_date, password_hash, subscription_plan)
VALUES
  ('u001', 'สมชาย', 'ใจดี',    'somchai@example.com', '0812345678', '1985-06-15',
   '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'starter'),
  ('u002', 'มาลี',   'เกษตรดี', 'malee@example.com',   '0898765432', '1990-03-22',
   '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'pro');

-- Seed farm for u001
INSERT IGNORE INTO farms (id, user_id, name, area_size_rai, crop_type, yield_ton_per_rai,
  avg_price_baht_per_kg, distribution_channels, soil_ph, soil_problems, water_source, latitude, longitude, active_sensor_id)
VALUES (
  'f001', 'u001', 'แปลงนาหัวทุ่ง', 12.5, 'rice', 0.8, 9.5,
  '["พ่อค้าคนกลาง","สหกรณ์"]', 6.2, '["ดินเปรี้ยว"]', 'น้ำชลประทาน',
  14.88, 100.99, 's001'
);
