package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"aquasense-backend/internal/logger"
	"aquasense-backend/internal/models"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// [Fix K] Sentinel errors — use errors.Is() in handlers instead of string comparison.
var (
	ErrFarmNotFound   = errors.New("farm not found or access denied")
	ErrNodeNotFound   = errors.New("node not found or access denied")
	ErrSensorNotFound = errors.New("sensor not found")
	ErrNodeDuplicate  = errors.New("sensor นี้เชื่อมต่ออยู่แล้ว")
	ErrNodeCapacity   = errors.New("เชื่อมต่อได้สูงสุด 5 nodes")
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

// HasFarm checks if a user has at least one farm (used for isFirstLogin detection).
func (r *AuthRepository) HasFarm(userID string) bool {
	var count int64
	r.db.Model(&models.Farm{}).Where("user_id = ?", userID).Count(&count)
	return count > 0
}

// ─── FarmRepository ──────────────────────────────────────────────────────────

type FarmRepository struct {
	db *gorm.DB
}

func NewFarmRepository(db *gorm.DB) *FarmRepository {
	return &FarmRepository{db: db}
}

func (r *FarmRepository) CreateFarm(userID string, req models.CreateFarmRequest) (*models.FarmJSON, error) {
	// [Fix I] handle json.Marshal errors
	distJSON, err := json.Marshal(req.DistributionChannels)
	if err != nil {
		return nil, fmt.Errorf("marshal distribution_channels: %w", err)
	}
	probJSON, err := json.Marshal(req.SoilProblems)
	if err != nil {
		return nil, fmt.Errorf("marshal soil_problems: %w", err)
	}

	// Check if user already has a farm — if so, update it instead of creating a duplicate
	var existing models.Farm
	if err := r.db.Where("user_id = ?", userID).Order("created_at DESC").First(&existing).Error; err == nil {
		// Farm exists → update it
		updates := map[string]interface{}{
			"name":                  req.Name,
			"area_size_rai":         req.AreaSizeRai,
			"crop_type":             req.CropType,
			"yield_ton_per_rai":     req.YieldTonPerRai,
			"avg_price_baht_per_kg": req.AvgPriceBahtPerKg,
			"distribution_channels": string(distJSON),
			"soil_ph":               req.SoilPh,
			"soil_problems":         string(probJSON),
			"water_source":          req.WaterSource,
		}
		if err := r.db.Model(&existing).Updates(updates).Error; err != nil {
			return nil, err
		}
		// Re-read from DB to get the actual updated values
		r.db.Where("id = ?", existing.ID).First(&existing)
		return r.farmToJSON(&existing, req.DistributionChannels, req.SoilProblems), nil
	}

	// No farm yet → create new
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

	return r.farmToJSON(farm, req.DistributionChannels, req.SoilProblems), nil
}

// farmToJSON converts a Farm model to FarmJSON using pre-parsed slices.
func (r *FarmRepository) farmToJSON(farm *models.Farm, dist, prob []string) *models.FarmJSON {
	return &models.FarmJSON{
		ID:                   farm.ID,
		UserID:               farm.UserID,
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
		CreatedAt:            farm.CreatedAt.Format(time.RFC3339),
		UpdatedAt:            farm.UpdatedAt.Format(time.RFC3339),
	}
}

func (r *FarmRepository) GetFarmByUserID(userID string) (*models.FarmJSON, error) {
	var farm models.Farm
	if err := r.db.Where("user_id = ?", userID).Order("created_at DESC").First(&farm).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	// [Fix H] log warning if JSON fields are malformed instead of silently ignoring
	var dist []string
	if err := json.Unmarshal([]byte(farm.DistributionChannels), &dist); err != nil {
		logger.Get().Warn("malformed distribution_channels JSON", zap.String("farm_id", farm.ID))
		dist = []string{}
	}
	var prob []string
	if err := json.Unmarshal([]byte(farm.SoilProblems), &prob); err != nil {
		logger.Get().Warn("malformed soil_problems JSON", zap.String("farm_id", farm.ID))
		prob = []string{}
	}

	return &models.FarmJSON{
		ID:                   farm.ID,
		UserID:               farm.UserID, // FK → users: ระบุเจ้าของ farm
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
		ActiveSensorID:       farm.ActiveSensorID, // FK → sensors: sensor ที่ Dashboard ใช้งานอยู่
		CreatedAt:            farm.CreatedAt.Format(time.RFC3339),
		UpdatedAt:            farm.UpdatedAt.Format(time.RFC3339),
	}, nil
}

// [Security #1] UpdateLocation verifies farm ownership before updating.
func (r *FarmRepository) UpdateLocation(userID, farmID string, lat, lng float64) error {
	lat = float64(int(lat*100)) / 100
	lng = float64(int(lng*100)) / 100
	result := r.db.Model(&models.Farm{}).Where("id = ? AND user_id = ?", farmID, userID).Updates(map[string]interface{}{
		"latitude":  lat,
		"longitude": lng,
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrFarmNotFound // [Fix K] sentinel error
	}
	return nil
}

// [Security #1] LinkSensor verifies farm ownership and sensor existence before linking.
func (r *FarmRepository) LinkSensor(userID, farmID, sensorID string) error {
	// Validate sensor exists
	var sensor models.Sensor
	if err := r.db.Where("id = ?", sensorID).First(&sensor).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrSensorNotFound
		}
		return err
	}

	result := r.db.Model(&models.Farm{}).Where("id = ? AND user_id = ?", farmID, userID).Update("active_sensor_id", sensorID)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrFarmNotFound // [Fix K] sentinel error
	}
	return nil
}

func (r *FarmRepository) GetFarmByID(farmID string) (*models.FarmJSON, error) {
	var farm models.Farm
	if err := r.db.Where("id = ?", farmID).First(&farm).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	// [Fix H] log warning if JSON fields are malformed
	var dist []string
	if err := json.Unmarshal([]byte(farm.DistributionChannels), &dist); err != nil {
		logger.Get().Warn("malformed distribution_channels JSON", zap.String("farm_id", farm.ID))
		dist = []string{}
	}
	var prob []string
	if err := json.Unmarshal([]byte(farm.SoilProblems), &prob); err != nil {
		logger.Get().Warn("malformed soil_problems JSON", zap.String("farm_id", farm.ID))
		prob = []string{}
	}

	return &models.FarmJSON{
		ID:                   farm.ID,
		UserID:               farm.UserID,
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
		CreatedAt:            farm.CreatedAt.Format(time.RFC3339),
		UpdatedAt:            farm.UpdatedAt.Format(time.RFC3339),
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
	// [Blocker #2] SELECT includes distance_km — GORM Raw Scan maps it to SensorJSON.DistanceKm
	// last_updated is formatted as ISO 8601 string to match SensorJSON.LastUpdated
	var sensors []models.SensorJSON
	err := r.db.Raw(`
		SELECT
			id, name, latitude, longitude, status, tds_value, temperature, ph,
			DATE_FORMAT(updated_at, '%Y-%m-%dT%H:%i:%sZ') AS last_updated,
			ROUND(
				6371 * ACOS(
					COS(RADIANS(?)) * COS(RADIANS(latitude)) * COS(RADIANS(longitude) - RADIANS(?))
					+ SIN(RADIANS(?)) * SIN(RADIANS(latitude))
				), 2
			) AS distance_km
		FROM sensors
		WHERE deleted_at IS NULL
		ORDER BY distance_km ASC
		LIMIT 50
	`, lat, lng, lat).Scan(&sensors).Error
	if sensors == nil {
		sensors = []models.SensorJSON{}
	}
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

	result := make([]models.WaterRecordJSON, 0, len(records))
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

	return r.db.Model(&existing).Updates(map[string]interface{}{
		"push_enabled":       s.PushEnabled,
		"tds_threshold":      s.TDSThreshold,
		"line_enabled":       s.LineEnabled,
		"daily_summary_time": s.DailySummaryTime,
	}).Error
}

// ─── NodeRepository ──────────────────────────────────────────────────────────

type NodeRepository struct {
	db *gorm.DB
}

func NewNodeRepository(db *gorm.DB) *NodeRepository {
	return &NodeRepository{db: db}
}

// GetUserNodes returns all sensor nodes linked to a user with sensor details.
func (r *NodeRepository) GetUserNodes(userID string) ([]models.NodeJSON, error) {
	var nodes []models.UserNode
	if err := r.db.Preload("Sensor").Where("user_id = ?", userID).Order("created_at ASC").Find(&nodes).Error; err != nil {
		return nil, err
	}

	var result []models.NodeJSON
	for _, n := range nodes {
		result = append(result, models.NodeJSON{
			ID:          n.ID,
			SensorID:    n.Sensor.ID,
			SensorName:  n.Sensor.Name,
			Status:      n.Sensor.Status,
			TDSValue:    n.Sensor.TDSValue,
			IsActive:    n.IsActive,
			LastUpdated: n.Sensor.UpdatedAt.Format(time.RFC3339),
		})
	}
	return result, nil
}

// AddNode links a sensor to a user. If it's the first node, set it as active.
// [Fix D] Entire operation wrapped in a DB transaction to prevent race conditions.
// [Fix G] Validates sensor exists before linking.
func (r *NodeRepository) AddNode(userID, sensorID string) (*models.NodeJSON, error) {
	var result *models.NodeJSON

	err := r.db.Transaction(func(tx *gorm.DB) error {
		// [Fix G] Validate sensor exists before linking
		var sensor models.Sensor
		if err := tx.Where("id = ?", sensorID).First(&sensor).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return ErrSensorNotFound
			}
			return err
		}

		// Check if already linked
		var existing models.UserNode
		if err := tx.Where("user_id = ? AND sensor_id = ?", userID, sensorID).First(&existing).Error; err == nil {
			return ErrNodeDuplicate // [Fix K] sentinel error
		}

		// Check capacity (inside transaction to prevent race condition)
		var count int64
		tx.Model(&models.UserNode{}).Where("user_id = ?", userID).Count(&count)
		if count >= 5 {
			return ErrNodeCapacity // [Fix K] sentinel error
		}

		isFirst := count == 0
		node := models.UserNode{
			ID:       uuid.NewString(),
			UserID:   userID,
			SensorID: sensorID,
			IsActive: isFirst,
		}
		if err := tx.Create(&node).Error; err != nil {
			return err
		}

		// [Fix M] If first node, sync farms.active_sensor_id automatically
		if isFirst {
			tx.Model(&models.Farm{}).Where("user_id = ?", userID).Update("active_sensor_id", sensorID)
		}

		result = &models.NodeJSON{
			ID:          node.ID,
			SensorID:    sensor.ID,
			SensorName:  sensor.Name,
			Status:      sensor.Status,
			TDSValue:    sensor.TDSValue,
			IsActive:    node.IsActive,
			LastUpdated: sensor.UpdatedAt.Format(time.RFC3339),
		}
		return nil
	})
	return result, err
}

// SetActiveNode sets a specific node as active and deactivates others.
// [Fix M] Also syncs farms.active_sensor_id to match the active node's sensor.
func (r *NodeRepository) SetActiveNode(userID, nodeID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Deactivate all nodes for this user
		if err := tx.Model(&models.UserNode{}).Where("user_id = ?", userID).Update("is_active", false).Error; err != nil {
			return err
		}

		// Activate the selected node
		result := tx.Model(&models.UserNode{}).Where("id = ? AND user_id = ?", nodeID, userID).Update("is_active", true)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrNodeNotFound // [Fix K] sentinel error
		}

		// [Fix M] Get the activated node's sensorID and sync to farms.active_sensor_id
		var node models.UserNode
		if err := tx.Select("sensor_id").Where("id = ?", nodeID).First(&node).Error; err == nil {
			// Best-effort sync — don't fail if user has no farm yet
			tx.Model(&models.Farm{}).Where("user_id = ?", userID).Update("active_sensor_id", node.SensorID)
		}

		return nil
	})
}

// RemoveNode unlinks a sensor from a user.
// If the removed node was active, promote the next available node or clear active_sensor_id.
func (r *NodeRepository) RemoveNode(userID, nodeID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Find the node before deleting (need to know if it was active)
		var node models.UserNode
		if err := tx.Where("id = ? AND user_id = ?", nodeID, userID).First(&node).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return ErrNodeNotFound
			}
			return err
		}

		wasActive := node.IsActive
		removedSensorID := node.SensorID

		// Delete the node
		if err := tx.Delete(&node).Error; err != nil {
			return err
		}

		// If the removed node was active, promote another or clear farm's active_sensor_id
		if wasActive {
			var nextNode models.UserNode
			if err := tx.Where("user_id = ?", userID).Order("created_at ASC").First(&nextNode).Error; err == nil {
				// Promote next node
				tx.Model(&nextNode).Update("is_active", true)
				tx.Model(&models.Farm{}).Where("user_id = ?", userID).Update("active_sensor_id", nextNode.SensorID)
			} else {
				// No nodes left — clear farm's active_sensor_id
				tx.Model(&models.Farm{}).Where("user_id = ? AND active_sensor_id = ?", userID, removedSensorID).Update("active_sensor_id", nil)
			}
		}

		return nil
	})
}

// ─── SubscriptionPlanRepository ──────────────────────────────────────────────

type SubscriptionPlanRepository struct {
	db *gorm.DB
}

func NewSubscriptionPlanRepository(db *gorm.DB) *SubscriptionPlanRepository {
	return &SubscriptionPlanRepository{db: db}
}

func (r *SubscriptionPlanRepository) GetAll() ([]models.SubscriptionPlanJSON, error) {
	var plans []models.SubscriptionPlan
	if err := r.db.Order("sort_order ASC").Find(&plans).Error; err != nil {
		return nil, err
	}

	result := make([]models.SubscriptionPlanJSON, 0, len(plans))
	for _, p := range plans {
		var features []string
		if err := json.Unmarshal([]byte(p.Features), &features); err != nil {
			features = []string{}
		}
		result = append(result, models.SubscriptionPlanJSON{
			ID:          p.ID,
			Name:        p.Name,
			Price:       p.Price,
			Period:      p.Period,
			Features:    features,
			Recommended: p.Recommended,
		})
	}
	return result, nil
}

func (r *SubscriptionPlanRepository) FindByID(planID string) bool {
	var count int64
	r.db.Model(&models.SubscriptionPlan{}).Where("id = ?", planID).Count(&count)
	return count > 0
}

// ─── AiRepository ────────────────────────────────────────────────────────────

type AiRepository struct {
	db *gorm.DB
}

func NewAiRepository(db *gorm.DB) *AiRepository {
	return &AiRepository{db: db}
}

func (r *AiRepository) GetRecommendations() ([]models.AiRecommendationJSON, error) {
	var recs []models.AiRecommendation
	if err := r.db.Order("created_at DESC").Find(&recs).Error; err != nil {
		return nil, err
	}

	result := make([]models.AiRecommendationJSON, 0, len(recs))
	for _, rec := range recs {
		var chips []models.ReasonChip
		if err := json.Unmarshal([]byte(rec.ReasonChips), &chips); err != nil {
			chips = []models.ReasonChip{}
		}
		result = append(result, models.AiRecommendationJSON{
			ID:              rec.ID,
			Title:           rec.Title,
			Body:            rec.Body,
			Type:            rec.Type,
			CreatedAt:       rec.CreatedAt.Format(time.RFC3339),
			ReasonChips:     chips,
			ConfidenceScore: rec.ConfidenceScore,
		})
	}
	return result, nil
}

func (r *AiRepository) GetRecommendationByID(id string) (*models.AiRecommendationJSON, error) {
	var rec models.AiRecommendation
	if err := r.db.Where("id = ?", id).First(&rec).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}

	var chips []models.ReasonChip
	if err := json.Unmarshal([]byte(rec.ReasonChips), &chips); err != nil {
		chips = []models.ReasonChip{}
	}

	return &models.AiRecommendationJSON{
		ID:              rec.ID,
		Title:           rec.Title,
		Body:            rec.Body,
		Type:            rec.Type,
		CreatedAt:       rec.CreatedAt.Format(time.RFC3339),
		ReasonChips:     chips,
		ConfidenceScore: rec.ConfidenceScore,
	}, nil
}

func (r *AiRepository) GetAdvisoryHistory() ([]models.AiRecommendationJSON, error) {
	var recs []models.AiRecommendation
	if err := r.db.Where("created_at >= DATE_SUB(NOW(), INTERVAL 30 DAY)").
		Order("created_at DESC").Find(&recs).Error; err != nil {
		return nil, err
	}

	result := make([]models.AiRecommendationJSON, 0, len(recs))
	for _, rec := range recs {
		var chips []models.ReasonChip
		if err := json.Unmarshal([]byte(rec.ReasonChips), &chips); err != nil {
			chips = []models.ReasonChip{}
		}
		result = append(result, models.AiRecommendationJSON{
			ID:              rec.ID,
			Title:           rec.Title,
			Body:            rec.Body,
			Type:            rec.Type,
			CreatedAt:       rec.CreatedAt.Format(time.RFC3339),
			ReasonChips:     chips,
			ConfidenceScore: rec.ConfidenceScore,
		})
	}
	return result, nil
}

func (r *AiRepository) GetCropSuggestions(tds float64) ([]models.CropSuggestionJSON, error) {
	var crops []models.CropSuggestion
	if err := r.db.Where("min_tds <= ? AND max_tds >= ?", tds, tds).
		Order("sort_order ASC").Find(&crops).Error; err != nil {
		return nil, err
	}

	// If no match (e.g. tds=0), return all
	if len(crops) == 0 {
		if err := r.db.Order("sort_order ASC").Find(&crops).Error; err != nil {
			return nil, err
		}
	}

	result := make([]models.CropSuggestionJSON, 0, len(crops))
	for _, c := range crops {
		result = append(result, models.CropSuggestionJSON{
			Name:                c.Name,
			NameTH:              c.NameTH,
			EstimatedPricePerKg: c.EstimatedPricePerKg,
			Reason:              c.Reason,
			Icon:                c.Icon,
		})
	}
	return result, nil
}
