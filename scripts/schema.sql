-- AquaSense MySQL Schema
-- Run this script to create the database and all tables.
-- Compatible with MySQL 8.0+

SET NAMES utf8mb4;
SET CHARACTER SET utf8mb4;

CREATE DATABASE IF NOT EXISTS aquasense
  CHARACTER SET utf8mb4
  COLLATE utf8mb4_unicode_ci;

USE aquasense;

-- ─── Subscription plans table (must be before users — FK dependency) ─────────

CREATE TABLE IF NOT EXISTS subscription_plans (
  id          VARCHAR(36)   NOT NULL PRIMARY KEY,
  name        VARCHAR(50)   NOT NULL,
  price       VARCHAR(50)   NOT NULL,
  period      VARCHAR(50)   NOT NULL DEFAULT '',
  features    JSON          NOT NULL DEFAULT (JSON_ARRAY()),
  recommended TINYINT(1)    NOT NULL DEFAULT 0,
  sort_order  INT           NOT NULL DEFAULT 0
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ─── Users table ─────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS users (
  id                VARCHAR(36)   NOT NULL PRIMARY KEY,          -- UUID v4
  first_name        VARCHAR(100)  NOT NULL,
  last_name         VARCHAR(100)  NOT NULL,
  email             VARCHAR(255)  NOT NULL UNIQUE,
  phone             VARCHAR(20)   NOT NULL DEFAULT '',
  birth_date        DATE          NOT NULL,
  password_hash     VARCHAR(255)  NOT NULL,
  subscription_plan VARCHAR(50)   NULL DEFAULT 'free',            -- FK → subscription_plans
  avatar_url        VARCHAR(500)  NULL,
  role              VARCHAR(20)   NOT NULL DEFAULT 'user',        -- user | admin
  created_at        DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at        DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at        DATETIME      NULL,                          -- soft-delete (GORM BaseModel)
  created_by        VARCHAR(36)   NULL,
  updated_by        VARCHAR(36)   NULL,

  INDEX idx_users_email (email),
  INDEX idx_users_deleted_at (deleted_at),
  CONSTRAINT fk_users_subscription_plan FOREIGN KEY (subscription_plan) REFERENCES subscription_plans(id) ON UPDATE CASCADE ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ─── Sensors table ───────────────────────────────────────────────────────────
-- NOTE: sensors MUST be created BEFORE farms (farms.active_sensor_id → sensors.id)

CREATE TABLE IF NOT EXISTS sensors (
  id          VARCHAR(36)   NOT NULL PRIMARY KEY,
  name        VARCHAR(255)  NOT NULL,
  latitude    DECIMAL(10,7) NOT NULL,
  longitude   DECIMAL(10,7) NOT NULL,
  status      ENUM('safe', 'warning', 'danger') NOT NULL DEFAULT 'safe',
  tds_value   DECIMAL(10,2) NOT NULL DEFAULT 0,
  temperature DECIMAL(5,2)  NULL,
  ph          DECIMAL(4,2)  NULL,
  created_at  DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at  DATETIME      NULL,
  created_by  VARCHAR(36)   NULL,
  updated_by  VARCHAR(36)   NULL,

  INDEX idx_sensors_deleted_at (deleted_at)
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
  deleted_at             DATETIME      NULL,
  created_by             VARCHAR(36)   NULL,
  updated_by             VARCHAR(36)   NULL,

  INDEX idx_farms_user_id (user_id),
  INDEX idx_farms_deleted_at (deleted_at),
  CONSTRAINT fk_farms_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  CONSTRAINT fk_farms_active_sensor FOREIGN KEY (active_sensor_id) REFERENCES sensors(id) ON DELETE SET NULL
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

-- ─── User nodes table (many-to-many: users ↔ sensors) ────────────────────────

CREATE TABLE IF NOT EXISTS user_nodes (
  id          VARCHAR(36)   NOT NULL PRIMARY KEY,                -- UUID v4
  user_id     VARCHAR(36)   NOT NULL,
  sensor_id   VARCHAR(36)   NOT NULL,
  is_active   TINYINT(1)    NOT NULL DEFAULT 0,
  created_at  DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at  DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at  DATETIME      NULL,
  created_by  VARCHAR(36)   NULL,
  updated_by  VARCHAR(36)   NULL,

  INDEX idx_user_nodes_user_id (user_id),
  INDEX idx_user_nodes_deleted_at (deleted_at),
  UNIQUE INDEX idx_user_nodes_user_sensor (user_id, sensor_id),
  CONSTRAINT fk_user_nodes_user   FOREIGN KEY (user_id)   REFERENCES users(id)   ON DELETE CASCADE,
  CONSTRAINT fk_user_nodes_sensor FOREIGN KEY (sensor_id) REFERENCES sensors(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ─── AI recommendations table ────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS ai_recommendations (
  id               VARCHAR(36)   NOT NULL PRIMARY KEY,
  title            VARCHAR(255)  NOT NULL,
  body             TEXT          NOT NULL,
  type             VARCHAR(50)   NOT NULL,               -- tds_alert | tds_danger | crop_suggestion | fertilizer
  reason_chips     JSON          NOT NULL DEFAULT (JSON_ARRAY()),
  confidence_score DECIMAL(3,2)  NOT NULL,
  created_at       DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at       DATETIME      NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  deleted_at       DATETIME      NULL,
  created_by       VARCHAR(36)   NULL,
  updated_by       VARCHAR(36)   NULL,

  INDEX idx_ai_recs_type (type),
  INDEX idx_ai_recs_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ─── Crop suggestions table ──────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS crop_suggestions (
  id                    VARCHAR(36)   NOT NULL PRIMARY KEY,
  name                  VARCHAR(100)  NOT NULL,
  name_th               VARCHAR(100)  NOT NULL,
  estimated_price_per_kg DECIMAL(10,2) NOT NULL,
  reason                TEXT          NOT NULL,
  icon                  VARCHAR(10)   NOT NULL DEFAULT '',
  min_tds               DECIMAL(10,2) NOT NULL DEFAULT 0,
  max_tds               DECIMAL(10,2) NOT NULL DEFAULT 9999,
  sort_order            INT           NOT NULL DEFAULT 0
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

-- Seed subscription plans (must be before users — FK dependency)
INSERT IGNORE INTO subscription_plans (id, name, price, period, features, recommended, sort_order) VALUES
  ('free',    'Free',    '฿0',   '',        '["ดูข้อมูลตัวอย่าง (Demo)","AI พื้นฐาน","ไม่มีการแจ้งเตือน"]', 0, 0),
  ('starter', 'Starter', '฿59',  '/ฤดูกาล', '["เชื่อมต่อ 1 Sensor","แจ้งเตือนผ่านแอป","บันทึกสถิติรายสัปดาห์","AI ระดับพื้นฐาน"]', 1, 1),
  ('pro',     'Pro',     '฿199', '/ปี',     '["เชื่อมต่อ 5 Sensors","AI Level 3 — วิเคราะห์เชิงลึก","พยากรณ์ผลผลิตรายเดือน","Export CSV/PDF","สรุปรายงานผ่าน LINE"]', 0, 2);

-- Seed demo users (matching MockAuthRepository)
-- password_hash is bcrypt of 'password123'
INSERT IGNORE INTO users (id, first_name, last_name, email, phone, birth_date, password_hash, subscription_plan, role)
VALUES
  ('u001', 'สมชาย', 'ใจดี',    'somchai@example.com', '0812345678', '1985-06-15',
   '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'starter', 'user'),
  ('u002', 'มาลี',   'เกษตรดี', 'malee@example.com',   '0898765432', '1990-03-22',
   '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'pro', 'user');

-- Seed farm for u001
INSERT IGNORE INTO farms (id, user_id, name, area_size_rai, crop_type, yield_ton_per_rai,
  avg_price_baht_per_kg, distribution_channels, soil_ph, soil_problems, water_source, latitude, longitude, active_sensor_id)
VALUES (
  'f001', 'u001', 'แปลงนาหัวทุ่ง', 12.5, 'rice', 0.8, 9.5,
  '["พ่อค้าคนกลาง","สหกรณ์"]', 6.2, '["ดินเปรี้ยว"]', 'น้ำชลประทาน',
  14.88, 100.99, 's001'
);

-- Seed AI recommendations
INSERT IGNORE INTO ai_recommendations (id, title, body, type, reason_chips, confidence_score, created_at) VALUES
  ('ai001', 'ค่า TDS มีแนวโน้มเพิ่มขึ้น',
   'พยากรณ์อากาศ 5 วันข้างหน้าไม่มีฝนในพื้นที่ของคุณ ค่า TDS อาจเพิ่มขึ้นถึง 450–500 ppm แนะนำให้เปิดประตูน้ำเพื่อเจือจางก่อนปลูก',
   'tds_alert', '[{"label":"TDS 420 ppm","category":"tds"},{"label":"เพิ่ม 3 วันติดกัน","category":"trend"},{"label":"ไม่มีฝน 5 วัน","category":"weather"}]',
   0.87, DATE_SUB(NOW(), INTERVAL 2 HOUR)),

  ('ai002', 'แนะนำพืชทางเลือก',
   'ราคาข้าวในตลาดมีแนวโน้มลดลง 8% ในฤดูกาลนี้ การปลูกถั่วเขียวหรือข้าวโพดหวานอาจให้ผลตอบแทนสูงกว่า',
   'crop_suggestion', '[{"label":"ราคาข้าว ↓ 8%","category":"market"},{"label":"TDS 350 ppm","category":"tds"}]',
   0.76, DATE_SUB(NOW(), INTERVAL 5 HOUR)),

  ('ai003', 'ระดับ TDS วิกฤต — Sensor 03',
   'Sensor 03 (คลองชลประทาน) วัดค่า TDS ที่ 720 ppm เกินระดับวิกฤต 500 ppm แนะนำให้ปิดการรับน้ำจากแหล่งนี้ทันทีและเปิดน้ำจาก Sensor 01 แทน',
   'tds_danger', '[{"label":"TDS 720 ppm","category":"tds"},{"label":"เกินเกณฑ์วิกฤต","category":"trend"},{"label":"ไม่มีฝน 5 วัน","category":"weather"}]',
   0.96, DATE_SUB(NOW(), INTERVAL 30 MINUTE)),

  ('ai004', 'ช่วงเวลาใส่ปุ๋ยที่เหมาะสม',
   'ค่า TDS อยู่ในระดับเหมาะสม (350 ppm) แนะนำให้ใส่ปุ๋ยสูตร 16-20-0 ในช่วงเช้า 06:00–08:00 น. ใน 3 วันข้างหน้า เนื่องจากมีฝนพยากรณ์ที่จะช่วยพาปุ๋ยลงดิน',
   'fertilizer', '[{"label":"TDS 350 ppm","category":"tds"},{"label":"มีฝน 3 วันข้างหน้า","category":"weather"}]',
   0.82, DATE_SUB(NOW(), INTERVAL 1 HOUR));

-- Seed crop suggestions
INSERT IGNORE INTO crop_suggestions (id, name, name_th, estimated_price_per_kg, reason, icon, min_tds, max_tds, sort_order) VALUES
  ('crop001', 'Mung Bean',  'ถั่วเขียว',    28.0, 'ทนน้ำเค็มปานกลาง เหมาะกับ TDS 300–500 ppm', '🫘', 0, 9999, 0),
  ('crop002', 'Sweet Corn', 'ข้าวโพดหวาน',  12.5, 'ราคาตลาดสูง ใช้น้ำน้อยกว่าข้าว 40%',        '🌽', 0,  600, 1),
  ('crop003', 'Rice',       'ข้าว',          9.5,  'พืชหลักปัจจุบัน',                            '🌾', 0,  400, 2);
