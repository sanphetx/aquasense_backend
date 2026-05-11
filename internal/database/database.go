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
			&models.SubscriptionPlan{},   // must be before User (FK dependency)
			&models.Sensor{},             // must be before Farm, UserNode, WaterRecord
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

	seedPlans(db)     // must run before seedAdmin (FK: users.subscription_plan → subscription_plans.id)
	seedAiData(db)
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
