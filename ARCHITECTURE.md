# 🏗️ สถาปัตยกรรมระบบ (Architecture Overview) - AquaSense Backend

โปรเจกต์นี้เขียนด้วยภาษา **Go (Golang)** โดยใช้เฟรมเวิร์ก **Gin** สำหรับทำ REST API และใช้ **MySQL** เป็นฐานข้อมูลหลักร่วมกับ **GORM** (Object-Relational Mapping) 

โครงสร้างของโปรเจกต์ออกแบบตามหลักการ **Layered Architecture (N-Tier)** หรือโครงสร้างแบบแบ่งชั้น เพื่อให้โค้ดเป็นระเบียบ แก้ไขง่าย และรองรับการขยายตัว (Scale) ในอนาคต โดยแต่ละส่วนจะถูกแยกหน้าที่กันอย่างชัดเจน (Single Responsibility)

---

## 📂 โครงสร้างโฟลเดอร์หลัก (Directory Structure)

```text
aquasense-go-backend/
├── cmd/
│   └── server/
│       └── main.go         <-- จุดเริ่มต้นของโปรแกรม (Entry Point)
├── internal/               <-- โค้ดหลักของระบบทั้งหมดถูกเก็บไว้ที่นี่
│   ├── config/
│   ├── database/
│   ├── router/
│   ├── handlers/
│   ├── service/
│   ├── repository/
│   ├── models/
│   ├── middleware/
│   └── logger/
├── .env                    <-- ไฟล์ตั้งค่า Environment
└── Makefile                <-- คำสั่งลัดสำหรับรันโปรเจกต์ (เช่น make run)
```

---

## ⚙️ อธิบายหน้าที่ของแต่ละ Layer (จากนอกสุด เข้าไปลึกสุด)

### 1. `cmd/server/main.go` (The Entry Point)
เป็นจุดสตาร์ทของระบบ หน้าที่หลักคือการ **"ประกอบร่าง (Dependency Injection)"** 
* โหลด Config จากไฟล์ `.env`
* สั่งเชื่อมต่อ Database
* นำ Repository, Service และ Handler มาเชื่อมต่อเข้าด้วยกัน
* เรียกใช้ Router เพื่อเปิดพอร์ต (เช่น `:8080`) และรอรับ Request

### 2. `internal/router/router.go` (The Traffic Controller)
ทำหน้าที่เป็น **"ตำรวจจราจร"**
* กำหนดเส้นทาง URL ทั้งหมด (เช่น `GET /api/v1/farms`)
* ผูกเส้นทางเข้ากับ Handler ที่ถูกต้อง
* ติดตั้ง Middleware (เช่น การตรวจเช็ค CORS, เช็ค JWT Token)

### 3. `internal/handlers/` (The Delivery Layer)
ทำหน้าที่เป็น **"พนักงานต้อนรับ"**
* รับ Request จากผู้ใช้ (เช่น รับ JSON, รับพารามิเตอร์จาก URL)
* ตรวจสอบความถูกต้องของข้อมูลเบื้องต้น (Validation)
* ส่งข้อมูลต่อไปให้ Service คิดคำนวณ
* นำผลลัพธ์จาก Service มาจัดรูปแบบเป็น JSON แล้วตอบกลับ (Response) กลับไปยังแอป

### 4. `internal/service/` (The Business Logic Layer) 🧠
ทำหน้าที่เป็น **"สมองของระบบ"**
* เก็บ Logic สำคัญทั้งหมดของแอปพลิเคชัน (Business Rules)
* เช่น การคำนวณสูตรต่างๆ, การสร้าง/แกะ JWT Token, การคุยกับ AI ภายนอก, การตรวจสอบสิทธิ์ที่ซับซ้อน
* จะไม่ยุ่งเกี่ยวกับการต่อ Database ตรงๆ แต่จะสั่งงานผ่าน Repository อีกที

### 5. `internal/repository/` (The Data Access Layer) 🗄️
ทำหน้าที่เป็น **"คนดูแลโกดังข้อมูล"**
* รับคำสั่งจาก Service เพื่อไปคุยกับ MySQL Database
* มีหน้าที่เขียนคำสั่ง GORM (หรือ SQL) เพื่อ `SELECT`, `INSERT`, `UPDATE`, `DELETE` เท่านั้น
* ไม่มีการใส่ Logic การคำนวณไว้ในนี้ มีหน้าที่แค่ดึงข้อมูลส่งกลับไปให้ Service

### 6. `internal/models/` (The Domain Models)
ทำหน้าที่เป็น **"พิมพ์เขียวของข้อมูล"**
* เก็บโครงสร้างของตารางใน Database (`struct` ที่ใช้กับ GORM)
* เก็บโครงสร้างของ JSON ที่จะรับเข้าหรือส่งออก (Request/Response JSON structs)
* โฟลเดอร์อื่นทุกโฟลเดอร์จะต้องเรียกใช้ Struct จากที่นี่

### 7. โฟลเดอร์เสริมอื่นๆ (Utilities)
* **`internal/config/`**: โหลดค่าตัวแปรระบบจาก `.env` มาเก็บเป็นตัวแปรให้ระบบเรียกใช้ง่ายๆ
* **`internal/database/`**: จัดการเรื่อง Connection Pool, ปิงเช็ค Database, และทำการสร้างตารางอัตโนมัติ (AutoMigrate)
* **`internal/middleware/`**: ด่านตรวจคัดกรอง เช่น `AuthMiddleware` (ตรวจว่าล็อกอินมาไหม) และ `AdminMiddleware` (ตรวจว่าเป็นแอดมินไหม)
* **`internal/logger/`**: ระบบจดบันทึก Log ของเซิร์ฟเวอร์ โดยใช้แพ็กเกจ `zap` ที่มีความเร็วสูง

---

## 🔄 สรุป Flow การทำงาน (เมื่อแอปยิง API เข้ามา)

1. **User (Flutter App)** ยิง Request มาที่ `GET /api/v1/dashboard/summary`
2. **Router** รับสาย แล้วส่งต่อให้ `AuthMiddleware` ตรวจดูว่ามี Token ที่ถูกต้องไหม
3. ถ้าผ่าน จะส่งไปให้ **Handler** (`SensorHandler.GetDashboardSummary`)
4. **Handler** จะแกะดูว่า User คนนี้ ID อะไร แล้วโทรสั่ง **Service**
5. **Service** จะโทรสั่ง **Repository** ให้ออกไปดึงค่าเซนเซอร์ล่าสุด และดึงกราฟ 7 วัน (ทำพร้อมกันด้วย Goroutine)
6. **Repository** วิ่งไปหยิบของจาก **Database (MySQL)** โดยใช้ **Models** เป็นกล่องใส่ของ แล้วส่งคืนกลับมาตามลำดับ
7. **Handler** นำกล่องข้อมูลนั้นมาห่อเป็น JSON และส่ง `200 OK` กลับไปยังหน้าแอปพลิเคชัน
