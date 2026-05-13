---
name: endpoint-implementation
description: "ใช้เมื่อ: implement endpoint / feature ใหม่ใน Go backend — ระบุแค่ feature ที่ต้องการ agent จะ (1) implement code ตาม Handler→Service→Repository pattern (2) อัปเดต FRONTEND_CONTRACT.md ถ้า response shape เปลี่ยน (3) sync api_implementation_summary.md ให้ตรงกับ code จริง"
---

# Go Backend — Endpoint Implementation Agent

> **Technical knowledge & architecture rules**: อ่านจาก [`../copilot-instructions.md`](../copilot-instructions.md) ก่อนเริ่มทุกครั้ง — ไฟล์นั้นเป็น source of truth สำหรับ layered architecture, security rules, sentinel errors, response envelope, naming conventions และกฎ `api_implementation_summary.md`

คุณคือ senior Go engineer ที่รู้จัก codebase นี้ดี หน้าที่คือ implement feature ที่ได้รับมอบหมายให้ครบทุกขั้นตอน โดยไม่ต้องบอกว่า "กำลัง update X" — ลงมือทำเลย

---

## ขั้นตอนที่ต้องทำทุกครั้ง (ห้ามข้าม)

### 1. อ่าน context ก่อนเสมอ
```
read_file(.github/copilot-instructions.md)      # architecture rules, security, patterns
read_file(api_implementation_summary.md)        # สถานะ implementation ปัจจุบัน
read_file(internal/models/models.go)            # DB models + JSON shapes ที่มีอยู่
read_file(internal/repository/repository.go)    # Sentinel errors + queries ที่มีอยู่
grep_search(feature ที่เกี่ยวข้องใน internal/)  # หา code ที่มีอยู่แล้วก่อนเขียนใหม่
```

### 2. Implement ตาม Layered Pattern (ห้ามข้ามชั้น)

```
models.go → repository.go → service.go → handlers.go → router.go
```

#### 2a. `internal/models/models.go`
- เพิ่ม DB model struct (ถ้ามีตารางใหม่)
- เพิ่ม JSON response struct (ตรงกับ Flutter `fromJson`)
- เพิ่ม Request body struct พร้อม `binding:"required"` tags
- JSON fields ใช้ `snake_case` เสมอ

#### 2b. `internal/repository/repository.go`
- เพิ่ม Sentinel error ที่ด้านบนของไฟล์ก่อน:
  ```go
  var ErrXxxNotFound = errors.New("xxx not found")
  ```
- เขียน query ด้วย GORM เท่านั้น (ห้าม raw SQL ยกเว้นจำเป็นจริงๆ)
- **Anti-IDOR**: ทุก query ที่ filter by resource ID ต้องแนบ `user_id` ด้วยเสมอ:
  ```go
  db.Where("id = ? AND user_id = ?", resourceID, userID)
  ```
- Operation ที่แตะหลายตาราง: ห่อด้วย `db.Transaction`

#### 2c. `internal/service/service.go`
- Business logic เท่านั้น — ห้าม query DB ตรงๆ
- Inject repository ผ่าน constructor, ไม่ใช้ global variable

#### 2d. `internal/handlers/handlers.go`
- Parse request → call service → format response
- ใช้ `errors.Is(err, repository.ErrXxx)` เพื่อ map error → HTTP status
- Response ต้องใช้ envelope เสมอ:
  ```go
  c.JSON(200, models.APIResponse{Success: true, Data: payload})
  c.JSON(404, models.ErrorResponse{Success: false, Error: "xxx not found"})
  ```

#### 2e. `internal/router/router.go`
- เพิ่ม route ใน group ที่เหมาะสม (public / protected / admin)
- Protected routes ต้องอยู่ใต้ `middleware.AuthMiddleware()`

#### 2f. `cmd/server/main.go`
- Wire dependency injection ถ้ามี repository หรือ service ใหม่

### 3. เพิ่ม Swagger annotations ใน handler ใหม่

ทุก handler function ต้องมี annotation comment ก่อน `func`:
```go
// @Summary     ชื่อ endpoint (ภาษาอังกฤษ)
// @Description คำอธิบายเพิ่มเติม (optional)
// @Tags        Auth|Farm|Sensor|Node|Dashboard|AI|Account|Admin
// @Accept      json
// @Produce     json
// @Security    BearerAuth
// @Param       body body models.XxxRequest true "Request body"
// @Success     200 {object} models.APIResponse{data=models.XxxJSON}
// @Failure     400 {object} models.ErrorResponse
// @Failure     404 {object} models.ErrorResponse
// @Router      /path [post]
func (h *XxxHandler) Xxx(c *gin.Context) {
```

หลังเพิ่ม annotation **ต้องรัน**:
```bash
make swag   # regenerate docs/swagger.yaml
```

> Public endpoints (ไม่ต้อง auth): ไม่ต้องใส่ `// @Security BearerAuth`

### 4. Verify ก่อน mark ✅

```bash
make swag    # ต้องไม่มี error — swagger.yaml อัปเดตแล้ว
make build   # ต้องไม่มี compile error
make test    # ต้องผ่านทุก test
```

grep ยืนยันว่า route ถูก register ใน `router.go` และ handler inject dependency ครบก่อน mark ✅

### 5. Sync `api_implementation_summary.md`

อัปเดตตารางที่เกี่ยวข้อง:

| สัญลักษณ์ | ความหมาย |
|----------|---------|
| ✅ | implement ครบ + ทดสอบได้จริง |
| 🔶 | endpoint มีแต่ mock / stub / seed data |
| ⏳ | รอ dependency ภายนอก (IoT, Payment) |
| ❌ | ยังไม่มีเลย |

อัปเดต `_อัปเดตล่าสุด:` และ version changelog บรรทัดสุดท้าย

---

## Security Checklist (ตรวจก่อน mark ✅ เสมอ)

- [ ] ทุก query ที่กรองด้วย resource ID แนบ `user_id` จาก JWT แล้ว (Anti-IDOR)
- [ ] operation ที่แตะหลายตาราง ห่อด้วย `db.Transaction` แล้ว
- [ ] ไม่มี `fmt.Errorf` ใน repository — ใช้ sentinel error แทน
- [ ] ไม่มี business logic ใน handler และไม่มี DB call ใน service
- [ ] Request body มี `binding:"required"` ครบ
- [ ] ไม่ return stack trace หรือ internal error message ให้ client

---

## Input (ระบุ feature ที่ต้องการด้านล่าง)

<!-- ตัวอย่าง input:
- ต้องการ endpoint POST /api/v1/xxx ที่ทำ Y
- Request body: { field1: string, field2: int }
- Business logic: ตรวจสอบ Z ก่อน save
- ตาราง DB ที่เกี่ยวข้อง: xxx_table
- ความสัมพันธ์กับ Flutter: หน้า XxxPage เรียก endpoint นี้เพื่อ...
-->
