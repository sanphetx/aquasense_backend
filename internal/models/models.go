package models

import "time"

// BaseModel is embedded in all database models to track audit fields.
type BaseModel struct {
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
	CreatedBy *string    `db:"created_by" json:"created_by,omitempty"`
	UpdatedBy *string    `db:"updated_by" json:"updated_by,omitempty"`
}

// ─── Auth ────────────────────────────────────────────────────────────────────

// LoginRequest mirrors POST /auth/login body.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// RegisterRequest mirrors POST /auth/register body.
// Flutter RegisterPage sends either:
//   - display_name (full name as single string, per MockAuthRepository.register signature)
//   - OR first_name + last_name separately
type RegisterRequest struct {
	DisplayName string    `json:"display_name"` // optional: "สมชาย ใจดี" → split into first/last
	FirstName   string    `json:"first_name"`   // optional if display_name provided
	LastName    string    `json:"last_name"`    // optional if display_name provided
	Email       string    `json:"email" binding:"required,email"`
	Phone       string    `json:"phone"`
	BirthDate   time.Time `json:"birth_date" binding:"required"`
	Password    string    `json:"password" binding:"required,min=6"`
}

// SocialLoginRequest mirrors POST /auth/social body.
type SocialLoginRequest struct {
	Provider    string `json:"provider" binding:"required"` // google | facebook
	AccessToken string `json:"access_token" binding:"required"`
}

// AuthResponse is returned on successful login/register.
type AuthResponse struct {
	Token string   `json:"token"`
	User  UserJSON `json:"user"`
}

// ─── User ────────────────────────────────────────────────────────────────────

// User is the database model (maps 1:1 with the users table).
type User struct {
	ID               string `gorm:"primaryKey;type:varchar(36)"`
	FirstName        string `gorm:"type:varchar(100)"`
	LastName         string `gorm:"type:varchar(100)"`
	Email            string `gorm:"type:varchar(255);uniqueIndex"`
	Phone            string `gorm:"type:varchar(20)"`
	BirthDate        time.Time
	PasswordHash     string  `gorm:"type:varchar(255)"`
	SubscriptionPlan string  `gorm:"type:varchar(50)"`
	AvatarURL        *string `gorm:"type:varchar(255)"`
	Role             string  `gorm:"type:varchar(20);default:'user'"`

	BaseModel
}

// UserJSON is the public-facing user shape sent to the Flutter app.
// Matches UserModel.fromJson() in the Flutter codebase.
type UserJSON struct {
	ID               string  `json:"id"`
	FirstName        string  `json:"first_name"`
	LastName         string  `json:"last_name"`
	Email            string  `json:"email"`
	Phone            string  `json:"phone"`
	BirthDate        string  `json:"birth_date"` // ISO 8601
	SubscriptionPlan string  `json:"subscription_plan"`
	AvatarURL        *string `json:"avatar_url,omitempty"`
	Role             string  `json:"role"`
}

// ToJSON converts the internal User model to the public UserJSON.
func (u *User) ToJSON() UserJSON {
	return UserJSON{
		ID:               u.ID,
		FirstName:        u.FirstName,
		LastName:         u.LastName,
		Email:            u.Email,
		Phone:            u.Phone,
		BirthDate:        u.BirthDate.Format(time.RFC3339),
		SubscriptionPlan: u.SubscriptionPlan,
		AvatarURL:        u.AvatarURL,
		Role:             u.Role,
	}
}

// UpdateProfileRequest mirrors PUT /users/profile body.
type UpdateProfileRequest struct {
	FirstName string `json:"first_name" binding:"required"`
	LastName  string `json:"last_name" binding:"required"`
	Phone     string `json:"phone"`
}

// ─── Farm ────────────────────────────────────────────────────────────────────

// Farm is the database model.
type Farm struct {
	ID                   string `gorm:"primaryKey"`
	UserID               string
	Name                 string
	AreaSizeRai          float64
	CropType             string
	YieldTonPerRai       *float64
	AvgPriceBahtPerKg    *float64
	DistributionChannels string `gorm:"type:json"` // stored as JSON
	SoilPh               *float64
	SoilProblems         string `gorm:"type:json"` // stored as JSON
	WaterSource          string
	Latitude             *float64
	Longitude            *float64
	ActiveSensorID       *string
	BaseModel
}

// FarmJSON matches FarmModel.fromJson() in the Flutter codebase.
type FarmJSON struct {
	ID                   string   `json:"id"`
	Name                 string   `json:"name"`
	AreaSizeRai          float64  `json:"area_size_rai"`
	CropType             string   `json:"crop_type"`
	YieldTonPerRai       *float64 `json:"yield_ton_per_rai,omitempty"`
	AvgPriceBahtPerKg    *float64 `json:"avg_price_baht_per_kg,omitempty"`
	DistributionChannels []string `json:"distribution_channels"`
	SoilPh               *float64 `json:"soil_ph,omitempty"`
	SoilProblems         []string `json:"soil_problems"`
	WaterSource          string   `json:"water_source"`
	Latitude             *float64 `json:"latitude,omitempty"`
	Longitude            *float64 `json:"longitude,omitempty"`
	ActiveSensorID       *string  `json:"active_sensor_id,omitempty"`
}

// CreateFarmRequest mirrors POST /farms body.
type CreateFarmRequest struct {
	Name                 string   `json:"name" binding:"required"`
	AreaSizeRai          float64  `json:"area_size_rai" binding:"required,gt=0"`
	CropType             string   `json:"crop_type" binding:"required"`
	YieldTonPerRai       *float64 `json:"yield_ton_per_rai"`
	AvgPriceBahtPerKg    *float64 `json:"avg_price_baht_per_kg"`
	DistributionChannels []string `json:"distribution_channels"`
	SoilPh               *float64 `json:"soil_ph"`
	SoilProblems         []string `json:"soil_problems"`
	WaterSource          string   `json:"water_source" binding:"required"`
}

// UpdateLocationRequest mirrors PUT /farms/{id}/location body.
// Note: binding:"required" is NOT used on float64 — 0.0 is a valid coordinate.
type UpdateLocationRequest struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// LinkSensorRequest mirrors POST /farms/{id}/sensor body.
type LinkSensorRequest struct {
	SensorID string `json:"sensor_id" binding:"required"`
}

// ─── Sensor ──────────────────────────────────────────────────────────────────

// Sensor is the database model.
type Sensor struct {
	ID          string   `db:"id"`
	Name        string   `db:"name"`
	Latitude    float64  `db:"latitude"`
	Longitude   float64  `db:"longitude"`
	Status      string   `db:"status"` // safe | warning | danger
	TDSValue    float64  `db:"tds_value"`
	Temperature *float64 `db:"temperature"`
	PH          *float64 `db:"ph"`
	BaseModel
}

// SensorJSON matches SensorModel.fromJson() in Flutter.
type SensorJSON struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Latitude    float64  `json:"latitude"`
	Longitude   float64  `json:"longitude"`
	Status      string   `json:"status"`
	TDSValue    float64  `json:"tds_value"`
	Temperature *float64 `json:"temperature,omitempty"`
	PH          *float64 `json:"ph,omitempty"`
	DistanceKm  float64  `json:"distance_km"`
	LastUpdated string   `json:"last_updated"` // ISO 8601
}

// ─── Water History ───────────────────────────────────────────────────────────

// WaterRecord is the database model for sensor history.
type WaterRecord struct {
	ID           int64     `db:"id"`
	SensorID     string    `db:"sensor_id"`
	Date         time.Time `db:"date"`
	TDS          float64   `db:"tds"`
	PH           *float64  `db:"ph"`
	Temperature  *float64  `db:"temperature"`
	SoilMoisture *float64  `db:"soil_moisture"`
	Status       string    `db:"status"`
}

// WaterRecordJSON matches WaterRecordModel.fromJson() in Flutter.
type WaterRecordJSON struct {
	Date         string   `json:"date"` // ISO 8601
	TDS          float64  `json:"tds"`
	PH           *float64 `json:"ph,omitempty"`
	Temperature  *float64 `json:"temperature,omitempty"`
	SoilMoisture *float64 `json:"soil_moisture,omitempty"`
	Status       string   `json:"status"`
}

// ─── AI / Recommendations ────────────────────────────────────────────────────

// ReasonChip matches the Flutter ReasonChip model.
type ReasonChip struct {
	Label    string `json:"label"`
	Category string `json:"category"` // tds | trend | weather | market
}

// AiRecommendationJSON matches AiRecommendationModel.fromJson() in Flutter.
type AiRecommendationJSON struct {
	ID              string       `json:"id"`
	Title           string       `json:"title"`
	Body            string       `json:"body"`
	Type            string       `json:"type"`       // tds_alert | tds_danger | crop_suggestion | fertilizer
	CreatedAt       string       `json:"created_at"` // ISO 8601
	ReasonChips     []ReasonChip `json:"reason_chips"`
	ConfidenceScore float64      `json:"confidence_score"`
}

// CropSuggestionJSON matches CropSuggestionModel.fromJson() in Flutter.
type CropSuggestionJSON struct {
	Name                string  `json:"name"`
	NameTH              string  `json:"name_th"`
	EstimatedPricePerKg float64 `json:"estimated_price_per_kg"`
	Reason              string  `json:"reason"`
	Icon                string  `json:"icon"`
}

// ─── Dashboard ───────────────────────────────────────────────────────────────

// DashboardSummaryJSON matches DashboardSummaryModel in Flutter.
type DashboardSummaryJSON struct {
	ActiveSensor    SensorJSON             `json:"active_sensor"`
	Recommendations []AiRecommendationJSON `json:"recommendations"`
	CropSuggestions []CropSuggestionJSON   `json:"crop_suggestions"`
	TrendHistory    []WaterRecordJSON      `json:"trend_history"`
}

// ─── Notification Settings ───────────────────────────────────────────────────

// NotificationSettings is the database model.
type NotificationSettings struct {
	UserID           string `gorm:"primaryKey"`
	PushEnabled      bool
	TDSThreshold     float64
	LineEnabled      bool
	DailySummaryTime string // none | morning | evening | both
}

// NotificationSettingsJSON matches NotificationSettingsModel.fromJson() in Flutter.
type NotificationSettingsJSON struct {
	PushEnabled      bool    `json:"push_enabled"`
	TDSThreshold     float64 `json:"tds_threshold"`
	LineEnabled      bool    `json:"line_enabled"`
	DailySummaryTime string  `json:"daily_summary_time"`
}

// ─── Subscription ────────────────────────────────────────────────────────────

// SubscriptionPlanJSON matches SubscriptionPlanModel.fromJson() in Flutter.
type SubscriptionPlanJSON struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Price       string   `json:"price"`
	Period      string   `json:"period"`
	Features    []string `json:"features"`
	Recommended bool     `json:"recommended"`
}

// SubscribeRequest mirrors POST /subscriptions/subscribe body.
type SubscribeRequest struct {
	PlanID string `json:"plan_id" binding:"required"`
}

// ─── API Response wrapper ────────────────────────────────────────────────────

// APIResponse is the standard JSON envelope sent back to Flutter.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// ErrorResponse is returned on errors.
type ErrorResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error"`
}

// Pagination holds the standard pagination metadata.
type Pagination struct {
	CurrentPage int   `json:"current_page"`
	PageSize    int   `json:"page_size"`
	TotalItems  int64 `json:"total_items"`
	TotalPages  int   `json:"total_pages"`
}

// PaginatedResponse is the standard JSON envelope for lists of data.
type PaginatedResponse struct {
	Success    bool        `json:"success"`
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}
