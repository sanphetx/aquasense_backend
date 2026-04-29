package repository

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"aquasense-backend/internal/models"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ─── AuthRepository ──────────────────────────────────────────────────────────

type AuthRepository struct {
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) *AuthRepository {
	return &AuthRepository{db: db}
}

func (r *AuthRepository) FindByEmail(email string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *AuthRepository) FindByID(id string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("id = ?", id).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

func (r *AuthRepository) CreateUser(req models.RegisterRequest) (*models.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("bcrypt: %w", err)
	}

	user := &models.User{
		ID:               uuid.NewString(),
		FirstName:        req.FirstName,
		LastName:         req.LastName,
		Email:            strings.ToLower(req.Email),
		Phone:            req.Phone,
		BirthDate:        req.BirthDate,
		PasswordHash:     string(hash),
		SubscriptionPlan: "free",
		Role:             "user",
	}

	if err := r.db.Create(user).Error; err != nil {
		return nil, err
	}
	return user, nil
}

func (r *AuthRepository) CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

func (r *AuthRepository) UpdateProfile(userID, firstName, lastName, phone string) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"first_name": firstName,
		"last_name":  lastName,
		"phone":      phone,
	}).Error
}

func (r *AuthRepository) UpdateSubscription(userID, planID string) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).Update("subscription_plan", planID).Error
}

// ─── FarmRepository ──────────────────────────────────────────────────────────

type FarmRepository struct {
	db *gorm.DB
}

func NewFarmRepository(db *gorm.DB) *FarmRepository {
	return &FarmRepository{db: db}
}

func (r *FarmRepository) CreateFarm(userID string, req models.CreateFarmRequest) (*models.FarmJSON, error) {
	distJSON, _ := json.Marshal(req.DistributionChannels)
	probJSON, _ := json.Marshal(req.SoilProblems)

	farm := &models.Farm{
		ID:                   uuid.NewString(),
		UserID:               userID,
		Name:                 req.Name,
		AreaSizeRai:          req.AreaSizeRai,
		CropType:             req.CropType,
		YieldTonPerRai:       req.YieldTonPerRai,
		AvgPriceBahtPerKg:    req.AvgPriceBahtPerKg,
		DistributionChannels: string(distJSON),
		SoilPh:               req.SoilPh,
		SoilProblems:         string(probJSON),
		WaterSource:          req.WaterSource,
	}

	if err := r.db.Create(farm).Error; err != nil {
		return nil, err
	}

	return &models.FarmJSON{
		ID:                   farm.ID,
		Name:                 farm.Name,
		AreaSizeRai:          farm.AreaSizeRai,
		CropType:             farm.CropType,
		YieldTonPerRai:       farm.YieldTonPerRai,
		AvgPriceBahtPerKg:    farm.AvgPriceBahtPerKg,
		DistributionChannels: req.DistributionChannels,
		SoilPh:               farm.SoilPh,
		SoilProblems:         req.SoilProblems,
		WaterSource:          farm.WaterSource,
	}, nil
}

func (r *FarmRepository) GetFarmByUserID(userID string) (*models.FarmJSON, error) {
	var farm models.Farm
	if err := r.db.Where("user_id = ?", userID).Order("created_at asc").First(&farm).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	var dist []string
	json.Unmarshal([]byte(farm.DistributionChannels), &dist)
	var prob []string
	json.Unmarshal([]byte(farm.SoilProblems), &prob)

	return &models.FarmJSON{
		ID:                   farm.ID,
		Name:                 farm.Name,
		AreaSizeRai:          farm.AreaSizeRai,
		CropType:             farm.CropType,
		YieldTonPerRai:       farm.YieldTonPerRai,
		AvgPriceBahtPerKg:    farm.AvgPriceBahtPerKg,
		DistributionChannels: dist,
		SoilPh:               farm.SoilPh,
		SoilProblems:         prob,
		WaterSource:          farm.WaterSource,
		Latitude:             farm.Latitude,
		Longitude:            farm.Longitude,
		ActiveSensorID:       farm.ActiveSensorID,
	}, nil
}

func (r *FarmRepository) UpdateLocation(farmID string, lat, lng float64) error {
	lat = float64(int(lat*100)) / 100
	lng = float64(int(lng*100)) / 100
	return r.db.Model(&models.Farm{}).Where("id = ?", farmID).Updates(map[string]interface{}{
		"latitude":  lat,
		"longitude": lng,
	}).Error
}

func (r *FarmRepository) LinkSensor(farmID, sensorID string) error {
	return r.db.Model(&models.Farm{}).Where("id = ?", farmID).Update("active_sensor_id", sensorID).Error
}

func (r *FarmRepository) GetFarmByID(farmID string) (*models.FarmJSON, error) {
	var farm models.Farm
	if err := r.db.Where("id = ?", farmID).First(&farm).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	var dist []string
	json.Unmarshal([]byte(farm.DistributionChannels), &dist)
	var prob []string
	json.Unmarshal([]byte(farm.SoilProblems), &prob)

	return &models.FarmJSON{
		ID:                   farm.ID,
		Name:                 farm.Name,
		AreaSizeRai:          farm.AreaSizeRai,
		CropType:             farm.CropType,
		YieldTonPerRai:       farm.YieldTonPerRai,
		AvgPriceBahtPerKg:    farm.AvgPriceBahtPerKg,
		DistributionChannels: dist,
		SoilPh:               farm.SoilPh,
		SoilProblems:         prob,
		WaterSource:          farm.WaterSource,
		Latitude:             farm.Latitude,
		Longitude:            farm.Longitude,
		ActiveSensorID:       farm.ActiveSensorID,
	}, nil
}

// ─── SensorRepository ────────────────────────────────────────────────────────

type SensorRepository struct {
	db *gorm.DB
}

func NewSensorRepository(db *gorm.DB) *SensorRepository {
	return &SensorRepository{db: db}
}

func (r *SensorRepository) GetNearbySensors(lat, lng float64) ([]models.SensorJSON, error) {
	var sensors []models.SensorJSON
	err := r.db.Raw(`
		SELECT id, name, latitude, longitude, status, tds_value, temperature, ph, updated_at as last_updated,
		(6371 * ACOS(COS(RADIANS(?)) * COS(RADIANS(latitude)) * COS(RADIANS(longitude) - RADIANS(?)) + SIN(RADIANS(?)) * SIN(RADIANS(latitude)))) AS distance_km
		FROM sensors ORDER BY distance_km ASC
	`, lat, lng, lat).Scan(&sensors).Error
	return sensors, err
}

func (r *SensorRepository) GetSensorLatest(sensorID string) (*models.SensorJSON, error) {
	var s models.Sensor
	if err := r.db.Where("id = ?", sensorID).First(&s).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &models.SensorJSON{
		ID:          s.ID,
		Name:        s.Name,
		Latitude:    s.Latitude,
		Longitude:   s.Longitude,
		Status:      s.Status,
		TDSValue:    s.TDSValue,
		Temperature: s.Temperature,
		PH:          s.PH,
		LastUpdated: s.UpdatedAt.Format(time.RFC3339),
	}, nil
}

func (r *SensorRepository) GetSensorHistory(sensorID, period string) ([]models.WaterRecordJSON, error) {
	days := 7
	if period == "30d" {
		days = 30
	}

	var records []models.WaterRecord
	err := r.db.Where("sensor_id = ? AND date >= DATE_SUB(NOW(), INTERVAL ? DAY)", sensorID, days).
		Order("date ASC").Find(&records).Error
	if err != nil {
		return nil, err
	}

	var result []models.WaterRecordJSON
	for _, rec := range records {
		result = append(result, models.WaterRecordJSON{
			Date:         rec.Date.Format(time.RFC3339),
			TDS:          rec.TDS,
			PH:           rec.PH,
			Temperature:  rec.Temperature,
			SoilMoisture: rec.SoilMoisture,
			Status:       rec.Status,
		})
	}
	return result, nil
}

// ─── NotificationRepository ──────────────────────────────────────────────────

type NotificationRepository struct {
	db *gorm.DB
}

func NewNotificationRepository(db *gorm.DB) *NotificationRepository {
	return &NotificationRepository{db: db}
}

func (r *NotificationRepository) GetSettings(userID string) (*models.NotificationSettingsJSON, error) {
	var s models.NotificationSettings
	err := r.db.Where("user_id = ?", userID).First(&s).Error
	if err == gorm.ErrRecordNotFound {
		return &models.NotificationSettingsJSON{
			PushEnabled:      true,
			TDSThreshold:     400,
			LineEnabled:      false,
			DailySummaryTime: "none",
		}, nil
	}
	if err != nil {
		return nil, err
	}

	return &models.NotificationSettingsJSON{
		PushEnabled:      s.PushEnabled,
		TDSThreshold:     s.TDSThreshold,
		LineEnabled:      s.LineEnabled,
		DailySummaryTime: s.DailySummaryTime,
	}, nil
}

func (r *NotificationRepository) SaveSettings(userID string, s models.NotificationSettingsJSON) error {
	var existing models.NotificationSettings
	if err := r.db.Where("user_id = ?", userID).First(&existing).Error; err == gorm.ErrRecordNotFound {
		return r.db.Create(&models.NotificationSettings{
			UserID:           userID,
			PushEnabled:      s.PushEnabled,
			TDSThreshold:     s.TDSThreshold,
			LineEnabled:      s.LineEnabled,
			DailySummaryTime: s.DailySummaryTime,
		}).Error
	} else if err != nil {
		return err
	}

	return r.db.Model(&existing).Updates(models.NotificationSettings{
		PushEnabled:      s.PushEnabled,
		TDSThreshold:     s.TDSThreshold,
		LineEnabled:      s.LineEnabled,
		DailySummaryTime: s.DailySummaryTime,
	}).Error
}
