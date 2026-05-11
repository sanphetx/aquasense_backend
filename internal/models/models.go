package models

import (
	"time"

	"gorm.io/gorm"
)

// BaseModel is embedded in all database models to track audit fields.
type BaseModel struct {
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index"          json:"-"` // [Security #12] proper soft-delete type
	CreatedBy *string        `json:"created_by,omitempty"`
	UpdatedBy *string        `json:"updated_by,omitempty"`
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
// Flutter sends the id_token from Google Sign-In SDK or Apple Sign-In SDK.
type SocialLoginRequest struct {
	Provider string `json:"provider" binding:"required"` // google | apple
	IDToken  string `json:"id_token" binding:"required"` // JWT from provider
}

// SocialUserInfo holds the verified user info extracted from a social provider token.
type SocialUserInfo struct {
	ProviderUserID string // unique ID from provider (Google sub / Apple sub)
	Email          string
	FirstName      string
	LastName       string
}

// AuthResponse is returned on successful login/register.
type AuthResponse struct {
	Token        string   `json:"token"`
	User         UserJSON `json:"user"`
	IsFirstLogin bool     `json:"is_first_login"`
}

// ForgotPasswordRequest mirrors POST /auth/forgot-password body.
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
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
	SubscriptionPlan    string            `gorm:"type:varchar(50)"`
	SubscriptionPlanRef *SubscriptionPlan  `gorm:"foreignKey:SubscriptionPlan;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL"`
	AvatarURL           *string            `gorm:"type:varchar(255)"`
	Role                string             `gorm:"type:varchar(20);default:'user'"`

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
	ID                   string   `gorm:"primaryKey;type:varchar(36)"`
	UserID               string   `gorm:"type:varchar(36);not null;index"` // FK → users
	User                 User     `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE,OnUpdate:CASCADE"`
	Name                 string   `gorm:"type:varchar(255);not null"`
	AreaSizeRai          float64
	CropType             string   `gorm:"type:varchar(50)"`
	YieldTonPerRai       *float64
	AvgPriceBahtPerKg    *float64
	DistributionChannels string   `gorm:"type:json"`
	SoilPh               *float64
	SoilProblems         string   `gorm:"type:json"`
	WaterSource          string   `gorm:"type:varchar(100)"`
	Latitude             *float64
	Longitude            *float64
	ActiveSensorID       *string  `gorm:"type:varchar(36)"` // FK → sensors (SET NULL on delete)
	ActiveSensor         *Sensor  `gorm:"foreignKey:ActiveSensorID;constraint:OnDelete:SET NULL"`
	BaseModel
}

// FarmJSON matches FarmModel.fromJson() in the Flutter codebase.
type FarmJSON struct {
	ID                   string   `json:"id"`
	UserID               string   `json:"user_id"`              // FK → users: ระบุว่า farm นี้เป็นของ user ไหน
	Name                 string   `json:"name"`
	AreaSizeRai          float64  `json:"area_size_rai"`
	CropType             string   `json:"crop_type"`
	YieldTonPerRai       *float64 `json:"yield_ton_per_rai"`
	AvgPriceBahtPerKg    *float64 `json:"avg_price_baht_per_kg"`
	DistributionChannels []string `json:"distribution_channels"`
	SoilPh               *float64 `json:"soil_ph"`
	SoilProblems         []string `json:"soil_problems"`
	WaterSource          string   `json:"water_source"`
	Latitude             *float64 `json:"latitude"`
	Longitude            *float64 `json:"longitude"`
	ActiveSensorID       *string  `json:"active_sensor_id"` // FK → sensors: sensor ที่ Dashboard ใช้งานอยู่
	CreatedAt            string   `json:"created_at"`
	UpdatedAt            string   `json:"updated_at"`
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
// [Fix F] Validate coordinate ranges: lat ∈ [-90, 90], lng ∈ [-180, 180].
type UpdateLocationRequest struct {
	Latitude  float64 `json:"latitude"  binding:"min=-90,max=90"`
	Longitude float64 `json:"longitude" binding:"min=-180,max=180"`
}

// LinkSensorRequest mirrors POST /farms/{id}/sensor body.
type LinkSensorRequest struct {
	SensorID string `json:"sensor_id" binding:"required"`
}

// ─── User Nodes (many-to-many) ───────────────────────────────────────────────

// UserNode is the database model for user-sensor link (supports multiple nodes).
// [Fix L] FK constraints: cascade delete when user or sensor is removed.
type UserNode struct {
	ID       string `gorm:"primaryKey;type:varchar(36)"`
	UserID   string `gorm:"type:varchar(36);not null;index;uniqueIndex:idx_user_sensor"`
	User     User   `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	SensorID string `gorm:"type:varchar(36);not null;uniqueIndex:idx_user_sensor"`
	Sensor   Sensor `gorm:"foreignKey:SensorID;constraint:OnDelete:CASCADE"`
	IsActive bool   `gorm:"default:false"`
	BaseModel
}

// NodeJSON is the response shape for node management endpoints.
type NodeJSON struct {
	ID         string   `json:"id"`
	SensorID   string   `json:"sensor_id"`
	SensorName string   `json:"sensor_name"`
	Status     string   `json:"status"`
	TDSValue   float64  `json:"tds_value"`
	IsActive   bool     `json:"is_active"`
	DistanceKm float64  `json:"distance_km,omitempty"`
	LastUpdated string  `json:"last_updated"`
}

// ─── Sensor ──────────────────────────────────────────────────────────────────

// Sensor is the database model.
type Sensor struct {
	ID          string   `gorm:"primaryKey;type:varchar(36)"`
	Name        string   `gorm:"type:varchar(255);not null"`
	Latitude    float64  `gorm:"type:decimal(10,7);not null"`
	Longitude   float64  `gorm:"type:decimal(10,7);not null"`
	Status      string   `gorm:"type:enum('safe','warning','danger');default:'safe'"` // safe | warning | danger
	TDSValue    float64  `gorm:"type:decimal(10,2);not null;default:0"`
	Temperature *float64 `gorm:"type:decimal(5,2)"`
	PH          *float64 `gorm:"type:decimal(4,2)"`
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
	ID           int64     `gorm:"primaryKey;autoIncrement"`
	SensorID     string    `gorm:"type:varchar(36);not null;index"`
	Sensor       Sensor    `gorm:"foreignKey:SensorID;constraint:OnDelete:CASCADE"`
	Date         time.Time `gorm:"not null;index"`
	TDS          float64   `gorm:"type:decimal(10,2);not null"`
	PH           *float64  `gorm:"type:decimal(4,2)"`
	Temperature  *float64  `gorm:"type:decimal(5,2)"`
	SoilMoisture *float64  `gorm:"type:decimal(5,2)"`
	Status       string    `gorm:"type:enum('safe','warning','danger');default:'safe'"`
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
	FarmID          string                 `json:"farm_id"`           // for _LocationCard
	FarmName        string                 `json:"farm_name"`         // for _LocationCard
	ActiveSensor    SensorJSON             `json:"active_sensor"`
	Recommendations []AiRecommendationJSON `json:"recommendations"`
	CropSuggestions []CropSuggestionJSON   `json:"crop_suggestions"`
	TrendHistory    []WaterRecordJSON      `json:"trend_history"`
}

// ─── Notification Settings ───────────────────────────────────────────────────

// NotificationSettings is the database model.
type NotificationSettings struct {
	UserID           string  `gorm:"primaryKey;type:varchar(36)"`
	User             User    `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	PushEnabled      bool    `gorm:"default:true"`
	TDSThreshold     float64 `gorm:"type:decimal(10,2);default:400"`
	LineEnabled      bool    `gorm:"default:false"`
	DailySummaryTime string  `gorm:"type:varchar(20);default:'none'"` // none | morning | evening | both
}

// NotificationSettingsJSON matches NotificationSettingsModel.fromJson() in Flutter.
type NotificationSettingsJSON struct {
	PushEnabled      bool    `json:"push_enabled"`
	TDSThreshold     float64 `json:"tds_threshold"`
	LineEnabled      bool    `json:"line_enabled"`
	DailySummaryTime string  `json:"daily_summary_time"`
}

// ─── Subscription ────────────────────────────────────────────────────────────

// SubscriptionPlan is the database model.
type SubscriptionPlan struct {
	ID          string `gorm:"primaryKey;type:varchar(36)"`
	Name        string `gorm:"type:varchar(50);not null"`
	Price       string `gorm:"type:varchar(50);not null"`
	Period      string `gorm:"type:varchar(50)"`
	Features    string `gorm:"type:json"` // JSON array of strings
	Recommended bool   `gorm:"default:false"`
	SortOrder   int    `gorm:"default:0"` // display order
}

// SubscriptionPlanJSON matches SubscriptionPlanModel.fromJson() in Flutter.
type SubscriptionPlanJSON struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Price       string   `json:"price"`
	Period      string   `json:"period"`
	Features    []string `json:"features"`
	Recommended bool     `json:"recommended"`
}

// ─── AI Recommendation ──────────────────────────────────────────────────────

// AiRecommendation is the database model.
type AiRecommendation struct {
	ID              string  `gorm:"primaryKey;type:varchar(36)"`
	Title           string  `gorm:"type:varchar(255);not null"`
	Body            string  `gorm:"type:text;not null"`
	Type            string  `gorm:"type:varchar(50);not null"` // tds_alert | tds_danger | crop_suggestion | fertilizer
	ReasonChips     string  `gorm:"type:json"`                 // JSON array of {label, category}
	ConfidenceScore float64 `gorm:"type:decimal(3,2);not null"`
	BaseModel
}

// ─── Crop Suggestion ─────────────────────────────────────────────────────────

// CropSuggestion is the database model.
type CropSuggestion struct {
	ID                  string  `gorm:"primaryKey;type:varchar(36)"`
	Name                string  `gorm:"type:varchar(100);not null"`
	NameTH              string  `gorm:"type:varchar(100);not null"`
	EstimatedPricePerKg float64 `gorm:"type:decimal(10,2);not null"`
	Reason              string  `gorm:"type:text;not null"`
	Icon                string  `gorm:"type:varchar(10)"`
	MinTDS              float64 `gorm:"type:decimal(10,2);default:0"`   // show when TDS >= this
	MaxTDS              float64 `gorm:"type:decimal(10,2);default:9999"` // show when TDS <= this
	SortOrder           int     `gorm:"default:0"`
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
