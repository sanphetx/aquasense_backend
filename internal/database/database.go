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
			&models.User{},
			&models.Farm{},
			&models.Sensor{},
			&models.WaterRecord{},
			&models.NotificationSettings{},
			&models.UserNode{},
		)
		if err != nil {
			logger.Get().Warn("AutoMigrate warning", zap.Error(err))
		}
		logger.Get().Info("AutoMigrate completed")
	}

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
