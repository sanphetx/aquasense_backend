package database

import (
	"fmt"
	"os"
	"time"

	"aquasense-backend/internal/logger"
	"aquasense-backend/internal/models"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Connect opens a MySQL connection pool using GORM and verifies connectivity.
// runMigrate: run AutoMigrate (true in development, false in production).
// adminPassword: seed the default admin user with this password.
func Connect(dsn string, runMigrate bool, adminPassword string) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("gorm.Open: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// Connection pool settings
	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	// Verify connection
	if err = sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("db.Ping: %w", err)
	}

	logger.Get().Info("Connected to MySQL successfully via GORM")

	// [Security #9] AutoMigrate only in development — use migration tool in production
	if runMigrate {
		err = db.AutoMigrate(
			&models.SubscriptionPlan{}, // must be before User (FK dependency)
			&models.Sensor{},           // must be before Farm, UserNode, WaterRecord
			&models.User{},
			&models.Farm{},
			&models.WaterRecord{},
			&models.NotificationSettings{},
			&models.UserNode{},
			&models.AiRecommendation{},
			&models.CropSuggestion{},
		)
		if err != nil {
			logger.Get().Warn("AutoMigrate warning", zap.Error(err))
		}
		logger.Get().Info("AutoMigrate completed")
	}

	seedPlans(db)   // must run before seedAdmin (FK: users.subscription_plan → subscription_plans.id)
	seedSensors(db) // must run before seedDemoUsers (FK: farms.active_sensor_id → sensors.id)
	seedAiData(db)
	seedDemoUsers(db) // must run after seedPlans + seedSensors
	seedAdmin(db, adminPassword)

	return db, nil
}

// seedAdmin creates the default admin user if it doesn't exist.
// [Security #8] Password comes from ADMIN_DEFAULT_PASSWORD env var.
func seedAdmin(db *gorm.DB, adminPassword string) {
	var count int64
	db.Model(&models.User{}).Where("email = ?", "admin@gmail.com").Count(&count)
	if count > 0 {
		return
	}

	// [Security #8] Use env var password; generate random if not set
	if adminPassword == "" {
		// Generate a simple random password from UUID
		adminPassword = uuid.NewString()[:16]
		logger.Get().Warn("ADMIN_DEFAULT_PASSWORD not set — generated random password",
			zap.String("email", "admin@gmail.com"),
			zap.String("password", adminPassword),
			zap.String("action", "SAVE THIS PASSWORD — it will not be shown again"),
		)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(adminPassword), bcrypt.DefaultCost)
	if err != nil {
		logger.Get().Error("Failed to hash admin password", zap.Error(err))
		return
	}

	// Check if running in production — warn about admin credentials
	if os.Getenv("GIN_MODE") == "release" {
		logger.Get().Warn("Seeding admin user in production",
			zap.String("email", "admin@gmail.com"),
			zap.String("action", "Change ADMIN_DEFAULT_PASSWORD after first login"),
		)
	}

	admin := models.User{
		ID:               uuid.NewString(),
		FirstName:        "System",
		LastName:         "Admin",
		Email:            "admin@gmail.com",
		Phone:            "0000000000",
		BirthDate:        time.Now(),
		PasswordHash:     string(hash),
		SubscriptionPlan: "pro",
		Role:             "admin",
	}
	if err := db.Create(&admin).Error; err != nil {
		logger.Get().Error("Failed to seed admin user", zap.Error(err))
	} else {
		logger.Get().Info("Default admin user seeded", zap.String("email", "admin@gmail.com"))
	}
}

// seedPlans creates default subscription plans if they don't exist.
func seedPlans(db *gorm.DB) {
	var count int64
	db.Model(&models.SubscriptionPlan{}).Count(&count)
	if count > 0 {
		return
	}

	plans := []models.SubscriptionPlan{
		{
			ID: "free", Name: "Free", Price: "฿0", Period: "",
			Features:    `["ดูข้อมูลตัวอย่าง (Demo)","AI พื้นฐาน","ไม่มีการแจ้งเตือน"]`,
			Recommended: false, SortOrder: 0,
		},
		{
			ID: "starter", Name: "Starter", Price: "฿59", Period: "/ฤดูกาล",
			Features:    `["เชื่อมต่อ 1 Sensor","แจ้งเตือนผ่านแอป","บันทึกสถิติรายสัปดาห์","AI ระดับพื้นฐาน"]`,
			Recommended: true, SortOrder: 1,
		},
		{
			ID: "pro", Name: "Pro", Price: "฿199", Period: "/ปี",
			Features:    `["เชื่อมต่อ 5 Sensors","AI Level 3 — วิเคราะห์เชิงลึก","พยากรณ์ผลผลิตรายเดือน","Export CSV/PDF","สรุปรายงานผ่าน LINE"]`,
			Recommended: false, SortOrder: 2,
		},
	}

	for _, p := range plans {
		db.Create(&p)
	}
	logger.Get().Info("Subscription plans seeded", zap.Int("count", len(plans)))
}

// seedAiData creates default AI recommendations and crop suggestions if they don't exist.
func seedAiData(db *gorm.DB) {
	// Seed AI recommendations
	var recCount int64
	db.Model(&models.AiRecommendation{}).Count(&recCount)
	if recCount == 0 {
		now := time.Now()
		recs := []models.AiRecommendation{
			{
				ID: "ai001", Title: "ค่า TDS มีแนวโน้มเพิ่มขึ้น",
				Body: "พยากรณ์อากาศ 5 วันข้างหน้าไม่มีฝนในพื้นที่ของคุณ ค่า TDS อาจเพิ่มขึ้นถึง 450–500 ppm แนะนำให้เปิดประตูน้ำเพื่อเจือจางก่อนปลูก",
				Type: "tds_alert", ConfidenceScore: 0.87,
				ReasonChips: `[{"label":"TDS 420 ppm","category":"tds"},{"label":"เพิ่ม 3 วันติดกัน","category":"trend"},{"label":"ไม่มีฝน 5 วัน","category":"weather"}]`,
				BaseModel:   models.BaseModel{CreatedAt: now.Add(-2 * time.Hour)},
			},
			{
				ID: "ai002", Title: "แนะนำพืชทางเลือก",
				Body: "ราคาข้าวในตลาดมีแนวโน้มลดลง 8% ในฤดูกาลนี้ การปลูกถั่วเขียวหรือข้าวโพดหวานอาจให้ผลตอบแทนสูงกว่า",
				Type: "crop_suggestion", ConfidenceScore: 0.76,
				ReasonChips: `[{"label":"ราคาข้าว ↓ 8%","category":"market"},{"label":"TDS 350 ppm","category":"tds"}]`,
				BaseModel:   models.BaseModel{CreatedAt: now.Add(-5 * time.Hour)},
			},
			{
				ID: "ai003", Title: "ระดับ TDS วิกฤต — Sensor 03",
				Body: "Sensor 03 (คลองชลประทาน) วัดค่า TDS ที่ 720 ppm เกินระดับวิกฤต 500 ppm แนะนำให้ปิดการรับน้ำจากแหล่งนี้ทันทีและเปิดน้ำจาก Sensor 01 แทน",
				Type: "tds_danger", ConfidenceScore: 0.96,
				ReasonChips: `[{"label":"TDS 720 ppm","category":"tds"},{"label":"เกินเกณฑ์วิกฤต","category":"trend"},{"label":"ไม่มีฝน 5 วัน","category":"weather"}]`,
				BaseModel:   models.BaseModel{CreatedAt: now.Add(-30 * time.Minute)},
			},
			{
				ID: "ai004", Title: "ช่วงเวลาใส่ปุ๋ยที่เหมาะสม",
				Body: "ค่า TDS อยู่ในระดับเหมาะสม (350 ppm) แนะนำให้ใส่ปุ๋ยสูตร 16-20-0 ในช่วงเช้า 06:00–08:00 น. ใน 3 วันข้างหน้า เนื่องจากมีฝนพยากรณ์ที่จะช่วยพาปุ๋ยลงดิน",
				Type: "fertilizer", ConfidenceScore: 0.82,
				ReasonChips: `[{"label":"TDS 350 ppm","category":"tds"},{"label":"มีฝน 3 วันข้างหน้า","category":"weather"}]`,
				BaseModel:   models.BaseModel{CreatedAt: now.Add(-1 * time.Hour)},
			},
		}
		for _, r := range recs {
			db.Create(&r)
		}
		logger.Get().Info("AI recommendations seeded", zap.Int("count", len(recs)))
	}

	// Seed crop suggestions
	var cropCount int64
	db.Model(&models.CropSuggestion{}).Count(&cropCount)
	if cropCount == 0 {
		crops := []models.CropSuggestion{
			{ID: "crop001", Name: "Mung Bean", NameTH: "ถั่วเขียว", EstimatedPricePerKg: 28.0,
				Reason: "ทนน้ำเค็มปานกลาง เหมาะกับ TDS 300–500 ppm", Icon: "🫘",
				MinTDS: 0, MaxTDS: 9999, SortOrder: 0},
			{ID: "crop002", Name: "Sweet Corn", NameTH: "ข้าวโพดหวาน", EstimatedPricePerKg: 12.5,
				Reason: "ราคาตลาดสูง ใช้น้ำน้อยกว่าข้าว 40%", Icon: "🌽",
				MinTDS: 0, MaxTDS: 600, SortOrder: 1},
			{ID: "crop003", Name: "Rice", NameTH: "ข้าว", EstimatedPricePerKg: 9.5,
				Reason: "พืชหลักปัจจุบัน", Icon: "🌾",
				MinTDS: 0, MaxTDS: 400, SortOrder: 2},
		}
		for _, c := range crops {
			db.Create(&c)
		}
		logger.Get().Info("Crop suggestions seeded", zap.Int("count", len(crops)))
	}
}

// seedSensors creates demo sensors if they don't exist.
func seedSensors(db *gorm.DB) {
	var count int64
	db.Model(&models.Sensor{}).Count(&count)
	if count > 0 {
		return
	}

	temp1, temp2, temp3, temp4, temp5 := 28.5, 30.1, 31.4, 27.8, 26.9
	ph1, ph2, ph3, ph4, ph5 := 6.8, 6.2, 5.8, 7.0, 7.2

	sensors := []models.Sensor{
		{ID: "s001", Name: "Sensor 01 — ลำคลองใหญ่", Latitude: 14.8820000, Longitude: 100.9940000, Status: "safe", TDSValue: 350, Temperature: &temp1, PH: &ph1},
		{ID: "s002", Name: "Sensor 02 — ท่อระบายน้ำหลัก", Latitude: 14.8760000, Longitude: 100.9910000, Status: "warning", TDSValue: 520, Temperature: &temp2, PH: &ph2},
		{ID: "s003", Name: "Sensor 03 — คลองชลประทาน", Latitude: 14.8840000, Longitude: 100.9960000, Status: "danger", TDSValue: 720, Temperature: &temp3, PH: &ph3},
		{ID: "s004", Name: "Sensor 04 — สระเก็บน้ำฝั่งเหนือ", Latitude: 14.8856000, Longitude: 100.9895000, Status: "safe", TDSValue: 280, Temperature: &temp4, PH: &ph4},
		{ID: "s005", Name: "Sensor 05 — แหล่งน้ำบาดาล", Latitude: 14.8808000, Longitude: 100.9975000, Status: "safe", TDSValue: 310, Temperature: &temp5, PH: &ph5},
	}

	for _, s := range sensors {
		db.Create(&s)
	}
	logger.Get().Info("Demo sensors seeded", zap.Int("count", len(sensors)))

	// Seed 7 days of water history
	seedWaterRecords(db)
}

// seedWaterRecords creates 7-day water history for demo sensors.
func seedWaterRecords(db *gorm.DB) {
	var count int64
	db.Model(&models.WaterRecord{}).Count(&count)
	if count > 0 {
		return
	}

	now := time.Now()
	var records []models.WaterRecord

	for _, base := range []struct {
		sensorID string
		baseTDS  float64
	}{
		{"s001", 340},
		{"s002", 510},
	} {
		for n := 0; n < 7; n++ {
			tds := base.baseTDS + float64((n%5)*25) + float64((n*7)%60)
			ph := 6.5 + float64(n%3)*0.2
			temp := 27.0 + float64(n%4)
			moisture := 55.0 + float64(n%6)*3

			status := "safe"
			if tds > 600 {
				status = "danger"
			} else if tds > 450 {
				status = "warning"
			}

			records = append(records, models.WaterRecord{
				SensorID:     base.sensorID,
				Date:         now.AddDate(0, 0, -n),
				TDS:          tds,
				PH:           &ph,
				Temperature:  &temp,
				SoilMoisture: &moisture,
				Status:       status,
			})
		}
	}

	db.Create(&records)
	logger.Get().Info("Water records seeded", zap.Int("count", len(records)))
}

// seedDemoUsers creates demo users + farm if they don't exist.
func seedDemoUsers(db *gorm.DB) {
	var count int64
	db.Model(&models.User{}).Where("id IN ?", []string{"u001", "u002"}).Count(&count)
	if count > 0 {
		return
	}

	// bcrypt hash of "password123"
	passwordHash := "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"

	users := []models.User{
		{
			ID: "u001", FirstName: "สมชาย", LastName: "ใจดี",
			Email: "somchai@example.com", Phone: "0812345678",
			BirthDate:    time.Date(1985, 6, 15, 0, 0, 0, 0, time.Local),
			PasswordHash: passwordHash, SubscriptionPlan: "starter", Role: "user",
		},
		{
			ID: "u002", FirstName: "มาลี", LastName: "เกษตรดี",
			Email: "malee@example.com", Phone: "0898765432",
			BirthDate:    time.Date(1990, 3, 22, 0, 0, 0, 0, time.Local),
			PasswordHash: passwordHash, SubscriptionPlan: "pro", Role: "user",
		},
	}

	for _, u := range users {
		db.Create(&u)
	}
	logger.Get().Info("Demo users seeded", zap.Int("count", len(users)))

	// Seed farm for u001
	var farmCount int64
	db.Model(&models.Farm{}).Where("id = ?", "f001").Count(&farmCount)
	if farmCount == 0 {
		lat, lng := 14.88, 100.99
		yieldPerRai := 0.8
		avgPrice := 9.5
		soilPh := 6.2
		activeSensor := "s001"

		farm := models.Farm{
			ID: "f001", UserID: "u001", Name: "แปลงนาหัวทุ่ง",
			AreaSizeRai: 12.5, CropType: "rice",
			YieldTonPerRai:       &yieldPerRai,
			AvgPriceBahtPerKg:    &avgPrice,
			DistributionChannels: `["พ่อค้าคนกลาง","สหกรณ์"]`,
			SoilPh:               &soilPh,
			SoilProblems:         `["ดินเปรี้ยว"]`,
			WaterSource:          "น้ำชลประทาน",
			Latitude:             &lat,
			Longitude:            &lng,
			ActiveSensorID:       &activeSensor,
		}
		db.Create(&farm)
		logger.Get().Info("Demo farm seeded", zap.String("id", "f001"))
	}
}
