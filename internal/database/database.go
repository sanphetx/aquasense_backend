package database

import (
	"fmt"
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
func Connect(dsn string) (*gorm.DB, error) {
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

	// Auto Migrate the schemas
	err = db.AutoMigrate(
		&models.User{},
		&models.Farm{},
		&models.Sensor{},
		&models.WaterRecord{},
		&models.NotificationSettings{},
	)
	if err != nil {
		logger.Get().Warn("AutoMigrate warning", zap.Error(err))
	}

	seedAdmin(db)

	return db, nil
}

func seedAdmin(db *gorm.DB) {
	var count int64
	db.Model(&models.User{}).Where("email = ?", "admin@gmail.com").Count(&count)
	if count == 0 {
		hash, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
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
			logger.Get().Info("Default admin user seeded: admin@gmail.com / 123456")
		}
	}
}
